# Design: Multi-CPU-Variant Support (68010/68020/68030/68040/68060/CPU32)

## Status

**Implemented.** Steps 1-7 of the delivery sequence below are done: the `CPU`
plumbing, 68010/CPU32 additions, the 68020+ full extension word addressing
rework, and the 68020 opcode additions (inherited automatically by
68030/68040/68060 via the cpuSet tagging). Along the way, several *pre-existing*
plain-68000 gaps were also found and fixed, since some of them are direct
prerequisites for 68020 extensions (see "Baseline gaps found and fixed"
below). Step 8 (FPU/PMMU) remains future work, and a handful of individual
68020+/CPU32 instructions were deliberately deferred — see "Explicitly
deferred" below.

### Baseline gaps found and fixed (not CPU-variant work, but blocking it)

While implementing 68010/CPU32 support, auditing the opcode table turned up
several plain-68000 instructions that were missing entirely or silently
mis-decoded as something else, due to opcode-space carve-outs the original
table didn't account for. Two passes found:

- **Missing entirely:** `LINK`, `UNLK`, `EXT`/`EXTB`, `CHK`, `EXG`, `RESET`,
  `RTE`, `ILLEGAL`, `RTR`, `NBCD`, `MOVEP`, and all of bucket `0x5` —
  `ADDQ`, `SUBQ`, `Scc`, `DBcc`.
- **Silently mis-decoded:** `EXG` fell through to `AND`; the real `ILLEGAL`
  opcode (`0x4AFC`) and `BGND`'s slot (`0x4AFA`) fell through to `TST`;
  `EXT.W`/`EXT.L` fell through to `MOVEM`; `TAS` fell through to `TST`;
  `ADDX`/`SUBX` fell through to `ADD`/`SUB` with a misread source operand.

That covers the full canonical 68000 instruction set — every mnemonic in the
standard 68000 opcode map now has a decoder.

These were fixed in [internal/decoders/baseline.go](../internal/decoders/baseline.go)
and [baseline2.go](../internal/decoders/baseline2.go) (`RTR`/`NBCD`/`TAS`/
`ADDX`/`SUBX`/`MOVEP`), [branch.go](../internal/decoders/branch.go)
(`Scc`/`DBcc`/`TRAPcc`), and [arithmetic.go](../internal/decoders/arithmetic.go)
(`ADDQ`/`SUBQ`), each requiring the new pattern to be registered *before* the
pre-existing overly-broad pattern it collided with (`FindDecoder` returns the
first matching pattern, so precedence order is load-bearing — see the inline
comments at each insertion point in [types.go](../internal/decoders/types.go)).

`ADDX`/`SUBX` needed a second fix after the first pass: the initial patterns
left the size field (bits 7-6) as a wildcard, which also swallowed `ADDA`/
`SUBA`'s long-form opcode (`opmode=111`, i.e. size `11` — reserved for the
address-register family, not a valid `ADDX`/`SUBX` size). Fixed by
registering one exact pattern per size (mirroring `ADDQ`/`SUBQ`) instead of
one wildcard-size pattern per register/memory form — caught by
`TestDecodeRegressionRawOpcodes/SUBA_long_register` in the existing test
suite, which is exactly the kind of regression that suite exists to catch.

Not fixed (out of scope even after the above): `ADDI`/`SUBI`/`ANDI`/`ORI`/
`EORI`/`CMPI` return a hard Go error (not a `DC.W` fallback) for the reserved
size-field value `11`, which also pre-dates this work. Left alone because
fixing it is an error-handling design question (should a decoder's error be
caught and converted to `DC.W` in `decodeInstruction`?) affecting every
decoder, not a narrow opcode-table fix.

### Explicitly deferred (not implemented, on purpose)

Each of these was skipped because encoding it from memory carried a real risk
of silently-wrong bit math for an instruction rare enough that the mistake
could go unnoticed:

- **`CAS2`** — packs two independent `{Dc,Du,Rn}` triples across two
  extension words; only the single-operand `CAS` is implemented
  ([cpu020_ops.go](../internal/decoders/cpu020_ops.go)).
- **CPU32's `TBLS`/`TBLU`/`TBLSN`/`TBLUN`** table-lookup-and-interpolate
  family — multiple addressing variants not confidently recalled. `BGND` (the
  other CPU32 addition) *is* implemented.
