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

// decodeMulDivLong decodes the CPU32/68020+ long multiply and divide:
// 0100 1100 0d eeeeee (d = divide), extension word 0lll sz00 0000 0hhh
// with s = signed, z = 64-bit product or dividend, lll = Dl/Dq and
// hhh = Dh/Dr. Following the Motorola syntax:
//
//	MULx.L <ea>,Dl       z=0 (hhh unused, 0)
//	MULx.L <ea>,Dh:Dl    z=1
//	DIVx.L <ea>,Dq       z=0, Dr = Dq
//	DIVxL.L <ea>,Dr:Dq   z=0, Dr != Dq (32-bit dividend, remainder kept)
//	DIVx.L <ea>,Dr:Dq    z=1 (64-bit dividend)
func decodeMulDivLong(mnemonicUnsigned, mnemonicSigned string, data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	if err := requireLength(data, 4, "long MUL/DIV extension word"); err != nil {
		return err
	}
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	ext := binary.BigEndian.Uint16(data[2:4])
	if ext&0x83F8 != 0 {
		return ErrInvalidEncoding // bit 15 and bits 9-3 are reserved
	}
	divide := mnemonicUnsigned == "DIVU"
	wide := ext&0x0400 != 0
	low := uint8((ext >> 12) & 0x7)
	high := uint8(ext & 0x7)

	mnemonic := mnemonicUnsigned
	if ext&0x0800 != 0 {
		mnemonic = mnemonicSigned
	}

	srcOperand, offset, srcMeta, err := decodeEAWithSize(data, inst.Address, 4, mode, reg, 4, cpu)
	if err != nil {
		return err
	}

	lowMeta := registerOperand(RegisterKindData, low)
	highMeta := registerOperand(RegisterKindData, high)
	destText := lowMeta.Text
	operands := []Operand{srcMeta, lowMeta}
	switch {
	case wide:
		destText = fmt.Sprintf("%s:%s", highMeta.Text, lowMeta.Text)
		operands = []Operand{srcMeta, highMeta, lowMeta}
	case divide && high != low:
		mnemonic += "L" // DIVUL/DIVSL: 32-bit dividend, remainder in Dr
		destText = fmt.Sprintf("%s:%s", highMeta.Text, lowMeta.Text)
		operands = []Operand{srcMeta, highMeta, lowMeta}
	case !divide && high != 0:
		return ErrInvalidEncoding // the 32-bit product leaves Dh zero
	}

	setInstruction(data, inst, offset, mnemonic+".L", fmt.Sprintf("%s, %s", srcOperand, destText), operands...)
	return nil
}
