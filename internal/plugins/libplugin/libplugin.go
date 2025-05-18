package libplugin

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/SeungKang/memshonk/internal/dl"
	"github.com/SeungKang/memshonk/internal/plugins"
)

type Plugin struct {
	lib              *dl.Library
	filePath         string
	name             string
	loadedAt         time.Time
	version          uint16
	getErrorStringFn func(code uint32) uintptr
	freeStringFn     func(uintptr)
	debufFn          func(bool)
	parsers          []*ParserLibraryPlugin
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

	parserFnsCsv := cstr.string()

	if len(parserFnsCsv) == 0 {
		return nil
	}

	parserNames := strings.Split(parserFnsCsv, ",")
	sort.Strings(parserNames)

	parsers := make([]*ParserLibraryPlugin, len(parserNames))

	for _, parserFnName := range strings.Split(parserFnsCsv, ",") {
		par := &ParserLibraryPlugin{
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

		parsers = append(o.parsers, par)
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

func (o *Plugin) EnableDebug() {
	if o.debufFn != nil {
		o.debufFn(true)
	}
}

func (o *Plugin) DisableDebug() {
	if o.debufFn != nil {
		o.debufFn(false)
	}
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

func (o *Plugin) ErrorStr(code uint32) string {
	// For rust impls:
	// https://users.rust-lang.org/t/whats-the-best-practice-to-get-string-by-ffi/39496/2
	cstr := copyCStrByNull{
		strPtr: o.getErrorStringFn(code),
		freeFn: o.freeStringFn,
	}

	return cstr.string()
}

func (o *Plugin) Parser(name string) (plugins.ParserPlugin, bool) {
	return o.getParser(name)
}

func (o *Plugin) getParser(name string) (plugins.ParserPlugin, bool) {
	for i := range o.parsers {
		if name == o.parsers[i].name {
			return o.parsers[i], true
		}
	}

	return nil, false
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
	return libName + pluginNamespaceSep + parserName
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
