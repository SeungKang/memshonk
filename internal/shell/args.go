package shell

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/SeungKang/memshonk/internal/commands"
)

// NewArgParser creates a new argument parser for the given schema.
func NewArgParser(schema commands.CommandSchema) *ArgParser {
	return &ArgParser{schema: schema}
}

// ArgParser parses command-line arguments according to a CommandSchema.
type ArgParser struct {
	schema commands.CommandSchema
}

// Parse parses the given arguments and returns a CommandConfig.
func (o *ArgParser) Parse(args []string) (commands.CommandConfig, error) {
	flagValues := make(map[string]interface{})
	argValues := make(map[string]interface{})

	// Initialize flags with default values
	for _, flag := range o.schema.Flags {
		if flag.DefaultVal != nil {
			flagValues[flag.Long] = flag.DefaultVal
		} else {
			// Set zero values based on type
			switch flag.DataType.(type) {
			case bool:
				flagValues[flag.Long] = false
			case string:
				flagValues[flag.Long] = ""
			case int:
				flagValues[flag.Long] = 0
			case int64:
				flagValues[flag.Long] = int64(0)
			case uint:
				flagValues[flag.Long] = uint(0)
			case uint64:
				flagValues[flag.Long] = uint64(0)
			}
		}
	}

	// Initialize non-flags with default values
	for _, nf := range o.schema.NonFlags {
		if nf.DefValue != nil {
			argValues[nf.Name] = nf.DefValue
		} else {
			switch nf.DataType.(type) {
			case string:
				argValues[nf.Name] = ""
			case []string:
				argValues[nf.Name] = []string{}
			case bool:
				argValues[nf.Name] = false
			case int:
				argValues[nf.Name] = 0
			case []int:
				argValues[nf.Name] = []int{}
			case int64:
				argValues[nf.Name] = int64(0)
			case []int64:
				argValues[nf.Name] = []int64{}
			case uint:
				argValues[nf.Name] = uint(0)
			case []uint:
				argValues[nf.Name] = []uint{}
			case uint64:
				argValues[nf.Name] = uint64(0)
			case []uint64:
				argValues[nf.Name] = []uint64{}
			}
		}
	}

	// Parse flags and collect positional arguments
	var positionalArgs []string
	i := 0
	for i < len(args) {
		arg := args[i]

		if strings.HasPrefix(arg, "--") {
			// Long flag
			flagName := strings.TrimPrefix(arg, "--")
			if err := o.parseFlag(flagName, args, &i, flagValues, false); err != nil {
				return commands.CommandConfig{}, err
			}
		} else if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			// Short flag
			flagName := strings.TrimPrefix(arg, "-")
			if err := o.parseFlag(flagName, args, &i, flagValues, true); err != nil {
				return commands.CommandConfig{}, err
			}
		} else {
			// Positional argument
			positionalArgs = append(positionalArgs, arg)
			i++
		}
	}

	// Assign positional arguments to non-flags
	if err := o.assignPositionalArgs(positionalArgs, argValues); err != nil {
		return commands.CommandConfig{}, err
	}

	return commands.CommandConfig{
		Flags:    &simpleFlagFetcher{data: flagValues},
		NonFlags: &simpleArgFetcher{data: argValues},
	}, nil
}

// parseFlag parses a single flag and its value.
func (o *ArgParser) parseFlag(name string, args []string, idx *int, values map[string]interface{}, isShort bool) error {
	// Find the flag schema
	var flagSchema *commands.FlagSchema
	for i := range o.schema.Flags {
		f := &o.schema.Flags[i]
		if (isShort && f.Short == name) || (!isShort && f.Long == name) {
			flagSchema = f
			break
		}
	}

	if flagSchema == nil {
		return fmt.Errorf("unknown flag: %q", name)
	}

	canonicalName := flagSchema.Long

	// Handle boolean flags (no value needed)
	if _, ok := flagSchema.DataType.(bool); ok {
		values[canonicalName] = true
		*idx++
		return nil
	}

	// Other flags need a value
	*idx++
	if *idx >= len(args) {
		return fmt.Errorf("flag %q requires a value", name)
	}

	valueStr := args[*idx]
	*idx++

	// Parse value based on type
	switch flagSchema.DataType.(type) {
	case string:
		values[canonicalName] = valueStr
	case int:
		v, err := strconv.Atoi(valueStr)
		if err != nil {
			return fmt.Errorf("invalid int value for flag %q: %q", name, valueStr)
		}
		values[canonicalName] = v
	case int64:
		v, err := strconv.ParseInt(valueStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid int64 value for flag %q: %q", name, valueStr)
		}
		values[canonicalName] = v
	case uint:
		v, err := strconv.ParseUint(valueStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid uint value for flag %q: %q", name, valueStr)
		}
		values[canonicalName] = uint(v)
	case uint64:
		v, err := strconv.ParseUint(valueStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid uint64 value for flag %q: %q", name, valueStr)
		}
		values[canonicalName] = v
	default:
		return fmt.Errorf("unsupported flag type for %q", name)
	}

	return nil
}

