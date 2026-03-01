package shell

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/SeungKang/memshonk/internal/apicompat"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// NewInterpreter creates a new shell interpreter.
func NewInterpreter(session apicompat.Session, cmdHandler *apicompat.CommandHandler) (*Interpreter, error) {
	i := &Interpreter{
		session: session,
		cmdHand: cmdHandler,
		parser:  syntax.NewParser(),
	}

	sio := session.IO()

	// Use an empty reader for stdin to avoid competing
	// with readline for input. Built-in commands don't
	// read from stdin, and external commands that need
	// stdin piping would require a different approach
	// (e.g., temporarily giving them exclusive access
	// to the terminal).
	emptyStdin := bytes.NewReader(nil)

	runner, err := interp.New(
		interp.StdIO(emptyStdin, sio.Stdout, sio.Stderr),
		interp.Env(expand.ListEnviron(session.SharedState().Vars.KeyValues(nil)...)),
		interp.ExecHandlers(i.execHandler),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create interpreter - %w", err)
	}

	i.runner = runner

	return i, nil
}

// Interpreter wraps mvdan/sh to provide shell interpretation with built-in
// command routing.
type Interpreter struct {
	session apicompat.Session
	cmdHand *apicompat.CommandHandler
	parser  *syntax.Parser
	runner  *interp.Runner
}

// Execute parses and executes a shell command line.
func (o *Interpreter) Execute(ctx context.Context, line string) error {
	// Parse the input
	file, err := o.parser.Parse(strings.NewReader(line), "")
	if err != nil {
		return fmt.Errorf("parse error - %w", err)
	}

	return o.runner.Run(ctx, file)
}

// execHandler routes command execution to built-in commands
// or external programs.
func (o *Interpreter) execHandler(interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	return func(ctx context.Context, argv []string) error {
		// interp.HandlerCtx is the prescribed mechanism
		// for accesing the stdin/out pipes managed by
		// the sh library.
		handlerCtx := interp.HandlerCtx(ctx)

		err := o.cmdHand.Run(ctx, apicompat.RunCommandConfig{
			Argv:   argv,
			Env:    execEnv(handlerCtx.Env),
			Cwd:    "",
			Stdin:  handlerCtx.Stdin,
			Stdout: handlerCtx.Stdout,
			Stderr: handlerCtx.Stderr,
		})
		if err != nil {
			exitStatus, hasIt := err.HasExitStatus()
			if !hasIt {
				exitStatus = 1
			}

			return interp.NewExitStatus(exitStatus)
		}

		return nil
	}
}

// execEnv is a copy of the private function of the same name from
// mvdan.cc/sh/v3, interp/vars.go
func execEnv(env expand.Environ) []string {
	list := make([]string, 0, 64)
	env.Each(func(name string, vr expand.Variable) bool {
		if !vr.IsSet() {
			// If a variable is set globally but unset in the
			// runner, we need to ensure it's not part of the final
			// list. Seems like zeroing the element is enough.
			// This is a linear search, but this scenario should be
			// rare, and the number of variables shouldn't be large.
			for i, kv := range list {
				if strings.HasPrefix(kv, name+"=") {
					list[i] = ""
				}
			}
		}
		if vr.Exported && vr.Kind == expand.String {
			list = append(list, name+"="+vr.String())
		}
		return true
	})
	return list
}
