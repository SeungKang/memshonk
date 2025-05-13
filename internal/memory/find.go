package memory

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strings"
)

func FindAllReader(pattern string, reader *BufferedReader) ([]Pointer, error) {
	parsedPattern, err := ParsePattern(pattern)
	if err != nil {
		return nil, err
	}

	needLength := uint64(parsedPattern.Length)
	reader.SetAdvanceBy(1)

	var matches []Pointer
	for reader.Next(context.Background(), needLength) {
		chunk := reader.Bytes()

		if match(chunk, parsedPattern) {
			matches = append(matches, reader.Addr())
		}
	}

	if len(matches) > 0 {
		return matches, nil
	}

	if reader.Err() != nil {
		return nil, reader.Err()
	}

	return nil, errors.New("pattern not found in data")
}

type ParsedPattern struct {
	Parts  []PatternPart
	Length int
}

type PatternPart struct {
	bytes     []byte
	wildcards int
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

		b, err := hex.DecodeString(p)
		if err != nil {
			return ParsedPattern{}, err
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
