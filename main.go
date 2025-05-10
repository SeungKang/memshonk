package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/SeungKang/memshonk/internal/app"
	"github.com/SeungKang/memshonk/internal/commands"
	"github.com/SeungKang/memshonk/internal/grsh"
)

func main() {
	log.SetFlags(0)

	err := mainWithError()
	if err != nil {
		log.Fatalln("fatal:", err)
	}
}

func mainWithError() error {
	ctx, cancelFn := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	defer cancelFn()

	proj := &app.Project{ExeName: "MassEffect3.exe"} // TODO parse arguments and create a project

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
