package decoders

import (
	"encoding/binary"
	"fmt"
)

func decodeLEA(data []byte, opcode uint16, inst *Instruction) error {
	regX := uint8((opcode >> 9) & 0x7)
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	operand, offset, err := decodeEA(data, 2, mode, reg)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, "LEA", fmt.Sprintf("%s, A%d", operand, regX))
	return nil
}

func decodePEA(data []byte, opcode uint16, inst *Instruction) error {
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	operand, offset, err := decodeEA(data, 2, mode, reg)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, "PEA", operand)
	return nil
}

func decodeSTOP(data []byte, opcode uint16, inst *Instruction) error {
	if len(data) < 4 {
		return fmt.Errorf("insufficient data for STOP")
	}
	immediate := binary.BigEndian.Uint16(data[2:4])
	setInstruction(data, inst, 4, "STOP", fmt.Sprintf("#%s", formatImmediate(uint32(immediate), 2)))
	return nil
}

func decodeTRAP(data []byte, opcode uint16, inst *Instruction) error {
	vector := opcode & 0xF
	setInstruction(data, inst, 2, "TRAP", fmt.Sprintf("#%d", vector))
	return nil
}

func decodeTRAPV(data []byte, opcode uint16, inst *Instruction) error {
	setInstruction(data, inst, 2, "TRAPV", "")
	return nil
}

func formatRegisterList(regListMask uint16, direction uint16) string {
	registers := []string{}
	if direction == 0 {
		for i := 0; i < 8; i++ {
			if regListMask&(1<<uint(i)) != 0 {
				registers = append(registers, fmt.Sprintf("D%d", i))
			}
		}
		for i := 0; i < 8; i++ {
			if regListMask&(1<<uint(8+i)) != 0 {
				registers = append(registers, fmt.Sprintf("A%d", i))
			}
		}
	} else {
		for i := 0; i < 8; i++ {
			if regListMask&(1<<uint(15-i)) != 0 {
				registers = append(registers, fmt.Sprintf("A%d", i))
			}
		}
		for i := 0; i < 8; i++ {
			if regListMask&(1<<uint(7-i)) != 0 {
				registers = append(registers, fmt.Sprintf("D%d", i))
			}
		}
	}
	return formatRegisterRange(registers)
}

func formatRegisterRange(registers []string) string {
	if len(registers) == 0 {
		return ""
	}
	result := ""
	i := 0
	for i < len(registers) {
		if result != "" {
			result += "/"
		}
		start := registers[i]
		end := start
		j := i + 1
		for j < len(registers) {
			prevNum := extractRegNum(registers[j-1])
			currNum := extractRegNum(registers[j])
			if prevNum >= 0 && currNum >= 0 && currNum == prevNum+1 {
				end = registers[j]
				j++
			} else {
				break
			}
		}
		if start == end {
			result += start
		} else {
			result += start + "-" + end
		}
		i = j
	}
	return result
}

func extractRegNum(regName string) int {
	if len(regName) < 2 {
		return -1
	}
	num := regName[1] - '0'
	if num <= 7 {
		return int(num)
	}
	return -1
}
