package decoders

import "fmt"

func decodeShiftRotate(data []byte, opcode uint16, inst *Instruction) error {
	direction := (opcode >> 8) & 0x1
	size := (opcode >> 6) & 0x3
	reg := uint8(opcode & 0x7)
	sizeStr := []string{"B", "W", "L", "?"}[size]
	rotType := (opcode >> 9) & 0x7
	countIsReg := (opcode >> 5) & 0x1
	dirStr := "L"
	mnemonicBase := ""
	if rotType <= 3 {
		switch rotType {
		case 0:
			mnemonicBase = "AS"
		case 1:
			mnemonicBase = "LS"
		case 2:
			mnemonicBase = "ROX"
		case 3:
			mnemonicBase = "RO"
		}
		if direction == 0 {
			dirStr = "R"
		} else {
			dirStr = "L"
		}
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
	} else {
		memMode := (opcode >> 3) & 0x7
		memReg := uint8(opcode & 0x7)
		memShiftType := (opcode >> 6) & 0x3
		switch memShiftType {
		case 0:
			mnemonicBase = "AS"
		case 1:
			mnemonicBase = "LS"
		case 2:
			mnemonicBase = "ROX"
		case 3:
			mnemonicBase = "RO"
		}
		if direction == 0 {
			dirStr = "R"
		} else {
			dirStr = "L"
		}
		inst.Mnemonic = fmt.Sprintf("%s%s.W", mnemonicBase, dirStr)
		operand, extraWords, err := decodeAddressingMode(data[2:], uint8(memMode), memReg)
		if err != nil {
			return err
		}
		inst.Operands = operand
		inst.Size = uint32(2 + extraWords*2)
		if len(data) >= int(inst.Size) {
			inst.Bytes = data[:inst.Size]
		}
		return nil
	}
	inst.Size = 2
	if len(data) >= 2 {
		inst.Bytes = data[:2]
	}
	return nil
}
