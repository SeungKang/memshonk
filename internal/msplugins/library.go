package msplugins

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unsafe"

	"github.com/SeungKang/memshonk/internal/dl"
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

func NewLibraryPluginCtl(todoProcessPlaceholder interface{}) (*LibraryPluginCtl, error) {
	ctl := &LibraryPluginCtl{}

	var err error

	ctl.readFromAddrCallback, err = dl.NewCallback(ctl.ReadFromAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create callback for ReadFromAddr - %w",
			err)
	}

	return ctl, nil
}

type LibraryPluginCtl struct {
	readFromAddrCallback uintptr

	rwMu           sync.RWMutex
	namesToPlugins map[string]*LibraryPlugin

	// TODO
	// 	process progctl.Process
}

func (o *LibraryPluginCtl) PrettyString(indent string) string {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	if len(o.namesToPlugins) == 0 {
		return ""
	}

	pluginsSlice := make([]*LibraryPlugin, len(o.namesToPlugins))

	i := 0

	for _, plugin := range o.namesToPlugins {
		plugin := plugin

		pluginsSlice[i] = plugin

		i++
	}

	sort.SliceStable(pluginsSlice, func(i int, j int) bool {
		return pluginsSlice[i].name > pluginsSlice[j].name
	})

	buf := bytes.Buffer{}

	innerIndent := indent + "  "

	for _, plugin := range pluginsSlice {
		if indent != "" {
			buf.WriteString(indent)
		}

		buf.WriteString(plugin.name + "\n")

		buf.WriteString(plugin.PrettyString(innerIndent))
	}

	return buf.String()
}

func (o *LibraryPluginCtl) ReadFromAddr(dst unsafe.Pointer, size uint64, srcAddr uintptr) uint8 {
	return 1

	// TODO
	//
	// 	data, err := o.process.ReadFromAddr(
	// 		context.Background(),
	// 		memory.AbsoluteAddrPointer(srcAddr),
	// 		size)
	// 	if err != nil {
	// 		return 1
	// 	}

	// 	if uint64(len(data)) > size {
	// 		return 2
	// 	}

	// return 0
}

func (o *LibraryPluginCtl) Get(pluginName string) (*LibraryPlugin, error) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	plugin, hasIt := o.namesToPlugins[pluginName]
	if !hasIt {
		return nil, errors.New("plugin not loaded")
	}

	return plugin, nil
}

