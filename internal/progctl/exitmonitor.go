package progctl

import "sync"

func newExitMonitor() *ExitMonitor {
	return &ExitMonitor{
		c: make(chan struct{}),
	}
}

type ExitMonitor struct {
	c    chan struct{}
	once sync.Once
	err  error
}

func (o *ExitMonitor) Done() <-chan struct{} {
	return o.c
}

func (o *ExitMonitor) Err() error {
	return o.err
}

func (o *ExitMonitor) SetExited(err error) {
	o.once.Do(func() {
		if err == nil {
			err = ErrExitedNormally
		}

		o.err = err

		close(o.c)
	})
}
