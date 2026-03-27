package decoders

func decodeCLR(data []byte, opcode uint16, inst *Instruction) error {
	return decodeSingleOp(data, opcode, inst, "CLR")
}

func decodeNEG(data []byte, opcode uint16, inst *Instruction) error {
	return decodeSingleOp(data, opcode, inst, "NEG")
}

func decodeNEGX(data []byte, opcode uint16, inst *Instruction) error {
	return decodeSingleOp(data, opcode, inst, "NEGX")
}

func decodeNOT(data []byte, opcode uint16, inst *Instruction) error {
	return decodeSingleOp(data, opcode, inst, "NOT")
}

func decodeTST(data []byte, opcode uint16, inst *Instruction) error {
	return decodeSingleOp(data, opcode, inst, "TST")
}

func decodeSingleOp(data []byte, opcode uint16, inst *Instruction, mnemonic string) error {
	sizeStr := getSizeString((opcode >> 6) & 0x3)
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	operand, offset, meta, err := decodeEA(data, 2, mode, reg)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, mnemonic+"."+sizeStr, operand, meta)
	return nil
}
