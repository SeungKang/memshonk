package progctl

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/SeungKang/memshonk/internal/events"
	"github.com/SeungKang/memshonk/internal/memory"
	"github.com/mitchellh/go-ps"
)

// various memory mode strings
const (
	kernel32MemoryMode = "kernel32"
	ptraceMemoryMode   = "ptrace"
	procfsMemoryMode   = "procfs"
)

var _ Process = (*Ctl)(nil)

var (
	ErrNotAttached    = errors.New("not attached")
	ErrDetached       = errors.New("detached")
	ErrExitedNormally = errors.New("process exited without error")
)

type AttachConfig struct {
	OptPID int
}

type Process interface {
	SetMemoryMode(string) error

	MemoryMode() string

	Attach(ctx context.Context, cfg AttachConfig) (int, error)

	ProcessInfo(ctx context.Context) (ProcessInfo, error)

	ExeInfo(ctx context.Context) (ExeInfo, error)

	Regions(ctx context.Context) (memory.Regions, error)

	ResolvePointer(ctx context.Context, ptr memory.Pointer) (uintptr, error)

	ReadFromAddr(ctx context.Context, addr memory.Pointer, sizeBytes uint64) ([]byte, uintptr, error)

	WriteToAddr(ctx context.Context, addr memory.Pointer, p []byte) (uintptr, error)

	WatchAddr(ctx context.Context, addr memory.Pointer, sizeBytes uint64) (*Watcher, error)

	Suspend(ctx context.Context) error

	Resume(ctx context.Context) error

	Detach(ctx context.Context) error

	ReadFromLookup(ctx context.Context, addr string, sizeBytes uint64) ([]byte, uintptr, error)

	WriteToLookup(ctx context.Context, addr string, p []byte) (uintptr, error)

	WatchLookup(ctx context.Context, addr string, sizeBytes uint64) (*Watcher, error)
}

type attachConfig struct {
	exeName        string
	pid            int
	exitMon        *ExitMonitor
	memoryModeName string
}

func unsupportedMemoryModeError(memoryMode string) error {
	return fmt.Errorf("unsupported memory mode: %q", memoryMode)
}

type attachedProcess interface {
	SetMemoryMode(string) error

	ExitMonitor() *ExitMonitor

	PID() int

	ExeInfo() ExeInfo

	ReadBytes(addr uintptr, sizeBytes uint64) ([]byte, error)

	WriteBytes(b []byte, addr uintptr) error

	ReadPtr(at uintptr) (uintptr, error)

	Regions() (memory.Regions, error)

	Suspend() error

	Resume() error

	Close() error
}

func NewCtl(exePath string, eventGroups *events.Groups) *Ctl {
	return &Ctl{
		exePath:       exePath,
		attachEvents:  events.NewPublisher[AttachedEvent](eventGroups),
		detachEvents:  events.NewPublisher[DetachedEvent](eventGroups),
		processExited: events.NewPublisher[ProcessExitedEvent](eventGroups),
	}
}

type Ctl struct {
	exePath       string
	attachEvents  *events.Publisher[AttachedEvent]
	detachEvents  *events.Publisher[DetachedEvent]
	processExited *events.Publisher[ProcessExitedEvent]
	rwMu          sync.RWMutex
	memMode       string
	current       *processThread
}

func (o *Ctl) SetMemoryMode(modeName string) error {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	return o.setMemoryMode(modeName)
}

func (o *Ctl) setMemoryMode(modeName string) error {
	switch runtime.GOOS {
	case "windows":
		switch modeName {
		case "", kernel32MemoryMode:
			o.memMode = kernel32MemoryMode
		default:
			return fmt.Errorf("unsupported memory mode: %q - supported options are: %q",
				modeName, kernel32MemoryMode)
		}
	default:
		switch modeName {
		case "", ptraceMemoryMode:
			o.memMode = ptraceMemoryMode
		case procfsMemoryMode:
			o.memMode = procfsMemoryMode
		default:
			return fmt.Errorf("unsupported memory mode: %q - supported options are: %q or %q",
				modeName, procfsMemoryMode, procfsMemoryMode)
		}
	}

	if o.current != nil {
		return o.current.Do(context.Background(), func(process attachedProcess) error {
			return process.SetMemoryMode(o.memMode)
		})
	}

	return nil
}

func (o *Ctl) MemoryMode() string {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	return o.memMode
}

