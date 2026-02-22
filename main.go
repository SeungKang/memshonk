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
	"github.com/SeungKang/memshonk/internal/events"
	"github.com/SeungKang/memshonk/internal/globalconfig"
	"github.com/SeungKang/memshonk/internal/plugins"
	"github.com/SeungKang/memshonk/internal/plugins/pluginscompat"
	"github.com/SeungKang/memshonk/internal/plugins/pluginsctl"
	"github.com/SeungKang/memshonk/internal/progctl"
	"github.com/SeungKang/memshonk/internal/project"
	"github.com/SeungKang/memshonk/internal/sessiond"
	"github.com/SeungKang/memshonk/internal/shvars"
	"github.com/SeungKang/memshonk/internal/vendored/goterm"

	"golang.org/x/term"
)

const (
	appName = "memshonk"

	usage = `SYNOPSIS
  ` + appName + ` -` + helpArg + `
  ` + appName + ` [options]` + ` EXECUTABLE-PATH
  ` + appName + ` [options] -` + projectPathArg + ` PROJECT-FILE-PATH

DESCRIPTION

OPTIONS
`

	helpArg        = "h"
	projectPathArg = "p"
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
		"Use a project file instead of an empty project based on a program path")

	flag.StringVar(
		&state.optSessionID,
		sessionIDArg,
		"",
		"Use a custom session ID")

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

	state.optExePath = flag.Arg(0)

	if state.optExePath == "" && state.optProjectFilePath == "" {
		return fmt.Errorf("please specify either a project path (-%s) or a program file path (as the last non-flag argument)",
			projectPathArg)
	}

	if state.optExePath != "" && state.optProjectFilePath != "" {
		return fmt.Errorf("both a project path (-%s) and a program file path cannot be specified together",
			projectPathArg)
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

	setupCtx, cancelFn := context.WithTimeout(context.Background(), time.Second)
	defer cancelFn()

	// Try to connect to an existing daemon process. If that
	// fails, try starting a new daemon process and connect
	// to the new  process.
	client, err := sessiond.SetupClient(setupCtx, clientConfig)
	if err != nil {
		cancelFn()

		setupCtx, cancelFn = context.WithTimeout(context.Background(), 2*time.Second)

		err = execDaemon(setupCtx)
		if err != nil {
			return fmt.Errorf("failed to start daemon - %w", err)
		}

		client, err = sessiond.SetupClient(setupCtx, clientConfig)
		cancelFn()
		if err != nil {
			return fmt.Errorf("failed to setup client after successfully starting daemon - %w", err)
		}
	}
	defer client.Close()

	termState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to make raw - %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), termState)

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

	optPluginsCtl, err := maybeCreatePluginCtl(progCtl, eventGroups)
	if err != nil {
		return fmt.Errorf("failed to setup plugins - %w", err)
	}

	sharedState := apicompat.SharedState{
		Events:  eventGroups,
		Vars:    state.globalVars,
		Progctl: progCtl,
		Project: state.project,
		Plugins: optPluginsCtl,
	}

	server, err := sessiond.NewServer(ctx, sharedState)
	if err != nil {
		return fmt.Errorf("failed to create new session server - %w", err)
	}
	defer server.Close()

	// TODO: Should loading plugins happen before setting up the session server?
	if optPluginsCtl != nil {
		for _, pluginConfig := range state.project.Plugins().Libraries {
			_, err := optPluginsCtl.Load(pluginConfig)
			if err != nil {
				return fmt.Errorf("failed to load plugin: %q - %w",
					pluginConfig.FilePath, err)
			}
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

	<-ctx.Done()

	return nil
}

func maybeCreatePluginCtl(progCtl *progctl.Ctl, eventGroups *events.Groups) (plugins.Ctl, error) {
	pluginsCtl, err := pluginsctl.New(plugins.CtlConfig{
		Events:  eventGroups,
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
