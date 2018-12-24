package secrethub

import (
	"fmt"
	"text/tabwriter"

	"io"

	"github.com/keylockerbv/secrethub-cli/pkg/ui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
)

// OrgRevokeCommand handles revoking a member from an organization.
type OrgRevokeCommand struct {
	orgName   api.OrgName
	username  string
	io        ui.IO
	newClient newClientFunc
}

// NewOrgRevokeCommand creates a new OrgRevokeCommand.
func NewOrgRevokeCommand(io ui.IO, newClient newClientFunc) *OrgRevokeCommand {
	return &OrgRevokeCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *OrgRevokeCommand) Register(r Registerer) {
	clause := r.Command("revoke", "Revoke a user from an organization. This automatically revokes the user from all of the organization's repositories. A list of repositories containing secrets that should be rotated will be printed out.")
	clause.Arg("org-name", "The organization name").Required().SetValue(&cmd.orgName)
	clause.Arg("username", "The username of the user").Required().StringVar(&cmd.username)

	BindAction(clause, cmd.Run)
}

// Run revokes an organization member.
func (cmd *OrgRevokeCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return errio.Error(err)
	}

	opts := &api.RevokeOpts{
		DryRun: true,
	}
	planned, err := client.Orgs().Members().Revoke(cmd.orgName.Value(), cmd.username, opts)
	if err != nil {
		return errio.Error(err)
	}

	if len(planned.Repos) > 0 {
		fmt.Fprintf(
			cmd.io.Stdout(),
			"[WARNING] Revoking %s from the %s organization will revoke the user from %d repositories, "+
				"automatically flagging secrets for rotation.\n\n"+
				"A revocation plan has been generated and is shown below. "+
				"Flagged repositories will contain secrets flagged for rotation, "+
				"failed repositories require a manual removal or access rule changes before proceeding and "+
				"OK repos will not require rotation.\n\n",
			cmd.username,
			cmd.orgName,
			len(planned.Repos),
		)

		err = writeOrgRevokeRepoList(cmd.io.Stdout(), planned.Repos...)
		if err != nil {
			return errio.Error(err)
		}

		flagged := planned.StatusCounts[api.StatusFlagged]
		failed := planned.StatusCounts[api.StatusFailed]
		unaffected := planned.StatusCounts[api.StatusOK]

		fmt.Fprintf(cmd.io.Stdout(), "Revocation plan: %d to flag, %d to fail, %d OK.\n\n", flagged, failed, unaffected)
	} else {
		fmt.Fprintf(
			cmd.io.Stdout(),
			"The user %s has no memberships to any of %s's repos and can be safely removed.\n\n",
			cmd.username,
			cmd.orgName,
		)
	}

	confirmed, err := ui.ConfirmCaseInsensitive(
		cmd.io,
		"Please type in the username of the user to confirm and proceed with revocation",
		cmd.username,
	)
	if err != nil {
		return err
	}

	if !confirmed {
		fmt.Fprintln(cmd.io.Stdout(), "Name does not match. Aborting.")
		return nil
	}

	fmt.Fprintf(cmd.io.Stdout(), "\nRevoking user...\n")

	revoked, err := client.Orgs().Members().Revoke(cmd.orgName.Value(), cmd.username, nil)
	if err != nil {
		return errio.Error(err)
	}

	if len(revoked.Repos) > 0 {
		fmt.Fprintln(cmd.io.Stdout(), "")
		err = writeOrgRevokeRepoList(cmd.io.Stdout(), revoked.Repos...)
		if err != nil {
			return errio.Error(err)
		}

		flagged := revoked.StatusCounts[api.StatusFlagged]
		failed := revoked.StatusCounts[api.StatusFailed]
		unaffected := revoked.StatusCounts[api.StatusOK]

		fmt.Fprintf(
			cmd.io.Stdout(),
			"Revoke complete! Repositories: %d flagged, %d failed, %d OK.\n",
			flagged,
			failed,
			unaffected,
		)
	} else {
		fmt.Fprintln(cmd.io.Stdout(), "Revoke complete!")
	}

	return nil
}

// writeOrgRevokeRepoList is a helper function that writes repos with a status.
func writeOrgRevokeRepoList(w io.Writer, repos ...*api.RevokeRepoResponse) error {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	for _, resp := range repos {
		fmt.Fprintf(tw, "\t%s/%s\t=> %s\n", resp.Namespace, resp.Name, resp.Status)
	}
	err := tw.Flush()
	if err != nil {
		return errio.Error(err)
	}
	fmt.Fprintln(w, "")
	return nil
}
