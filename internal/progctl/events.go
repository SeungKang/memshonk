package progctl

type AttachedEvent struct {
	Pid   int
	Acked chan struct{}
}

type DetachedEvent struct {
	Acked chan struct{}
}

type ProcessExitedEvent struct {
	Reason error
}
