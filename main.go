package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/SeungKang/memshonk/internal/app"
	"github.com/SeungKang/memshonk/internal/commands"
	"github.com/SeungKang/memshonk/internal/events"
	"github.com/SeungKang/memshonk/internal/grsh"
	"github.com/SeungKang/memshonk/internal/plugins"
	"github.com/SeungKang/memshonk/internal/plugins/pluginscompat"
	"github.com/SeungKang/memshonk/internal/plugins/pluginsctl"
	"github.com/SeungKang/memshonk/internal/progctl"
	"github.com/SeungKang/memshonk/internal/project"
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

	projectFilePath := flag.Arg(0)
	if projectFilePath == "" {
		return errors.New("please specify a project file path as the last argument")
	}

	proj, err := project.FromFilePath(projectFilePath)
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

	application := app.NewApp(eventGroups, proj, progCtl, optPluginsCtl)

	session := application.NewSession(commands.IO{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})

	sh, err := grsh.NewShell(ctx, session)
	if err != nil {
		return err
	}

	if optPluginsCtl != nil {
		for _, pluginConfig := range proj.Plugins().Libraries {
			_, err := optPluginsCtl.Load(pluginConfig)
			if err != nil {
				return fmt.Errorf("failed to load plugin: %q - %w",
					pluginConfig.FilePath, err)
			}
		}
	}

	log.SetFlags(log.LstdFlags)

	return sh.Run()
}

func maybeCreatePluginCtl(progCtl *progctl.Ctl, eventGroups *events.Groups) (plugins.Ctl, error) {
	pluginsCtl, err := pluginsctl.New(plugins.CtlConfig{
		Events:  eventGroups,
		Process: pluginscompat.WrapProcess(progCtl),
	})
	if err != nil && !errors.Is(err, plugins.ErrPluginsDisabled) {
		if errors.Is(err, plugins.ErrPluginsDisabled) {
			return nil, nil
		}

		return nil, err
	}

	return pluginsCtl, nil
}
