package commands

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/fx"
	"github.com/SeungKang/memshonk/internal/shell"
)

const (
	MrunCommandName = "mrun"
)

func NewMrunCommand(config apicompat.NewCommandConfig) *fx.Command {
	cmd := MrunCommand{
		session: config.Session,
		stdin:   config.Stdin,
	}

	root := fx.NewCommand(MrunCommandName, "run a memshonk script", cmd.run)

	root.OptLongDesc = `EXAMPLES
  $ cat memshonk-script-example.sh
  #!/bin/memshonk

  set -eu

  attach

  rw="$(vmmap -f | grep ' rw')"

  echo "here are the first 16 bytes of all the rw regions:"

  (
  IFS=$'\n'

  for l in ${rw}
  do
    start=$(echo ${l} | cut -d '-' -f 1)

    echo "region: ${l}"
    echo -n "    "

    readm -a "${start}" -d raw -s 16
  done
  )
`

	root.FlagSet.StringNf(&cmd.asFilePath, fx.ArgConfig{
		Name:        "script-file-path",
		Description: "File `path` to memshonk script to execute (use \"-\" to read from stdin)",
		Required:    true,
	})

	return root
}

type MrunCommand struct {
	session    apicompat.Session
	asFilePath string
	stdin      io.Reader
}

func (o *MrunCommand) run(ctx context.Context) (fx.CommandResult, error) {
	var src io.Reader
	var name string

	switch {
	case o.asFilePath == "-":
		src = o.stdin
		name = "(from-stdin)"
	case o.asFilePath != "":
		f, err := os.Open(o.asFilePath)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		src = f

		name = f.Name()
	default:
		return nil, fmt.Errorf("please specify a script to run")
	}

	interpreter, err := shell.NewInterpreter(o.session, apicompat.NewCommandHandler(o.session))
	if err != nil {
		return nil, fmt.Errorf("failed to create new shell - %w", err)
	}

	err = interpreter.ExecuteScript(ctx, src, name)
	if err != nil {
		return nil, fmt.Errorf("memshonk script %q - %w",
			name, err)
	}

	return nil, nil
}
