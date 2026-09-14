package m68kdasm

import "testing"

// These tests hand-construct byte sequences rather than going through
// m68kasm: labels are a pure post-processing layer over already-decoded
// output (see docs/design-labels.md's Testing strategy), so there's no
// need for an external assembler as ground truth here — only for the
// decoder output the labels pass consumes, which is already verified
// elsewhere.

// TestLabelsForwardBranch covers the basic case: a forward branch to a
// later instruction in the same range gets a label at the target, and the
// branch operand renders the label name instead of raw hex.
func TestLabelsForwardBranch(t *testing.T) {
	data := []byte{
		0x60, 0x02, // BRA.S $1004 (disp8=2, target = 0x1000+2+2)
		0x4E, 0x71, // NOP
		0x4E, 0x75, // RTS
	}

	instrs, err := DisassembleRangeWithOptions(data, 0x1000, DecodeOptions{Labels: &LabelOptions{}})
	if err != nil {
		t.Fatalf("DisassembleRangeWithOptions: %v", err)
	}
	if len(instrs) != 3 {
		t.Fatalf("expected 3 instructions, got %d", len(instrs))
	}

	if instrs[2].Label != "l00001004" {
		t.Errorf("expected RTS at 0x1004 to carry label %q, got %q", "l00001004", instrs[2].Label)
	}
	if instrs[0].Assembly() != "BRA.S l00001004" {
		t.Errorf("expected BRA operand to render the label, got %q", instrs[0].Assembly())
	}
	if instrs[1].Label != "" {
		t.Errorf("NOP should carry no label, got %q", instrs[1].Label)
	}
}

// TestLabelsBackwardBranch covers a loop: a backward branch to an earlier
// instruction labels that earlier instruction.
func TestLabelsBackwardBranch(t *testing.T) {
	data := []byte{
		0x4E, 0x71, // NOP           (0x1000)
		0x60, 0xFC, // BRA.S $1000   (0x1002, disp8=-4, target = 0x1002+2-4)
	}

	instrs, err := DisassembleRangeWithOptions(data, 0x1000, DecodeOptions{Labels: &LabelOptions{}})
	if err != nil {
		t.Fatalf("DisassembleRangeWithOptions: %v", err)
	}

	if instrs[0].Label != "l00001000" {
		t.Errorf("expected NOP at 0x1000 to carry label %q, got %q", "l00001000", instrs[0].Label)
	}
	if instrs[1].Assembly() != "BRA.S l00001000" {
		t.Errorf("expected BRA operand to render the label, got %q", instrs[1].Assembly())
	}
}

// TestLabelsDedup covers two different branches to the same target: exactly
// one label, both operands render it.
func TestLabelsDedup(t *testing.T) {
	data := []byte{
		0x67, 0x04, // BEQ.S $1006   (0x1000, disp8=4)
		0x66, 0x02, // BNE.S $1006   (0x1002, disp8=2)
		0x4E, 0x71, // NOP           (0x1004)
		0x4E, 0x75, // RTS           (0x1006)
	}

	instrs, err := DisassembleRangeWithOptions(data, 0x1000, DecodeOptions{Labels: &LabelOptions{}})
	if err != nil {
		t.Fatalf("DisassembleRangeWithOptions: %v", err)
	}

	if instrs[3].Label != "l00001006" {
		t.Errorf("expected RTS at 0x1006 to carry label %q, got %q", "l00001006", instrs[3].Label)
	}
	if instrs[0].Assembly() != "BEQ.S l00001006" {
		t.Errorf("BEQ operand mismatch: %q", instrs[0].Assembly())
	}
	if instrs[1].Assembly() != "BNE.S l00001006" {
		t.Errorf("BNE operand mismatch: %q", instrs[1].Assembly())
	}
}

// TestLabelsTargetNotOnInstructionBoundary confirms a branch target that
// doesn't land on a decoded instruction's own address (here, the middle of
// a multi-word MOVE.L) stays raw hex rather than becoming a dangling label.
func TestLabelsTargetNotOnInstructionBoundary(t *testing.T) {
	data := []byte{
		0x20, 0x3C, 0x12, 0x34, 0x56, 0x78, // MOVE.L #$12345678,D0 (0x1000-0x1005)
		0x60, 0xFA, // BRA.S $1002 (0x1006, disp8=-6, target = 0x1006+2-6 = 0x1002 — mid-instruction)
	}

	instrs, err := DisassembleRangeWithOptions(data, 0x1000, DecodeOptions{Labels: &LabelOptions{}})
	if err != nil {
		t.Fatalf("DisassembleRangeWithOptions: %v", err)
	}
	if len(instrs) != 2 {
		t.Fatalf("expected 2 instructions, got %d", len(instrs))
	}

	if instrs[1].Assembly() != "BRA.S $1002" {
		t.Errorf("expected raw hex for an off-boundary target, got %q", instrs[1].Assembly())
	}
	for i, inst := range instrs {
		if inst.Label != "" {
			t.Errorf("instruction %d unexpectedly carries a label %q", i, inst.Label)
		}
	}
}

