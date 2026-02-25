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
	inst.Mnemonic = mn
	if addressingMode == 0 {
		inst.Operands = fmt.Sprintf("D%d, D%d", srcReg, dstReg)
	} else {
		inst.Operands = fmt.Sprintf("-(A%d), -(A%d)", srcReg, dstReg)
	}
	inst.Size = 2
	if len(data) >= 2 {
		inst.Bytes = data[:2]
	}
	return nil
}
