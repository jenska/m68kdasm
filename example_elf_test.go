package m68kdasm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jenska/m68kasm"
)

func TestELFDisassemblyRoundTrip(t *testing.T) {
	// Assembly source code
	source := `
		ORG $1000
		.text
		start:
			NOP
			MOVE.W #$1234, D0
			MOVE.W D0, D1
			ADD.W #5, D1
			CMP.W #10, D1
			BNE.S skip
			CLR.W D2
		skip:
			RTS
	`

	// Create temporary directory for test files
	tmpDir := t.TempDir()
	elfPath := filepath.Join(tmpDir, "test.elf")

	// Assemble source code to ELF binary format
	elfData, err := m68kasm.AssembleStringELF(source)
	if err != nil {
		t.Fatalf("Failed to assemble to ELF: %v", err)
	}

	// Write ELF data to file
	err = os.WriteFile(elfPath, elfData, 0644)
	if err != nil {
		t.Fatalf("Failed to write ELF file: %v", err)
	}
	defer os.Remove(elfPath)

	// Open and disassemble the ELF file
	elf, err := OpenELF(elfPath)
	if err != nil {
		t.Fatalf("Failed to open ELF file: %v", err)
	}
	defer elf.Close()

	// List sections
	sections := elf.ListSections()
	if len(sections) == 0 {
		t.Fatalf("No sections found in ELF file")
	}

	// Find and disassemble .text section
	var textSectionFound bool
	for _, sec := range sections {
		if sec.Name == ".text" {
			textSectionFound = true
			break
		}
	}

	if !textSectionFound {
		t.Fatalf("No .text section found in ELF file")
	}

	instrs, err := elf.DisassembleSection(".text")
	if err != nil {
		t.Fatalf("Failed to disassemble .text section: %v", err)
	}

	if len(instrs) == 0 {
		t.Fatalf("No instructions disassembled from .text section")
	}

	gotAssembly := make([]string, len(instrs))
	for i, instr := range instrs {
		gotAssembly[i] = instr.Assembly()
	}

	expectedAssembly := []string{
		"NOP",
		"MOVE.W #$1234, D0",
		"MOVE.W D0, D1",
		"ADD.W #5, D1",
		"CMP.W #10, D1",
		"BNE.S $1014",
		"CLR.W D2",
		"RTS",
	}

	if len(gotAssembly) != len(expectedAssembly) {
		t.Fatalf("Expected %d instructions, got %d:\n%s", len(expectedAssembly), len(gotAssembly), strings.Join(gotAssembly, "\n"))
	}

	for i, want := range expectedAssembly {
		if gotAssembly[i] != want {
			t.Fatalf("Instruction %d mismatch: want %q, got %q", i, want, gotAssembly[i])
		}
	}

	expectedAddresses := []uint32{0x1000, 0x1002, 0x1006, 0x1008, 0x100C, 0x1010, 0x1012, 0x1014}
	for i, want := range expectedAddresses {
		if instrs[i].Address != want {
			t.Fatalf("Instruction %d address mismatch: want %08X, got %08X", i, want, instrs[i].Address)
		}
	}
}

func TestELFDisassemblesAllExecutableSections(t *testing.T) {
	source := `
		ORG $2000
		.text
		main:
			MOVE.W #100, D0
			MOVE.W #50, D1
			ADD.W D1, D0
			RTS
	`

	elfData, err := m68kasm.AssembleStringELF(source)
	if err != nil {
		t.Fatalf("Failed to assemble ELF: %v", err)
	}

	tmpDir := t.TempDir()
	elfPath := filepath.Join(tmpDir, "example.elf")

	err = os.WriteFile(elfPath, elfData, 0644)
	if err != nil {
		t.Fatalf("Failed to write ELF file: %v", err)
	}

	elf, err := OpenELF(elfPath)
	if err != nil {
		t.Fatalf("Failed to open ELF: %v", err)
	}
	defer elf.Close()

	sections := elf.ListSections()
	if len(sections) == 0 {
		t.Fatal("Expected at least one loadable section")
	}

	var textFound bool
	for _, sec := range sections {
		if sec.Name == ".text" {
			textFound = true
			if !sec.IsExec {
				t.Fatalf(".text section should be executable: %+v", sec)
			}
			if sec.Addr != 0x2000 {
				t.Fatalf(".text section address mismatch: want %#x, got %#x", 0x2000, sec.Addr)
			}
		}
	}
	if !textFound {
		t.Fatal("Expected .text section in ELF")
	}

	allExec, err := elf.DisassembleAllExecutableSections()
	if err != nil {
		t.Fatalf("Failed to disassemble executable sections: %v", err)
	}

	instrs, ok := allExec[".text"]
	if !ok {
		t.Fatalf("Expected .text in executable sections, got keys: %v", mapsKeys(allExec))
	}

	gotAssembly := make([]string, len(instrs))
	for i, instr := range instrs {
		gotAssembly[i] = instr.Assembly()
	}

	expectedAssembly := []string{
		"MOVE.W #$0064, D0",
		"MOVE.W #50, D1",
		"ADD.W D1, D0",
		"RTS",
	}
	if len(gotAssembly) != len(expectedAssembly) {
		t.Fatalf("Expected %d instructions in .text, got %d:\n%s", len(expectedAssembly), len(gotAssembly), strings.Join(gotAssembly, "\n"))
	}
	for i, want := range expectedAssembly {
		if gotAssembly[i] != want {
			t.Fatalf("Executable section instruction %d mismatch: want %q, got %q", i, want, gotAssembly[i])
		}
	}
}

func mapsKeys(m map[string][]Instruction) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
