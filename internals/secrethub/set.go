package secrethub

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secretspec"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/spf13/cobra"
)

// Errors
var (
	ErrFileNotFound      = errMain.Code("file_not_found").ErrorPref("configuration file `%s` does not exist")
	ErrCannotReadFile    = errMain.Code("cannot_read_file").ErrorPref("cannot read file at %s: %v")
	ErrSecretsNotCleared = errMain.Code("secrets_not_cleared").Error("exiting without having cleared all secrets")
	ErrNoSourcesInSpec   = errMain.Code("no_sources_in_spec").Error("cannot find any sources in the .yml spec file")
)

// SetCommand parses a secret spec file and presents secrets on the system.
type SetCommand struct {
	in        string
	io        ui.IO
	newClient newClientFunc
}

// NewSetCommand creates a new SetCommand.
func NewSetCommand(io ui.IO, newClient newClientFunc) *SetCommand {
	return &SetCommand{
		io:        io,
		newClient: newClient,
	}
}

func (cmd *SetCommand) Register(c *cobra.Command) {
	command := &cobra.Command{
		Use:    "set",
		Short:  "Set the secrets in your local environment. This reads and parses the secrets.yml file in the current working directory.",
		Hidden: true,
		RunE:   cmd.Run,
	}
	command.Flags().StringVarP(&cmd.in, "in", "i", "secrets.yml", "The path to a secrets.yml file to read")
	c.AddCommand(command)
}

// Run parses a secret spec file and presents secrets on the system.
func (cmd *SetCommand) Run(_ *cobra.Command, args []string) error {
	presenter, err := secretspec.NewPresenter("", true, secretspec.DefaultParsers...)
	if err != nil {
		return err
	}

	_, err = os.Stat(cmd.in)
	if os.IsNotExist(err) {
		return ErrFileNotFound(cmd.in)
	}

	spec, err := ioutil.ReadFile(cmd.in)
	if err != nil {
		return ErrCannotReadFile(cmd.in, err)
	}

	err = presenter.Parse(spec)
	if err != nil {
		return err
	}

	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	paths := presenter.Sources()
	if len(paths) == 0 {
		return ErrNoSourcesInSpec
	}

	for _, c := range presenter.EmptyConsumables() {
		fmt.Fprintf(cmd.io.Output(), "Warning: %s contains no secret declarations.\n", c)
	}

	secrets := make(map[string]api.SecretVersion)
	for path := range paths {
		secret, err := client.Secrets().Versions().GetWithData(path)
		if err != nil {
			return err
		}
		secrets[path] = *secret
	}

	fmt.Fprintln(cmd.io.Output(), "Setting secrets...")

	err = presenter.Set(secrets)
	if err != nil {
		return err
	}

	fmt.Fprintln(cmd.io.Output(), "Set complete! The secrets are now available on your system.")

	return nil
}
