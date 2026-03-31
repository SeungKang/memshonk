package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/commands"
	"github.com/SeungKang/memshonk/internal/events"
	"github.com/SeungKang/memshonk/internal/flagsctl"
	"github.com/SeungKang/memshonk/internal/globalconfig"
	"github.com/SeungKang/memshonk/internal/plugins"
	"github.com/SeungKang/memshonk/internal/plugins/pluginscompat"
	"github.com/SeungKang/memshonk/internal/plugins/pluginsctl"
	"github.com/SeungKang/memshonk/internal/progctl"
	"github.com/SeungKang/memshonk/internal/project"
	"github.com/SeungKang/memshonk/internal/sessiond"
	"github.com/SeungKang/memshonk/internal/shell"
	"github.com/SeungKang/memshonk/internal/shvars"
	"github.com/SeungKang/memshonk/internal/vendored/goterm"

	"golang.org/x/term"
)

const (
	appName = "memshonk"

	usage = `SYNOPSIS
  ` + appName + ` -` + helpArg + `
  ` + appName + ` [options] -` + exePathArg + ` EXECUTABLE-FILE-PATH
  ` + appName + ` [options] -` + projectPathArg + ` PROJECT-FILE-PATH

DESCRIPTION

OPTIONS
`

	helpArg        = "h"
	projectPathArg = "p"
	exePathArg     = "e"
	sessionIDArg   = "S"
)

func main() {
	log.SetFlags(0)

	err := mainWithError()
	if err != nil {
		log.Fatalln("fatal:", err)
	}
}

func mainWithError() error {
	isDaemon := false
	args := os.Args[1:]

	if len(args) > 0 && args[0] == "daemon" {
		isDaemon = true
		args = args[2:]
	}

	var state mainState

	help := flag.Bool(
		helpArg,
		false,
		"Display this information")

	flag.StringVar(
		&state.optProjectFilePath,
		projectPathArg,
		"",
		"Load a project file by its `path`")

	flag.StringVar(
		&state.optExePath,
		exePathArg,
		"",
		"Load the specified executable by its `path` and use an empty project ")

	flag.StringVar(
		&state.optSessionID,
		sessionIDArg,
		"",
		"Use a custom session `id`")

	flag.CommandLine.Parse(args)

	if *help {
		out := os.Stderr

		stdoutInfo, _ := os.Stdout.Stat()
		if stdoutInfo != nil && stdoutInfo.Mode()&os.ModeNamedPipe != 0 {
			out = os.Stdout
			flag.CommandLine.SetOutput(out)
		}

		out.WriteString(usage)
		flag.PrintDefaults()

		os.Exit(1)

		return nil
	}

	if flag.NArg() > 0 {
		return fmt.Errorf("unrecognized non-flag arguments were provided - verify that you provided the correct command-line arguments")
	}

	if state.optExePath == "" && state.optProjectFilePath == "" {
		return fmt.Errorf("please specify either a project path (-%s) or a program file path (-%s)",
			projectPathArg, exePathArg)
	}

	if state.optExePath != "" && state.optProjectFilePath != "" {
		return fmt.Errorf("both a project path (-%s) and an executable file path (-%s) cannot be specified together",
			projectPathArg, exePathArg)
	}

	state.globalVars = &shvars.Variables{}

	var err error

	state.globalConf, err = globalconfig.Setup()
	if err != nil {
		return fmt.Errorf("failed to setup global configuration - %w", err)
	}

	for _, keyValuePair := range os.Environ() {
		name, value, _ := strings.Cut(keyValuePair, "=")

		if varMayBeSecret(name) {
			os.Unsetenv(name)

			continue
		}

		state.globalVars.Set(shvars.Variable{
			Name:      name,
			Value:     value,
			Source:    shvars.ProcEnvVarsSrc,
			Immutable: true,
		})
	}

	projConfig := project.ProjectConfig{
		GlobalVars: state.globalVars,
		GlobalConf: state.globalConf,
	}

	if state.optExePath != "" {
		state.project, err = project.EmptyForExePath(state.optExePath, projConfig)
	} else {
		state.project, err = project.FromFilePath(state.optProjectFilePath, projConfig)
	}

	if err != nil {
		return fmt.Errorf("failed to setup project - %w", err)
	}

	state.wsConf = state.globalConf.ProjectWorkspaceConfig(state.project.Name())

	if isDaemon {
		return beDaemon(state)
	}

	return beClient(state)
}

func varMayBeSecret(name string) bool {
	lower := strings.ToLower(name)

	for _, secretLookingStr := range []string{"secret", "token", "password", "pass"} {
		if strings.Contains(lower, secretLookingStr) {
			return true
		}
	}

	return false
}

