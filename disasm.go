package m68kdasm

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jenska/m68kdasm/internal/decoders"
)

// Instruction repräsentiert eine einzelne assemblierte Instruktion.
type Instruction struct {
	Address        uint32
	Opcode         uint16
	Mnemonic       string
	Operands       string
	Size           uint32 // Größe der Instruktion in Bytes (2, 4, 6, etc.)
	Bytes          []byte // Die Rohdaten der Instruktion
	ExtensionWords []uint16
	Metadata       DecodeMetadata
	// Label is the resolved name for this instruction's own address —
	// the caller's own Symbolizer name if one resolves it, else a
	// synthetic label — set only when DecodeOptions.Labels was non-nil
	// and some operand elsewhere in the disassembled range targets this
	// address. Empty otherwise. Uses the same Symbolizer-first
	// precedence as operand rendering, so Label always matches whatever
	// name other instructions' operands show when they reference this
	// address. See docs/design-labels.md.
	Label string
}

// Assembly liefert den reinen Assembler-Code (Mnemonic + Operanden).
func (i Instruction) Assembly() string {
	if i.Operands == "" {
		return i.Mnemonic
	}
	return fmt.Sprintf("%s %s", i.Mnemonic, i.Operands)
}

// String liefert eine lesbare Repräsentation der Instruktion (z.B. für CLI-Output).
// When Label is set (see DecodeOptions.Labels, docs/design-labels.md), a
// "name:" line precedes the address/assembly line, matching how a real
// assembly listing shows a label definition on its own line above the
// instruction it names.
func (i Instruction) String() string {
	if i.Label != "" {
		return fmt.Sprintf("%s:\n%08X: %s", i.Label, i.Address, i.Assembly())
	}
	return fmt.Sprintf("%08X: %s", i.Address, i.Assembly())
}

type addressReader interface {
	ReadAtAddress(address uint32, p []byte) (int, error)
}

func (f ReadFunc) ReadAtAddress(address uint32, p []byte) (int, error) {
	return f(address, p)
}

type readerAtAdapter struct {
	reader io.ReaderAt
}

func (a readerAtAdapter) ReadAtAddress(address uint32, p []byte) (int, error) {
	return a.reader.ReadAt(p, int64(address))
}

// Decode liest eine einzelne Instruktion an der gegebenen Adresse aus dem Byte-Slice.
func Decode(data []byte, address uint32) (*Instruction, error) {
	return DecodeWithOptions(data, address, DecodeOptions{})
}

func DecodeWithOptions(data []byte, address uint32, opts DecodeOptions) (*Instruction, error) {
	return decodeInstruction(data, address, nil, opts)
}

func DecodeReaderAt(reader io.ReaderAt, address uint32) (*Instruction, error) {
	return DecodeReaderAtWithOptions(reader, address, DecodeOptions{})
}

func DecodeReaderAtWithOptions(reader io.ReaderAt, address uint32, opts DecodeOptions) (*Instruction, error) {
	return decodeInstruction(nil, address, readerAtAdapter{reader: reader}, opts)
}

func DecodeFunc(read ReadFunc, address uint32) (*Instruction, error) {
	return DecodeFuncWithOptions(read, address, DecodeOptions{})
}

func DecodeFuncWithOptions(read ReadFunc, address uint32, opts DecodeOptions) (*Instruction, error) {
	return decodeInstruction(nil, address, read, opts)
}

// DisassembleRange disassembliert einen Speicherbereich sequenziell.
func DisassembleRange(data []byte, startAddress uint32) ([]Instruction, error) {
	return DisassembleRangeWithOptions(data, startAddress, DecodeOptions{})
}

func DisassembleRangeWithOptions(data []byte, startAddress uint32, opts DecodeOptions) ([]Instruction, error) {
	// Every instruction is at least one 16-bit word, so len(data)/2 is an upper
	// bound on the instruction count.
	instructions := make([]Instruction, 0, len(data)/2)
	offset := 0

	for offset < len(data) {
		inst, err := DecodeWithOptions(data[offset:], startAddress+uint32(offset), opts)
		if err != nil {
			return instructions, err
		}
		instructions = append(instructions, *inst)
		offset += int(inst.Size)
	}

	if opts.Labels != nil {
		applyLabels(instructions, opts)
	}

	return instructions, nil
}

