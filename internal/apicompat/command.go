package apicompat

import (
	"container/list"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"syscall"

	"github.com/SeungKang/memshonk/internal/fx"
	"github.com/SeungKang/memshonk/internal/jobsctl"
)

type CommandResult interface {
	Serialize() []byte
}

// NewEmptyCommandRegistry creates a new empty command registry.
func NewEmptyCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		byName:  make(map[string]func(NewCommandConfig) *fx.Command),
		aliases: make(map[string]string),
	}
}

// CommandRegistry stores command schemas and allows lookup by name or alias.
// It is safe for concurrent read access after initialization.
type CommandRegistry struct {
	rwMu    sync.RWMutex
	byName  map[string]func(NewCommandConfig) *fx.Command
	names   []string
	aliases map[string]string // alias -> canonical name
}

type NewCommandConfig struct {
	Session Session
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

// Register adds a command schema to the registry.
func (o *CommandRegistry) Register(name string, newCommandFn func(NewCommandConfig) *fx.Command) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	o.byName[name] = newCommandFn
	o.names = append(o.names, name)
}

// Unregister removes a command from the registry by name.
func (o *CommandRegistry) Unregister(name string) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	_, ok := o.byName[name]
	if !ok {
		return
	}

	delete(o.byName, name)

	// Remove from names slice
	for i, n := range o.names {
		if n == name {
			o.names = append(o.names[:i], o.names[i+1:]...)
			break
		}
	}

	// Remove aliases
	//for _, alias := range schema.Aliases {
	//	delete(o.aliases, alias)
	//}
}

// Lookup finds a command schema by name or alias.
//
// Returns the schema and true if found, or nil and false if not.
func (o *CommandRegistry) Lookup(nameOrAlias string) (func(NewCommandConfig) *fx.Command, bool) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	// Try direct name lookup first
	if schema, ok := o.byName[nameOrAlias]; ok {
		return schema, true
	}

	// Try alias lookup
	if canonicalName, ok := o.aliases[nameOrAlias]; ok {
		if schema, ok := o.byName[canonicalName]; ok {
			return schema, true
		}
	}

	return nil, false
}

// AsSlice returns a fx.Command instance for each registered command.
func (o *CommandRegistry) AsSlice(session Session) []*fx.Command {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	cmds := make([]*fx.Command, 0, len(o.names))

	for _, newCmdFn := range o.byName {
		cmds = append(cmds, newCmdFn(NewCommandConfig{Session: session}))
	}

	sort.SliceStable(cmds, func(i, j int) bool {
		return cmds[i].Name() < cmds[j].Name()
	})

	return cmds
}

// Names returns all registered command names (not aliases).
func (o *CommandRegistry) Names() []string {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	result := make([]string, len(o.names))
	copy(result, o.names)
	return result
}

// AllNamesAndAliases returns all command names and aliases for completion.
func (o *CommandRegistry) AllNamesAndAliases() []string {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	result := make([]string, 0, len(o.names)+len(o.aliases))
	result = append(result, o.names...)
	for alias := range o.aliases {
		result = append(result, alias)
	}

	return result
}

// CommandStorage stores information about the session's previously-run
// commands.
type CommandStorage struct {
	rwMu           sync.RWMutex
	namesToOutputs map[string]*list.List
}

func (o *CommandStorage) AddOutput(output fx.CommandResultWrapper) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if o.namesToOutputs == nil {
		o.namesToOutputs = make(map[string]*list.List)
	}

	id := serializeCommandID(output.Commands)

	outputs, hasIt := o.namesToOutputs[id]
	if !hasIt {
		outputs = list.New()

		o.namesToOutputs[id] = outputs
	}

	if outputs.Len() == 2 {
		outputs.Remove(outputs.Back())
	}

	outputs.PushFront(output)
}

func (o *CommandStorage) PreviousOutput(commandID []string) (fx.CommandResultWrapper, bool) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	id := serializeCommandID(commandID)

	outputs, hasAny := o.namesToOutputs[id]
	if !hasAny {
		return fx.CommandResultWrapper{}, false
	}

	output := outputs.Front().Value.(fx.CommandResultWrapper)

	return output, true
}

func serializeCommandID(eachCmd []string) string {
	return strings.Join(eachCmd, "-")
}

func NewCommandHandler(session Session) *CommandHandler {
	return &CommandHandler{
		session: session,
	}
}

type CommandHandler struct {
	session Session
}

