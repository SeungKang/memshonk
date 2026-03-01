package shell

import (
	"sort"
	"strings"
	"unicode"

	"github.com/SeungKang/memshonk/internal/apicompat"
)

// NewCompleter creates a new completer for the given command registry.
func NewCompleter(registry *apicompat.CommandRegistry) *Completer {
	return &Completer{registry: registry}
}

// Completer implements readline.AutoCompleter for the shell.
type Completer struct {
	registry *apicompat.CommandRegistry
}

// Do implements readline.AutoCompleter.
// It returns completion candidates for the current line and cursor position.
func (o *Completer) Do(line []rune, pos int) ([][]rune, int) {
	// Get the text up to the cursor
	lineStr := string(line[:pos])

	// Split into words, respecting quotes
	words := splitWords(lineStr)

	// Find the word being completed and its prefix
	var prefix string

	if len(lineStr) > 0 && !unicode.IsSpace(rune(lineStr[len(lineStr)-1])) {
		// We're in the middle of a word
		if len(words) > 0 {
			prefix = words[len(words)-1]
		}
	}

	// Determine what to complete
	wordCount := len(words)
	if prefix != "" {
		wordCount-- // Don't count the partial word
	}

	var candidates []string

	if wordCount == 0 {
		// Complete command names
		candidates = o.completeCommandNames(prefix)
	} else {
		// TODO: Maybe implement command argument completion someday?
		// Complete flags or arguments for a command
		//cmdName := words[0]
		//candidates = o.completeCommandArgs(cmdName, prefix, words[1:])
	}

	if len(candidates) == 0 {
		return nil, 0
	}

	// Convert to rune slices, removing the prefix
	result := make([][]rune, len(candidates))
	for i, cand := range candidates {
		suffix := strings.TrimPrefix(cand, prefix)
		result[i] = []rune(suffix)
	}

	return result, len(prefix)
}

// completeCommandNames returns command names matching the prefix.
func (o *Completer) completeCommandNames(prefix string) []string {
	allNames := o.registry.AllNamesAndAliases()
	var matches []string

	for _, name := range allNames {
		if strings.HasPrefix(name, prefix) {
			matches = append(matches, name)
		}
	}

	sort.Strings(matches)
	return matches
}

// splitWords splits a string into words, handling quoted strings.
func splitWords(s string) []string {
	var words []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, r := range s {
		switch {
		case r == '"' || r == '\'':
			if inQuote && r == quoteChar {
				inQuote = false
				quoteChar = 0
			} else if !inQuote {
				inQuote = true
				quoteChar = r
			} else {
				current.WriteRune(r)
			}
		case unicode.IsSpace(r) && !inQuote:
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}
