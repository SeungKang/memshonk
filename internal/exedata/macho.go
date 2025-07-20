package exedata

import (
	"debug/macho"
	"fmt"
	"io"
	"runtime"
)

var _ Exe = (*Macho)(nil)

type readerAtSeeker interface {
	io.ReaderAt
	io.Seeker
}

func ParseMachoCurrentArch(readerAt readerAtSeeker, options ParserOptions) (Macho, error) {
	fat, err := macho.NewFatFile(readerAt)
	switch {
	case err == nil:
		arch := runtime.GOARCH

		machoCpu, err := goArchToMachoCpu(arch)
		if err != nil {
			return Macho{}, fmt.Errorf("failed to convert arch string to macho cpu type - %w", err)
		}

		for _, arch := range fat.Arches {
			if arch.Cpu == machoCpu {
				return ParseMachoObj(arch.File, options)
			}
		}

		return Macho{}, fmt.Errorf("failed to find matching macho object in fat file for architecture %q", arch)
	default:
		_, err := readerAt.Seek(0, 0)
		if err != nil {
			return Macho{}, fmt.Errorf("failed to reset macho file seeker to 0 offset - %w", err)
		}

		machoFile, err := macho.NewFile(readerAt)
		if err != nil {
			return Macho{}, err
		}

		return ParseMachoObj(machoFile, options)
	}
}

func goArchToMachoCpu(goArch string) (macho.Cpu, error) {
	// {uint32(Cpu386), "Cpu386"},
	// {uint32(CpuAmd64), "CpuAmd64"},
	// {uint32(CpuArm), "CpuArm"},
	// {uint32(CpuArm64), "CpuArm64"},
	// {uint32(CpuPpc), "CpuPpc"},
	// {uint32(CpuPpc64), "CpuPpc64"},

	switch goArch {
	case "386":
		return macho.Cpu386, nil
	case "amd64":
		return macho.CpuAmd64, nil
	case "arm":
		return macho.CpuArm, nil
	case "arm64":
		return macho.CpuArm64, nil
	case "ppc":
		return macho.CpuPpc, nil
	case "ppc64":
		return macho.CpuPpc64, nil
	default:
		return 0, fmt.Errorf("unknown cpu arch: %q", goArch)
	}
}

func ParseMachoSlim(readerAt io.ReaderAt, options ParserOptions) (Macho, error) {
	machoFile, err := macho.NewFile(readerAt)
	if err != nil {
		return Macho{}, err
	}

	return ParseMachoObj(machoFile, options)
}

func ParseMachoObj(machoFile *macho.File, options ParserOptions) (Macho, error) {
	syms, err := parseMachoSymbols(machoFile)
	if err != nil {
		return Macho{}, fmt.Errorf("failed to parse macho symbols - %w", err)
	}

	return Macho{
		machoFile: machoFile,
		syms:      syms,
	}, nil
}

type Macho struct {
	machoFile *macho.File
	syms      []Symbol
}

func (o Macho) Symbols() []Symbol {
	return o.syms
}

func parseMachoSymbols(machoFile *macho.File) ([]Symbol, error) {
	syms := make([]Symbol, len(machoFile.Symtab.Syms))

	for i, s := range machoFile.Symtab.Syms {
		syms[i] = Symbol{
			Name:     s.Name,
			Location: s.Value,
			Size:     0, // TODO
		}
	}

	return syms, nil
}
