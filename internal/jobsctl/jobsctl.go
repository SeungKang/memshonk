package jobsctl

import (
	"context"
	"fmt"
	"sync"
)

func New() *Ctl {
	return &Ctl{}
}

type Ctl struct {
	rwMu    sync.RWMutex
	jobs    map[*Job]struct{}
	stopped bool
}

type RegisterConfig struct{}

func (o *Ctl) Register(ctx context.Context, config RegisterConfig) (context.Context, *Job, error) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if o.stopped {
		return nil, nil, fmt.Errorf("not accepting new jobs because job ctl was shutdown")
	}

	if o.jobs == nil {
		o.jobs = make(map[*Job]struct{})
	}

	jobCtx, cancelFn := context.WithCancel(ctx)

	job := &Job{
		parent:   o,
		config:   config,
		cancelFn: cancelFn,
		exited:   make(chan struct{}),
	}

	o.jobs[job] = struct{}{}

	return jobCtx, job, nil
}

func (o *Ctl) deregister(job *Job) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if o.stopped {
		return
	}

	delete(o.jobs, job)
}

func (o *Ctl) Shutdown(ctx context.Context) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if o.stopped {
		return
	}

	o.stopped = true

	for job := range o.jobs {
		job.cancelFn()

		select {
		case <-ctx.Done():
		case <-job.exited:
		}
	}
}

type Job struct {
	parent   *Ctl
	config   RegisterConfig
	once     sync.Once
	cancelFn func()
	exited   chan struct{}
}

func (o *Job) Finished() {
	o.once.Do(func() {
		close(o.exited)

		o.parent.deregister(o)
	})
}
