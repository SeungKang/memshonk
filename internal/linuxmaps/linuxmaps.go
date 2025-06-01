package linuxmaps

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/SeungKang/memshonk/internal/memory"
)

func Vmmap(pid int) (memory.Regions, error) {
	path := filepath.Join("/proc", fmt.Sprintf("%d", pid), "maps")
	file, err := os.Open(path)
	if err != nil {
		return memory.Regions{}, fmt.Errorf("failed to open proc maps path: %s - %w", path, err)
	}

	defer file.Close()

	return vmmap(file)
}

func vmmap(reader io.Reader) (memory.Regions, error) {
	var regions memory.Regions

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		region, err := lineToRegion(line)
		if err != nil {
			return memory.Regions{}, fmt.Errorf("failed to parse line into region: %q - %w", line, err)
		}

		regions.Add(region)
	}

	err := scanner.Err()
	if err != nil {
		return memory.Regions{}, err
	}

	return regions, nil
}

func lineToRegion(line string) (memory.Region, error) {
	fields := strings.Fields(line)
	if len(fields) < 5 {
		return memory.Region{}, fmt.Errorf("failed to parse line - %s", line)
	}

	addrRange := strings.Split(fields[0], "-")
	if len(addrRange) != 2 {
		return memory.Region{}, fmt.Errorf("failed to split line - %s", line)
	}

	baseAddr, err := strconv.ParseUint(addrRange[0], 16, 64)
	if err != nil {
		return memory.Region{}, err
	}

	endAddr, err := strconv.ParseUint(addrRange[1], 16, 64)
	if err != nil {
		return memory.Region{}, err
	}

	region := memory.Region{
		BaseAddr: uintptr(baseAddr),
		EndAddr:  uintptr(endAddr),
		Size:     endAddr - baseAddr,
		State:    memory.MemCommit,
		Type:     memory.MemAnon,
	}

	perms := fields[1]
	if strings.Contains(perms, "r") {
		region.Readable = true
	}

	if strings.Contains(perms, "w") {
		region.Writeable = true
	}

	if strings.Contains(perms, "x") {
		region.Executable = true
	}

	if strings.Contains(perms, "p") {
		region.Copyable = true
	}

	if strings.Contains(perms, "s") {
		region.Shared = true
	}

	var name string
	if len(fields) == 6 {
		name = fields[5]
		switch name {
		case "[heap]":
			region.Type = memory.MemHeap
		case "[stack]":
			region.Type = memory.MemStack
		case "[vvar]":
			region.Type = memory.MemVvar
		case "[vdso]":
			region.Type = memory.MemVdso
		default:
			region.Type = memory.MemMapped
		}
	}

	inodeStr := fields[4]
	if inodeStr != "0" {
		inode, err := strconv.ParseUint(inodeStr, 10, 64)
		if err != nil {
			return memory.Region{}, fmt.Errorf("failed to parse inode: %q - %w", inodeStr, err)
		}

		region.Parent = memory.ObjectMeta{
			IsSet:    true,
			ID:       memory.ObjectID(inode),
			FilePath: name,
			FileName: filepath.Base(name),
		}
	}

	return region, nil
}
