package elfparser

import (
	"context"
	"fmt"

	"github.com/SeungKang/memshonk/internal/exekit/epc"
)

func (p *Parser) parseSymtab32(ctx context.Context, sh Elf32_Shdr, secIdx uint32) error {
	// Check if we need to parse this section at all
	isDynsym := sh.Type == SHT_DYNSYM
	needSymbols := p.cfg.OnSymbolFn != nil
	needFunctions := p.cfg.OnFunctionFn != nil
	needImports := p.cfg.OnImportedCodeFn != nil && isDynsym
	needExports := p.cfg.OnExportedCodeFn != nil && isDynsym

	if !needSymbols && !needFunctions && !needImports && !needExports {
		return nil
	}

	if sh.Size == 0 || sh.Entsize == 0 {
		return nil
	}

	// Load associated string table
	strtab, err := p.loadOrReadStrtab32(sh.Link)
	if err != nil {
		return err
	}

	count := sh.Size / sh.Entsize
	buf := make([]byte, sh.Entsize)

	for i := uint32(0); i < count; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		offset := int64(sh.Offset) + int64(i)*int64(sh.Entsize)
		if _, err := p.r.ReadAt(buf, offset); err != nil {
			return fmt.Errorf("reading symbol %d: %w", i, err)
		}

		sym := Elf32_Sym{
			Name:  p.byteOrder.Uint32(buf[0:4]),
			Value: p.byteOrder.Uint32(buf[4:8]),
			Size:  p.byteOrder.Uint32(buf[8:12]),
			Info:  buf[12],
			Other: buf[13],
			Shndx: p.byteOrder.Uint16(buf[14:16]),
		}

		name := p.getString(strtab, sym.Name)
		symType := ST_TYPE(sym.Info)
		binding := ST_BIND(sym.Info)

		// Report symbol
		if needSymbols {
			s := epc.Symbol{
				Name:    name,
				Offset:  uint64(offset),
				Addr:    uint64(sym.Value),
				Size:    uint64(sym.Size),
				Type:    symType,
				Binding: binding,
				Section: sym.Shndx,
				Other:   sym.Other,
			}
			if err := p.cfg.OnSymbolFn(p.exeID, p.index, s); err != nil {
				return err
			}
		}

		// Report function if this is a defined function symbol
		if needFunctions && symType == STT_FUNC && sym.Value != 0 && sym.Shndx != SHN_UNDEF {
			f := epc.Function{
				Name:   name,
				Offset: uint64(sym.Value),
				Addr:   uint64(sym.Value),
				Size:   uint64(sym.Size),
			}
			if err := p.cfg.OnFunctionFn(p.exeID, p.index, f); err != nil {
				return err
			}
		}

		// Report imported code if this is an undefined symbol in dynsym
		// These are symbols that need to be resolved from external libraries
		if needImports && sym.Shndx == SHN_UNDEF && name != "" {
			// Only report functions and objects (not NOTYPE which is often the null symbol)
			if symType == STT_FUNC || symType == STT_OBJECT {
				ic := epc.ImportedCode{
					Name:    name,
					Library: "", // Library association requires version info parsing
					Offset:  uint64(offset),
					Type:    symType,
					Binding: binding,
				}
				if err := p.cfg.OnImportedCodeFn(p.exeID, p.index, ic); err != nil {
					return err
				}
			}
		}

		// Report exported code if this is a defined, global/weak symbol in dynsym
		// These are symbols that can be used by other executables or libraries
		if needExports && sym.Shndx != SHN_UNDEF && name != "" {
			// Only report global or weak symbols (not local)
			if binding == STB_GLOBAL || binding == STB_WEAK {
				// Only report functions and objects
				if symType == STT_FUNC || symType == STT_OBJECT {
					ec := epc.ExportedCode{
						Name:    name,
						Offset:  uint64(offset),
						Addr:    uint64(sym.Value),
						Size:    uint64(sym.Size),
						Type:    symType,
						Binding: binding,
					}
					if err := p.cfg.OnExportedCodeFn(p.exeID, p.index, ec); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func (p *Parser) parseSymtab64(ctx context.Context, sh Elf64_Shdr, secIdx uint32) error {
	// Check if we need to parse this section at all
	isDynsym := sh.Type == SHT_DYNSYM
	needSymbols := p.cfg.OnSymbolFn != nil
	needFunctions := p.cfg.OnFunctionFn != nil
	needImports := p.cfg.OnImportedCodeFn != nil && isDynsym
	needExports := p.cfg.OnExportedCodeFn != nil && isDynsym

	if !needSymbols && !needFunctions && !needImports && !needExports {
		return nil
	}

	if sh.Size == 0 || sh.Entsize == 0 {
		return nil
	}

	// Load associated string table
	strtab, err := p.loadOrReadStrtab64(sh.Link)
	if err != nil {
		return err
	}

	count := sh.Size / sh.Entsize
	buf := make([]byte, sh.Entsize)

	for i := uint64(0); i < count; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		offset := int64(sh.Offset) + int64(i)*int64(sh.Entsize)
		if _, err := p.r.ReadAt(buf, offset); err != nil {
			return fmt.Errorf("reading symbol %d: %w", i, err)
		}

		sym := Elf64_Sym{
			Name:  p.byteOrder.Uint32(buf[0:4]),
			Info:  buf[4],
			Other: buf[5],
			Shndx: p.byteOrder.Uint16(buf[6:8]),
			Value: p.byteOrder.Uint64(buf[8:16]),
			Size:  p.byteOrder.Uint64(buf[16:24]),
		}

		name := p.getString(strtab, sym.Name)
		symType := ST_TYPE(sym.Info)
		binding := ST_BIND(sym.Info)

		// Report symbol
		if needSymbols {
			s := epc.Symbol{
				Name:    name,
				Offset:  uint64(offset),
				Addr:    sym.Value,
				Size:    sym.Size,
				Type:    symType,
				Binding: binding,
				Section: sym.Shndx,
				Other:   sym.Other,
			}
			if err := p.cfg.OnSymbolFn(p.exeID, p.index, s); err != nil {
				return err
			}
		}

		// Report function if this is a defined function symbol
		if needFunctions && symType == STT_FUNC && sym.Value != 0 && sym.Shndx != SHN_UNDEF {
			f := epc.Function{
				Name:   name,
				Offset: sym.Value,
				Addr:   sym.Value,
				Size:   sym.Size,
			}
			if err := p.cfg.OnFunctionFn(p.exeID, p.index, f); err != nil {
				return err
			}
		}

		// Report imported code if this is an undefined symbol in dynsym
		// These are symbols that need to be resolved from external libraries
		if needImports && sym.Shndx == SHN_UNDEF && name != "" {
			// Only report functions and objects (not NOTYPE which is often the null symbol)
			if symType == STT_FUNC || symType == STT_OBJECT {
				ic := epc.ImportedCode{
					Name:    name,
					Library: "", // Library association requires version info parsing
					Offset:  uint64(offset),
					Type:    symType,
					Binding: binding,
				}
				if err := p.cfg.OnImportedCodeFn(p.exeID, p.index, ic); err != nil {
					return err
				}
			}
		}

		// Report exported code if this is a defined, global/weak symbol in dynsym
		// These are symbols that can be used by other executables or libraries
		if needExports && sym.Shndx != SHN_UNDEF && name != "" {
			// Only report global or weak symbols (not local)
			if binding == STB_GLOBAL || binding == STB_WEAK {
				// Only report functions and objects
				if symType == STT_FUNC || symType == STT_OBJECT {
					ec := epc.ExportedCode{
						Name:    name,
						Offset:  uint64(offset),
						Addr:    sym.Value,
						Size:    sym.Size,
						Type:    symType,
						Binding: binding,
					}
					if err := p.cfg.OnExportedCodeFn(p.exeID, p.index, ec); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

// loadOrReadStrtab32 loads a string table by section index for 32-bit ELF.
func (p *Parser) loadOrReadStrtab32(linkIdx uint32) ([]byte, error) {
	if strtab, ok := p.strtabs[linkIdx]; ok {
		return strtab, nil
	}
	// String table should have been loaded in first pass
	return nil, nil
}

// loadOrReadStrtab64 loads a string table by section index for 64-bit ELF.
func (p *Parser) loadOrReadStrtab64(linkIdx uint32) ([]byte, error) {
	if strtab, ok := p.strtabs[linkIdx]; ok {
		return strtab, nil
	}
	// String table should have been loaded in first pass
	return nil, nil
}
