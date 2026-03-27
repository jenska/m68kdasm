package decoders

import "strings"

// common masks and values used by the decoder jump table
const (
	// masks
	maskFFFF  = 0xFFFF
	maskFFF0  = 0xFFF0
	maskF1F0  = 0xF1F0
	maskF1C0  = 0xF1C0
	maskF100  = 0xF100
	maskF000  = 0xF000
	maskFFC0  = 0xFFC0
	maskFF00  = 0xFF00
	maskFB80  = 0xFB80
	maskBitOp = 0xFFC0 // alias for bit-op mask

	// exact opcode values
	valNOP   = 0x4E71
	valRTS   = 0x4E75
	valSTOP  = 0x4E72
	valTRAPV = 0x4E76
	valTRAP  = 0x4E40

	valMOVEMReg = 0x4880
	valMOVEMMem = 0x4C80

	valCLR  = 0x4200
	valNEG  = 0x4400
	valNEGX = 0x4000
	valNOT  = 0x4600
	valTST  = 0x4A00

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

	// immediate families
	valADDI  = 0x0600
	valSUBI  = 0x0400
	valANDI  = 0x0200
	valORI   = 0x0000
	valEORI  = 0x0A00
	valCMPI  = 0x0C00
	valMOVEQ = 0x7000

	// move sizes
	valMOVE_B = 0x1000
	valMOVE_L = 0x2000
	valMOVE_W = 0x3000

	valOR    = 0x8000
	valSUB   = 0x9000
	valCMP   = 0xB000
	valAND   = 0xC000
	valADD   = 0xD000
	valSHIFT = 0xE000
)

// Instruction represents a single disassembled instruction.
// This mirrors the type from m68kdasm to avoid circular imports.
type Instruction struct {
	Address  uint32
	Opcode   uint16
	Mnemonic string
	Operands string
	Size     uint32 // Size in bytes
	Bytes    []byte // Raw instruction data
	// ExtensionWords holds any decoded words that follow the opcode word.
	ExtensionWords []uint16
	Metadata       Metadata
}

type Metadata struct {
	Mnemonic        string
	MnemonicBase    string
	SizeSuffix      string
	Operands        []Operand
	BranchTarget    *uint32
	ImmediateValues []ImmediateValue
}

type OperandKind string

const (
	OperandKindRegister      OperandKind = "register"
	OperandKindImmediate     OperandKind = "immediate"
	OperandKindEffectiveAddr OperandKind = "effective_address"
	OperandKindRegisterList  OperandKind = "register_list"
	OperandKindBranchTarget  OperandKind = "branch_target"
)

type RegisterKind string

const (
	RegisterKindData    RegisterKind = "data"
	RegisterKindAddress RegisterKind = "address"
	RegisterKindPC      RegisterKind = "pc"
)

type Register struct {
	Kind   RegisterKind
	Number uint8
}

type ImmediateValue struct {
	Value  uint32
	Signed int32
	Size   uint8
}

type EffectiveAddressKind string

const (
	EAKindDataRegisterDirect    EffectiveAddressKind = "data_register_direct"
	EAKindAddressRegisterDirect EffectiveAddressKind = "address_register_direct"
	EAKindAddressIndirect       EffectiveAddressKind = "address_indirect"
	EAKindPostIncrement         EffectiveAddressKind = "post_increment"
	EAKindPreDecrement          EffectiveAddressKind = "pre_decrement"
	EAKindDisplacement          EffectiveAddressKind = "displacement"
	EAKindIndex                 EffectiveAddressKind = "index"
	EAKindAbsoluteShort         EffectiveAddressKind = "absolute_short"
	EAKindAbsoluteLong          EffectiveAddressKind = "absolute_long"
	EAKindPCDisplacement        EffectiveAddressKind = "pc_displacement"
	EAKindPCIndex               EffectiveAddressKind = "pc_index"
	EAKindImmediate             EffectiveAddressKind = "immediate"
)

type IndexRegister struct {
	Register Register
	Size     string
}

