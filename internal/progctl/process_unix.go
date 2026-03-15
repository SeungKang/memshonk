//go:build unix

package progctl

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/SeungKang/memshonk/internal/ptrace"

	"golang.org/x/sys/unix"
)

var errPtraceNotEnabled = fmt.Errorf("ptrace memory mode is not enabled")

func attach(config attachConfig) (*process, error) {
	proc := &process{
		pid:     config.pid,
		endian:  binary.LittleEndian,
		exitMon: config.exitMon,
		memMode: config.memoryModeName,
	}

	err := proc.SetMemoryMode(config.memoryModeName)
	if err != nil {
		_ = proc.Close()

		return nil, fmt.Errorf("failed to set memory mode - %w",
			err)
	}

	regions, err := proc.Regions()
	if err != nil {
		_ = proc.Close()

		return nil, fmt.Errorf("failed to get memory regions - %w",
			err)
	}

	proc.exeInfo.Obj, err = regions.FirstObjectMatching(config.exeName)
	if err != nil {
		_ = proc.Close()

		return nil, fmt.Errorf("failed to get mapped object for exe - %w",
			err)
	}

	elfType, err := elfFileType(proc.exeInfo.Obj.Path)
	if err != nil {
		_ = proc.Close()

		return nil, fmt.Errorf("failed to get elf file type for %q - %w",
			proc.exeInfo.Obj.Path, err)
	}

	switch elfType {
	case 32:
		proc.exeInfo.Bits = 32
	default:
		proc.exeInfo.Bits = 64
	}

	// TODO: exit monitor
	//
	// go func() {
	// 	_, err := osProc.Wait()
	// 	proc.exitMon.SetExited(err)
	// }()

	return proc, nil
}

type process struct {
	pid     int
	stopped bool
	endian  binary.ByteOrder
	exeInfo ExeInfo
	exitMon *ExitMonitor
	memMode string

	optPtrace    *ptrace.TracerThread
	optProcFsMem *os.File
}

func (o *process) SetMemoryMode(modeName string) error {
	o.memMode = modeName
	switch modeName {
	case procfsMemoryMode:
		if o.optProcFsMem == nil {
			f, err := os.OpenFile(fmt.Sprintf("/proc/%d/mem", o.pid), os.O_RDWR, 0)
			if err != nil {
				return err
			}

			o.optProcFsMem = f
		}
	case ptraceMemoryMode:
		if o.optPtrace == nil {
			o.optPtrace = ptrace.NewTracerThread(o.pid)
		}
	}

	return nil
}

func (o *process) ExeInfo() ExeInfo {
	return o.exeInfo
}

func (o *process) ReadBytes(addr uintptr, sizeBytes uint64) ([]byte, error) {
	switch o.memMode {
	case ptraceMemoryMode:
		return o.readBytesPtrace(addr, sizeBytes)
	case procfsMemoryMode:
		return o.readBytesProcfs(addr, sizeBytes)
	default:
		return nil, unsupportedMemoryModeError(o.memMode)
	}
}

func (o *process) readBytesPtrace(addr uintptr, sizeBytes uint64) ([]byte, error) {
	if o.optPtrace == nil {
		return nil, errPtraceNotEnabled
	}

	needToResume := false

	if !o.stopped {
		needToResume = true

		err := o.Suspend()
		if err != nil {
			return nil, fmt.Errorf("failed to suspend process prior to peek data - %w", err)
		}
	}

	b := make([]byte, sizeBytes)

	_, peakErr := o.optPtrace.PeekData(context.Background(), addr, b)

	if needToResume {
		err := o.Resume()
		if err != nil {
			return nil, fmt.Errorf("failed to resume process after peek data - %w", err)
		}
	}

	return b, peakErr
}

