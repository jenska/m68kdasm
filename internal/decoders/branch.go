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
		if err := requireLength(data, offset+2, mnemonic+".W displacement"); err != nil {
			return err
		}
		displacement16 := int16(binary.BigEndian.Uint16(data[offset : offset+2]))
		offset += 2
		target := uint32(int32(inst.Address) + int32(offset) + int32(displacement16))
		targetText := formatBranchTarget(target)
		setInstruction(data, inst, offset, mnemonic+".W", targetText, branchOperand(targetText, target))
	case -1:
		if err := requireLength(data, offset+4, mnemonic+".L displacement"); err != nil {
			return err
		}
		displacement32 := int32(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4
		target := uint32(int32(inst.Address) + int32(offset) + displacement32)
		targetText := formatBranchTarget(target)
		setInstruction(data, inst, offset, mnemonic+".L", targetText, branchOperand(targetText, target))
	default:
		target := uint32(int32(inst.Address) + int32(offset) + int32(displacement))
		targetText := formatBranchTarget(target)
		setInstruction(data, inst, offset, mnemonic+".S", targetText, branchOperand(targetText, target))
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
	operand, offset, meta, err := decodeEA(data, 2, mode, reg)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, "JSR", operand, meta)
	return nil
}

func decodeJMP(data []byte, opcode uint16, inst *Instruction) error {
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	operand, offset, meta, err := decodeEA(data, 2, mode, reg)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, "JMP", operand, meta)
	return nil
}
