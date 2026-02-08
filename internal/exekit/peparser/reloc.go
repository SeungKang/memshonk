package peparser

import (
	"context"

	"github.com/SeungKang/memshonk/internal/exekit/epc"
)

// parseRelocations parses the base relocation table and emits callbacks.
func (p *Parser) parseRelocations(ctx context.Context) error {
	if p.cfg.OnRelocFn == nil {
		return nil
	}

	dir := p.getDataDirectory(IMAGE_DIRECTORY_ENTRY_BASERELOC)
	if dir.VirtualAddress == 0 || dir.Size == 0 {
		return nil
	}

	offset, ok := p.rvaToOffset(dir.VirtualAddress)
	if !ok {
		return nil
	}

	endOffset := offset + dir.Size
	currentOffset := offset

	for currentOffset < endOffset {
		if err := ctx.Err(); err != nil {
			return err
		}

		// Read block header (8 bytes)
		var buf [8]byte
		if _, err := p.r.ReadAt(buf[:], int64(currentOffset)); err != nil {
			return nil // End of data
		}

		block := BaseRelocation{
			VirtualAddress: p.byteOrder.Uint32(buf[0:4]),
			SizeOfBlock:    p.byteOrder.Uint32(buf[4:8]),
		}

		// Validate block
		if block.SizeOfBlock == 0 {
			break // End of table
		}
		if block.SizeOfBlock < 8 {
			break // Invalid block size
		}
		if currentOffset+block.SizeOfBlock > endOffset {
			break // Block extends beyond table
		}

		// Calculate number of entries
		numEntries := (block.SizeOfBlock - 8) / 2

		// Read all entries for this block
		if numEntries > 0 {
			entriesBuf := make([]byte, numEntries*2)
			if _, err := p.r.ReadAt(entriesBuf, int64(currentOffset+8)); err != nil {
				break
			}

			for i := uint32(0); i < numEntries; i++ {
				if err := ctx.Err(); err != nil {
					return err
				}

				entry := p.byteOrder.Uint16(entriesBuf[i*2 : i*2+2])
				relType := RelocType(entry)
				relOffset := RelocOffset(entry)

				// Skip padding entries (type 0)
				if relType == IMAGE_REL_BASED_ABSOLUTE {
					continue
				}

				// Calculate the full RVA
				rva := block.VirtualAddress + uint32(relOffset)

				// Calculate file offset if possible
				var fileOffset uint64
				if off, ok := p.rvaToOffset(rva); ok {
					fileOffset = uint64(off)
				} else {
					fileOffset = uint64(rva) // Use RVA as fallback
				}

				reloc := epc.Reloc{
					Offset:   fileOffset,
					Type:     uint32(relType),
					Symbol:   "", // Base relocations don't have symbol names
					Addend:   0,
					SymIndex: 0,
				}

				if err := p.cfg.OnRelocFn(p.exeID, p.index, reloc); err != nil {
					return err
				}
			}
		}

		currentOffset += block.SizeOfBlock
	}

	return nil
}

// RelocTypeString returns a human-readable string for a relocation type.
func RelocTypeString(relType uint8) string {
	switch relType {
	case IMAGE_REL_BASED_ABSOLUTE:
		return "ABSOLUTE"
	case IMAGE_REL_BASED_HIGH:
		return "HIGH"
	case IMAGE_REL_BASED_LOW:
		return "LOW"
	case IMAGE_REL_BASED_HIGHLOW:
		return "HIGHLOW"
	case IMAGE_REL_BASED_HIGHADJ:
		return "HIGHADJ"
	case IMAGE_REL_BASED_MIPS_JMPADDR:
		return "MIPS_JMPADDR/ARM_MOV32/RISCV_HIGH20"
	case IMAGE_REL_BASED_THUMB_MOV32:
		return "THUMB_MOV32/RISCV_LOW12I"
	case IMAGE_REL_BASED_RISCV_LOW12S:
		return "RISCV_LOW12S/LOONGARCH_MARK_LA"
	case IMAGE_REL_BASED_MIPS_JMPADDR16:
		return "MIPS_JMPADDR16"
	case IMAGE_REL_BASED_DIR64:
		return "DIR64"
	default:
		return "UNKNOWN"
	}
}
