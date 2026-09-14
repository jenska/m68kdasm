package m68kdasm

import (
	"testing"

	"github.com/jenska/m68kasm"
)

// pmmuTarget assembles with the PMMU feature on a 68020 core, matching
// the decoder's own current scope (see internal/decoders/pmmu.go and
// docs/design-fpu-mmu.md's delivery sequence, step 9).
var pmmuTarget = m68kasm.ParseOptions{
	Target: m68kasm.Target{CPU: m68kasm.CPU68020, Features: m68kasm.FeatPMMU},
}

// TestPMOVERoundTrip covers PMOVE's full register set (except BAD0-7/
// BAC0-7, a genuinely different numbered-register shape left for a
// follow-up) and PMOVEFD — see decodePMOVEFamily in
// internal/decoders/pmmu.go.
func TestPMOVERoundTrip(t *testing.T) {
	testCases := []string{
		"PMOVE.L (A0), TC",
		"PMOVE.L TC, (A0)",
		"PMOVE.L (A0), DRP",
		"PMOVE.L DRP, (A0)",
		"PMOVE.L (A0), SRP",
		"PMOVE.L SRP, (A0)",
		"PMOVE.L (A0), CRP",
		"PMOVE.L CRP, (A0)",
		"PMOVE.B (A0), CAL",
		"PMOVE.B CAL, (A0)",
		"PMOVE.B (A0), VAL",
		"PMOVE.B VAL, (A0)",
		"PMOVE.B (A0), SCC",
		"PMOVE.B SCC, (A0)",
		"PMOVE.W (A0), AC",
		"PMOVE.W AC, (A0)",
		"PMOVE.W PCSR, (A0)",
		"PMOVE.L (A0), TT0",
		"PMOVE.L TT0, (A0)",
		"PMOVE.L (A0), TT1",
		"PMOVE.L TT1, (A0)",
		"PMOVE.W (A0), MMUSR",
		"PMOVE.W MMUSR, (A0)",

		"PMOVEFD.L (A0), TC",
		"PMOVEFD.L (A0), DRP",
		"PMOVEFD.L (A0), SRP",
		"PMOVEFD.L (A0), CRP",
		"PMOVEFD.L (A0), TT0",
		"PMOVEFD.L (A0), TT1",

		"PFLUSHA",
	}

	for _, source := range testCases {
		t.Run(source, func(t *testing.T) {
			data, err := m68kasm.AssembleStringWithOptions(source, pmmuTarget)
			if err != nil {
				t.Fatalf("assembler error for %q: %v", source, err)
			}
			inst, err := DecodeWithOptions(data, 0, DecodeOptions{CPU: M68020, MMU: true})
			if err != nil {
				t.Fatalf("decode error for %q (bytes % X): %v", source, data, err)
			}
			if int(inst.Size) != len(data) {
				t.Errorf("%q: decoded size %d, assembled %d bytes (% X)", source, inst.Size, len(data), data)
			}
			if got := inst.Assembly(); got != source {
				t.Errorf("mismatch\n want: %q\n  got: %q\nbytes: % X", source, got, data)
			}
		})
	}
}

// TestPFlushLoadTestRoundTrip covers PFLUSH/PFLUSHS/PFLUSHR, PLOADR/
// PLOADW, and PTESTR/PTESTW — the PMMU cache/TLB management instructions,
// all sharing the new "function code specifier" operand shape (SFC/DFC, a
// Dn, or #<imm>) — see decodePFLUSH/decodePLOAD/decodePTEST in
// internal/decoders/pmmu.go.
func TestPFlushLoadTestRoundTrip(t *testing.T) {
	testCases := []string{
		"PFLUSH SFC, #0",
		"PFLUSH DFC, #31",
		"PFLUSH D3, #5",
		"PFLUSH #6, #5",
		"PFLUSHS SFC, #0",
		"PFLUSH SFC, #0, (A0)",
		"PFLUSHS DFC, #12, (A0)",
		"PFLUSHR (A0)",

		"PLOADR SFC, (A0)",
		"PLOADR D2, (A0)",
		"PLOADR #3, (A0)",
		"PLOADW DFC, (A0)",

		"PTESTR SFC, (A0), #0",
		"PTESTR DFC, (A0), #7",
		"PTESTR D1, (A0), #3",
		"PTESTR #2, (A0), #3",
		"PTESTR SFC, (A0), #3, A5",
		"PTESTW SFC, (A0), #4",
		"PTESTW SFC, (A0), #4, A2",
	}

	for _, source := range testCases {
		t.Run(source, func(t *testing.T) {
			data, err := m68kasm.AssembleStringWithOptions(source, pmmuTarget)
			if err != nil {
				t.Fatalf("assembler error for %q: %v", source, err)
			}
			inst, err := DecodeWithOptions(data, 0, DecodeOptions{CPU: M68020, MMU: true})
			if err != nil {
				t.Fatalf("decode error for %q (bytes % X): %v", source, data, err)
			}
			if int(inst.Size) != len(data) {
				t.Errorf("%q: decoded size %d, assembled %d bytes (% X)", source, inst.Size, len(data), data)
			}
			if got := inst.Assembly(); got != source {
				t.Errorf("mismatch\n want: %q\n  got: %q\nbytes: % X", source, got, data)
			}
		})
	}
}

