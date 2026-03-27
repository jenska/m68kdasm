# m68kdasm

A Go disassembler for the Motorola 68000 CPU.

`m68kdasm` translates big-endian m68k machine code into readable assembly and structured decode metadata. It is intended for emulators, debuggers, trace tools, and binary-analysis workflows that need more than just formatted text.

## Features

- Fast opcode dispatch using a hierarchical jump table.
- Broad 68000 instruction coverage including branches, arithmetic, logic, shifts, BCD, and control flow.
- Full 68000 addressing-mode decoding, including PC-relative and immediate forms.
- Exact decoded instruction length via `Instruction.Size`.
- Decoded extension words via `Instruction.ExtensionWords`.
- Structured metadata for mnemonic, operands, branch targets, immediates, and effective-address kinds.
- Slice, `io.ReaderAt`, and callback-based decode entry points.
- Precise partial-decode errors that report missing-byte counts.
- Optional symbol formatting hooks for resolved addresses.
- ELF helpers for disassembling 68000 ELF binaries.

## Install

```bash
go get github.com/jenska/m68kdasm
```

## API Overview

Single-instruction decode:

- `Decode(data []byte, address uint32)`
- `DecodeWithOptions(data []byte, address uint32, opts DecodeOptions)`
- `DecodeReaderAt(reader io.ReaderAt, address uint32)`
- `DecodeReaderAtWithOptions(reader io.ReaderAt, address uint32, opts DecodeOptions)`
- `DecodeFunc(read ReadFunc, address uint32)`
- `DecodeFuncWithOptions(read ReadFunc, address uint32, opts DecodeOptions)`

Sequential decode:

- `DisassembleRange(data []byte, startAddress uint32)`
- `DisassembleRangeWithOptions(data []byte, startAddress uint32, opts DecodeOptions)`

## Quick Start

```go
package main

import (
	"fmt"
	"log"

	"github.com/jenska/m68kdasm"
)

func main() {
	code := []byte{
		0x20, 0x7C, 0x00, 0x00, 0x21, 0x40, // MOVEA.L #$00002140, A0
		0x4E, 0x75, // RTS
	}

	instrs, err := m68kdasm.DisassembleRange(code, 0x1000)
	if err != nil {
		log.Fatal(err)
	}

	for _, inst := range instrs {
		fmt.Printf("%08X  %-24s size=%d ext=%v\n",
			inst.Address,
			inst.Assembly(),
			inst.Size,
			inst.ExtensionWords,
		)
	}
}
```

Example output:

```text
00001000  MOVEA.L #$00002140, A0  size=6 ext=[0 8512]
00001006  RTS                      size=2 ext=[]
```

## Structured Metadata

Each decoded instruction includes both rendered text and structured fields:

```go
inst, err := m68kdasm.Decode([]byte{
	0x20, 0x7C, 0x00, 0x00, 0x21, 0x40, // MOVEA.L #$00002140, A0
}, 0x2000)
if err != nil {
	log.Fatal(err)
}

fmt.Println(inst.Mnemonic)              // MOVEA.L
fmt.Println(inst.Operands)              // #$00002140, A0
fmt.Println(inst.Size)                  // 6
fmt.Println(inst.ExtensionWords)        // [0 8512]
fmt.Println(inst.Metadata.MnemonicBase) // MOVEA
fmt.Println(inst.Metadata.SizeSuffix)   // L

src := inst.Metadata.Operands[0]
fmt.Println(src.Kind)                           // effective_address
fmt.Println(src.EffectiveAddress.Kind)          // immediate
fmt.Println(src.EffectiveAddress.Immediate.Value) // 8512

dst := inst.Metadata.Operands[1]
fmt.Println(dst.Kind)               // register
fmt.Println(dst.Register.Kind)      // address
fmt.Println(dst.Register.Number)    // 0
```

Useful metadata fields:

- `Instruction.Size`: exact decoded byte length.
- `Instruction.Bytes`: exact bytes consumed by the instruction.
- `Instruction.ExtensionWords`: decoded words after the opcode word.
- `Instruction.Metadata.BranchTarget`: resolved branch target when applicable.
- `Instruction.Metadata.ImmediateValues`: immediate operands collected in structured form.
- `Instruction.Metadata.Operands`: per-operand metadata, including effective-address details.

## Streaming Decode

If your emulator or debugger fetches bytes from a bus instead of a prebuilt slice, you can decode directly from an `io.ReaderAt` or callback.

Using `io.ReaderAt`:

```go
reader := bytes.NewReader([]byte{
	0x4E, 0xB9, 0x00, 0x00, 0x12, 0x34, // JSR $00001234
})

inst, err := m68kdasm.DecodeReaderAt(reader, 0x1000)
if err != nil {
	log.Fatal(err)
}

fmt.Println(inst.Assembly()) // JSR $00001234
```

Using a callback:

```go
mem := []byte{0x67, 0x08} // BEQ.S $000A

inst, err := m68kdasm.DecodeFunc(func(address uint32, p []byte) (int, error) {
	if int(address) >= len(mem) {
		return 0, io.EOF
	}
	n := copy(p, mem[address:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}, 0)
if err != nil {
	log.Fatal(err)
}

fmt.Println(inst.Assembly()) // BEQ.S $000A
```

## Symbolized Output

You can keep raw metadata while rendering resolved addresses as symbols:

```go
inst, err := m68kdasm.DecodeWithOptions([]byte{
	0x4E, 0xB9, 0x00, 0x00, 0x12, 0x34, // JSR $00001234
}, 0, m68kdasm.DecodeOptions{
	Symbolizer: m68kdasm.SymbolizeFunc(func(address uint32) (string, bool) {
		if address == 0x1234 {
			return "_bios_init", true
		}
		return "", false
	}),
})
if err != nil {
	log.Fatal(err)
}

fmt.Println(inst.Assembly())                 // JSR _bios_init
fmt.Println(inst.Metadata.Operands[0].Text)  // $00001234
```

This is useful for:

- symbolized trace logs
- debugger disassembly views
- breakpoint or stop-condition logic based on structured targets

## Partial Decode Errors

Truncated fetches return `*PartialDecodeError` with the missing-byte count and context:

```go
_, err := m68kdasm.Decode([]byte{0x4E, 0x72}, 0x2000) // STOP without immediate word
if err != nil {
	var partial *m68kdasm.PartialDecodeError
	if errors.As(err, &partial) {
		fmt.Println(partial.Missing) // 2
		fmt.Println(partial.Context) // STOP immediate
	}
}
```

This is especially handy for trace logs and emulator diagnostics where you want to distinguish a broken fetch stream from an unknown opcode.

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
	elf, err := m68kdasm.OpenELF("program.elf")
	if err != nil {
		log.Fatal(err)
	}
	defer elf.Close()

	instrs, err := elf.DisassembleSection(".text")
	if err != nil {
		log.Fatal(err)
	}

	for _, inst := range instrs {
		fmt.Println(inst.String())
	}
}
```

## Building And Testing

```bash
make build
make test
make fmt
```

Tests include assembler round trips, decoder dispatch parity, streaming decode coverage, metadata checks, symbolized rendering, and partial-error behavior.

## Architecture

The decoder uses a two-level dispatch mechanism:

1. A top-level jump table partitions the opcode space by high nibble.
2. Per-region pattern tables apply masks in precedence order to select the final decoder.

Opcode masks and values live in `internal/decoders/types.go`, which keeps the decoder table explicit and easy to extend.

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
