package shell

import (
	"sync"

	"github.com/SeungKang/memshonk/internal/commands"
)

// NewCommandRegistry creates a new empty command registry.
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		byName:  make(map[string]commands.CommandSchema),
		aliases: make(map[string]string),
	}
}

// CommandRegistry stores command schemas and allows lookup by name or alias.
// It is safe for concurrent read access after initialization.
type CommandRegistry struct {
	rwMu    sync.RWMutex
	byName  map[string]commands.CommandSchema
	names   []string
	aliases map[string]string // alias -> canonical name
}

// Register adds a command schema to the registry.
func (o *CommandRegistry) Register(schema commands.CommandSchema) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	o.byName[schema.Name] = schema
	o.names = append(o.names, schema.Name)

	for _, alias := range schema.Aliases {
		o.aliases[alias] = schema.Name
	}
}

// Unregister removes a command from the registry by name.
func (o *CommandRegistry) Unregister(name string) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	schema, ok := o.byName[name]
	if !ok {
		return
	}

	delete(o.byName, name)

	// Remove from names slice
	for i, n := range o.names {
		if n == name {
			o.names = append(o.names[:i], o.names[i+1:]...)
			break
		}
	}

	// Remove aliases
	for _, alias := range schema.Aliases {
		delete(o.aliases, alias)
	}
}

// Lookup finds a command schema by name or alias.
// Returns the schema and true if found, or an empty schema and false if not.
func (o *CommandRegistry) Lookup(nameOrAlias string) (commands.CommandSchema, bool) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	// Try direct name lookup first
	if schema, ok := o.byName[nameOrAlias]; ok {
		return schema, true
	}

	// Try alias lookup
	if canonicalName, ok := o.aliases[nameOrAlias]; ok {
		if schema, ok := o.byName[canonicalName]; ok {
			return schema, true
		}
	}

	return commands.CommandSchema{}, false
}

// Names returns all registered command names (not aliases).
func (o *CommandRegistry) Names() []string {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	result := make([]string, len(o.names))
	copy(result, o.names)
	return result
}

// AllNamesAndAliases returns all command names and aliases for completion.
func (o *CommandRegistry) AllNamesAndAliases() []string {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	result := make([]string, 0, len(o.names)+len(o.aliases))
	result = append(result, o.names...)
	for alias := range o.aliases {
		result = append(result, alias)
	}
	return result
}
