package decoders

import (
	"encoding/binary"
	"fmt"
)

func decodeMULU(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeMulDiv("MULU", data, opcode, inst, cpu)
}

func decodeMULS(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeMulDiv("MULS", data, opcode, inst, cpu)
}

func decodeDIVU(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeMulDiv("DIVU", data, opcode, inst, cpu)
}

func decodeDIVS(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeMulDiv("DIVS", data, opcode, inst, cpu)
}

func decodeMulDiv(mn string, data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	dstReg := uint8((opcode >> 9) & 0x7)
	srcMode := uint8((opcode >> 3) & 0x7)
	srcReg := uint8(opcode & 0x7)
	srcStr, offset, srcMeta, err := decodeEA(data, inst.Address, 2, srcMode, srcReg, cpu)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, mn, fmt.Sprintf("%s, D%d", srcStr, dstReg), srcMeta, registerOperand(RegisterKindData, dstReg))
	return nil
}

// decodeMULLong / decodeDIVLong - 32x32 MULU/MULS/DIVU/DIVS.L (68020+).
// Opcode: 0100 1100 0d mmm rrr (d=0 mul, d=1 div), followed by an extension
// word: bit15=sign(0=unsigned,1=signed), bits14-12=Dh, bit10=size
// (0=32-bit result in Dl, 1=64-bit result in Dh:Dl), bits2-0=Dl.
func decodeMULLong(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeMulDivLong("MULU", "MULS", data, opcode, inst, cpu)
}

func decodeDIVLong(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeMulDivLong("DIVU", "DIVS", data, opcode, inst, cpu)
}

func decodeMulDivLong(mnemonicUnsigned, mnemonicSigned string, data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	if err := requireLength(data, 4, "long MUL/DIV extension word"); err != nil {
		return err
	}
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	ext := binary.BigEndian.Uint16(data[2:4])

	mnemonic := mnemonicUnsigned
	if ext&0x8000 != 0 {
		mnemonic = mnemonicSigned
	}
	dlMeta := registerOperand(RegisterKindData, uint8(ext&0x7))

	srcOperand, offset, srcMeta, err := decodeEAWithSize(data, inst.Address, 4, mode, reg, 4, cpu)
	if err != nil {
		return err
	}

	destText := dlMeta.Text
	operands := []Operand{srcMeta, dlMeta}
	if ext&0x0400 != 0 { // 64-bit result in Dh:Dl
		dhMeta := registerOperand(RegisterKindData, uint8((ext>>12)&0x7))
		destText = fmt.Sprintf("%s:%s", dhMeta.Text, dlMeta.Text)
		operands = []Operand{srcMeta, dhMeta, dlMeta}
	}

	setInstruction(data, inst, offset, mnemonic+".L", fmt.Sprintf("%s, %s", srcOperand, destText), operands...)
	return nil
}
