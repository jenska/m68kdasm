# AI Coding Agent Instructions for m68kdasm

## Project Overview
m68kdasm is a **Motorola 68000 disassembler library written in Go**. It translates raw machine code bytes (big-endian) into readable assembly mnemonics. The public API is minimal but powerful: `Decode()` for single instructions and `DisassembleRange()` for byte sequences. Heavy lifting is delegated to internal decoder functions organized by instruction category.

## Architecture Pattern: Decoder Jump Table

The core design uses a **priority-ordered opcode pattern matching table** (`OpcodeTable` in [internal/decoders/types.go](internal/decoders/types.go)):

1. **Highest priority**: Exact opcode matches (e.g., `NOP=0x4E71`, `RTS=0x4E75`)
2. **Medium priority**: Multi-byte patterns with bit masks (e.g., `MOVEM=0xFB80`, `BRA=0xF100`)
3. **Lowest priority**: Generic patterns (e.g., `MOVE.x=0xF000` matches `0x1000`, `0x2000`, `0x3000`)

**Pattern ordering matters**: MUL/DIV, BIT operations, and BCD instructions must appear *before* generic patterns to avoid false matches.

Each entry maps a `(Mask, Value)` pair to a decoder function with signature:
```go
func decode*(data []byte, opcode uint16, inst *Instruction) error
```

The decoder extracts instruction metadata (mnemonic, operands, size) into the `Instruction` struct passed by reference.

## Data Flow

1. Public API: `Decode()` reads 2+ bytes as big-endian opcode → searches `OpcodeTable` sequentially
2. Matched decoder invoked with raw data slice, opcode word, and instruction struct
3. Decoder **advances through data** extracting operands via `decodeAddressingMode()`—must track extra words (extension data)
4. Total instruction size computed: opcode (always 2) + extension words (`×2` bytes)
5. Result: mnemonic + operands + byte slice + size

Example: `MOVE.W D0, (5,A2)` = opcode 2 + displacement 2 + index operand 2 = **6 bytes total**

## Key Patterns to Follow

### 1. Addressing Mode Decoding
[internal/decoders/addressing.go](internal/decoders/addressing.go) provides `decodeAddressingMode()` that:
- Takes byte data, **mode** (bits 3-5 or 6-8), **register** (bits 0-2 or 9-11)
- Returns: `(formatted_string, extra_words_consumed, error)`
- **Critical**: Caller must advance data offset by `extra_words * 2` bytes before next decode
- Examples: mode=0→`D3`, mode=2→`(A5)`, mode=7/0→`$8000` (absolute short, 1 extra word)

### 2. Register Field Extraction
68000 uses split register fields depending on instruction class:
- **Generic instructions**: destination mode/reg at bits 6-11, source mode/reg at bits 0-5
- **Special instructions** (MOVEM, shifts): field positions vary—always check the specific decoder

### 3. Immediate Values
Helper `formatImmediate(uint32, size)` chooses decimal vs hex intelligently:
- Values < 100 → decimal: `#5`
- Otherwise → hex with size-based width: `#$FF04`
- For MOVEQ (signed 8-bit): use `formatImmediateForMOVEQ()`

### 4. Size Encoding
Most instructions encode size in bits 12-13:
- `01` = Byte (.B)
- `11` = Word (.W)
- `10` = Long (.L)
- Pattern varies; always verify against hardware docs before decoder

### 5. Instruction Size Calculation
**Every decoder must set `inst.Size`**—the *total* byte count:
- Minimum 2 (opcode word)
- Add 2 for each extension word (displacement, immediate, absolute address, index)
- Must match actual data consumed, or disassembly loops will fail

## Testing Approach: Round-Trip Verification

Tests in [disasm_test.go](disasm_test.go) use an **assembler round-trip pattern**:
1. Source string (e.g., `"MOVE.W D0, D1"`) → machine code via `m68kasm.AssembleString()`
2. Machine code → `DisassembleRange()` → `Instruction.Assembly()`
3. Verify reconstructed assembly **exactly matches** source (byte-for-byte, not semantic)

This ensures syntax fidelity. New decoders **must be tested this way**—add test cases to `TestDisassembleRoundTrip`. Failing test = mnemonic or operand formatting mismatch, or size calculation wrong.

## Integration: m68kasm Dependency

[go.mod](go.mod) depends on `github.com/jenska/m68kasm v1.1.4` (assembler). In tests, use `m68kasm.AssembleString()` to validate disassembler output. Decoder syntax must align with that assembler's output format.

## Build & Test Workflow

```bash
make test    # Run all tests (must pass before commit)
make build   # Compile library
make fmt     # Run go fmt
```

Ensure all tests pass. Unknown opcodes fall back to `DC.W $XXXX` (data word constant).

## Adding New Instruction Support

1. **Add opcode pattern** to `OpcodeTable` in [internal/decoders/types.go](internal/decoders/types.go) at correct priority position
2. **Implement decoder function** in appropriate file:
   - Arithmetic: [internal/decoders/arithmetic.go](internal/decoders/arithmetic.go)
   - Move/Control: [internal/decoders/move.go](internal/decoders/move.go)
   - Shift/Rotate: [internal/decoders/shift.go](internal/decoders/shift.go)
   - Bit ops: [internal/decoders/bit.go](internal/decoders/bit.go)
   - Other: [internal/decoders/special.go](internal/decoders/special.go)
3. **Compute correct size**: account for all extension words
4. **Add test case** to `TestDisassembleRoundTrip` with expected mnemonic
5. **Run `make test`** and iterate until it passes

## Language Notes

Codebase uses German comments and error messages (e.g., "nicht genügend Daten"). Maintain consistency; new comments should follow this convention.
