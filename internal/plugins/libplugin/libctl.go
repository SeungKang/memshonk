package libplugin

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/SeungKang/memshonk/internal/dl"
	"github.com/SeungKang/memshonk/internal/plugins"
)

const (
	pluginNamespaceSep = "::"
)

// Required functions for library-based plugins.
const (
	versionFnName         = "version"
	errorStringFnName     = "error_string_v0"
	setReadFromAddrFnName = "set_read_from_addr_v0"
	freeStrFnName         = "free_string_v0"
)

// Optional functions for library-based plugins.
const (
	debugFnName   = "debug"
	parsersFnName = "parsers_v0"
)

var _ plugins.Ctl = (*Ctl)(nil)

func NewCtl(args plugins.CtlConfig) (*Ctl, error) {
	ctl := &Ctl{
		process: args.Process,
	}

	var err error

	ctl.readFromAddrCallback, err = dl.NewCallback(ctl.readFromAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create callback for ReadFromAddr - %w",
			err)
	}

	for _, filePath := range args.InitialPlugins {
		_, err := ctl.Load(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load plugin: %q - %w",
				filePath, err)
		}
	}

	return ctl, nil
}

type Ctl struct {
	process plugins.Process

	readFromAddrCallback uintptr

	rwMu    sync.RWMutex
	plugins []*Plugin
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

func (o *Ctl) readFromAddr(dst uintptr, size uintptr, srcAddr uintptr) uintptr {
	data, err := o.process.ReadFromAddr(srcAddr, uint64(size))
	if err != nil {
		return 1
	}

	if uintptr(len(data)) > size {
		return 2
	}

	dstPtr := dst

	for i := uintptr(0); i < size; i++ {
		b := (*byte)(unsafe.Pointer(dstPtr))

		*b = data[i]

		dstPtr++
	}

	return 0
}

func (o *Ctl) Plugin(pluginName string) (plugins.Plugin, bool) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	return o.isLoaded(pluginName)
}

func (o *Ctl) Parser(parserID string) (plugins.ParserPlugin, error) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	pluginName, parserName, hasIt := separateNamespace(parserID)
	if !hasIt {
		return nil, fmt.Errorf("parser must use <plugin-name>%s<parser-name> syntax",
			pluginNamespaceSep)
	}

	plugin, isLoaded := o.isLoaded(pluginName)
	if !isLoaded {
		return nil, fmt.Errorf("library is not loaded (%q)", pluginName)
	}

	parser, hasIt := plugin.Parser(parserName)
	if !hasIt {
		return nil, fmt.Errorf("library does not contain parser: %q",
			parserName)
	}

	return parser, nil
}

func (o *Ctl) isLoaded(pluginName string) (plugins.Plugin, bool) {
	for _, plugin := range o.plugins {
		if plugin.name == pluginName {
			plugin := plugin

			return plugin, true
		}
	}

	return nil, false
}

func (o *Ctl) addPlugin(plugin *Plugin) {
	o.plugins = append(o.plugins, plugin)

	sort.SliceStable(o.plugins, func(i int, j int) bool {
		return o.plugins[i].Name() > o.plugins[j].Name()
	})
}

func (o *Ctl) rmPlugin(plugin *Plugin) {
	// Not the most efficient way to do this,
	// but it is far less error-prone.
	var newSlice []*Plugin

	for i := range o.plugins {
		if plugin != o.plugins[i] {
			newSlice = append(newSlice, o.plugins[i])
		}
	}

	o.plugins = newSlice
}

func (o *Ctl) Load(pluginFilePath string) (plugins.Plugin, error) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	absFilePath, err := filepath.Abs(pluginFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute file path for library - %w", err)
	}

	name := filepath.Base(absFilePath)
	before, _, found := strings.Cut(name, ".")
	if found {
		name = before
	}

	_, alreadyLoaded := o.isLoaded(name)
	if alreadyLoaded {
		return nil, fmt.Errorf("plugin is already loaded (%q)", name)
	}

	lib, err := dl.Open(absFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load library - %w", err)
	}

	// setupPlugin allows us to call lib.Release
	// in one place if the function fails.
	libPlugin, err := o.setupPlugin(absFilePath, name, lib)
	if err != nil {
		lib.Release()
		return nil, fmt.Errorf("failed to setup plugin - %w", err)
	}

	o.addPlugin(libPlugin)

	return libPlugin, nil
}