// TestLabelsWithSymbolizer confirms precedence: a caller-supplied
// Symbolizer wins for any address it recognizes; the synthetic label fills
// in only for addresses the Symbolizer returns ok: false for.
func TestLabelsWithSymbolizer(t *testing.T) {
	data := []byte{
		0x67, 0x04, // BEQ.S $1006   (0x1000)
		0x66, 0x04, // BNE.S $1008   (0x1002)
		0x4E, 0x71, // NOP           (0x1004)
		0x4E, 0x71, // NOP           (0x1006)
		0x4E, 0x75, // RTS           (0x1008)
	}

	symbolizer := SymbolizeFunc(func(address uint32) (string, bool) {
		if address == 0x1006 {
			return "_named_target", true
		}
		return "", false
	})

	instrs, err := DisassembleRangeWithOptions(data, 0x1000, DecodeOptions{
		Symbolizer: symbolizer,
		Labels:     &LabelOptions{},
	})
	if err != nil {
		t.Fatalf("DisassembleRangeWithOptions: %v", err)
	}

	if instrs[0].Assembly() != "BEQ.S _named_target" {
		t.Errorf("expected the Symbolizer's own name to win, got %q", instrs[0].Assembly())
	}
	if instrs[1].Assembly() != "BNE.S l00001008" {
		t.Errorf("expected a synthetic label where the Symbolizer returned ok=false, got %q", instrs[1].Assembly())
	}
	// Label uses the same Symbolizer-first precedence as operand
	// rendering, so a Symbolizer-resolved target's Label is the
	// Symbolizer's own name, not a separate synthetic one.
	if instrs[3].Label != "_named_target" {
		t.Errorf("expected Label to carry the Symbolizer's own name, got %q", instrs[3].Label)
	}
	if instrs[4].Label != "l00001008" {
		t.Errorf("expected a synthetic Label at the non-Symbolized target, got %q", instrs[4].Label)
	}
}

// TestLabelsUniversalRendering confirms an address only needs one
// control-flow reference to become labeled everywhere it's mentioned in
// the range — a plain data reference (here, an absolute MOVE.L operand,
// not yet a label-creating mnemonic per step 3 of the delivery sequence)
// to the same address a BRA targets picks up the label too. This step (4
// in docs/design-labels.md's delivery sequence) falls out of applyLabels'
// existing "re-render every instruction, unconditionally" loop for free —
// nothing instruction-kind-specific was needed to get it.
func TestLabelsUniversalRendering(t *testing.T) {
	data := []byte{
		0x60, 0x04, // BRA.S $1006      (0x1000, disp8=4)
		0x20, 0x38, 0x10, 0x06, // MOVE.L $1006,D0 (0x1002, absolute short)
		0x4E, 0x75, // RTS              (0x1006)
	}

	instrs, err := DisassembleRangeWithOptions(data, 0x1000, DecodeOptions{Labels: &LabelOptions{}})
	if err != nil {
		t.Fatalf("DisassembleRangeWithOptions: %v", err)
	}
	if instrs[1].Assembly() != "MOVE.L l00001006, D0" {
		t.Errorf("expected the MOVE.L's data reference to render the label too, got %q", instrs[1].Assembly())
	}
}

