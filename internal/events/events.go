package events

import (
	"context"
	"reflect"
	"sync"
	"time"
)

func NewGroups() *Groups {
	return &Groups{}
}

type Groups struct {
	rwMu   sync.RWMutex
	groups map[interface{}]interface{}
}

func getOrAddPubSubGroup[C comparable](groups *Groups) *group[C] {
	groups.rwMu.Lock()
	defer groups.rwMu.Unlock()

	if groups.groups == nil {
		groups.groups = make(map[interface{}]interface{})
	}

	var category C

	target, hasIt := groups.groups[category]
	if !hasIt {
		group := newGroup[C]()

		target = group

		groups.groups[category] = group
	}

	return target.(*group[C])
}

func newGroup[C comparable]() *group[C] {
	return &group[C]{}
}

type group[C comparable] struct {
	rwMu sync.RWMutex
	subs map[*Sub[C]]struct{}
}

func (o *group[C]) NewSub() *Sub[C] {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if o.subs == nil {
		o.subs = make(map[*Sub[C]]struct{})
	}

	sub := &Sub[C]{
		parent: o,
		ch:     make(chan C, 10),
		done:   make(chan struct{}),
	}

	o.subs[sub] = struct{}{}

	return sub
}

func (o *group[C]) Send(ctx context.Context, event C) error {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	for sub := range o.subs {
		err := sub.send(ctx, event)
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *group[C]) Unsub(sub *Sub[C]) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	delete(o.subs, sub)

	if len(o.subs) == 0 {
		o.subs = nil
	}
}

func NewPublisher[C comparable](pubSub *Groups) *Publisher[C] {
	return &Publisher[C]{
		parent: getOrAddPubSubGroup[C](pubSub),
	}
}

type Publisher[C comparable] struct {
	parent *group[C]
}

func (o *Publisher[C]) Send(ctx context.Context, event C) error {
	return o.parent.Send(ctx, event)
}

func NewSubscriber[C comparable](pubSub *Groups) *Sub[C] {
	group := getOrAddPubSubGroup[C](pubSub)

	return group.NewSub()
}

type Sub[C comparable] struct {
	parent *group[C]
	ch     chan C
	once   sync.Once
	done   chan struct{}
}

func (o *Sub[C]) Recv(ctx context.Context) (C, error) {
	select {
	case <-ctx.Done():
		var empty C
		return empty, ctx.Err()
	case event := <-o.ch:
		return event, nil
	}
}

func (o *Sub[C]) RecvCh() <-chan C {
	return o.ch
}

func (o *Sub[C]) Unsubscribe() {
	o.once.Do(func() {
		close(o.done)

		o.parent.Unsub(o)
	})
}

func (o *Sub[C]) send(ctx context.Context, event C) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-o.done:
		return nil
	case <-time.After(time.Second):
		return nil
	case o.ch <- event:
		return nil
	}
}

func typeToString(o interface{}) string {
	r := reflect.TypeOf(o)

	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}

	return r.PkgPath() + "." + r.Name()
}
