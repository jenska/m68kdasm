# m68kdasm

A disassembler for the Motorola 68000 CPU, written in Go.

This library translates raw m68k machine code (big-endian) into readable assembly mnemonics. It is designed for clean integration into emulators, debuggers, and analysis tools.

## Features

* **Fast opcode dispatch** using a hierarchical jump-table pattern (top-nibble switch for 4K-aligned regions).
* **Comprehensive instruction support** including branches, arithmetic, logical, bit operations, shifts, BCD, and control flow.
* **Addressing mode handling** for all 68000 modes: direct registers, indirect, pre/post-increment/decrement, displacement, index, PC-relative, and immediate.
* **Clean, readable decoder architecture** with named constants for all opcode patterns and bit masks.
* **Round-trip testing** - assembly → machine code → disassembly round-trips validate syntax fidelity.
* **Minimal public API** – just `Decode()` for single instructions and `DisassembleRange()` for byte sequences.
* **ELF file support** – `OpenELF()` to read and disassemble 68000 ELF binaries, with section selection.

## Architecture

The decoder uses a two-level dispatch mechanism:

1. **Top-level jump table**: Dispatches on the high nibble (bits 12-15) of the opcode to partition the 64K instruction space into 4K regions.
2. **Region decoders**: Within each region, specific patterns are matched using bit masks to identify exact instructions.

All opcode patterns and masks are defined as named constants at the top of `internal/decoders/types.go`, making the lookup table self-documenting and maintainable.

## Building & Testing

```bash
make build    # Compile the library
make test     # Run all unit tests (requires external m68kasm assembler)
make fmt      # Format code
```

Tests use an assembler round-trip pattern: source → machine code → disassembly → verify syntax matches, and include decoder-dispatch parity checks so the fast path stays aligned with the canonical opcode table.

## Usage

```go
package main

import (
 "fmt"
 "log"

 "github.com/jenska/m68kdasm"
)

func main() {
 code := []byte{0x4E, 0x71, 0x4E, 0x75} // NOP, RTS
 startAddr := uint32(0x1000)

 instrs, err := m68kdasm.DisassembleRange(code, startAddr)
 if err != nil {
  log.Fatal(err)
 }

 for _, i := range instrs {
  fmt.Println(i.String())
 }
}
```

## ELF Disassembly

Disassemble sections from a Motorola 68000 ELF binary:

```go
package main

import (
 "fmt"
 "log"

 "github.com/jenska/m68kdasm"
)

func main() {
 // Open ELF file (must be EM_68K architecture)
 elf, err := m68kdasm.OpenELF("program.elf")
 if err != nil {
  log.Fatal(err)
 }
 defer elf.Close()

 // List all loadable sections
 sections := elf.ListSections()
 for _, sec := range sections {
  fmt.Printf("%s: %#x  (size %d, %s)\n",
   sec.Name, sec.Addr, sec.Size,
   map[bool]string{true: "executable", false: ""}[sec.IsExec])
 }

 // Disassemble .text section
 instrs, err := elf.DisassembleSection(".text")
 if err != nil {
  log.Fatal(err)
 }

 for _, i := range instrs {
  fmt.Println(i.String())
 }

 // Or disassemble all executable sections at once
 allExec, err := elf.DisassembleAllExecutableSections()
 if err != nil {
  log.Fatal(err)
 }
 for sectionName, instrs := range allExec {
  fmt.Printf("\n###### Section %s ######\n", sectionName)
  for _, i := range instrs {
   fmt.Println(i.String())
  }
 }
}
```

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