- **68040's `MOVE16`, `CINV`, `CPUSH`** — F-line opcode space shared with FPU
  encoding, flagged in this doc from the start as the highest-risk area to
  free-hand; 68040 otherwise has the full inherited 68020+ instruction set.
- FPU and PMMU coprocessor instructions — out of scope per the Non-goals
  section from the start.

## Current state

`m68kdasm` decodes plain 68000 opcodes only:

- [internal/decoders/types.go](../internal/decoders/types.go) dispatches purely on a `[16][]OpcodePattern`
  jump table (`opcodeBuckets`) keyed by the opcode's top nibble, with no notion of "which CPU".
- [internal/decoders/addressing.go](../internal/decoders/addressing.go) `decodeAddressingMode` only implements
  the **brief** extension word format (8-bit displacement, D/A index, `.W`/`.L` size). There is no full
  extension-word support (memory indirect, base/index suppress, scaled index, base/outer displacement of
  0/16/32 bits) required by 68020 and up.
- There are no decoders at all for MOVEC/MOVES/RTD (68010+), the 68020+ bitfield instructions
  (BFxxx), CAS/CAS2, CHK2/CMP2, TRAPcc, PACK/UNPK, EXTB.L, CALLM/RTM, long MUL/DIV, LINK with 32-bit
  displacement, MOVE16 (68040), or any FPU/PMMU coprocessor opcodes.
- `DecodeOptions` ([public_types.go](../public_types.go)) has a single field, `Symbolizer`. There is no
  target-CPU option anywhere in the public API.

So "enable support for m68010, m68020, ..." is really two separable problems:

1. **New opcodes and addressing modes** that these CPUs add.
2. **A CPU-selection mechanism** so callers can pick a target, decoding only opcodes valid for that
   CPU (and, ideally, rejecting/flagging opcodes that aren't) — otherwise "support" just means "decode
   everything, unconditionally," which produces wrong results for 68000 binaries when a later opcode's
   encoding happens to collide with something else, and gives callers no way to get period-accurate
   disassembly of older CPUs.

## Goals

- Add a public `CPU` type enumerating the targets: `M68000`, `M68010`, `M68020`, `M68030`, `M68040`,
  `M68060`, `CPU32`.
- Add `DecodeOptions.CPU` (default zero value = `M68000`, preserving today's behavior with no source
  changes required from existing callers).
- Gate every new opcode and addressing mode behind the minimum CPU that introduced it, so:
  - Decoding with `M68000` (or omitting `CPU`) behaves exactly as today, byte-for-byte.
  - Decoding with a later CPU recognizes strictly more opcodes/modes, per the real hardware.
- Do **not** attempt cycle-accurate timing, privilege/exception modeling, or full coprocessor ID decode
  (cpGEN dispatch to arbitrary coprocessors). Scope is limited to what a disassembler needs: correct
  mnemonic, operands, and instruction length.

## Non-goals

- FPU (68881/68882/68040/68060 built-in) instruction decoding. This is a large, separate opcode space
  (cpGEN, F-line `1111`) and should be its own follow-up design once base-CPU gating exists.
- PMMU (68851/68030) instruction decoding — same reasoning.
- Emulating CPU-specific *illegal instruction* traps precisely (e.g. 68060 removing `MOVEP`,
  `CHK2/CMP2` availability quirks in early steppings). We aim for "the mainstream documented opcode map
  per generation," not per-stepping errata.

## Proposed CPU model

```go
// public_types.go
type CPU uint8

const (
    M68000 CPU = iota // default zero value
    M68010
    CPU32
    M68020
    M68030
    M68040
    M68060
)
```

`CPU32` is placed after `M68010` and before `M68020` in feature terms (it's a 68010-based core with a
subset of 68020 addressing modes and its own additions), but it does **not** form a linear
"newer-is-superset" chain with 68020/30/40/60 — see below. So `CPU` is not used as an ordered
`>=` comparison in general; each opcode/mode declares the explicit set of CPUs it's valid on.

### Why not a simple ordinal `>=` check?

The naive approach — `if opts.CPU >= M68010 { ... }` — breaks down because the 68k family is not a
strict feature chain:

- CPU32 (used in 68302/68306/68330 etc.) is 68010-based plus its own extensions (`TBLS`/`TBLU`
  table-lookup instructions, `BGND`, subset of 68020 addressing) but **lacks** most 68020+ addressing
  modes and instructions.
- 68040 **removes** `CALLM`/`RTM` (68020-only, already deprecated in 68030) and replaces generic cpGEN
  coprocessor dispatch with dedicated FPU opcodes plus `MOVE16`, `CINV`, `CPUSH`.
- 68060 removes hardware `BCD` support (`ABCD`/`SBCD`/`NBCD`/`PACK`/`UNPK` trap-and-emulate) and drops
  a few other opcodes (e.g. `CAS2` timing quirks aside, `MOVEP` still exists; `CHK2`/`CMP2` still exist).
  For a disassembler this mostly doesn't matter — we still want to *decode* the bytes if they appear in
  a 68060 binary produced by a compiler that assumed availability — so 68060 is modeled as "68040 opcode
  map plus 68060-specific removals we choose to still decode" rather than a hard gate.

Given that, each `OpcodePattern` (or table) is tagged with an explicit **CPU set** (a bitmask), not a
minimum version, and `FindDecoder` checks `set.Contains(opts.CPU)`. This keeps the model correct for
CPU32's divergence and cheap to extend later (e.g. adding a `68012` or `68EC030` variant is just a new
bit reused across existing tables).

