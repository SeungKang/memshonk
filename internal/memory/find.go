package memory

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strings"
)

func FindAll(pattern string, data []byte) ([]int, error) {
	parsedPattern, err := ParsePattern(pattern)
	if err != nil {
		return nil, nil
	}

	return FindAllParsed(parsedPattern, data)
}

func FindAllParsed(parsedPattern ParsedPattern, data []byte) ([]int, error) {
	var matches []int
	for i := 0; i <= len(data)-parsedPattern.Length; i++ {
		if match(data, i, parsedPattern.Parts) {
			matches = append(matches, i)
		}
	}

	if len(matches) == 0 {
		return nil, errors.New("pattern not found in data")
	}

	return matches, nil
}

func FindAllReader(pattern string, reader *BufferedReader) ([]Pointer, error) {
	parsedPattern, err := ParsePattern(pattern)
	if err != nil {
		return nil, err
	}

	needLength := uint64(parsedPattern.Length * 2)
	reader.SetAdvanceBy(needLength - uint64(parsedPattern.Length))

	var matches []Pointer
	for reader.Next(context.Background(), needLength) {
		chunk := reader.Chunk()
		for i := 0; i <= len(chunk.Data)-parsedPattern.Length; i++ {
			if match(chunk.Data, i, parsedPattern.Parts) {
				matches = append(matches, chunk.Addr.Advance(uint64(i)))
			}
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

type PatternPart struct {
	bytes     []byte
	wildcards int
}

type ParsedPattern struct {
	Parts  []PatternPart
	Length int
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

func match(data []byte, offset int, patternParts []PatternPart) bool {
	pos := offset
	for _, patternPart := range patternParts {
		if !bytes.Equal(data[pos:pos+len(patternPart.bytes)], patternPart.bytes) {
			return false
		}
		pos += len(patternPart.bytes) + patternPart.wildcards
	}
	return true
}
