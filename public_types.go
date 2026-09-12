package m68kdasm

import (
	"fmt"

	"github.com/jenska/m68kdasm/internal/decoders"
)

type DecodeOptions struct {
	Symbolizer Symbolizer
	// CPU selects the target 68k family member. The zero value, M68000,
	// decodes plain 68000 opcodes only, preserving prior behavior.
	CPU CPU
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
	CPU                  = decoders.CPU
)

const (
	M68000 = decoders.M68000
	M68010 = decoders.M68010
	CPU32  = decoders.CPU32
	M68020 = decoders.M68020
	M68030 = decoders.M68030
	M68040 = decoders.M68040
	M68060 = decoders.M68060
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