```go
// internal/decoders/types.go
type cpuSet uint8

const (
    cpu000 cpuSet = 1 << iota
    cpu010
    cpu32
    cpu020
    cpu030
    cpu040
    cpu060

    cpuAll010up = cpu010 | cpu32 | cpu020 | cpu030 | cpu040 | cpu060
    cpuAll020up = cpu020 | cpu030 | cpu040 | cpu060
    cpu020_030  = cpu020 | cpu030            // e.g. CALLM/RTM
    cpuAll      = cpu000 | cpuAll010up
)
```

Each `OpcodePattern` gains a `CPUs cpuSet` field (default `cpu000` is wrong as a zero value since Go
zero-values would mean "68000 only" for *every* existing entry — which is actually what we want for the
untouched 68000 table, so this is convenient: existing patterns need no edits, only new patterns for
newer CPUs set `CPUs` explicitly).

`FindDecoder` becomes CPU-aware:

```go
func FindDecoder(opcode uint16, cpu CPU) OpcodeDecoder {
    bit := cpuBit(cpu)
    for _, pattern := range opcodeBuckets[opcode>>12] {
        if (opcode&pattern.Mask) == pattern.Value && pattern.CPUs&bit != 0 {
            return pattern.Decoder
        }
    }
    return nil
}
```

This is the key mechanical change: `FindDecoder` needs the target CPU threaded through from
`DecodeOptions`, which means it needs to flow from `disasm.go`'s `decodeInstruction` down to
`decoders.FindDecoder`. `decoders.CPU` should be defined in the `decoders` package (mirroring the
existing pattern where `public_types.go` re-exports internal types) and aliased publicly:

```go
// public_types.go
type CPU = decoders.CPU

const (
    M68000 = decoders.M68000
    M68010 = decoders.M68010
    CPU32  = decoders.CPU32
    M68020 = decoders.M68020
    M68030 = decoders.M68030
    M68040 = decoders.M68040
    M68060 = decoders.M68060
)
```

## Per-CPU feature inventory (disassembler-relevant subset)

### 68010 additions
- `MOVEC` (move to/from control register: SFC/DFC/USP/VBR) — new opcode `0x4E7A`/`0x4E7B`.
- `MOVES` (move address space) — new opcode family `0x0E00`.
- `RTD` (return and deallocate) — `0x4E74`.
- Loop-mode `DBcc` behaves differently at runtime but decodes identically — no disassembler change.
- (Bus/address error stack frame format changes — not disassembler-visible.)

### CPU32 additions (on top of 68010 base)
- `TBLS`/`TBLU`/`TBLSN`/`TBLUN` (table lookup and interpolate) — opcode family `0x8xC0`/`0x8x40`
  region overlaps with existing DIVU/DIVS bucket-0x8, needs careful mask disambiguation.