type mainState struct {
	globalConf globalconfig.Config
	globalVars *shvars.Variables
	wsConf     globalconfig.WorkspaceConfig
	project    *project.Project

	optProjectFilePath string
	optExePath         string
	optSessionID       string
}

func beClient(state mainState) error {
	terminal, err := goterm.NewStdioTerminal()
	if err != nil {
		return fmt.Errorf("failed to create new fd terminal - %w", err)
	}

	resizeEvents, stopResizeEventsFn := terminal.OnResize()
	defer stopResizeEventsFn()

	clientConfig := sessiond.ClientConfig{
		SocketPath:   state.wsConf.SocketFilePath,
		Stdin:        os.Stdin,
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
		OptSessionID: state.optSessionID,

		OptTerminalResizes: resizeEvents,
	}

	termState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to make raw - %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), termState)

	setupCtx, cancelFn := context.WithTimeout(context.Background(), time.Second)
	defer cancelFn()

	// Try to connect to an existing daemon process. If that
	// fails, try starting a new daemon process and connect
	// to the new  process.
	fmt.Fprint(os.Stderr, "connecting to daemon...\r\n")

	client, err := sessiond.SetupClient(setupCtx, clientConfig)
	if err != nil {
		cancelFn()

		fmt.Fprint(os.Stderr, "starting daemon...\r\n")

		setupCtx, cancelFn = context.WithTimeout(context.Background(), 2*time.Second)

		err = execDaemon(setupCtx)
		if err != nil {
			cancelFn()
			return fmt.Errorf("failed to start daemon - %w", err)
		}

		client, err = sessiond.SetupClient(setupCtx, clientConfig)
		cancelFn()
		if err != nil {
			return fmt.Errorf("failed to setup client after successfully starting daemon - %w", err)
		}
	}
	defer client.Close()

	<-client.Done()

	return client.Err()
}

func execDaemon(setupCtx context.Context) error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	tmpDirPath, err := os.MkdirTemp("", "memshonk-")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory for daemon setup - %w", err)
	}
	defer os.RemoveAll(tmpDirPath)

	initSocketPath := filepath.Join(tmpDirPath, "init.sock")

	initSocket, err := net.Listen("unix", initSocketPath)
	if err != nil {
		return fmt.Errorf("failed to create init daemon socket - %w", err)
	}
	defer initSocket.Close()

	args := make([]string, len(os.Args)+1)
	args[0] = "daemon"
	args[1] = "--"
	copy(args[2:], os.Args[1:])

	daemon := exec.Command(exePath, args...)

	daemon.SysProcAttr = sessiond.DaemonSysProcAttr()

	// TODO: grumble needs these to be set to fds connected to a terminal.
	// Remove once MEMSHONK_INIT_SOCKET_HACK is removed.
	daemon.Stdin = os.Stdin
	daemon.Stdout = os.Stdout
	daemon.Stderr = os.Stderr

	// TODO: Remove once MEMSHONK_INIT_SOCKET_HACK is removed.
	daemon.Env = os.Environ()
	daemon.Env = append(daemon.Env, "MEMSHONK_INIT_SOCKET_HACK="+initSocketPath)

	// TODO: Uncomment once MEMSHONK_INIT_SOCKET_HACK is removed.
	//stdout, err := daemon.StdoutPipe()
	//if err != nil {
	//	return fmt.Errorf("failed to create pipe for daemon's stdout - %w",
	//		err)
	//}
	//defer stdout.Close()

	err = daemon.Start()
	if err != nil {
		return fmt.Errorf("exec start failed (argv: %q) - %w",
			daemon.String(), err)
	}

	ready := make(chan struct{})
	waitErr := make(chan error, 1)

	go func() {
		conn, err := initSocket.Accept()
		if err != nil {
			waitErr <- err
		} else {
			conn.Close()
			close(ready)
		}
	}()

	// TODO: Uncomment once MEMSHONK_INIT_SOCKET_HACK is removed.
	// go func() {
	// 	scanner := bufio.NewScanner(stdout)

	// 	readFromStdout := scanner.Scan()

	// 	if !readFromStdout {
	// 		var err error
	// 		if scanner.Err() != nil {
	// 			err = scanner.Err()
	// 		} else {
	// 			err = errors.New("stdout scanner failed without error")
	// 		}

	// 		waitErr <- fmt.Errorf("failed to read from daemon's stdout - %w", err)

	// 		return
	// 	}

	// 	if scanner.Text() != "ready" {
	// 		waitErr <- fmt.Errorf("got unexpected ready string: %q",
	// 			scanner.Text())

	// 		return
	// 	}

	// 	close(ready)
	// }()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	defer signal.Stop(sigs)

	select {
	case s := <-sigs:
		err = fmt.Errorf("received signal while waiting for daemon to become ready: %q", s.String())
	case <-setupCtx.Done():
		err = fmt.Errorf("timed-out waiting for daemon process to become ready - %w", setupCtx.Err())
	case err := <-waitErr:
		err = fmt.Errorf("daemon process communication failed - %w", err)
	case <-ready:
		return nil
	}

	_ = daemon.Process.Kill()

	return err
}