// TestLabelsJSR covers a JSR to an absolute address inside the range
// (labeled) and outside the range (left as raw hex) — see step 3 of
// docs/design-labels.md's delivery sequence.
func TestLabelsJSR(t *testing.T) {
	t.Run("inside range", func(t *testing.T) {
		data := []byte{
			0x4E, 0xB8, 0x10, 0x06, // JSR $1006.W (0x1000-0x1003, absolute short)
			0x4E, 0x71, // NOP        (0x1004)
			0x4E, 0x75, // RTS        (0x1006)
		}
		instrs, err := DisassembleRangeWithOptions(data, 0x1000, DecodeOptions{Labels: &LabelOptions{}})
		if err != nil {
			t.Fatalf("DisassembleRangeWithOptions: %v", err)
		}
		if instrs[0].Assembly() != "JSR l00001006" {
			t.Errorf("expected JSR operand to render the label, got %q", instrs[0].Assembly())
		}
		if instrs[2].Label != "l00001006" {
			t.Errorf("expected RTS at 0x1006 to carry the label, got %q", instrs[2].Label)
		}
	})

	t.Run("outside range", func(t *testing.T) {
		data := []byte{
			0x4E, 0xB8, 0x20, 0x00, // JSR $2000.W (well outside this 4-byte range)
		}
		instrs, err := DisassembleRangeWithOptions(data, 0x1000, DecodeOptions{Labels: &LabelOptions{}})
		if err != nil {
			t.Fatalf("DisassembleRangeWithOptions: %v", err)
		}
		if instrs[0].Assembly() != "JSR $2000" {
			t.Errorf("expected raw hex for an out-of-range JSR target, got %q", instrs[0].Assembly())
		}
	})
}

// TestLabelsJMP mirrors TestLabelsJSR for JMP.
func TestLabelsJMP(t *testing.T) {
	data := []byte{
		0x4E, 0xF8, 0x10, 0x06, // JMP $1006.W (0x1000-0x1003, absolute short)
		0x4E, 0x71, // NOP       (0x1004)
		0x4E, 0x75, // RTS       (0x1006)
	}
	instrs, err := DisassembleRangeWithOptions(data, 0x1000, DecodeOptions{Labels: &LabelOptions{}})
	if err != nil {
		t.Fatalf("DisassembleRangeWithOptions: %v", err)
	}
	if instrs[0].Assembly() != "JMP l00001006" {
		t.Errorf("expected JMP operand to render the label, got %q", instrs[0].Assembly())
	}
}

// TestLabelsPEA covers PEA as a label-creating mnemonic.
func TestLabelsPEA(t *testing.T) {
	data := []byte{
		0x48, 0x78, 0x10, 0x06, // PEA $1006.W (0x1000-0x1003, absolute short)
		0x4E, 0x71, // NOP       (0x1004)
		0x4E, 0x75, // RTS       (0x1006)
	}
	instrs, err := DisassembleRangeWithOptions(data, 0x1000, DecodeOptions{Labels: &LabelOptions{}})
	if err != nil {
		t.Fatalf("DisassembleRangeWithOptions: %v", err)
	}
	if instrs[0].Assembly() != "PEA l00001006" {
		t.Errorf("expected PEA operand to render the label, got %q", instrs[0].Assembly())
	}
}

// TestLabelsLEATrampoline covers the indirect-call trampoline pattern
// (LEA sub,A0 then JSR (A0)) Decision 1 in docs/design-labels.md exists to
// catch: sub's address gets a label even though the JSR (A0) itself can
// never be traced back to it (a computed jump through a register has no
// statically knowable target).
func TestLabelsLEATrampoline(t *testing.T) {
	data := []byte{
		0x41, 0xF8, 0x10, 0x08, // LEA $1008.W,A0 (0x1000-0x1003, absolute short)
		0x4E, 0x90, // JSR (A0)        (0x1004, address-register indirect)
		0x4E, 0x71, // NOP             (0x1006)
		0x4E, 0x75, // RTS             (0x1008)
	}
	instrs, err := DisassembleRangeWithOptions(data, 0x1000, DecodeOptions{Labels: &LabelOptions{}})
	if err != nil {
		t.Fatalf("DisassembleRangeWithOptions: %v", err)
	}
	if instrs[0].Assembly() != "LEA l00001008, A0" {
		t.Errorf("expected LEA operand to render the label, got %q", instrs[0].Assembly())
	}
	if instrs[1].Assembly() != "JSR (A0)" {
		t.Errorf("JSR (A0) has no statically knowable target and should be untouched, got %q", instrs[1].Assembly())
	}
	if instrs[3].Label != "l00001008" {
		t.Errorf("expected RTS at 0x1008 to carry the label, got %q", instrs[3].Label)
	}
}

// TestLabelsLEADataBuffer documents the accepted trade-off from Decision 1:
// a LEA loading a plain data-buffer address — never itself a branch/JSR/
// JMP target — still gets a label. This is a deliberate scope choice, not
// a bug.
func TestLabelsLEADataBuffer(t *testing.T) {
	data := []byte{
		0x41, 0xF8, 0x10, 0x06, // LEA $1006.W,A0 (0x1000-0x1003, absolute short)
		0x4E, 0x71, // NOP             (0x1004)
		0x4E, 0x71, // "buffer" (any valid bytes; content is irrelevant to this test) (0x1006)
	}
	instrs, err := DisassembleRangeWithOptions(data, 0x1000, DecodeOptions{Labels: &LabelOptions{}})
	if err != nil {
		t.Fatalf("DisassembleRangeWithOptions: %v", err)
	}
	if instrs[0].Assembly() != "LEA l00001006, A0" {
		t.Errorf("expected the data-buffer LEA to be labeled too (accepted trade-off), got %q", instrs[0].Assembly())
	}
}