- `BGND` (background debug mode entry) — `0x4AFA`.
- A **subset** of 68020 addressing modes (brief extension word only, like 68000/68010 — CPU32 does
  *not* get full extension words). So CPU32 reuses the 68000/68010 addressing decoder, not the new
  68020+ one.

### 68020 additions (the big one)
- **Full extension word addressing modes** on top of brief format: memory indirect (pre-indexed,
  post-indexed), base register suppress, index register suppress, scaled index (`*1/*2/*4/*8`), base
  displacement (0/16/32-bit), outer displacement (0/16/32-bit). This requires rewriting
  `decodeAddressingMode` mode 6 and mode 7/reg 3 (index modes) to branch on the extension word's bit 8
  ("full" flag) when `CPU >= 020`-equivalent.
- 32-bit displacement for `Bcc`/`BSR` (`0xFF` displacement byte = use next 32 bits, vs. today's 68000
  `0x00`=16-bit only).
- `LINK` with 32-bit displacement (`0x4808` family, vs. existing 16-bit `LINK`).
- `CALLM`/`RTM` (68020/68030 only, removed 68040+).
- `CAS`/`CAS2` (compare-and-swap).
- `CHK2`/`CMP2` (bounds check / compare against bounds, opcode `0x00C0` family).
- `PACK`/`UNPK` (BCD pack/unpack).
- `TRAPcc` (conditional trap, opcode `0x50FC` family — same condition codes as `Bcc`).
- Bitfield instructions: `BFTST`, `BFCHG`, `BFCLR`, `BFSET`, `BFEXTU`, `BFEXTS`, `BFFFO`, `BFINS`
  (opcode `0xE8C0`-`0xEFC0` family, each taking a bitfield extension word with offset/width fields
  that can be immediate or register-indirect).
- Long multiply/divide: `MULS.L`/`MULU.L`/`DIVS.L`/`DIVU.L` (32×32→32/64), extending the existing
  `0xC1C0`/`0x81C0` opcodes with an extension word when the operand size is long.
- `EXTB.L` (sign-extend byte to long, `0x49C0`).
- Generic coprocessor interface (cpGEN, F-line) — **out of scope** per Non-goals.

### 68030 additions
- Same instruction set as 68020 **minus** `CALLM`/`RTM` (still decodes them as 68020-compatible if we
  choose leniency, or treats them as illegal — recommend: keep decoding them under a `cpu020_030` set
  matching real silicon, i.e. 68030 *does* still support them per Motorola docs, only later 68040 drops
  them — verify against the datasheet before finalizing, since some references list `CALLM` as already
  faulting on 68030 in some steppings).
- PMMU instructions (`PFLUSH`, `PLOAD`, `PMOVE`, `PTEST`) — **out of scope** per Non-goals (same
  reasoning as FPU: separate large opcode space, own design).

### 68040 additions
- `MOVE16` (`0xF600`-`0xF620` family, moves 16-byte aligned blocks) — note this lives in F-line space,
  shared territory with FPU; needs a narrow mask so it doesn't collide with a future FPU implementation.
- `CINV`/`CPUSH` (cache invalidate/push, `0xF400` family).
- Drops `CALLM`/`RTM`, drops generic cpGEN (has dedicated FPU opcodes instead — out of scope).
- `MOVES` still present.

### 68060 additions/removals
- Removes hardware `CAS2`? (verify) and BCD (`ABCD`/`SBCD`/`NBCD`/`PACK`/`UNPK`) execution, but these
  still need to *decode* (they trap-and-emulate in software, so real 68060 binaries can contain them,
  and a disassembler should show them, not treat them as illegal). **Recommendation: 68060 decodes the
  68040 opcode map unchanged** (superset for our purposes); do not model removals at the decode level
  unless a concrete need (e.g. a "strict" flag) comes up later.

## Addressing-mode rework (the hard part)

This is the largest single piece of work, larger than adding the opcode tables above, because
`decodeAddressingMode`'s mode-6 (address + index) and mode-7/reg-3 (PC + index) cases currently assume
the brief extension word format unconditionally.