func (o *Ctl) Attach(ctx context.Context, cfg AttachConfig) (int, error) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if o.current != nil {
		select {
		case <-o.current.ExitMonitor().Done():
			// Go ahead with reattach.
		default:
			return 0, fmt.Errorf("already attached to pid: %d", o.current.PID())
		}
	}

	targetExeName := filepath.Base(o.exePath)
	targetExeNameLower := strings.ToLower(targetExeName)

	processes, err := ps.Processes()
	if err != nil {
		return 0, fmt.Errorf("failed to get active processes - %w", err)
	}

	foundPID := -1
	var foundExeName string

	if cfg.OptPID > 0 {
		for _, psProc := range processes {
			if psProc.Pid() == cfg.OptPID {
				foundPID = psProc.Pid()
				foundExeName = psProc.Executable()
				break
			}
		}

		if foundExeName == "" {
			return 0, fmt.Errorf("failed to find a process with pid: %d", cfg.OptPID)
		}
	} else {
		for _, psProc := range processes {
			if strings.ToLower(psProc.Executable()) == targetExeNameLower {
				foundPID = psProc.Pid()
				foundExeName = psProc.Executable()
				break
			}
		}

		if foundExeName == "" {
			return 0, fmt.Errorf("failed to find a matching process for: %q",
				targetExeName)
		}
	}

	// set memory mode to default if none specified
	if o.memMode == "" {
		err := o.setMemoryMode("")
		if err != nil {
			return 0, fmt.Errorf("failed to set initial memory mode, this should never happen: %w", err)
		}
	}

	unexpectedExitMon := newExitMonitor(o.processExited)

	proc, err := newProcessThread(attachConfig{
		exeName:        foundExeName,
		pid:            foundPID,
		exitMon:        unexpectedExitMon,
		memoryModeName: o.memMode,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to attach to process %d (%q) - %w",
			foundPID, foundExeName, err)
	}

	o.current = proc

	info, _ := o.processInfo(ctx)

	_ = o.attachEvents.SendAndWait(ctx, AttachedEvent{
		ProcessInfo: info,
		acker:       events.NewAcker(),
	})

	return foundPID, nil
}

func (o *Ctl) ProcessInfo(ctx context.Context) (ProcessInfo, error) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	return o.processInfo(ctx)
}

func (o *Ctl) processInfo(context.Context) (ProcessInfo, error) {
	if o.current == nil {
		return ProcessInfo{}, ErrNotAttached
	}

	return ProcessInfo{
		PID: o.current.PID(),
	}, nil
}

type ProcessInfo struct {
	PID int
}

func (o *Ctl) ExeInfo(context.Context) (ExeInfo, error) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	if o.current == nil {
		return ExeInfo{}, ErrNotAttached
	}

	return o.current.process.ExeInfo(), nil
}

type ExeInfo struct {
	Bits uint8
	Obj  memory.Object
}

func (o *Ctl) Regions(ctx context.Context) (memory.Regions, error) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	if o.current == nil {
		return memory.Regions{}, ErrNotAttached
	}

	return o.regions(ctx)
}

func (o *Ctl) regions(ctx context.Context) (memory.Regions, error) {
	var regions memory.Regions
	err := o.current.Do(ctx, func(process attachedProcess) error {
		var err error
		regions, err = process.Regions()
		return err
	})

	return regions, err
}

func (o *Ctl) ResolvePointer(ctx context.Context, ptr memory.Pointer) (uintptr, error) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	if o.current == nil {
		return 0, ErrNotAttached
	}

	return o.resolvePointer(ctx, ptr)
}

func (o *Ctl) resolvePointer(ctx context.Context, ptr memory.Pointer) (uintptr, error) {
	if ptr.Type() == memory.AbsoluteAddrPointerType {
		return ptr.FirstAddr(), nil
	}

	baseAddr := o.current.ExeObj().Obj.BaseAddr

	if ptr.OptModule != "" {
		regions, err := o.regions(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to get memeory regions - %w",
				err)
		}

		// TODO: This is a non-exact match. Will that
		// lead to unexpected things happening? Or do
		// we want that level of convenience?
		object, err := regions.FirstObjectMatching(ptr.OptModule)
		if err != nil {
			return 0, fmt.Errorf("failed to resolve object name - %w",
				err)
		}

		baseAddr = object.BaseAddr
	}

	var addr uintptr
	err := o.current.Do(ctx, func(process attachedProcess) error {
		var err error
		addr, err = resolvePointerChain(baseAddr, ptr, process.ReadPtr)
		return err
	})
	if err != nil {
		return 0, fmt.Errorf("failed to resolve pointer chain - %w",
			err)
	}

	return addr, nil
}

