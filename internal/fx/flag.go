package fx

import (
	"encoding"
	"flag"
	"time"
)

// ArgConfig configures an argument's name, description, and required status.
type ArgConfig struct {
	Name        string
	Description string
	Required    bool

	OptShortName string
}

// BoolFlag defines a bool flag with the specified default value and config.
//
// A short alias using the first character is added if available.
func (o *FlagSet) BoolFlag(p *bool, defValue bool, cfg ArgConfig) {
	o.internal.BoolVar(p, cfg.Name, defValue, cfg.Description)

	o.registerFlag(registerFlagConfig{
		argConfig: cfg,
		optAddShortFn: func(short string) {
			o.internal.BoolVar(p, short, defValue, cfg.Description)
		},
	})
}

// IntFlag defines an int flag with the specified default value and config.
//
// A short alias using the first character is added if available.
func (o *FlagSet) IntFlag(p *int, defValue int, cfg ArgConfig) {
	o.internal.IntVar(p, cfg.Name, defValue, cfg.Description)

	o.registerFlag(registerFlagConfig{
		argConfig: cfg,
		optAddShortFn: func(short string) {
			o.internal.IntVar(p, short, defValue, cfg.Description)
		},
	})
}

// Int64Flag defines an int64 flag with the specified default value and config.
//
// A short alias using the first character is added if available.
func (o *FlagSet) Int64Flag(p *int64, defValue int64, cfg ArgConfig) {
	o.internal.Int64Var(p, cfg.Name, defValue, cfg.Description)

	o.registerFlag(registerFlagConfig{
		argConfig: cfg,
		optAddShortFn: func(short string) {
			o.internal.Int64Var(p, short, defValue, cfg.Description)
		},
	})
}

// UintFlag defines a uint flag with the specified default value and config.
//
// A short alias using the first character is added if available.
func (o *FlagSet) UintFlag(p *uint, defValue uint, cfg ArgConfig) {
	o.internal.UintVar(p, cfg.Name, defValue, cfg.Description)

	o.registerFlag(registerFlagConfig{
		argConfig: cfg,
		optAddShortFn: func(short string) {
			o.internal.UintVar(p, short, defValue, cfg.Description)
		},
	})
}

// Uint64Flag defines a uint64 flag with the specified default value and config.
//
// A short alias using the first character is added if available.
func (o *FlagSet) Uint64Flag(p *uint64, defValue uint64, cfg ArgConfig) {
	o.internal.Uint64Var(p, cfg.Name, defValue, cfg.Description)

	o.registerFlag(registerFlagConfig{
		argConfig: cfg,
		optAddShortFn: func(short string) {
			o.internal.Uint64Var(p, short, defValue, cfg.Description)
		},
	})
}

// StringFlag defines a string flag with the specified default value and config.
//
// A short alias using the first character is added if available.
func (o *FlagSet) StringFlag(p *string, defValue string, cfg ArgConfig) {
	o.internal.StringVar(p, cfg.Name, defValue, cfg.Description)

	o.registerFlag(registerFlagConfig{
		argConfig: cfg,
		optAddShortFn: func(short string) {
			o.internal.StringVar(p, short, defValue, cfg.Description)
		},
	})
}

// Float64Flag defines a float64 flag with the specified default value
// and config.
//
// A short alias using the first character is added if available.
func (o *FlagSet) Float64Flag(p *float64, defValue float64, cfg ArgConfig) {
	o.internal.Float64Var(p, cfg.Name, defValue, cfg.Description)

	o.registerFlag(registerFlagConfig{
		argConfig: cfg,
		optAddShortFn: func(short string) {
			o.internal.Float64Var(p, short, defValue, cfg.Description)
		},
	})
}

// DurationFlag defines a time.Duration flag with the specified default
// value and config.
//
// A short alias using the first character is added if available.
func (o *FlagSet) DurationFlag(p *time.Duration, defValue time.Duration, cfg ArgConfig) {
	o.internal.DurationVar(p, cfg.Name, defValue, cfg.Description)

	o.registerFlag(registerFlagConfig{
		argConfig: cfg,
		optAddShortFn: func(short string) {
			o.internal.DurationVar(p, short, defValue, cfg.Description)
		},
	})
}

// TextFlag defines a flag with the specified default value and config.
//
// The argument p must implement encoding.TextUnmarshaler, and defValue
// must implement encoding.TextMarshaler.
//
// A short alias using the first character is added if available.
func (o *FlagSet) TextFlag(p encoding.TextUnmarshaler, defValue encoding.TextMarshaler, cfg ArgConfig) {
	o.internal.TextVar(p, cfg.Name, defValue, cfg.Description)

	o.registerFlag(registerFlagConfig{
		argConfig: cfg,
		optAddShortFn: func(short string) {
			o.internal.TextVar(p, short, defValue, cfg.Description)
		},
	})
}

// FuncFlag defines a flag with the specified config.
//
// The fn function is called with the value of the flag when parsing.
//
// A short alias using the first character is added if available.
func (o *FlagSet) FuncFlag(fn func(string) error, cfg ArgConfig) {
	o.internal.Func(cfg.Name, cfg.Description, fn)

	o.registerFlag(registerFlagConfig{
		argConfig: cfg,
		optAddShortFn: func(short string) {
			o.internal.Func(short, cfg.Description, fn)
		},
	})
}

// BoolFuncFlag defines a flag with the specified config without requiring
// a value.
//
// The fn function is called when the flag is parsed.
//
// A short alias using the first character is added if available.
func (o *FlagSet) BoolFuncFlag(fn func(string) error, cfg ArgConfig) {
	o.internal.BoolFunc(cfg.Name, cfg.Description, fn)

	o.registerFlag(registerFlagConfig{
		argConfig: cfg,
		optAddShortFn: func(short string) {
			o.internal.BoolFunc(short, cfg.Description, fn)
		},
	})
}

// VarFlag defines a flag with the specified value and config.
//
// The value must implement the flag.Value interface.
//
// A short alias using the first character is added if available.
func (o *FlagSet) VarFlag(defValue flag.Value, cfg ArgConfig) {
	o.internal.Var(defValue, cfg.Name, cfg.Description)

	o.registerFlag(registerFlagConfig{
		argConfig: cfg,
		optAddShortFn: func(short string) {
			o.internal.Var(defValue, short, cfg.Description)
		},
	})
}