Plan:
1. Rename current `decodeAddressingMode` to `decodeAddressingModeBrief` (or keep as the CPU32/68010/68000
   path) and introduce `decodeAddressingModeFull` for CPU ∈ {68020, 68030, 68040, 68060}, selected by a
   `cpu CPU` parameter threaded through `decodeEA`/`decodeEAWithSize` (which currently take no CPU
   parameter — this is a call-site-breaking change across every decoder in `common.go`, `move.go`,
   `single_op.go`, etc. — see Migration below).
2. `decodeAddressingModeFull` reads the extension word, checks bit 8:
   - `0` → same brief format as before (68020 still accepts brief extension words for compatibility).
   - `1` → full format: parse D/A field, W/L field, scale (bits 10-9), bit 6 = base register suppress,
     bits 5-4 = base displacement size (00=reserved/null, 01=none... per Motorola encoding, actually
     00=reserved,01=null,10=word,11=long), bits 2-0 = index/indirection (I/IS field) selecting one of
     the memory-indirect variants (no memory indirect, indirect pre-indexed with null/word/long outer
     displacement, indirect post-indexed variants, no-index indirect variants).
   - This needs a new `EffectiveAddressKind` set: `EAKindMemoryIndirect`, `EAKindMemoryIndirectPreIndexed`,
     `EAKindMemoryIndirectPostIndexed` (or a single kind plus a `PreIndexed bool` + `Indirect bool` flag
     pair — prefer flags to avoid combinatorial kind explosion), plus `IndexRegister.Scale int` (1/2/4/8)
     and `BaseDisplacementSize`/`OuterDisplacementSize` fields, and an `OuterDisplacement *int32` field
     on `EffectiveAddress`.
3. `formatImmediate`/text rendering needs new format strings, e.g. `([bd,An,Xn.SIZE*SCALE],od)` for
   memory indirect post-indexed.
4. Extend `Instruction.Size`/`ExtensionWords` accounting: full extension words can pull in 0-2 *additional*
   16-bit words beyond the extension word itself (base displacement + outer displacement), on top of
   what brief mode needed — the existing `NeedMoreError`/two-pass "ask for more, retry" decode loop in
   `disasm.go`'s `decodeInstruction` already supports this pattern generically, so no change needed
   there, just correct `Missing` counts from the addressing decoder.

## Public API changes

```go
// public_types.go
type DecodeOptions struct {
    Symbolizer Symbolizer
    CPU        CPU // zero value = M68000, preserves current behavior
}
```

No other public signatures change. `Decode`, `DecodeWithOptions`, etc. stay as-is; `CPU` just becomes a
second knob on `DecodeOptions`, consistent with how `Symbolizer` was added.

Unrecognized opcodes for the selected CPU (e.g. `MOVEC` decoded with `CPU: M68000`) fall through to the
existing `DecodeUnknown` DC.W-pseudo-op path — i.e., "not on this CPU" and "not a real opcode" produce
the same observable result, which matches today's philosophy of never hard-erroring on unknown bytes.

## Migration / call-site impact

`FindDecoder` and every function in the `decoders` package that currently hardcodes 68000-only
addressing assumptions needs the CPU threaded through. Concretely:

- `decoders.FindDecoder(opcode uint16)` → `decoders.FindDecoder(opcode uint16, cpu CPU)`.
- `decodeEA`/`decodeEAWithSize` in `common.go` gain a `cpu CPU` parameter, which every one of their
  ~20+ call sites across `move.go`, `single_op.go`, `arithmetic.go`, `logical.go`, `compare.go`,
  `bit.go`, `shift.go`, `special.go` must pass through. Because `OpcodeDecoder` is
  `func(data []byte, opcode uint16, inst *Instruction) error` with no CPU parameter, **the decoder
  function signature itself must gain `cpu CPU`**, i.e. `OpcodeDecoder = func(data []byte, opcode
  uint16, inst *Instruction, cpu CPU) error`. This is the real "blast radius" of the feature: every
  existing decoder function's signature changes, even ones (like `decodeMOVEQ`) that never look at
  `cpu`. This is mechanical (add a parameter, ignore it in ~80% of decoders) but touches every file in
  `internal/decoders/`.
- `decodeInstruction` in `disasm.go` passes `opts.CPU` down to `decoders.FindDecoder` and into the
  decoder-call loop.

