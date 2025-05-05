package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/SeungKang/memshonk/internal/app"
	"github.com/SeungKang/memshonk/internal/commands"
	"log"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/desertbit/grumble"
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

	proj := &app.Project{} // TODO parse arguments and create a project
	application := app.NewApp(proj)
	session := application.NewSession()
	session.RunCommand(ctx, commands.NewAttachCommand())

	grumbleApp := grumble.New(&grumble.Config{
		Name:        "xmempg",
		Description: "Wrapper for mempg",
		Flags: func(f *grumble.Flags) {
			f.String(
				"E",
				"mempg-exe",
				"Path to mempg executable",
				"mempg")

			f.Bool(
				"D",
				"insecure-disable-sandbox",
				false,
				"Disable mempg sandbox")
		},
	})

	grumbleApp.SetInterruptHandler(func(a *grumble.App, count int) {
		a.Close()
	})

	sh := shell{
		ga:  grumbleApp,
		ctx: ctx,
	}

	grumbleApp.OnInit(sh.onInit)

	grumbleApp.OnClose(func() error {
	})

	grumbleApp.AddCommand(&grumble.Command{
		Name:    "seek",
		Aliases: []string{"s"},
		Help:    "set current address",
		Args: func(a *grumble.Args) {
			a.String("addr", "address to seek to")
		},
		Run: sh.seek,
	})

	grumbleApp.AddCommand(&grumble.Command{
		Name: "malloc",
		Help: "malloc n sized chunks",
		Flags: func(f *grumble.Flags) {
			f.Bool("s", "seek", false, "seek to the chunk address")
		},
		Args: func(a *grumble.Args) {
			a.Uint("size", "size of chunk", grumble.Default(uint(128)))
			a.Uint("n", "number of chunks", grumble.Default(uint(1)))
		},
		Run: sh.malloc,
	})

	grumbleApp.AddCommand(&grumble.Command{
		Name:    "read",
		Aliases: []string{"r"},
		Help:    "read n bytes from addr",
		Flags: func(f *grumble.Flags) {
			f.String("e", "encoding", "hexdump", "output encoding format")
		},
		Args: func(a *grumble.Args) {
			a.Uint("size", "number of bytes to read")
			a.String("addr", "address to read from", grumble.Default(""))
		},
		Run: sh.read,
	})

	grumbleApp.AddCommand(&grumble.Command{
		Name:    "write",
		Aliases: []string{"w"},
		Help:    "write value to addr",
		Flags: func(f *grumble.Flags) {
			f.String("e", "encoding", "raw", "input encoding format")
		},
		Args: func(a *grumble.Args) {
			a.String("data", "data to write")
			a.String("addr", "address to write to", grumble.Default(""))
		},
		Run: sh.write,
	})

	grumbleApp.AddCommand(&grumble.Command{
		Name: "flag",
		Help: "write the address of the flag function to stdout",
		Run:  sh.flag,
	})

	grumbleApp.AddCommand(&grumble.Command{
		Name:    "call",
		Aliases: []string{"c"},
		Help:    "execute the data at addr",
		Args: func(a *grumble.Args) {
			a.String("addr", "address of the data", grumble.Default(""))
		},
		Run: sh.call,
	})

	grumbleApp.AddCommand(&grumble.Command{
		Name: "free",
		Help: "free target addr",
		Flags: func(f *grumble.Flags) {
			f.Bool("a", "all", false, "free all allocated chunks")
		},
		Args: func(a *grumble.Args) {
			a.String("addr", "address of the data", grumble.Default(""))
		},
		Run: sh.free,
	})

	grumbleApp.AddCommand(&grumble.Command{
		Name: "stack",
		Help: "write the address of the stack to stdout",
		Run:  sh.stack,
	})

	grumbleApp.AddCommand(&grumble.Command{
		Name:    "heapstat",
		Aliases: []string{"hs"},
		Help:    "write a table of chunk stats",
		Run:     sh.heapstat,
	})

	// TODO: Default to "info", and change pid to "dead" if process stopped
	grumbleApp.AddCommand(&grumble.Command{
		Name: "mempg",
		Help: "mempg management commands",
		Args: func(a *grumble.Args) {
			a.String("manage", "manage the mempg process", grumble.Default(""))
		},
		Run: sh.mempg,
	})

	grumble.Main(grumbleApp)

	return nil
}

