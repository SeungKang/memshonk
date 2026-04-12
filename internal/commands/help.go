package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/fx"
)

const (
	HelpCommandName = "help"

	addressTopicName       = "address"
	addressTopicReferStr   = `(refer to "` + HelpCommandName + ` ` + addressTopicName + `")`
	datatypesTopicName     = "datatypes"
	datatypesTopicReferStr = `(refer to "` + HelpCommandName + ` ` + datatypesTopicName + `")`
	formatsTopicName       = "formats"
	formatsTopicReferStr   = `(refer to "` + HelpCommandName + ` ` + formatsTopicName + `")`
	patternTopicName       = "pattern"
	patternTopicReferStr   = `(refer to "` + HelpCommandName + ` ` + patternTopicName + `")`

	AppDescription = `  memshonk is an experimental command-line debugger companion that tries to
  fill the functionality gaps between debuggers. Think of it as a cross
  between gdb, rizin, and Cheat Engine. It is not meant to replace
  a debugger, but supplement it.`
)

func NewHelpCommand(config apicompat.NewCommandConfig) *fx.Command {
	cmd := &HelpCommand{
		session: config.Session,
	}

	root := fx.NewCommand(HelpCommandName, "list available commands and help topics", cmd.run)

	root.FlagSet.StringNf(&cmd.optTopic, fx.ArgConfig{
		Name:        "command-or-topic",
		Description: "Optionally display help for a specific command or topic",
	})

	return root
}

type HelpCommand struct {
	session  apicompat.Session
	optTopic string
}

