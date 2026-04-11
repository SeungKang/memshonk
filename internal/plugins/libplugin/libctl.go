package libplugin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/SeungKang/memshonk/internal/dl"
	"github.com/SeungKang/memshonk/internal/exedata"
	"github.com/SeungKang/memshonk/internal/plugins"
)

const (
	appName = "memshonk"

	pluginFnSuffix = "_ms"
)

// Required functions for library-based plugins.
const (
	allocFnName           = "alloc_v0"
	freeFnName            = "free_v0"
	setReadFromProcFnName = "set_read_from_process_v0"
	setWriteToProcFnName  = "set_write_to_process_v0"
)

// Optional functions for library-based plugins.
const (
	versionFnName     = "version"
	unloadFnName      = "unload"
	debugFnName       = "debug"
	descriptionFnName = "description_v0"
	newCtxFnName      = "new_ctx_v0"
	cancelCtxFnName   = "cancel_ctx_v0"
)

var _ plugins.Ctl = (*Ctl)(nil)

func NewCtl(args plugins.CtlConfig) (*Ctl, error) {
	ctl := &Ctl{
		process:       args.Process,
		callbacksList: newGoCallbacksList(args.Process),
	}

	return ctl, nil
}

type Ctl struct {
	process       plugins.Process
	callbacksList *goCallbacksList
	rwMu          sync.RWMutex
	plugins       []*Plugin
}

func (o *Ctl) PrettyString(indent string) string {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	if len(o.plugins) == 0 {
		return "no plugins loaded"
	}

	buf := bytes.Buffer{}

	innerIndent := indent + "  "

	for _, plugin := range o.plugins {
		if indent != "" {
			buf.WriteString(indent)
		}

		buf.WriteString(plugin.name + "\n")

		buf.WriteString(plugin.PrettyString(innerIndent))
	}

	return buf.String()
}

func (o *Ctl) Plugin(pluginName string) (plugins.Plugin, error) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	plugin, hasIt := o.isLoaded(pluginName)
	if !hasIt {
		return nil, fmt.Errorf("%q: %w",
			pluginName, plugins.ErrPluginNotLoaded)
	}

	return plugin, nil
}

func (o *Ctl) Reload(ctx context.Context, args plugins.ReloadPluginArgs) error {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	plugin, isLoaded := o.isLoaded(args.Name)
	if !isLoaded {
		return plugins.ErrPluginNotLoaded
	}

	err := o.unload(args.Name)
	if err != nil {
		return err
	}

	if len(plugin.config.ExecOnReload) > 0 {
		err = execReload(ctx, args, plugin.config)
		if err != nil {
			_, loadErr := o.load(plugin.config)
			if loadErr == nil {
				return fmt.Errorf("exec on reload failed (managed load anyways) - %w", err)
			}

			return fmt.Errorf("exec on reload failed - %w", err)
		}
	}
	_, err = o.load(plugin.config)
	if err != nil {
		return err
	}

	return nil
}

func (o *Ctl) isLoaded(pluginName string) (*Plugin, bool) {
	for _, plugin := range o.plugins {
		if plugin.name == pluginName {
			plugin := plugin

			return plugin, true
		}
	}

	return nil, false
}

func (o *Ctl) Load(config plugins.PluginConfig) (plugins.Plugin, error) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	return o.load(config)
}

func (o *Ctl) load(config plugins.PluginConfig) (plugins.Plugin, error) {
	absFilePath, err := filepath.Abs(config.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute file path for library - %w", err)
	}

	name := filepath.Base(absFilePath)
	before, _, found := strings.Cut(name, ".")
	if found {
		name = before
	}

	name = strings.TrimSuffix(name, "-"+appName)
	name = strings.TrimSuffix(name, "_"+appName)

	_, alreadyLoaded := o.isLoaded(name)
	if alreadyLoaded {
		return nil, fmt.Errorf("plugin is already loaded (%q)", name)
	}

	symbols, err := relevantLibrarySymbols(absFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin symbols from library file - %w", err)
	}

	lib, err := dl.Open(absFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load library - %w", err)
	}

	// setupPlugin allows us to call lib.Release
	// in one place if the function fails.
	libPlugin, err := o.setupPlugin(setupPluginArgs{
		config:   config,
		filePath: absFilePath,
		name:     name,
		symbols:  symbols,
		lib:      lib,
	})
	if err != nil {
		lib.Release()
		return nil, fmt.Errorf("failed to setup plugin - %w", err)
	}

	o.addPlugin(libPlugin)

	return libPlugin, nil
}

func relevantLibrarySymbols(libFilePath string) (pluginSymbols, error) {
	libExe, err := exedata.ParsePathForCurrentPlatform(libFilePath, exedata.ParserOptions{})
	if err != nil {
		return pluginSymbols{}, err
	}

	var syms pluginSymbols

	const cmdSuffix = pluginFnSuffix + "cmd"
	const parserSuffix = pluginFnSuffix + "par"

	for _, sym := range libExe.Symbols() {
		switch {
		case strings.HasSuffix(sym.Name, cmdSuffix):
			cut, err := cleanupLibrarySymbol(sym.Name, cmdSuffix)
			if err != nil {
				return pluginSymbols{}, err
			}

			syms.commands = append(syms.commands, pluginSymbolInfo{
				symName:   sym.Name,
				finalName: cut,
			})
		case strings.HasSuffix(sym.Name, parserSuffix):
			cut, err := cleanupLibrarySymbol(sym.Name, parserSuffix)
			if err != nil {
				return pluginSymbols{}, err
			}

			syms.parsers = append(syms.parsers, pluginSymbolInfo{
				symName:   sym.Name,
				finalName: cut,
			})
		}
	}

	sort.SliceStable(syms.commands, func(i int, j int) bool {
		return syms.commands[i].finalName == syms.commands[j].finalName
	})

	sort.SliceStable(syms.parsers, func(i int, j int) bool {
		return syms.parsers[i].finalName == syms.parsers[j].finalName
	})

	return syms, nil
}

