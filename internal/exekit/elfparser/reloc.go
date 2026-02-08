package elfparser

import (
	"context"
	"fmt"

	"github.com/SeungKang/memshonk/internal/exekit/epc"
)

func (p *Parser) parseRel32(ctx context.Context, sh Elf32_Shdr) error {
	if p.cfg.OnRelocFn == nil {
		return nil
	}

	if sh.Size == 0 || sh.Entsize == 0 {
		return nil
	}

	// Load associated symbol table's string table for symbol names
	symStrtab, _ := p.loadOrReadStrtab32(sh.Link)

	count := sh.Size / sh.Entsize
	buf := make([]byte, sh.Entsize)

	for i := uint32(0); i < count; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		offset := int64(sh.Offset) + int64(i)*int64(sh.Entsize)
		if _, err := p.r.ReadAt(buf, offset); err != nil {
			return fmt.Errorf("reading relocation %d: %w", i, err)
		}

		rel := Elf32_Rel{
			Offset: p.byteOrder.Uint32(buf[0:4]),
			Info:   p.byteOrder.Uint32(buf[4:8]),
		}

		symIdx := ELF32_R_SYM(rel.Info)
		relType := ELF32_R_TYPE(rel.Info)

		// Get symbol name if possible
		symName := ""
		if symStrtab != nil {
			// Would need to look up symbol table entry to get name index
			// For now, leave empty
		}

		r := epc.Reloc{
			Offset:   uint64(rel.Offset),
			Type:     relType,
			Symbol:   symName,
			Addend:   0,
			SymIndex: symIdx,
		}
		if err := p.cfg.OnRelocFn(p.exeID, p.index, r); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) parseRela32(ctx context.Context, sh Elf32_Shdr) error {
	if p.cfg.OnRelocFn == nil {
		return nil
	}

	if sh.Size == 0 || sh.Entsize == 0 {
		return nil
	}

	count := sh.Size / sh.Entsize
	buf := make([]byte, sh.Entsize)

	for i := uint32(0); i < count; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		offset := int64(sh.Offset) + int64(i)*int64(sh.Entsize)
		if _, err := p.r.ReadAt(buf, offset); err != nil {
			return fmt.Errorf("reading relocation %d: %w", i, err)
		}

		rela := Elf32_Rela{
			Offset: p.byteOrder.Uint32(buf[0:4]),
			Info:   p.byteOrder.Uint32(buf[4:8]),
			Addend: int32(p.byteOrder.Uint32(buf[8:12])),
		}

		symIdx := ELF32_R_SYM(rela.Info)
		relType := ELF32_R_TYPE(rela.Info)

		r := epc.Reloc{
			Offset:   uint64(rela.Offset),
			Type:     relType,
			Symbol:   "",
			Addend:   int64(rela.Addend),
			SymIndex: symIdx,
		}
		if err := p.cfg.OnRelocFn(p.exeID, p.index, r); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) parseRel64(ctx context.Context, sh Elf64_Shdr) error {
	if p.cfg.OnRelocFn == nil {
		return nil
	}

	if sh.Size == 0 || sh.Entsize == 0 {
		return nil
	}

	count := sh.Size / sh.Entsize
	buf := make([]byte, sh.Entsize)

	for i := uint64(0); i < count; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		offset := int64(sh.Offset) + int64(i)*int64(sh.Entsize)
		if _, err := p.r.ReadAt(buf, offset); err != nil {
			return fmt.Errorf("reading relocation %d: %w", i, err)
		}

		rel := Elf64_Rel{
			Offset: p.byteOrder.Uint64(buf[0:8]),
			Info:   p.byteOrder.Uint64(buf[8:16]),
		}

		symIdx := ELF64_R_SYM(rel.Info)
		relType := ELF64_R_TYPE(rel.Info)

		r := epc.Reloc{
			Offset:   rel.Offset,
			Type:     relType,
			Symbol:   "",
			Addend:   0,
			SymIndex: symIdx,
		}
		if err := p.cfg.OnRelocFn(p.exeID, p.index, r); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) parseRela64(ctx context.Context, sh Elf64_Shdr) error {
	if p.cfg.OnRelocFn == nil {
		return nil
	}

	if sh.Size == 0 || sh.Entsize == 0 {
		return nil
	}

	count := sh.Size / sh.Entsize
	buf := make([]byte, sh.Entsize)

	for i := uint64(0); i < count; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		offset := int64(sh.Offset) + int64(i)*int64(sh.Entsize)
		if _, err := p.r.ReadAt(buf, offset); err != nil {
			return fmt.Errorf("reading relocation %d: %w", i, err)
		}

		rela := Elf64_Rela{
			Offset: p.byteOrder.Uint64(buf[0:8]),
			Info:   p.byteOrder.Uint64(buf[8:16]),
			Addend: int64(p.byteOrder.Uint64(buf[16:24])),
		}

		symIdx := ELF64_R_SYM(rela.Info)
		relType := ELF64_R_TYPE(rela.Info)

		r := epc.Reloc{
			Offset:   rela.Offset,
			Type:     relType,
			Symbol:   "",
			Addend:   rela.Addend,
			SymIndex: symIdx,
		}
		if err := p.cfg.OnRelocFn(p.exeID, p.index, r); err != nil {
			return err
		}
	}

	return nil
}
