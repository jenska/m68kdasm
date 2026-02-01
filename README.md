# m68kdasm

A disassembler for the Motorola 68000 CPU, written in Go.

This project provides a library and interfaces to translate m68k machine code into readable assembly mnemonics. It is designed to be easily extensible and handles the 68000's big-endian architecture cleanly.

## Features

*   Disassembly of raw byte slices.
*   Support for addressing modes and operands.
*   Simple API for integration into emulators or tools.
*   Unit tests based on assembler round-trips and direct decoder checks.
*   Coverage for core 68000 instructions including branches, logical/arithmetic ops, bit operations, and traps.

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

## Lizenz

Dieses Projekt ist unter der MIT-Lizenz lizenziert. Siehe [LICENSE](LICENSE) für Details.