// labelCreatingMnemonics is the narrow, mnemonic-aware classification that
// decides which instructions can create a label from their own
// OperandKindEffectiveAddr operand (see docs/design-labels.md's
// "Classification creates a label; rendering is universal" — this set
// decides what gets a name, but every operand referencing that address,
// regardless of its own instruction's mnemonic, renders it once created).
// JSR/JMP are unambiguous control-flow transfers. PEA/LEA are included to
// catch indirect-call trampolines ("LEA sub,A0" then "JSR (A0)") even
// though this means a LEA/PEA loading a plain data-buffer address becomes
// a label too — an accepted trade-off, not an oversight (see Decisions in
// docs/design-labels.md). No other mnemonic is classified this way: a
// plain MOVE.L $addr,D0 never creates a label on its own.
var labelCreatingMnemonics = map[string]bool{
	"JSR": true,
	"JMP": true,
	"PEA": true,
	"LEA": true,
}

// applyLabels is DisassembleRangeWithOptions's pass 2 (see
// docs/design-labels.md): pass 1 above decodes sequentially and has no
// knowledge of instructions later in the stream, so a forward branch can't
// be named until every instruction in the range is known. This pass walks
// the now-complete slice to collect candidate target addresses, names the
// ones that land on a decoded instruction, and re-renders every
// instruction's Operands through the existing Symbolizer-driven rendering
// path — via a composite Symbolizer that tries the caller's own first —
// so label rendering reuses formatOperand's precedence chain rather than
// adding a parallel one.
func applyLabels(instructions []Instruction, opts DecodeOptions) {
	prefix := "l"
	if opts.Labels.Prefix != "" {
		prefix = opts.Labels.Prefix
	}

	instByAddr := make(map[uint32]int, len(instructions))
	for i, inst := range instructions {
		instByAddr[inst.Address] = i
	}

	candidates := make(map[uint32]bool)
	for _, inst := range instructions {
		labelCreating := labelCreatingMnemonics[inst.Metadata.MnemonicBase]
		for _, operand := range inst.Metadata.Operands {
			if operand.Kind == OperandKindBranchTarget && operand.BranchTarget != nil {
				candidates[*operand.BranchTarget] = true
				continue
			}
			if operand.Kind == OperandKindEffectiveAddr && labelCreating && operand.EffectiveAddress != nil {
				if addr := operand.EffectiveAddress.ResolvedAddress; addr != nil {
					candidates[*addr] = true
				} else if addr := operand.EffectiveAddress.AbsoluteAddress; addr != nil {
					candidates[*addr] = true
				}
			}
		}
	}

	labels := make(map[uint32]string, len(candidates))
	for addr := range candidates {
		if _, ok := instByAddr[addr]; !ok {
			// Doesn't land on a decoded instruction (self-modifying code,
			// data-in-code, outside the range, or inside a gap a partial
			// decode left undecoded) — stays raw hex, never a dangling
			// label with nothing to attach a definition line to.
			continue
		}
		labels[addr] = fmt.Sprintf("%s%08X", prefix, addr)
	}

	if len(labels) == 0 {
		return
	}

	symbolizer := labelSymbolizer{primary: opts.Symbolizer, labels: labels}
	for i := range instructions {
		// Label uses the same Symbolizer-first precedence as operand
		// rendering below, so it always matches whatever name other
		// instructions' operands show when they reference this address
		// — a caller-supplied Symbolizer name, not just a synthetic one,
		// when the Symbolizer covers this address.
		if _, isTarget := labels[instructions[i].Address]; isTarget {
			if name, ok := symbolizer.Symbolize(instructions[i].Address); ok {
				instructions[i].Label = name
			}
		}
		if len(instructions[i].Metadata.Operands) > 0 {
			instructions[i].Operands = formatOperands(instructions[i].Metadata.Operands, symbolizer)
		}
	}
}

// labelSymbolizer composes a caller-supplied Symbolizer (tried first, so a
// caller's own naming always wins) with the synthetic label table applyLabels
// built. It implements Symbolizer itself, so it plugs directly into the
// existing formatOperand/formatOperands rendering path with no changes to
// either.
type labelSymbolizer struct {
	primary Symbolizer
	labels  map[uint32]string
}

