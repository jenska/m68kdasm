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
