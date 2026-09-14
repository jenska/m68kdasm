# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.3.0] - 2026-09-14

### Added
- **FPU decoding.** `DecodeOptions.FPU` opts in to the 68881/68882 (or the 68040/68060's built-in) FPU instruction set: `FMOVE`/`FADD`/`FSUB`/`FMUL`/`FDIV`/`FCMP`/`FABS`/`FNEG`/`FSQRT`/`FTST`/`FNOP` in all 7 data formats (including packed BCD with k-factor), `FMOVEM` (both the `FPn` and `FPCR`/`FPSR`/`FPIAR` register-list forms), the full transcendental and math-extension function sets, `FMOVECR`, `FSINCOS`, the `FBcc`/`FDBcc`/`FScc`/`FTRAPcc` condition family, and `FSAVE`/`FRESTORE`.
- **PMMU decoding.** `DecodeOptions.MMU` opts in to the 68851 (or the 68030's built-in) PMMU instruction set: `PMOVE`'s full register set (including `BAD0`-`BAD7`/`BAC0`-`BAC7`), `PMOVEFD`, `PFLUSHA`/`PFLUSH`/`PFLUSHS`/`PFLUSHR`, `PLOADR`/`PLOADW`, `PTESTR`/`PTESTW`, `PSAVE`/`PRESTORE`, the `PBcc`/`PDBcc`/`PScc`/`PTRAPcc` condition family, and the 68040's own simplified single-word PMMU forms.
- **Auto-generated disassembly labels.** `DecodeOptions.Labels`/`LabelOptions` opt in to synthetic label generation (e.g. `l00001010`) for branch/call targets that fall within a disassembled range and land on a decoded instruction — created by `Bcc`/`BSR`/`DBcc`/`FBcc`/`FDBcc`/`PBcc`/`PDBcc` and by `JSR`/`JMP`/`PEA`/`LEA` targeting an absolute or PC-relative address, and rendered on every operand that references the address once created, not just the one that created it. A caller-supplied `Symbolizer` always takes precedence over a synthetic label. `Instruction.Label` and `Instruction.String()`'s label-line formatting expose the result.
- `ELFDisassembler.DisassembleSectionWithOptions` and `DisassembleAllExecutableSectionsWithOptions`. Previously ELF-sourced disassembly had no `DecodeOptions` parameter at all and always decoded as plain `M68000` with no `Symbolizer` — CPU/FPU/MMU selection and the new `Labels` option are now reachable from ELF input too.

### Fixed
- **`Bcc`/`BSR`.W/.L and `DBcc` branch-target math.** The target was computed relative to (opcode address + total instruction length) instead of (opcode address + 2) — the address of the extension word — off by 2 bytes for the `.W` form and 4 for `.L`. The 8-bit (`.S`) form was accidentally correct, since for that form the two formulas happen to coincide, which is why no prior test caught this.

### Changed
- Bumped the `github.com/jenska/m68kasm` dependency to v1.5.0, which added FPU/PMMU/full-CPU-tier assembly support — used throughout as the verification oracle for everything above, round-tripping real assembled bytes rather than trusting bit-layout derivations alone.

Not implemented (see [`docs/design-fpu-mmu.md`](docs/design-fpu-mmu.md) and [`docs/design-labels.md`](docs/design-labels.md) for rationale and the full delivery history): `PVALID`; per-CPU-tier gating for the wider PMMU surface (every PMMU pattern besides the 68040-specific forms currently decodes on any target CPU). `CAS2`, CPU32's `TBLS`/`TBLU` family, and 68040's `MOVE16`/`CINV`/`CPUSH` remain unimplemented from 1.2.0.

## [1.2.0] - 2026-09-12

### Added
- **Multi-CPU-variant support.** `DecodeOptions.CPU` selects the target 68k family member — `M68000` (default, unchanged behavior), `M68010`, `CPU32`, `M68020`, `M68030`, `M68040`, `M68060`. Opcode and addressing-mode availability is tracked per CPU via an explicit bitset rather than an ordinal comparison, since the family isn't a strict newer-implies-older chain (CPU32 branches off 68010 with its own extensions; 68040 drops `CALLM`/`RTM` that 68020/68030 have).
- **68010/CPU32**: `MOVEC`, `MOVES`, `RTD`, `BGND`.
- **68020+**: full extension-word addressing (memory indirect, scaled/suppressed index, 0/16/32-bit base and outer displacements), the `BFxxx` bitfield family (`BFTST`/`BFCHG`/`BFCLR`/`BFSET`/`BFEXTU`/`BFEXTS`/`BFFFO`/`BFINS`), `CAS`, `CHK2`/`CMP2`, 32×32 `MULU.L`/`MULS.L`/`DIVU.L`/`DIVS.L`, `PACK`/`UNPK`, `CALLM`/`RTM`, `TRAPcc`, `LINK.L`, `EXTB.L`, `CHK.L`. 68030/68040/68060 inherit all of this automatically (68040/68060 correctly exclude `CALLM`/`RTM`).
- **Completed 68000 baseline coverage**: `LINK`, `UNLK`, `EXT`, `CHK`, `EXG`, `RESET`, `RTE`, `RTR`, `ILLEGAL`, `NBCD`, `MOVEP`, `ADDQ`, `SUBQ`, `Scc`, `DBcc` were previously missing entirely. Every canonical 68000 mnemonic now has a decoder.

### Fixed
- **Silent mis-decodes**: `TAS`, `ADDX`, and `SUBX` opcodes previously fell through to `TST`, `ADD`, and `SUB` respectively (with a garbled operand for `ADDX`/`SUBX`), and `EXG` fell through to `AND`, due to opcode-space collisions the original dispatch table didn't account for.

### Changed
- Reorganized `internal/decoders/*.go`: split the overloaded `types.go` into `types.go` (data model), `cpu.go` (`CPU`/`cpuSet`), and `opcodetable.go` (the single opcode-dispatch table); folded several session-specific files back into the existing per-instruction-family files; deduplicated near-identical decoders behind small shared factories.

Not implemented (see `docs/design-cpu-variants.md` for rationale): `CAS2`, CPU32's `TBLS`/`TBLU` family, 68040's `MOVE16`/`CINV`/`CPUSH`, and FPU/PMMU coprocessor instructions.

## [1.1.0] - 2026-09-03

### Changed
- **Internal simplification**: the structured decode types (`Operand`, `Register`, `EffectiveAddress`, …) are now defined once in `internal/decoders` and re-exported from the public package via type aliases, removing ~150 lines of value-copying glue. No API or behavior change.
- Consolidated duplicated instruction decoders (immediate ALU ops, address-register ALU ops, unary EA ops, predecrement/postincrement operands) behind shared helpers. Non-test code shrinks from ~2270 to ~2000 lines.
- **Minimum Go version raised to 1.27.** Adopted the `new(expr)` builtin and `range`-over-int; `DisassembleRange` now preallocates its result slice.
- Removed a stray root `go.yml` that duplicated the real CI workflow.

## [1.0.3] - 2026-04-03

### Fixed
- **PC-relative symbolizer regression**: restored correct PC-relative target resolution.
- CI build fix.

## [1.0.2] - 2026-04-03

### Fixed
- **MOVEM predecrement register lists**: Fixed reversed register-mask decoding for register-to-memory predecrement forms. `MOVEM.L D0/A0, -(A7)` now disassembles correctly instead of decoding as `MOVEM.L D7/A7, -(A7)`.
- **PC-relative symbolization metadata**: PC-relative effective addresses now populate `ResolvedAddress`, allowing `DecodeOptions.Symbolizer` to resolve operands such as `JSR (disp,PC)` and `LEA (disp,PC), Ax`.
- **Absolute short address resolution**: Absolute short operands now sign-extend correctly in structured metadata, so `$FF80.W` resolves to `0xFFFFFF80` instead of `0x0000FF80`.

### Added
- Regression tests for MOVEM predecrement decoding, PC-relative symbolization, and absolute short address sign extension.

## [1.0.1] - 2026-03-28

### Fixed
- **MOVEM register list decoding**: Fixed incorrect register mask interpretation for Mem→Reg operations. Previously `4C DF 0C 04` decoded as `MOVEM.L (A7)+, A4-A5/D5` instead of the correct `MOVEM.L (A7)+, D2/A2-A3`.
- **SWAP instruction decoding**: Added proper SWAP instruction decoder to distinguish from PEA. Previously `48 40` decoded as `PEA D0` instead of `SWAP D0`.
- **Code cleanup**: Removed unused direction parameter from `formatRegisterList` function and added documentation for new constants.

### Added
- Comprehensive test cases for MOVEM, SWAP, and branch instructions to prevent regressions.

## [1.0.0] - 2026-03-XX

Initial release of m68kdasm, a Go disassembler for the Motorola 68000 CPU.

### Features
- Fast opcode dispatch using hierarchical jump table
- Broad 68000 instruction coverage
- Full addressing mode decoding
- Structured metadata output
- Multiple decode entry points (slice, io.ReaderAt, callback)
- ELF binary support
- Precise error reporting
