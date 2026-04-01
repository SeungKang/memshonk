package commands

import (
	"errors"
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
