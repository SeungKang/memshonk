package sessiond

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/SeungKang/memshonk/internal/apicompat"

	"github.com/SeungKang/memshonk/internal/vendored/goterm"
)

// various signal message types
const (
	unknownSignalType uint8 = iota
	IntSignalType
)

type Session struct {
	shared    apicompat.SharedState
	info      apicompat.SessionInfo
	isDefault bool
	io        apicompat.SessionIO
	cmdStore  *apicompat.CommandStorage
	shell     Shell
	ctx       context.Context
	ocne      sync.Once
	cancelFn  func()
	apiConn   io.Closer

	cancelCmdMu  sync.Mutex
	cancelCmdCtx context.CancelFunc
}

func (o *Session) Ctx() context.Context {
	return o.ctx
}

func (o *Session) Done() <-chan struct{} {
	return o.ctx.Done()
}

func (o *Session) Close() error {
	var err error

	o.ocne.Do(func() {
		o.cancelFn()

		if o.shell != nil {
			_ = o.shell.Close()
		}

		err = o.apiConn.Close()
	})

	return err
}

func (o *Session) SharedState() apicompat.SharedState {
	return o.shared
}

func (o *Session) Info() apicompat.SessionInfo {
	return o.info
}

func (o *Session) IO() apicompat.SessionIO {
	return o.io
}

func (o *Session) OnSignal(signalType uint8) {
	switch signalType {
	case IntSignalType:
		o.cancelCurrentCommand()

		if o.shell != nil {
			o.shell.Signal(nil)
		}
	default:
		// ignore or log unknown signals
	}
}

func (o *Session) Terminal() (*goterm.VirtualTerminal, bool) {
	if o.io.OptTerminal != nil {
		return o.io.OptTerminal, true
	}

	return nil, false
}

func (o *Session) CommandStorage() *apicompat.CommandStorage {
	return o.cmdStore
}

func (o *Session) RunCommandNext(parent context.Context, config apicompat.RunCommandConfig) (bool, error) {
	if len(config.Argv) == 0 {
		return false, nil
	}

	newCmdFn, hasIt := o.shared.Commands.Lookup(config.Argv[0])
	if !hasIt {
		return false, nil
	}

	ctx, cancelFn := context.WithCancel(parent)
	defer func() {
		o.clearCancel()
		cancelFn()
	}()

	go func() {
		select {
		case <-ctx.Done():
		case <-o.ctx.Done():
			cancelFn()
		}
	}()

	// install the cancel lever for OnSignal
	o.setCancel(func() {
		cancelFn()
	})

	cmd := newCmdFn(o)

	result, err := cmd.Run(ctx, config.Argv[1:])
	if err != nil {
		return true, fmt.Errorf("%s failed: %w", cmd.Name(), err)
	}

	o.cmdStore.AddOutput(result)

	if result.Result != nil {
		config.Stdout.Write([]byte(result.Result.Human()))
		config.Stdout.Write([]byte{'\n'})
	}

	return true, nil
}

func (o *Session) RunCommand(parent context.Context, cmd apicompat.Command) error {
	ctx, cancelFn := context.WithCancel(parent)
	defer func() {
		o.clearCancel()
		cancelFn()
	}()

	go func() {
		select {
		case <-ctx.Done():
		case <-o.ctx.Done():
			cancelFn()
		}
	}()

	// install the cancel lever for OnSignal
	o.setCancel(func() {
		cancelFn()
	})

	result, err := cmd.Run(ctx, o)
	if err != nil {
		return err
	}

	if result != nil {
		o.io.Stdout.Write(result.Serialize())
		o.io.Stdout.Write([]byte{'\n'})
	}

	return nil
}

func (o *Session) setCancel(fn context.CancelFunc) {
	o.cancelCmdMu.Lock()
	defer o.cancelCmdMu.Unlock()
	o.cancelCmdCtx = fn
}

func (o *Session) clearCancel() {
	o.cancelCmdMu.Lock()
	defer o.cancelCmdMu.Unlock()
	o.cancelCmdCtx = nil
}

func (o *Session) cancelCurrentCommand() {
	o.cancelCmdMu.Lock()
	fn := o.cancelCmdCtx
	o.cancelCmdMu.Unlock()

	if fn != nil {
		fn()
	}
}
