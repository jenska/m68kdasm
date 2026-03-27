package decoders

import "fmt"

func decodeCMP(data []byte, opcode uint16, inst *Instruction) error {
	opmode := (opcode >> 6) & 0x7
	if (opcode & 0xF138) == 0xB108 {
		return decodeCMPM(data, opcode, inst)
	}
	if opmode == 3 || opmode == 7 {
		return decodeCMPA(data, opcode, inst)
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

	srcStr, offset, srcMeta, err := decodeEAWithSize(data, 2, srcMode, srcReg, sizeBytes)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, "CMP."+sizeStr, fmt.Sprintf("%s, D%d", srcStr, dstReg), srcMeta, registerOperand(RegisterKindData, dstReg))
	return nil
}

func decodeCMPA(data []byte, opcode uint16, inst *Instruction) error {
	opmode := (opcode >> 6) & 0x7
	dstReg := uint8((opcode >> 9) & 0x7)
	srcMode := uint8((opcode >> 3) & 0x7)
	srcReg := uint8(opcode & 0x7)

	sizeStr := "W"
	sizeBytes := 2
	if opmode == 7 {
		sizeStr = "L"
		sizeBytes = 4
	}

	srcStr, offset, srcMeta, err := decodeEAWithSize(data, 2, srcMode, srcReg, sizeBytes)
	if err != nil {
		return err
	}

	setInstruction(data, inst, offset, "CMPA."+sizeStr, fmt.Sprintf("%s, A%d", srcStr, dstReg), srcMeta, registerOperand(RegisterKindAddress, dstReg))
	return nil
}

func decodeCMPM(data []byte, opcode uint16, inst *Instruction) error {
	sizeBits := (opcode >> 6) & 0x3
	sizeStr := getSizeString(sizeBits)
	srcReg := uint8(opcode & 0x7)
	dstReg := uint8((opcode >> 9) & 0x7)
	srcText := fmt.Sprintf("(A%d)+", srcReg)
	dstText := fmt.Sprintf("(A%d)+", dstReg)
	setInstruction(data, inst, 2, "CMPM."+sizeStr, fmt.Sprintf("%s, %s", srcText, dstText), effectiveAddressOperand(srcText, EffectiveAddress{
		Kind:     EAKindPostIncrement,
		Base:     &Register{Kind: RegisterKindAddress, Number: srcReg},
		Register: srcReg,
	}), effectiveAddressOperand(dstText, EffectiveAddress{
		Kind:     EAKindPostIncrement,
		Base:     &Register{Kind: RegisterKindAddress, Number: dstReg},
		Register: dstReg,
	}))
	return nil
}

func decodeCMPI(data []byte, opcode uint16, inst *Instruction) error {
	sizeStr, immSize, err := immediateSpec((opcode>>6)&0x3, false, "CMPI")
	if err != nil {
		return err
	}
	dstMode := uint8((opcode >> 3) & 0x7)
	dstReg := uint8(opcode & 0x7)
	immediate, offset, err := readImmediate(data, 2, immSize, "CMPI")
	if err != nil {
		return err
	}
	dstOperand, offset, dstMeta, err := decodeEA(data, offset, dstMode, dstReg)
	if err != nil {
		return err
	}
	immText := fmt.Sprintf("#%s", formatImmediate(immediate, immSize))
	setInstruction(data, inst, offset, "CMPI."+sizeStr, fmt.Sprintf("%s, %s", immText, dstOperand), immediateOperand(immText, immediate, immSize), dstMeta)
	return nil
}
