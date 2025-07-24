package memory

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode/utf16"
)

func FindAllReader(parsedPattern ParsedPattern, reader *BufferedReader) ([]Pointer, error) {
	needLength := uint64(parsedPattern.Length)
	reader.SetAdvanceBy(1)

	i := 0

	var matches []Pointer
	for reader.Next(context.Background(), needLength) {
		chunk := reader.Bytes()

		if match(chunk, parsedPattern) {
			matches = append(matches, reader.Addr())
		}

		i++
	}

	if reader.Err() != nil {
		return nil, reader.Err()
	}

	if len(matches) > 0 {
		return matches, nil
	}

	return nil, nil
}

type ParsedPattern struct {
	Parts  []PatternPart
	Length int
}

type PatternPart struct {
	bytes     []byte
	wildcards int
}

func ParsePatternFromUtf8(s string) (ParsedPattern, error) {
	return ParsedPattern{
		Parts: []PatternPart{
			{
				bytes: []byte(s),
			},
		},
		Length: len(s),
	}, nil
}

func ParsePatternFromUtf16(s string, endianness binary.ByteOrder) (ParsedPattern, error) {
	b := stringToUTF16Bytes(s, endianness)

	return ParsedPattern{
		Parts: []PatternPart{
			{
				bytes: b,
			},
		},
		Length: len(b),
	}, nil
}

func stringToUTF16Bytes(s string, endianness binary.ByteOrder) []byte {
	// Encode string to UTF-16 (as []uint16)
	utf16Units := utf16.Encode([]rune(s))

	// Convert UTF-16 code units to byte slice (little endian)
	b := make([]byte, len(utf16Units)*2)
	for i, v := range utf16Units {
		endianness.PutUint16(b[i*2:], v)
	}

	return b
}

func ParsePattern(pattern string) (ParsedPattern, error) {
	parts := strings.Fields(pattern)
	var result []PatternPart
	current := PatternPart{}

	for _, p := range parts {
		if p == "??" {
			current.wildcards++
			continue
		}

		b, err := hex.DecodeString(strings.TrimPrefix(p, "0x"))
		if err != nil {
			return ParsedPattern{}, fmt.Errorf("failed to hex decode pattern: %q - %w", p, err)
		}

		if len(b) > 1 {
			return ParsedPattern{}, errors.New("pattern part must be only one byte")
		}

		if current.wildcards > 0 {
			result = append(result, current)
			current = PatternPart{}
		}

		current.bytes = append(current.bytes, b[0])
	}

	if len(current.bytes) > 0 || current.wildcards > 0 {
		result = append(result, current)
	}

	return ParsedPattern{
		Parts:  result,
		Length: len(parts),
	}, nil
}

func match(data []byte, parsedPattern ParsedPattern) bool {
	if len(data) != parsedPattern.Length {
		return false
	}

	var pos int
	for _, part := range parsedPattern.Parts {
		if !bytes.Equal(data[pos:pos+len(part.bytes)], part.bytes) {
			return false
		}

		pos += len(part.bytes) + part.wildcards
	}

	return true
}
