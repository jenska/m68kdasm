package decoders

// This file is the single source of truth for opcode recognition: the
// mask/value constants, the OpcodePattern/OpcodeDecoder types, and the
// opcodeBuckets jump table itself. Keeping every pattern registration in one
// place (rather than scattered across the per-instruction-family files)
// makes the precedence ordering within each bucket auditable at a glance —
// several entries below only decode correctly because a more specific
// pattern is listed before a broader one that would otherwise claim the
// same bits first (see the inline "must precede" comments).

// masks used by the decoder jump table
const (
	maskFFFF = 0xFFFF
	maskFFF0 = 0xFFF0
	maskFFF8 = 0xFFF8 // SWAP instruction mask
	maskF1F0 = 0xF1F0
	maskF1F8 = 0xF1F8
	maskF1C0 = 0xF1C0
	maskF100 = 0xF100
	maskF000 = 0xF000
	maskFFC0 = 0xFFC0
	maskFF00 = 0xFF00
	maskFB80 = 0xFB80
	maskF0C0 = 0xF0C0
	maskF0F8 = 0xF0F8
	maskF138 = 0xF138
	maskFFE0 = 0xFFE0 // FBcc: 5-bit condition field in bits 4-0
)

// exact opcode values
const (
	valNOP     = 0x4E71
	valRTS     = 0x4E75
	valSTOP    = 0x4E72
	valTRAPV   = 0x4E76
	valTRAP    = 0x4E40
	valRESET   = 0x4E70
	valRTE     = 0x4E73
	valRTR     = 0x4E77
	valILLEGAL = 0x4AFC // must precede TST below

	valMOVEMReg = 0x4880
	valMOVEMMem = 0x4C80

	valCLR  = 0x4200
	valNEG  = 0x4400
	valNEGX = 0x4000
	valNOT  = 0x4600
	valTST  = 0x4A00
	valNBCD = 0x4800
	valTAS  = 0x4AC0 // must precede TST below

	valBxx = 0x6000
	valJSR = 0x4E80
	valJMP = 0x4EC0
	valLEA = 0x41C0
	valPEA = 0x4840

	valMULU = 0xC0C0
	valMULS = 0xC1C0
	valDIVU = 0x80C0
	valDIVS = 0x81C0

	// bit op values (register form)
	valBTSTReg = 0x0500
	valBCHGReg = 0x0540
	valBCLRReg = 0x0580
	valBSETReg = 0x05C0

	// bit op values (immediate form)
	valBTSTImm = 0x0800
	valBCHGImm = 0x0840
	valBCLRImm = 0x0880
	valBSETImm = 0x08C0

	// BCD
	valABCD = 0xC100
	valSBCD = 0x8100
	valPACK = 0x8140 // 68020+, must precede OR below
	valUNPK = 0x8180 // 68020+, must precede OR below

	// immediate families
	valADDI  = 0x0600
	valSUBI  = 0x0400
	valANDI  = 0x0200
	valORI   = 0x0000
	valEORI  = 0x0A00
	valCMPI  = 0x0C00
	valMOVEQ = 0x7000

	// move sizes / other move-family opcodes
	valMOVE_B = 0x1000
	valMOVE_L = 0x2000
	valMOVE_W = 0x3000
	valMOVEP  = 0x0108
	valMOVES  = 0x0E00 // 68010+

	valOR  = 0x8000
	valSUB = 0x9000
	valCMP = 0xB000
	valAND = 0xC000
	valADD = 0xD000

	valUNLK  = 0x4E58
	valLINK  = 0x4E50
	valLINKL = 0x4808 // 68020+
	valEXT   = 0x4880
	valEXTL  = 0x48C0
	valEXTB  = 0x49C0 // 68020+
	valCHK   = 0x4180
	valCHKL  = 0x4100 // 68020+
	valEXGDD = 0xC140 // must precede AND below
	valEXGAA = 0xC148
	valEXGDA = 0xC188

	valMOVECFromCtl = 0x4E7A // 68010+
	valMOVECToCtl   = 0x4E7B // 68010+
	valRTD          = 0x4E74 // 68010+
	valBGND         = 0x4AFA // CPU32 only, must precede ILLEGAL/TST above

	// ADDQ/SUBQ (mask fixes the size field exactly, unlike most other ops,
	// since ss=11 is reserved for the Scc/DBcc/TRAPcc family below)
	valADDQB = 0x5000
	valADDQW = 0x5040
	valADDQL = 0x5080
	valSUBQB = 0x5100
	valSUBQW = 0x5140
	valSUBQL = 0x5180

	valScc    = 0x50C0
	valDBcc   = 0x50C8 // must precede Scc above
	valTRAPcc = 0x50F8 // 68020+; mask covers sss bits, only 3 of 8 are valid — must precede Scc above

	// 68020+ 32x32 MUL/DIV, CALLM/RTM (68020/68030 only), CHK2/CMP2, CAS
	valMULLong = 0x4C00
	valDIVLong = 0x4C40
	valRTMDn   = 0x06C0 // must precede CALLM below
	valRTMAn   = 0x06C8 // must precede CALLM below
	valCALLM   = 0x06C0
	valCHK2B   = 0x00C0
	valCHK2W   = 0x02C0
	valCHK2L   = 0x04C0
	valCASB    = 0x0AC0
	valCASW    = 0x0CC0
	valCASL    = 0x0EC0

	// ADDX/SUBX need the size field (bits7-6) fixed exactly, like ADDQ/SUBQ:
	// bits8-6=111 (i.e. size "11") is reserved for ADDA/SUBA/CMPA, not a
	// valid ADDX/SUBX size, so leaving it as a wildcard would swallow those.
	valADDXRegB = 0xD100
	valADDXRegW = 0xD140
	valADDXRegL = 0xD180
	valADDXMemB = 0xD108
	valADDXMemW = 0xD148
	valADDXMemL = 0xD188
	valSUBXRegB = 0x9100
	valSUBXRegW = 0x9140
	valSUBXRegL = 0x9180
	valSUBXMemB = 0x9108
	valSUBXMemW = 0x9148
	valSUBXMemL = 0x9188

	// PMMU PMOVE/PMOVEFD/PFLUSHA family (68851/68030): word1 is 0xF000
	// with the <ea> mode/reg in bits 5-0. See pmmu.go for the full
	// bit-layout derivation. PMMU's word1 has no coprocessor-ID bit (CpId
	// 0), unlike the FPU family just below (CpId 1, 0xF200).
	valPMOVE = 0xF000

	// PSAVE/PRESTORE: single-word PMMU state-frame save/restore, no
	// command word2 at all (see fpu.go's decodeFSaveRestore for the
	// structurally-identical FPU equivalent). No fpuWord1Base-style
	// coprocessor-ID adjustment applies here either — these literals are
	// used as-is.
	valPSAVE    = 0xF100
	valPRESTORE = 0xF140

	// PMMU conditional branch/set/trap family (16 conditions, unlike the
	// FPU's 32) — see pmmu_cond.go's decodePBcc/decodePDBcc/decodePScc/
	// decodePTRAPcc for the bit layout.
	valPBccW      = 0xF080 // word displacement
	valPBccL      = 0xF0C0 // long displacement
	valPDBcc      = 0xF048 // occupies PScc's address-register-direct EA slot
	valPScc       = 0xF040
	valPTRAPccW   = 0xF07A // occupies PScc's mode-7/reg-2 EA slot
	valPTRAPccL   = 0xF07B // occupies PScc's mode-7/reg-3 EA slot
	valPTRAPccNil = 0xF07C // occupies PScc's mode-7/reg-4 EA slot

	// 68040's own single-word PMMU forms — a simplified, re-encoded
	// interface distinct from the 68030/68851 two-word coprocessor forms
	// above (0xF000-prefixed, word2-dispatched): each of these is a
	// single fixed opcode word with, at most, an address register in
	// bits 2-0 (no <ea> mode field, no coprocessor command word2). See
	// pmmu040.go. PFLUSHA/PFLUSHAN/PFLUSHN/PFLUSH extend to 68060;
	// PTESTR/PTESTW do not (68060 dropped them) — see their cpuSet
	// tagging in opcodetable.go's bucket 0xF.
	valPFLUSHA040  = 0xF518
	valPFLUSHAN040 = 0xF510
	valPFLUSHN040  = 0xF500
	valPFLUSH040   = 0xF508
	valPTESTR040   = 0xF568
	valPTESTW040   = 0xF548

	// FPU "general instruction" family (68881/68882/68040/68060 built-in
	// FPU): word1 is 0xF200 with the <ea> mode/reg in bits 5-0 (unused,
	// left 0, for the register-to-register form). See fpu.go for the full
	// bit-layout derivation, cross-checked against github.com/jenska/
	// m68kasm's verified encoder rather than recalled from memory.
	valFPGeneric = 0xF200

	// FPU conditional branch/set/trap family (a distinct 32-condition
	// space from the integer ISA's 16) — see fpu.go's decodeFBcc/
	// decodeFDBcc/decodeFScc/decodeFTRAPcc for the bit layout.
	valFBccW      = 0xF280 // word displacement; cc==0 && disp==0 renders as FNOP
	valFBccL      = 0xF2C0 // long displacement
	valFDBcc      = 0xF248 // occupies FScc's address-register-direct EA slot
	valFScc       = 0xF240
	valFTRAPccW   = 0xF27A // occupies FScc's mode-7/reg-2 EA slot
	valFTRAPccL   = 0xF27B // occupies FScc's mode-7/reg-3 EA slot
	valFTRAPccNil = 0xF27C // occupies FScc's mode-7/reg-4 EA slot

	// FSAVE/FRESTORE: single-word coprocessor state-frame save/restore,
	// no command word2 (see fpu.go's decodeFSaveRestore). fpuWord1Base
	// (0xF200) | 0x0100/0x0140 | <ea> in bits 5-0.
	valFSAVE    = 0xF300
	valFRESTORE = 0xF340

	valBFTST  = 0xE0C0
	valBFEXTU = 0xE1C0
	valBFCHG  = 0xE2C0
	valBFEXTS = 0xE3C0
	valBFCLR  = 0xE4C0
	valBFFFO  = 0xE5C0
	valBFSET  = 0xE6C0
	valBFINS  = 0xE7C0
	valSHIFT  = 0xE000
)