func (l labelSymbolizer) Symbolize(address uint32) (string, bool) {
	if l.primary != nil {
		if name, ok := l.primary.Symbolize(address); ok {
			return name, ok
		}
	}
	name, ok := l.labels[address]
	return name, ok
}

func decodeInstruction(initial []byte, address uint32, reader addressReader, opts DecodeOptions) (*Instruction, error) {
	data := initial
	if reader != nil {
		data = append([]byte(nil), initial...)
	}

	if len(data) < 2 {
		if reader == nil {
			return nil, &PartialDecodeError{
				Address: address,
				Have:    len(data),
				Missing: 2 - len(data),
				Context: "opcode",
			}
		}
		if err := readUntil(&data, address, reader, 2); err != nil {
			return nil, partialError(address, len(data), 2, "opcode", err)
		}
	}

	opcode := binary.BigEndian.Uint16(data[:2])
	decoder := decoders.FindDecoder(opcode, opts.CPU, opts.FPU, opts.MMU)
	if decoder == nil {
		return finalizeInstruction(decoders.DecodeUnknown(data, address, opcode), opts), nil
	}

	for {
		decoderInst := &decoders.Instruction{
			Address: address,
			Opcode:  opcode,
			Size:    2,
			Bytes:   data[:2],
		}

		err := decoder(data, opcode, decoderInst, opts.CPU)
		if err == nil {
			return finalizeInstruction(decoderInst, opts), nil
		}

		var needMore *decoders.NeedMoreError
		if !errors.As(err, &needMore) {
			return nil, err
		}

		requiredLen := len(data) + needMore.Missing
		if reader == nil {
			return nil, partialError(address, len(data), requiredLen, needMore.Context, nil)
		}
		fillErr := readUntil(&data, address, reader, requiredLen)
		if fillErr != nil {
			return nil, partialError(address, len(data), requiredLen, needMore.Context, fillErr)
		}
	}
}

func readUntil(data *[]byte, address uint32, reader addressReader, need int) error {
	for len(*data) < need {
		chunk := make([]byte, need-len(*data))
		n, err := reader.ReadAtAddress(address+uint32(len(*data)), chunk)
		if n > 0 {
			*data = append(*data, chunk[:n]...)
		}
		if len(*data) >= need {
			return nil
		}
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrUnexpectedEOF
		}
	}
	return nil
}

func partialError(address uint32, have, required int, context string, cause error) error {
	return &PartialDecodeError{
		Address: address,
		Have:    have,
		Missing: required - have,
		Context: context,
		Cause:   cause,
	}
}

func finalizeInstruction(decoderInst *decoders.Instruction, opts DecodeOptions) *Instruction {
	inst := &Instruction{
		Address:        decoderInst.Address,
		Opcode:         decoderInst.Opcode,
		Mnemonic:       decoderInst.Mnemonic,
		Operands:       decoderInst.Operands,
		Size:           decoderInst.Size,
		Bytes:          append([]byte(nil), decoderInst.Bytes...),
		ExtensionWords: append([]uint16(nil), decoderInst.ExtensionWords...),
		Metadata:       decoderInst.Metadata,
	}

	if opts.Symbolizer != nil && len(inst.Metadata.Operands) > 0 {
		inst.Operands = formatOperands(inst.Metadata.Operands, opts.Symbolizer)
	}

	return inst
}

func formatOperands(operands []Operand, symbolizer Symbolizer) string {
	rendered := make([]string, len(operands))
	for i, operand := range operands {
		rendered[i] = formatOperand(operand, symbolizer)
	}
	return strings.Join(rendered, ", ")
}

func formatOperand(operand Operand, symbolizer Symbolizer) string {
	if operand.BranchTarget != nil {
		if symbol, ok := symbolizer.Symbolize(*operand.BranchTarget); ok {
			return symbol
		}
	}
	if operand.EffectiveAddress != nil {
		if operand.EffectiveAddress.ResolvedAddress != nil {
			if symbol, ok := symbolizer.Symbolize(*operand.EffectiveAddress.ResolvedAddress); ok {
				return symbol
			}
		}
		if operand.EffectiveAddress.AbsoluteAddress != nil {
			if symbol, ok := symbolizer.Symbolize(*operand.EffectiveAddress.AbsoluteAddress); ok {
				return symbol
			}
		}
	}
	return operand.Text
}