// TestBADBACRoundTrip covers PMOVE's BAD0-BAD7/BAC0-BAC7 forms — a
// numbered-register shape distinct from every other PMOVE register, with
// an inverted load/store direction bit — see decodeBADBAC in
// internal/decoders/pmmu.go.
func TestBADBACRoundTrip(t *testing.T) {
	testCases := []string{
		"PMOVE.L (A0), BAD0",
		"PMOVE.L (A0), BAD7",
		"PMOVE.L BAD3, (A0)",
		"PMOVE.L (A0), BAC0",
		"PMOVE.L (A0), BAC7",
		"PMOVE.L BAC5, (A0)",
	}
	for _, source := range testCases {
		t.Run(source, func(t *testing.T) {
			data, err := m68kasm.AssembleStringWithOptions(source, pmmuTarget)
			if err != nil {
				t.Fatalf("assembler error for %q: %v", source, err)
			}
			inst, err := DecodeWithOptions(data, 0, DecodeOptions{CPU: M68020, MMU: true})
			if err != nil {
				t.Fatalf("decode error for %q (bytes % X): %v", source, data, err)
			}
			if int(inst.Size) != len(data) {
				t.Errorf("%q: decoded size %d, assembled %d bytes (% X)", source, inst.Size, len(data), data)
			}
			if got := inst.Assembly(); got != source {
				t.Errorf("mismatch\n want: %q\n  got: %q\nbytes: % X", source, got, data)
			}
		})
	}
}

// TestPSAVERestoreRoundTrip covers PSAVE/PRESTORE's instruction shell
// (mnemonic + <ea>) — see decodePSAVE/decodePRESTORE in
// internal/decoders/pmmu.go. Frame contents are out of scope (see
// docs/design-fpu-mmu.md's Non-goals).
func TestPSAVERestoreRoundTrip(t *testing.T) {
	testCases := []string{
		"PSAVE (A0)",
		"PSAVE -(A0)",
		"PRESTORE (A0)",
		"PRESTORE (A0)+",
	}
	for _, source := range testCases {
		t.Run(source, func(t *testing.T) {
			data, err := m68kasm.AssembleStringWithOptions(source, pmmuTarget)
			if err != nil {
				t.Fatalf("assembler error for %q: %v", source, err)
			}
			inst, err := DecodeWithOptions(data, 0, DecodeOptions{CPU: M68020, MMU: true})
			if err != nil {
				t.Fatalf("decode error for %q (bytes % X): %v", source, data, err)
			}
			if int(inst.Size) != len(data) {
				t.Errorf("%q: decoded size %d, assembled %d bytes (% X)", source, inst.Size, len(data), data)
			}
			if got := inst.Assembly(); got != source {
				t.Errorf("mismatch\n want: %q\n  got: %q\nbytes: % X", source, got, data)
			}
		})
	}
}

// TestPccRoundTrip covers the PMMU's own conditional branch/set/trap
// family (PBcc/PDBcc/PScc/PTRAPcc), a 16-condition space structurally
// identical to FBcc/FDBcc/FScc/FTRAPcc (fpu_test.go) — see
// internal/decoders/pmmu_cond.go.
func TestPccRoundTrip(t *testing.T) {
	testCases := []string{
		"PBBS.W $0010",
		"PBBC.L $00010000",
		"PBAS.W $0004",

		"PDBLS D0, $0010",
		"PDBLC D3, $0020",

		"PSSS D0",
		"PSSC (A0)",
		"PSWS (A0)+",
		"PSWC -(A0)",

		"PTRAPGS",
		"PTRAPGC.W #$04D2",
		"PTRAPCS.L #$0001E240",
	}

	for _, source := range testCases {
		t.Run(source, func(t *testing.T) {
			data, err := m68kasm.AssembleStringWithOptions(source, pmmuTarget)
			if err != nil {
				t.Fatalf("assembler error for %q: %v", source, err)
			}
			inst, err := DecodeWithOptions(data, 0, DecodeOptions{CPU: M68020, MMU: true})
			if err != nil {
				t.Fatalf("decode error for %q (bytes % X): %v", source, data, err)
			}
			if int(inst.Size) != len(data) {
				t.Errorf("%q: decoded size %d, assembled %d bytes (% X)", source, inst.Size, len(data), data)
			}
			if got := inst.Assembly(); got != source {
				t.Errorf("mismatch\n want: %q\n  got: %q\nbytes: % X", source, got, data)
			}
		})
	}
}