Recommend doing this as a dedicated first PR ("thread CPU through the decoder pipeline, no new opcodes
yet") so the mechanical signature churn is reviewed separately from the semantic addition of new
instructions — this matches the pattern already used in this repo for the `Symbolizer`/metadata
refactor (per commit `422c0f4`, "Simplify decoders, dedupe public/internal types").

## Suggested delivery sequence

1. **Plumbing PR**: add `CPU` type, `DecodeOptions.CPU`, thread `cpu CPU` through `OpcodeDecoder` and
   `FindDecoder`, tag all *existing* opcode patterns with `CPUs: cpuAll` (i.e., no behavior change —
   everything still decodes on every CPU, since none of the current opcodes are actually 68000-exclusive
   in a way that conflicts with newer encodings). Add exhaustive regression tests asserting
   `M68000`-targeted decode output is byte-identical to today's output for the existing test corpus.
2. **68010 + CPU32 PR**: `MOVEC`, `MOVES`, `RTD` (68010/CPU32/020+), `TBLS`/`TBLU`/`BGND` (CPU32 only).
   No addressing-mode changes needed yet (both stay on brief format).
3. **68020 addressing-mode PR**: full extension word support (`decodeAddressingModeFull`), new
   `EffectiveAddress` fields, 32-bit `Bcc`/`LINK` displacement. This is the highest-risk PR — needs a
   large table-driven test suite against known-good 68020 disassembly (e.g. cross-check against
   `vasm`/`gdb`/`objdump -m68020` output for a corpus of real 68020 binaries).
4. **68020 opcode PR**: `CAS`/`CAS2`, `CHK2`/`CMP2`, `PACK`/`UNPK`, `TRAPcc`, `BFxxx` bitfield family,
   long `MULS`/`MULU`/`DIVS`/`DIVU`, `EXTB.L`, `CALLM`/`RTM`.
5. **68030 PR**: reuses 68020 tables verbatim (tag existing patterns with `cpu020_030` where CALLM/RTM
   are 020/030-only); confirm against datasheet whether any 68020 opcode is actually invalid on 68030
   before just aliasing the CPU bit.
6. **68040 PR**: `MOVE16`, `CINV`/`CPUSH`; remove `CALLM`/`RTM` from the CPU set (already tagged
   `cpu020_030`, so 68040/68060 naturally exclude them — no table edit needed if step 5 tagged correctly).
7. **68060 PR**: tag as reusing the 68040 opcode map (`cpu040 | cpu060` wherever 68040 patterns live);
   add any 68060-specific opcodes if research turns up ones not already covered (none currently known
   beyond FPU, which is out of scope).
8. **(Future, separate design doc)** FPU and PMMU coprocessor instruction decoding.

## Testing strategy

- Extend the existing `internal/decoders/types_test.go` pattern: table-driven tests per CPU asserting
  both *positive* cases (new mnemonic decodes correctly on its minimum CPU) and *negative* cases (same
  bytes decode as `DC.W`/unknown on an older CPU that doesn't have the opcode).
- For addressing modes specifically, build a fixture corpus of hand-verified 68020 full-extension-word
  encodings (all combinations of base suppress × index suppress × scale × bd size × od size × pre/post
  indirect) — this is the part most likely to have off-by-one bugs in bit-field extraction.
- Add a `CPU` dimension to any existing golden-file/snapshot tests so regressions on the default
  (`M68000`) path are caught immediately if the plumbing PR accidentally changes default behavior.

## Open questions for the user

1. Is FPU/PMMU decoding wanted eventually, or is "integer core only" the permanent scope? (Affects
   whether `CPU` should reserve bit space / naming for `68881`/`68882` as separate coprocessor
   selectors now vs. later.)
2. Should an unsupported opcode-for-CPU combination be observable in `Instruction`/`Metadata` (e.g. a
   `SupportedOn CPU` field or a warning), or is falling through to the existing `DC.W` unknown-op
   rendering sufficient? Today's design doc assumes the latter (simplest, consistent with existing
   philosophy).
3. Priority order across steps 2-7 above — e.g. is 68020 (the addressing-mode superset most other
   tooling cares about) more urgent than finishing 68010/CPU32 first?
