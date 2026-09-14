# Design: Disassembly Labels

## Status

**Implemented.** All 5 delivery-sequence steps are done: the ELF options-plumbing prerequisite,
`LabelOptions`/`DecodeOptions.Labels`/`Instruction.Label`, the full two-pass `applyLabels`
implementation for the `Bcc`/`BSR`/`DBcc`/`FBcc`/`FDBcc`/`PBcc`/`PDBcc` branch family plus
`JSR`/`JMP`/`PEA`/`LEA` classification, universal rendering (which needed no dedicated work at all — see
step 2's note below), and `String()`'s label-line formatting. This was a new, independent feature — it
did not follow on from [design-cpu-variants.md](design-cpu-variants.md) or
[design-fpu-mmu.md](design-fpu-mmu.md) and had no dependency on either being finished (both happened to
be done at the time, but nothing here required that).

## Current state

- [`DecodeOptions.Symbolizer`](../public_types.go) exists today, but it only *resolves* addresses a
  caller already has names for — it never invents a name. `formatOperand`
  ([disasm.go:214](../disasm.go#L214)) tries, in order: `Operand.BranchTarget`, then
  `EffectiveAddress.ResolvedAddress`, then `EffectiveAddress.AbsoluteAddress`, calling
  `symbolizer.Symbolize(addr)` at each step and falling through to the raw `Operand.Text` (e.g.
  `$00001234`) if the symbolizer returns `ok: false` or none of those fields are set. This is gated
  entirely on `opts.Symbolizer != nil` ([disasm.go:199](../disasm.go#L199)) — with no `Symbolizer`, every
  address renders as raw hex, always, with no way for the library itself to give a within-range branch
  target a readable name.
- `DisassembleRangeWithOptions` ([disasm.go:84](../disasm.go#L84)) is a single sequential pass: decode
  one instruction, advance by its `Size`, repeat. It returns only `([]Instruction, error)` — nothing
  about the pass carries forward or backward between instructions, and nothing outside `disasm.go` ever
  sees the full set of instructions until the whole call returns. There is exactly one precedent for
  "know something before you need it" in this codebase — the `NeedMoreError` retry loop
  ([disasm.go:128-154](../disasm.go#L128)) — and it is a strictly *intra*-instruction refetch (more
  bytes for the instruction currently being decoded), not a cross-instruction mechanism. A labels
  feature is the first thing in this codebase that needs to know about instructions the decoder hasn't
  reached yet (a forward branch to code later in the stream).
- `Instruction` ([disasm.go:14](../disasm.go#L14)) has no field to hang a label definition on, and
  `String()` ([disasm.go:34](../disasm.go#L34)) renders only `"%08X: %s"` — there is nowhere today for
  "this address is the start of something worth naming" to live.
- Branch/call target addresses already surface in exactly two shapes, and they are not equally easy to
  work with:
  - **`Operand.BranchTarget`** (`Kind: OperandKindBranchTarget`), set uniformly by `branchOperand()`
    ([internal/decoders/common.go:255](../internal/decoders/common.go#L255)) for every
    conditional/unconditional branch and decrement-branch family this project decodes: `Bcc`/`BRA`/`BSR`
    ([internal/decoders/branch.go:64](../internal/decoders/branch.go#L64)), `DBcc` (same file, operand
    index 1), `FBcc`/`FDBcc` ([internal/decoders/fpu.go](../internal/decoders/fpu.go)), `PBcc`/`PDBcc`
    ([internal/decoders/pmmu_cond.go](../internal/decoders/pmmu_cond.go)). The target address itself is
    always computed by the shared `branchTarget()` helper
    ([internal/decoders/branch.go:28](../internal/decoders/branch.go#L28)) — `address + 2 + disp`,
    uniformly, regardless of displacement width — so every branch-shaped instruction already reports its
    target the same way, with no per-family special-casing needed to collect them.
  - **`EffectiveAddress.ResolvedAddress`/`AbsoluteAddress`** (`Kind: OperandKindEffectiveAddr`), set for
    *any* instruction whose operand is an absolute or PC-relative `<ea>` — including `JSR`/`JMP`
    ([internal/decoders/branch.go:137-144](../internal/decoders/branch.go#L137), via `decodeUnaryEA` →
    `decodeEA` → `resolveEffectiveAddress`) and `PEA` (same path,
    [internal/decoders/special.go:50](../internal/decoders/special.go#L50)) and `LEA`'s own source
    operand ([internal/decoders/special.go:38](../internal/decoders/special.go#L38) — the source `<ea>`
    is structured operand index 0, the destination `An` register is index 1 and never a candidate) — but
    just as much for a plain `MOVE.L $1234,D0`. **None of these are distinguishable from a data
    reference at the operand-kind level** — only the *instruction's own mnemonic* says "this `<ea>` is a
    control-flow target or a computed address," which means classifying `JSR`/`JMP`/`PEA`/`LEA` targets
    needs `Metadata.MnemonicBase`, not just operand shape.
  - `Metadata.BranchTarget` ([internal/decoders/types.go:133](../internal/decoders/types.go#L133))
    mirrors only the *first* branch operand a decoder happened to populate — convenient for callers who
    just want "the" target of a simple branch, but lossy, so a label-collecting pass must walk
    `Metadata.Operands` directly rather than relying on this shortcut.
- ~~`ELFDisassembler.DisassembleSection` takes no `DecodeOptions` at all~~ — **fixed** as delivery-sequence
  step 1: `DisassembleSectionWithOptions`/`DisassembleAllExecutableSectionsWithOptions` now exist (see
  "The ELF gap this depends on" below). This was a real, pre-existing gap independent of labels — it
  predated this doc and already meant `CPU`/`FPU`/`MMU` selection was unreachable on ELF input — closed
  first so the rest of this feature has somewhere to reach ELF callers from.

## Goals

- After a `DisassembleRange`/`DisassembleRangeWithOptions` call with labels enabled, every branch/call
  target address that (a) falls within the disassembled instruction stream and (b) lands exactly on some
  decoded instruction's own address gets a synthetic name (e.g. `l00001010`) whenever no
  caller-supplied `Symbolizer` name already covers it.
- Every instruction whose own address *is* such a target carries that name somewhere machine-readable
  (a new `Instruction.Label` field) — so a caller can render a label line, build a cross-reference view,
  or do anything else with it, without re-deriving the target set themselves.
- Every operand referencing a labeled address — not just the branch/`JSR`/`JMP` operand that first
  justified the label's existence — renders that name instead of raw hex, through the *same*
  `Symbolizer`-driven rendering path already used for caller-supplied names (`formatOperand`'s existing
  precedence chain), not a parallel one built just for this feature.
- Precedence: caller-supplied `Symbolizer` name wins over a synthetic label, which wins over raw hex —
  a caller's own naming always takes priority, exactly as `Symbolizer` already behaves when it returns
  `ok: false` for something else that then falls through.
- Zero behavior change, zero extra cost, when the feature is off (the default). This is an opt-in
  second pass layered on top of the existing single-pass decode, not a rewrite of it.
- Extend labeling to `JSR`/`JMP`/`PEA`/`LEA` call/jump/address-computation targets — classified via
  `MnemonicBase`, not every absolute/PC-relative operand of every instruction — so that a target address
  is only *created* as a label by a genuine control-flow or address-computation reference. `PEA`/`LEA`
  are included specifically to catch indirect-call trampolines (`LEA sub,A0` then `JSR (A0)`), even
  though this also means a `LEA`/`PEA` loading a plain data-buffer address becomes a label too — an
  accepted trade-off, not an oversight (see Decisions below). A plain `MOVE.L $1234,D0` still never
  creates a label on its own, but once `$1234` is a label for some other reason, that same `MOVE.L`
  renders it too (see "Classification creates a label; rendering is universal" below) — this is what
  keeps a real function entry point readable everywhere it's mentioned, without inventing a label for
  every data address the code happens to touch.

## Non-goals

- No jump-table or computed-jump target discovery. `JMP (A0,D1.L*4)`-style indexed or indirect jumps
  have no statically knowable target — out of scope, the same "no data-flow analysis" boundary
  [design-cpu-variants.md](design-cpu-variants.md) and [design-fpu-mmu.md](design-fpu-mmu.md) already
  draw for themselves (cycle timing, exception semantics, coprocessor state-frame contents).
- No code/data classification and no recursive or exploratory disassembly. Labels only ever attach to
  addresses that coincide with an instruction this library *already decided* to decode as part of the
  caller's requested range — the library never decides "something jumps here, so decode from here too."
- No cross-reference ("referenced from N places") tracking, and no export of a symbol table format.
  `Instruction.Label` plus the rendered `Operands` string is the entire surface for this pass — a caller
  who wants an `address -> name` map can build one by scanning `Label` across the returned slice (see
  Decisions below).
- No persistence of labels across separate `DisassembleRange*` calls — labels are recomputed fresh every
  call, exactly like every other piece of decode state in this library.
- No customizable naming scheme beyond a name-prefix knob (`l` by default — see Decisions below) in this
  first pass, and no `sub_`/`loc_`-style split by reference kind. A fully pluggable naming strategy (a
  caller-supplied callback, for instance) is a plausible follow-up, not a blocker.
- No change to `Decode`/`DecodeWithOptions`/`DecodeReaderAt*`/`DecodeFunc*`. Labels are a range-level,
  two-pass concern; a single decoded instruction has no "rest of the stream" to find a forward reference
  in, so `DecodeOptions.Labels` is documented as a no-op on these entry points rather than silently
  pretending to be supported or rejected as an error.

## Proposed model

### Classification creates a label; rendering is universal

The central design decision, and the one that resolves the `JSR $1234` vs. `MOVE.L $1234,D0` ambiguity:
**which instructions can *create* a label is a narrow, mnemonic-aware classification step (`JSR`, `JMP`,
`PEA`, `LEA`), but once an address has a label, *every* operand referencing that address renders it —
regardless of the referencing instruction's own mnemonic.** A `JSR` a hundred bytes earlier in the same
range establishes that `$00001234` is `l00001234`; a `MOVE.L $00001234,D0` elsewhere in the same range
then renders `MOVE.L l00001234,D0` automatically, without `MOVE` needing its own classification rule. This mirrors
how a real assembly listing behaves — once a location has a name, every mention of it uses that name —
and keeps the label-*creating* rule (the part that must avoid false positives) small and auditable,
independent of the label-*rendering* rule (which should be as inclusive as possible once a name exists).

### Two-pass architecture inside `DisassembleRangeWithOptions`

**Pass 1 (existing, unchanged):** the current sequential decode loop, producing the full `[]Instruction`
slice with complete `Metadata`, exactly as today. No decoder-level changes at all — labels are entirely
a `disasm.go`-level concern; `internal/decoders` needs no new code or awareness of this feature.

**Pass 2 (new, only runs when `DecodeOptions.Labels` is non-nil):**

1. **Collect label candidates** by walking each instruction's `Metadata.Operands`:
   - Any `Operand` with `Kind == OperandKindBranchTarget` contributes its `BranchTarget` value
     unconditionally. This alone covers `Bcc`/`BRA`/`BSR`/`DBcc`/`FBcc`/`FDBcc`/`PBcc`/`PDBcc` — and any
     future branch-shaped family — with no mnemonic-string matching, since `branchOperand()` already
     tags every one of them identically.
   - Any `Operand` with `Kind == OperandKindEffectiveAddr`, belonging to an instruction whose
     `Metadata.MnemonicBase` is `"JSR"`, `"JMP"`, `"PEA"`, or `"LEA"`, contributes its `ResolvedAddress`
     if set, else its `AbsoluteAddress` if set (the PC-relative and absolute `<ea>` forms respectively).
     For `LEA` this only ever matches its source `<ea>` operand (index 0); the destination `An` register
     (index 1) is `OperandKindRegister` and never a candidate. This is the only mnemonic-gated rule
     needed.
2. **Keep only candidates that land on a decoded instruction.** Build `instByAddr map[uint32]int` from
   the pass-1 slice; a candidate address with no entry (self-modifying code, data-in-code, an address
   outside the requested range, or simply a target that fell inside a gap a partial/truncated decode
   left undecoded) is dropped — it stays raw hex rather than becoming a label with nothing to attach a
   definition line to.
3. **Name each surviving address deterministically** from the address alone —
   `fmt.Sprintf("%s%08X", prefix, address)` — never from a sequential counter, so output never depends
   on iteration order and is stable across reruns or overlapping sub-ranges.
4. **Attach labels**: for every instruction at a named address, set `Instruction.Label` to that name.
5. **Re-render every instruction's `Operands`** via the existing `formatOperands`/`formatOperand` path
   (unchanged internally, see below), but with a composite `Symbolizer` that tries the caller's own
   `Symbolizer` first (if any) and falls back to the synthetic label table. Because this re-render walks
   *every* operand exactly as it does for `Symbolizer` today, it naturally implements "rendering is
   universal": any operand anywhere in the range whose resolved/absolute/branch address matches a
   labeled address picks up the name, not only the operand that originally justified creating it.

### Precedence and the Metadata-stays-raw invariant

The composite `Symbolizer` is the whole mechanism — no new rendering path, no new precedence logic
outside it:

```go
type labelSymbolizer struct {
    primary Symbolizer      // the caller's own, or nil
    labels  map[uint32]string
}

func (l labelSymbolizer) Symbolize(address uint32) (string, bool) {
    if l.primary != nil {
        if name, ok := l.primary.Symbolize(address); ok {
            return name, ok
        }
    }
    name, ok := l.labels[address]
    return name, ok
}
```

`formatOperand`/`formatOperands` ([disasm.go:206-233](../disasm.go#L206)) do not change at all — they
already accept a `Symbolizer` interface value, and `labelSymbolizer` is one. The gate at
[disasm.go:199](../disasm.go#L199) widens from `opts.Symbolizer != nil` to `opts.Symbolizer != nil ||
opts.Labels != nil`, constructing a `labelSymbolizer` (with `primary: opts.Symbolizer`, possibly nil)
whenever either is set.

`Instruction.Metadata.Operands[i].Text` is never touched by any of this — it stays the raw `$00001234`
text exactly as decoded, matching the invariant the README already documents for `Symbolizer`
(`Metadata.Operands[0].Text` stays raw even when `Assembly()` shows a resolved symbol). Labels are
additive to `Instruction.Operands` (the rendered string) and the new `Instruction.Label` field only.

### API shape

```go
// public_types.go
type DecodeOptions struct {
    Symbolizer Symbolizer
    CPU        CPU
    FPU        bool
    MMU        bool
    // Labels enables synthetic label generation for branch/call targets
    // that fall within a disassembled range and land on a decoded
    // instruction. Nil (the zero value) preserves prior behavior — no
    // labels, no rendering change, no extra cost. Meaningful only for
    // DisassembleRange/DisassembleRangeWithOptions (and, once added,
    // DisassembleSectionWithOptions); silently has no effect on
    // Decode/DecodeWithOptions/DecodeReaderAt*/DecodeFunc*, which have no
    // "rest of the stream" to find a forward reference in.
    Labels *LabelOptions
}

type LabelOptions struct {
    // Prefix is prepended to the 8-hex-digit address to form a synthetic
    // label name (e.g. "l" -> "l00001010"). Defaults to "l" when Labels
    // is non-nil but Prefix is empty.
    Prefix string
}
```

A pointer (not a bare `bool`) makes "labels on, defaults otherwise" a one-line `&m68kdasm.LabelOptions{}`
while leaving room for `Prefix` without a second option parameter or a breaking signature change later.

```go
// disasm.go
type Instruction struct {
    Address        uint32
    Opcode         uint16
    Mnemonic       string
    Operands       string
    Size           uint32
    Bytes          []byte
    ExtensionWords []uint16
    Metadata       DecodeMetadata
    // Label is the synthetic or caller-provided name for this
    // instruction's own address, when DecodeOptions.Labels was set and
    // something in the disassembled range targets this address. Empty
    // otherwise.
    Label string
}
```

`String()` gains a label line when `Label` is non-empty (exact formatting is a review-time judgment
call, not load-bearing to this design):

```go
func (i Instruction) String() string {
    if i.Label != "" {
        return fmt.Sprintf("%s:\n%08X: %s", i.Label, i.Address, i.Assembly())
    }
    return fmt.Sprintf("%08X: %s", i.Address, i.Assembly())
}
```

### The ELF gap this depended on — closed

`ELFDisassembler.DisassembleSection` ([elf.go:41](../elf.go#L41)) had no `DecodeOptions` parameter at
all — it called the no-options `DisassembleRange` ([elf.go:111](../elf.go#L111)). This predated labels
and already blocked `CPU`/`FPU`/`MMU` selection on ELF input; it also would have meant labels were
unreachable from ELF. Closed as delivery-sequence step 1:

```go
func (ed *ELFDisassembler) DisassembleSectionWithOptions(sectionName string, opts DecodeOptions) ([]Instruction, error)
func (ed *ELFDisassembler) DisassembleAllExecutableSectionsWithOptions(opts DecodeOptions) (map[string][]Instruction, error)
```

mirroring the `Xxx`/`XxxWithOptions` pairing already used everywhere else in this package
(`Decode`/`DecodeWithOptions`, `DisassembleRange`/`DisassembleRangeWithOptions`, etc.), with the existing
no-options `DisassembleSection`/`DisassembleAllExecutableSections` becoming thin wrappers calling them
with `DecodeOptions{}` — the same relationship `Decode` already has to `DecodeWithOptions`. The shared
internal `disassembleSection` helper both public methods call now threads `opts` through to
`DisassembleRangeWithOptions` instead of hardcoding `DisassembleRange`.

## Migration / call-site impact

Additive only, no breaking changes:

- `DecodeOptions.Labels *LabelOptions` — new field, zero value `nil`; no existing caller is affected.
- `Instruction.Label string` — new field, zero value `""`; `String()`'s output changes only when `Label`
  is non-empty, which only happens when a caller explicitly opts in via `Labels`.
- `formatOperand`/`formatOperands` gain the composite-`Symbolizer` indirection internally; both are
  unexported, and a nil-`Labels` composite degenerates to exactly today's single-`Symbolizer` path, so
  existing `Symbolizer`-only callers see no behavior change.
- `ELFDisassembler` gained two new methods (step 1, already landed);
  `DisassembleSection`/`DisassembleAllExecutableSections`' existing signatures and behavior are
  unchanged.

Every existing test should pass unmodified; nothing here touches `internal/decoders`.

## Suggested delivery sequence

1. ~~**ELF options-plumbing PR**~~ — done: added `DisassembleSectionWithOptions` and
   `DisassembleAllExecutableSectionsWithOptions`, both mirroring the `Xxx`/`XxxWithOptions` pairing used
   everywhere else in this package; the existing no-options methods now call them with `DecodeOptions{}`.
   `disassembleSection` (the shared internal helper both call) threads `opts` through to
   `DisassembleRangeWithOptions` instead of hardcoding `DisassembleRange`. Verified with a real FPU
   opcode assembled into an ELF section (`TestELFDisassembleSectionWithOptions`,
   `example_elf_test.go`): unrecognized (`DC.W`) via the old no-options method, correctly decoded via
   the new `WithOptions` ones — confirming `DecodeOptions` genuinely reaches ELF-sourced disassembly now,
   not just that the new methods compile.
2. ~~**Plumbing PR**~~ — done: `LabelOptions`, `DecodeOptions.Labels`, `Instruction.Label`, and the pass-2
   implementation (`applyLabels`/`labelSymbolizer`, `disasm.go`), wired into `DisassembleRangeWithOptions`
   after the existing pass-1 loop. Collects only `OperandKindBranchTarget` candidates
   (`Bcc`/`BSR`/`DBcc`/`FBcc`/`FDBcc`/`PBcc`/`PDBcc`) — the zero-mnemonic-matching case, so this step
   touched no instruction-family-specific logic. One real correction made before shipping: `Instruction.Label`
   was first implemented as always the *synthetic* name, but that's inconsistent with operand rendering's
   own Symbolizer-first precedence — a target address the caller's `Symbolizer` already names should
   report that same name via `Label`, not silently prefer a synthetic one nobody's operand text actually
   uses. Fixed to route `Label` through the identical `labelSymbolizer.Symbolize` call operand rendering
   uses, caught by `TestLabelsWithSymbolizer` before merging. Also confirmed empirically
   (`TestLabelsNoOpOnSingleDecode`) that `Decode`/`DecodeWithOptions` need no code change at all to honor
   the "silently ignored" contract — an unread struct field is a no-op by construction in Go.
   **~~Universal-rendering (step 4)~~ turned out to already be built-in**, not a separate step: because
   `applyLabels` re-renders every instruction's operands unconditionally (not gated by which instruction
   created the label), a plain `MOVE.L $addr,D0` referencing an address only a `BRA` elsewhere targets
   already renders the label, confirmed by `TestLabelsUniversalRendering` — with none of step 3's
   `JSR`/`JMP`/`PEA`/`LEA` classification landed yet. Step 4 below is now just documentation of an
   already-verified property, not remaining work.
3. ~~**`JSR`/`JMP`/`PEA`/`LEA` PR**~~ — done: `labelCreatingMnemonics` (a plain `map[string]bool`,
   `disasm.go`) gates `OperandKindEffectiveAddr` candidate collection to these four mnemonics via
   `Metadata.MnemonicBase`, contributing `ResolvedAddress` if set else `AbsoluteAddress` — exactly the
   rule this doc specified, no surprises. Verified: `JSR`/`JMP`/`PEA` to an absolute address inside the
   range get labeled (and outside the range, correctly don't); the `LEA sub,A0` / `JSR (A0)` trampoline
   pattern labels `sub`'s address while leaving `JSR (A0)` itself untouched (a computed jump through a
   register contributes no candidate — `ResolvedAddress`/`AbsoluteAddress` are both nil for
   register-indirect addressing, so the classification rule naturally does nothing there, no special
   case needed); and a `LEA` loading a plain data-buffer address gets labeled too, confirming Decision 1's
   accepted trade-off actually behaves as decided, not just as described.
4. ~~**Universal-rendering PR**~~ — see step 2's note: this fell out of the pass-2 implementation for
   free and needs no dedicated work.
5. ~~**`String()`/formatting PR`**~~ — done: landed exactly the convention this doc's API-shape section
   proposed (`"%s:\n%08X: %s"` — the label on its own line, ahead of the usual `address: assembly`
   line), unchanged after seeing steps 2-4's actual output. An unlabeled instruction's `String()` is
   byte-for-byte identical to its pre-labels rendering — verified directly
   (`TestInstructionStringWithoutLabelsUnchanged`), not just assumed from the `if i.Label != ""` guard.

Each step is independently mergeable and testable; nothing later depends on a choice made in an earlier
step beyond what is already fixed by this document.

## Testing strategy

Unlike [design-fpu-mmu.md](design-fpu-mmu.md), this feature needs no external assembler as a
ground-truth oracle — labels are a pure post-processing layer over already-decoded output, so tests can
hand-construct byte sequences the way the existing hand-crafted-bytes tests already do (e.g.
`TestDecodeBranchInstructions` in `disasm_test.go`). Cases to cover:

- A forward branch to a later instruction in the same range gets a label at the target, and the branch
  operand renders the label name.
- A backward branch (loop) to an earlier instruction: same.
- Two different branches to the same target: exactly one label, both operands render it (map-based
  dedup, verified rather than assumed).
- A `JSR`/`JMP` to an absolute address inside the range: labeled. To an address outside the range: left
  as raw hex.
- A `LEA sub(PC),A0` followed elsewhere by `JSR (A0)`: `sub`'s address gets a label — the indirect-call
  trampoline case Decision 1 (below) exists to catch — even though the `JSR (A0)` itself can never be
  traced back to it (a computed jump through a register has no statically knowable target; out of scope
  per Non-goals).
- A `LEA buffer,A0` (or `PEA`) whose address is never otherwise a branch/`JSR`/`JMP` target: still gets a
  label. This documents the accepted trade-off from Decision 1 — a plain data-buffer address loaded via
  `LEA`/`PEA` reads as a label too, not only genuine subroutine entries — as a deliberate scope choice,
  not a bug to fix later.
- A `MOVE.L $addr,Dn` referencing the same address a `JSR` elsewhere in the range targets: renders the
  label too — confirms an address needs only one control-flow reference to become labeled everywhere it
  appears (the "rendering is universal" behavior, step 4 above).
- A `MOVE.L $addr,Dn` referencing an address nothing ever jumps to: stays raw hex — a data reference
  alone never creates a label.
- A branch target that does not land on a decoded instruction boundary (mid-instruction, or past the end
  of a truncated/partial decode result): stays raw hex, no panic, no dangling label.
- `Symbolizer` and `Labels` both set: the `Symbolizer`'s name wins for any address it recognizes; the
  synthetic label fills in only for addresses the `Symbolizer` returns `ok: false` for.
- `Decode`/`DecodeWithOptions` with a non-nil `Labels`: no error, no label produced — documents the
  no-op explicitly rather than leaving it unspecified.
- `DisassembleRange` (no-options): completely unaffected; `Instruction.Label` is always `""`.

## Decisions

Four open questions were resolved before implementation starts, rather than left for whoever picks this
up:

1. **`LEA`/`PEA` are label-creating**, alongside `JSR`/`JMP` — see "Classification creates a label;
   rendering is universal" above. Accepted trade-off: a `LEA`/`PEA` loading a plain data-buffer address
   also becomes a label, not only genuine subroutine entries reached via an indirect-call trampoline
   (`LEA sub,A0` then `JSR (A0)`) — judged worth it since missing a real subroutine entry point is a
   worse failure mode for a disassembler than one extra label on a data address.
2. **A single naming prefix, not a `sub_`/`loc_`-style split by reference kind.** Default prefix: `l`
   (e.g. `l00001010`) — short, and consistent with the terse `$00001010`-style hex text
   `formatBranchTarget` already renders today; `LabelOptions.Prefix` remains the escape hatch for a
   caller who wants something else.
3. **No richer return shape than `Instruction.Label` plus the rendered `Operands` string** in this first
   pass — no separate `address -> name` map. A caller who wants one can build it by scanning `Label`
   across the returned instruction slice.
4. **The ELF options-plumbing prerequisite is folded into this effort**, as delivery-sequence step 1,
   rather than filed and timed as a fully separate PR.