// OpcodeDecoder is the type for decoder functions
type OpcodeDecoder func(data []byte, opcode uint16, inst *Instruction, cpu CPU) error

// OpcodePattern defines a pattern for opcode recognition
type OpcodePattern struct {
	Mask    uint16        // Bit mask for recognition
	Value   uint16        // Expected value after masking
	Decoder OpcodeDecoder // Decoder function
	CPUs    cpuSet        // CPUs this pattern is valid on
	// RequiresFPU marks a pattern that only decodes when the caller opts in
	// via DecodeOptions.FPU. FPU presence is an attached-coprocessor
	// question independent of the integer CPU tier (a bare 68020 with an
	// external 68881 and a 68040's built-in FPU both just need FPU: true),
	// so this is orthogonal to CPUs rather than folded into cpuSet — see
	// docs/design-fpu-mmu.md's "Coprocessor availability model".
	RequiresFPU bool
	// RequiresMMU is RequiresFPU's PMMU counterpart, gated by
	// DecodeOptions.MMU instead.
	RequiresMMU bool
}

func exact(value uint16, decoder OpcodeDecoder) OpcodePattern {
	return OpcodePattern{Mask: maskFFFF, Value: value, Decoder: decoder, CPUs: cpuAll}
}

func masked(mask, value uint16, decoder OpcodeDecoder) OpcodePattern {
	return OpcodePattern{Mask: mask, Value: value, Decoder: decoder, CPUs: cpuAll}
}

