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
}

// OpcodeTable is the canonical ordered pattern table used by tests and tooling.
var OpcodeTable = flattenOpcodeBuckets()

// FindDecoder uses the opcode's high nibble as a jump-table index, then matches
// only against the patterns that can exist in that 4K region of the opcode space
// and are valid on the given target CPU.
func FindDecoder(opcode uint16, cpu CPU) OpcodeDecoder {
	bit := cpuBit(cpu)
	for _, pattern := range opcodeBuckets[opcode>>12] {
		if (opcode&pattern.Mask) == pattern.Value && pattern.CPUs&bit != 0 {
			return pattern.Decoder
		}
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
