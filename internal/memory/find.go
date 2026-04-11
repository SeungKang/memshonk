package memory

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode/utf16"
)

type ScanResult struct {
	Size uint64
	Addr Pointer
}

func FindAllReader(ctx context.Context, parsedPattern ParsedPattern, reader *BufferedReader) ([]ScanResult, error) {
	needLength := uint64(parsedPattern.length)
	reader.SetAdvanceBy(1)

	var matches []ScanResult

	for reader.Next(ctx, needLength) {
		chunk := reader.Bytes()

		if parsedPattern.Matches(chunk) {
			matches = append(matches, ScanResult{
				Size: uint64(len(chunk)),
				Addr: reader.Addr(),
			})
		}
	}

	if reader.Err() != nil {
		return nil, fmt.Errorf("memory reader failed - %w", reader.Err())
	}

	if len(matches) > 0 {
		return matches, nil
	}

	return nil, nil
}

func ParsePatternFromUtf8(s string) (ParsedPattern, error) {
	return PatternForRawBytes([]byte(s)), nil
}

func ParsePatternFromUtf16(s string, endianness binary.ByteOrder) (ParsedPattern, error) {
	return PatternForRawBytes(stringToUTF16Bytes(s, endianness)), nil
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
	var parts []PatternByte

	for _, field := range strings.Fields(pattern) {
		field = strings.TrimPrefix(field, "0x")

		partStrs := splitByStrLen(field, 2)

		for _, partStr := range partStrs {
			part, err := patternByteFromString(partStr)
			if err != nil {
				return ParsedPattern{}, fmt.Errorf("failed to parse pattern byte: %q - %w",
					part, err)
			}

			parts = append(parts, part)
		}
	}

	return ParsedPattern{
		parts:  parts,
		length: len(parts),
	}, nil
}

// Based on work by Igor Mikushkin:
// https://stackoverflow.com/a/61469854
func splitByStrLen(s string, chunkSize int) []string {
	if len(s) == 0 {
		return nil
	}

	if len(s) <= chunkSize {
		return []string{s}
	}

	chunks := make([]string, 0, (len(s)-1)/chunkSize+1)
	currentLen := 0
	currentStart := 0

	for i := range s {
		if currentLen == chunkSize {
			chunks = append(chunks, s[currentStart:i])
			currentLen = 0
			currentStart = i
		}

		currentLen++
	}

	chunks = append(chunks, s[currentStart:])

	return chunks
}

func PatternForRawBytes(b []byte) ParsedPattern {
	parts := make([]PatternByte, len(b))

	for i := range b {
		parts[i].b = b[i]
	}

	return ParsedPattern{
		parts:  parts,
		length: len(parts),
	}
}

type ParsedPattern struct {
	parts  []PatternByte
	length int
}

func (o ParsedPattern) Matches(data []byte) bool {
	if len(data) != o.length || len(data) == 0 || o.length == 0 {
		return false
	}

	for i := range data {
		if !o.parts[i].Matches(data[i]) {
			return false
		}
	}

	return true
}

func patternByteFromString(str string) (PatternByte, error) {
	if len(str) > 2 {
		return PatternByte{}, errors.New("pattern byte string is more than two characters")
	}

	if str == "??" {
		return PatternByte{wildcard: true}, nil
	}

	if len(str) == 1 {
		str = "0" + str
	}

	b, err := hex.DecodeString(str)
	if err != nil {
		return PatternByte{}, fmt.Errorf("hex decode failed - %w", err)
	}

	if len(b) > 1 {
		return PatternByte{}, fmt.Errorf("pattern byte decoded to more than one byte (%d)",
			len(b))
	}

	return PatternByte{b: b[0]}, nil
}

type PatternByte struct {
	b        byte
	wildcard bool
}

func (o PatternByte) Matches(data byte) bool {
	if o.wildcard {
		return true
	}

	return data == o.b
}

func (o PatternByte) String() string {
	if o.wildcard {
		return "??"
	}

	return hex.EncodeToString([]byte{o.b})
}
