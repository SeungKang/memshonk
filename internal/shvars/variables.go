package shvars

import (
	"fmt"
	"sort"
	"sync"
)

// Various variable sources.
const (
	ProcEnvVarsSrc = "process environment"
	ProjectVarsSrc = "project"
	ClientVarsSrc  = "client"
)

type Variables struct {
	rwMu sync.RWMutex
	vars map[string]Variable
}

func (o *Variables) Len() int {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	return len(o.vars)
}

type Variable struct {
	Name      string
	Value     string
	Source    string
	Immutable bool
}

func (o *Variables) Set(v Variable) error {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if o.vars == nil {
		o.vars = make(map[string]Variable)
	} else {
		target, hasIt := o.vars[v.Name]
		if hasIt && v.Immutable {
			return newImmuntableErr(target)
		}
	}

	o.vars[v.Name] = v

	return nil
}

func (o *Variables) Get(name string) (string, bool) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	v, hasIt := o.vars[name]

	return v.Value, hasIt
}

func (o *Variables) Delete(name string) error {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	target, hasIt := o.vars[name]
	if hasIt {
		if target.Immutable {
			return newImmuntableErr(target)
		}

		delete(o.vars, name)
	}

	return nil
}

func newImmuntableErr(v Variable) error {
	return fmt.Errorf("variable is immutable (variable sourced from: %q)",
		v.Source)
}

func (o *Variables) KeyValues(optFilterFn interface{}) []string {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	strs := make([]string, 0, len(o.vars))

	for k, v := range o.vars {
		strs = append(strs, k+"="+v.Value)
	}

	sort.Strings(strs)

	return strs
}
