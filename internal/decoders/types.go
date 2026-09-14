package decoders

import "strings"

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
	// RegisterKindFP identifies an FPU data register (FP0-FP7).
	RegisterKindFP RegisterKind = "fp"
)

type Register struct {
	Kind   RegisterKind
	Number uint8
}

type ImmediateValue struct {
	Value  uint32
	Signed int32
	Size   uint8
	// RawBytes holds the big-endian encoded bytes for immediates wider than
	// 32 bits (Size > 4) that Value/Signed cannot represent — the FPU's
	// double (8 bytes) and extended/packed-BCD (12 bytes) immediate
	// formats. Nil whenever Size <= 4, where Value/Signed are authoritative.
	RawBytes []byte
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
	// EAKindMemoryIndirect covers the 68020+ full-extension-word memory
	// indirect modes (pre-indexed and post-indexed, with An or PC base).
	// EffectiveAddress.PreIndexed distinguishes the two when Index != nil;
	// with Index == nil there is no pre/post distinction.
	EAKindMemoryIndirect EffectiveAddressKind = "memory_indirect"
)

type IndexRegister struct {
	Register Register
	Size     string
	// Scale is the 68020+ index scale factor (1, 2, 4, or 8). Brief-mode
	// (68000/68010/CPU32) index operands always have Scale 1.
	Scale uint8
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
	// OuterDisplacement is set for EAKindMemoryIndirect operands that encode
	// a non-null outer displacement.
	OuterDisplacement *int32
	// PreIndexed is meaningful only for EAKindMemoryIndirect with Index != nil:
	// true means the index is applied before the memory indirection, false
	// means after (post-indexed).
	PreIndexed bool
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
		imm.RawBytes = append([]byte(nil), operand.Immediate.RawBytes...)
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
			imm.RawBytes = append([]byte(nil), operand.EffectiveAddress.Immediate.RawBytes...)
			ea.Immediate = &imm
		}
		if operand.EffectiveAddress.Index != nil {
			idx := *operand.EffectiveAddress.Index
			ea.Index = &idx
		}
		if operand.EffectiveAddress.OuterDisplacement != nil {
			od := *operand.EffectiveAddress.OuterDisplacement
			ea.OuterDisplacement = &od
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
