package jobsctl

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"
)

func New() *Ctl {
	return &Ctl{
		jobs: make(map[*Job]struct{}),
	}
}

type Ctl struct {
	rwMu sync.RWMutex
	jobs map[*Job]struct{}
	jids [100000]bool

	stopped bool
}

func (o *Ctl) List() []*Job {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	jobs := make([]*Job, 0, len(o.jobs))

	for j := range o.jobs {
		jobs = append(jobs, j)
	}

	return jobs
}

func (o *Ctl) Lookup(id string) (*Job, error) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	for j := range o.jobs {
		if j.ID() == id {
			return j, nil
		}

	}

	return nil, fmt.Errorf("no such job")
}

type RegisterConfig struct {
	Namespace string

	Argv []string
}

func (o *Ctl) Register(ctx context.Context, config RegisterConfig) (context.Context, *Job, error) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if o.stopped {
		return nil, nil, fmt.Errorf("not accepting new jobs because job ctl was shutdown")
	}

	var jid uint64

	for i, inUse := range o.jids[1:] {
		if !inUse {
			jid = uint64(i) + 1

			o.jids[i] = true

			break
		}
	}

	if jid == 0 {
		return nil, nil, fmt.Errorf("out of jids - try stopping some jobs :(")
	}

	jobCtx, cancelFn := context.WithCancel(ctx)

	job := &Job{
		parent:    o,
		config:    config,
		jid:       jid,
		startedAt: time.Now(),
		cancelFn:  cancelFn,
		exited:    make(chan struct{}),
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

	o.jids[job.jid] = false

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
	parent *Ctl
	config RegisterConfig

	startedAt time.Time

	idRwMu sync.RWMutex
	jid    uint64
	pid    int

	cancelFn func()

	setExitedOnce sync.Once
	exited        chan struct{}
}

func (o *Job) Info() JobInfo {
	info := JobInfo{
		RegisterConfig: o.config,
		ID:             o.ID(),
		StartedAt:      o.startedAt,
	}

	o.idRwMu.RLock()

	info.HasPID = o.pid > 0
	info.PID = o.pid

	o.idRwMu.RUnlock()

	return info
}

type JobInfo struct {
	RegisterConfig RegisterConfig
	ID             string
	HasPID         bool
	PID            int
	StartedAt      time.Time
}

func (o *Job) ID() string {
	return strconv.FormatUint(o.jid, 10)
}

func (o *Job) SetPID(pid int) {
	o.idRwMu.Lock()
	defer o.idRwMu.Unlock()

	o.pid = pid
}

func (o *Job) Cancel() {
	o.cancelFn()
}

func (o *Job) CancelSync(ctx context.Context) {
	o.cancelFn()

	select {
	case <-ctx.Done():
	case <-o.exited:
	}
}

func (o *Job) SetFinished() {
	o.setExitedOnce.Do(func() {
		close(o.exited)

		o.parent.deregister(o)
	})
}