func (o *Ctl) setupPlugin(filePath string, name string, lib *dl.Library) (*Plugin, error) {
	var versionFn func() uint16

	err := lib.Func(versionFnName, &versionFn)
	if err != nil {
		return nil, fmt.Errorf("failed to get version function in library - %w", err)
	}

	plugin := &Plugin{
		lib:      lib,
		name:     name,
		loadedAt: time.Now(),
		filePath: filePath,
		version:  versionFn(),
	}

	_, err = findFirstFunc(
		[]string{errorStringFnName},
		&plugin.getErrorStringFn,
		lib)
	if err != nil {
		return nil, fmt.Errorf("failed to setup error string fn - %w", err)
	}

	_, err = findFirstFunc(
		[]string{freeStrFnName},
		&plugin.freeStringFn,
		lib)
	if err != nil {
		return nil, fmt.Errorf("failed to setup free string fn - %w", err)
	}

	err = registerCallbackFn(
		[]string{setReadFromAddrFnName},
		o.readFromAddrCallback,
		lib)
	if err != nil {
		return nil, fmt.Errorf("failed to setup read from addr fn - %w", err)
	}

	_ = lib.Func(debugFnName, &plugin.debufFn)

	err = plugin.loadParsers()
	if err != nil {
		return nil, fmt.Errorf("failed to load parsers - %w", err)
	}

	return plugin, nil
}

func registerCallbackFn(funcNames []string, callbackFnPtr uintptr, lib *dl.Library) error {
	var setCallbackFn func(cb uintptr) uint8

	fnName, err := findFirstFunc(funcNames, &setCallbackFn, lib)
	if err != nil {
		return fmt.Errorf("failed to find first matching function - %w", err)
	}

	result := setCallbackFn(callbackFnPtr)
	if result != 0 {
		return fmt.Errorf("%q failed - got status %d",
			fnName, result)
	}

	return nil
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

type copyCStrByLen struct {
	strPtr uintptr
	len    uintptr
	freeFn func(uintptr)
}

func (o *copyCStrByLen) string() string {
	return string(o.slice())
}

func (o *copyCStrByLen) slice() []byte {
	if o.strPtr == 0 || o.len == 0 {
		return nil
	}

	if o.freeFn == nil {
		panic("free function pointer is nil")
	}

	ptr := (*byte)(unsafe.Pointer(o.strPtr))

	origStr := unsafe.String(ptr, o.len)

	copied := make([]byte, o.len)

	for i := range origStr {
		copied[i] = origStr[i]
	}

	o.freeFn(o.strPtr)

	o.strPtr = 0
	o.len = 0

	return copied
}

type copyCStrByNull struct {
	strPtr uintptr
	freeFn func(uintptr)
}

func (o *copyCStrByNull) string() string {
	return string(o.slice())
}

func (o *copyCStrByNull) slice() []byte {
	if o.strPtr == 0 {
		return nil
	}

	if o.freeFn == nil {
		panic("free function pointer is nil")
	}

	walker := o.strPtr

	buf := bytes.Buffer{}

	for {
		b := *(*byte)(unsafe.Pointer(walker))
		if b == 0x00 {
			break
		}

		buf.WriteByte(b)

		walker++
	}

	o.freeFn(o.strPtr)

	o.strPtr = 0

	return buf.Bytes()
}

func separateNamespace(str string) (string, string, bool) {
	return strings.Cut(str, pluginNamespaceSep)
}

func strUsesNamespaceSep(str string) bool {
	return strings.Contains(str, pluginNamespaceSep)
}