// TestPMMU040RoundTrip covers the 68040's own single-word PMMU forms — a
// simplified, re-encoded interface distinct from the 68030/68851 two-word
// coprocessor forms above, sharing a mnemonic with PFLUSHA/PFLUSHN/PFLUSH
// but never colliding at the bit level (word1 0xF500-0xF56F here vs.
// 0xF000-prefixed there) — see internal/decoders/pmmu040.go.
func TestPMMU040RoundTrip(t *testing.T) {
	target68040 := m68kasm.ParseOptions{
		Target: m68kasm.Target{CPU: m68kasm.CPU68040, Features: m68kasm.FeatPMMU},
	}

	testCases := []string{
		"PFLUSHA",
		"PFLUSHAN",
		"PFLUSHN (A0)",
		"PFLUSHN (A7)",
		"PFLUSH (A3)",
		"PTESTR (A0)",
		"PTESTW (A5)",
	}

	for _, source := range testCases {
		t.Run(source, func(t *testing.T) {
			data, err := m68kasm.AssembleStringWithOptions(source, target68040)
			if err != nil {
				t.Fatalf("assembler error for %q: %v", source, err)
			}
			if len(data) != 2 {
				t.Fatalf("%q: expected a 2-byte single-word encoding, got % X", source, data)
			}
			inst, err := DecodeWithOptions(data, 0, DecodeOptions{CPU: M68040, MMU: true})
			if err != nil {
				t.Fatalf("decode error for %q (bytes % X): %v", source, data, err)
			}
			if int(inst.Size) != len(data) {
				t.Errorf("%q: decoded size %d, assembled %d bytes (% X)", source, inst.Size, len(data), data)
			}
			if got := inst.Assembly(); got != source {
				t.Errorf("mismatch\n want: %q\n  got: %q\nbytes: % X", source, got, data)
			}
		})
	}
}

// TestPMMU040NotOnM68030 confirms the 68040's single-word PMMU forms are
// not recognized when targeting M68030 (where only the older 0xF000-based
// two-word PFLUSHA form is valid), and that PTESTR/PTESTW's single-word
// form — dropped on the 68060 — is not recognized there either.
func TestPMMU040NotOnM68030(t *testing.T) {
	// PFLUSHAN's 68040-only 2-byte encoding, decoded with CPU: M68030.
	data := []byte{0xF5, 0x10}
	inst, err := DecodeWithOptions(data, 0, DecodeOptions{CPU: M68030, MMU: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inst.Mnemonic == "PFLUSHAN" {
		t.Fatalf("PFLUSHAN's 68040-only single-word form should not decode on M68030")
	}

	// PTESTR (A0)'s 68040-only encoding, decoded with CPU: M68060.
	data = []byte{0xF5, 0x68}
	inst, err = DecodeWithOptions(data, 0, DecodeOptions{CPU: M68060, MMU: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inst.Mnemonic == "PTESTR" {
		t.Fatalf("PTESTR's single-word form should not decode on M68060 (dropped there)")
	}
}

// TestPMMUWithoutOptIn confirms PMMU F-line opcodes still fall through to
// the unknown-opcode DC.W path when the caller does not opt in via
// DecodeOptions.MMU, preserving today's behavior by default.
func TestPMMUWithoutOptIn(t *testing.T) {
	data, err := m68kasm.AssembleStringWithOptions("PMOVE.L (A0), TC", pmmuTarget)
	if err != nil {
		t.Fatalf("assembler error: %v", err)
	}

	inst, err := DecodeWithOptions(data, 0, DecodeOptions{CPU: M68020})
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if inst.Mnemonic != "DC.W" {
		t.Fatalf("expected PMOVE to be unrecognized without MMU: true, got %q", inst.Assembly())
	}
}