type shell struct {
	ga  *grumble.App
	pg  *mempg.Mempg
	fm  grumble.FlagMap
	ctx context.Context
}

func (o *shell) onInit(_ *grumble.App, flags grumble.FlagMap) error {
	if o.pg != nil {
		// don't bother with this error
		// if it is already dead an error will return
		o.pg.Kill()
	}

	var mempgArgs []string
	if flags.Bool("insecure-disable-sandbox") {
		mempgArgs = append(mempgArgs, "--insecure-disable-sandbox")
	}

	pg, err := mempg.Start(o.ctx, flags.String("mempg-exe"), mempgArgs...)
	if err != nil {
		return err
	}

	o.pg = pg
	o.fm = flags
	o.setPrompt()

	return nil
}

func (o *shell) seek(c *grumble.Context) error {
	addr, err := strconv.ParseUint(strings.TrimPrefix(c.Args.String("addr"), "0x"), 16, 64)
	if err != nil {
		return err
	}

	err = o.pg.Seek(uintptr(addr))
	if err != nil {
		return err
	}

	o.setPrompt()

	return nil
}

func (o *shell) malloc(c *grumble.Context) error {
	sizeInt := int(c.Args.Uint("size"))
	var lastMallocAddress uintptr
	for range c.Args.Uint("n") {
		chunkInfo, err := o.pg.Malloc(sizeInt)
		if err != nil {
			return err
		}

		lastMallocAddress = chunkInfo.Addr

		o.ga.Printf("0x%x (%s), num times reused: %d\n",
			chunkInfo.Addr,
			chunkInfo.Status,
			chunkInfo.NumAllocations-1)
	}

	if c.Flags.Bool("seek") {
		err := o.pg.Seek(lastMallocAddress)
		if err != nil {
			return err
		}

		o.setPrompt()
	}

	return nil
}

func (o *shell) setPrompt() {
	o.ga.SetPrompt(fmt.Sprintf("[0x%x] $ ", o.pg.CurrentSeekedAddr()))
}

func (o *shell) read(c *grumble.Context) error {
	var fmtFn func([]byte) error

	// TODO: Document encoding formats
	encodingFormat := c.Flags.String("encoding")
	switch encodingFormat {
	case "hexdump":
		fmtFn = func(b []byte) error {
			o.ga.Print(hex.Dump(b))
			return nil
		}
	case "hex":
		fmtFn = func(b []byte) error {
			o.ga.Println(hex.EncodeToString(b))
			return nil
		}
	case "b64", "base64":
		fmtFn = func(b []byte) error {
			o.ga.Println(base64.StdEncoding.EncodeToString(b))
			return nil
		}
	default:
		return fmt.Errorf("unknown encoding format: %q", encodingFormat)
	}

	addrStr := c.Args.String("addr")
	var memory []byte
	var err error
	if addrStr == "" {
		memory, err = o.pg.ReadMemoryAtSeekedAddr(c.Args.Uint("size"))
	} else {
		var addr uint64
		addr, err = strconv.ParseUint(strings.TrimPrefix(addrStr, "0x"), 16, 64)
		if err != nil {
			return err
		}

		memory, err = o.pg.ReadMemoryAtAddr(c.Args.Uint("size"), uintptr(addr))
	}

	if err != nil {
		return err
	}

	return fmtFn(memory)
}

