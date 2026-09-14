# Design: FPU and MMU Coprocessor Support (68881/68882/68851, 68040/68060 built-in)

## Status

**FPU: implemented** (delivery-sequence steps 1-8.5 below — the entire 68881/68882/68040/68060 FPU
instruction set this doc scoped in, all 7 data formats, both directions). **PMMU: implemented**
(delivery-sequence steps 9-10 — every mainstream 68851/68030 PMMU instruction: `PMOVE`'s entire
register set, `PMOVEFD`, `PFLUSHA`, `PFLUSH`, `PFLUSHS`, `PFLUSHR`, `PLOADR`, `PLOADW`, `PTESTR`,
`PTESTW`, `PSAVE`, `PRESTORE`, and the full `Pcc` condition/branch/set/trap family). Two narrow items
remain open (the 68040's own simplified single-word PMMU forms, and `PVALID`) — see step 10's own notes
below — but this doc's originally-scoped FPU and PMMU work is otherwise done. This was originally the
"Step 8" follow-up flagged as future work in
[design-cpu-variants.md](design-cpu-variants.md), whose Non-goals section explicitly scoped FPU and
PMMU decoding out: "a large, separate opcode space (cpGEN, F-line `1111`) and should be its own
follow-up design once base-CPU gating exists." Base-CPU gating (the `CPU`/`cpuSet` machinery) existed
by the time this doc was written; the FPU side was then implemented against
[github.com/jenska/m68kasm](https://github.com/jenska/m68kasm) v1.5.0's verified encoder as ground
truth, round-tripping every mnemonic through the real assembler rather than trusting bit-layout
derivations alone (see [internal/decoders/fpu.go](../internal/decoders/fpu.go) and
[fpu_test.go](../fpu_test.go)) — this caught several real bugs before they shipped (see the delivery
sequence below for specifics).

## Current state

*(This section is the original pre-implementation snapshot that motivated this doc — kept for context,
not maintained to track ongoing progress. See the Status section above and the delivery sequence below
for what's actually implemented today.)*

- Opcode bucket `0xF` (the top nibble `1111`, i.e. "F-line") in `opcodeBuckets`
  ([internal/decoders/opcodetable.go](../internal/decoders/opcodetable.go)) is **completely empty**.
  Every F-line opcode — FPU, PMMU, and anything else a real coprocessor might occupy — currently falls
  through `FindDecoder` and renders as the generic `DC.W` unknown-opcode pseudo-instruction, on every
  CPU. This is true even for `M68040`/`M68060`, which have a real built-in FPU: today's decoder cannot
  tell an `FADD` apart from garbage.
- **Correction (this doc originally claimed a gap here that doesn't exist — an earlier pass grepped
  `move.go` with an incomplete keyword list and never actually read the full map):**
  `controlRegisterNames` in [internal/decoders/move.go:193](../internal/decoders/move.go#L193) (used by
  `MOVEC`, 68010+) already lists `SFC`/`DFC`/`USP`/`VBR`/`CACR`/`CAAR`/`MSP`/`ISP`/`TC`/`ITT0`/`ITT1`/
  `DTT0`/`DTT1`/`MMUSR`/`URP`/`SRP`/`PCR` — it was filled in as part of the original multi-CPU-variant
  work (commit `5c40fc0`), before this document existed. So 68040/68060 MMU-register disassembly via
  `MOVEC` is **already implemented**, not a gap this doc needs to close. The one register still missing
  from the map is 68060's `BUSCR` (Bus Control Register, code `0x008` per Motorola's MC68060 User's
  Manual — verify before adding, same as everything else in this doc); everything else is done.
- The `CPU`/`cpuSet` gating model (`decoders.CPU`, `cpuSet`, `exactCPU`/`maskedCPU`,
  `FindDecoder(opcode, cpu)`, and the `cpu CPU` parameter already threaded through every
  `OpcodeDecoder`) is exactly the mechanism this doc needs to reuse — see "Coprocessor availability
  model" below for why it needs one extra dimension, not a replacement.
- The full-extension-word addressing infrastructure added for 68020
  ([internal/decoders/addressing020.go](../internal/decoders/addressing020.go)) and the brief-format
  decoder ([internal/decoders/addressing.go](../internal/decoders/addressing.go)) already produce the
  `EffectiveAddress` operand shape FPU/PMMU memory operands need — they reuse standard `<ea>` encoding,
  not a new one (see "Addressing modes: reuse, not reinvent" below).
- The `NeedMoreError`/two-pass retry loop in `disasm.go`'s `decodeInstruction` already supports decoders
  that discover they need more trailing bytes than they were first given (used today for 68020's
  variable-length full extension words). FPU extension words (up to 3 additional 16-bit words for an
  extended-precision or packed-decimal immediate) are the same shape of problem and need no new
  mechanism, only correct `Missing` counts from the new decoders.

## Goals

- Decode the classic **68881/68882 discrete FPU** instruction set: register-to-register and
  memory-to-register FP data-movement and arithmetic (`FADD`, `FSUB`, `FMUL`, `FDIV`, `FSGLDIV`,
  `FCMP`, `FTST`, `FABS`, `FNEG`, `FSQRT`, `FMOVE`, `FMOVEM`, `FMOVECR`), the transcendental subset
  (`FSIN`, `FCOS`, `FTAN`, `FATAN`, `FLOGN`, `FLOG2`, `FETOX`, `FGETEXP`, `FGETMAN`, ...), the FP
  condition/branch family (`FBcc`, `FDBcc`, `FScc`, `FTRAPcc`, `FNOP` as `FBcc.b #0`), and
  `FMOVEM`/`FMOVE` to/from the FP control registers (`FPCR`, `FPSR`, `FPIAR`).
- Decode the **68040/68060 built-in FPU**, which is opcode-compatible with 68881/68882 for the
  hardware-implemented subset (the rest traps and is emulated in software — not disassembler-visible;
  the encoding is unchanged either way).
- ~~Decode 68040/68060 **MMU control-register access via `MOVEC`**~~ — already done (see the
  "Current state" correction above); only 68060's `BUSCR` register is still missing from
  `controlRegisterNames`, a one-line follow-up whenever someone verifies its exact code.
- Decode the classic **68851 discrete PMMU** and **68030 built-in PMMU** F-line instruction set:
  `PLOAD`, `PFLUSH`, `PFLUSHA`, `PMOVE`, `PTEST`, `PVALID`, and the PMMU condition/branch family
  (`PBcc`, `PDBcc`, `PScc`, `PTRAPcc`), gated appropriately (68851 usable with 68020/68030; the 68030's
  own built-in PMMU is a fixed subset; 68040/68060 have **no** PMMU F-line opcodes at all — their MMU is
  configured entirely through `MOVEC` control registers, covered above).
- Extend the operand/metadata model (`Operand`, `EffectiveAddress`, `Metadata`) with whatever new
  structured fields FP/PMMU operands need (FP registers, FP data formats, k-factors, MMU function
  codes/masks), following the existing pattern of adding fields rather than new parallel types.
- Scope is, as with the base-CPU work: correct mnemonic, operands, and instruction length. Not cycle
  timing, not exception/trap semantics, not state-frame *content* interpretation.

## Non-goals

- Cycle-accurate timing or exception modeling (same as the base-CPU doc).
- Full generic **cpGEN** coprocessor dispatch for arbitrary coprocessor IDs. Only CpId values actually
  used by real 68k systems are in scope: FPU at CpId 1, PMMU at CpId 0 (see "Coprocessor ID field"
  below) — verify against the datasheet before coding, but this is the well-established convention
  every 68k toolchain (gcc, binutils, vasm) assumes, not a free choice.
- Interpreting the *contents* of `FSAVE`/`FRESTORE` and `PSAVE`/`PRESTORE` coprocessor state frames.
  These instructions themselves (opcode, `<ea>`, idle vs. busy vs. null frame format byte) are in
  scope to decode as instructions; the variable-length frame body they read/write at runtime is a
  hardware/runtime state dump with no disassembly-time meaning and should render as an opaque `<ea>`
  operand, same as `MOVEM`'s register list does not need to know what values are in memory.
- 68851-specific ATC (address translation cache) diagnostic instructions beyond the mainstream
  documented set, unless research turns up they're commonly seen in real binaries.
- Emulating per-stepping illegal-instruction quirks (e.g. exact FPU/PMMU opcodes that specific early
  68882 or 68030 steppings lacked). Same "mainstream documented opcode map per generation" philosophy
  as the base-CPU doc.

## Why this needs its own document, not just more `cpuSet` bits

The base-CPU doc's model is: for a given target `CPU`, is opcode X valid, yes or no. That works because
68000/68010/CPU32/68020/68030/68040/68060 form (almost) a fixed hardware identity — you don't choose
"a 68020 with extra instructions bolted on" at decode time, you choose which *chip*.

FPU and PMMU don't fit that cleanly, because on 68000–68030 they are **separate, optional chips**:

- A 68000, 68010, or 68020 board may have *no* FPU, a 68881, or a 68882 (68882 adds a couple of extra
  addressing-mode/timing optimizations that are opcode-*compatible* with 68881 — no new mnemonics for
  our purposes).
- A 68020 or 68030 board may have *no* PMMU, or a 68851.
- 68030 has a PMMU **built in** (a fixed, smaller instruction subset of the full 68851) but *no* FPU
  built in — FPU is still a separate 68881/68882 chip choice on a 68030 board.
- 68040 and 68060 have **both** FPU and MMU built in, but the MMU is *not* PMMU-opcode-compatible at
  all — it's configured via `MOVEC` control registers, not F-line instructions. So "MMU support" on
  68040/68060 and "MMU support" on 68020/68030/68851 are two unrelated instruction sets that happen to
  share a name.

So availability is **not** a function of `CPU` alone; it's `CPU` plus an independent "what coprocessors
does this target have" choice, matching how real toolchains handle it (e.g. GCC's `-m68881`/`-msoft-float`
flags are orthogonal to `-m68020`/`-m68030`/etc.).

## Coprocessor availability model

Add two independent capability flags to `DecodeOptions`/the decoder's CPU context, alongside (not
replacing) `CPU`:

