package decoders

import "fmt"

func decodeABCD(data []byte, opcode uint16, inst *Instruction) error {
	return decodeBCD("ABCD", data, opcode, inst)
}

func decodeSBCD(data []byte, opcode uint16, inst *Instruction) error {
	return decodeBCD("SBCD", data, opcode, inst)
}

func decodeBCD(mn string, data []byte, opcode uint16, inst *Instruction) error {
	srcReg := uint8(opcode & 0x7)
	dstReg := uint8((opcode >> 9) & 0x7)
	addressingMode := (opcode >> 3) & 0x1
	if addressingMode == 0 {
		setInstruction(data, inst, 2, mn, fmt.Sprintf("D%d, D%d", srcReg, dstReg))
		return nil
	}
	setInstruction(data, inst, 2, mn, fmt.Sprintf("-(A%d), -(A%d)", srcReg, dstReg))
	return nil
}
