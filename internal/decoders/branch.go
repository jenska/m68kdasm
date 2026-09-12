package decoders

import (
	"encoding/binary"
	"fmt"
)

var branchCondNames = [...]string{
	"BRA", "BSR", "BHI", "BLS", "BHS", "BLO", "BNE", "BEQ",
	"BVC", "BVS", "BPL", "BMI", "BGE", "BLT", "BGT", "BLE",
}

// standardCondNames is the condition-code suffix table shared by Scc, DBcc,
// and TRAPcc (unlike Bcc, conditions 0/1 here are the real "always true" /
// "always false" tests, not BRA/BSR).
var standardCondNames = [...]string{
	"T", "F", "HI", "LS", "CC", "CS", "NE", "EQ",
	"VC", "VS", "PL", "MI", "GE", "LT", "GT", "LE",
}

func decodeBxx(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
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

// decodeScc - Set According to Condition (opcode family 0101 cccc 11 mmm rrr)
func decodeScc(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	cond := (opcode >> 8) & 0xF
	mnemonic := "S" + standardCondNames[cond]
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	operand, offset, meta, err := decodeEAWithSize(data, inst.Address, 2, mode, reg, 1, cpu)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, mnemonic, operand, meta)
	return nil
}

// decodeDBcc - Test Condition, Decrement, and Branch
// (opcode family 0101 cccc 11001 rrr) — must be registered before Scc,
// since it occupies what would otherwise be Scc's address-register-direct
// EA slot.
func decodeDBcc(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	cond := (opcode >> 8) & 0xF
	reg := uint8(opcode & 0x7)
	mnemonic := "DB" + standardCondNames[cond]

	if err := requireLength(data, 4, mnemonic+" displacement"); err != nil {
		return err
	}
	disp := int32(int16(binary.BigEndian.Uint16(data[2:4])))
	// Matches decodeBxx's existing displacement-base convention for the
	// 16-bit form (offset counted after consuming the displacement word).
	target := uint32(int32(inst.Address) + 4 + disp)
	targetText := formatBranchTarget(target)
	regText := fmt.Sprintf("D%d", reg)
	setInstruction(data, inst, 4, mnemonic, fmt.Sprintf("%s, %s", regText, targetText), registerOperand(RegisterKindData, reg), branchOperand(targetText, target))
	return nil
}

// decodeTRAPcc - Trap on Condition (68020+), opcode family 0101 cccc 11111 sss.
// Must be registered before Scc for the same reason as DBcc.
func decodeTRAPcc(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	cond := (opcode >> 8) & 0xF
	mnemonic := "TRAP" + standardCondNames[cond]
	switch opcode & 0x7 {
	case 0x4: // no operand
		setInstruction(data, inst, 2, mnemonic, "")
	case 0x2: // word operand
		if err := requireLength(data, 4, mnemonic+" word operand"); err != nil {
			return err
		}
		imm := uint32(binary.BigEndian.Uint16(data[2:4]))
		immText := fmt.Sprintf("#%s", formatImmediate(imm, 2))
		setInstruction(data, inst, 4, mnemonic, immText, immediateOperand(immText, imm, 2))
	case 0x3: // long operand
		if err := requireLength(data, 6, mnemonic+" long operand"); err != nil {
			return err
		}
		imm := binary.BigEndian.Uint32(data[2:6])
		immText := fmt.Sprintf("#%s", formatImmediate(imm, 4))
		setInstruction(data, inst, 6, mnemonic, immText, immediateOperand(immText, imm, 4))
	default:
		return fmt.Errorf("unknown %s operand selector: %d", mnemonic, opcode&0x7)
	}
	return nil
}

func decodeJSR(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeUnaryEA("JSR", data, opcode, inst, cpu)
}

func decodeJMP(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeUnaryEA("JMP", data, opcode, inst, cpu)
}