func (o *HelpCommand) run(_ context.Context) (fx.CommandResult, error) {
	var sb strings.Builder

	if o.optTopic == "" {
		// Note: It is important that we render the
		// command list in this method instead of
		// making it part of the command's usage
		// because the PrintUsage will trigger
		// infinite recursive calls to the PrintUsage
		// method... and yeah, no bueno.
		sb.WriteString(`OVERVIEW
` + AppDescription + `

TOPICS
  ` + addressTopicName + `   - How memshonk handles memory addresses (pointer chains)
  ` + datatypesTopicName + ` - Data types usable with various memory manipulation commands
  ` + formatsTopicName + `   - Supported data formatting (encoding) options
  ` + patternTopicName + `   - Pattern string format used in the ` + ScanCommandName + ` command and potentially
              other commands

COMMANDS
`)

		cmds := o.session.SharedState().Commands.AsSlice(o.session)

		longest := 0

		for _, cmd := range cmds {
			cmdNameLen := len(cmd.Name())

			if cmdNameLen > longest {
				longest = cmdNameLen
			}
		}

		for i, cmd := range cmds {
			name := cmd.Name()
			nameLen := len(name)

			var sep string

			switch {
			case nameLen == longest:
				sep = " - "
			case nameLen > longest:
				sep = strings.Repeat(" ", nameLen-longest) + " - "
			case longest > nameLen:
				sep = strings.Repeat(" ", longest-nameLen) + " - "
			}

			sb.WriteString("  " + name + sep + cmd.Description)

			if len(cmds) > 1 && i != len(cmds)-1 {
				sb.WriteByte('\n')
			}
		}

		return fx.NewHumanCommandResult(sb.String()), nil
	}

	switch o.optTopic {
	case formatsTopicName:
		return fx.NewHumanCommandResult(`FORMATS
  ` + rawEncoding + `     - Raw data (i.e., no parsing or validation)
  ` + binaryEncoding + `  - Binary string
  ` + hexEncoding + `     - Hex-encoded string
  ` + hexdumpEncoding + ` - Similar to the output of the "hexdump" program
  ` + base64Encoding + `  - Base64-encoded string
  ` + b64Encoding + `     - Alias to ` + base64Encoding), nil

	case datatypesTopicName:
		return fx.NewHumanCommandResult(`DATA TYPES
  ` + rawDataType + `       - Raw data (i.e., no parsing or validation)
  ` + utf8leDataType + `    - UTF-8 string in little endian byte order
  ` + utf8DataType + `      - Alias to ` + utf8leDataType + `
  ` + utf8beDataType + `    - UTF-8 string in big endian byte order
  ` + stringleDataType + `  - Alias to ` + utf8leDataType + `
  ` + stringDataType + `    - Alias to ` + utf8leDataType + `
  ` + stringbeDataType + `  - Alias to ` + utf8beDataType + `
  ` + utf16leDataType + `   - UTF-16 string in little endian byte order
  ` + utf16DataType + `     - Alias to ` + utf16leDataType + `
  ` + utf16beDataType + `   - UTF-16 string in big endian byte order
  ` + wstringleDataType + ` - Alias to ` + utf16leDataType + `
  ` + wstringDataType + `   - Alias to ` + utf16leDataType + `
  ` + wstringbeDataType + ` - Alias to ` + utf16beDataType + `
  ` + cstringleDataType + ` - Null-terminated string in little endian byte order
  ` + cstringDataType + `   - Alias to ` + cstringleDataType + `
  ` + cstringbeDataType + ` - Null-terminated string in big endian byte order
  ` + float32leDataType + ` - A 32-bit float in little endian byte order
  ` + float32DataType + `   - Alias to ` + float32leDataType + `
  ` + float32beDataType + ` - A 32-bit float in big endian byte order
  ` + float64leDataType + ` - A 64-bit float in little endian byte order
  ` + float64DataType + `   - Alias to ` + float64leDataType + `
  ` + float64beDataType + ` - A 64-bit float in big endian byte order
  ` + patternDataType + `   - Pattern string ` + patternTopicReferStr), nil
	case patternTopicName:
		return fx.NewHumanCommandResult(`PATTERN STRING
  memshonk supports a pattern string format for searching for byte sequences.
  This format is heavily inspired by video game modding tools which employ
  a similar pattern format.

  Users can specify a pattern as a hexadecimal string optionally separated
  by space characters. For example, to match four "A" characters, the
  string would be:
    41 41 41 41

  ... or:
    41414141

  Wildcard values can be specified using "??". For example, to match one
  "A" followed by two bytes of any value and then a "B" the string would
  look like this:
    41 ?? ?? 42`), nil
	case addressTopicName:
		return fx.NewHumanCommandResult(`MEMORY ADDRESSES
  memshonk has a special representation of memory addresses inspired by
  Cheat Engine which are referred to as pointer chains.

  A pointer chain is a way to express navigation to a memory address and can
  be used anywhere memshonk accepts an address (e.g. ` + ReadCommandName + `, ` + WriteCommandName + `).

  This is useful for targeting dynamic addresses that change each time
  a process starts, such as those found with tools like Cheat Engine.
  Instead of an address that shifts around, you provide a stable base and
  a series of offsets that lead to the final address.

FORMAT
  [module:]base[,offset1,offset2,...]

  Note: All values are hexadecimal. The 0x prefix is optional.

COMPONENTS
  module                    optional name of a loaded object (e.g. a DLL or
                            shared library) to use as the base. If omitted,
                            the executable itself is used as the base. Module
                            names can be found using ` + VmmapCommandName + `

  base                      the first address or offset (relative to the
                            module/executable base for a chain, or an absolute
                            address when used alone)

  offset1, offset2, ...     each offset is added to the pointer read from the
                            previous step to arrive at the next address

HOW IT WORKS
  1. Resolve the base address (module/executable base + first offset, or the
     absolute address if no offsets follow)
  2. Read the pointer value stored at that address
  3. Add the next offset to get the next address
  4. Repeat until all offsets are consumed, the result is the final address

EXAMPLES
  0xd5a351                            Absolute address 0xd5a351

  0xd5a351,0x20,0x5,0xC0              Pointer chain starting at offset
                                      0xd5a351 from the executable base

  buh.dll:0x20,0x5,0xC0               Pointer chain starting at offset 0x20
                                      from buh.dll's base

  MassEffect3.exe:0158439C,28,3C,2E8  Pointer chain starting at offset
                                      0x0158439C from MassEffect3.exe's base`), nil
	default:
		cmdFn, found := o.session.SharedState().Commands.Lookup(o.optTopic)
		if !found {
			return nil, fmt.Errorf("unknown help topic or command: %q", o.optTopic)
		}

		cmd := cmdFn(apicompat.NewCommandConfig{Session: o.session})
		cmd.FlagSet.Actual().SetOutput(&sb)
		cmd.PrintUsage()

		return fx.NewHumanCommandResult(strings.TrimRight(sb.String(), "\n")), nil
	}
}