type EffectiveAddress struct {
	Kind            EffectiveAddressKind
	Mode            uint8
	Register        uint8
	Base            *Register
	Displacement    *int32
	AbsoluteAddress *uint32
	ResolvedAddress *uint32
	Immediate       *ImmediateValue
	Index           *IndexRegister
}

type Operand struct {
	Text             string
	Kind             OperandKind
	Register         *Register
	Immediate        *ImmediateValue
	EffectiveAddress *EffectiveAddress
	RegisterList     []string
	BranchTarget     *uint32
}

// OpcodeDecoder is the type for decoder functions
type OpcodeDecoder func(data []byte, opcode uint16, inst *Instruction) error

// OpcodePattern defines a pattern for opcode recognition
type OpcodePattern struct {
	Mask    uint16        // Bit mask for recognition
	Value   uint16        // Expected value after masking
	Decoder OpcodeDecoder // Decoder function
}

func exact(value uint16, decoder OpcodeDecoder) OpcodePattern {
	return OpcodePattern{Mask: maskFFFF, Value: value, Decoder: decoder}
}

func masked(mask, value uint16, decoder OpcodeDecoder) OpcodePattern {
	return OpcodePattern{Mask: mask, Value: value, Decoder: decoder}
}

// opcodeBuckets is a top-level jump table keyed by the opcode's high nibble.
// Each bucket keeps the original precedence for that 4K region of the opcode space.
var opcodeBuckets = [16][]OpcodePattern{
	0x0: {
		masked(maskBitOp, valBTSTReg, decodeBTST), // BTST (register)
		masked(maskBitOp, valBTSTImm, decodeBTST), // BTST (immediate)
		masked(maskBitOp, valBCHGReg, decodeBCHG), // BCHG (register)
		masked(maskBitOp, valBCHGImm, decodeBCHG), // BCHG (immediate)
		masked(maskBitOp, valBCLRReg, decodeBCLR), // BCLR (register)
		masked(maskBitOp, valBCLRImm, decodeBCLR), // BCLR (immediate)
		masked(maskBitOp, valBSETReg, decodeBSET), // BSET (register)
		masked(maskBitOp, valBSETImm, decodeBSET), // BSET (immediate)
		masked(maskFF00, valADDI, decodeADDI),     // ADDI
		masked(maskFF00, valSUBI, decodeSUBI),     // SUBI
		masked(maskFF00, valANDI, decodeANDI),     // ANDI
		masked(maskFF00, valORI, decodeORI),       // ORI
		masked(maskFF00, valEORI, decodeEORI),     // EORI
		masked(maskFF00, valCMPI, decodeCMPI),     // CMPI
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
		masked(maskFFF0, valTRAP, decodeTRAP),
		masked(maskFB80, valMOVEMReg, decodeMOVEM), // MOVEM Reg→Mem
		masked(maskFB80, valMOVEMMem, decodeMOVEM), // MOVEM Mem→Reg
		masked(maskFF00, valCLR, decodeCLR),        // CLR
		masked(maskFF00, valNEG, decodeNEG),        // NEG
		masked(maskFF00, valNEGX, decodeNEGX),      // NEGX
		masked(maskFF00, valNOT, decodeNOT),        // NOT
		masked(maskFF00, valTST, decodeTST),        // TST
		masked(maskFFC0, valJSR, decodeJSR),        // JSR
		masked(maskFFC0, valJMP, decodeJMP),        // JMP
		masked(maskF1C0, valLEA, decodeLEA),        // LEA
		masked(maskFFC0, valPEA, decodePEA),        // PEA
	},
	0x6: {
		masked(maskF000, valBxx, decodeBxx), // BRA/BSR/Bcc
	},
	0x7: {
		masked(maskF100, valMOVEQ, decodeMOVEQ), // MOVEQ
	},
	0x8: {
		masked(maskF1F0, valSBCD, decodeSBCD), // SBCD
		masked(maskF1C0, valDIVU, decodeDIVU), // DIVU
		masked(maskF1C0, valDIVS, decodeDIVS), // DIVS
		masked(maskF000, valOR, decodeOR),     // OR
	},
	0x9: {
		masked(maskF000, valSUB, decodeSUB), // SUB
	},
	0xB: {
		masked(maskF000, valCMP, decodeCMP), // CMP/CMPA/CMPM/EOR
	},
	0xC: {
		masked(maskF1F0, valABCD, decodeABCD), // ABCD
		masked(maskF1C0, valMULU, decodeMULU), // MULU
		masked(maskF1C0, valMULS, decodeMULS), // MULS
		masked(maskF000, valAND, decodeAND),   // AND
	},
	0xD: {
		masked(maskF000, valADD, decodeADD), // ADD
	},
	0xE: {
		masked(maskF000, valSHIFT, decodeShiftRotate), // All ASL/ASR/LSL/LSR/ROL/ROR/ROXL/ROXR
	},
}

