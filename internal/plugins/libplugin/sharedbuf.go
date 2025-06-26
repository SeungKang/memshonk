package libplugin

import (
	"encoding/binary"
	"unsafe"
)

func stringFromSharedBufRef(bufRef uintptr, freeMemFn func(SharedBuf)) string {
	return string(bytesFromSharedBufRef(bufRef, freeMemFn))
}

func bytesFromSharedBufRef(bufRef uintptr, freeMemFn func(SharedBuf)) []byte {
	if bufRef == 0 {
		return nil
	}

	buf := sharedBufFromPtr(bufRef)

	b := buf.CopyBytes()

	freeMemFn(buf)

	return b
}

func sharedBufFromPtr(ptr uintptr) SharedBuf {
	sizeSlice := make([]byte, 4)

	for i := uintptr(0); i < 4; i++ {
		sizeSlice[i] = *(*byte)(unsafe.Pointer(ptr + i))
	}

	size := binary.LittleEndian.Uint32(sizeSlice)

	dataPtr := ptr + 4

	startOfData := (*byte)(unsafe.Pointer(dataPtr))

	dataSlice := unsafe.Slice(startOfData, int(size))

	return SharedBuf{
		ptr:  ptr,
		size: size,
		data: dataSlice,
	}
}

func allocSharedString(str string, allocFn func(sizeBytes uint32) (SharedBuf, error)) SharedBuf {
	buf, err := allocFn(uint32(len(str)))
	if err != nil {
		panic("plugin failed to allocate memory for shared buffer - " + err.Error())
	}

	buf.WriteString(str)

	return buf
}

type SharedBuf struct {
	ptr  uintptr
	size uint32
	data []byte
}

func (o *SharedBuf) Pointer() uintptr {
	return o.ptr
}

func (o *SharedBuf) WriteString(str string) {
	for i := uint32(0); i < uint32(len(str)); i++ {
		o.data[i] = str[i]
	}
}

func (o *SharedBuf) CopyString() string {
	return string(o.CopyBytes())
}

func (o *SharedBuf) CopyBytes() []byte {
	tmp := make([]byte, len(o.data))

	for i := range o.data {
		tmp[i] = o.data[i]
	}

	return tmp
}