```go
// public_types.go
type DecodeOptions struct {
    Symbolizer Symbolizer
    CPU        CPU
    FPU        bool // decode 68881/68882-compatible FPU opcodes
    MMU        bool // decode 68851/68030-PMMU-compatible F-line MMU opcodes
}
```

Defaults (`false`, `false`) preserve today's behavior: F-line still renders as `DC.W` unless a caller
opts in, matching the zero-value-preserves-behavior pattern used for `CPU` itself.

Resolution rules, decided once per decode call (cheap to compute, not per-instruction):

- `FPU: true` is meaningful for any `CPU` — it means "assume a coprocessor/built-in FPU is present,"
  independent of which integer core. `M68040`/`M68060` do not strictly need the flag (they always have
  one), but requiring it uniformly keeps the rule simple and matches "opt in to F-line decoding" being
  one flag rather than a CPU-dependent special case; recommend defaulting `FPU: true` automatically
  when `CPU` is `M68040`/`M68060` and the caller left it `false`, so the common case ("I'm targeting a
  68040") doesn't require remembering a second flag, but an explicit `false` should still be honorable
  for a soft-float-only 68040 binary — i.e. treat it as "at least true", not force it.
- `MMU: true` selects the 68851/68030-style PMMU opcode set. It is **not** applicable to
  `M68040`/`M68060` (no such opcodes exist there); the decoder should not gate on it for those CPUs —
  MMU control-register access on 68040/68060 is unconditionally available through `MOVEC` once the
  `controlRegisterNames` table is extended, same as any other control register today.
- Internally, extend `cpuSet`-style gating with a small parallel `coproSet` (or fold two more bits into
  the existing pattern struct as `RequiresFPU bool` / `RequiresMMU bool`, whichever reads cleaner once
  written) so `FindDecoder` becomes: pattern's `CPUs` bit is set for target CPU, **and** (pattern
  doesn't require FPU, or `opts.FPU`), **and** (pattern doesn't require MMU, or `opts.MMU`).
