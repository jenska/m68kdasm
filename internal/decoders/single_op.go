package decoders

func decodeCLR(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeSingleOp(data, opcode, inst, "CLR", cpu)
}

func decodeNEG(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeSingleOp(data, opcode, inst, "NEG", cpu)
}

func decodeNEGX(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeSingleOp(data, opcode, inst, "NEGX", cpu)
}

func decodeNOT(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeSingleOp(data, opcode, inst, "NOT", cpu)
}

func decodeTST(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeSingleOp(data, opcode, inst, "TST", cpu)
}

func decodeSingleOp(data []byte, opcode uint16, inst *Instruction, mnemonic string, cpu CPU) error {
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	if (opcode>>6)&0x3 == 0x3 {
		return decodeMoveStatus(data, opcode, inst, mnemonic, mode, reg, cpu)
	}
	sizeStr := getSizeString((opcode >> 6) & 0x3)
	size, err := operandSize((opcode>>6)&0x3, mnemonic)
	if err != nil {
		return err
	}
	operand, offset, meta, err := decodeEAWithSize(data, inst.Address, 2, mode, reg, size, cpu)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, mnemonic+"."+sizeStr, operand, meta)
	return nil
}

// decodeMoveStatus decodes the size-11 slots of NEGX/CLR/NEG/NOT: MOVE
// SR,<ea>, MOVE CCR,<ea> (68010+), MOVE <ea>,CCR and MOVE <ea>,SR — all
// word-sized.
func decodeMoveStatus(data []byte, opcode uint16, inst *Instruction, slot string, mode, reg uint8, cpu CPU) error {
	operand, offset, meta, err := decodeEAWithSize(data, inst.Address, 2, mode, reg, 2, cpu)
	if err != nil {
		return err
	}
	srOperand := registerOperand(RegisterKindSR, 0)
	ccrOperand := registerOperand(RegisterKindCCR, 0)
	switch slot {
	case "NEGX":
		setInstruction(data, inst, offset, "MOVE", "SR, "+operand, srOperand, meta)
	case "CLR":
		setInstruction(data, inst, offset, "MOVE", "CCR, "+operand, ccrOperand, meta)
	case "NEG":
		setInstruction(data, inst, offset, "MOVE", operand+", CCR", meta, ccrOperand)
	case "NOT":
		setInstruction(data, inst, offset, "MOVE", operand+", SR", meta, srOperand)
	default:
		return ErrInvalidEncoding
	}
	return nil
}

// decodeTAS - Test and Set an Operand (opcode family 0100 1010 11 mmm rrr).
// Always byte-sized, unlike CLR/NEG/NEGX/NOT/TST above. Must be registered
// before TST, since it otherwise falls inside TST's broad opcode range.
func decodeTAS(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	operand, offset, meta, err := decodeEAWithSize(data, inst.Address, 2, mode, reg, 1, cpu)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, "TAS", operand, meta)
	return nil
}
