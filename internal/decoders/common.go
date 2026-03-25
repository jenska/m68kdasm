package decoders

import (
	"encoding/binary"
	"fmt"
)

var sizeNames = [...]string{"B", "W", "L", "?"}

// decodeNOP - No Operation (exact opcode: 0x4E71)
func decodeNOP(data []byte, opcode uint16, inst *Instruction) error {
	setInstruction(data, inst, 2, "NOP", "")
	return nil
}

// decodeRTS - Return from Subroutine (exact opcode: 0x4E75)
func decodeRTS(data []byte, opcode uint16, inst *Instruction) error {
	setInstruction(data, inst, 2, "RTS", "")
	return nil
}

// getSizeString converts 68000 size field (bits 6-7) to string
// Maps: 0=B (byte), 1=W (word), 2=L (long), 3=? (undefined)
func getSizeString(size uint16) string {
	if int(size) < len(sizeNames) {
		return sizeNames[size]
	}
	return "?"
}

// buildDirectedOperands creates reversed operands based on direction bit
// direction == 0: src, dstReg format
// direction != 0: dstReg, src format
func buildDirectedOperands(direction uint16, src string, dstReg uint8) string {
	if direction == 0 {
		return fmt.Sprintf("%s, D%d", src, dstReg)
	}
	return fmt.Sprintf("D%d, %s", dstReg, src)
}

// setInstructionSize sets the instruction's Size and Bytes fields
func setInstructionSize(data []byte, inst *Instruction, offset int) {
	inst.Size = uint32(offset)
	if len(data) >= offset {
		inst.Bytes = data[:offset]
	}
}

func setInstruction(data []byte, inst *Instruction, size int, mnemonic, operands string) {
	inst.Mnemonic = mnemonic
	inst.Operands = operands
	setInstructionSize(data, inst, size)
}

func decodeEA(data []byte, offset int, mode, reg uint8) (string, int, error) {
	operand, extraWords, err := decodeAddressingMode(data[offset:], mode, reg)
	if err != nil {
		return "", offset, err
	}
	return operand, offset + extraWords*2, nil
}

func decodeDirectedBinaryOp(mnemonic string, data []byte, opcode uint16, inst *Instruction) error {
	direction := (opcode >> 8) & 0x1
	sizeStr := getSizeString((opcode >> 6) & 0x3)
	dstReg := uint8((opcode >> 9) & 0x7)
	srcMode := uint8((opcode >> 3) & 0x7)
	srcReg := uint8(opcode & 0x7)

	srcOperand, offset, err := decodeEA(data, 2, srcMode, srcReg)
	if err != nil {
		return err
	}

	setInstruction(data, inst, offset, mnemonic+"."+sizeStr, buildDirectedOperands(direction, srcOperand, dstReg))
	return nil
}

func immediateSpec(size uint16, longImmediate bool, mnemonic string) (string, int, error) {
	switch size {
	case 0:
		return "B", 2, nil
	case 1:
		return "W", 2, nil
	case 2:
		if longImmediate {
			return "L", 4, nil
		}
		return "L", 2, nil
	default:
		return "", 0, fmt.Errorf("unknown %s size: %d", mnemonic, size)
	}
}

func readImmediate(data []byte, offset, size int, mnemonic string) (uint32, int, error) {
	if len(data) < offset+size {
		return 0, offset, fmt.Errorf("insufficient data for %s immediate", mnemonic)
	}

	switch size {
	case 2:
		return uint32(binary.BigEndian.Uint16(data[offset : offset+2])), offset + 2, nil
	case 4:
		return binary.BigEndian.Uint32(data[offset : offset+4]), offset + 4, nil
	default:
		return 0, offset, fmt.Errorf("unsupported immediate size for %s: %d", mnemonic, size)
	}
}
