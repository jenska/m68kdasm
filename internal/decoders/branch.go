package decoders

import (
	"encoding/binary"
	"fmt"
)

var branchCondNames = [...]string{
	"T", "F", "HI", "LS", "CC", "CS", "NE", "EQ",
	"VC", "VS", "PL", "MI", "GE", "LT", "GT", "LE",
}

func decodeBRA(data []byte, opcode uint16, inst *Instruction) error {
	offset := 2
	condition := (opcode >> 8) & 0x0F
	condStr := "?"
	if condition < uint16(len(branchCondNames)) {
		condStr = branchCondNames[condition]
	}
	displacement := int8(opcode & 0xFF)
	switch displacement {
	case 0:
		if len(data) < offset+2 {
			return fmt.Errorf("insufficient data for B%s.W", condStr)
		}
		displacement16 := int16(binary.BigEndian.Uint16(data[offset : offset+2]))
		offset += 2
		inst.Mnemonic = fmt.Sprintf("B%s.W", condStr)
		inst.Operands = fmt.Sprintf("$%04X", uint16(int32(displacement16)+int32(offset)-2))
	case -1:
		if len(data) < offset+4 {
			return fmt.Errorf("insufficient data for B%s.L", condStr)
		}
		displacement32 := int32(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4
		inst.Mnemonic = fmt.Sprintf("B%s.L", condStr)
		inst.Operands = fmt.Sprintf("$%08X", uint32(displacement32+int32(offset)-4))
	default:
		inst.Mnemonic = fmt.Sprintf("B%s.S", condStr)
		targetAddr := int32(displacement) + int32(offset)
		inst.Operands = fmt.Sprintf("$%04X", uint16(targetAddr))
	}
	inst.Size = uint32(offset)
	if len(data) >= offset {
		inst.Bytes = data[:offset]
	}
	return nil
}

func decodeJSR(data []byte, opcode uint16, inst *Instruction) error {
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	operand, extraWords, err := decodeAddressingMode(data[2:], mode, reg)
	if err != nil {
		return err
	}
	inst.Mnemonic = "JSR"
	inst.Operands = operand
	inst.Size = uint32(2 + extraWords*2)
	if len(data) >= int(inst.Size) {
		inst.Bytes = data[:inst.Size]
	}
	return nil
}

func decodeJMP(data []byte, opcode uint16, inst *Instruction) error {
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	operand, extraWords, err := decodeAddressingMode(data[2:], mode, reg)
	if err != nil {
		return err
	}
	inst.Mnemonic = "JMP"
	inst.Operands = operand
	inst.Size = uint32(2 + extraWords*2)
	if len(data) >= int(inst.Size) {
		inst.Bytes = data[:inst.Size]
	}
	return nil
}