func (o *Ctl) ReadFromAddr(ctx context.Context, from memory.Pointer, sizeBytes uint64) ([]byte, uintptr, error) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	return o.readFromAddr(ctx, from, sizeBytes)
}

func (o *Ctl) readFromAddr(ctx context.Context, from memory.Pointer, sizeBytes uint64) ([]byte, uintptr, error) {
	if o.current == nil {
		return nil, 0, ErrNotAttached
	}

	addr, err := o.resolvePointer(ctx, from)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to resolve pointer - %w", err)
	}

	var buf []byte
	err = o.current.Do(ctx, func(process attachedProcess) error {
		var err error
		buf, err = process.ReadBytes(addr, sizeBytes)
		return err
	})

	return buf, addr, err
}

func (o *Ctl) ReadFromLookup(ctx context.Context, addr string, sizeBytes uint64) ([]byte, uintptr, error) {
	ptr, err := memory.CreatePointerFromString(addr)
	if err != nil {
		return nil, 0, err
	}

	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	return o.readFromAddr(ctx, ptr, sizeBytes)
}

func (o *Ctl) WriteToAddr(ctx context.Context, to memory.Pointer, data []byte) (uintptr, error) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	return o.writeToAddr(ctx, to, data)
}

func (o *Ctl) writeToAddr(ctx context.Context, to memory.Pointer, data []byte) (uintptr, error) {
	if o.current == nil {
		return 0, ErrNotAttached
	}

	addr, err := o.resolvePointer(ctx, to)
	if err != nil {
		return 0, fmt.Errorf("failed to resolve pointer - %w", err)
	}

	return addr, o.current.Do(ctx, func(process attachedProcess) error {
		return process.WriteBytes(data, addr)
	})
}

func (o *Ctl) WriteToLookup(ctx context.Context, addr string, p []byte) (uintptr, error) {
	ptr, err := memory.CreatePointerFromString(addr)
	if err != nil {
		return 0, err
	}

	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	return o.writeToAddr(ctx, ptr, p)
}

func (o *Ctl) WatchAddr(ctx context.Context, ptr memory.Pointer, sizeBytes uint64) (*Watcher, error) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	return o.watchAddr(ctx, ptr, sizeBytes)
}

func (o *Ctl) watchAddr(ctx context.Context, ptr memory.Pointer, sizeBytes uint64) (*Watcher, error) {
	if o.current == nil {
		return nil, ErrNotAttached
	}

	addr, err := o.resolvePointer(ctx, ptr)
	if err != nil {
		return nil, err
	}

	watcher := newWatcher(ctx, addr, sizeBytes)

	err = o.current.AddWatcher(ctx, watcher)
	if err != nil {
		return nil, err
	}

	return watcher, nil
}

func (o *Ctl) WatchLookup(ctx context.Context, addr string, sizeBytes uint64) (*Watcher, error) {
	ptr, err := memory.CreatePointerFromString(addr)
	if err != nil {
		return nil, err
	}

	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	return o.watchAddr(ctx, ptr, sizeBytes)
}

func (o *Ctl) Suspend(ctx context.Context) error {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if o.current == nil {
		return nil
	}

	return o.current.Do(ctx, func(process attachedProcess) error {
		return process.Suspend()
	})
}

func (o *Ctl) Resume(ctx context.Context) error {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if o.current == nil {
		return nil
	}

	return o.current.Do(ctx, func(process attachedProcess) error {
		return process.Resume()
	})
}

func (o *Ctl) Detach(ctx context.Context) error {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if o.current == nil {
		return nil
	}

	err := o.current.Close(ctx)

	o.current = nil

	_ = o.detachEvents.SendAndWait(ctx, DetachedEvent{
		acker: events.NewAcker(),
	})

	return err
}

// resolvePointerChain resolves a chain-type memory.Pointer.
//
// Note: This code assumes the Pointer is a chain and that the upstream
// code has ensured that is the case.
func resolvePointerChain(baseAddr uintptr, ptr memory.Pointer, addrFn func(uintptr) (uintptr, error)) (uintptr, error) {
	addr := baseAddr
	var offsets = ptr.Addrs()
	var err error

	// We are purposely skipping the last offset.
	for i, offset := range offsets[:len(offsets)-1] {
		currentTarget := addr + offset

		addr, err = addrFn(currentTarget)
		if err != nil {
			return 0, fmt.Errorf("failed to read offset index %d from process (offset: %#x | addr: %#x) - %w",
				i, offset, currentTarget, err)
		}
	}

	// sfox: I believe this is based on what Cheat Engine does.
	addr += offsets[len(offsets)-1]

	return addr, nil
}
