package machoparser

import (
	"context"
	"fmt"

	"github.com/SeungKang/memshonk/internal/exekit/epc"
)

// SectionReloc holds section information needed for relocation parsing.
type SectionReloc struct {
	Name   string
	RelOff uint32
	NReloc uint32
}

// parseRelocations parses relocation entries for a section.
func (p *Parser) parseRelocations(ctx context.Context, sec SectionReloc) error {
	if p.cfg.OnRelocFn == nil || sec.NReloc == 0 {
		return nil
	}

	const relocSize = 8 // Each relocation entry is 8 bytes
	buf := make([]byte, relocSize)

	for i := uint32(0); i < sec.NReloc; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		offset := p.baseOff + int64(sec.RelOff) + int64(i)*relocSize
		if _, err := p.r.ReadAt(buf, offset); err != nil {
			return fmt.Errorf("reading relocation %d for section %s: %w", i, sec.Name, err)
		}

		// Parse the relocation entry
		rAddress := int32(p.byteOrder.Uint32(buf[0:4]))
		rInfo := p.byteOrder.Uint32(buf[4:8])

		// Check if this is a scattered relocation (high bit of r_address set)
		if rAddress < 0 { // High bit set indicates scattered
			reloc := p.parseScatteredReloc(rAddress, rInfo, sec.Name)
			if err := p.cfg.OnRelocFn(p.exeID, p.index, reloc); err != nil {
				return err
			}
		} else {
			reloc := p.parseStandardReloc(rAddress, rInfo, sec.Name)
			if err := p.cfg.OnRelocFn(p.exeID, p.index, reloc); err != nil {
				return err
			}
		}
	}

	return nil
}

// parseStandardReloc parses a standard (non-scattered) relocation entry.
func (p *Parser) parseStandardReloc(rAddress int32, rInfo uint32, secName string) epc.Reloc {
	// Extract fields from packed rInfo:
	// r_symbolnum: 24 bits (bits 0-23)
	// r_pcrel: 1 bit (bit 24)
	// r_length: 2 bits (bits 25-26)
	// r_extern: 1 bit (bit 27)
	// r_type: 4 bits (bits 28-31)
	rSymbolNum := rInfo & 0x00FFFFFF
	rPCRel := (rInfo>>24)&1 != 0
	rLength := uint8((rInfo >> 25) & 0x3)
	rExtern := (rInfo>>27)&1 != 0
	rType := uint8((rInfo >> 28) & 0xF)

	var symbolName string
	if rExtern {
		// r_symbolnum is an index into the symbol table
		symbolName = p.getSymbolName(rSymbolNum)
	} else {
		// r_symbolnum is a section number
		symbolName = fmt.Sprintf("section_%d", rSymbolNum)
	}

	// Calculate addend based on r_length (this is a simplification;
	// actual addend may need to be read from the relocation target)
	var addend int64
	if rPCRel {
		// For PC-relative relocations, addend often relates to instruction length
		addend = -int64(1 << rLength)
	}

	return epc.Reloc{
		Offset:   uint64(rAddress),
		Type:     uint32(rType),
		Symbol:   symbolName,
		Addend:   addend,
		SymIndex: rSymbolNum,
	}
}

// parseScatteredReloc parses a scattered relocation entry.
// Scattered relocations are used for expressions involving section differences
// or non-zero constants.
func (p *Parser) parseScatteredReloc(rAddress int32, rValue uint32, secName string) epc.Reloc {
	// For scattered relocations, the first word is packed differently:
	// r_scattered: 1 bit (bit 31, always 1)
	// r_pcrel: 1 bit (bit 30)
	// r_length: 2 bits (bits 28-29)
	// r_type: 4 bits (bits 24-27)
	// r_address: 24 bits (bits 0-23)

	// The rAddress parameter has the high bit set, so we need to interpret it as unsigned
	firstWord := uint32(rAddress)
	rPCRel := (firstWord>>30)&1 != 0
	rLength := uint8((firstWord >> 28) & 0x3)
	rType := uint8((firstWord >> 24) & 0xF)
	address := firstWord & 0x00FFFFFF

	_ = rPCRel  // Used for addend calculation if needed
	_ = rLength // Could be used for size information

	return epc.Reloc{
		Offset:   uint64(address),
		Type:     uint32(rType),
		Symbol:   fmt.Sprintf("value_0x%x", rValue),
		Addend:   int64(int32(rValue)), // r_value is the address of the relocatable expression
		SymIndex: 0,
	}
}

