package elfparser

import (
	"context"
	"fmt"

	"github.com/SeungKang/memshonk/internal/exekit/epc"
)

func (p *Parser) parseDynamic32(ctx context.Context, sh Elf32_Shdr) error {
	if p.cfg.OnImportedLibraryFn == nil {
		return nil
	}

	if sh.Size == 0 || sh.Entsize == 0 {
		return nil
	}

	// Load the dynamic string table (sh.Link points to it)
	dynstr, _ := p.loadOrReadStrtab32(sh.Link)

	count := sh.Size / sh.Entsize
	buf := make([]byte, sh.Entsize)

	for i := uint32(0); i < count; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		offset := int64(sh.Offset) + int64(i)*int64(sh.Entsize)
		if _, err := p.r.ReadAt(buf, offset); err != nil {
			return fmt.Errorf("reading dynamic entry %d: %w", i, err)
		}

		tag := int32(p.byteOrder.Uint32(buf[0:4]))
		val := p.byteOrder.Uint32(buf[4:8])

		if tag == DT_NULL {
			break
		}

		if tag == DT_NEEDED && dynstr != nil {
			name := p.getString(dynstr, val)
			lib := epc.ImportedLibrary{
				Name:   name,
				Offset: uint64(offset),
			}
			if err := p.cfg.OnImportedLibraryFn(p.exeID, p.index, lib); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Parser) parseDynamic64(ctx context.Context, sh Elf64_Shdr) error {
	if p.cfg.OnImportedLibraryFn == nil {
		return nil
	}

	if sh.Size == 0 || sh.Entsize == 0 {
		return nil
	}

	// Load the dynamic string table (sh.Link points to it)
	dynstr, _ := p.loadOrReadStrtab64(sh.Link)

	count := sh.Size / sh.Entsize
	buf := make([]byte, sh.Entsize)

	for i := uint64(0); i < count; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		offset := int64(sh.Offset) + int64(i)*int64(sh.Entsize)
		if _, err := p.r.ReadAt(buf, offset); err != nil {
			return fmt.Errorf("reading dynamic entry %d: %w", i, err)
		}

		tag := int64(p.byteOrder.Uint64(buf[0:8]))
		val := p.byteOrder.Uint64(buf[8:16])

		if tag == DT_NULL {
			break
		}

		if tag == DT_NEEDED && dynstr != nil {
			name := p.getString(dynstr, uint32(val))
			lib := epc.ImportedLibrary{
				Name:   name,
				Offset: uint64(offset),
			}
			if err := p.cfg.OnImportedLibraryFn(p.exeID, p.index, lib); err != nil {
				return err
			}
		}
	}

	return nil
}
