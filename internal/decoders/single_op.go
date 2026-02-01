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
	size := (opcode >> 6) & 0x3
	sizeStr := []string{"B", "W", "L", "?"}[size]
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	operand, extraWords, err := decodeAddressingMode(data[2:], mode, reg)
	if err != nil {
		return err
	}
	inst.Mnemonic = mnemonic + "." + sizeStr
	inst.Operands = operand
	inst.Size = uint32(2 + extraWords*2)
	if len(data) >= int(inst.Size) {
		inst.Bytes = data[:inst.Size]
	}
	return nil
}
