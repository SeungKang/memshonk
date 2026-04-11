package libplugin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/SeungKang/memshonk/internal/dl"
	"github.com/SeungKang/memshonk/internal/plugins"
)

type Plugin struct {
	config      plugins.PluginConfig
	filePath    string
	name        string
	loadedAt    time.Time
	version     uint32
	desc        string
	callbacks   *goCallbacks
	allocFn     func(uint32) uintptr
	freeMemFn   func(uintptr)
	optNewCtxFn func(*uintptr, *uintptr)
	optCanCtxFn func(uintptr)
	optUnloadFn func()
	optDebugFn  func(bool)
	parsers     []*parser
	commands    []*command
	unloadRwMu  sync.RWMutex
	unloaded    bool
	lib         *dl.Library
}

func (o *Plugin) loadParsers(symbols []pluginSymbolInfo) error {
	parsers := make([]*parser, len(symbols))

	for i, sym := range symbols {
		par := &parser{
			name:      sym.finalName,
			freeBufFn: o.Free,
			newCtxFn:  o.maybeNewCtx,
		}

		if o.optUnloadFn != nil {
			par.parentMu = &o.unloadRwMu
			par.parentUnl = &o.unloaded
		}

		err := o.lib.Func(sym.symName, &par.parseFn)
		if err != nil {
			return fmt.Errorf("failed to find parser fn %q - %w",
				sym.symName, err)
		}

		parsers[i] = par
	}

	o.parsers = parsers

	return nil
}

func (o *Plugin) loadCommands(symbols []pluginSymbolInfo) error {
	commands := make([]*command, len(symbols))

	for i, sym := range symbols {
		cmd := &command{
			name:      sym.finalName,
			allocFn:   o.allocFn,
			freeBufFn: o.Free,
			newCtxFn:  o.maybeNewCtx,
		}

		if o.optUnloadFn != nil {
			cmd.parentMu = &o.unloadRwMu
			cmd.parentUnl = &o.unloaded
		}

		err := o.lib.Func(sym.symName, &cmd.commandFn)
		if err != nil {
			return fmt.Errorf("failed to find command fn %q - %w",
				sym.symName, err)
		}

		commands[i] = cmd
	}

	o.commands = commands

	return nil
}

func (o *Plugin) PrettyString(indent string) string {
	buf := bytes.Buffer{}

	if indent != "" {
		buf.WriteString(indent)
	}
	buf.WriteString("name: " + o.name + "\n")

	if indent != "" {
		buf.WriteString(indent)
	}
	buf.WriteString("path: " + o.filePath + "\n")

	if indent != "" {
		buf.WriteString(indent)
	}
	buf.WriteString("loaded: " + o.loadedAt.Format(time.Stamp) + "\n")

	if indent != "" {
		buf.WriteString(indent)
	}
	buf.WriteString(fmt.Sprintf("version: %s\n", o.versionStr()))

	if indent != "" {
		buf.WriteString(indent)
	}
	buf.WriteString(fmt.Sprintf("description: %s\n", o.desc))

	if indent != "" {
		buf.WriteString(indent)
	}
	buf.WriteString(fmt.Sprintf("context-type: %t\n", o.optNewCtxFn != nil))

	if indent != "" {
		buf.WriteString(indent)
	}
	buf.WriteString(fmt.Sprintf("unloadable: %t\n", o.optUnloadFn != nil))

	if indent != "" {
		buf.WriteString(indent)
	}
	buf.WriteString(fmt.Sprintf("debugfn: %v\n", o.optDebugFn != nil))

	if indent != "" {
		buf.WriteString(indent)
	}
	buf.WriteString("parsers:")

	if len(o.parsers) == 0 {
		buf.WriteString(" (none)\n")
	} else {
		if indent != "" {
			buf.WriteString(indent)
		}
		buf.WriteByte('\n')

		buf.WriteString(o.ParsersPrettyString(indent+"  ") + "\n")
	}

	if indent != "" {
		buf.WriteString(indent)
	}

	buf.WriteString("commands:")

	if len(o.commands) == 0 {
		buf.WriteString(" (none)")
	} else {
		if indent != "" {
			buf.WriteString(indent)
		}
		buf.WriteByte('\n')

		buf.WriteString(o.CommandsPrettyString(indent + "  "))
	}

	return buf.String()
}

