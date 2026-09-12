package decoders

import "fmt"

// decodeADD - Add (generisch für alle Adressierungsmodi)
// ADD Format: 1101 ddd ooo sss rrr
func decodeADD(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	if isAddressRegisterArithmetic(opcode) {
		return decodeAddressRegisterOp("ADD", data, opcode, inst, cpu)
	}
	return decodeDirectedBinaryOp("ADD", data, opcode, inst, cpu)
}

// decodeSUB - Subtract (generisch für alle Adressierungsmodi)
// SUB Format: 1001 ddd ooo sss rrr
func decodeSUB(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	if isAddressRegisterArithmetic(opcode) {
		return decodeAddressRegisterOp("SUB", data, opcode, inst, cpu)
	}
	return decodeDirectedBinaryOp("SUB", data, opcode, inst, cpu)
}

// decodeADDI - Add Immediate
// Format: 0000 0110 sz 000 mmm rrr (sz: 00=Byte, 01=Word, 10=Long)
func decodeADDI(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeImmediateBinaryOp("ADDI", data, opcode, inst, true, cpu)
}

// decodeSUBI - Subtract Immediate
// Format: 0000 0100 sz 000 mmm rrr
func decodeSUBI(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeImmediateBinaryOp("SUBI", data, opcode, inst, true, cpu)
}

func decodeImmediateBinaryOp(mnemonic string, data []byte, opcode uint16, inst *Instruction, longImmediate bool, cpu CPU) error {
	sizeStr, immSize, err := immediateSpec((opcode>>6)&0x3, longImmediate, mnemonic)
	if err != nil {
		return err
	}

	dstMode := uint8((opcode >> 3) & 0x7)
	dstReg := uint8(opcode & 0x7)

	immediate, offset, err := readImmediate(data, 2, immSize, mnemonic)
	if err != nil {
		return err
	}

	dstOperand, offset, dstMeta, err := decodeEA(data, inst.Address, offset, dstMode, dstReg, cpu)
	if err != nil {
		return err
	}

	immText := fmt.Sprintf("#%s", formatImmediate(immediate, immSize))
	setInstruction(data, inst, offset, fmt.Sprintf("%s.%s", mnemonic, sizeStr), fmt.Sprintf("%s, %s", immText, dstOperand), immediateOperand(immText, immediate, immSize), dstMeta)
	return nil
}

func isAddressRegisterArithmetic(opcode uint16) bool {
	opmode := (opcode >> 6) & 0x7
	return opmode == 3 || opmode == 7
}

func decodeADDQ(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeQuick("ADDQ", data, opcode, inst, cpu)
}

func decodeSUBQ(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeQuick("SUBQ", data, opcode, inst, cpu)
}

// decodeQuick - ADDQ/SUBQ. Format: 0101 ddd s ss mmm rrr, where ddd is a
// 3-bit immediate (0 means 8) and s selects ADDQ(0)/SUBQ(1).
func decodeQuick(mnemonic string, data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	count := (opcode >> 9) & 0x7
	if count == 0 {
		count = 8
	}
	sizeStr := getSizeString((opcode >> 6) & 0x3)
	sizeBytes, err := operandSize((opcode>>6)&0x3, mnemonic)
	if err != nil {
		return err
	}
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)

	operand, offset, meta, err := decodeEAWithSize(data, inst.Address, 2, mode, reg, sizeBytes, cpu)
	if err != nil {
		return err
	}
	immText := fmt.Sprintf("#%d", count)
	setInstruction(data, inst, offset, mnemonic+"."+sizeStr, fmt.Sprintf("%s, %s", immText, operand), immediateOperand(immText, uint32(count), 1), meta)
	return nil
}

// decodeADDX / decodeSUBX - Add/Subtract with Extend. Format: 1101/1001 ddd 1
// ss 00 m sss, where m=0 selects the Dy,Dx register form and m=1 selects the
// -(Ay),-(Ax) memory form. Must be registered before the broad ADD/SUB
// catch-alls, since ADDX/SUBX occupy what would otherwise be an invalid
// (mode 0/1, direction=1) ADD/SUB destination slot.
func decodeADDX(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeAddSubX("ADDX", data, opcode, inst)
}

func decodeSUBX(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeAddSubX("SUBX", data, opcode, inst)
}

func decodeAddSubX(mnemonic string, data []byte, opcode uint16, inst *Instruction) error {
	sizeStr := getSizeString((opcode >> 6) & 0x3)
	dstReg := uint8((opcode >> 9) & 0x7)
	srcReg := uint8(opcode & 0x7)

	if opcode&0x0008 != 0 { // memory form: -(Ay),-(Ax)
		srcText := fmt.Sprintf("-(A%d)", srcReg)
		dstText := fmt.Sprintf("-(A%d)", dstReg)
		setInstruction(data, inst, 2, mnemonic+"."+sizeStr, fmt.Sprintf("%s, %s", srcText, dstText),
			addrIndirectOperand(EAKindPreDecrement, srcReg, srcText),
			addrIndirectOperand(EAKindPreDecrement, dstReg, dstText))
		return nil
	}

	srcMeta := registerOperand(RegisterKindData, srcReg)
	dstMeta := registerOperand(RegisterKindData, dstReg)
	setInstruction(data, inst, 2, mnemonic+"."+sizeStr, fmt.Sprintf("%s, %s", srcMeta.Text, dstMeta.Text), srcMeta, dstMeta)
	return nil
}