// assignPositionalArgs assigns positional arguments to non-flag parameters.
func (o *ArgParser) assignPositionalArgs(args []string, values map[string]interface{}) error {
	argIdx := 0
	for _, nf := range o.schema.NonFlags {
		if argIdx >= len(args) {
			break
		}

		switch nf.DataType.(type) {
		case string:
			values[nf.Name] = args[argIdx]
			argIdx++
		case []string:
			// String list consumes remaining arguments
			values[nf.Name] = args[argIdx:]
			argIdx = len(args)
		case bool:
			v, err := strconv.ParseBool(args[argIdx])
			if err != nil {
				return fmt.Errorf("invalid bool value for %q: %q", nf.Name, args[argIdx])
			}
			values[nf.Name] = v
			argIdx++
		case int:
			v, err := strconv.Atoi(args[argIdx])
			if err != nil {
				return fmt.Errorf("invalid int value for %q: %q", nf.Name, args[argIdx])
			}
			values[nf.Name] = v
			argIdx++
		case []int:
			var intList []int
			for _, s := range args[argIdx:] {
				v, err := strconv.Atoi(s)
				if err != nil {
					return fmt.Errorf("invalid int value for %q: %q", nf.Name, s)
				}
				intList = append(intList, v)
			}
			values[nf.Name] = intList
			argIdx = len(args)
		case int64:
			v, err := strconv.ParseInt(args[argIdx], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid int64 value for %q: %q", nf.Name, args[argIdx])
			}
			values[nf.Name] = v
			argIdx++
		case []int64:
			var intList []int64
			for _, s := range args[argIdx:] {
				v, err := strconv.ParseInt(s, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid int64 value for %q: %q", nf.Name, s)
				}
				intList = append(intList, v)
			}
			values[nf.Name] = intList
			argIdx = len(args)
		case uint:
			v, err := strconv.ParseUint(args[argIdx], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid uint value for %q: %q", nf.Name, args[argIdx])
			}
			values[nf.Name] = uint(v)
			argIdx++
		case []uint:
			var uintList []uint
			for _, s := range args[argIdx:] {
				v, err := strconv.ParseUint(s, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid uint value for %q: %q", nf.Name, s)
				}
				uintList = append(uintList, uint(v))
			}
			values[nf.Name] = uintList
			argIdx = len(args)
		case uint64:
			v, err := strconv.ParseUint(args[argIdx], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid uint64 value for %q: %q", nf.Name, args[argIdx])
			}
			values[nf.Name] = v
			argIdx++
		case []uint64:
			var uintList []uint64
			for _, s := range args[argIdx:] {
				v, err := strconv.ParseUint(s, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid uint64 value for %q: %q", nf.Name, s)
				}
				uintList = append(uintList, v)
			}
			values[nf.Name] = uintList
			argIdx = len(args)
		}
	}

	return nil
}

// simpleFlagFetcher implements commands.FlagFetcher.
type simpleFlagFetcher struct {
	data map[string]interface{}
}

func (o *simpleFlagFetcher) Bool(name string) bool {
	if v, ok := o.data[name]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func (o *simpleFlagFetcher) String(name string) string {
	if v, ok := o.data[name]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (o *simpleFlagFetcher) Int(name string) int {
	if v, ok := o.data[name]; ok {
		if i, ok := v.(int); ok {
			return i
		}
	}
	return 0
}

func (o *simpleFlagFetcher) Int64(name string) int64 {
	if v, ok := o.data[name]; ok {
		if i, ok := v.(int64); ok {
			return i
		}
	}
	return 0
}

func (o *simpleFlagFetcher) Uint(name string) uint {
	if v, ok := o.data[name]; ok {
		if u, ok := v.(uint); ok {
			return u
		}
	}
	return 0
}

func (o *simpleFlagFetcher) Uint64(name string) uint64 {
	if v, ok := o.data[name]; ok {
		if u, ok := v.(uint64); ok {
			return u
		}
	}
	return 0
}

// simpleArgFetcher implements commands.ArgFetcher.
type simpleArgFetcher struct {
	data map[string]interface{}
}

func (o *simpleArgFetcher) Bool(name string) bool {
	if v, ok := o.data[name]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func (o *simpleArgFetcher) String(name string) string {
	if v, ok := o.data[name]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (o *simpleArgFetcher) Int(name string) int {
	if v, ok := o.data[name]; ok {
		if i, ok := v.(int); ok {
			return i
		}
	}
	return 0
}

func (o *simpleArgFetcher) Int64(name string) int64 {
	if v, ok := o.data[name]; ok {
		if i, ok := v.(int64); ok {
			return i
		}
	}
	return 0
}

func (o *simpleArgFetcher) Uint(name string) uint {
	if v, ok := o.data[name]; ok {
		if u, ok := v.(uint); ok {
			return u
		}
	}
	return 0
}

func (o *simpleArgFetcher) Uint64(name string) uint64 {
	if v, ok := o.data[name]; ok {
		if u, ok := v.(uint64); ok {
			return u
		}
	}
	return 0
}

func (o *simpleArgFetcher) StringList(name string) []string {
	if v, ok := o.data[name]; ok {
		if sl, ok := v.([]string); ok {
			return sl
		}
	}
	return nil
}
