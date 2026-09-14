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
