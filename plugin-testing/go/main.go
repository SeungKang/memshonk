package main

import (
	"bytes"
	"context"
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

	log.Println("loading plugin...")

	plugin, err := pluginCtl.Load(plugins.PluginConfig{
		FilePath:     "/home/u/libmemshonk_plugin.so",
		ExecOnReload: []string{"echo", "hello"},
	})
	if err != nil {
		return err
	}

	fmt.Println(pluginCtl.PrettyString(""))

	b, err := plugin.RunParser("parse_enemies", 0x00)
	if err != nil {
		return fmt.Errorf("parser failed - %w", err)
	}

	fmt.Println(hex.Dump(b))

	log.Println("reloading plugin...")

	err = pluginCtl.Reload(context.Background(), plugin.Name())
	if err != nil {
		return err
	}

	fmt.Println(pluginCtl.PrettyString(""))

	log.Println("unloading plugin...")

	err = pluginCtl.Unload(plugin.Name())
	if err != nil {
		return err
	}

	fmt.Println(pluginCtl.PrettyString(""))

	return nil
}

type fakeProcess struct {
}

func (o fakeProcess) ReadFromAddr(addr uintptr, size uint64) ([]byte, error) {
	return bytes.Repeat([]byte{0x41}, int(size)), nil
}
