//go:build unix

package progctl

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/SeungKang/memshonk/internal/memory"
	"github.com/SeungKang/memshonk/internal/ptrace"
)

var _ attachedProcess = (*unixProcess)(nil)

func attach(exeName string, pid int) (*unixProcess, error) {
	proc := &unixProcess{
		pid:     pid,
		ptrace:  ptrace.New(pid),
		endian:  binary.LittleEndian,
		exitMon: newExitMonitor(),
	}

	regions, err := proc.Regions()
	if err != nil {
		proc.Close()

		return nil, fmt.Errorf("failed to get memory regions - %w",
			err)
	}

	proc.exeObj, err = regions.FirstObjectMatching(exeName)
	if err != nil {
		proc.Close()

		return nil, fmt.Errorf("failed to get mapped object for exe - %w",
			err)
	}

	elfType, err := elfFileType(proc.exeObj.Path)
	if err != nil {
		proc.Close()

		return nil, fmt.Errorf("failed to get elf file type for %q - %w",
			proc.exeObj.Path, err)
	}

	switch elfType {
	case 32:
		proc.is32b = true
	}

	// TODO: exit monitor
	//
	// go func() {
	// 	_, err := osProc.Wait()
	// 	proc.exitMon.SetExited(err)
	// }()

	return proc, nil
}

type unixProcess struct {
	pid     int
	ptrace  *ptrace.Tracer
	stopped bool
	endian  binary.ByteOrder
	is32b   bool
	exeObj  memory.Object
	exitMon *ExitMonitor
}

func (o *unixProcess) ExitMonitor() *ExitMonitor {
	return o.exitMon
}

func (o *unixProcess) PID() int {
	return o.pid
}

func (o *unixProcess) ExeObj() memory.Object {
	return o.exeObj
}

func (o *unixProcess) ReadBytes(addr uintptr, sizeBytes uint64) ([]byte, error) {
	b := make([]byte, sizeBytes)

	_, err := o.ptrace.PeekData(addr, b)
	if err != nil {
		return nil, fmt.Errorf("failed to peek data - %w", err)
	}

	return b, nil
}

func (o *unixProcess) WriteBytes(b []byte, addr uintptr) error {
	_, err := o.ptrace.PokeData(addr, b)
	if err != nil {
		return fmt.Errorf("failed to poke data - %w", err)
	}

	return nil
}

func (o *unixProcess) ReadPtr(at uintptr) (uintptr, error) {
	if o.is32b {
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

func (o *unixProcess) Suspend() error {
	err := o.ptrace.AttachAndWaitStopped()
	if err != nil {
		return fmt.Errorf("failed to attach to process - %w", err)
	}

	o.stopped = true

	return nil
}

func (o *unixProcess) Resume() error {
	err := o.ptrace.Detach()
	if err != nil {
		return fmt.Errorf("failed to ptrace detach - %w", err)
	}

	o.stopped = false

	return nil
}

func (o *unixProcess) Close() error {
	return o.ptrace.Detach()
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