type RunCommandConfig struct {
	Argv   []string
	Env    []string
	Cwd    string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func (o *CommandHandler) Run(ctx context.Context, config RunCommandConfig) *CommandHandlerError {
	if len(config.Argv) == 0 {
		return nil
	}

	wasHandled, err := o.runInternalCommand(ctx, config)
	if err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(config.Stderr, err.Error())
		}

		return err
	}

	if wasHandled {
		return nil
	}

	err = o.execProgram(ctx, config)
	if err != nil {
		_, hasExitStatus := err.HasExitStatus()
		if !hasExitStatus {
			fmt.Fprintln(config.Stderr, err.Error())
		}

		return err
	}

	return nil
}

func (o *CommandHandler) runInternalCommand(ctx context.Context, config RunCommandConfig) (bool, *CommandHandlerError) {
	newCmdFn, hasIt := o.session.SharedState().Commands.Lookup(config.Argv[0])
	if !hasIt {
		return false, nil
	}

	cmd := newCmdFn(NewCommandConfig{
		Session: o.session,
		Stdin:   config.Stdin,
		Stdout:  config.Stdout,
		Stderr:  config.Stderr,
	})

	ctx, job, err := o.session.Jobs().Register(ctx, jobsctl.RegisterConfig{
		Namespace: "memshonk",
		Argv:      config.Argv,
	})
	if err != nil {
		return false, NewCommandHandlerError(0, fmt.Errorf("failed to register internal command job - %w",
			err))
	}
	defer job.SetFinished()

	usageWriter := config.Stderr

	stdoutFd, stdoutIsFd := config.Stdout.(*os.File)
	if stdoutIsFd {
		info, _ := stdoutFd.Stat()
		if info != nil && info.Mode()&os.ModeNamedPipe > 0 {
			usageWriter = config.Stdout
		}
	}

	cmd.VisitAll(func(c *fx.Command) {
		c.FlagSet.Actual().SetOutput(usageWriter)
	})

	result, err := cmd.Run(ctx, config.Argv[1:])
	if err != nil {
		return true, NewCommandHandlerError(1, fmt.Errorf("%s failed: %w", cmd.Name(), err))
	}

	o.session.CommandStorage().AddOutput(result)

	if result.Result != nil {
		config.Stdout.Write([]byte(result.Result.Human()))
		config.Stdout.Write([]byte{'\n'})
	}

	return true, nil
}

func (o *CommandHandler) execProgram(ctx context.Context, config RunCommandConfig) *CommandHandlerError {
	ctx, job, err := o.session.Jobs().Register(ctx, jobsctl.RegisterConfig{
		Namespace: "program",
		Argv:      config.Argv,
	})
	if err != nil {
		return NewCommandHandlerError(0, fmt.Errorf("failed to register external program job - %w", err))
	}
	defer job.SetFinished()

	program := exec.CommandContext(ctx, config.Argv[0], config.Argv[1:]...)

	program.Dir = config.Cwd
	program.Env = config.Env

	program.Stdin = config.Stdin
	program.Stdout = config.Stdout
	program.Stderr = config.Stderr

	err = program.Start()
	if err != nil {
		return NewCommandHandlerError(0, fmt.Errorf("failed to exec new process - %w",
			err))
	}

	job.SetPID(program.Process.Pid)

	var exitErr *exec.ExitError

	err = program.Wait()
	switch {
	case err == nil:
		return nil
	case errors.As(err, &exitErr):
		// Based on code from the DefaultExecHandler
		// function in:
		// mvdan.cc/sh/v3/interp/handler.go
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			if status.Signaled() {
				if ctx.Err() != nil {
					return NewCommandHandlerError(1, err)
				}

				return NewCommandHandlerError(uint8(128+status.Signal()), err)
			}

			return NewCommandHandlerError(uint8(status.ExitStatus()), err)
		}

		fallthrough
	default:
		return NewCommandHandlerError(1, err)
	}
}

func NewCommandHandlerError(exitStatus uint8, err error) *CommandHandlerError {
	return &CommandHandlerError{
		err:    err,
		status: exitStatus,
	}
}

type CommandHandlerError struct {
	err    error
	status uint8
}

func (o CommandHandlerError) Unwrap() error {
	return o.err
}

func (o CommandHandlerError) Error() string {
	return o.err.Error()
}

func (o CommandHandlerError) HasExitStatus() (uint8, bool) {
	return o.status, o.status != 0
}
