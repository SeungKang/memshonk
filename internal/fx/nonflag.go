package fx

import (
	"strconv"
	"time"
)

// nonflagDef describes a positional (non-flag) argument.
type nonflagDef struct {
	name     string
	required bool
	isSlice  bool
	setter   func(string) error
}

// IntNf defines a positional int argument with the specified config.
func (o *FlagSet) IntNf(p *int, cfg FlagConfig) {
	o.nonflags = append(o.nonflags, nonflagDef{
		name:     cfg.Name,
		required: cfg.Required,
		isSlice:  false,
		setter: func(s string) error {
			val, err := strconv.Atoi(s)
			if err != nil {
				return err
			}
			*p = val
			return nil
		},
	})
}

// Int64Nf defines a positional int64 argument with the specified config.
func (o *FlagSet) Int64Nf(p *int64, cfg FlagConfig) {
	o.nonflags = append(o.nonflags, nonflagDef{
		name:     cfg.Name,
		required: cfg.Required,
		isSlice:  false,
		setter: func(s string) error {
			val, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return err
			}
			*p = val
			return nil
		},
	})
}

// UintNf defines a positional uint argument with the specified config.
func (o *FlagSet) UintNf(p *uint, cfg FlagConfig) {
	o.nonflags = append(o.nonflags, nonflagDef{
		name:     cfg.Name,
		required: cfg.Required,
		isSlice:  false,
		setter: func(s string) error {
			val, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				return err
			}
			*p = uint(val)
			return nil
		},
	})
}

// Uint64Nf defines a positional uint64 argument with the specified config.
func (o *FlagSet) Uint64Nf(p *uint64, cfg FlagConfig) {
	o.nonflags = append(o.nonflags, nonflagDef{
		name:     cfg.Name,
		required: cfg.Required,
		isSlice:  false,
		setter: func(s string) error {
			val, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				return err
			}
			*p = val
			return nil
		},
	})
}

// StringNf defines a positional string argument with the specified config.
func (o *FlagSet) StringNf(p *string, cfg FlagConfig) {
	o.nonflags = append(o.nonflags, nonflagDef{
		name:     cfg.Name,
		required: cfg.Required,
		isSlice:  false,
		setter: func(s string) error {
			*p = s
			return nil
		},
	})
}

// Float64Nf defines a positional float64 argument with the specified config.
func (o *FlagSet) Float64Nf(p *float64, cfg FlagConfig) {
	o.nonflags = append(o.nonflags, nonflagDef{
		name:     cfg.Name,
		required: cfg.Required,
		isSlice:  false,
		setter: func(s string) error {
			val, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return err
			}
			*p = val
			return nil
		},
	})
}

// DurationNf defines a positional time.Duration argument with the
// specified config.
func (o *FlagSet) DurationNf(p *time.Duration, cfg FlagConfig) {
	o.nonflags = append(o.nonflags, nonflagDef{
		name:     cfg.Name,
		required: cfg.Required,
		isSlice:  false,
		setter: func(s string) error {
			val, err := time.ParseDuration(s)
			if err != nil {
				return err
			}
			*p = val
			return nil
		},
	})
}

// IntSliceNf defines a positional int slice argument that consumes all
// remaining positional arguments.
func (o *FlagSet) IntSliceNf(p *[]int, cfg FlagConfig) {
	o.nonflags = append(o.nonflags, nonflagDef{
		name:     cfg.Name,
		required: cfg.Required,
		isSlice:  true,
		setter: func(s string) error {
			val, err := strconv.Atoi(s)
			if err != nil {
				return err
			}
			*p = append(*p, val)
			return nil
		},
	})
}

// Int64SliceNf defines a positional int64 slice argument that consumes all
// remaining positional arguments.
func (o *FlagSet) Int64SliceNf(p *[]int64, cfg FlagConfig) {
	o.nonflags = append(o.nonflags, nonflagDef{
		name:     cfg.Name,
		required: cfg.Required,
		isSlice:  true,
		setter: func(s string) error {
			val, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return err
			}
			*p = append(*p, val)
			return nil
		},
	})
}

// UintSliceNf defines a positional uint slice argument that consumes all
// remaining positional arguments.
func (o *FlagSet) UintSliceNf(p *[]uint, cfg FlagConfig) {
	o.nonflags = append(o.nonflags, nonflagDef{
		name:     cfg.Name,
		required: cfg.Required,
		isSlice:  true,
		setter: func(s string) error {
			val, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				return err
			}
			*p = append(*p, uint(val))
			return nil
		},
	})
}

// Uint64SliceNf defines a positional uint64 slice argument that consumes
// all remaining positional arguments.
func (o *FlagSet) Uint64SliceNf(p *[]uint64, cfg FlagConfig) {
	o.nonflags = append(o.nonflags, nonflagDef{
		name:     cfg.Name,
		required: cfg.Required,
		isSlice:  true,
		setter: func(s string) error {
			val, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				return err
			}
			*p = append(*p, val)
			return nil
		},
	})
}

// StringSliceNf defines a positional string slice argument that consumes
// all remaining positional arguments.
func (o *FlagSet) StringSliceNf(p *[]string, cfg FlagConfig) {
	o.nonflags = append(o.nonflags, nonflagDef{
		name:     cfg.Name,
		required: cfg.Required,
		isSlice:  true,
		setter: func(s string) error {
			*p = append(*p, s)
			return nil
		},
	})
}

// Float64SliceNf defines a positional float64 slice argument that consumes
// all remaining positional arguments.
func (o *FlagSet) Float64SliceNf(p *[]float64, cfg FlagConfig) {
	o.nonflags = append(o.nonflags, nonflagDef{
		name:     cfg.Name,
		required: cfg.Required,
		isSlice:  true,
		setter: func(s string) error {
			val, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return err
			}
			*p = append(*p, val)
			return nil
		},
	})
}

// DurationSliceNf defines a positional time.Duration slice argument that
// consumes all remaining positional arguments.
func (o *FlagSet) DurationSliceNf(p *[]time.Duration, cfg FlagConfig) {
	o.nonflags = append(o.nonflags, nonflagDef{
		name:     cfg.Name,
		required: cfg.Required,
		isSlice:  true,
		setter: func(s string) error {
			val, err := time.ParseDuration(s)
			if err != nil {
				return err
			}
			*p = append(*p, val)
			return nil
		},
	})
}
