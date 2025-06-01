package fbsdmaps

import (
	"path/filepath"
	"unsafe"

	"github.com/SeungKang/memshonk/internal/memory"

	"golang.org/x/sys/unix"
)

type Ptrace interface {
	RequestPtr(request int, addr unsafe.Pointer, data int) error
}

func Vmmap(stoppedPtrace Ptrace) (memory.Regions, error) {
	var regions memory.Regions

	// From "man 2 ptrace":
	//
	// The first entry is returned by setting pve_entry to
	// zero. Subsequent entries are returned by leaving
	// pve_entry unmodified from the value returned by
	// previous requests.
	lastID := int32(0)
	var gotOne bool

	for {
		var entry ptrace_vm_entry
		entry.PveEntry = lastID

		pathBuf := make([]byte, 4096)
		entry.PvePathlen = uint32(len(pathBuf))
		entry.PvePath = uintptr(unsafe.Pointer(&pathBuf[0]))

		err := stoppedPtrace.RequestPtr(unix.PT_VM_ENTRY, unsafe.Pointer(&entry), 0)
		if err != nil {
			if gotOne {
				return regions, nil
			}

			return memory.Regions{}, err
		}

		gotOne = true

		lastID = entry.PveEntry

		regions.Add(vmEntryToMemoryRegions(entry, pathBuf))
	}
}

// From "man 2 ptrace" and /sys/sys/ptrace.h:
//
//	struct ptrace_vm_entry {
//	    int       pve_entry;     /* Entry number used for iteration. */
//	    int       pve_timestamp; /* Generation number of VM map. */
//	    u_long    pve_start;     /* Start VA of range. */
//	    u_long    ppve_end;      /* End VA of range (incl). */
//	    u_long    pve_offset;    /* Offset in backing object. */
//	    u_int     pve_prot;      /* Protection of memory range. */
//	    u_int     pve_pathlen;   /* Size of path. */
//	    long      pve_fileid;    /* File ID. */
//	    uint32_t  pve_fsid;      /* File system ID. */
//	    char      *pve_path;     /* Path name of object. */
//	};
type ptrace_vm_entry struct {
	PveEntry     int32
	PveTimestamp int32
	PveStart     uintptr
	PveEnd       uintptr
	PveOffset    uintptr
	PveProt      uint32
	PvePathlen   uint32
	PveFileid    int64
	PveFsid      uint32
	PvePath      uintptr
}

func vmEntryToMemoryRegions(entry ptrace_vm_entry, pathBuf []byte) memory.Region {
	region := memory.Region{
		BaseAddr: entry.PveStart,
		EndAddr:  entry.PveEnd,
		Size:     uint64(entry.PveEnd - entry.PveStart),
		State:    memory.MemCommit,
	}

	if entry.PveFileid > 0 {
		region.Type = memory.MemMapped

		region.Parent = memory.ObjectMeta{
			IsSet: true,
			ID:    memory.ObjectID(entry.PveFileid),
		}
	} else {
		region.Type = memory.MemPrivate
	}

	// From "man 2 ptrace":
	//
	// The pve_pathlen field is updated with the actual
	// length of the pathname (including the terminating
	// null character).
	if entry.PvePathlen > 0 && entry.PvePathlen < uint32(len(pathBuf)) {
		// abcd(0x00)
		// ----- 5
		// buf[0:5]
		p := string(pathBuf[0 : entry.PvePathlen-1])
		region.Parent.FilePath = p
		region.Parent.FileName = filepath.Base(p)
	}

	// From sys/vm/vm.h:
	//
	// VM_PROT_NONE       ((vm_prot_t) 0x00)
	// VM_PROT_READ	      ((vm_prot_t) 0x01)
	// VM_PROT_WRITE      ((vm_prot_t) 0x02)
	// VM_PROT_EXECUTE    ((vm_prot_t) 0x04)
	// VM_PROT_COPY       ((vm_prot_t) 0x08) /* copy-on-read */
	// VM_PROT_PRIV_FLAG  ((vm_prot_t) 0x10)

	if entry.PveProt&0x01 != 0 {
		region.Readable = true
	}

	if entry.PveProt&0x02 != 0 {
		region.Writeable = true
	}

	if entry.PveProt&0x04 != 0 {
		region.Executable = true
	}

	if entry.PveProt&0x08 != 0 {
		region.Copyable = true
	}

	if entry.PveProt&0x010 != 0 {
		region.Type = memory.MemPrivate
	}

	return region
}

func cstrToString(buf []byte) string {
	var i int

	for ; i < len(buf); i++ {
		if buf[i] == 0x00 {
			break
		}
	}

	return string(buf[0:i])
}
