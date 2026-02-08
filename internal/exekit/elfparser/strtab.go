package elfparser

import (
	"context"
	"fmt"

	"github.com/SeungKang/memshonk/internal/exekit/epc"
)

func (p *Parser) parseStrtab32(ctx context.Context, sh Elf32_Shdr, secIdx uint32) error {
	if sh.Size == 0 {
		return nil
	}

	// Read and cache the string table
	strtab := make([]byte, sh.Size)
	if _, err := p.r.ReadAt(strtab, int64(sh.Offset)); err != nil {
		return fmt.Errorf("reading string table: %w", err)
	}
	p.strtabs[secIdx] = strtab

	// Report strings if callback is set
	if p.cfg.OnStringFn == nil {
		return nil
	}

	secName := p.getSectionName(sh.Name)
	source := "strtab"
	if secName == ".dynstr" {
		source = "dynstr"
	} else if secName == ".shstrtab" {
		source = "shstrtab"
	}

	// Parse individual strings from the table
	return p.emitStrings(ctx, strtab, uint64(sh.Offset), source)
}

func (p *Parser) parseStrtab64(ctx context.Context, sh Elf64_Shdr, secIdx uint32) error {
	if sh.Size == 0 {
		return nil
	}

	// Read and cache the string table
	strtab := make([]byte, sh.Size)
	if _, err := p.r.ReadAt(strtab, int64(sh.Offset)); err != nil {
		return fmt.Errorf("reading string table: %w", err)
	}
	p.strtabs[secIdx] = strtab

	// Report strings if callback is set
	if p.cfg.OnStringFn == nil {
		return nil
	}

	secName := p.getSectionName(sh.Name)
	source := "strtab"
	if secName == ".dynstr" {
		source = "dynstr"
	} else if secName == ".shstrtab" {
		source = "shstrtab"
	}

	// Parse individual strings from the table
	return p.emitStrings(ctx, strtab, sh.Offset, source)
}

func (p *Parser) emitStrings(ctx context.Context, strtab []byte, baseOffset uint64, source string) error {
	i := uint64(0)
	for i < uint64(len(strtab)) {
		if err := ctx.Err(); err != nil {
			return err
		}

		// Skip null bytes
		if strtab[i] == 0 {
			i++
			continue
		}

		// Find end of string
		start := i
		for i < uint64(len(strtab)) && strtab[i] != 0 {
			i++
		}

		str := string(strtab[start:i])
		if len(str) > 0 {
			s := epc.String{
				Value:  str,
				Offset: baseOffset + start,
				Source: source,
			}
			if err := p.cfg.OnStringFn(p.exeID, p.index, s); err != nil {
				return err
			}
		}

		i++ // Skip null terminator
	}

	return nil
}
