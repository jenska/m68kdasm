package decoders

import (
	"encoding/binary"
	"fmt"
)

var branchCondNames = [...]string{
	"BRA", "BSR", "BHI", "BLS", "BHS", "BLO", "BNE", "BEQ",
	"BVC", "BVS", "BPL", "BMI", "BGE", "BLT", "BGT", "BLE",
}

func decodeBxx(data []byte, opcode uint16, inst *Instruction) error {
	offset := 2
	condition := (opcode >> 8) & 0x0F
	mnemonic := "?"
	if condition < uint16(len(branchCondNames)) {
		mnemonic = branchCondNames[condition]
	}
	displacement := int8(opcode & 0xFF)
	switch displacement {
	case 0:
		if len(data) < offset+2 {
			return fmt.Errorf("insufficient data for %s.W", mnemonic)
		}
		displacement16 := int16(binary.BigEndian.Uint16(data[offset : offset+2]))
		offset += 2
		target := uint32(int32(inst.Address) + int32(offset) + int32(displacement16))
		setInstruction(data, inst, offset, mnemonic+".W", formatBranchTarget(target))
	case -1:
		if len(data) < offset+4 {
			return fmt.Errorf("insufficient data for %s.L", mnemonic)
		}
		displacement32 := int32(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4
		target := uint32(int32(inst.Address) + int32(offset) + displacement32)
		setInstruction(data, inst, offset, mnemonic+".L", formatBranchTarget(target))
	default:
		target := uint32(int32(inst.Address) + int32(offset) + int32(displacement))
		setInstruction(data, inst, offset, mnemonic+".S", formatBranchTarget(target))
	}
	return nil
}

func formatBranchTarget(target uint32) string {
	if target <= 0xFFFF {
		return fmt.Sprintf("$%04X", target)
	}
	return fmt.Sprintf("$%08X", target)
}

func decodeJSR(data []byte, opcode uint16, inst *Instruction) error {
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	operand, offset, err := decodeEA(data, 2, mode, reg)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, "JSR", operand)
	return nil
}

func decodeJMP(data []byte, opcode uint16, inst *Instruction) error {
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	operand, offset, err := decodeEA(data, 2, mode, reg)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, "JMP", operand)
	return nil
}