func beDaemon(state mainState) error {
	ctx, cancelFn := signal.NotifyContext(context.Background(),
		syscall.SIGQUIT, syscall.SIGTERM)
	defer cancelFn()

	eventGroups := events.NewGroups()

	progCtl := progctl.NewCtl(state.project.General().ExePath, eventGroups)

	optPluginsCtl, err := maybeCreatePluginCtl(progCtl)
	if err != nil {
		return fmt.Errorf("failed to setup plugins - %w", err)
	}

	sharedState := apicompat.SharedState{
		Events:   eventGroups,
		Vars:     state.globalVars,
		Progctl:  progCtl,
		Project:  state.project,
		Commands: setupCommands(),
		Plugins:  optPluginsCtl,
		Flags:    flagsctl.New(),
	}

	server, err := sessiond.NewServer(ctx, sessiond.ServerConfig{
		SharedState: sharedState,
		NewShellFn: func(s apicompat.Session) (sessiond.Shell, error) {
			return shell.NewShell(s)
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create new session server - %w", err)
	}
	defer server.Close()

	// TODO: Should loading plugins happen before setting up the session server?
	if optPluginsCtl != nil {
		for _, pluginConfig := range state.project.Plugins().Libraries {
			plugin, err := optPluginsCtl.Load(pluginConfig)
			if err != nil {
				return fmt.Errorf("failed to load plugin: %q - %w",
					pluginConfig.FilePath, err)
			}

			commands.RegisterPlugin(plugin, sharedState.Commands)
		}
	}

	log.SetFlags(log.LstdFlags)

	// _, err = os.Stdout.WriteString("ready\n")
	// if err != nil {
	// 	return err
	// }

	// TODO: Remove once MEMSHONK_INIT_SOCKET_HACK is removed.
	initSocketPath := os.Getenv("MEMSHONK_INIT_SOCKET_HACK")
	if initSocketPath == "" {
		return errors.New("MEMSHONK_INIT_SOCKET_HACK is not set")
	}

	dialer := net.Dialer{}

	tmp, err := dialer.DialContext(ctx, "unix", initSocketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to init socket - %w", err)
	}
	tmp.Close()

	select {
	case <-ctx.Done():
	case <-server.Done():
	}

	return nil
}

func setupCommands() *apicompat.CommandRegistry {
	reg := apicompat.NewEmptyCommandRegistry()

	reg.Register(commands.HelpCommandName, commands.NewHelpCommand)
	reg.Register(commands.AttachCommandName, commands.NewAttachCommand)
	reg.Register(commands.DaemonCommandName, commands.NewDaemonCommand)
	reg.Register(commands.DetachCommandName, commands.NewDetachCommand)
	reg.Register(commands.FindCommandName, commands.NewFindCommand)
	reg.Register(commands.JobsCommandName, commands.NewJobsCommand)
	reg.Register(commands.PluginsCommandName, commands.NewPluginsCommand)
	reg.Register(commands.QuitCommandName, commands.NewQuitCommand)
	reg.Register(commands.ReadCommandName, commands.NewReadCommand)
	reg.Register(commands.SessionCommandName, commands.NewSessionCommand)
	reg.Register(commands.ShonksetCommandName, commands.NewShonksetCommand)
	reg.Register(commands.VmmapCommandName, commands.NewVmmapCommand)
	reg.Register(commands.WatchCommandName, commands.NewWatchCommand)
	reg.Register(commands.WriteCommandName, commands.NewWriteCommand)

	return reg
}

func maybeCreatePluginCtl(progCtl *progctl.Ctl) (plugins.Ctl, error) {
	pluginsCtl, err := pluginsctl.New(plugins.CtlConfig{
		Process: pluginscompat.WrapProcess(progCtl),
	})
	if err != nil {
		if errors.Is(err, plugins.ErrPluginsDisabled) {
			return nil, nil
		}

		return nil, err
	}

	return pluginsCtl, nil
}
