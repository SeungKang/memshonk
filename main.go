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
	"github.com/SeungKang/memshonk/internal/grsh"
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
		syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	defer cancelFn()

	application := app.NewApp(proj)

	session := application.NewSession(commands.IO{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})

	sh, err := grsh.NewShell(ctx, session)
	if err != nil {
		return err
	}

	return sh.Run()
}