// TestLabelsCustomPrefix covers LabelOptions.Prefix.
func TestLabelsCustomPrefix(t *testing.T) {
	data := []byte{
		0x60, 0x02, // BRA.S $1004
		0x4E, 0x71, // NOP
		0x4E, 0x75, // RTS
	}

	instrs, err := DisassembleRangeWithOptions(data, 0x1000, DecodeOptions{
		Labels: &LabelOptions{Prefix: "loc_"},
	})
	if err != nil {
		t.Fatalf("DisassembleRangeWithOptions: %v", err)
	}
	if instrs[2].Label != "loc_00001004" {
		t.Errorf("expected custom-prefixed label, got %q", instrs[2].Label)
	}
}

// TestLabelsDisabledByDefault confirms DisassembleRange (no options) is
// completely unaffected: Instruction.Label is always empty and rendering
// is unchanged.
func TestLabelsDisabledByDefault(t *testing.T) {
	data := []byte{
		0x60, 0x02, // BRA.S $1004
		0x4E, 0x71, // NOP
		0x4E, 0x75, // RTS
	}

	instrs, err := DisassembleRange(data, 0x1000)
	if err != nil {
		t.Fatalf("DisassembleRange: %v", err)
	}
	if instrs[0].Assembly() != "BRA.S $1004" {
		t.Errorf("expected raw hex with labels disabled, got %q", instrs[0].Assembly())
	}
	for i, inst := range instrs {
		if inst.Label != "" {
			t.Errorf("instruction %d unexpectedly carries a label %q with Labels unset", i, inst.Label)
		}
	}
}

// TestLabelsNoOpOnSingleDecode confirms a non-nil Labels causes no error
// and produces no label on the single-instruction entry points, which have
// no "rest of the stream" to find a forward reference in.
func TestLabelsNoOpOnSingleDecode(t *testing.T) {
	data := []byte{0x4E, 0x71} // NOP

	inst, err := DecodeWithOptions(data, 0x1000, DecodeOptions{Labels: &LabelOptions{}})
	if err != nil {
		t.Fatalf("DecodeWithOptions: %v", err)
	}
	if inst.Label != "" {
		t.Errorf("expected no label from single-instruction decode, got %q", inst.Label)
	}
}

// TestInstructionStringWithLabel covers step 5 of docs/design-labels.md:
// String()'s label-line rendering convention. A labeled instruction gets a
// "name:" line before the usual "address: assembly" line, matching how a
// real assembly listing shows a label definition on its own line; an
// unlabeled instruction's String() is completely unchanged.
func TestInstructionStringWithLabel(t *testing.T) {
	data := []byte{
		0x60, 0x02, // BRA.S $1004 (0x1000)
		0x4E, 0x71, // NOP         (0x1002)
		0x4E, 0x75, // RTS         (0x1004)
	}

	instrs, err := DisassembleRangeWithOptions(data, 0x1000, DecodeOptions{Labels: &LabelOptions{}})
	if err != nil {
		t.Fatalf("DisassembleRangeWithOptions: %v", err)
	}

	want := "l00001004:\n00001004: RTS"
	if got := instrs[2].String(); got != want {
		t.Errorf("labeled instruction String() mismatch:\n want: %q\n  got: %q", want, got)
	}

	// An instruction with no label renders exactly as before this
	// feature existed — no blank label line, no behavior change.
	wantUnlabeled := "00001002: NOP"
	if got := instrs[1].String(); got != wantUnlabeled {
		t.Errorf("unlabeled instruction String() mismatch:\n want: %q\n  got: %q", wantUnlabeled, got)
	}
}

// TestInstructionStringWithoutLabelsUnchanged confirms String() is
// byte-for-byte identical to its pre-labels behavior when DecodeOptions.Labels
// is unset — Label is always "", so the new branch in String() never fires.
func TestInstructionStringWithoutLabelsUnchanged(t *testing.T) {
	data := []byte{0x4E, 0x75} // RTS
	inst, err := Decode(data, 0x2000)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	want := "00002000: RTS"
	if got := inst.String(); got != want {
		t.Errorf("String() mismatch:\n want: %q\n  got: %q", want, got)
	}
}
