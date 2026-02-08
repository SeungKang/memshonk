// Example program demonstrating how to use the exedata library to parse executables.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"exedata"
)

func main() {
	timeout := flag.Duration("timeout", 30*time.Second, "parsing timeout")
	showSegments := flag.Bool("segments", false, "show segments (program headers)")
	showSections := flag.Bool("sections", false, "show sections")
	showSymbols := flag.Bool("symbols", false, "show symbols")
	showFunctions := flag.Bool("functions", false, "show functions")
	showStrings := flag.Bool("strings", false, "show strings from string tables")
	showExportedCode := flag.Bool("exported-code", false, "show exported functions/symbols")
	showImportedCode := flag.Bool("imported-code", false, "show imported functions/symbols")
	showImportedLibs := flag.Bool("imported-libs", false, "show imported libraries")
	showRelocs := flag.Bool("relocs", false, "show relocations")
	showAll := flag.Bool("all", false, "show all information")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <executable>\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	filename := flag.Arg(0)

	f, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	cfg := &exedata.ParserConfig{
		Src: f,

		// Always show basic info
		OnInfoFn: func(exeID string, index uint, info exedata.Info) error {
			fmt.Printf("=== Executable Info ===\n")
			fmt.Printf("  Format:      %s\n", info.Format)
			fmt.Printf("  Class:       %d-bit\n", info.Class)
			fmt.Printf("  Endian:      %s\n", info.Endian)
			fmt.Printf("  Type:        %s\n", info.Type)
			fmt.Printf("  Machine:     %s\n", info.Machine)
			fmt.Printf("  OS/ABI:      %s\n", info.OSABI)
			fmt.Printf("  ABI Version: %d\n", info.ABIVersion)
			fmt.Printf("  Entry Point: 0x%x\n", info.EntryPoint)
			fmt.Printf("  Flags:       0x%x\n", info.Flags)
			fmt.Println()
			return nil
		},
	}

	if *showAll || *showSegments {
		cfg.OnSegmentFn = func(exeID string, index uint, seg exedata.Segment) error {
			fmt.Printf("Segment: type=0x%x flags=0x%x offset=0x%x vaddr=0x%x filesz=%d memsz=%d align=%d\n",
				seg.Type, seg.Flags, seg.Offset, seg.VAddr, seg.FileSize, seg.MemSize, seg.Align)
			return nil
		}
	}

	if *showAll || *showSections {
		cfg.OnSectionFn = func(exeID string, index uint, sec exedata.Section) error {
			flags := ""
			if sec.IsCode {
				flags += "X"
			}
			if sec.IsWritable {
				flags += "W"
			}
			if sec.IsData && !sec.IsCode {
				flags += "A"
			}
			fmt.Printf("Section: %-20s type=0x%02x flags=%-3s addr=0x%08x offset=0x%06x size=%d\n",
				sec.Name, sec.Type, flags, sec.Addr, sec.Offset, sec.Size)
			return nil
		}
	}

	if *showAll || *showSymbols {
		cfg.OnSymbolFn = func(exeID string, index uint, sym exedata.Symbol) error {
			bindStr := symbolBindingString(sym.Binding)
			typeStr := symbolTypeString(sym.Type)
			fmt.Printf("Symbol: %-40s bind=%-6s type=%-7s addr=0x%08x size=%d\n",
				truncate(sym.Name, 40), bindStr, typeStr, sym.Addr, sym.Size)
			return nil
		}
	}

	if *showAll || *showFunctions {
		cfg.OnFunctionFn = func(exeID string, index uint, fn exedata.Function) error {
			fmt.Printf("Function: %-50s addr=0x%08x size=%d\n",
				truncate(fn.Name, 50), fn.Addr, fn.Size)
			return nil
		}
	}

	if *showAll || *showStrings {
		cfg.OnStringFn = func(exeID string, index uint, s exedata.String) error {
			// Only show non-empty strings
			if len(s.Value) > 0 {
				fmt.Printf("String: [%s] offset=0x%x %q\n",
					s.Source, s.Offset, truncate(s.Value, 60))
			}
			return nil
		}
	}

	if *showAll || *showExportedCode {
		cfg.OnExportedCodeFn = func(exeID string, index uint, ec exedata.ExportedCode) error {
			typeStr := symbolTypeString(ec.Type)
			bindStr := symbolBindingString(ec.Binding)
			fwdStr := ""
			if ec.Forwarder != "" {
				fwdStr = fmt.Sprintf(" -> %s", ec.Forwarder)
			}
			fmt.Printf("ExportedCode: %-40s type=%-7s bind=%-6s addr=0x%08x%s\n",
				truncate(ec.Name, 40), typeStr, bindStr, ec.Addr, fwdStr)
			return nil
		}
	}

	if *showAll || *showImportedCode {
		cfg.OnImportedCodeFn = func(exeID string, index uint, ic exedata.ImportedCode) error {
			typeStr := symbolTypeString(ic.Type)
			libStr := ""
			if ic.Library != "" {
				libStr = fmt.Sprintf(" from %s", ic.Library)
			}
			fmt.Printf("ImportedCode: %-40s type=%-7s%s\n",
				truncate(ic.Name, 40), typeStr, libStr)
			return nil
		}
	}

	if *showAll || *showImportedLibs {
		cfg.OnImportedLibraryFn = func(exeID string, index uint, lib exedata.ImportedLibrary) error {
			fmt.Printf("ImportedLibrary: %s (offset=0x%x)\n", lib.Name, lib.Offset)
			return nil
		}
	}

	if *showAll || *showRelocs {
		cfg.OnRelocFn = func(exeID string, index uint, rel exedata.Reloc) error {
			fmt.Printf("Reloc: offset=0x%08x type=%d sym_idx=%d addend=%d\n",
				rel.Offset, rel.Type, rel.SymIndex, rel.Addend)
			return nil
		}
	}

	if err := exedata.Parse(ctx, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing executable: %v\n", err)
		os.Exit(1)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func symbolBindingString(b uint8) string {
	switch b {
	case 0:
		return "LOCAL"
	case 1:
		return "GLOBAL"
	case 2:
		return "WEAK"
	default:
		return fmt.Sprintf("(%d)", b)
	}
}

func symbolTypeString(t uint8) string {
	switch t {
	case 0:
		return "NOTYPE"
	case 1:
		return "OBJECT"
	case 2:
		return "FUNC"
	case 3:
		return "SECTION"
	case 4:
		return "FILE"
	case 5:
		return "COMMON"
	case 6:
		return "TLS"
	default:
		return fmt.Sprintf("(%d)", t)
	}
}