func (o *shell) write(c *grumble.Context) error {
	dataStr := c.Args.String("data")
	var data []byte

	encodingFormat := c.Flags.String("encoding")
	switch encodingFormat {
	case "raw":
		data = []byte(dataStr)
	case "hexdump":
		return errors.New("TODO: someday invert hexdump -C output into bytes")
	case "hex":
		var err error
		data, err = hex.DecodeString(strings.TrimPrefix(dataStr, "0x"))
		if err != nil {
			return fmt.Errorf("failed to hex decode string - %w", err)
		}
	case "b64", "base64":
		var err error
		data, err = base64.StdEncoding.DecodeString(dataStr)
		if err != nil {
			return fmt.Errorf("failed to base64 decode string - %w", err)
		}
	case "ptr", "pointer":
		var err error
		data, err = hex.DecodeString(strings.TrimPrefix(dataStr, "0x"))
		if err != nil {
			return fmt.Errorf("failed to hex decode string - %w", err)
		}

		if len(data) > 8 {
			return fmt.Errorf("pointer cannot be greater than 8 bytes, got %d", len(data))
		}

		switch {
		case len(data) > 8:
			return fmt.Errorf("pointer cannot be greater than 8 bytes, got %d", len(data))
		case len(data) < 8:
			data = append(bytes.Repeat([]byte{0}, 8-len(data)), data...)
		}

		binary.LittleEndian.PutUint64(data, binary.BigEndian.Uint64(data))
	default:
		return fmt.Errorf("unknown encoding format: %q", encodingFormat)
	}

	addrStr := c.Args.String("addr")
	var err error
	if addrStr == "" {
		err = o.pg.WriteMemoryAtSeekedAddr(data)
	} else {
		var addr uint64
		addr, err = strconv.ParseUint(strings.TrimPrefix(addrStr, "0x"), 16, 64)
		if err != nil {
			return err
		}

		err = o.pg.WriteMemoryAtAddr(data, uintptr(addr))
	}

	if err != nil {
		return err
	}

	return nil
}

func (o *shell) flag(*grumble.Context) error {
	addr, err := o.pg.Flag()
	if err != nil {
		return err
	}

	o.ga.Printf("0x%x\n", addr)

	return nil
}

func (o *shell) call(c *grumble.Context) error {
	addrStr := c.Args.String("addr")
	var out []byte
	var err error
	if addrStr == "" {
		out, err = o.pg.CallSeekedAddr()
	} else {
		var addr uint64
		addr, err = strconv.ParseUint(strings.TrimPrefix(addrStr, "0x"), 16, 64)
		if err != nil {
			return err
		}

		out, err = o.pg.CallAddr(uintptr(addr))
	}

	if err != nil {
		return err
	}

	o.ga.Print(string(out))

	return nil
}

func (o *shell) free(c *grumble.Context) error {
	if c.Flags.Bool("all") {
		err := o.pg.FreeAll()
		if err != nil {
			return err
		}

		return nil
	}

	addrStr := c.Args.String("addr")
	var out []byte
	var err error
	if addrStr == "" {
		err = o.pg.FreeSeekedAddr()
	} else {
		var addr uint64
		addr, err = strconv.ParseUint(strings.TrimPrefix(addrStr, "0x"), 16, 64)
		if err != nil {
			return err
		}

		err = o.pg.FreeAddr(uintptr(addr))
	}

	if err != nil {
		return err
	}

	o.ga.Print(string(out))

	return nil
}

func (o *shell) stack(c *grumble.Context) error {
	addr, err := o.pg.StackAddr()
	if err != nil {
		return err
	}

	o.ga.Printf("0x%x\n", addr)

	return nil
}

func (o *shell) heapstat(c *grumble.Context) error {
	statTable, err := o.pg.HeapStat()
	if err != nil {
		return err
	}

	o.ga.Print(statTable)

	return nil
}

func (o *shell) mempg(c *grumble.Context) error {
	manageArg := c.Args.String("manage")
	status, execCmd := o.pg.Info()
	switch manageArg {
	case "info":
		o.ga.Printf("status: %s\n", status)
		o.ga.Printf("pid: %d\n", execCmd.Process.Pid)
	case "restart":
		if status == "running" {
			o.ga.Println("process is already running")
			return nil
		}

		o.ga.Println("restarting...")
		err := o.onInit(o.ga, o.fm)
		if err != nil {
			return err
		}
	}
	return nil
}
