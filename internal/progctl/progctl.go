package progctl

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/SeungKang/memshonk/internal/memory"
	"github.com/mitchellh/go-ps"
)

var _ Process = (*Ctl)(nil)

var (
	ErrNotAttached = errors.New("not attached")
)

type Notifier interface {
	ProgramStarted(exename string)
	ProgramStopped(exename string, err error)
}

type Process interface {
	Attach(ctx context.Context) (int, error)

	ExeObject(ctx context.Context) (memory.MappedObject, error)

	MappedObjects(ctx context.Context) (memory.MappedObjects, error)

	ResolvePointer(ctx context.Context, ptr memory.Pointer) (uintptr, error)

	ReadFromAddr(ctx context.Context, addr memory.Pointer, sizeBytes uint64) ([]byte, error)

	WriteToAddr(ctx context.Context, p []byte, addr memory.Pointer) error

	Detach(ctx context.Context) error
}

func NewCtl(exeName string) *Ctl {
	return &Ctl{
		Notif:   nil,
		exeName: exeName,
	}
}

type Ctl struct {
	Notif   Notifier
	exeName string
	rwMu    sync.RWMutex
	current *process
}

func (o *Ctl) Attach(ctx context.Context) (int, error) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if o.current != nil {
		select {
		case <-o.current.Done():
			// Go ahead with reattach.
		default:
			return 0, fmt.Errorf("already attached to pid: %d", o.current.pid)
		}
	}

	err := o.attach()
	if err != nil {
		return 0, err
	}

	return o.current.pid, nil

}

func (o *Ctl) ExeObject(ctx context.Context) (memory.MappedObject, error) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	if o.current == nil {
		return memory.MappedObject{}, ErrNotAttached
	}

	return o.current.exeMod, nil
}

func (o *Ctl) MappedObjects(context.Context) (memory.MappedObjects, error) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	if o.current == nil {
		return memory.MappedObjects{}, ErrNotAttached
	}

	return o.current.objects(), nil
}

func (o *Ctl) ResolvePointer(_ context.Context, ptr memory.Pointer) (uintptr, error) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	if o.current == nil {
		return 0, ErrNotAttached
	}

	addr, err := o.current.resolvePointer(ptr)
	if err != nil {
		return 0, fmt.Errorf("failed to resolve address - %w", err)
	}

	return addr, nil
}

func (o *Ctl) ReadFromAddr(ctx context.Context, from memory.Pointer, sizeBytes uint64) ([]byte, error) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	if o.current == nil {
		return nil, ErrNotAttached
	}

	return o.current.read(from, sizeBytes)
}

func (o *Ctl) WriteToAddr(ctx context.Context, data []byte, to memory.Pointer) error {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	if o.current == nil {
		return ErrNotAttached
	}

	return o.current.write(data, to)
}

func (o *Ctl) Detach(ctx context.Context) error {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if o.current == nil {
		return nil
	}

	o.current.Stop()

	o.current = nil

	return nil
}

func (o *Ctl) attach() error {
	processes, err := ps.Processes()
	if err != nil {
		return fmt.Errorf("failed to get active processes - %w", err)
	}

	possiblePID := -1
	var exeName string
	for _, psProc := range processes {
		if strings.ToLower(psProc.Executable()) == strings.ToLower(o.exeName) {
			possiblePID = psProc.Pid()
			exeName = psProc.Executable()
			break
		}
	}

	if possiblePID == -1 {
		return errors.New("failed to find a matching process")
	}

	proc, err := newProcess(exeName, possiblePID)
	if err != nil {
		return fmt.Errorf("failed to create new running program routine - %w", err)
	}

	o.current = proc
	if o.Notif != nil {
		o.Notif.ProgramStarted(o.exeName)
	}

	return nil
}
