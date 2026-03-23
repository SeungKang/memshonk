package fx

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"strings"
	"unicode"
)

// NewFlagSet creates a new FlagSet with the given name
// and error handling.
func NewFlagSet(name string) *FlagSet {
	return NewCustomFlagSet(name, flag.ContinueOnError)
}

// NewCustomFlagSet creates a new FlagSet with the given name
// and error handling.
func NewCustomFlagSet(name string, errorHandling flag.ErrorHandling) *FlagSet {
	return Wrap(flag.NewFlagSet(name, errorHandling))
}

// Wrap wraps an existing flag.FlagSet with short-flag support.
func Wrap(fset *flag.FlagSet) *FlagSet {
	return &FlagSet{internal: fset}
}

// FlagSet wraps flag.FlagSet with automatic short-flag support.
//
// When a flag with a multi-character name is added, a short alias using
// the first character is automatically registered if not already taken.
type FlagSet struct {
	internal      *flag.FlagSet
	flags         map[string]*ArgConfig
	requiredFlags map[string][]*flag.Flag
	nonflags      []nonflagDef
}

// Actual returns the underlying flag.FlagSet.
func (o *FlagSet) Actual() *flag.FlagSet {
	return o.internal
}

func (o *FlagSet) VisitAll(fn func(ArgInfo)) {
	o.internal.VisitAll(func(f *flag.Flag) {
		config := o.flags[f.Name]

		if config == nil {
			return
		}

		fn(ArgInfo{
			Config:  *config,
			IsFlag:  true,
			OptFlag: f,
		})
	})

	for _, nf := range o.nonflags {
		fn(ArgInfo{
			Config: nf.config,
		})
	}
}

type ArgInfo struct {
	Config  ArgConfig
	IsFlag  bool
	OptFlag *flag.Flag
}

// Parse parses flag definitions from the argument list.
// After parsing flags, it processes positional arguments and validates
// that all required flags and positional arguments were provided.
func (o *FlagSet) Parse(arguments []string) error {
	err := o.internal.Parse(arguments)
	if err != nil {
		return err
	}

	// Build a provided of flags that were actually provided
	provided := make(map[string]bool)
	o.internal.Visit(func(f *flag.Flag) {
		provided[f.Name] = true
	})

	// Check that all required flags were provided
nextRequiredFlag:
	for _, candidates := range o.requiredFlags {
		for _, candidate := range candidates {
			if provided[candidate.Name] {
				continue nextRequiredFlag
			}
		}

		var flagsStr string

		for i, candidate := range candidates {
			if len(candidate.Name) == 1 {
				flagsStr += "-" + candidate.Name
			} else {
				flagsStr += "--" + candidate.Name
			}

			if len(candidates) > 1 && i != len(candidates)-1 {
				flagsStr += " or "
			}
		}

		return fmt.Errorf("required flag(s) not provided: %s", flagsStr)
	}

	// Process positional arguments
	args := o.internal.Args()
	argIdx := 0

	for _, nf := range o.nonflags {
		if nf.isSlice {
			// Slice consumes all remaining arguments
			startIdx := argIdx
			for argIdx < len(args) {
				err = nf.setter(args[argIdx])
				if err != nil {
					return fmt.Errorf("invalid value %q for %s: %w",
						args[argIdx], nf.config.Name, err)
				}
				argIdx++
			}

			if nf.config.Required && argIdx == startIdx {
				return fmt.Errorf("required argument not provided: %s",
					nf.config.Name)
			}
		} else {
			if argIdx >= len(args) {
				if nf.config.Required {
					return fmt.Errorf("required argument not provided: %s",
						nf.config.Name)
				}

				continue
			}

			err = nf.setter(args[argIdx])
			if err != nil {
				return fmt.Errorf("invalid value %q for %s: %w",
					args[argIdx], nf.config.Name, err)
			}

			argIdx++
		}
	}

	return nil
}

type registerFlagConfig struct {
	argConfig     ArgConfig
	optAddShortFn func(shortName string)
}

func (o *FlagSet) registerFlag(config registerFlagConfig) {
	if o.flags == nil {
		o.flags = make(map[string]*ArgConfig)
	}

	o.flags[config.argConfig.Name] = &config.argConfig

	var short *flag.Flag

	if config.optAddShortFn != nil && len(config.argConfig.Name) > 1 {
		shortName := config.argConfig.Name[:1]

		short = o.internal.Lookup(shortName)

		if short == nil {
			config.argConfig.OptShortName = shortName

			config.optAddShortFn(shortName)

			short = o.internal.Lookup(shortName)
		}
	}

	if config.argConfig.Required {
		if o.requiredFlags == nil {
			o.requiredFlags = make(map[string][]*flag.Flag)
		}

		long := o.internal.Lookup(config.argConfig.Name)

		var flagCandidates []*flag.Flag
		if short == nil {
			flagCandidates = []*flag.Flag{long}
		} else {
			flagCandidates = []*flag.Flag{long, short}
		}

		o.requiredFlags[config.argConfig.Name] = flagCandidates
	}
}