- An opcode whose bits match an FPU/MMU pattern but the corresponding flag is off should fall through
  to `DC.W`, exactly like an opcode not valid for the target `CPU` does today — same "unsupported and
  unrecognized look the same" philosophy carried over from the base-CPU doc.

## Coprocessor ID field and F-line layout

Every F-line opcode `1111 ccc ppppppppp` (bits 15-12 = `1111`) carries a 3-bit coprocessor-ID field at
bits 11-9 (`ccc`) for the generic cpGEN dispatch mechanism. By hardware/toolchain convention (verify
against the MC68881/MC68882/MC68851 Programmer's Reference Manuals before coding, but this is the
standard nearly every 68k target assumes, not a per-project choice):

- **CpId 1** (`0xF2xx`–`0xF3xx` range) → FPU. Sub-selected by a 3-bit "mode" field (bits 8-6) into:
  general FP instruction (register/memory source, opmode in a following extension word — this is where
  `FADD`/`FSUB`/`FMOVE`/etc. all live), `FDBcc`, `FScc`/`FTRAPcc`, `FBcc` (short and long displacement
  forms), and `FSAVE`/`FRESTORE`.
- **CpId 0** (`0xF0xx`–`0xF1xx` range) → PMMU. Similarly mode-field-selected into `PLOAD`/`PFLUSH`/
  `PMOVE`/`PTEST`/`PVALID` (general form) plus `PDBcc`/`PScc`/`PTRAPcc`/`PBcc` and `PSAVE`/`PRESTORE`.

