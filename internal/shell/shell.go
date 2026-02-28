package shell

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/commands"
	"github.com/SeungKang/memshonk/internal/events"
	"github.com/SeungKang/memshonk/internal/plugins"
	"github.com/SeungKang/memshonk/internal/progctl"

	"github.com/chzyer/readline"
	"github.com/fatih/color"
)

// NewShell creates a new shell for the given session.
func NewShell(session apicompat.Session) (*Shell, error) {
	terminal, hasTerm := session.Terminal()
	if !hasTerm {
		return nil, fmt.Errorf("the current session does not provide a terminal, which is required for shell functionality")
	}

	// Create the command registry
	registry := NewCommandRegistry()
	for _, cmdSchema := range commands.BuiltinCommands() {
		registry.Register(cmdSchema)
	}

	// Build readline configuration for virtual terminal
	readlineConfig := buildReadlineConfig(readlineIO{
		Stdin:    io.NopCloser(session.IO().Stdin),
		Stdout:   session.IO().Stdout,
		Stderr:   session.IO().Stderr,
		Terminal: terminal,
	})
	readlineConfig.AutoComplete = NewCompleter(registry)

	// History setup
	wsConfig := session.SharedState().Project.WorkspaceConfig()
	historyFilePath, historyEnabled := wsConfig.HistoryFilePath(session.Info().ID)
	if historyEnabled {
		readlineConfig.HistoryFile = historyFilePath
	}

	// Create readline instance
	readLine, err := readline.NewEx(readlineConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create readline - %w", err)
	}

	interpreter, err := NewInterpreter(session, apicompat.NewCommandHandler(session), registry)
	if err != nil {
		return nil, fmt.Errorf("failed to create interpreter - %w", err)
	}

	return &Shell{
		session:  session,
		rl:       readLine,
		interp:   interpreter,
		registry: registry,
		colorFn:  color.New(color.FgCyan).SprintFunc(),
		prompt:   "$ ",
	}, nil
}

// Shell provides an interactive command-line shell with readline support
// and mvdan/sh shell interpretation.
type Shell struct {
	session  apicompat.Session
	rl       *readline.Instance
	interp   *Interpreter
	registry *CommandRegistry
	colorFn  func(a ...interface{}) string
	promptMu sync.RWMutex
	prompt   string
	ctx      context.Context
	cancelFn func()

	cancelCmdCtxFnMu sync.Mutex
	cancelCmdCtxFn   func()

	attachEvents   *events.Sub[progctl.AttachedEvent]
	detachEvents   *events.Sub[progctl.DetachedEvent]
	exitedEvents   *events.Sub[progctl.ProcessExitedEvent]
	loadedEvents   *events.Sub[plugins.LoadedEvent]
	unloadedEvents *events.Sub[plugins.UnloadedEvent]
}

func (o *Shell) Signal(interface{}) {
	o.cancelCmdCtxFnMu.Lock()
	defer o.cancelCmdCtxFnMu.Unlock()

	if o.cancelCmdCtxFn != nil {
		o.cancelCmdCtxFn()

		o.cancelCmdCtxFn = nil
	}
}

func (o *Shell) Close() error {
	o.cancelFn()

	return nil
}

