package fx

import (
	"strconv"
	"strings"
	"time"
)

// intSliceValue implements flag.Value for []int.
type intSliceValue struct {
	p *[]int
}

func (o *intSliceValue) String() string {
	if o.p == nil || len(*o.p) == 0 {
		return ""
	}
	strs := make([]string, len(*o.p))
	for i, val := range *o.p {
		strs[i] = strconv.Itoa(val)
	}
	return strings.Join(strs, ",")
}

func (o *intSliceValue) Set(s string) error {
	val, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	*o.p = append(*o.p, val)
	return nil
}

// int64SliceValue implements flag.Value for []int64.
type int64SliceValue struct {
	p *[]int64
}

func (o *int64SliceValue) String() string {
	if o.p == nil || len(*o.p) == 0 {
		return ""
	}
	strs := make([]string, len(*o.p))
	for i, val := range *o.p {
		strs[i] = strconv.FormatInt(val, 10)
	}
	return strings.Join(strs, ",")
}

func (o *int64SliceValue) Set(s string) error {
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*o.p = append(*o.p, val)
	return nil
}

// uintSliceValue implements flag.Value for []uint.
type uintSliceValue struct {
	p *[]uint
}

func (o *uintSliceValue) String() string {
	if o.p == nil || len(*o.p) == 0 {
		return ""
	}
	strs := make([]string, len(*o.p))
	for i, val := range *o.p {
		strs[i] = strconv.FormatUint(uint64(val), 10)
	}
	return strings.Join(strs, ",")
}

func (o *uintSliceValue) Set(s string) error {
	val, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return err
	}
	*o.p = append(*o.p, uint(val))
	return nil
}

// uint64SliceValue implements flag.Value for []uint64.
type uint64SliceValue struct {
	p *[]uint64
}

func (o *uint64SliceValue) String() string {
	if o.p == nil || len(*o.p) == 0 {
		return ""
	}
	strs := make([]string, len(*o.p))
	for i, val := range *o.p {
		strs[i] = strconv.FormatUint(val, 10)
	}
	return strings.Join(strs, ",")
}

func (o *uint64SliceValue) Set(s string) error {
	val, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return err
	}
	*o.p = append(*o.p, val)
	return nil
}

// stringSliceValue implements flag.Value for []string.
type stringSliceValue struct {
	p *[]string
}

func (o *stringSliceValue) String() string {
	if o.p == nil || len(*o.p) == 0 {
		return ""
	}
	return strings.Join(*o.p, ",")
}

func (o *stringSliceValue) Set(s string) error {
	*o.p = append(*o.p, s)
	return nil
}

// float64SliceValue implements flag.Value for []float64.
type float64SliceValue struct {
	p *[]float64
}

func (o *float64SliceValue) String() string {
	if o.p == nil || len(*o.p) == 0 {
		return ""
	}
	strs := make([]string, len(*o.p))
	for i, val := range *o.p {
		strs[i] = strconv.FormatFloat(val, 'g', -1, 64)
	}
	return strings.Join(strs, ",")
}

func (o *float64SliceValue) Set(s string) error {
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	*o.p = append(*o.p, val)
	return nil
}

// durationSliceValue implements flag.Value for []time.Duration.
type durationSliceValue struct {
	p *[]time.Duration
}

func (o *durationSliceValue) String() string {
	if o.p == nil || len(*o.p) == 0 {
		return ""
	}
	strs := make([]string, len(*o.p))
	for i, val := range *o.p {
		strs[i] = val.String()
	}
	return strings.Join(strs, ",")
}

func (o *durationSliceValue) Set(s string) error {
	val, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*o.p = append(*o.p, val)
	return nil
}

// IntSliceFlag defines a repeatable int flag with the specified config.
// Each occurrence of the flag appends to the slice.
//
// A short alias using the first character is added if available.
func (o *FlagSet) IntSliceFlag(p *[]int, cfg ArgConfig) {
	v := &intSliceValue{p: p}
	o.internal.Var(v, cfg.Name, cfg.Description)
	o.addShort(cfg.Name, func(short string) {
		o.internal.Var(v, short, cfg.Description)
	})
	o.trackRequired(cfg)
}

// Int64SliceFlag defines a repeatable int64 flag with the specified
// config. Each occurrence of the flag appends to the slice.
//
// A short alias using the first character is added if available.
func (o *FlagSet) Int64SliceFlag(p *[]int64, cfg ArgConfig) {
	v := &int64SliceValue{p: p}
	o.internal.Var(v, cfg.Name, cfg.Description)
	o.addShort(cfg.Name, func(short string) {
		o.internal.Var(v, short, cfg.Description)
	})
	o.trackRequired(cfg)
}

// UintSliceFlag defines a repeatable uint flag with the specified config.
// Each occurrence of the flag appends to the slice.
//
// A short alias using the first character is added if available.
func (o *FlagSet) UintSliceFlag(p *[]uint, cfg ArgConfig) {
	v := &uintSliceValue{p: p}
	o.internal.Var(v, cfg.Name, cfg.Description)
	o.addShort(cfg.Name, func(short string) {
		o.internal.Var(v, short, cfg.Description)
	})
	o.trackRequired(cfg)
}

// Uint64SliceFlag defines a repeatable uint64 flag with the specified
// config. Each occurrence of the flag appends to the slice.
//
// A short alias using the first character is added if available.
func (o *FlagSet) Uint64SliceFlag(p *[]uint64, cfg ArgConfig) {
	v := &uint64SliceValue{p: p}
	o.internal.Var(v, cfg.Name, cfg.Description)
	o.addShort(cfg.Name, func(short string) {
		o.internal.Var(v, short, cfg.Description)
	})
	o.trackRequired(cfg)
}

// StringSliceFlag defines a repeatable string flag with the specified
// config. Each occurrence of the flag appends to the slice.
//
// A short alias using the first character is added if available.
func (o *FlagSet) StringSliceFlag(p *[]string, cfg ArgConfig) {
	v := &stringSliceValue{p: p}
	o.internal.Var(v, cfg.Name, cfg.Description)
	o.addShort(cfg.Name, func(short string) {
		o.internal.Var(v, short, cfg.Description)
	})
	o.trackRequired(cfg)
}

// Float64SliceFlag defines a repeatable float64 flag with the specified
// config. Each occurrence of the flag appends to the slice.
//
// A short alias using the first character is added if available.
func (o *FlagSet) Float64SliceFlag(p *[]float64, cfg ArgConfig) {
	v := &float64SliceValue{p: p}
	o.internal.Var(v, cfg.Name, cfg.Description)
	o.addShort(cfg.Name, func(short string) {
		o.internal.Var(v, short, cfg.Description)
	})
	o.trackRequired(cfg)
}

// DurationSliceFlag defines a repeatable time.Duration flag with the
// specified config. Each occurrence of the flag appends to the slice.
//
// A short alias using the first character is added if available.
func (o *FlagSet) DurationSliceFlag(p *[]time.Duration, cfg ArgConfig) {
	v := &durationSliceValue{p: p}
	o.internal.Var(v, cfg.Name, cfg.Description)
	o.addShort(cfg.Name, func(short string) {
		o.internal.Var(v, short, cfg.Description)
	})
	o.trackRequired(cfg)
}
