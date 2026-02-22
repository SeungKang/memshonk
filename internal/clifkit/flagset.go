package clifkit

import (
	"flag"
	"fmt"
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
	internal *flag.FlagSet
	required []string
	nonflags []nonflagDef
}

// Actual returns the underlying flag.FlagSet.
func (o *FlagSet) Actual() *flag.FlagSet {
	return o.internal
}

// Parse parses flag definitions from the argument list.
// After parsing flags, it processes positional arguments and validates
// that all required flags and positional arguments were provided.
func (o *FlagSet) Parse(arguments []string) error {
	err := o.internal.Parse(arguments)
	if err != nil {
		return err
	}

	// Build a set of flags that were actually set
	set := make(map[string]bool)
	o.internal.Visit(func(f *flag.Flag) {
		set[f.Name] = true
	})

	// Check that all required flags were provided
	for _, name := range o.required {
		if !set[name] {
			return fmt.Errorf("required flag not provided: -%s", name)
		}
	}

	// Process positional arguments
	args := o.internal.Args()
	argIdx := 0
	for _, nf := range o.nonflags {
		if nf.isSlice {
			// Slice consumes all remaining arguments
			for argIdx < len(args) {
				err = nf.setter(args[argIdx])
				if err != nil {
					return fmt.Errorf("invalid value %q for %s: %w",
						args[argIdx], nf.name, err)
				}
				argIdx++
			}
		} else {
			if argIdx >= len(args) {
				if nf.required {
					return fmt.Errorf("required argument not provided: %s",
						nf.name)
				}
				continue
			}
			err = nf.setter(args[argIdx])
			if err != nil {
				return fmt.Errorf("invalid value %q for %s: %w",
					args[argIdx], nf.name, err)
			}
			argIdx++
		}
	}

	return nil
}

// addShort adds a short alias for a flag if the name is multi-character
// and the short form is not already registered.
func (o *FlagSet) addShort(name string, adder func(short string)) {
	if len(name) > 1 {
		short := name[:1]
		if o.internal.Lookup(short) == nil {
			adder(short)
		}
	}
}

// trackRequired adds the flag name to the required list if cfg.Required is true.
func (o *FlagSet) trackRequired(cfg FlagConfig) {
	if cfg.Required {
		o.required = append(o.required, cfg.Name)
	}
}
