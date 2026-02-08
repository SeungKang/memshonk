package machoparser

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/SeungKang/memshonk/internal/exekit/epc"
)

// Errors for fat binary parsing
var (
	ErrInvalidFatMagic = errors.New("invalid fat binary magic")
	ErrNoMatchingArch  = errors.New("no matching architecture in fat binary")
)

// ParseFat parses a fat (universal) binary.
func ParseFat(ctx context.Context, cfg *epc.ParserConfig) error {
	// Read fat header
	buf := make([]byte, 8)
	if _, err := cfg.Src.ReadAt(buf, 0); err != nil {
		if errors.Is(err, io.EOF) {
			return ErrTruncated
		}
		return fmt.Errorf("reading fat header: %w", err)
	}

	// Fat binaries are always big-endian
	magic := binary.BigEndian.Uint32(buf[0:4])
	nArch := binary.BigEndian.Uint32(buf[4:8])

	is64 := false
	switch magic {
	case FAT_MAGIC:
		is64 = false
	case FAT_MAGIC_64:
		is64 = true
	case FAT_CIGAM:
		// Little-endian fat magic - need to swap
		nArch = binary.LittleEndian.Uint32(buf[4:8])
		is64 = false
	case FAT_CIGAM_64:
		nArch = binary.LittleEndian.Uint32(buf[4:8])
		is64 = true
	default:
		return ErrInvalidFatMagic
	}

	// Check if user wants a specific architecture
	targetCPU := cfg.OptCPU
	targetBits := cfg.OptBits

	if is64 {
		return parseFat64(ctx, cfg, nArch, targetCPU, targetBits)
	}
	return parseFat32(ctx, cfg, nArch, targetCPU, targetBits)
}

func parseFat32(ctx context.Context, cfg *epc.ParserConfig, nArch uint32, targetCPU string, targetBits uint8) error {
	const archSize = 20 // Size of fat_arch (32-bit)

	var archs []FatArch32
	buf := make([]byte, archSize)

	for i := uint32(0); i < nArch; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		offset := int64(8 + i*archSize)
		if _, err := cfg.Src.ReadAt(buf, offset); err != nil {
			return fmt.Errorf("reading fat arch %d: %w", i, err)
		}

		arch := FatArch32{
			CPUType:    int32(binary.BigEndian.Uint32(buf[0:4])),
			CPUSubtype: int32(binary.BigEndian.Uint32(buf[4:8])),
			Offset:     binary.BigEndian.Uint32(buf[8:12]),
			Size:       binary.BigEndian.Uint32(buf[12:16]),
			Align:      binary.BigEndian.Uint32(buf[16:20]),
		}
		archs = append(archs, arch)
	}

	// If specific CPU requested, find it
	if targetCPU != "" || targetBits != 0 {
		for i, arch := range archs {
			if matchesTarget(arch.CPUType, targetCPU, targetBits) {
				exeID := cpuTypeString(arch.CPUType)
				return ParseAt(ctx, cfg, int64(arch.Offset), exeID, uint(i))
			}
		}
		return ErrNoMatchingArch
	}

	// Parse all architectures
	for i, arch := range archs {
		if err := ctx.Err(); err != nil {
			return err
		}

		exeID := cpuTypeString(arch.CPUType)
		if err := ParseAt(ctx, cfg, int64(arch.Offset), exeID, uint(i)); err != nil {
			return err
		}
	}

	return nil
}

func parseFat64(ctx context.Context, cfg *epc.ParserConfig, nArch uint32, targetCPU string, targetBits uint8) error {
	const archSize = 32 // Size of fat_arch_64

	var archs []FatArch64
	buf := make([]byte, archSize)

	for i := uint32(0); i < nArch; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		offset := int64(8 + i*archSize)
		if _, err := cfg.Src.ReadAt(buf, offset); err != nil {
			return fmt.Errorf("reading fat arch %d: %w", i, err)
		}

		arch := FatArch64{
			CPUType:    int32(binary.BigEndian.Uint32(buf[0:4])),
			CPUSubtype: int32(binary.BigEndian.Uint32(buf[4:8])),
			Offset:     binary.BigEndian.Uint64(buf[8:16]),
			Size:       binary.BigEndian.Uint64(buf[16:24]),
			Align:      binary.BigEndian.Uint32(buf[24:28]),
			Reserved:   binary.BigEndian.Uint32(buf[28:32]),
		}
		archs = append(archs, arch)
	}

	// If specific CPU requested, find it
	if targetCPU != "" || targetBits != 0 {
		for i, arch := range archs {
			if matchesTarget(arch.CPUType, targetCPU, targetBits) {
				exeID := cpuTypeString(arch.CPUType)
				return ParseAt(ctx, cfg, int64(arch.Offset), exeID, uint(i))
			}
		}
		return ErrNoMatchingArch
	}

	// Parse all architectures
	for i, arch := range archs {
		if err := ctx.Err(); err != nil {
			return err
		}

		exeID := cpuTypeString(arch.CPUType)
		if err := ParseAt(ctx, cfg, int64(arch.Offset), exeID, uint(i)); err != nil {
			return err
		}
	}

	return nil
}

func matchesTarget(cpuType int32, targetCPU string, targetBits uint8) bool {
	if targetCPU != "" {
		cpuName := cpuTypeString(cpuType)
		if cpuName != targetCPU {
			return false
		}
	}

	if targetBits != 0 {
		is64 := cpuType&CPU_ARCH_ABI64 != 0
		if targetBits == 64 && !is64 {
			return false
		}
		if targetBits == 32 && is64 {
			return false
		}
	}

	return true
}

// IsFatBinary checks if the magic number indicates a fat binary.
func IsFatBinary(magic uint32) bool {
	return magic == FAT_MAGIC || magic == FAT_CIGAM ||
		magic == FAT_MAGIC_64 || magic == FAT_CIGAM_64
}
