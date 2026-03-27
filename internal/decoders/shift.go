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
	direction := (opcode >> 8) & 0x1
	size := (opcode >> 6) & 0x3
	reg := uint8(opcode & 0x7)
	sizeStr := []string{"B", "W", "L", "?"}[size]
	rotType := (opcode >> 9) & 0x7
	countIsReg := (opcode >> 5) & 0x1
	dirStr := getDirectionStr(direction)

	if rotType <= 3 {
		// Register shift: extract shift count
		mnemonicBase := getMnemonicBase(uint16(rotType))
		var countStr string
		if countIsReg == 0 {
			count := (opcode >> 9) & 0x7
			if count == 0 {
				count = 8
			}
			countStr = fmt.Sprintf("#%d", count)
			setInstruction(data, inst, 2, fmt.Sprintf("%s%s.%s", mnemonicBase, dirStr, sizeStr), fmt.Sprintf("%s, D%d", countStr, reg), immediateOperand(countStr, uint32(count), 1), registerOperand(RegisterKindData, reg))
		} else {
			countReg := (opcode >> 9) & 0x7
			countStr = fmt.Sprintf("D%d", countReg)
			setInstruction(data, inst, 2, fmt.Sprintf("%s%s.%s", mnemonicBase, dirStr, sizeStr), fmt.Sprintf("%s, D%d", countStr, reg), registerOperand(RegisterKindData, uint8(countReg)), registerOperand(RegisterKindData, reg))
		}
	} else {
		// Memory shift: extract addressing mode
		memMode := uint8((opcode >> 3) & 0x7)
		memReg := uint8(opcode & 0x7)
		memShiftType := (opcode >> 6) & 0x3
		mnemonicBase := getMnemonicBase(memShiftType)
		mnemonic := fmt.Sprintf("%s%s.W", mnemonicBase, dirStr)
		operand, extraWords, meta, err := decodeAddressingMode(data[2:], memMode, memReg, 2)
		if err != nil {
			return err
		}
		setInstruction(data, inst, 2+extraWords*2, mnemonic, operand, meta)
	}
	return nil
}
