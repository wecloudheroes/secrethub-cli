package secrethub

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/clip"
	"github.com/secrethub/secrethub-cli/internals/cli/filemode"
	"github.com/secrethub/secrethub-cli/internals/cli/posix"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"

	"github.com/docker/go-units"
	"github.com/spf13/cobra"
)

// ReadCommand is a command to read a secret.
type ReadCommand struct {
	io                  ui.IO
	path                api.SecretPath
	useClipboard        bool
	clearClipboardAfter time.Duration
	clipper             clip.Clipper
	outFile             string
	fileMode            filemode.FileMode
	noNewLine           bool
	newClient           newClientFunc
}

// NewReadCommand creates a new ReadCommand.
func NewReadCommand(io ui.IO, newClient newClientFunc) *ReadCommand {
	return &ReadCommand{
		clipper:             clip.NewClipboard(),
		clearClipboardAfter: defaultClearClipboardAfter,
		io:                  io,
		newClient:           newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *ReadCommand) Register(r cli.Registerer) {
	clause := r.Command("read", "Read a secret.")
	clause.Cmd.Args = cobra.ExactValidArgs(1)

	clause.Flags().BoolVarP(&cmd.useClipboard,
		"clip", "c", false,
		fmt.Sprintf(
			"Copy the secret value to the clipboard. The clipboard is automatically cleared after %s.",
			units.HumanDuration(cmd.clearClipboardAfter),
		),
	)
	clause.Flags().StringVarP(&cmd.outFile, "out-file", "o", "", "Write the secret value to this file.")
	clause.Flags().BoolVarP(&cmd.noNewLine, "no-newline", "n", false, "Do not print a new line after the secret")

	fileModeFlag := clause.Flags().VarPF(&cmd.fileMode, "file-mode", "", "Set filemode for the output file. Defaults to 0600 (read and write for current user) and is ignored without the --out-file flag.")
	fileModeFlag.DefValue = "0600"

	clause.BindAction(cmd.Run)
	clause.BindArguments([]cli.ArgValue{&cmd.path})
}

// Run handles the command with the options as specified in the command.
func (cmd *ReadCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	secret, err := client.Secrets().Versions().GetWithData(cmd.path.Value())
	if err != nil {
		return err
	}

	if cmd.useClipboard {
		err = WriteClipboardAutoClear(secret.Data, cmd.clearClipboardAfter, cmd.clipper)
		if err != nil {
			return err
		}

		_, _ = fmt.Fprintf(
			cmd.io.Output(),
			"Copied %s to clipboard. It will be cleared after %s.\n",
			cmd.path,
			units.HumanDuration(cmd.clearClipboardAfter),
		)
	}

	secretData := secret.Data
	if !cmd.noNewLine {
		secretData = posix.AddNewLine(secretData)
	}

	if cmd.outFile != "" {
		err = ioutil.WriteFile(cmd.outFile, secretData, cmd.fileMode.FileMode())
		if err != nil {
			return ErrCannotWrite(cmd.outFile, err)
		}
	}

	if cmd.outFile == "" && !cmd.useClipboard {
		_, _ = fmt.Fprintf(cmd.io.Output(), "%s", string(secretData))
	}

	return nil
}
