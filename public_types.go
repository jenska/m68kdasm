package m68kdasm

import "fmt"

type DecodeOptions struct {
	Symbolizer Symbolizer
}

type Symbolizer interface {
	Symbolize(address uint32) (string, bool)
}

type SymbolizeFunc func(address uint32) (string, bool)

func (f SymbolizeFunc) Symbolize(address uint32) (string, bool) {
	return f(address)
}

type ReadFunc func(address uint32, p []byte) (int, error)

type DecodeMetadata struct {
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

type PartialDecodeError struct {
	Address uint32
	Have    int
	Missing int
	Context string
	Cause   error
}

func (e *PartialDecodeError) Error() string {
	msg := fmt.Sprintf("need %d more byte(s) for %s at address %08X", e.Missing, e.Context, e.Address)
	if e.Cause != nil {
		return msg + ": " + e.Cause.Error()
	}
	return msg
}