func (o *process) readBytesProcfs(addr uintptr, sizeBytes uint64) ([]byte, error) {
	if o.optProcFsMem == nil {
		return nil, fmt.Errorf("procfsmem is nil (this should never happen)")
	}

	out := make([]byte, sizeBytes)

	_, err := unix.Pread(int(o.optProcFsMem.Fd()), out, int64(addr))
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (o *process) WriteBytes(data []byte, addr uintptr) error {
	switch o.memMode {
	case ptraceMemoryMode:
		return o.writeBytesPtrace(data, addr)
	case procfsMemoryMode:
		return o.writeBytesProcfs(data, addr)
	default:
		return unsupportedMemoryModeError(o.memMode)
	}
}

func (o *process) writeBytesPtrace(data []byte, addr uintptr) error {
	if o.optPtrace == nil {
		return errPtraceNotEnabled
	}

	needToResume := false

	if !o.stopped {
		needToResume = true

		err := o.Suspend()
		if err != nil {
			return fmt.Errorf("failed to suspend process prior to poke data - %w", err)
		}
	}

	_, pokeErr := o.optPtrace.PokeData(context.Background(), addr, data)

	if needToResume {
		err := o.Resume()
		if err != nil {
			return fmt.Errorf("failed to resume process after poke data - %w", err)
		}
	}

	return pokeErr
}

func (o *process) writeBytesProcfs(data []byte, addr uintptr) error {
	if o.optProcFsMem == nil {
		return fmt.Errorf("procfsmem is nil (this should never happen)")
	}

	_, err := unix.Pwrite(int(o.optProcFsMem.Fd()), data, int64(addr))
	return err
}

func (o *process) ReadPtr(at uintptr) (uintptr, error) {
	if o.exeInfo.Bits == 32 {
		b, err := o.ReadBytes(at, 4)
		if err != nil {
			return 0, err
		}

		if len(b) != 4 {
			return 0, fmt.Errorf("tried to read 4 bytes - only got %d bytes",
				len(b))
		}

		return uintptr(o.endian.Uint32(b)), nil
	} else {
		b, err := o.ReadBytes(at, 8)
		if err != nil {
			return 0, err
		}

		if len(b) != 8 {
			return 0, fmt.Errorf("tried to read 8 bytes - only got %d bytes",
				len(b))
		}

		return uintptr(o.endian.Uint64(b)), nil
	}
}

func (o *process) Suspend() error {
	if o.optPtrace == nil {
		return errPtraceNotEnabled
	}

	err := o.optPtrace.AttachAndWaitStopped(context.Background())
	if err != nil {
		return fmt.Errorf("failed to attach to process - %w", err)
	}

	o.stopped = true

	return nil
}

func (o *process) Resume() error {
	if o.optPtrace == nil {
		return errPtraceNotEnabled
	}

	err := o.optPtrace.Detach(context.Background())
	if err != nil {
		return fmt.Errorf("failed to ptrace detach - %w", err)
	}

	o.stopped = false

	return nil
}

func (o *process) Close() error {
	o.exitMon.SetExited(ErrDetached)

	if o.optProcFsMem != nil {
		_ = o.optProcFsMem.Close()
	}

	// everything after this point is ptrace related, since there is no
	// ptrace skip the rest of this function
	if o.optPtrace == nil {
		return nil
	}

	// TODO: on linux the process needs to be stopped according to the
	// PTRACE_DETACH section in the linux manual page unsure what other
	// unix-like operating systems require
	if !o.stopped {
		err := o.Suspend()
		if err != nil {
			return fmt.Errorf("failed to suspend process prior to ptrace detach - %w", err)
		}
	}

	err := o.optPtrace.Detach(context.Background())
	if err != nil {
		return fmt.Errorf("failed to ptrace detach - %w", err)
	}

	_ = o.optPtrace.Close()

	return nil
}

func elfFileType(elfPath string) (uint32, error) {
	// Based on this post by Stackexchange user Alexios:
	// https://unix.stackexchange.com/a/106235
	//
	// $ file /bin/sh
	// /bin/sh: ELF 64-bit LSB pie executable (...)
	// $ head -c 5 /bin/sh | hexdump -C
	// 00000000  7f 45 4c 46 02  |.ELF.|
	elfFile, err := os.Open(elfPath)
	if err != nil {
		return 0, err
	}
	defer elfFile.Close()

	header := make([]byte, 5)

	_, err = elfFile.Read(header)
	if err != nil {
		return 0, err
	}

	switch {
	case bytes.Equal(
		header,
		[]byte{0x7f, 0x45, 0x4c, 0x46, 0x01},
	):
		return 32, nil
	case bytes.Equal(
		header,
		[]byte{0x7f, 0x45, 0x4c, 0x46, 0x02},
	):
		return 64, nil
	default:
		return 0, fmt.Errorf("unknown elf header bytes: 0x%x", header)
	}
}