func (o *LibraryPluginCtl) Load(pluginFilePath string) (*LibraryPlugin, error) {
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

	_, alreadyLoaded := o.namesToPlugins[name]
	if alreadyLoaded {
		return nil, fmt.Errorf("plugin is already loaded (%q)", name)
	}

	if o.namesToPlugins == nil {
		o.namesToPlugins = make(map[string]*LibraryPlugin)
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

	o.namesToPlugins[name] = libPlugin

	return libPlugin, nil
}

func (o *LibraryPluginCtl) setupPlugin(filePath string, name string, lib *dl.Library) (*LibraryPlugin, error) {
	var versionFn func() uint16

	err := lib.Func(versionFnName, &versionFn)
	if err != nil {
		return nil, fmt.Errorf("failed to get version function in library - %w", err)
	}

	plugin := &LibraryPlugin{
		lib:      lib,
		name:     name,
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

func (o *LibraryPlugin) loadParsers() error {
	var getParsersFn func() uintptr

	_, err := findFirstFunc(
		[]string{parsersFnName},
		&getParsersFn,
		o.lib)
	if err != nil {
		return nil
	}

	cstr := copyCStrByNull{
		strPtr: getParsersFn(),
		freeFn: o.freeStringFn,
	}

	parserFnsCsv := cstr.string()

	if len(parserFnsCsv) == 0 {
		return nil
	}

	parsers := make(map[string]ParserLibraryPlugin)

	for _, parserFnName := range strings.Split(parserFnsCsv, ",") {
		par := ParserLibraryPlugin{
			parent:    o.name,
			name:      parserFnName,
			errStrFn:  o.ErrorStr,
			freeStrFn: o.freeStringFn,
		}

		err := o.lib.Func(parserFnName, &par.parseFn)
		if err != nil {
			return fmt.Errorf("failed to find parser fn %q - %w",
				parserFnName, err)
		}

		parsers[parserFnName] = par
	}

	o.parsers = parsers

	return nil
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

type LibraryPlugin struct {
	lib              *dl.Library
	filePath         string
	name             string
	version          uint16
	getErrorStringFn func(code uint32) uintptr
	freeStringFn     func(uintptr)
	debufFn          func(bool)
	parsers          map[string]ParserLibraryPlugin
}

func (o *LibraryPlugin) PrettyString(indent string) string {
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
	buf.WriteString(fmt.Sprintf("version: %d\n", o.version))

	if indent != "" {
		buf.WriteString(indent)
	}
	buf.WriteString(fmt.Sprintf("debugfn: %v\n", o.debufFn != nil))

	if indent != "" {
		buf.WriteString(indent)
	}
	buf.WriteString("parsers:")

	if len(o.parsers) == 0 {
		buf.WriteString(" (none)")
	} else {
		if indent != "" {
			buf.WriteString(indent)
		}
		buf.WriteByte('\n')

		buf.WriteString(o.ParsersPrettyString(indent + "  "))
	}

	return buf.String()
}

func (o *LibraryPlugin) ParsersPrettyString(indent string) string {
	if len(o.parsers) == 0 {
		return ""
	}

	buf := bytes.Buffer{}

	names := make([]string, len(o.parsers))
	i := 0

	for _, parser := range o.parsers {
		names[i] = parser.name

		i++
	}

	sort.Strings(names)

	for i, s := range names {
		if indent != "" {
			buf.WriteString(indent)
		}

		buf.WriteString(s)

		if i != len(names)-1 {
			buf.WriteString("\n")
		}
	}

	return buf.String()
}

func (o *LibraryPlugin) EnableDebug() {
	if o.debufFn != nil {
		o.debufFn(true)
	}
}

func (o *LibraryPlugin) DisableDebug() {
	if o.debufFn != nil {
		o.debufFn(false)
	}
}

func (o *LibraryPlugin) FilePath() string {
	return o.filePath
}

func (o *LibraryPlugin) Name() string {
	return o.name
}

func (o *LibraryPlugin) Version() uint16 {
	return o.version
}

func (o *LibraryPlugin) ErrorStr(code uint32) string {
	// For rust impls:
	// https://users.rust-lang.org/t/whats-the-best-practice-to-get-string-by-ffi/39496/2
	cstr := copyCStrByNull{
		strPtr: o.getErrorStringFn(code),
		freeFn: o.freeStringFn,
	}

	return cstr.string()
}

func (o *LibraryPlugin) Parser(name string) (ParserLibraryPlugin, bool) {
	parser, hasIt := o.parsers[name]
	if !hasIt {
		// Try with ID.

		id := parserID(o.name, name)

		parser, hasIt = o.parsers[id]
	}

	return parser, hasIt
}

type ParserLibraryPlugin struct {
	parent    string
	name      string
	parseFn   func(addr uintptr, strPtr *uintptr) uint32
	errStrFn  func(uint32) string
	freeStrFn func(uintptr)
}

func (o *ParserLibraryPlugin) PrettyString(indent string) string {
	buf := bytes.Buffer{}

	if indent != "" {
		buf.WriteString(indent)
	}
	buf.WriteString(o.ID())

	return buf.String()
}

func (o *ParserLibraryPlugin) ID() string {
	return parserID(o.parent, o.name)
}

func parserID(libName string, parserName string) string {
	return libName + "::" + parserName
}

func (o *ParserLibraryPlugin) Run(addr uintptr) ([]byte, error) {
	cstr := copyCStrByNull{
		freeFn: o.freeStrFn,
	}

	result := o.parseFn(addr, &cstr.strPtr)
	if result != 0 {
		return nil, fmt.Errorf("parser failed with code %d: %s",
			result, o.errStrFn(result))
	}

	return cstr.slice(), nil
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
