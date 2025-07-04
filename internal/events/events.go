package events

import (
	"context"
	"reflect"
	"sync"
	"time"
)

func NewEventsPubSub() *EventsPubSub {
	return &EventsPubSub{}
}

type EventsPubSub struct {
	rwMu   sync.RWMutex
	groups map[interface{}]*EventGroup
}

func (o *EventsPubSub) Subscribe(category interface{}) *EventSub {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	group, hasIt := o.groups[category]
	if !hasIt {
		if o.groups == nil {
			o.groups = make(map[interface{}]*EventGroup)
		}

		group = newEventGroup()

		o.groups[category] = group
	}

	return group.sub()
}

func (o *EventsPubSub) Publisher(category interface{}) *EventGroup {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	group, hasIt := o.groups[category]
	if !hasIt {
		if o.groups == nil {
			o.groups = make(map[interface{}]*EventGroup)
		}

		group = newEventGroup()

		o.groups[category] = group
	}

	return group
}

func newEventGroup() *EventGroup {
	return &EventGroup{}
}

type EventGroup struct {
	rwMu sync.RWMutex
	subs map[*EventSub]struct{}
}

func (o *EventGroup) sub() *EventSub {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if o.subs == nil {
		o.subs = make(map[*EventSub]struct{})
	}

	eventSub := newEventor(o)

	o.subs[eventSub] = struct{}{}

	return eventSub
}

func (o *EventGroup) Send(ctx context.Context, event interface{}) error {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	for eventor := range o.subs {
		err := eventor.send(ctx, event)
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *EventGroup) unsub(e *EventSub) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	delete(o.subs, e)

	if len(o.subs) == 0 {
		o.subs = nil
	}
}

func newEventor(parent *EventGroup) *EventSub {
	return &EventSub{
		parent: parent,
		ch:     make(chan interface{}, 10),
		done:   make(chan struct{}),
	}
}

type EventSub struct {
	parent *EventGroup
	ch     chan interface{}
	once   sync.Once
	done   chan struct{}
}

func (o *EventSub) Recv(ctx context.Context) (interface{}, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case event := <-o.ch:
		return event, nil
	}
}

func (o *EventSub) Unsubscribe() {
	o.once.Do(func() {
		close(o.done)

		o.parent.unsub(o)
	})
}

func (o *EventSub) send(ctx context.Context, event interface{}) error {
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
