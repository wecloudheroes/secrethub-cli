package secrethub

import "github.com/keylockerbv/secrethub-cli/pkg/ui"

// ServiceCommand handles operations on services.
type ServiceCommand struct {
	io        ui.IO
	newClient newClientFunc
}

// NewServiceCommand creates a new ServiceCommand.
func NewServiceCommand(io ui.IO, newClient newClientFunc) *ServiceCommand {
	return &ServiceCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command and its sub-commands on the provided Registerer.
func (cmd *ServiceCommand) Register(r Registerer) {
	clause := r.Command("service", "Manage a service account.")
	NewServiceDeployCommand(cmd.io).Register(clause)
	NewServiceInitCommand(cmd.io, cmd.newClient).Register(clause)
}
