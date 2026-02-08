package flagsctl

import (
	"errors"
	"slices"
	"sort"
	"sync"
)

// Various built-in namespace strings.
const (
	AllNamespaces         = "*"
	SegmentsNamespace     = "segments"
	SectionsNamespace     = "sections"
	ImportedLibsNamespace = "imported_libs"
	ImportedCodeNamespace = "imported_code"
	ExportsNamespace      = "exports"
	SymbolsNamespace      = "symbols"
	FunctionsNamespace    = "functions"
	RelocationsNamespace  = "relocations"
	StringsNamespace      = "strings"
)

var (
	ErrStopIterating = errors.New("stop iterating")
)

func IsBuiltInNamespace(namespace string) bool {
	return slices.Contains(BuiltInNamespaces(), namespace)
}

func BuiltInNamespaces() []string {
	return []string{AllNamespaces, SegmentsNamespace, SectionsNamespace,
		ImportedLibsNamespace, ImportedCodeNamespace,
		ExportsNamespace, SymbolsNamespace, FunctionsNamespace,
		RelocationsNamespace, StringsNamespace}
}

func New() *Ctl {
	return &Ctl{}
}

type Ctl struct {
	rwMu              sync.RWMutex
	namespacesToFlags map[string]FlagList
	deleted           map[string]FlagList
}

type FlagList []Flag

func (o FlagList) Iter(fn func(*Flag) error) error {
	for _, f := range o {
		err := fn(&f)
		if err != nil {
			if errors.Is(err, ErrStopIterating) {
				return nil
			}

			return err
		}

	}

	return nil
}

type Flag struct {
	Name   string
	Offset uint64
	Addr   uint64
	Type   string
	Value  string
	Notes  string

	autoAdded bool
}

func (o *Ctl) Namespaces() []string {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	namespaces := make([]string, 0, len(o.namespacesToFlags))

	for ns := range o.namespacesToFlags {
		namespaces = append(namespaces, ns)
	}

	sort.Strings(namespaces)

	return namespaces
}

func (o *Ctl) FlagsInNamespace(namespace string) []Flag {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	var flags FlagList

	if namespace == AllNamespaces {
		for _, fl := range o.namespacesToFlags {
			flags = append(flags, fl...)
		}
	} else {
		flags, _ = o.namespacesToFlags[namespace]
	}

	return flags
}

func (o *Ctl) AddUserFlag(namespace string, flag Flag) {
	o.addFlag(namespace, flag)
}

func (o *Ctl) AddAutoFlag(namespace string, flag Flag) {
	flag.autoAdded = true

	o.addFlag(namespace, flag)
}

func (o *Ctl) addFlag(namespace string, flag Flag) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	flag.Type = namespace

	if o.namespacesToFlags == nil {
		o.namespacesToFlags = make(map[string]FlagList)
	}

	flags := o.namespacesToFlags[namespace]

	flags = append(flags, flag)

	o.namespacesToFlags[namespace] = flags
}

func (o *Ctl) DeleteNamespace(namespace string) {
	if IsBuiltInNamespace(namespace) {
		return
	}

	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	flags := o.namespacesToFlags[namespace]

	for _, f := range flags {
		o.saveDeleted(namespace, f)
	}

	delete(o.namespacesToFlags, namespace)
}

func (o *Ctl) DeleteFlag(namespace string, flagName string) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if len(o.namespacesToFlags) == 0 {
		return
	}

	flags := o.namespacesToFlags[namespace]

	var without FlagList

	for _, f := range flags {
		if f.Name == flagName {
			o.saveDeleted(namespace, f)
		} else {
			without = append(without, f)
		}
	}

	if len(without) == 0 {
		delete(o.namespacesToFlags, namespace)
	} else {
		o.namespacesToFlags[namespace] = without
	}
}

func (o *Ctl) saveDeleted(namespace string, flag Flag) {
	if o.deleted == nil {
		o.deleted = make(map[string]FlagList)
	}

	deletedFlags := o.deleted[namespace]

	for i, deleted := range deletedFlags {
		if deleted.Name == flag.Name {
			deletedFlags[i] = flag

			break
		}
	}

	o.deleted[namespace] = deletedFlags
}

func (o *Ctl) DeletedFlags(namespace string) FlagList {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	flags, _ := o.deleted[namespace]

	return flags
}

func (o *Ctl) DeleteAutoAddedFlags() {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	for namespace, fl := range o.namespacesToFlags {
		var without FlagList

		for _, f := range fl {
			if !f.autoAdded {
				without = append(without, f)
			}
		}

		if len(without) == 0 {
			delete(o.namespacesToFlags, namespace)
		} else {
			o.namespacesToFlags[namespace] = without
		}
	}

	for namespace, fl := range o.deleted {
		var without FlagList

		for _, f := range fl {
			if !f.autoAdded {
				without = append(without, f)
			}
		}

		if len(without) == 0 {
			delete(o.deleted, namespace)
		} else {
			o.deleted[namespace] = without
		}
	}
}