func LongArgsUsage(set *FlagSet, maxLineLength uint64) error {
	var last *flag.Flag

	i := 0

	var finalErr error

	writer := set.internal.Output()

	set.internal.VisitAll(func(f *flag.Flag) {
		if finalErr != nil {
			return
		}

		var argHeading string

		if len(f.Name) == 1 {
			if unicode.IsUpper(rune(f.Name[0])) {
				// Single capital letter flag that has no
				// corresponding long flag.
				argHeading = "-" + f.Name

				last = nil
			} else {
				// Normal, lower-case single-letter flag.
				last = f

				return
			}
		} else {
			// Multi-letter (long) flag.
			if last == nil {
				finalErr = fmt.Errorf("missing corresponding short flag for '--%s'",
					f.Name)

				return
			}

			argHeading = "-" + last.Name + ", --" + f.Name

			last = nil
		}

		if i != 0 {
			// Need to add new lines this way because there
			// is no way to get total number of flags.
			_, err := writer.Write([]byte{'\n'})
			if err != nil {
				finalErr = err
				return
			}
		}

		usageInfo, err := writeArgAndDatatypeLine(argHeading, f, 2, writer)
		if err != nil {
			finalErr = err
			return
		}

		var usage string

		if usageInfo.RemoveBackticks {
			usage = strings.ReplaceAll(f.Usage, "`", "")
		} else {
			usage = f.Usage
		}

		_, err = writer.Write([]byte{'\n'})
		if err != nil {
			finalErr = err
			return
		}

		err = writeStringWithIndent(writeStringWithIndentArgs{
			str:    usage,
			indent: 6,
			max:    int(maxLineLength),
			w:      writer,
		})
		if err != nil {
			finalErr = err
			return
		}

		i++
	})

	return finalErr
}

func writeArgAndDatatypeLine(argsWithDashes string, f *flag.Flag, indent int, w io.Writer) (flagUsageInfo, error) {
	_, err := w.Write(bytes.Repeat([]byte{' '}, indent))
	if err != nil {
		return flagUsageInfo{}, err
	}

	_, err = w.Write([]byte(argsWithDashes))
	if err != nil {
		return flagUsageInfo{}, err
	}

	usageInfo := getFlagUsageInfo(f)

	if usageInfo.DatatypeStr != "" {
		if usageInfo.DoNotMarkup {
			_, err = w.Write([]byte(usageInfo.DatatypeStr))
		} else {
			_, err = fmt.Fprintf(w, " <%s>", usageInfo.DatatypeStr)
		}

		if err != nil {
			return flagUsageInfo{}, err
		}

		defValueStr := " (default: "
		if usageInfo.QuoteDefaultValue {
			defValueStr += `"` + usageInfo.DefaultValueStr + `"`
		} else {
			defValueStr += usageInfo.DefaultValueStr
		}
		defValueStr += ")"

		_, err := w.Write([]byte(defValueStr))
		if err != nil {
			return flagUsageInfo{}, err
		}
	}

	return usageInfo, nil
}

type argWriterInfo struct {
	RemoveBackticks bool
}

type TypeHinter interface {
	TypeHint() string
}

type FullTypeHinter interface {
	FullTypeHint() string
}

type DefaultValueHinter interface {
	DefaultValueHint() string
}

type writeStringWithIndentArgs struct {
	str    string
	indent int
	max    int
	w      io.Writer
}

func writeStringWithIndent(args writeStringWithIndentArgs) error {
	i := 0

	for {
		if i != -1 {
			// Write indent string.
			_, err := args.w.Write(bytes.Repeat([]byte{' '}, args.indent))
			if err != nil {
				return err
			}
		}

		if len(args.str[i:]) < args.max {
			// Write remainder of string and return.
			_, err := args.w.Write([]byte(args.str[i:]))
			if err != nil {
				return err
			}

			_, err = args.w.Write([]byte{'\n'})

			return err
		}

		next := i + walkBackUntilWords(args.str[i:i+args.max]) + 1

		_, err := args.w.Write([]byte(args.str[i:next]))
		if err != nil {
			return err
		}

		_, err = args.w.Write([]byte{'\n'})
		if err != nil {
			return err
		}

		i = next
	}
}

func walkBackUntilWords(str string) int {
	numWords := 0

	for i := len(str) - 1; i >= 0; i-- {
		if str[i] == ' ' {
			numWords++

			if numWords == 2 {
				return i
			}
		}
	}

	return len(str)
}

func getFlagUsageInfo(f *flag.Flag) flagUsageInfo {
	var info flagUsageInfo

	//
	// Begin flag default value information.
	//

	var actualDefaultValue interface{}

	if getter, isGetter := f.Value.(flag.Getter); isGetter {
		actualDefaultValue = getter.Get()

		switch actualDefaultValue.(type) {
		case bool:
			return info
		case string:
			info.QuoteDefaultValue = true
		}
	}

	hinter, ok := f.Value.(DefaultValueHinter)
	switch {
	case ok:
		info.DefaultValueStr = hinter.DefaultValueHint()
	case f.DefValue != "":
		info.DefaultValueStr = f.DefValue
	}

	//
	// Begin flag data type information.
	//

	backtickStart := strings.Index(f.Usage, "`")

	if backtickStart > -1 {
		end := strings.Index(f.Usage[backtickStart+1:], "`")

		if end > -1 {
			// "foo `bar` bazz"
			//  0123456789....
			// start: 4
			//
			// "bar` bazz"
			//  012345...
			// end: 3
			//
			// end += 5 == 8
			end += backtickStart + 1

			info.DatatypeStr = f.Usage[backtickStart+1 : end]
			info.RemoveBackticks = true

			return info
		}
	}

	switch flagValue := f.Value.(type) {
	case FullTypeHinter:
		info.DatatypeStr = flagValue.FullTypeHint()
		info.DoNotMarkup = true
	case TypeHinter:
		info.DatatypeStr = flagValue.TypeHint()
	default:
		if actualDefaultValue != nil {
			info.DatatypeStr = fmt.Sprintf("%T", actualDefaultValue)
		} else {
			info.DatatypeStr = fmt.Sprintf("%T", flagValue)
		}
	}

	return info
}

type flagUsageInfo struct {
	DatatypeStr       string
	DefaultValueStr   string
	QuoteDefaultValue bool
	DoNotMarkup       bool
	RemoveBackticks   bool
}
