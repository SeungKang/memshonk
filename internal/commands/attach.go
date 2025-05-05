package commands

import "context"

var _ Command = (*AttachCommand)(nil)

func NewAttachCommand() AttachCommand {
	return AttachCommand{}
}

type AttachCommand struct {
}

func (o AttachCommand) Run(ctx context.Context, inputOutput IO, s Session) error {

}
