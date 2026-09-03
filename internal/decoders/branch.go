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
	condition := (opcode >> 8) & 0x0F
	mnemonic := "?"
	if condition < uint16(len(branchCondNames)) {
		mnemonic = branchCondNames[condition]
	}

	offset := 2
	var suffix string
	var disp int32
	switch d8 := int8(opcode & 0xFF); d8 {
	case 0:
		if err := requireLength(data, offset+2, mnemonic+".W displacement"); err != nil {
			return err
		}
		disp = int32(int16(binary.BigEndian.Uint16(data[offset : offset+2])))
		offset += 2
		suffix = "W"
	case -1:
		if err := requireLength(data, offset+4, mnemonic+".L displacement"); err != nil {
			return err
		}
		disp = int32(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4
		suffix = "L"
	default:
		disp = int32(d8)
		suffix = "S"
	}

	target := uint32(int32(inst.Address) + int32(offset) + disp)
	targetText := formatBranchTarget(target)
	setInstruction(data, inst, offset, mnemonic+"."+suffix, targetText, branchOperand(targetText, target))
	return nil
}

func formatBranchTarget(target uint32) string {
	if target <= 0xFFFF {
		return fmt.Sprintf("$%04X", target)
	}
	return fmt.Sprintf("$%08X", target)
}

func decodeJSR(data []byte, opcode uint16, inst *Instruction) error {
	return decodeUnaryEA("JSR", data, opcode, inst)
}

func decodeJMP(data []byte, opcode uint16, inst *Instruction) error {
	return decodeUnaryEA("JMP", data, opcode, inst)
}