func (o *Plugin) versionStr() string {
	if o.version == 0 {
		return "0.0.0"
	}

	major := strconv.FormatUint(uint64(o.version>>24), 10)

	minor := strconv.FormatUint(uint64((o.version>>16)&0xFF), 10)

	patch := strconv.FormatUint(uint64((o.version>>8)&0xFF), 10)

	//idk := strconv.FormatUint(uint64(o.version&0xFF), 10)

	return major + "." + minor + "." + patch
}

func (o *Plugin) ParsersPrettyString(indent string) string {
	if len(o.parsers) == 0 {
		return ""
	}

	buf := bytes.Buffer{}

	for i, s := range o.parsers {
		if indent != "" {
			buf.WriteString(indent)
		}

		buf.WriteString(s.name)

		if i != len(o.parsers)-1 {
			buf.WriteString("\n")
		}
	}

	return buf.String()
}

func (o *Plugin) CommandsPrettyString(indent string) string {
	if len(o.commands) == 0 {
		return ""
	}

	buf := bytes.Buffer{}

	for i, s := range o.commands {
		if indent != "" {
			buf.WriteString(indent)
		}

		buf.WriteString(s.name)

		if i != len(o.commands)-1 {
			buf.WriteString("\n")
		}
	}

	return buf.String()
}

func (o *Plugin) FilePath() string {
	return o.filePath
}

func (o *Plugin) Name() string {
	return o.name
}

func (o *Plugin) Version() uint32 {
	return o.version
}

func (o *Plugin) Description() string {
	return o.desc
}

func (o *Plugin) EnableDebug() {
	if o.optUnloadFn != nil {
		o.unloadRwMu.RLock()
		defer o.unloadRwMu.RUnlock()
	}

	if o.optDebugFn != nil {
		o.optDebugFn(true)
	}
}

func (o *Plugin) DisableDebug() {
	if o.optUnloadFn != nil {
		o.unloadRwMu.RLock()
		defer o.unloadRwMu.RUnlock()
	}

	if o.optDebugFn != nil {
		o.optDebugFn(false)
	}
}

func (o *Plugin) Alloc(sizeBytes uint32) (SharedBuf, error) {
	if o.optUnloadFn != nil {
		o.unloadRwMu.RLock()
		defer o.unloadRwMu.RUnlock()

		if o.unloaded {
			return SharedBuf{}, plugins.ErrPluginUnloaded
		}
	}

	return sharedBufFromPtr(o.allocFn(sizeBytes)), nil
}

func (o *Plugin) Free(buf SharedBuf) {
	if o.optUnloadFn != nil {
		o.unloadRwMu.RLock()
		defer o.unloadRwMu.RUnlock()
	}

	if o.freeMemFn != nil {
		o.freeMemFn(buf.ptr)
	}
}

func (o *Plugin) maybeNewCtx(ctx context.Context) (uintptr, func()) {
	if o.optNewCtxFn == nil {
		return 0, nil
	}

	var ctxPtr uintptr
	var ctxCloserPtr uintptr

	o.optNewCtxFn(&ctxPtr, &ctxCloserPtr)
	if ctxPtr == 0 {
		return 0, nil
	}

	giveUp := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
		case <-giveUp:
		}

		if o.optUnloadFn != nil {
			o.unloadRwMu.RLock()
			defer o.unloadRwMu.RUnlock()

			if o.unloaded {
				return
			}
		}

		o.optCanCtxFn(ctxCloserPtr)
	}()

	once := &sync.Once{}

	cancelFn := func() {
		once.Do(func() {
			close(giveUp)
		})
	}

	return ctxPtr, cancelFn
}

