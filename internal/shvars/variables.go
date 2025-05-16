package shvars

import "sync"

type Variables struct {
	rwMu sync.RWMutex
	vars map[string]string
}

func (o *Variables) Len() int {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	return len(o.vars)
}

func (o *Variables) Set(name string, value string) error {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if o.vars == nil {
		o.vars = make(map[string]string)
	}

	o.vars[name] = value

	return nil
}

func (o *Variables) Get(name string) (string, bool) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	value, hasIt := o.vars[name]

	return value, hasIt
}
