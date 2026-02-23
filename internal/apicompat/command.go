package apicompat

import (
	"container/list"
	"context"
	"strings"
	"sync"

	"github.com/SeungKang/memshonk/internal/fx"
)

// Command represents a command that can be run by a client.
type Command interface {
	// Name is the name of the command.
	Name() string

	// Run executes the command.
	Run(context.Context, Session) (CommandResult, error)
}

type CommandResult interface {
	Serialize() []byte
}

// NewEmptyCommandRegistry creates a new empty command registry.
func NewEmptyCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		byName:  make(map[string]func(Session) fx.Command),
		aliases: make(map[string]string),
	}
}

// CommandRegistry stores command schemas and allows lookup by name or alias.
// It is safe for concurrent read access after initialization.
type CommandRegistry struct {
	rwMu    sync.RWMutex
	byName  map[string]func(Session) fx.Command
	names   []string
	aliases map[string]string // alias -> canonical name
}

// Register adds a command schema to the registry.
func (o *CommandRegistry) Register(name string, newCommandFn func(Session) fx.Command) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	o.byName[name] = newCommandFn
	o.names = append(o.names, name)
}

// Unregister removes a command from the registry by name.
func (o *CommandRegistry) Unregister(name string) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	_, ok := o.byName[name]
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
	//for _, alias := range schema.Aliases {
	//	delete(o.aliases, alias)
	//}
}

// Lookup finds a command schema by name or alias.
//
// Returns the schema and true if found, or nil and false if not.
func (o *CommandRegistry) Lookup(nameOrAlias string) (func(Session) fx.Command, bool) {
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

	return nil, false
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

// CommandStorage stores information about the session's previously-run
// commands.
type CommandStorage struct {
	rwMu           sync.RWMutex
	namesToOutputs map[string]*list.List
}

func (o *CommandStorage) AddOutput(output fx.CommandResultWrapper) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if o.namesToOutputs == nil {
		o.namesToOutputs = make(map[string]*list.List)
	}

	id := serializeCommandID(output.Commands)

	outputs, hasIt := o.namesToOutputs[id]
	if !hasIt {
		outputs = list.New()

		o.namesToOutputs[id] = outputs
	}

	if outputs.Len() == 2 {
		outputs.Remove(outputs.Back())
	}

	outputs.PushFront(output)
}

func (o *CommandStorage) PreviousOutput(commandID []string) (fx.CommandResultWrapper, bool) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	id := serializeCommandID(commandID)

	outputs, hasAny := o.namesToOutputs[id]
	if !hasAny {
		return fx.CommandResultWrapper{}, false
	}

	output := outputs.Front().Value.(fx.CommandResultWrapper)

	return output, true
}

func serializeCommandID(eachCmd []string) string {
	return strings.Join(eachCmd, "-")
}