func (o *Plugin) IterParsers(fn func(plugins.Parser) error) error {
	if o.optUnloadFn != nil {
		o.unloadRwMu.RLock()
		defer o.unloadRwMu.RUnlock()

		if o.unloaded {
			return plugins.ErrPluginUnloaded
		}
	}

	for i := range o.parsers {
		err := fn(o.parsers[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *Plugin) IterCommands(fn func(plugins.Command) error) error {
	if o.optUnloadFn != nil {
		o.unloadRwMu.RLock()
		defer o.unloadRwMu.RUnlock()

		if o.unloaded {
			return plugins.ErrPluginUnloaded
		}
	}

	for i := range o.commands {
		err := fn(o.commands[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *Plugin) Unload() error {
	o.unloadRwMu.Lock()
	defer o.unloadRwMu.Unlock()

	if o.unloaded {
		return errors.New("already unloaded")
	}

	if o.optUnloadFn == nil {
		return errors.New("plugin is not designed to be unloaded")
	}

	o.unloaded = true

	o.optUnloadFn()

	o.optUnloadFn = func() {}
	o.allocFn = func(uint32) uintptr { return 0 }
	o.freeMemFn = func(uintptr) {}
	o.parsers = nil
	o.commands = nil
	o.optDebugFn = nil

	err := o.lib.Release()

	o.lib = nil

	if err != nil {
		return fmt.Errorf("failed to release underlying library - %w", err)
	}

	return nil
}

type parser struct {
	name      string
	parseFn   func(cancel uintptr, addr uintptr, dstStrPtr *uintptr) uintptr
	freeBufFn func(SharedBuf)
	newCtxFn  func(context.Context) (uintptr, func())
	parentMu  *sync.RWMutex
	parentUnl *bool
}

func (o *parser) Name() string {
	return o.name
}

func (o *parser) PrettyString(indent string) string {
	buf := bytes.Buffer{}

	if indent != "" {
		buf.WriteString(indent)
	}
	buf.WriteString(o.name)

	return buf.String()
}

func (o *parser) Run(ctx context.Context, addr uintptr) ([]byte, error) {
	if o.parentMu != nil {
		o.parentMu.RLock()
		defer o.parentMu.RUnlock()

		if *o.parentUnl {
			return nil, plugins.ErrPluginUnloaded
		}
	}

	ctxPtr, cancelFn := o.newCtxFn(ctx)
	if cancelFn != nil {
		defer cancelFn()
	}

	var strPtr uintptr

	result := o.parseFn(ctxPtr, addr, &strPtr)
	if result != 0 {
		msg := stringFromSharedBufRef(result, o.freeBufFn)

		return nil, fmt.Errorf("parser failed: %s", msg)
	}

	return bytesFromSharedBufRef(strPtr, o.freeBufFn), nil
}

type command struct {
	name      string
	commandFn func(cancel uintptr, argsListPtr uintptr, outputStrPtr *uintptr) uintptr
	allocFn   func(uint32) uintptr
	freeBufFn func(SharedBuf)
	newCtxFn  func(context.Context) (uintptr, func())
	parentMu  *sync.RWMutex
	parentUnl *bool
}

func (o *command) Name() string {
	return o.name
}

func (o *command) PrettyString(indent string) string {
	buf := bytes.Buffer{}

	if indent != "" {
		buf.WriteString(indent)
	}
	buf.WriteString(o.name)

	return buf.String()
}

func (o *command) Run(ctx context.Context, args []string) ([]byte, error) {
	if o.parentMu != nil {
		o.parentMu.RLock()
		defer o.parentMu.RUnlock()

		if *o.parentUnl {
			return nil, plugins.ErrPluginUnloaded
		}
	}

	ctxPtr, cancelFn := o.newCtxFn(ctx)
	if cancelFn != nil {
		defer cancelFn()
	}

	tmp := make([]string, 1+len(args))

	tmp[0] = o.name
	copy(tmp[1:], args)

	argsNull := []byte(strings.Join(tmp, "\x00"))

	argsSharedBuf := sharedBufFromPtr(o.allocFn(uint32(len(argsNull))))
	argsSharedBuf.WriteBytes(argsNull)

	argsPtr := argsSharedBuf.Pointer()

	var outputPtr uintptr

	result := o.commandFn(ctxPtr, argsPtr, &outputPtr)
	if result != 0 {
		msg := stringFromSharedBufRef(result, o.freeBufFn)

		return nil, fmt.Errorf("command failed: %s", msg)
	}

	return bytesFromSharedBufRef(outputPtr, o.freeBufFn), nil
}