// exactCPU and maskedCPU declare patterns restricted to a specific cpuSet,
// for opcodes introduced by (or removed after) particular 68k family members.
func exactCPU(value uint16, decoder OpcodeDecoder, cpus cpuSet) OpcodePattern {
	return OpcodePattern{Mask: maskFFFF, Value: value, Decoder: decoder, CPUs: cpus}
}

func maskedCPU(mask, value uint16, decoder OpcodeDecoder, cpus cpuSet) OpcodePattern {
	return OpcodePattern{Mask: mask, Value: value, Decoder: decoder, CPUs: cpus}
}

// fpuExact and fpuMasked declare patterns for FPU coprocessor opcodes: valid
// on any base CPU tier (CPUs: cpuAll) but gated by RequiresFPU, so they only
// decode when the caller passes DecodeOptions.FPU: true.
func fpuExact(value uint16, decoder OpcodeDecoder) OpcodePattern {
	return OpcodePattern{Mask: maskFFFF, Value: value, Decoder: decoder, CPUs: cpuAll, RequiresFPU: true}
}

func fpuMasked(mask, value uint16, decoder OpcodeDecoder) OpcodePattern {
	return OpcodePattern{Mask: mask, Value: value, Decoder: decoder, CPUs: cpuAll, RequiresFPU: true}
}

