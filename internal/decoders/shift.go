package decoders

import "fmt"

// getMnemonicBase returns the mnemonic base for shift/rotate operations
func getMnemonicBase(shiftType uint16) string {
	switch shiftType {
	case 0:
		return "AS"
	case 1:
		return "LS"
	case 2:
		return "ROX"
	case 3:
		return "RO"
	default:
		return "?"
	}
}

// getDirectionStr returns "L" for left/up, "R" for right/down
func getDirectionStr(direction uint16) string {
	if direction == 0 {
		return "R"
	}
	return "L"
}

func decodeShiftRotate(data []byte, opcode uint16, inst *Instruction) error {
	dirStr := getDirectionStr((opcode >> 8) & 0x1)
	rotType := (opcode >> 9) & 0x7

	if rotType <= 3 {
		// Register form: shift a data register by an immediate or register count.
		reg := uint8(opcode & 0x7)
		mnemonic := fmt.Sprintf("%s%s.%s", getMnemonicBase(rotType), dirStr, getSizeString((opcode>>6)&0x3))

		var countStr string
		var countMeta Operand
		if (opcode>>5)&0x1 == 0 {
			count := (opcode >> 9) & 0x7
			if count == 0 {
				count = 8
			}
			countStr = fmt.Sprintf("#%d", count)
			countMeta = immediateOperand(countStr, uint32(count), 1)
		} else {
			countReg := uint8((opcode >> 9) & 0x7)
			countStr = fmt.Sprintf("D%d", countReg)
			countMeta = registerOperand(RegisterKindData, countReg)
		}
		setInstruction(data, inst, 2, mnemonic, fmt.Sprintf("%s, D%d", countStr, reg), countMeta, registerOperand(RegisterKindData, reg))
		return nil
	}

	// Memory form: shift a memory word by one.
	memMode := uint8((opcode >> 3) & 0x7)
	memReg := uint8(opcode & 0x7)
	mnemonic := fmt.Sprintf("%s%s.W", getMnemonicBase((opcode>>6)&0x3), dirStr)
	operand, extraWords, meta, err := decodeAddressingMode(data[2:], memMode, memReg, 2)
	if err != nil {
		return err
	}
	setInstruction(data, inst, 2+extraWords*2, mnemonic, operand, meta)
	return nil
}
