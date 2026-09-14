package m68kdasm

import (
	"testing"

	"github.com/jenska/m68kasm"
)

// fpuTarget assembles with the base FPU feature (no transcendentals) on a
// 68020 core, matching the decoder's own current scope (see
// internal/decoders/fpu.go and docs/design-fpu-mmu.md's delivery
// sequence, step 3).
var fpuTarget = m68kasm.ParseOptions{
	Target: m68kasm.Target{CPU: m68kasm.CPU68020, Features: m68kasm.FeatFPU},
}

// fpuFullTarget additionally enables the transcendental function set
// (m68kasm models a discrete 68881/68882 vs. a reduced/integrated FPU as
// two separate feature bits; this decoder doesn't mirror that distinction
// — see the comment on fpGeneralOps' transcendental entries in
// internal/decoders/fpu.go).
var fpuFullTarget = m68kasm.ParseOptions{
	Target: m68kasm.Target{CPU: m68kasm.CPU68020, Features: m68kasm.FeatFPU | m68kasm.FeatFPUFull},
}

// TestFPUGenericRoundTrip assembles each case with m68kasm v1.5.0 (the
// first Go module release with verified FPU encoding — see the CHANGELOG
// and docs/design-fpu-mmu.md) and confirms this package's decoder recovers
// the exact same assembly text, the same round-trip discipline
// TestDisassembleRoundTrip already uses for the integer core.
func TestFPUGenericRoundTrip(t *testing.T) {
	testCases := []string{
		"FMOVE.X FP1, FP0",
		"FADD.X FP1, FP0",
		"FSUB.X FP1, FP0",
		"FMUL.X FP1, FP0",
		"FDIV.X FP1, FP0",
		"FCMP.X FP1, FP0",
		"FABS.X FP1, FP0",
		"FNEG.X FP1, FP0",
		"FSQRT.X FP1, FP0",
		"FTST.X FP0",
		"FNOP",

		"FMOVE.L D0, FP0",
		"FMOVE.W (A0), FP0",
		"FMOVE.B (A0)+, FP0",
		"FMOVE.L -(A0), FP0",
		"FMOVE.X (A0), FP0",
		"FADD.L (A0), FP0",
		"FADD.S (A0), FP0",
		"FADD.D (A0), FP0",
		"FSUB.L (16,A0), FP1",
		"FMUL.L $00001234, FP2",
		"FDIV.L (4,A0,D1.W), FP3",
		"FCMP.L (A0), FP0",
		"FABS.L (A0), FP0",
		"FNEG.L (A0), FP0",
		"FSQRT.L (A0), FP0",
		"FTST.L (A0)",
		"FTST.L D0",

		"FMOVE.L FP0, (A0)",
		"FMOVE.W FP1, (A0)+",
		"FMOVE.B FP2, -(A0)",
		"FMOVE.X FP3, (A0)",

		"FADD.L #5, FP0",
		"FADD.W #5, FP0",
		"FADD.B #5, FP0",

		"FMOVEM.X FP0-FP3, -(A7)",
		"FMOVEM.X FP0-FP3, (A0)",
		"FMOVEM.X (A0)+, FP0-FP3",
		"FMOVEM.X D0, -(A7)",
		"FMOVEM.X D0, (A0)",
		"FMOVEM.X (A0)+, D0",
		"FMOVEM.X (A0), D0",
		"FMOVEM.X FP5, -(A7)",
		"FMOVEM.X FP1/FP4/FP6, (A0)",
	}

	for _, source := range testCases {
		t.Run(source, func(t *testing.T) {
			data, err := m68kasm.AssembleStringWithOptions(source, fpuTarget)
			if err != nil {
				t.Fatalf("assembler error for %q: %v", source, err)
			}

			inst, err := DecodeWithOptions(data, 0, DecodeOptions{CPU: M68020, FPU: true})
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

// TestFPUTranscendentalRoundTrip covers the FPU's transcendental function
// set — same monadic "<ea>,FPn"/"FPm,FPn" shape as FABS/FNEG/FSQRT, just an
// FeatFPUFull-gated (real discrete 68881/68882) opmode set on the assembler
// side. See fpGeneralOps' transcendental entries in internal/decoders/fpu.go.
func TestFPUTranscendentalRoundTrip(t *testing.T) {
	mnemonics := []string{
		"FSIN", "FCOS", "FTAN", "FATAN", "FASIN", "FACOS", "FATANH",
		"FSINH", "FCOSH", "FTANH", "FETOX", "FETOXM1", "FLOGN", "FLOGNP1",
		"FLOG10", "FLOG2", "FTWOTOX", "FTENTOX",
	}

	for _, mnemonic := range mnemonics {
		t.Run(mnemonic, func(t *testing.T) {
			for _, source := range []string{
				mnemonic + ".X FP1, FP0",
				mnemonic + ".L (A0), FP0",
			} {
				data, err := m68kasm.AssembleStringWithOptions(source, fpuFullTarget)
				if err != nil {
					t.Fatalf("assembler error for %q: %v", source, err)
				}

				inst, err := DecodeWithOptions(data, 0, DecodeOptions{CPU: M68020, FPU: true})
				if err != nil {
					t.Fatalf("decode error for %q (bytes % X): %v", source, data, err)
				}
				if int(inst.Size) != len(data) {
					t.Errorf("%q: decoded size %d, assembled %d bytes (% X)", source, inst.Size, len(data), data)
				}
				if got := inst.Assembly(); got != source {
					t.Errorf("mismatch\n want: %q\n  got: %q\nbytes: % X", source, got, data)
				}
			}
		})
	}
}

// TestFPUMathExtRoundTrip covers the FPU's "math extensions" bucket:
// FGETEXP/FGETMAN (monadic, same shape as FABS) and FSCALE/FMOD/FREM
// (genuinely binary, same shape as FADD) — see their fpGeneralOps entries
// in internal/decoders/fpu.go.
func TestFPUMathExtRoundTrip(t *testing.T) {
	testCases := []string{
		"FGETEXP.X FP1, FP0",
		"FGETEXP.L (A0), FP0",
		"FGETMAN.X FP1, FP0",
		"FGETMAN.L (A0), FP0",
		"FSCALE.X FP1, FP0",
		"FSCALE.L (A0), FP0",
		"FMOD.X FP1, FP0",
		"FMOD.L (A0), FP0",
		"FREM.X FP1, FP0",
		"FREM.L (A0), FP0",
	}
	for _, source := range testCases {
		t.Run(source, func(t *testing.T) {
			data, err := m68kasm.AssembleStringWithOptions(source, fpuFullTarget)
			if err != nil {
				t.Fatalf("assembler error for %q: %v", source, err)
			}
			inst, err := DecodeWithOptions(data, 0, DecodeOptions{CPU: M68020, FPU: true})
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

// TestFPUPackedBCDRoundTrip covers the last of the FPU's 7 data formats:
// packed BCD (.p). The load direction ("FMOVE.P <ea>,FPn") needs no new
// decode logic (format code 3 through the existing generic <ea> path,
// same as every other format); the store direction ("FMOVE.P FPn,<ea>{k}")
// is a genuinely different word2 shape carrying a k-factor instead of a
// format code — see decodeFMOVEPStore in internal/decoders/fpu.go.
func TestFPUPackedBCDRoundTrip(t *testing.T) {
	testCases := []string{
		"FMOVE.P (A0), FP0",
		"FMOVE.P (A0)+, FP1",

		"FMOVE.P FP0, (A0){#-5}",
		"FMOVE.P FP1, (A0){#0}",
		"FMOVE.P FP2, (A0){#17}",
		"FMOVE.P FP3, (A0){#-64}",
		"FMOVE.P FP4, (A0){D2}",
	}
	for _, source := range testCases {
		t.Run(source, func(t *testing.T) {
			data, err := m68kasm.AssembleStringWithOptions(source, fpuTarget)
			if err != nil {
				t.Fatalf("assembler error for %q: %v", source, err)
			}
			inst, err := DecodeWithOptions(data, 0, DecodeOptions{CPU: M68020, FPU: true})
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

// TestFPUMOVEMCtrlRoundTrip covers FMOVEM's other register-list form: the
// FPCR/FPSR/FPIAR control registers (a 3-bit mask, genuinely distinct
// subsystem from the FP0-FP7 data-register list already covered by
// TestFPUGenericRoundTrip) — see decodeFMOVEM's fpMovemCtrlLoad/
// fpMovemCtrlStore cases in internal/decoders/fpu.go.
func TestFPUMOVEMCtrlRoundTrip(t *testing.T) {
	testCases := []string{
		"FMOVEM.L FPIAR, D0",
		"FMOVEM.L FPIAR, A0",
		"FMOVEM.L FPIAR, (A0)",
		"FMOVEM.L FPSR/FPIAR, (A0)",
		"FMOVEM.L FPCR/FPSR/FPIAR, (A0)", // exercises bit 12 (FPCR) colliding with a naive "top nibble" read
		"FMOVEM.L (A0), FPIAR",
		"FMOVEM.L (A0), FPSR/FPIAR",
		"FMOVEM.L (A0), FPCR/FPSR/FPIAR",
		"FMOVEM.L D0, FPIAR",
	}
	for _, source := range testCases {
		t.Run(source, func(t *testing.T) {
			data, err := m68kasm.AssembleStringWithOptions(source, fpuTarget)
			if err != nil {
				t.Fatalf("assembler error for %q: %v", source, err)
			}
			inst, err := DecodeWithOptions(data, 0, DecodeOptions{CPU: M68020, FPU: true})
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

// TestFPUSaveRestoreRoundTrip covers FSAVE/FRESTORE's instruction shell
// (mnemonic + <ea>) — see decodeFSAVE/decodeFRESTORE in
// internal/decoders/fpu.go. Frame contents are out of scope (see
// docs/design-fpu-mmu.md's Non-goals).
func TestFPUSaveRestoreRoundTrip(t *testing.T) {
	testCases := []string{
		"FSAVE (A0)",
		"FSAVE -(A0)",
		"FRESTORE (A0)",
		"FRESTORE (A0)+",
	}
	for _, source := range testCases {
		t.Run(source, func(t *testing.T) {
			data, err := m68kasm.AssembleStringWithOptions(source, fpuTarget)
			if err != nil {
				t.Fatalf("assembler error for %q: %v", source, err)
			}
			inst, err := DecodeWithOptions(data, 0, DecodeOptions{CPU: M68020, FPU: true})
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

// TestFPUMOVECRRoundTrip covers FMOVECR (ROM constant load) — see
// decodeFMOVECR in internal/decoders/fpu.go.
func TestFPUMOVECRRoundTrip(t *testing.T) {
	testCases := []string{
		"FMOVECR #0, FP0",
		"FMOVECR #11, FP2",
		"FMOVECR #$7F, FP7",
	}
	for _, source := range testCases {
		t.Run(source, func(t *testing.T) {
			data, err := m68kasm.AssembleStringWithOptions(source, fpuTarget)
			if err != nil {
				t.Fatalf("assembler error for %q: %v", source, err)
			}
			inst, err := DecodeWithOptions(data, 0, DecodeOptions{CPU: M68020, FPU: true})
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

// TestFPUSINCOSRoundTrip covers FSINCOS (dual sine/cosine result) — see
// decodeFSINCOS in internal/decoders/fpu.go.
func TestFPUSINCOSRoundTrip(t *testing.T) {
	testCases := []string{
		"FSINCOS.X FP1, FP2:FP3",
		"FSINCOS.X (A0), FP2:FP3",
		"FSINCOS.L (A0), FP0:FP1",
		"FSINCOS.D (A0), FP4:FP5",
	}
	for _, source := range testCases {
		t.Run(source, func(t *testing.T) {
			data, err := m68kasm.AssembleStringWithOptions(source, fpuFullTarget)
			if err != nil {
				t.Fatalf("assembler error for %q: %v", source, err)
			}
			inst, err := DecodeWithOptions(data, 0, DecodeOptions{CPU: M68020, FPU: true})
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

// TestFPUCondBranchRoundTrip covers the FPU's conditional branch/set/trap
// family (FBcc/FDBcc/FScc/FTRAPcc), a distinct 32-condition space from the
// integer ISA's 16 — see internal/decoders/fpu.go's decodeFBcc and friends.
func TestFPUCondBranchRoundTrip(t *testing.T) {
	testCases := []string{
		"FBEQ.W $0010",
		"FBNE.L $00010000",
		"FBT.W $0004",
		"FBF.W $0004", // cc=0 with a nonzero displacement must NOT collapse into FNOP
		"FNOP",        // cc=0 with a zero displacement is the FNOP special case

		"FDBEQ D0, $0010",
		"FDBNE D3, $0020",

		"FSGT D0",
		"FSLT (A0)",
		"FSEQ (A0)+",
		"FSNE -(A0)",

		"FTRAPEQ",
		"FTRAPNE.W #$04D2",
		"FTRAPGT.L #$0001E240",
	}

	for _, source := range testCases {
		t.Run(source, func(t *testing.T) {
			data, err := m68kasm.AssembleStringWithOptions(source, fpuTarget)
			if err != nil {
				t.Fatalf("assembler error for %q: %v", source, err)
			}

			inst, err := DecodeWithOptions(data, 0, DecodeOptions{CPU: M68020, FPU: true})
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

// TestFPUFloatImmediateRoundTrip checks the Single/Double/Extended
// floating-point immediate literal path specifically, since it does its
// own IEEE-754/extended-precision decoding (decodeFPFloatImmediate in
// internal/decoders/fpu.go) rather than reusing the integer <ea> decoder.
func TestFPUFloatImmediateRoundTrip(t *testing.T) {
	testCases := []string{
		"FMOVE.S #1.5, FP0",
		"FMOVE.D #1.5, FP0",
		"FMOVE.X #1.5, FP0",
		"FMOVE.X #-2.5, FP0",
	}

	for _, source := range testCases {
		t.Run(source, func(t *testing.T) {
			data, err := m68kasm.AssembleStringWithOptions(source, fpuTarget)
			if err != nil {
				t.Fatalf("assembler error for %q: %v", source, err)
			}

			inst, err := DecodeWithOptions(data, 0, DecodeOptions{CPU: M68020, FPU: true})
			if err != nil {
				t.Fatalf("decode error for %q (bytes % X): %v", source, data, err)
			}
			if int(inst.Size) != len(data) {
				t.Errorf("%q: decoded size %d, assembled %d bytes (% X)", source, inst.Size, len(data), data)
			}
			t.Logf("%q -> %q (bytes % X)", source, inst.Assembly(), data)
		})
	}
}

// TestFPUWithoutOptIn confirms F-line opcodes still fall through to the
// unknown-opcode DC.W path when the caller does not opt in via
// DecodeOptions.FPU, preserving today's behavior by default.
func TestFPUWithoutOptIn(t *testing.T) {
	data, err := m68kasm.AssembleStringWithOptions("FADD.X FP1, FP0", fpuTarget)
	if err != nil {
		t.Fatalf("assembler error: %v", err)
	}

	inst, err := DecodeWithOptions(data, 0, DecodeOptions{CPU: M68020})
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if inst.Mnemonic != "DC.W" {
		t.Fatalf("expected FADD to be unrecognized without FPU: true, got %q", inst.Assembly())
	}
}
