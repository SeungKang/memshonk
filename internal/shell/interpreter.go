package shell

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/commands"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// NewInterpreter creates a new shell interpreter.
func NewInterpreter(session apicompat.Session, registry *CommandRegistry) (*Interpreter, error) {
	i := &Interpreter{
		session:  session,
		registry: registry,
		parser:   syntax.NewParser(),
	}

	sio := session.IO()

	// Use an empty reader for stdin to avoid competing with readline for input.
	// Built-in commands don't read from stdin, and external commands that need
	// stdin piping would require a different approach (e.g., temporarily giving
	// them exclusive access to the terminal).
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

// Interpreter wraps mvdan/sh to provide shell interpretation with built-in command routing.
type Interpreter struct {
	session  apicompat.Session
	registry *CommandRegistry
	parser   *syntax.Parser
	runner   *interp.Runner
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

// execHandler routes command execution to built-in commands or external commands.
func (o *Interpreter) execHandler(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	return func(ctx context.Context, args []string) error {
		if len(args) == 0 {
			return nil
		}

		cmdName := args[0]

		// Check if it's a built-in command
		schema, found := o.registry.Lookup(cmdName)
		if found {
			return o.runBuiltin(ctx, schema, args[1:])
		}

		// Fall back to external command execution
		return next(ctx, args)
	}
}

// runBuiltin executes a built-in command.
func (o *Interpreter) runBuiltin(ctx context.Context, schema commands.CommandSchema, args []string) error {
	// Parse arguments according to the schema
	parser := NewArgParser(schema)
	config, err := parser.Parse(args)
	if err != nil {
		return fmt.Errorf("argument error - %w", err)
	}

	// Create the command
	cmd, err := schema.CreateFn(config)
	if err != nil {
		return fmt.Errorf("failed to create command - %w", err)
	}

	// Run the command through the session
	return o.session.RunCommand(ctx, cmd)
}

// ExecuteSimple executes a simple command without full shell parsing.
// This is useful for built-in commands that don't need shell features.
func (o *Interpreter) ExecuteSimple(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return nil
	}

	cmdName := args[0]

	// Check if it's a built-in command
	schema, found := o.registry.Lookup(cmdName)
	if found {
		return o.runBuiltin(ctx, schema, args[1:])
	}

	// For simple execution, we don't support external commands
	return fmt.Errorf("unknown command - %s", cmdName)
}

// PrintHelp prints help for a command or all commands.
func (o *Interpreter) PrintHelp(w io.Writer, cmdName string) {
	if cmdName == "" {
		// Print all commands
		fmt.Fprintln(w, "Available commands:")
		for _, name := range o.registry.Names() {
			schema, _ := o.registry.Lookup(name)
			fmt.Fprintf(w, "  %-15s %s\n", name, schema.ShortHelp)
		}
		return
	}

	// Print help for specific command
	schema, found := o.registry.Lookup(cmdName)
	if !found {
		fmt.Fprintf(w, "Unknown command - %s\n", cmdName)
		return
	}

	fmt.Fprintf(w, "%s - %s\n", schema.Name, schema.ShortHelp)
	if schema.LongHelp != "" {
		fmt.Fprintf(w, "\n%s\n", schema.LongHelp)
	}

	if len(schema.Aliases) > 0 {
		fmt.Fprintf(w, "\nAliases - %s\n", strings.Join(schema.Aliases, ", "))
	}

	if len(schema.Flags) > 0 {
		fmt.Fprintln(w, "\nFlags:")
		for _, f := range schema.Flags {
			if f.Short != "" {
				fmt.Fprintf(w, "  -%s, --%s  %s\n", f.Short, f.Long, f.Desc)
			} else {
				fmt.Fprintf(w, "      --%s  %s\n", f.Long, f.Desc)
			}
		}
	}

	if len(schema.NonFlags) > 0 {
		fmt.Fprintln(w, "\nArguments:")
		for _, nf := range schema.NonFlags {
			fmt.Fprintf(w, "  %-15s %s\n", nf.Name, nf.Desc)
		}
	}
}