**This document deliberately does not pin down exact mask/value constants for the mode-field
submnemonics**, unlike most of the base-CPU doc. The base-CPU doc's own "Explicitly deferred" section
already established the precedent: instructions rare enough, and bit-packed densely enough, that
encoding from memory risks silently-wrong bit math are better looked up than guessed (that's exactly
why `CAS2`, CPU32's `TBLS`/`TBLU` family, and 68040's `MOVE16`/`CINV`/`CPUSH` were skipped rather than
free-handed). The general FP instruction extension word in particular packs: R/M bit, source-specifier
field (register number or `<ea>` data format selector), and a 7-bit opmode field selecting among ~50
operations, several of which alias between "monadic" and "dyadic" depending on other bits — this is the
single highest-risk area in the entire FPU instruction set to get wrong from recall. Each opcode group
below must be implemented against a primary source (the MC68881/68882 PRM, the MC68851 PRM, or
cross-checked against `binutils`' `m68k-opc.c` / an existing disassembler's table), not from memory.

## New operand/metadata needs

FP and MMU operands don't fit the existing `Register`/`EffectiveAddress` shapes cleanly:

- **FP registers** (`FP0`-`FP7`): add `RegisterKindFP` to `RegisterKind` — same `Register{Kind, Number}`
  shape already used for D/A/PC registers, so this is additive, not a new type.
