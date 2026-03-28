package decoders

import (
	"encoding/binary"
	"fmt"
)

func decodeLEA(data []byte, opcode uint16, inst *Instruction) error {
	regX := uint8((opcode >> 9) & 0x7)
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	operand, offset, meta, err := decodeEA(data, 2, mode, reg)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, "LEA", fmt.Sprintf("%s, A%d", operand, regX), meta, registerOperand(RegisterKindAddress, regX))
	return nil
}

func decodePEA(data []byte, opcode uint16, inst *Instruction) error {
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	operand, offset, meta, err := decodeEA(data, 2, mode, reg)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, "PEA", operand, meta)
	return nil
}

func decodeSWAP(data []byte, opcode uint16, inst *Instruction) error {
	reg := uint8(opcode & 0x7)
	regText := fmt.Sprintf("D%d", reg)
	setInstruction(data, inst, 2, "SWAP", regText, registerOperand(RegisterKindData, reg))
	return nil
}

func decodeSTOP(data []byte, opcode uint16, inst *Instruction) error {
	if err := requireLength(data, 4, "STOP immediate"); err != nil {
		return err
	}
	immediate := binary.BigEndian.Uint16(data[2:4])
	immText := fmt.Sprintf("#%s", formatImmediate(uint32(immediate), 2))
	setInstruction(data, inst, 4, "STOP", immText, immediateOperand(immText, uint32(immediate), 2))
	return nil
}

func decodeTRAP(data []byte, opcode uint16, inst *Instruction) error {
	vector := opcode & 0xF
	immText := fmt.Sprintf("#%d", vector)
	setInstruction(data, inst, 2, "TRAP", immText, immediateOperand(immText, uint32(vector), 1))
	return nil
}

func decodeTRAPV(data []byte, opcode uint16, inst *Instruction) error {
	setInstruction(data, inst, 2, "TRAPV", "")
	return nil
}

func formatRegisterList(regListMask uint16) (string, []string) {
	registers := []string{}
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
	return formatRegisterRange(registers), registers
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
