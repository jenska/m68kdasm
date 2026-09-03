package m68kdasm

import (
	"fmt"

	"github.com/jenska/m68kdasm/internal/decoders"
)

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

// The structured decode types are defined once in the internal decoders package
// and re-exported here so callers have a single source of truth.
type (
	DecodeMetadata       = decoders.Metadata
	Operand              = decoders.Operand
	OperandKind          = decoders.OperandKind
	Register             = decoders.Register
	RegisterKind         = decoders.RegisterKind
	ImmediateValue       = decoders.ImmediateValue
	EffectiveAddress     = decoders.EffectiveAddress
	EffectiveAddressKind = decoders.EffectiveAddressKind
	IndexRegister        = decoders.IndexRegister
)

const (
	OperandKindRegister      = decoders.OperandKindRegister
	OperandKindImmediate     = decoders.OperandKindImmediate
	OperandKindEffectiveAddr = decoders.OperandKindEffectiveAddr
	OperandKindRegisterList  = decoders.OperandKindRegisterList
	OperandKindBranchTarget  = decoders.OperandKindBranchTarget

	RegisterKindData    = decoders.RegisterKindData
	RegisterKindAddress = decoders.RegisterKindAddress
	RegisterKindPC      = decoders.RegisterKindPC

	EAKindDataRegisterDirect    = decoders.EAKindDataRegisterDirect
	EAKindAddressRegisterDirect = decoders.EAKindAddressRegisterDirect
	EAKindAddressIndirect       = decoders.EAKindAddressIndirect
	EAKindPostIncrement         = decoders.EAKindPostIncrement
	EAKindPreDecrement          = decoders.EAKindPreDecrement
	EAKindDisplacement          = decoders.EAKindDisplacement
	EAKindIndex                 = decoders.EAKindIndex
	EAKindAbsoluteShort         = decoders.EAKindAbsoluteShort
	EAKindAbsoluteLong          = decoders.EAKindAbsoluteLong
	EAKindPCDisplacement        = decoders.EAKindPCDisplacement
	EAKindPCIndex               = decoders.EAKindPCIndex
	EAKindImmediate             = decoders.EAKindImmediate
)

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