- **FP data formats**: integer operands have 3 sizes (B/W/L via the existing `sizeNames`/size-field
  convention); FP memory operands have 7: byte, word, long, single (32-bit float), double (64-bit
  float), extended (96-bit, 80 bits + padding), and packed decimal (96 bits, BCD digits + exponent).
  `ImmediateValue.Size` is currently a `uint8` byte count, which already generalizes to "12" for
  extended/packed without a shape change — but rendering a 96-bit immediate needs new formatting, not
  just a wider `Value` field (today's `ImmediateValue.Value uint32` cannot hold it). Recommend adding
  `ImmediateValue.RawBytes []byte` (or a dedicated `FPImmediateValue` alongside it) holding the raw
  extension-word bytes for single/double/extended/packed immediates, formatted at render time rather
  than parsed to a Go `float64`/`float32` at decode time (extended precision has no native Go
  representation, and packed BCD needs digit-by-digit formatting, not numeric parsing) — this mirrors
  how `EffectiveAddress.OuterDisplacement` etc. were added incrementally rather than reshaping
  `ImmediateValue`.
- **FP condition codes**: `FBcc`/`FDBcc`/`FScc`/`FTRAPcc` use a **6-bit** condition field (32 defined
  conditions covering NaN-aware orderings: `EQ`, `NE`, `GT`, `NGT`, `UN` (unordered), `OR` (ordered),
  etc.) — a materially larger and different set than the integer `Bcc`/`DBcc`/`Scc` 4-bit condition
  field. Do not attempt to unify with the existing integer condition-code table; add a parallel
  `fpConditionNames` table, same shape as the existing one for integer `cc`.
- **K-factor** (`FMOVE`/packed-decimal operations' rounding-precision/digit-count specifier): either a
  static 7-bit signed immediate or a dynamic data-register reference, encoded in the extension word's
  low 7 bits. Needs a small new `Operand`/metadata shape (`KFactor *int8` static or `KFactorRegister
  *Register` dynamic) — follow the existing `ImmediateValue`/`Register` pointer-field pattern already
  used throughout `EffectiveAddress`.
- **MMU function-code/mask/register operands** (`PMOVE`'s TT/TC/CRP/SRP/DRP register selectors,
  `PLOAD`'s FC field which can be immediate, a data register, SFC, or DFC): reuse `Register` and
  `ImmediateValue` as-is; only need a small enum-to-string table per instruction, similar to
  `controlRegisterNames`.
- **FPCR/FPSR/FPIAR** and MMU's `TC`/`CRP`/`SRP`/`DRP`/`TT0`/`TT1`/`MMUSR` are *not* `MOVEC` control
  registers (those are core-integer registers) — they're addressed directly in the FP/PMMU general
  instruction's own register-select field, a separate namespace from `controlRegisterNames`. Do not
  conflate the two; add dedicated name tables (`fpControlRegisterNames`, `mmuRegisterNames`).

## Addressing modes: reuse, not reinvent

FPU and PMMU `<ea>` operands use the *exact same* effective-address encoding as the integer core
(modes 0-7, brief or full extension words per `CPU`) — this is one of the few places 68k coprocessor
design kept things simple. So `decodeEA`/`decodeEAWithSize` (already CPU-aware since the 68020 work)
should be called unchanged; **no new `EffectiveAddressKind` values are needed for FPU/PMMU**.

The one wrinkle: FP `<ea>` operand *size* is selected by a 3-bit data-format field (0-6, for the 7
formats above) living in the FP general instruction's extension word, not by the 2-bit B/W/L size field
the integer decoders read from the opcode word itself. `decodeEAWithSize`'s `operandSize int` parameter
already exists for exactly this purpose (integer decoders pass 1/2/4 for B/W/L today) — FP decoders can
pass 1/2/4/4/8/12/12 for byte/word/long/single/double/extended/packed respectively, reusing the same
parameter, with the byte-count-to-format mapping only mattering at the caller (FP decoder) and
render-time formatting layer, not inside `decodeEAWithSize` itself.

## 68040/68060 MMU via `MOVEC`

Already implemented (see the "Current state" correction above) — `controlRegisterNames`
([internal/decoders/move.go:193](../internal/decoders/move.go#L193)) already covers `TC`/`ITT0`/`ITT1`/
`DTT0`/`DTT1`/`MMUSR`/`URP`/`SRP`/`PCR`/`MSP`/`ISP`. Only `BUSCR` (68060) remains, whenever its exact
code is verified against the datasheet.

## Suggested delivery sequence

1. ~~**68040/68060 MMU-via-MOVEC PR**~~ — already done; not part of this doc's remaining work.
2. ~~**Capability-flag plumbing PR**~~ — done: `DecodeOptions.FPU` exists, `RegisterKindFP` added, gating
   threaded through `FindDecoder` via `OpcodePattern.RequiresFPU`. (`MMU` was deliberately *not* added
   yet, to avoid a dead/no-op public field ahead of any PMMU pattern actually using it — add it in the
   PMMU step instead.)
3. ~~**FPU register-to-register PR**~~ and 4. ~~**FPU memory-operand PR**~~ — done, and combined into one
   slice rather than split: `FMOVE`/`FADD`/`FSUB`/`FMUL`/`FDIV`/`FCMP`/`FABS`/`FNEG`/`FSQRT`/`FTST`/
   `FNOP`, register-to-register and `<ea>` forms (load and store), all 7 data formats including verified
   IEEE-754/68881-extended-precision immediate decoding. See
   [internal/decoders/fpu.go](../internal/decoders/fpu.go) and [fpu_test.go](../fpu_test.go), verified
   against `github.com/jenska/m68kasm` v1.5.0's encoder rather than a datasheet lookup (a stronger source
   than "verify against the PRM," since it's machine-checked by round-tripping real assembled bytes).
   Packed-decimal (`.p`) store's k-factor, deferred from this step's original scope, landed later —
   see step 8.5 below.
   `FMOVEM` and `FMOVECR` were originally listed here too but landed as later, separately-verified
   steps (5 and 6.5 below).
5. ~~**FMOVEM PR**~~ — done: static/dynamic register list, to/from general memory or predecrement
   addressing (see [internal/decoders/fpu.go](../internal/decoders/fpu.go)'s `decodeFMOVEM`), verified
   against m68kasm's `cpu020_fpu_movem.go` including a byte-level cross-check against its own literal
   test vectors before any decode logic was written. The separate `FPCR`/`FPSR`/`FPIAR`
   control-register-list form (`cpu020_fpu_movem_ctrl.go` in m68kasm) landed later, folded into step 8.
6. ~~**FPU transcendental PR**~~ — done: `FSIN`/`FCOS`/`FTAN`/`FATAN`/`FASIN`/`FACOS`/`FATANH`/`FSINH`/
   `FCOSH`/`FTANH`/`FETOX`/`FETOXM1`/`FLOGN`/`FLOGNP1`/`FLOG10`/`FLOG2`/`FTWOTOX`/`FTENTOX` — all 18
   are the exact same monadic shape as `FABS`/`FNEG`/`FSQRT`, so this was purely opmode-table entries in
   `fpGeneralOps`, no new decode logic. Gated by the same `DecodeOptions.FPU` flag as everything else,
   not a separate "full FPU" capability (m68kasm's `FeatFPUFull` encoder-side distinction between a
   discrete 68881/68882 and a reduced/integrated FPU isn't disassembler-visible — the opcode decodes
   identically either way). The "math extensions" bucket (`FGETEXP`/`FGETMAN`/`FSCALE`/`FMOD`/`FREM`)
   originally deferred out of this step landed separately — see step 6.6 below.
6.5. ~~**FMOVECR and FSINCOS**~~ — done: `FMOVECR #<romIndex>,FPn` (word2's top 6 bits, `0xFC00` mask /
   `0x5C00` value, distinguish it from the general arithmetic family — its format-code field would
   otherwise read as the reserved value 7) and `FSINCOS <ea>,FPc:FPs` / `FPm,FPc:FPs` (the one FPU
   instruction with two destination registers — sine at the usual bits 9-7, cosine at bits 2-0). Both
   needed their own decode path outside `fpGeneralOps`'s single-opcode/single-dst model. One real bug
   caught before it shipped: the initial `FSINCOS` detection checked the full 7-bit opmode field
   against the literal `0x30`, missing that bits 2-0 of that literal are the *variable* cosine-register
   field, not fixed opcode bits — any FSINCOS with a nonzero cosine register failed to decode at all
   until narrowed to the actual fixed selector, bits 6-3 (`0x78` mask). Caught by
   `TestFPUSINCOSRoundTrip` (fpu_test.go) before merging, not after.
6.6. ~~**Math extensions**~~ — done: `FGETEXP`/`FGETMAN` (monadic, same shape as `FABS`/`FNEG`/`FSQRT`)
   and `FSCALE`/`FMOD`/`FREM` (genuinely binary — two FPn operands, same `{hasDst: true, canStore:
   false}` shape `FADD`/`FSUB` already use, not `FABS`'s). All five were pure `fpGeneralOps` table
   additions, no new decode logic, same as the transcendental set (step 6). opBase values from
   m68kasm's `cpu020_fpu_mathext.go`.
7. ~~**FP condition/branch PR**~~ — done: `FBcc` (word/long displacement), `FDBcc`, `FScc`, `FTRAPcc`
   (bare/word/long), with the 32-entry (5-bit, not 6 as this doc originally guessed before the real
   condition table was read from m68kasm) FP condition table. Implementing this surfaced and fixed a
   real, independent bug: `decodeBxx`'s `.W`/`.L` forms and `decodeDBcc` computed the branch target
   relative to the instruction's *total length* rather than *(address + 2)*, off by 2-4 bytes on every
   16/32-bit-displacement branch — undetected because no prior test exercised those forms via the
   assembler. Fixed in the same session (`branch.go`'s new `branchTarget` helper), reused by
   `FBcc`/`FDBcc` so both families share one verified formula. See commit history for detail.
8. ~~**FSAVE/FRESTORE PR**~~ — done: the instruction shell only (mnemonic + `<ea>`), no frame-content
   interpretation, per Non-goals. Unlike every other instruction this doc covers, these are single-word
   opcodes with no coprocessor command word2 at all — `fpuWord1Base | 0x0100`/`0x0140 | <ea>`. Also
   folded in here (found to be a small, cleanly-scoped remainder of step 5, not worth its own step):
   FMOVEM's separate `FPCR`/`FPSR`/`FPIAR` control-register-list form (a 3-bit mask at word2 bits 12-10,
   a genuinely different subsystem from the `FP0`-`FP7` 8-bit mask). Implementing it required replacing
   `decodeFMOVEM`'s dispatch — a naive "top nibble" read (bits 15-12), which had been correct for the
   FPn-list forms since their mask fields never touch bit 12, breaks for the control-register form: its
   3-bit mask sits at bits 12-10 and can set bit 12 itself (whenever `FPCR` is included), so e.g.
   `FMOVEM FPCR/FPSR/FPIAR,(A0)` encodes word2 as `$BC00`, nothing like its own `$A000` base literal.
   Replaced with a `word2 & 0xE000` ("class") dispatch, the only bit range genuinely stable across all
   four FMOVEM word2 shapes — caught by `TestFPUMOVEMCtrlRoundTrip`'s all-three-registers case before
   it shipped wrong, not after. See `decodeFMOVEM` in `internal/decoders/fpu.go` for the full
   derivation.
8.5. ~~**Packed-BCD (`.p`) store, k-factor**~~ — done, closing out the last open item from step 4 and
   the FPU side of this doc entirely (all 7 data formats now decode in both directions). The load
   direction (`FMOVE.P <ea>,FPn`) needed no new code — format code 3 flows through the existing generic
   `<ea>` path like every other format. The store direction (`FMOVE.P FPn,<ea>{k}`) is a genuinely
   different word2 shape: real hardware has no format-code field there at all (a store destination is
   always packed, implied by the mnemonic), so the bits that would otherwise be `FFPFormat`'s R/M+format
   field are repurposed for a k-factor (mantissa-digit count) instead — a static 7-bit two's-complement
   value, or a Dn register holding it at runtime. Detected via a `word2&0xEC00==0x6C00` check before the
   generic dispatch, deliberately excluding bit 12 (the one bit that differs between the static/dynamic
   sub-forms) from the match mask. Without this check, `decodeFPStore` would have silently misdecoded
   it as `FMOVE.P <ea>,FPn` — reading the k-factor bits as a destination FPn register — since format
   code 3 is a perfectly ordinary `<ea>` format for every *other* store direction; verified this was a
   real, not hypothetical, collision by tracing the bit math by hand before writing the fix, the same
   way the FMOVECR/FSINCOS/FMOVEM collisions earlier in this sequence were each confirmed. K-factor
   renders as a `{...}` suffix appended directly to the destination `<ea>` text (GAS's own syntax,
   e.g. `FMOVE.P FP3,BUFFER{#-5}`), not a separate operand.
9. **68851/68030 PMMU PR** — in progress. `PMOVE` (now including `BAD0`-`BAD7`/`BAC0`-`BAC7` — see
   below), `PMOVEFD`, and `PFLUSHA` done: `DecodeOptions.MMU` added (deferred from step 2 as
   planned, now that real PMMU patterns exist to gate), `RequiresMMU` threaded through `FindDecoder` the
   same way `RequiresFPU` already was, and `decodePMMUGeneral` (`internal/decoders/pmmu.go`) — the single
   dispatch point for every PMMU instruction sharing the bare `0xF000|<ea>` word1 shape, the PMMU
   analogue of `decodeFPGeneric`'s word2-based dispatch — decoding `TC`, `DRP`, `SRP`, `CRP`, `CAL`,
   `VAL`, `SCC`, `AC`, `PCSR`, `TT0`, `TT1`, `MMUSR` (22 `PMOVE` forms plus 6 `PMOVEFD` forms), all
   verified against `m68kasm`'s encoder and passing round-trip on the first try (no collision or
   bit-math surprise this time, unlike almost every FPU step).
   ~~`PFLUSH`/`PFLUSHS`/`PFLUSHR`, `PLOADR`/`PLOADW`, `PTESTR`/`PTESTW`~~ — also done: the new "function
   code specifier" operand (`SFC`/`DFC`/a `Dn`/an immediate — a 2-bit mode + 3-bit value field at word2
   bits 4-0, `fcSpecOperand` in `pmmu.go`), `PFLUSH`/`PFLUSHS`'s optional trailing `<ea>` (two word2
   sub-variants each, distinguished by a class mask), and `PTESTR`/`PTESTW`'s optional trailing `An`
   result register — rendered only when nonzero, since real hardware has no separate "An present" flag
   distinct from the register value itself, so `PTESTR FC,<ea>,#level` and `...,#level,A0` are
   bit-identical (an inherent ambiguity, not a decoder gap — m68kasm's own encoder produces the same
   bytes for both spellings). All 19 test cases passed round-trip on the first try, including every
   FC-spec spelling across every instruction family and both the omitted/explicit trailing-`An` cases.
   ~~`BAD0`-`BAD7`/`BAC0`-`BAC7`~~ (breakpoint address/access registers) — also done: a numbered-register
   shape (register number 0-7 at bits 4-2, `decodeBADBAC` in `pmmu.go`) distinct from every other `PMOVE`
   register, with an inverted load/store direction bit relative to them (bit 9 set means *load* here,
   not store — every other `PMOVE` register uses that bit the other way around). Traced m68kasm's own
   `cpu030_pmmu_badbac.go` bit math by hand before coding, given the inverted-bit convention is exactly
   the kind of easy-to-transpose detail that bit past bugs in this sequence — all 6 test cases passed
   round-trip on the first try, confirming the hand derivation held up. This completes `PMOVE`'s entire
   register set. ~~`PSAVE`/`PRESTORE`~~ — also done, exactly as quick as predicted: structurally
   identical to `FSAVE`/`FRESTORE` (single-word, no coprocessor command word2, `-(An)`-only/
   `(An)+`-only — `decodePSAVE`/`decodePRESTORE` in `pmmu.go`), the one difference being PMMU's word1
   literals (`0xF100`/`0xF140`) are used exactly as GAS's own table lists them, with no
   `fpuWord1Base`-style coprocessor-ID adjustment (PMMU's word1 never carries one). All 4 test cases
   passed round-trip on the first try. ~~The 68040's own single-word `PFLUSHA`/`PFLUSHAN`/`PFLUSHN`/
   `PFLUSH`/`PTESTR`/`PTESTW` forms~~ — also done, in a new `pmmu040.go`: a simplified, re-encoded
   interface distinct from 68030/68851's two-word coprocessor forms — a single fixed opcode word, at
   most an address register in bits 2-0, no `<ea>` mode field, no coprocessor command word2 at all.
   `PFLUSHA`/`PFLUSHN`/`PFLUSH` share a mnemonic with their existing two-word 68030/68851 counterparts
   but never collide at the bit level (word1 `0xF500`-`0xF56F` here vs. `0xF000`-prefixed there), so both
   opcode-table patterns simply coexist. This is also the **first PMMU pattern with real `CPU`-tier
   gating**: added `cpu040up` (`internal/decoders/cpu.go`, `cpu040|cpu060`) and `mmuExactCPU`/
   `mmuMaskedCPU` (the `RequiresMMU`-plus-specific-`cpuSet` analogue of `mmuExact`/`mmuMasked`, which
   until now always used `cpuAll`) — `PFLUSHA`/`PFLUSHAN`/`PFLUSHN`/`PFLUSH` are tagged `cpu040up`,
   `PTESTR`/`PTESTW` tagged bare `cpu040` only (the 68060 dropped them, matching GAS's own `m68040up` vs.
   `m68040` opcode-table tier tags). All 7 round-trip cases and both negative CPU-gating cases (the
   68040-only encoding not recognized on `M68030`, `PTESTR`'s form not recognized on `M68060`) passed on
   the first try. Only `PVALID` remains open — not yet located in m68kasm's own coverage; may need
   datasheet verification independent of the encoder-as-ground-truth approach used everywhere else in
   this doc. Full `CPU`-tier gating for the rest of the PMMU surface (68851 usable with 68020/68030;
   confirm which instructions the 68030's own built-in PMMU actually implements vs. requiring an
   external 68851) remains unapplied — every other PMMU pattern is still tagged `cpuAll`.
10. ~~**PMMU condition/branch PR**~~ — done: `PBcc` (word/long displacement), `PDBcc`, `PScc`, `PTRAPcc`
    (bare/word/long), in a new `pmmu_cond.go`. Exactly as predicted, a near-mechanical adaptation of
    step 7's `FBcc`/`FDBcc`/`FScc`/`FTRAPcc` code: 16 conditions instead of 32 (`pmmuConditions`, copied
    verbatim from m68kasm's own table), word1 base `0xF0xx` instead of `0xF2xx` (no coprocessor-ID bit
    to fold in), and reusing `branchTarget` (`branch.go`) for the target-address math — the same helper
    that fixed the real `.W`/`.L`/`DBcc` bug found while building step 7. `PDBcc`/`PTRAPcc` occupy `PScc`'s
    own EA sub-slots exactly like their FPU (and integer) counterparts, so they're registered ahead of it
    in `opcodetable.go` for the same reason. All 12 test cases passed round-trip on the first try — the
    only PMMU sub-step so far with zero surprises of any kind, reflecting how directly step 7's already-
    verified code and bug-fix carried over.

    **This completes every mainstream 68851/68030 PMMU instruction this doc scoped in — steps 9 and 10
    are both done.** Two items remain open, each already flagged above as its own follow-up rather than
    blocking this doc's completion: the 68040's own simplified single-word PMMU forms, and `PVALID`
    (not yet located in m68kasm's own coverage — would need direct datasheet verification, unlike
    everything else in this doc). `CPU`-tier gating (which PMMU instructions the 68030's own built-in
    PMMU actually implements vs. requiring an external 68851) also remains unapplied, per the note above.

Each step that touches actual opmode/mode-field bit values (5, 6, 7, 9, 10) should cite its source
(PRM section/page, or the specific `binutils`/`vasm` table entry cross-checked) in the PR description —
this is the FPU/MMU-specific analogue of the base-CPU doc's "must precede" precedence comments: a
place where silent correctness bugs are easy to introduce and hard to notice without a fixture corpus.

## Testing strategy

- Same table-driven-per-CPU pattern as `internal/decoders/types_test.go`, with an added dimension for
  `FPU`/`MMU` flags: positive cases (opcode decodes correctly with the flag on) and negative cases
  (same bytes render as `DC.W` with the flag off, and on `CPU`s where the built-in/discrete-chip
  combination doesn't apply, e.g. `MMU: true` PMMU opcodes on `M68040`).
- Build a hand-verified fixture corpus specifically for the FP data-format/k-factor/extended-precision
  formatting path — this is new numeric-formatting surface area (96-bit values) the integer decoders
  never had to handle, so it needs its own coverage independent of opcode-bit correctness.
- Cross-check against real compiler output where possible: `gcc -m68881`/`-m68040` and `gas`
  (`m68k-elf-as`) can generate known-good FPU object code from small C snippets with floating-point
  arithmetic, giving a low-effort source of realistic (not just hand-crafted) FPU instruction streams
  to disassemble and compare against `objdump -d`.
- For `MOVEC`, extend the existing `MOVEC` tests with the new 68040/68060 register codes — this is
  low-risk table data, not new decode logic, so coverage is mostly "does the map have the right string
  for the right code," not bit-math verification.

## Open questions for the user

1. Priority: is 68040/68060 (built-in FPU + `MOVEC`-based MMU, one combined real-world target) the
   actual motivating use case, making steps 1-5 the priority and discrete-chip/PMMU (steps 8-9) lower
   priority or skippable? Or is 68020/68030 + discrete 68881/68851 (common in Amiga/Sun/NeXT-era
   binaries) equally or more important?
2. Should `FSAVE`/`FRESTORE`/`PSAVE`/`PRESTORE` frame-shell decoding (step 7) be in scope at all, given
   it's rare in ordinary application-level disassembly (mostly OS context-switch code) — or should it
   be dropped entirely rather than deferred, to keep scope tighter?
3. Is a primary source (Motorola/NXP MC68881/68882 and MC68851 Programmer's Reference Manuals, or a
   specific existing open-source disassembler's opcode table) already available to cross-check bit
   encodings against, or does sourcing one become a blocking prerequisite before step 3 can start?
