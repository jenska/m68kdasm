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
	sizeStr := getSizeString((opcode >> 6) & 0x3)
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	operand, offset, meta, err := decodeEA(data, inst.Address, 2, mode, reg, cpu)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, mnemonic+"."+sizeStr, operand, meta)
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