// OpcodeTable is the canonical ordered pattern table used by tests and tooling.
var OpcodeTable = flattenOpcodeBuckets()

// FindDecoder uses the opcode's high nibble as a jump-table index, then matches
// only against the patterns that can exist in that 4K region of the opcode space.
func FindDecoder(opcode uint16) OpcodeDecoder {
	for _, pattern := range opcodeBuckets[opcode>>12] {
		if (opcode & pattern.Mask) == pattern.Value {
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

func populateMetadata(inst *Instruction, mnemonic string, operands []Operand) {
	base, suffix, _ := strings.Cut(mnemonic, ".")
	inst.Metadata = Metadata{
		Mnemonic:     mnemonic,
		MnemonicBase: base,
		SizeSuffix:   suffix,
		Operands:     cloneOperands(operands),
	}

	for _, operand := range operands {
		if operand.BranchTarget != nil && inst.Metadata.BranchTarget == nil {
			target := *operand.BranchTarget
			inst.Metadata.BranchTarget = &target
		}
		if operand.Immediate != nil {
			inst.Metadata.ImmediateValues = append(inst.Metadata.ImmediateValues, *operand.Immediate)
		}
		if operand.EffectiveAddress != nil && operand.EffectiveAddress.Immediate != nil {
			inst.Metadata.ImmediateValues = append(inst.Metadata.ImmediateValues, *operand.EffectiveAddress.Immediate)
		}
	}
}

func cloneOperands(src []Operand) []Operand {
	if len(src) == 0 {
		return nil
	}
	dst := make([]Operand, len(src))
	for i, operand := range src {
		dst[i] = cloneOperand(operand)
	}
	return dst
}

func cloneOperand(operand Operand) Operand {
	cloned := operand
	if operand.Register != nil {
		reg := *operand.Register
		cloned.Register = &reg
	}
	if operand.Immediate != nil {
		imm := *operand.Immediate
		cloned.Immediate = &imm
	}
	if operand.EffectiveAddress != nil {
		ea := *operand.EffectiveAddress
		if operand.EffectiveAddress.Base != nil {
			base := *operand.EffectiveAddress.Base
			ea.Base = &base
		}
		if operand.EffectiveAddress.Displacement != nil {
			disp := *operand.EffectiveAddress.Displacement
			ea.Displacement = &disp
		}
		if operand.EffectiveAddress.AbsoluteAddress != nil {
			addr := *operand.EffectiveAddress.AbsoluteAddress
			ea.AbsoluteAddress = &addr
		}
		if operand.EffectiveAddress.ResolvedAddress != nil {
			addr := *operand.EffectiveAddress.ResolvedAddress
			ea.ResolvedAddress = &addr
		}
		if operand.EffectiveAddress.Immediate != nil {
			imm := *operand.EffectiveAddress.Immediate
			ea.Immediate = &imm
		}
		if operand.EffectiveAddress.Index != nil {
			idx := *operand.EffectiveAddress.Index
			ea.Index = &idx
		}
		cloned.EffectiveAddress = &ea
	}
	if operand.RegisterList != nil {
		cloned.RegisterList = append([]string(nil), operand.RegisterList...)
	}
	if operand.BranchTarget != nil {
		target := *operand.BranchTarget
		cloned.BranchTarget = &target
	}
	return cloned
}
