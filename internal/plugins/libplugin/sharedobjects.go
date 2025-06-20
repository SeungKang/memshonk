package libplugin

import (
	"errors"
	"sync"
	"unsafe"
)

func newSharedObjects() *sharedObjects {
	// Note: We always start from non-zero value
	// so that an error object's ID implicitly
	// indicates that a failure occurred.
	return &sharedObjects{
		id:      1,
		idToObj: make(map[uint32]sharedObject),
		defErr:  []byte("unknown error id"),
	}
}

type sharedObjects struct {
	rwMu    sync.RWMutex
	id      uint32
	idToObj map[uint32]sharedObject
	defErr  []byte
}

func (o *sharedObjects) addErrorFromMemshonk(err error) uint32 {
	msg := []byte(err.Error() + "\x00")

	return o.addFromMemshonk(msg)
}

func (o *sharedObjects) addFromMemshonk(b []byte) uint32 {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	id := o.id

	o.id++

	o.idToObj[id] = sharedObject{
		data: b,
	}

	return id
}

func (o *sharedObjects) addFromPlugin(ptr uintptr) uintptr {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	// TODO: Should we free the string after?
	cstr := copyCStrByNull{
		strPtr: ptr,
	}

	id := o.id

	o.id++

	o.idToObj[id] = sharedObject{
		data: cstr.slice(),
	}

	return uintptr(id)
}

type sharedObject struct {
	data []byte
}

func (o *sharedObjects) getErrorFromMemshonk(id uint32) error {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	var err error

	errHolder, hasIt := o.idToObj[uint32(id)]
	if hasIt {
		err = errors.New(string(errHolder.data))
	} else {
		err = errors.New(string(o.defErr))
	}

	return err
}

func (o *sharedObjects) getFromPlugin(id uintptr) uintptr {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	var ptr unsafe.Pointer

	err, hasIt := o.idToObj[uint32(id)]
	if hasIt {
		ptr = unsafe.Pointer(&err.data[0])
	} else {
		ptr = unsafe.Pointer(&o.defErr[0])
	}

	// uintptr -> uintptr
	//rawPtr := (*uintptr)(unsafe.Pointer(pluginErrPtr))

	return uintptr(ptr)
}

func (o *sharedObjects) freeFromMemshonk(id uint32) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	delete(o.idToObj, id)
}

func (o *sharedObjects) freeFromPlugin(id uintptr) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	delete(o.idToObj, uint32(id))
}
