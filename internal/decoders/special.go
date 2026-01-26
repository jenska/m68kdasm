package decoders

import (
	"encoding/binary"
	"fmt"
)

func decodeLEA(data []byte, opcode uint16, inst *Instruction) error {
	regX := uint8((opcode >> 9) & 0x7)
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	operand, extraWords, err := decodeAddressingMode(data, mode, reg)
	if err != nil {
		return err
	}
	inst.Mnemonic = "LEA"
	inst.Operands = fmt.Sprintf("%s, A%d", operand, regX)
	inst.Size = uint32(2 + extraWords*2)
	if len(data) >= int(inst.Size) {
		inst.Bytes = data[:inst.Size]
	}
	return nil
}

func decodePEA(data []byte, opcode uint16, inst *Instruction) error {
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	operand, extraWords, err := decodeAddressingMode(data, mode, reg)
	if err != nil {
		return err
	}
	inst.Mnemonic = "PEA"
	inst.Operands = operand
	inst.Size = uint32(2 + extraWords*2)
	if len(data) >= int(inst.Size) {
		inst.Bytes = data[:inst.Size]
	}
	return nil
}

func decodeMOVEM(data []byte, opcode uint16, inst *Instruction) error {
	if len(data) < 4 {
		return fmt.Errorf("insufficient data for MOVEM")
	}
	direction := (opcode >> 10) & 0x1
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	regListMask := binary.BigEndian.Uint16(data[2:4])
	offset := 4
	var addrModeStr string
	var extraWords int
	var err error
	if mode != 0 || reg != 0 {
		addrModeStr, extraWords, err = decodeAddressingMode(data[offset:], mode, reg)
		if err != nil {
			return err
		}
		offset += extraWords * 2
	}
	inst.Mnemonic = "MOVEM"
	regList := formatRegisterList(regListMask, direction)
	if direction == 0 {
		inst.Operands = fmt.Sprintf("%s, %s", regList, addrModeStr)
	} else {
		inst.Operands = fmt.Sprintf("%s, %s", addrModeStr, regList)
	}
	inst.Size = uint32(offset)
	if len(data) >= offset {
		inst.Bytes = data[:offset]
	}
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