// Run starts the shell's REPL loop.
func (o *Shell) Run(ctx context.Context) error {
	defer o.rl.Close()

	ctx, o.cancelFn = context.WithCancel(ctx)

	// Initialize prompt based on current state
	o.initPrompt(ctx)

	eventGroups := o.session.SharedState().Events

	o.attachEvents = events.NewSubscriber[progctl.AttachedEvent](eventGroups)
	defer o.attachEvents.Unsubscribe()

	o.detachEvents = events.NewSubscriber[progctl.DetachedEvent](eventGroups)
	defer o.detachEvents.Unsubscribe()

	o.exitedEvents = events.NewSubscriber[progctl.ProcessExitedEvent](eventGroups)
	defer o.exitedEvents.Unsubscribe()

	o.loadedEvents = events.NewSubscriber[plugins.LoadedEvent](eventGroups)
	defer o.loadedEvents.Unsubscribe()

	o.unloadedEvents = events.NewSubscriber[plugins.UnloadedEvent](eventGroups)
	defer o.unloadedEvents.Unsubscribe()

	go o.handleEvents(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		o.promptMu.RLock()
		prompt := o.prompt
		o.promptMu.RUnlock()

		o.rl.SetPrompt(prompt)

		line, err := o.rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				// Handle Ctrl+C - just continue to next prompt
				continue
			}
			if err == io.EOF {
				// Handle Ctrl+D - exit gracefully
				return nil
			}
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Handle built-in shell commands
		switch o.handleBuiltinShellCommand(line) {
		case builtinHandled:
			continue
		case builtinExit:
			return nil
		case builtinNotOne:
			// Keep going.
		}

		// Create a context.Context for only *this* shell
		// command execution so that backgrounded shell
		// jobs are not cancelled if the shell is signaled.
		cmdCtx, cancelFn := context.WithCancel(ctx)

		o.cancelCmdCtxFnMu.Lock()
		o.cancelCmdCtxFn = cancelFn
		o.cancelCmdCtxFnMu.Unlock()

		// Execute through interpreter
		_ = o.interp.Execute(cmdCtx, line)
	}
}

type handleBuiltinShellCommandResult uint8

const (
	builtinHandled handleBuiltinShellCommandResult = iota
	builtinExit
	builtinNotOne
)

// handleBuiltinShellCommand handles special shell commands like help, exit.
// Returns true if the command was handled.
func (o *Shell) handleBuiltinShellCommand(line string) handleBuiltinShellCommandResult {
	words := strings.Fields(line)
	if len(words) == 0 {
		return builtinNotOne
	}

	switch words[0] {
	case "help":
		cmdName := ""
		if len(words) > 1 {
			cmdName = words[1]
		}

		o.interp.PrintHelp(o.session.IO().Stdout, cmdName)

		return builtinHandled
	case "exit", "quit":
		return builtinExit
	default:
		return builtinNotOne
	}
}

// initPrompt sets the initial prompt based on current state.
func (o *Shell) initPrompt(ctx context.Context) {
	info, err := o.session.SharedState().Progctl.ProcessInfo(ctx)
	if err == nil {
		o.setPrompt(info.PID)
	} else {
		o.setPrompt(0)
	}
}

// setPrompt updates the shell prompt.
func (o *Shell) setPrompt(pid int) {
	o.promptMu.Lock()
	defer o.promptMu.Unlock()

	if pid == 0 {
		o.prompt = o.colorFn(fmt.Sprintf("(%s) $ ", o.session.Info().ID))
	} else {
		o.prompt = o.colorFn(fmt.Sprintf("(%s) [%d] $ ", o.session.Info().ID, pid))
	}
}

// handleEvents handles lifecycle events from the application.
func (o *Shell) handleEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case e := <-o.attachEvents.RecvCh():
			o.setPrompt(e.ProcessInfo.PID)
			e.Acker().Ack()
		case e := <-o.detachEvents.RecvCh():
			o.setPrompt(0)
			e.Acker().Ack()
		case e := <-o.exitedEvents.RecvCh():
			o.setPrompt(0)
			log.Printf("process exited - %v", e.Reason)
		case e := <-o.loadedEvents.RecvCh():
			o.registerPlugin(e.Plugin)
		case e := <-o.unloadedEvents.RecvCh():
			o.registry.Unregister(e.Plugin.Name())
		}
	}
}

// registerPlugin registers plugin commands in the registry.
func (o *Shell) registerPlugin(plugin plugins.Plugin) {
	// Create a command schema for the plugin
	schema := commands.CommandSchema{
		Name:      plugin.Name(),
		ShortHelp: plugin.Description(),
		CreateFn: func(cfg commands.CommandConfig) (apicompat.Command, error) {
			// Plugin commands are handled specially
			return nil, fmt.Errorf("plugin commands should use plugin execution path")
		},
	}

	o.registry.Register(schema)

	// TODO: Register subcommands for plugin commands and parsers
	// This requires extending the registry to support hierarchical commands
}

// Registry returns the command registry for external use.
func (o *Shell) Registry() *CommandRegistry {
	return o.registry
}
