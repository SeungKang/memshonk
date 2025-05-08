package progctl

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/SeungKang/memshonk/internal/appconfig"
	"github.com/SeungKang/memshonk/internal/memory"
	"github.com/mitchellh/go-ps"
)

var (
	programExitedNormallyErr = errors.New("program exited without error")
)

type Notifier interface {
	ProgramStarted(exename string)
	ProgramStopped(exename string, err error)
}

func NewRoutine(ctx context.Context, config *appconfig.ProgramConfig) *Routine {
	routine := &Routine{
		Program: config,
		Notif:   nil,
		done:    make(chan struct{}),
	}

	go routine.loop(ctx)

	return routine
}

type Routine struct {
	Program  *appconfig.ProgramConfig
	Notif    Notifier
	doAttach chan *attachCallback
	current  *process
	done     chan struct{}
	err      error
}

func (o *Routine) Done() <-chan struct{} {
	return o.done
}

func (o *Routine) Err() error {
	return o.err
}

func (o *Routine) loop(ctx context.Context) {
	var cancelFn func()
	ctx, cancelFn = context.WithCancel(ctx)
	defer cancelFn()

	o.err = o.loopWithError(ctx)
	close(o.done)
}

func (o *Routine) Attach(ctx context.Context) (int, error) {
	cb := &attachCallback{done: make(chan struct{})}

	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-o.done:
		return 0, o.err
	case o.doAttach <- cb:
		// keep going
	}

	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-o.done:
		return 0, o.err
	case <-cb.done:
		return cb.pid, cb.err
	}
}

type attachCallback struct {
	done chan struct{}
	err  error
	pid  int
}

func (o *Routine) ReadFromAddr(ctx context.Context, addr memory.Pointer, size uint) ([]byte, error) {
	// TODO
	o.current.read(addr, size)
}

func (o *Routine) WriteToAddr(ctx context.Context, p []byte, addr memory.Pointer) error {
	// TODO
}

func (o *Routine) Detach(ctx context.Context) error {
	// TODO
}

func (o *Routine) loopWithError(ctx context.Context) error {
	defer func() {
		if o.current != nil {
			o.current.Stop()
		}
	}()

	log.Printf("checking for program running with exe name: %s", o.Program.General.ExeName)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case cb := <-o.doAttach:
			if o.current != nil {
				cb.pid = o.current.pid
				close(cb.done)
				continue
			}

			err := o.checkProgramRunning()
			if err != nil {
				cb.err = fmt.Errorf("failed to handle program startup - %w", err)
				close(cb.done)
				continue
			}

			cb.pid = o.current.pid
			close(cb.done)
		case <-o.current.Done():
			log.Printf("%s routine exited - %s", o.Program.General.ExeName, o.current.Err())

			if o.Notif != nil {
				if errors.Is(o.current.Err(), programExitedNormallyErr) {
					o.Notif.ProgramStopped(o.Program.General.ExeName, nil)
				} else {
					o.Notif.ProgramStopped(o.Program.General.ExeName, o.current.Err())
				}
			}

			o.current = nil
		}
	}
}

func (o *Routine) checkProgramRunning() error {
	// TODO: logger to make prefix with exename
	processes, err := ps.Processes()
	if err != nil {
		return fmt.Errorf("failed to get active processes - %w", err)
	}

	possiblePID := -1
	for _, process := range processes {
		if strings.ToLower(process.Executable()) == o.Program.General.ExeName {
			possiblePID = process.Pid()
			break
		}
	}

	if possiblePID == -1 {
		return errors.New("failed to find a matching process")
	}

	runningProgram, err := newProcess(o.Program, possiblePID)
	if err != nil {
		return fmt.Errorf("failed to create new running program routine - %w", err)
	}

	o.current = runningProgram
	if o.Notif != nil {
		o.Notif.ProgramStarted(o.Program.General.ExeName)
	}

	return nil
}
