package decoders

import (
	"encoding/binary"
	"fmt"
)

func decodeABCD(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeBCD("ABCD", data, opcode, inst)
}

func decodeSBCD(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeBCD("SBCD", data, opcode, inst)
}

func decodeBCD(mn string, data []byte, opcode uint16, inst *Instruction) error {
	srcReg := uint8(opcode & 0x7)
	dstReg := uint8((opcode >> 9) & 0x7)
	addressingMode := (opcode >> 3) & 0x1
	if addressingMode == 0 {
		setInstruction(data, inst, 2, mn, fmt.Sprintf("D%d, D%d", srcReg, dstReg), registerOperand(RegisterKindData, srcReg), registerOperand(RegisterKindData, dstReg))
		return nil
	}
	srcText := fmt.Sprintf("-(A%d)", srcReg)
	dstText := fmt.Sprintf("-(A%d)", dstReg)
	setInstruction(data, inst, 2, mn, fmt.Sprintf("%s, %s", srcText, dstText),
		addrIndirectOperand(EAKindPreDecrement, srcReg, srcText),
		addrIndirectOperand(EAKindPreDecrement, dstReg, dstText))
	return nil
}

// decodeNBCD - Negate Decimal with Extend (opcode family 0100 1000 00 mmm rrr)
func decodeNBCD(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	operand, offset, meta, err := decodeEAWithSize(data, inst.Address, 2, mode, reg, 1, cpu)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, "NBCD", operand, meta)
	return nil
}

// decodePACK / decodeUNPK - convert between BCD and packed/unpacked forms
// (68020+). Format: 1000 yyy1 oooo Rxxx (register form when R=0, memory
// pre-decrement form when R=1), followed by a 16-bit adjustment word.
func decodePACK(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodePackUnpk("PACK", data, opcode, inst)
}

func decodeUNPK(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodePackUnpk("UNPK", data, opcode, inst)
}

func decodePackUnpk(mnemonic string, data []byte, opcode uint16, inst *Instruction) error {
	dstReg := uint8((opcode >> 9) & 0x7)
	srcReg := uint8(opcode & 0x7)
	memoryForm := opcode&0x0008 != 0

	if err := requireLength(data, 4, mnemonic+" adjustment"); err != nil {
		return err
	}
	adj := binary.BigEndian.Uint16(data[2:4])
	adjText := fmt.Sprintf("#%s", formatImmediate(uint32(adj), 2))

	var srcMeta, dstMeta Operand
	if memoryForm {
		srcText := fmt.Sprintf("-(A%d)", srcReg)
		dstText := fmt.Sprintf("-(A%d)", dstReg)
		srcMeta = addrIndirectOperand(EAKindPreDecrement, srcReg, srcText)
		dstMeta = addrIndirectOperand(EAKindPreDecrement, dstReg, dstText)
	} else {
		srcMeta = registerOperand(RegisterKindData, srcReg)
		dstMeta = registerOperand(RegisterKindData, dstReg)
	}

	setInstruction(data, inst, 4, mnemonic, fmt.Sprintf("%s, %s, %s", srcMeta.Text, dstMeta.Text, adjText), srcMeta, dstMeta, immediateOperand(adjText, uint32(adj), 2))
	return nil
}
