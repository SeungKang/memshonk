package commands

import (
	"errors"
)

var (
	errCommandNeedsTerminal = errors.New("this command requires a terminal, but the session does not provide a terminal")
)
