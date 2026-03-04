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
		} else {
			countReg := (opcode >> 9) & 0x7
			countStr = fmt.Sprintf("D%d", countReg)
		}
		inst.Mnemonic = fmt.Sprintf("%s%s.%s", mnemonicBase, dirStr, sizeStr)
		inst.Operands = fmt.Sprintf("%s, D%d", countStr, reg)
		inst.Size = 2
		if len(data) >= 2 {
			inst.Bytes = data[:2]
		}
	} else {
		// Memory shift: extract addressing mode
		memMode := uint8((opcode >> 3) & 0x7)
		memReg := uint8(opcode & 0x7)
		memShiftType := (opcode >> 6) & 0x3
		mnemonicBase := getMnemonicBase(memShiftType)
		inst.Mnemonic = fmt.Sprintf("%s%s.W", mnemonicBase, dirStr)
		operand, extraWords, err := decodeAddressingMode(data[2:], memMode, memReg)
		if err != nil {
			return err
		}
		inst.Operands = operand
		inst.Size = uint32(2 + extraWords*2)
		if len(data) >= int(inst.Size) {
			inst.Bytes = data[:inst.Size]
		}
	}
	return nil
}
