package decoders

import "fmt"

func decodeCMP(data []byte, opcode uint16, inst *Instruction) error {
	opmode := (opcode >> 6) & 0x7
	if (opcode & 0xF138) == 0xB108 {
		return decodeCMPM(data, opcode, inst)
	}
	if opmode == 3 || opmode == 7 {
		return decodeAddressRegisterOp("CMP", data, opcode, inst)
	}
	if opmode >= 4 && opmode <= 6 {
		return decodeEOR(data, opcode, inst)
	}

	sizeStr := getSizeString(opmode)
	sizeBytes, err := operandSize(opmode, "CMP")
	if err != nil {
		return err
	}
	dstReg := uint8((opcode >> 9) & 0x7)
	srcMode := uint8((opcode >> 3) & 0x7)
	srcReg := uint8(opcode & 0x7)

	srcStr, offset, srcMeta, err := decodeEAWithSize(data, inst.Address, 2, srcMode, srcReg, sizeBytes)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, "CMP."+sizeStr, fmt.Sprintf("%s, D%d", srcStr, dstReg), srcMeta, registerOperand(RegisterKindData, dstReg))
	return nil
}

func decodeCMPM(data []byte, opcode uint16, inst *Instruction) error {
	sizeBits := (opcode >> 6) & 0x3
	sizeStr := getSizeString(sizeBits)
	srcReg := uint8(opcode & 0x7)
	dstReg := uint8((opcode >> 9) & 0x7)
	srcText := fmt.Sprintf("(A%d)+", srcReg)
	dstText := fmt.Sprintf("(A%d)+", dstReg)
	setInstruction(data, inst, 2, "CMPM."+sizeStr, fmt.Sprintf("%s, %s", srcText, dstText),
		addrIndirectOperand(EAKindPostIncrement, srcReg, srcText),
		addrIndirectOperand(EAKindPostIncrement, dstReg, dstText))
	return nil
}

func decodeCMPI(data []byte, opcode uint16, inst *Instruction) error {
	return decodeImmediateBinaryOp("CMPI", data, opcode, inst, false)
}