func cleanupLibrarySymbol(symName string, suffix string) (string, error) {
	// symName: abcd_foo
	//          01234567 (len: 8)
	//
	// suufix:  _foo (len: 4)
	//
	// [0 : 8 - 4]
	// [0 : 4] == abcd
	//            0123
	cut := symName[0 : len(symName)-len(suffix)]

	// Some executable formats add a "_" to the start of
	// exported symbols like macho (or maybe that is an
	// Apple thing... idk).
	cut = strings.TrimPrefix(cut, "_")

	if cut == "" {
		return "", fmt.Errorf("library symbol %q is invalid (results in empty string)",
			symName)
	}

	return cut, nil
}

type pluginSymbols struct {
	commands []pluginSymbolInfo
	parsers  []pluginSymbolInfo
}

type pluginSymbolInfo struct {
	symName   string
	finalName string
}

func (o *Ctl) addPlugin(plugin *Plugin) {
	o.plugins = append(o.plugins, plugin)

	sort.SliceStable(o.plugins, func(i int, j int) bool {
		return o.plugins[i].Name() > o.plugins[j].Name()
	})
}

func (o *Ctl) Unload(name string) error {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	return o.unload(name)
}

func (o *Ctl) unload(name string) error {
	// Not the most efficient way to do this,
	// but it is far less error-prone.
	var newSlice []*Plugin

	for i := range o.plugins {
		if name == o.plugins[i].name {
			err := o.plugins[i].Unload()
			if err != nil {
				return fmt.Errorf("failed to unload plugin - %w", err)
			}

			o.callbacksList.deregister(o.plugins[i].callbacks)
		} else {
			newSlice = append(newSlice, o.plugins[i])
		}
	}

	if len(o.plugins) == len(newSlice) {
		return fmt.Errorf("%q: %w", name, plugins.ErrPluginNotLoaded)
	}

	o.plugins = newSlice

	return nil
}

type setupPluginArgs struct {
	config   plugins.PluginConfig
	filePath string
	name     string
	symbols  pluginSymbols
	lib      *dl.Library
}

func (o *Ctl) setupPlugin(args setupPluginArgs) (*Plugin, error) {
	var versionFn func() uint32
	var version uint32

	_ = args.lib.Func(versionFnName, &versionFn)

	if versionFn != nil {
		version = versionFn()
	}

	plugin := &Plugin{
		config:   args.config,
		lib:      args.lib,
		name:     args.name,
		loadedAt: time.Now(),
		filePath: args.filePath,
		version:  version,
	}

	var err error

	plugin.callbacks, err = o.callbacksList.register(plugin)
	if err != nil {
		return nil, fmt.Errorf("failed to register callbacks - %w", err)
	}

	_, err = findFirstFunc(
		[]string{allocFnName},
		&plugin.allocFn,
		args.lib)
	if err != nil {
		return nil, fmt.Errorf("failed to setup alloc fn - %w", err)
	}

	_, err = findFirstFunc(
		[]string{freeFnName},
		&plugin.freeMemFn,
		args.lib)
	if err != nil {
		return nil, fmt.Errorf("failed to setup free fn - %w", err)
	}

	_ = args.lib.Func(unloadFnName, &plugin.optUnloadFn)

	_ = args.lib.Func(debugFnName, &plugin.optDebugFn)

	_ = args.lib.Func(newCtxFnName, &plugin.optNewCtxFn)

	if plugin.optNewCtxFn != nil {
		err = args.lib.Func(cancelCtxFnName, &plugin.optCanCtxFn)
		if err != nil {
			return nil, fmt.Errorf("failed to setup cancel ctx fn - %w", err)
		}
	}

	var getDescriptionFn func() uintptr

	_ = args.lib.Func(descriptionFnName, &getDescriptionFn)
	if getDescriptionFn != nil {
		plugin.desc = stringFromSharedBufRef(getDescriptionFn(), plugin.Free)
	}

	if len(args.symbols.parsers) > 0 {
		err = plugin.loadParsers(args.symbols.parsers)
		if err != nil {
			return nil, fmt.Errorf("failed to load parsers - %w", err)
		}
	}

	if len(args.symbols.commands) > 0 {
		err = plugin.loadCommands(args.symbols.commands)
		if err != nil {
			return nil, fmt.Errorf("failed to load commands - %w", err)
		}
	}

	return plugin, nil
}

func findFirstFunc(funcNames []string, goFnPtr interface{}, lib *dl.Library) (string, error) {
	if len(funcNames) == 0 {
		return "", errors.New("function names slice is empty")
	}

	var lastErr error

	for _, name := range funcNames {
		err := lib.Func(name, goFnPtr)
		if err != nil {
			lastErr = err

			continue
		}

		return name, nil
	}

	if lastErr != nil {
		return "", fmt.Errorf("failed to find functions matching: %q (last error: %w)",
			funcNames, lastErr)
	}

	return "", fmt.Errorf("failed to find functions matching: %q (no additional info available)",
		funcNames)
}
