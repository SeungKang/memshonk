package libplugin

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/SeungKang/memshonk/internal/dl"
	"github.com/SeungKang/memshonk/internal/plugins"
)

type Plugin struct {
	config           plugins.PluginConfig
	filePath         string
	name             string
	loadedAt         time.Time
	version          uint16
	getErrorStringFn func(code uint32) uintptr
	freeStringFn     func(uintptr)
	optUnloadFn      func()
	optDebugFn       func(bool)
	parsers          []*parserFnConfig
	unloadRwMu       sync.RWMutex
	lib              *dl.Library
}

func (o *Plugin) loadParsers() error {
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

	parserFnsStr := cstr.string()

	if len(parserFnsStr) == 0 {
		return nil
	}

	parserNames := strings.Split(parserFnsStr, " ")
	sort.Strings(parserNames)

	parsers := make([]*parserFnConfig, len(parserNames))

	for i, parserFnName := range parserNames {
		par := &parserFnConfig{
			name:      parserFnName,
			errStrFn:  o.ErrorStr,
			freeStrFn: o.freeStringFn,
		}

		err := o.lib.Func(parserFnName, &par.parseFn)
		if err != nil {
			return fmt.Errorf("failed to find parser fn %q - %w",
				parserFnName, err)
		}

		parsers[i] = par
	}

	o.parsers = parsers

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
	buf.WriteString(fmt.Sprintf("version: %d\n", o.version))

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

func (o *Plugin) FilePath() string {
	return o.filePath
}

func (o *Plugin) Name() string {
	return o.name
}

func (o *Plugin) Version() uint16 {
	return o.version
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

func (o *Plugin) ErrorStr(code uint32) string {
	if o.optUnloadFn != nil {
		o.unloadRwMu.RLock()
		defer o.unloadRwMu.RUnlock()

		err := o.isUnloaded()
		if err != nil {
			return "plugin was unloaded"
		}
	}

	// For rust impls:
	// https://users.rust-lang.org/t/whats-the-best-practice-to-get-string-by-ffi/39496/2
	cstr := copyCStrByNull{
		strPtr: o.getErrorStringFn(code),
		freeFn: o.freeStringFn,
	}

	return cstr.string()
}

func (o *Plugin) RunParser(name string, targetAddr uintptr) ([]byte, error) {
	if o.optUnloadFn != nil {
		o.unloadRwMu.RLock()
		defer o.unloadRwMu.RUnlock()
	}

	parser, hasIt := o.getParser(name)
	if !hasIt {
		return nil, fmt.Errorf("unknown parser: %q", name)
	}

	return parser.run(targetAddr)
}

func (o *Plugin) getParser(name string) (*parserFnConfig, bool) {
	for i := range o.parsers {
		if name == o.parsers[i].name {
			return o.parsers[i], true
		}
	}

	return nil, false
}

func (o *Plugin) isUnloaded() error {
	if o.lib == nil {
		return errors.New("library was unloaded")
	}

	return nil
}

func (o *Plugin) Unload() error {
	if o.optUnloadFn == nil {
		return errors.New("plugin is not designed to be unloaded")
	}

	o.unloadRwMu.Lock()
	defer o.unloadRwMu.Unlock()

	if o.lib == nil {
		return errors.New("already unloaded")
	}

	o.optUnloadFn()

	err := o.lib.Release()
	if err != nil {
		return fmt.Errorf("failed to release underlying library - %w", err)
	}

	o.lib = nil

	o.optUnloadFn = func() {}
	o.freeStringFn = func(uintptr) {}
	o.getErrorStringFn = func(uint32) uintptr { return 0 }
	o.parsers = nil
	o.optDebugFn = nil

	return nil
}

type parserFnConfig struct {
	name      string
	parseFn   func(addr uintptr, strPtr *uintptr) uint32
	errStrFn  func(uint32) string
	freeStrFn func(uintptr)
}

func (o *parserFnConfig) PrettyString(indent string) string {
	buf := bytes.Buffer{}

	if indent != "" {
		buf.WriteString(indent)
	}
	buf.WriteString(o.name)

	return buf.String()
}

func (o *parserFnConfig) run(addr uintptr) ([]byte, error) {
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
