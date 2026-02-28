package shell

import (
	"io"

	"github.com/SeungKang/memshonk/internal/vendored/goterm"

	"github.com/chzyer/readline"
)

type readlineIO struct {
	Stdin    io.ReadCloser
	Stdout   io.Writer
	Stderr   io.Writer
	Terminal goterm.TerminalWithNotifications
}

// buildReadlineConfig creates a readline.Config configured for virtual terminal I/O.
// This avoids the global state issues in the readline library by providing explicit
// I/O streams and terminal function overrides for each session.
func buildReadlineConfig(rlIO readlineIO) *readline.Config {
	return &readline.Config{
		Stdin:  rlIO.Stdin,
		Stdout: rlIO.Stdout,
		Stderr: rlIO.Stderr,

		// Force interactive mode even though we're not connected to a real TTY.
		FuncIsTerminal:      func() bool { return true },
		ForceUseInteractive: true,

		// No-op raw mode functions since we're using a virtual terminal.
		// The actual terminal handling is done by the client.
		FuncMakeRaw: func() error { return nil },
		FuncExitRaw: func() error { return nil },

		FuncGetWidth: func() int {
			size, err := rlIO.Terminal.Size()
			if err != nil {
				return 80 // fallback
			}
			return size.Cols
		},

		// Set up per-session resize notification.
		// This avoids the global SIGWINCH handler race condition in readline.
		FuncOnWidthChanged: func(callback func()) {
			setupWidthChangeHandler(rlIO.Terminal, callback)
		},
	}
}

// setupWidthChangeHandler sets up a per-session width change handler
// that subscribes to the terminal's resize notifications.
func setupWidthChangeHandler(term goterm.TerminalWithNotifications, callback func()) {
	resizeCh, cancelFn := term.OnResize()

	go func() {
		defer cancelFn()
		for range resizeCh {
			callback()
		}
	}()
}
