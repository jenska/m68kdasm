package decoders

import "fmt"

func decodeABCD(data []byte, opcode uint16, inst *Instruction) error {
	srcReg := uint8((opcode >> 9) & 0x7)
	dstReg := uint8(opcode & 0x7)
	addressingMode := (opcode >> 3) & 0x1
	inst.Mnemonic = "ABCD"
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

func decodeSBCD(data []byte, opcode uint16, inst *Instruction) error {
	srcReg := uint8((opcode >> 9) & 0x7)
	dstReg := uint8(opcode & 0x7)
	addressingMode := (opcode >> 3) & 0x1
	inst.Mnemonic = "SBCD"
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
