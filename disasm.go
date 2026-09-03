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
}

// Assembly liefert den reinen Assembler-Code (Mnemonic + Operanden).
func (i Instruction) Assembly() string {
	if i.Operands == "" {
		return i.Mnemonic
	}
	return fmt.Sprintf("%s %s", i.Mnemonic, i.Operands)
}

// String liefert eine lesbare Repräsentation der Instruktion (z.B. für CLI-Output).
func (i Instruction) String() string {
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
	var instructions []Instruction
	offset := 0

	for offset < len(data) {
		inst, err := DecodeWithOptions(data[offset:], startAddress+uint32(offset), opts)
		if err != nil {
			return instructions, err
		}
		instructions = append(instructions, *inst)
		offset += int(inst.Size)
	}

	return instructions, nil
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
	decoder := decoders.FindDecoder(opcode)
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

		err := decoder(data, opcode, decoderInst)
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
