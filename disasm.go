package m68kdasm

import (
	"encoding/binary"
	"fmt"
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
		return nil, fmt.Errorf("nicht genügend Daten für Opcode an Adresse %08X", address)
	}

	// 68000 ist Big Endian
	opcode := binary.BigEndian.Uint16(data)

	inst := &Instruction{
		Address: address,
		Opcode:  opcode,
		Size:    2, // Minimale Größe ist ein Word (2 Bytes)
		Bytes:   data[:2],
	}

	// Hier beginnt die Dekodier-Logik.
	// Dies ist ein Skelett. Die 68k-ISA erfordert hier Bitmasken-Checks.
	switch {
	case opcode == 0x4E71:
		inst.Mnemonic = "NOP"
	case opcode == 0x4E75:
		inst.Mnemonic = "RTS"
	// MOVE.W Dn, Dm (Opcode 0011 ... -> 3xxx)
	// Vereinfacht: Nur Register-zu-Register (Mode 0)
	case opcode&0xF000 == 0x3000:
		dstReg := (opcode >> 9) & 0x7
		dstMode := (opcode >> 6) & 0x7
		srcMode := (opcode >> 3) & 0x7
		srcReg := opcode & 0x7

		if dstMode == 0 && srcMode == 0 {
			inst.Mnemonic = "MOVE.W"
			inst.Operands = fmt.Sprintf("D%d, D%d", srcReg, dstReg)
		}
	// Beispiel für ADD.W Dn, Dm (Opcode 1101 ...)
	// Maske 0xF000 prüft auf 'ADD' (0xD...), vereinfachtes Beispiel
	case opcode&0xF000 == 0xD000:
		regDest := (opcode >> 9) & 0x7
		regSrc := opcode & 0x7
		inst.Mnemonic = "ADD.W"
		inst.Operands = fmt.Sprintf("D%d, D%d", regSrc, regDest)
	default:
		// Unbekannte Instruktion als Hex-Werte ausgeben (DC.W)
		inst.Mnemonic = "DC.W"
		inst.Operands = fmt.Sprintf("$%04X", opcode)
	}

	// WICHTIG: Wenn die Instruktion Operanden im Speicher hat (Immediate Data, Offsets),
	// muss hier 'inst.Size' erhöht und 'inst.Bytes' erweitert werden.

	return inst, nil
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
