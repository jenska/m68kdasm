package decoders

import (
	"encoding/binary"
	"fmt"
)

// decodeADD - Add (generisch für alle Adressierungsmodi)
// ADD Format: 1101 ddd ooo sss rrr
func decodeADD(data []byte, opcode uint16, inst *Instruction) error {
	direction := (opcode >> 8) & 0x1
	size := (opcode >> 6) & 0x3

	sizeStr := "?"
	switch size {
	case 0:
		sizeStr = "B"
	case 1:
		sizeStr = "W"
	case 2:
		sizeStr = "L"
	default:
		return fmt.Errorf("unbekannte ADD-Größe: %d", size)
	}

	dstReg := uint8((opcode >> 9) & 0x7)
	srcMode := uint8((opcode >> 3) & 0x7)
	srcReg := uint8(opcode & 0x7)

	offset := 2
	srcStr, srcExtraWords, err := decodeAddressingMode(data, srcMode, srcReg)
	if err != nil {
		return err
	}
	offset += srcExtraWords * 2

	var dstStr string

	if direction == 1 {
		dstMode := uint8((opcode >> 6) & 0x7)
		var dstExtra int
		dstStr, dstExtra, _ = decodeAddressingMode(data[offset:], dstMode, dstReg)
		offset += dstExtra * 2
	}

	inst.Mnemonic = "ADD." + sizeStr

	if direction == 0 {
		inst.Operands = fmt.Sprintf("%s, D%d", srcStr, dstReg)
	} else {
		inst.Operands = fmt.Sprintf("D%d, %s", dstReg, dstStr)
	}

	inst.Size = uint32(offset)
	if len(data) >= offset {
		inst.Bytes = data[:offset]
	}

	return nil
}

// decodeSUB - Subtract (generisch für alle Adressierungsmodi)
// SUB Format: 1001 ddd ooo sss rrr
func decodeSUB(data []byte, opcode uint16, inst *Instruction) error {
	direction := (opcode >> 8) & 0x1
	size := (opcode >> 6) & 0x3

	sizeStr := "?"
	switch size {
	case 0:
		sizeStr = "B"
	case 1:
		sizeStr = "W"
	case 2:
		sizeStr = "L"
	default:
		return fmt.Errorf("unbekannte SUB-Größe: %d", size)
	}

	dstReg := uint8((opcode >> 9) & 0x7)
	srcMode := uint8((opcode >> 3) & 0x7)
	srcReg := uint8(opcode & 0x7)

	offset := 2
	srcStr, srcExtraWords, err := decodeAddressingMode(data, srcMode, srcReg)
	if err != nil {
		return err
	}
	offset += srcExtraWords * 2

	var dstStr string

	if direction == 1 {
		dstMode := uint8((opcode >> 6) & 0x7)
		var dstExtra int
		dstStr, dstExtra, _ = decodeAddressingMode(data[offset:], dstMode, dstReg)
		offset += dstExtra * 2
	}

	inst.Mnemonic = "SUB." + sizeStr

	if direction == 0 {
		inst.Operands = fmt.Sprintf("%s, D%d", srcStr, dstReg)
	} else {
		inst.Operands = fmt.Sprintf("D%d, %s", dstReg, dstStr)
	}

	inst.Size = uint32(offset)
	if len(data) >= offset {
		inst.Bytes = data[:offset]
	}

	return nil
}

// decodeADDI - Add Immediate
// Format: 0000 0110 sz 000 mmm rrr (sz: 00=Byte, 01=Word, 10=Long)
func decodeADDI(data []byte, opcode uint16, inst *Instruction) error {
	if len(data) < 4 {
		return fmt.Errorf("nicht genügend Daten für ADDI")
	}

	size := (opcode >> 6) & 0x3
	sizeStr := "?"
	immSize := 2
	switch size {
	case 0:
		sizeStr = "B"
		immSize = 2
	case 1:
		sizeStr = "W"
		immSize = 2
	case 2:
		sizeStr = "L"
		immSize = 4
	default:
		return fmt.Errorf("unbekannte ADDI-Größe: %d", size)
	}

	dstMode := uint8((opcode >> 3) & 0x7)
	dstReg := uint8(opcode & 0x7)

	offset := 2
	immediate := uint32(0)

	if immSize == 2 {
		if len(data) < offset+2 {
			return fmt.Errorf("nicht genügend Daten für ADDI Immediate")
		}
		immediate = uint32(binary.BigEndian.Uint16(data[offset : offset+2]))
		offset += 2
	} else {
		if len(data) < offset+4 {
			return fmt.Errorf("nicht genügend Daten für ADDI Immediate")
		}
		immediate = binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4
	}

	dstOperand, extraWords, err := decodeAddressingMode(data[offset:], dstMode, dstReg)
	if err != nil {
		return err
	}
	offset += extraWords * 2

	inst.Mnemonic = fmt.Sprintf("ADDI.%s", sizeStr)
	immStr := formatImmediate(immediate, immSize)
	inst.Operands = fmt.Sprintf("#%s, %s", immStr, dstOperand)
	inst.Size = uint32(offset)
	if len(data) >= offset {
		inst.Bytes = data[:offset]
	}

	return nil
}

// decodeSUBI - Subtract Immediate
// Format: 0000 0100 sz 000 mmm rrr
func decodeSUBI(data []byte, opcode uint16, inst *Instruction) error {
	if len(data) < 4 {
		return fmt.Errorf("nicht genügend Daten für SUBI")
	}

	size := (opcode >> 6) & 0x3
	sizeStr := "?"
	immSize := 2
	switch size {
	case 0:
		sizeStr = "B"
		immSize = 2
	case 1:
		sizeStr = "W"
		immSize = 2
	case 2:
		sizeStr = "L"
		immSize = 4
	default:
		return fmt.Errorf("unbekannte SUBI-Größe: %d", size)
	}

	dstMode := uint8((opcode >> 3) & 0x7)
	dstReg := uint8(opcode & 0x7)

	offset := 2
	immediate := uint32(0)

	if immSize == 2 {
		if len(data) < offset+2 {
			return fmt.Errorf("nicht genügend Daten für SUBI Immediate")
		}
		immediate = uint32(binary.BigEndian.Uint16(data[offset : offset+2]))
		offset += 2
	} else {
		if len(data) < offset+4 {
			return fmt.Errorf("nicht genügend Daten für SUBI Immediate")
		}
		immediate = binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4
	}

	dstOperand, extraWords, err := decodeAddressingMode(data[offset:], dstMode, dstReg)
	if err != nil {
		return err
	}
	offset += extraWords * 2

	inst.Mnemonic = fmt.Sprintf("SUBI.%s", sizeStr)
	immStr := formatImmediate(immediate, immSize)
	inst.Operands = fmt.Sprintf("#%s, %s", immStr, dstOperand)
	inst.Size = uint32(offset)
	if len(data) >= offset {
		inst.Bytes = data[:offset]
	}

	return nil
}
