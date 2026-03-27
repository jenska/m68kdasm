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
		setInstruction(data, inst, 2, mn, fmt.Sprintf("D%d, D%d", srcReg, dstReg), registerOperand(RegisterKindData, srcReg), registerOperand(RegisterKindData, dstReg))
		return nil
	}
	srcText := fmt.Sprintf("-(A%d)", srcReg)
	dstText := fmt.Sprintf("-(A%d)", dstReg)
	setInstruction(data, inst, 2, mn, fmt.Sprintf("%s, %s", srcText, dstText), effectiveAddressOperand(srcText, EffectiveAddress{
		Kind:     EAKindPreDecrement,
		Base:     &Register{Kind: RegisterKindAddress, Number: srcReg},
		Register: srcReg,
	}), effectiveAddressOperand(dstText, EffectiveAddress{
		Kind:     EAKindPreDecrement,
		Base:     &Register{Kind: RegisterKindAddress, Number: dstReg},
		Register: dstReg,
	}))
	return nil
}