// getSymbolName returns the name of a symbol given its index.
func (p *Parser) getSymbolName(symIndex uint32) string {
	// We need to read the symbol table entry to get the string table index.
	// This requires knowing the symbol table offset, which we get from LC_SYMTAB.
	// For now, we'll return a placeholder if we can't resolve.
	// The actual implementation would need to cache or re-read symbol entries.

	// Since we process symbols during parseSymtab, we could cache symbol names,
	// but for now we'll return an index-based name.
	// A proper implementation would maintain a symbol name cache.
	return fmt.Sprintf("sym_%d", symIndex)
}

// relocTypeName returns a human-readable name for a relocation type.
// This is architecture-specific.
func (p *Parser) relocTypeName(rType uint8, cpuType int32) string {
	switch cpuType {
	case CPU_TYPE_X86_64:
		switch rType {
		case X86_64_RELOC_UNSIGNED:
			return "UNSIGNED"
		case X86_64_RELOC_SIGNED:
			return "SIGNED"
		case X86_64_RELOC_BRANCH:
			return "BRANCH"
		case X86_64_RELOC_GOT_LOAD:
			return "GOT_LOAD"
		case X86_64_RELOC_GOT:
			return "GOT"
		case X86_64_RELOC_SUBTRACTOR:
			return "SUBTRACTOR"
		case X86_64_RELOC_SIGNED_1:
			return "SIGNED_1"
		case X86_64_RELOC_SIGNED_2:
			return "SIGNED_2"
		case X86_64_RELOC_SIGNED_4:
			return "SIGNED_4"
		case X86_64_RELOC_TLV:
			return "TLV"
		}
	case CPU_TYPE_ARM64:
		switch rType {
		case ARM64_RELOC_UNSIGNED:
			return "UNSIGNED"
		case ARM64_RELOC_SUBTRACTOR:
			return "SUBTRACTOR"
		case ARM64_RELOC_BRANCH26:
			return "BRANCH26"
		case ARM64_RELOC_PAGE21:
			return "PAGE21"
		case ARM64_RELOC_PAGEOFF12:
			return "PAGEOFF12"
		case ARM64_RELOC_GOT_LOAD_PAGE21:
			return "GOT_LOAD_PAGE21"
		case ARM64_RELOC_GOT_LOAD_PAGEOFF12:
			return "GOT_LOAD_PAGEOFF12"
		case ARM64_RELOC_POINTER_TO_GOT:
			return "POINTER_TO_GOT"
		case ARM64_RELOC_TLVP_LOAD_PAGE21:
			return "TLVP_LOAD_PAGE21"
		case ARM64_RELOC_TLVP_LOAD_PAGEOFF12:
			return "TLVP_LOAD_PAGEOFF12"
		case ARM64_RELOC_ADDEND:
			return "ADDEND"
		}
	case CPU_TYPE_X86: // CPU_TYPE_I386 is an alias for CPU_TYPE_X86
		switch rType {
		case GENERIC_RELOC_VANILLA:
			return "VANILLA"
		case GENERIC_RELOC_PAIR:
			return "PAIR"
		case GENERIC_RELOC_SECTDIFF:
			return "SECTDIFF"
		case GENERIC_RELOC_PB_LA_PTR:
			return "PB_LA_PTR"
		case GENERIC_RELOC_LOCAL_SECTDIFF:
			return "LOCAL_SECTDIFF"
		case GENERIC_RELOC_TLV:
			return "TLV"
		}
	case CPU_TYPE_ARM:
		switch rType {
		case ARM_RELOC_VANILLA:
			return "VANILLA"
		case ARM_RELOC_PAIR:
			return "PAIR"
		case ARM_RELOC_SECTDIFF:
			return "SECTDIFF"
		case ARM_RELOC_LOCAL_SECTDIFF:
			return "LOCAL_SECTDIFF"
		case ARM_RELOC_PB_LA_PTR:
			return "PB_LA_PTR"
		case ARM_RELOC_BR24:
			return "BR24"
		case ARM_THUMB_RELOC_BR22:
			return "THUMB_BR22"
		case ARM_THUMB_32BIT_BRANCH:
			return "THUMB_32BIT_BRANCH"
		case ARM_RELOC_HALF:
			return "HALF"
		case ARM_RELOC_HALF_SECTDIFF:
			return "HALF_SECTDIFF"
		}
	}
	return fmt.Sprintf("TYPE_%d", rType)
}
