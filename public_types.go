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
	// FPU enables decoding of the 68881/68882 (or 68040/68060-integrated)
	// FPU instruction set (the F-line "general instruction" family:
	// FMOVE/FADD/FSUB/FMUL/FDIV/FCMP/FABS/FNEG/FSQRT/FTST/FNOP). FPU
	// presence is an attached-coprocessor question independent of CPU — a
	// bare 68020 with an external 68881 and a 68040's built-in FPU both
	// just set FPU: true. The zero value, false, preserves prior behavior
	// (F-line opcodes render as DC.W). See docs/design-fpu-mmu.md.
	FPU bool
	// MMU enables decoding of the 68851/68030 PMMU instruction set (PMOVE,
	// PMOVEFD, PFLUSHA, and friends — the F-line CpId-0 coprocessor space,
	// distinct from FPU's CpId-1 space). Like FPU, MMU presence is an
	// attached-coprocessor question independent of CPU — a bare 68020 with
	// an external 68851 and a 68030's on-chip PMMU both just set MMU:
	// true. The zero value, false, preserves prior behavior. Does not
	// apply to 68040/68060 MMU register access, which uses the ordinary
	// MOVEC opcode (already decoded unconditionally) rather than any
	// F-line opcode. See docs/design-fpu-mmu.md.
	MMU bool
	// Labels enables synthetic label generation for branch/call targets
	// that fall within a disassembled range and land on a decoded
	// instruction (e.g. rendering "BRA l00001010" and setting
	// Instruction.Label = "l00001010" on the instruction at that
	// address). Nil (the zero value) preserves prior behavior — no
	// labels, no rendering change, no extra cost. Meaningful only for
	// DisassembleRange/DisassembleRangeWithOptions and the
	// ELFDisassembler DisassembleSection*WithOptions methods; silently
	// has no effect on Decode/DecodeWithOptions/DecodeReaderAt*/
	// DecodeFunc*, which have no "rest of the stream" to find a forward
	// reference in. See docs/design-labels.md.
	Labels *LabelOptions
}

// LabelOptions configures synthetic label generation. The zero value
// (Prefix "") uses the default prefix "l" once Labels is non-nil.
type LabelOptions struct {
	// Prefix is prepended to the 8-hex-digit address to form a synthetic
	// label name (e.g. "l" -> "l00001010"). Defaults to "l" when Labels
	// is non-nil but Prefix is empty.
	Prefix string
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
	RegisterKindFP      = decoders.RegisterKindFP

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
