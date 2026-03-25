package m68kdasm

import (
	"debug/elf"
	"fmt"
)

// ELFDisassembler holds an ELF file and provides disassembly functions
type ELFDisassembler struct {
	file *elf.File
}

// OpenELF opens an ELF file and returns an ELFDisassembler
func OpenELF(filePath string) (*ELFDisassembler, error) {
	f, err := elf.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open ELF file: %w", err)
	}

	// Verify it's a Motorola 68000 binary
	if f.Machine != elf.EM_68K {
		if closeErr := f.Close(); closeErr != nil {
			return nil, fmt.Errorf("ELF file is for architecture %v, not Motorola 68000 (EM_68K): close failed: %w", f.Machine, closeErr)
		}
		return nil, fmt.Errorf("ELF file is for architecture %v, not Motorola 68000 (EM_68K)", f.Machine)
	}

	return &ELFDisassembler{file: f}, nil
}

// Close closes the underlying ELF file
func (ed *ELFDisassembler) Close() error {
	if ed.file != nil {
		return ed.file.Close()
	}
	return nil
}

// DisassembleSection disassembles a named ELF section by name (e.g., ".text")
// Returns instructions with addresses from the section's VA (virtual address)
func (ed *ELFDisassembler) DisassembleSection(sectionName string) ([]Instruction, error) {
	section := ed.file.Section(sectionName)
	if section == nil {
		return nil, fmt.Errorf("section %q not found in ELF file", sectionName)
	}
	return ed.disassembleSection(section)
}

// ListSections returns information about all loadable sections in the ELF file.
func (ed *ELFDisassembler) ListSections() []SectionInfo {
	var sections []SectionInfo

	for _, section := range ed.file.Sections {
		// Only include sections with ALLOC flag (are loaded into memory)
		if section.Flags&elf.SHF_ALLOC == 0 {
			continue
		}

		sections = append(sections, SectionInfo{
			Name:   section.Name,
			Addr:   section.Addr,
			Offset: section.Offset,
			Size:   section.Size,
			Flags:  uint32(section.Flags),
			IsExec: section.Flags&elf.SHF_EXECINSTR != 0,
		})
	}

	return sections
}

// SectionInfo describes a section in an ELF file
type SectionInfo struct {
	Name   string // Section name (e.g., ".text")
	Addr   uint64 // Virtual address
	Offset uint64 // File offset
	Size   uint64 // Size in bytes
	Flags  uint32 // Raw section flags
	IsExec bool   // True if executable (SHF_EXECINSTR)
}

// DisassembleAllExecutableSections disassembles all executable sections (SHF_EXECINSTR flag)
// Returns a map of section name → instructions
func (ed *ELFDisassembler) DisassembleAllExecutableSections() (map[string][]Instruction, error) {
	result := make(map[string][]Instruction)

	for _, section := range ed.file.Sections {
		// Only process executable sections
		if section.Flags&elf.SHF_EXECINSTR == 0 {
			continue
		}

		instrs, err := ed.disassembleSection(section)
		if err != nil {
			return nil, fmt.Errorf("failed to disassemble section %q: %w", section.Name, err)
		}

		result[section.Name] = instrs
	}

	return result, nil
}

func (ed *ELFDisassembler) disassembleSection(section *elf.Section) ([]Instruction, error) {
	data, err := section.Data()
	if err != nil {
		return nil, fmt.Errorf("failed to read section %q: %w", section.Name, err)
	}

	// Use the section's virtual address as the starting address.
	return DisassembleRange(data, uint32(section.Addr))
}
