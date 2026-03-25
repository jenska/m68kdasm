package m68kdasm

import (
	"encoding/binary"
	"fmt"

	"github.com/jenska/m68kdasm/internal/decoders"
)

// Instruction repräsentiert eine einzelne assemblierte Instruktion.
type Instruction struct {
	Address  uint32
	Opcode   uint16
	Mnemonic string
	Operands string
	Size     uint32 // Größe der Instruktion in Bytes (2, 4, 6, etc.)
	Bytes    []byte // Die Rohdaten der Instruktion
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

// Decode liest eine einzelne Instruktion an der gegebenen Adresse aus dem Byte-Slice.
func Decode(data []byte, address uint32) (*Instruction, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("insufficient data for opcode at address %08X", address)
	}

	// 68000 ist Big Endian
	opcode := binary.BigEndian.Uint16(data)

	// Use hierarchical jump table for efficient decoder lookup
	decoder := decoders.FindDecoder(opcode)
	if decoder != nil {
		decoderInst := &decoders.Instruction{
			Address: address,
			Opcode:  opcode,
			Size:    2,
			Bytes:   data[:2],
		}
		if err := decoder(data, opcode, decoderInst); err != nil {
			return nil, err
		}
		return &Instruction{
			Address:  decoderInst.Address,
			Opcode:   decoderInst.Opcode,
			Mnemonic: decoderInst.Mnemonic,
			Operands: decoderInst.Operands,
			Size:     decoderInst.Size,
			Bytes:    decoderInst.Bytes,
		}, nil
	}

	// Unbekannte Instruktion als Hex-Werte ausgeben (DC.W)
	return &Instruction{
		Address:  address,
		Opcode:   opcode,
		Mnemonic: "DC.W",
		Operands: fmt.Sprintf("$%04X", opcode),
		Size:     2, // Minimale Größe ist ein Word (2 Bytes)
		Bytes:    data[:2],
	}, nil
}

// DisassembleRange disassembliert einen Speicherbereich sequenziell.
func DisassembleRange(data []byte, startAddress uint32) ([]Instruction, error) {
	var instructions []Instruction
	offset := 0

	for offset < len(data) {
		inst, err := Decode(data[offset:], startAddress+uint32(offset))
		if err != nil {
			return instructions, err
		}
		instructions = append(instructions, *inst)
		offset += int(inst.Size)
	}

	return instructions, nil
}
