package sessiond

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/cstlv"
	"github.com/SeungKang/memshonk/internal/jobsctl"

	"github.com/SeungKang/memshonk/internal/vendored/goterm"
)

// various signal message types
const (
	unknownSignalType uint8 = iota
	IntSignalType
)

type Session struct {
	shared   apicompat.SharedState
	info     apicompat.SessionInfo
	jobs     *jobsctl.Ctl
	io       apicompat.SessionIO
	cmdStore *apicompat.CommandStorage
	shell    Shell
	ctx      context.Context
	once     sync.Once
	cancelFn func()
	apiConn  net.Conn

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

	o.once.Do(func() {
		o.cancelFn()

		shutdownCtx, cancelFn := context.WithTimeout(
			context.Background(), 2*time.Second)
		defer cancelFn()

		o.jobs.Shutdown(shutdownCtx)

		if o.shell != nil {
			_ = o.shell.Close()
		}

		o.apiConn.SetDeadline(time.Now().Add(time.Second))

		o.apiConn.Write(
			cstlv.MinimalBytes(0, 0, sessionExitedClientMessage))

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

func (o *Session) Jobs() *jobsctl.Ctl {
	return o.jobs
}

func (o *Session) OnSignal(signalType uint8) {
	switch signalType {
	case IntSignalType:
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
