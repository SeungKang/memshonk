package progctl

import (
	"context"
	"fmt"
)

func attachShared(config attachConfig) (*attachedProcess, error) {
	proc, err := attach(config)
	if err != nil {
		return nil, fmt.Errorf("attach failure - %w", err)
	}

	attached := &attachedProcess{
		config:   config,
		process:  proc,
		watchers: newWatcherCtl(proc),
	}

	return attached, nil
}

type attachedProcess struct {
	config   attachConfig
	watchers *watcherCtl
	process  *process
}

func (o *attachedProcess) Close(ctx context.Context) error {
	_ = o.watchers.Close()

	err := o.process.Close()

	return err
}
