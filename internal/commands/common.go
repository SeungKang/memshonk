package commands

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/SeungKang/memshonk/internal/hexdump"
)

// Various encoding formats.
const (
	rawEncoding     = "raw"
	binaryEncoding  = "binary"
	hexEncoding     = "hex"
	hexdumpEncoding = "hexdump"
	base64Encoding  = "base64"
	b64Encoding     = "b64"
)

// Various data types.
const (
	rawDataType = "raw"

	stringDataType   = "string"
	stringleDataType = "stringle"
	stringbeDataType = "stringbe"
	utf8DataType     = "utf8"
	utf8leDataType   = "utf8le"
	utf8beDataType   = "utf8be"

	cstringDataType   = "cstring"
	cstringleDataType = "cstringle"
	cstringbeDataType = "cstringbe"

	utf16DataType     = "utf16"
	utf16leDataType   = "utf16le"
	wstringleDataType = "wstringle"
	wstringDataType   = "wstring"
	wstringbeDataType = "wstringbe"
	utf16beDataType   = "utf16be"

	uint16DataType   = "uint16"
	uint16leDataType = "uint16le"
	uint16beDataType = "uint16be"

	uint32DataType   = "uint32"
	uint32leDataType = "uint32le"
	uint32beDataType = "uint32be"

	uint64DataType   = "uint64"
	uint64leDataType = "uint64le"
	uint64beDataType = "uint64be"

	float32DataType   = "float32"
	float32leDataType = "float32le"
	float32beDataType = "float32be"

	float64DataType   = "float64"
	float64leDataType = "float64le"
	float64beDataType = "float64be"

	patternDataType = "pattern"
)

var (
	errCommandNeedsTerminal = errors.New("this command requires a terminal, but the session does not provide a terminal")
)

func hexdumpStyle(paramStr string) (hexdump.Style, error) {
	params := parseOutputParams(paramStr)

	var unknownKeyErrs []string
	for k, v := range params {
		switch k {
		case "s":
		default:
			unknownKeyErrs = append(unknownKeyErrs, fmt.Sprintf("unknown key %q (from %s=%s)", k, k, v))
		}
	}
	if len(unknownKeyErrs) > 0 {
		sort.Strings(unknownKeyErrs)
		return nil, fmt.Errorf("%s", strings.Join(unknownKeyErrs, ", "))
	}

	s, ok := params["s"]
	if !ok || s == "" {
		return hexdump.DefaultStyle{Colors: hexdump.NewByteColors()}, nil
	}
	switch s {
	case "heap":
		return hexdump.HeapStyle{Colors: hexdump.NewByteColors()}, nil
	default:
		return nil, fmt.Errorf("unknown style %q (from -p s=%s)", s, s)
	}
}

func parseOutputParams(raw string) map[string]string {
	result := make(map[string]string)
	for _, pair := range strings.Split(raw, ",") {
		kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(kv) == 2 {
			result[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return result
}
