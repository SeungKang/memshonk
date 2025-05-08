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

type Notifier interface {
	ProgramStarted(exename string)
	ProgramStopped(exename string, err error)
}

func NewCtl(ctx context.Context, exeName string) *Ctl {
	return &Ctl{
		Notif:   nil,
		exeName: exeName,
		done:    make(chan struct{}),
	}
}

type Ctl struct {
	Notif   Notifier
	exeName string
	rwMu    sync.RWMutex
	current *process
	done    chan struct{}
	err     error
}

func (o *Ctl) Done() <-chan struct{} {
	return o.done
}

func (o *Ctl) Err() error {
	return o.err
}

func (o *Ctl) Attach(ctx context.Context) (int, error) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if o.current != nil {
		return 0, fmt.Errorf("already attached to pid: %d", o.current.pid)
	}

	err := o.checkProgramRunning()
	if err != nil {
		return 0, err
	}

	return o.current.pid, nil

}

func (o *Ctl) ReadFromAddr(ctx context.Context, from memory.Pointer, size uint) ([]byte, error) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	if o.current == nil {
		return nil, errors.New("not attached")
	}

	return o.current.read(from, size)
}

func (o *Ctl) WriteToAddr(ctx context.Context, data []byte, to memory.Pointer) error {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	if o.current == nil {
		return errors.New("not attached")
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

func (o *Ctl) checkProgramRunning() error {
	processes, err := ps.Processes()
	if err != nil {
		return fmt.Errorf("failed to get active processes - %w", err)
	}

	possiblePID := -1
	for _, process := range processes {
		if strings.ToLower(process.Executable()) == strings.ToLower(o.exeName) {
			possiblePID = process.Pid()
			break
		}
	}

	if possiblePID == -1 {
		return errors.New("failed to find a matching process")
	}

	proc, err := newProcess(o.exeName, possiblePID)
	if err != nil {
		return fmt.Errorf("failed to create new running program routine - %w", err)
	}

	o.current = proc
	if o.Notif != nil {
		o.Notif.ProgramStarted(o.exeName)
	}

	return nil
}
