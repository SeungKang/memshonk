package libplugin

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/SeungKang/memshonk/internal/dl"
	"github.com/SeungKang/memshonk/internal/plugins"
)

func newGoCallbacksList(process plugins.Process) *goCallbacksList {
	return &goCallbacksList{
		proc: process,
	}
}

// goCallbacksList allows us to reuse callback pointers (i.e., pointers that
// reference code in this program which can be used by plugins to execute
// said code). The purego library can only create a limited number of
// callback pointers and the pointers cannot be released once allocated.
//
// goCallbacksList works by managing access to one or more goCallbacks
// objects. The goCallbacks object stores the callback pointers and
// a reference to the plugin that is currently using them. This allows
// the Go callbacks to "know" which plugin they are associated with.
// The callback can then call code specific to each plugin such as
// memory allocation / free-ing functions.
//
// An advantage of this design is there is no additional locking in the
// "plugin host" (this) code outside of loading and unloading plugins.
// The disadvantage is that this code exists.
type goCallbacksList struct {
	proc       plugins.Process
	registerMu sync.Mutex
	list       []*goCallbacks
}

func (o *goCallbacksList) register(plugin *Plugin) (*goCallbacks, error) {
	o.registerMu.Lock()
	defer o.registerMu.Unlock()

	var target *goCallbacks

	for _, candidate := range o.list {
		if candidate.inUseBy == nil {
			candidate.inUseBy = plugin

			target = candidate

			break
		}
	}

	if target == nil {
		callbacks, err := newGoCallbacks(plugin, o.proc)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize new callbacks - %w", err)
		}

		o.list = append(o.list, callbacks)

		target = callbacks
	}

	err := setGoCallbackInPlugin(
		[]string{setReadFromProcFnName},
		target.readFromProcPtr,
		plugin.lib)
	if err != nil {
		return nil, fmt.Errorf("failed to set read from proc callback - %w", err)
	}

	err = setGoCallbackInPlugin(
		[]string{setWriteToProcFnName},
		target.writeToProcPtr,
		plugin.lib)
	if err != nil {
		return nil, fmt.Errorf("failed to set write to proc callback - %w", err)
	}

	return target, nil
}

func (o *goCallbacksList) deregister(target *goCallbacks) {
	o.registerMu.Lock()
	defer o.registerMu.Unlock()

	target.inUseBy = nil
}

func newGoCallbacks(plugin *Plugin, process plugins.Process) (*goCallbacks, error) {
	callbacks := &goCallbacks{
		inUseBy: plugin,
		process: process,
	}

	var err error

	callbacks.readFromProcPtr, err = dl.NewCallback(callbacks.readFromProc)
	if err != nil {
		return nil, err
	}

	callbacks.writeToProcPtr, err = dl.NewCallback(callbacks.writeToProc)
	if err != nil {
		return nil, err
	}

	return callbacks, nil
}

// goCallbacks creates a relationship between a plugin and "plugin-host" (this)
// code that plugins can execute. Please refer to goCallbacksList for more
// information.
type goCallbacks struct {
	inUseBy *Plugin
	process plugins.Process

	readFromProcPtr uintptr
	writeToProcPtr uintptr
}

func (o *goCallbacks) readFromProc(pluginAddr uintptr, size uintptr, procAddr uintptr) uintptr {
	if o.inUseBy == nil {
		panic("library is nil when go callback (readFromProc) was executed - this should never happen")
	}

	data, err := o.process.ReadFromAddr(procAddr, uint64(size))
	if err != nil {
		buf := allocSharedString(
			fmt.Sprintf("memshonk failed to read from process - %s", err),
			o.inUseBy.Alloc)

		return buf.Pointer()
	}

	if uintptr(len(data)) > size {
		msg := fmt.Sprintf("size of data returned from process "+
			"(%d bytes) is greater than dest buffer size (%d bytes)",
			size, len(data))

		buf := allocSharedString(msg, o.inUseBy.Alloc)

		return buf.Pointer()
	}

	pluginAddrCopy := pluginAddr

	for i := uintptr(0); i < size; i++ {
		b := (*byte)(unsafe.Pointer(pluginAddrCopy))

		*b = data[i]

		pluginAddrCopy++
	}

	return 0
}

func (o *goCallbacks) writeToProc(procAddr uintptr, size uintptr, pluginAddr uintptr) uintptr {
	if o.inUseBy == nil {
		panic("library is nil when go callback (writeToProc) was executed - this should never happen")
	}

	data := unsafe.Slice((*byte)(unsafe.Pointer(pluginAddr)), int(size))

	err := o.process.WriteToAddr(procAddr, data)
	if err != nil {
		buf := allocSharedString(
			fmt.Sprintf("memshonk failed to write to process - %s", err),
			o.inUseBy.Alloc)

		return buf.Pointer()
	}

	return 0
}
