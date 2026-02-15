package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/events"
	"github.com/SeungKang/memshonk/internal/globalconfig"
	"github.com/SeungKang/memshonk/internal/plugins"
	"github.com/SeungKang/memshonk/internal/plugins/pluginscompat"
	"github.com/SeungKang/memshonk/internal/plugins/pluginsctl"
	"github.com/SeungKang/memshonk/internal/progctl"
	"github.com/SeungKang/memshonk/internal/project"
	"github.com/SeungKang/memshonk/internal/sessiond"
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
)

func main() {
	log.SetFlags(0)

	err := mainWithError()
	if err != nil {
		log.Fatalln("fatal:", err)
	}
}

func mainWithError() error {
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

	flag.Parse()

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

	var err error

	state.globalConf, err = globalconfig.Setup()
	if err != nil {
		return fmt.Errorf("failed to setup global configuration - %w", err)
	}

	if state.optExePath != "" {
		state.project, err = project.Empty(state.optExePath, state.globalConf)
	} else {
		state.project, err = project.FromFilePath(state.optProjectFilePath, state.globalConf)
	}

	if err != nil {
		return fmt.Errorf("failed to setup project - %w", err)
	}

	state.wsConf = state.globalConf.ProjectWorkspaceConfig(state.project.Name())

	if sessiond.IsServerRunning(context.Background(), state.project.WorkspaceConfig().SocketFilePath) {
		return doClient(state)
	} else {
		return doServer(state)
	}
}

type mainState struct {
	globalConf globalconfig.Config
	wsConf     globalconfig.WorkspaceConfig
	project    *project.Project

	optProjectFilePath string
	optExePath         string
}

func doClient(state mainState) error {
	conn, err := net.Dial("unix", state.wsConf.SocketFilePath)
	if err != nil {
		return fmt.Errorf("failed to connect to server - %w", err)
	}
	defer conn.Close()

	client, err := sessiond.NewClient(context.Background(), sessiond.ClientConfig{
		ServerConn: conn,
		Stdin:      os.Stdin,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	})
	if err != nil {
		return fmt.Errorf("failed to create client - %w", err)
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

func doServer(state mainState) error {
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
		Progctl: progCtl,
		Project: state.project,
		Plugins: optPluginsCtl,
	}

	terminal, _ := goterm.NewStdioTerminal()

	server, err := sessiond.NewServer(ctx, sharedState)
	if err != nil {
		return fmt.Errorf("failed to create new session server - %w", err)
	}
	defer server.Close()

	session, err := server.NewSession(ctx, sessiond.SessionConfig{
		IsDefault: true,
		IO: apicompat.SessionIO{
			Stdin:       os.Stdin,
			Stdout:      os.Stdout,
			Stderr:      os.Stderr,
			OptTerminal: terminal,
		},
		OptID: "default",
	})
	if err != nil {
		return fmt.Errorf("failed to create default app session - %w", err)
	}

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

	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, syscall.SIGINT)
	defer signal.Stop(interrupts)

	go func() {
		for range interrupts {
			session.OnSignal(sessiond.IntSignalType)
		}
	}()

	log.SetFlags(log.LstdFlags)

	<-session.Done()

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
