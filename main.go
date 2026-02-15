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

DESCRIPTION

OPTIONS
`

	helpArg = "h"
)

func main() {
	log.SetFlags(0)

	err := mainWithError()
	if err != nil {
		log.Fatalln("fatal:", err)
	}
}

func mainWithError() error {
	help := flag.Bool(
		helpArg,
		false,
		"Display this information")

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

	firstArg := flag.Arg(0)
	switch firstArg {
	case "client":
		return doClient()
	case "":
		return errors.New("please specify a project file path as the last argument")
	default:
		return doServer(firstArg)
	}
}

func doClient() error {
	projectName := flag.Arg(1)
	if projectName == "" {
		return errors.New("please specify a project name as the last argument")
	}

	globalConfig, err := globalconfig.Setup()
	if err != nil {
		return fmt.Errorf("failed to setup global config - %w", err)
	}

	wsConfig, err := globalConfig.SetupWorkspace(projectName)
	if err != nil {
		return fmt.Errorf("failed to setup workspace - %w", err)
	}

	conn, err := net.Dial("unix", wsConfig.SocketFilePath)
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

	state, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to make raw - %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), state)

	<-client.Done()

	return client.Err()
}

func doServer(projectFilePath string) error {
	globalConfig, err := globalconfig.Setup()
	if err != nil {
		return fmt.Errorf("failed to setup global config - %w", err)
	}

	proj, err := project.FromFilePath(projectFilePath, globalConfig)
	if err != nil {
		return fmt.Errorf("failed to setup project - %w", err)
	}

	ctx, cancelFn := signal.NotifyContext(context.Background(),
		syscall.SIGQUIT, syscall.SIGTERM)
	defer cancelFn()

	eventGroups := events.NewGroups()

	progCtl := progctl.NewCtl(proj.General().ExeName, eventGroups)

	optPluginsCtl, err := maybeCreatePluginCtl(progCtl, eventGroups)
	if err != nil {
		return fmt.Errorf("failed to setup plugins - %w", err)
	}

	sharedState := apicompat.SharedState{
		Events:  eventGroups,
		Progctl: progCtl,
		Project: proj,
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
		for _, pluginConfig := range proj.Plugins().Libraries {
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
