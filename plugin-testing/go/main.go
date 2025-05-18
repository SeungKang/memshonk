package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/SeungKang/memshonk/internal/plugins"
	"github.com/SeungKang/memshonk/internal/plugins/libplugin"
)

func main() {
	log.SetFlags(0)

	err := mainWithError()
	if err != nil {
		log.Fatalln("fatal:", err)
	}
}

func mainWithError() error {
	pluginCtl, err := libplugin.NewCtl(plugins.CtlConfig{
		Process: &fakeProcess{},
	})
	if err != nil {
		return err
	}

	plugin, err := pluginCtl.Load(
		"/home/u/libmemshonk_plugin.so",
	)
	if err != nil {
		return err
	}

	_ = plugin

	fmt.Println(pluginCtl.PrettyString(""))

	parser, hasIt := plugin.Parser("parse_enemies")
	if !hasIt {
		return fmt.Errorf("does not have it")
	}

	b, err := parser.Run(0x00)
	if err != nil {
		return fmt.Errorf("parser failed - %w", err)
	}

	log.Println(hex.Dump(b))

	return nil
}

type fakeProcess struct {
}

func (o fakeProcess) ReadFromAddr(addr uintptr, size uint64) ([]byte, error) {
	return bytes.Repeat([]byte{0x41}, int(size)), nil
}