// mmuExact and mmuMasked are fpuExact/fpuMasked's PMMU counterpart: valid
// on any base CPU tier, gated by RequiresMMU (DecodeOptions.MMU).
func mmuExact(value uint16, decoder OpcodeDecoder) OpcodePattern {
	return OpcodePattern{Mask: maskFFFF, Value: value, Decoder: decoder, CPUs: cpuAll, RequiresMMU: true}
}

func mmuMasked(mask, value uint16, decoder OpcodeDecoder) OpcodePattern {
	return OpcodePattern{Mask: mask, Value: value, Decoder: decoder, CPUs: cpuAll, RequiresMMU: true}
}

// mmuExactCPU and mmuMaskedCPU restrict an MMU pattern to a specific cpuSet
// instead of cpuAll — for the 68040's own single-word PMMU forms, which
// don't exist on earlier CPUs (and, for PTESTR/PTESTW, not on 68060 either
// — see their exact-tier registrations in opcodetable.go's bucket 0xF).
func mmuExactCPU(value uint16, decoder OpcodeDecoder, cpus cpuSet) OpcodePattern {
	return OpcodePattern{Mask: maskFFFF, Value: value, Decoder: decoder, CPUs: cpus, RequiresMMU: true}
}

func mmuMaskedCPU(mask, value uint16, decoder OpcodeDecoder, cpus cpuSet) OpcodePattern {
	return OpcodePattern{Mask: mask, Value: value, Decoder: decoder, CPUs: cpus, RequiresMMU: true}
}

// opcodeBuckets is a top-level jump table keyed by the opcode's high nibble.
// Each bucket keeps the original precedence for that 4K region of the opcode space.
var opcodeBuckets = [16][]OpcodePattern{
	0x0: {
		masked(maskF138, valMOVEP, decodeMOVEP),
		maskedCPU(maskFFF8, valRTMDn, decodeRTM, cpu020_030),
		maskedCPU(maskFFF8, valRTMAn, decodeRTM, cpu020_030),
		maskedCPU(maskFFC0, valCALLM, decodeCALLM, cpu020_030),
		maskedCPU(maskFFC0, valCHK2B, decodeCHK2CMP2, cpu020up),
		maskedCPU(maskFFC0, valCHK2W, decodeCHK2CMP2, cpu020up),
		maskedCPU(maskFFC0, valCHK2L, decodeCHK2CMP2, cpu020up),
		maskedCPU(maskFFC0, valCASB, decodeCAS, cpu020up),
		maskedCPU(maskFFC0, valCASW, decodeCAS, cpu020up),
		maskedCPU(maskFFC0, valCASL, decodeCAS, cpu020up),
		masked(maskFFC0, valBTSTReg, decodeBTST),
		masked(maskFFC0, valBTSTImm, decodeBTST),
		masked(maskFFC0, valBCHGReg, decodeBCHG),
		masked(maskFFC0, valBCHGImm, decodeBCHG),
		masked(maskFFC0, valBCLRReg, decodeBCLR),
		masked(maskFFC0, valBCLRImm, decodeBCLR),
		masked(maskFFC0, valBSETReg, decodeBSET),
		masked(maskFFC0, valBSETImm, decodeBSET),
		masked(maskFF00, valADDI, decodeADDI),
		masked(maskFF00, valSUBI, decodeSUBI),
		masked(maskFF00, valANDI, decodeANDI),
		masked(maskFF00, valORI, decodeORI),
		masked(maskFF00, valEORI, decodeEORI),
		masked(maskFF00, valCMPI, decodeCMPI),
		maskedCPU(maskFF00, valMOVES, decodeMOVES, cpu010up),
	},
	0x1: {
		masked(maskF000, valMOVE_B, decodeMOVE), // MOVE.B
	},
	0x2: {
		masked(maskF000, valMOVE_L, decodeMOVE), // MOVE.L
	},
	0x3: {
		masked(maskF000, valMOVE_W, decodeMOVE), // MOVE.W
	},
	0x4: {
		exact(valNOP, decodeNOP),
		exact(valRTS, decodeRTS),
		exact(valSTOP, decodeSTOP),
		exact(valTRAPV, decodeTRAPV),
		exact(valRESET, decodeRESET),
		exact(valRTE, decodeRTE),
		exact(valRTR, decodeRTR),
		exactCPU(valRTD, decodeRTD, cpu010up),
		exactCPU(valMOVECFromCtl, decodeMOVECFromControl, cpu010up),
		exactCPU(valMOVECToCtl, decodeMOVECToControl, cpu010up),
		exactCPU(valBGND, decodeBGND, cpu32),
		exact(valILLEGAL, decodeILLEGAL),
		maskedCPU(maskFFC0, valMULLong, decodeMULLong, cpu020up),
		maskedCPU(maskFFC0, valDIVLong, decodeDIVLong, cpu020up),
		masked(maskFFF0, valTRAP, decodeTRAP),
		masked(maskFFF8, valUNLK, decodeUNLK),
		masked(maskFFF8, valLINK, decodeLINK),
		maskedCPU(maskFFF8, valLINKL, decodeLINKLong, cpu020up),
		masked(maskFFF8, valEXT, decodeEXT),
		masked(maskFFF8, valEXTL, decodeEXT),
		maskedCPU(maskFFF8, valEXTB, decodeEXTB, cpu020up),
		masked(maskF1C0, valCHK, decodeCHK),
		maskedCPU(maskF1C0, valCHKL, decodeCHK, cpu020up),
		masked(maskFB80, valMOVEMReg, decodeMOVEM),
		masked(maskFB80, valMOVEMMem, decodeMOVEM),
		masked(maskFF00, valCLR, decodeCLR),
		masked(maskFF00, valNEG, decodeNEG),
		masked(maskFF00, valNEGX, decodeNEGX),
		masked(maskFF00, valNOT, decodeNOT),
		masked(maskFFC0, valNBCD, decodeNBCD),
		masked(maskFFC0, valTAS, decodeTAS),
		masked(maskFF00, valTST, decodeTST),
		masked(maskFFC0, valJSR, decodeJSR),
		masked(maskFFC0, valJMP, decodeJMP),
		masked(maskF1C0, valLEA, decodeLEA),
		masked(maskFFF8, valPEA, decodeSWAP),
		masked(maskFFC0, valPEA, decodePEA),
	},
	0x5: {
		masked(maskF1C0, valADDQB, decodeADDQ),
		masked(maskF1C0, valADDQW, decodeADDQ),
		masked(maskF1C0, valADDQL, decodeADDQ),
		masked(maskF1C0, valSUBQB, decodeSUBQ),
		masked(maskF1C0, valSUBQW, decodeSUBQ),
		masked(maskF1C0, valSUBQL, decodeSUBQ),
		masked(maskF0F8, valDBcc, decodeDBcc),
		maskedCPU(maskF0F8, valTRAPcc, decodeTRAPcc, cpu020up),
		masked(maskF0C0, valScc, decodeScc),
	},
	0x6: {
		masked(maskF000, valBxx, decodeBxx), // BRA/BSR/Bcc
	},
	0x7: {
		masked(maskF100, valMOVEQ, decodeMOVEQ), // MOVEQ
	},
	0x8: {
		masked(maskF1F0, valSBCD, decodeSBCD),
		masked(maskF1C0, valDIVU, decodeDIVU),
		masked(maskF1C0, valDIVS, decodeDIVS),
		maskedCPU(maskF1F0, valPACK, decodePACK, cpu020up),
		maskedCPU(maskF1F0, valUNPK, decodeUNPK, cpu020up),
		masked(maskF000, valOR, decodeOR),
	},
	0x9: {
		masked(maskF1F8, valSUBXRegB, decodeSUBX),
		masked(maskF1F8, valSUBXRegW, decodeSUBX),
		masked(maskF1F8, valSUBXRegL, decodeSUBX),
		masked(maskF1F8, valSUBXMemB, decodeSUBX),
		masked(maskF1F8, valSUBXMemW, decodeSUBX),
		masked(maskF1F8, valSUBXMemL, decodeSUBX),
		masked(maskF000, valSUB, decodeSUB),
	},
	0xB: {
		masked(maskF000, valCMP, decodeCMP), // CMP/CMPA/CMPM/EOR
	},
	0xC: {
		masked(maskF1F0, valABCD, decodeABCD),
		masked(maskF1C0, valMULU, decodeMULU),
		masked(maskF1C0, valMULS, decodeMULS),
		masked(maskF1F8, valEXGDD, decodeEXG),
		masked(maskF1F8, valEXGAA, decodeEXG),
		masked(maskF1F8, valEXGDA, decodeEXG),
		masked(maskF000, valAND, decodeAND),
	},
	0xD: {
		masked(maskF1F8, valADDXRegB, decodeADDX),
		masked(maskF1F8, valADDXRegW, decodeADDX),
		masked(maskF1F8, valADDXRegL, decodeADDX),
		masked(maskF1F8, valADDXMemB, decodeADDX),
		masked(maskF1F8, valADDXMemW, decodeADDX),
		masked(maskF1F8, valADDXMemL, decodeADDX),
		masked(maskF000, valADD, decodeADD),
	},
	0xE: {
		maskedCPU(maskFFC0, valBFTST, decodeBFTST, cpu020up),
		maskedCPU(maskFFC0, valBFEXTU, decodeBFEXTU, cpu020up),
		maskedCPU(maskFFC0, valBFCHG, decodeBFCHG, cpu020up),
		maskedCPU(maskFFC0, valBFEXTS, decodeBFEXTS, cpu020up),
		maskedCPU(maskFFC0, valBFCLR, decodeBFCLR, cpu020up),
		maskedCPU(maskFFC0, valBFFFO, decodeBFFFO, cpu020up),
		maskedCPU(maskFFC0, valBFSET, decodeBFSET, cpu020up),
		maskedCPU(maskFFC0, valBFINS, decodeBFINS, cpu020up),
		masked(maskF000, valSHIFT, decodeShiftRotate), // All ASL/ASR/LSL/LSR/ROL/ROR/ROXL/ROXR
	},
	0xF: {
		// PMMU: word1 0xF000-0xF03F, disjoint from every FPU pattern below
		// (all of which start at 0xF200+), so ordering relative to them
		// doesn't matter.
		mmuMasked(maskFFC0, valPMOVE, decodePMMUGeneral),
		mmuMasked(maskFFC0, valPSAVE, decodePSAVE),
		mmuMasked(maskFFC0, valPRESTORE, decodePRESTORE),

		// valPDBcc and the three valPTRAPcc literals must precede valPScc:
		// same EA-sub-slot precedence shape as valFDBcc/valFTRAPcc ahead
		// of valFScc just below (and integer DBcc/TRAPcc ahead of Scc in
		// bucket 0x5).
		mmuMasked(maskFFF8, valPDBcc, decodePDBcc),
		mmuExact(valPTRAPccNil, decodePTRAPccBare),
		mmuExact(valPTRAPccW, decodePTRAPccWord),
		mmuExact(valPTRAPccL, decodePTRAPccLong),
		mmuMasked(maskFFC0, valPScc, decodePScc),
		mmuMasked(maskFFF0, valPBccW, decodePBcc),
		mmuMasked(maskFFF0, valPBccL, decodePBcc),

		// 68040's own single-word PMMU forms — disjoint word1 range
		// (0xF500-0xF56F) from every PMMU/FPU pattern above and below, so
		// ordering doesn't matter; cpuSet-restricted (not cpuAll like
		// every other MMU pattern so far) since these opcodes don't exist
		// before the 68040, and PTESTR/PTESTW not even on the 68060.
		mmuExactCPU(valPFLUSHA040, decodePFLUSHA040, cpu040up),
		mmuExactCPU(valPFLUSHAN040, decodePFLUSHAN040, cpu040up),
		mmuMaskedCPU(maskFFF8, valPFLUSHN040, decodePFLUSHN040, cpu040up),
		mmuMaskedCPU(maskFFF8, valPFLUSH040, decodePFLUSH040, cpu040up),
		mmuMaskedCPU(maskFFF8, valPTESTR040, decodePTESTR040, cpu040),
		mmuMaskedCPU(maskFFF8, valPTESTW040, decodePTESTW040, cpu040),

		// valFDBcc and the three valFTRAPcc literals must precede valFScc:
		// each occupies a specific EA sub-slot (address-register-direct for
		// FDBcc; mode-7/reg-2,3,4 for FTRAPcc) that valFScc's own mask would
		// otherwise also match — the exact same precedence shape integer
		// DBcc/TRAPcc already require ahead of Scc (bucket 0x5 above).
		fpuMasked(maskFFF8, valFDBcc, decodeFDBcc),
		fpuExact(valFTRAPccNil, decodeFTRAPccBare),
		fpuExact(valFTRAPccW, decodeFTRAPccWord),
		fpuExact(valFTRAPccL, decodeFTRAPccLong),
		fpuMasked(maskFFC0, valFScc, decodeFScc),
		fpuMasked(maskFFE0, valFBccW, decodeFBcc),
		fpuMasked(maskFFE0, valFBccL, decodeFBcc),
		fpuMasked(maskFFC0, valFPGeneric, decodeFPGeneric),
		fpuMasked(maskFFC0, valFSAVE, decodeFSAVE),
		fpuMasked(maskFFC0, valFRESTORE, decodeFRESTORE),
	},
}

// OpcodeTable is the canonical ordered pattern table used by tests and tooling.
var OpcodeTable = flattenOpcodeBuckets()

// FindDecoder uses the opcode's high nibble as a jump-table index, then matches
// only against the patterns that can exist in that 4K region of the opcode space,
// are valid on the given target CPU, and (for coprocessor patterns) are enabled
// by the fpu/mmu capability flags.
func FindDecoder(opcode uint16, cpu CPU, fpu, mmu bool) OpcodeDecoder {
	bit := cpuBit(cpu)
	for _, pattern := range opcodeBuckets[opcode>>12] {
		if (opcode & pattern.Mask) != pattern.Value {
			continue
		}
		if pattern.CPUs&bit == 0 {
			continue
		}
		if pattern.RequiresFPU && !fpu {
			continue
		}
		if pattern.RequiresMMU && !mmu {
			continue
		}
		return pattern.Decoder
	}
	return nil
}

func flattenOpcodeBuckets() []OpcodePattern {
	total := 0
	for _, bucket := range opcodeBuckets {
		total += len(bucket)
	}

	flat := make([]OpcodePattern, 0, total)
	for _, bucket := range opcodeBuckets {
		flat = append(flat, bucket...)
	}
	return flat
}
