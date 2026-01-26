package decoders

func decodeCLR(data []byte, opcode uint16, inst *Instruction) error {
	size := (opcode >> 6) & 0x3
	sizeStr := []string{"B", "W", "L", "?"}[size]
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	operand, extraWords, err := decodeAddressingMode(data, mode, reg)
	if err != nil {
		return err
	}
	inst.Mnemonic = "CLR." + sizeStr
	inst.Operands = operand
	inst.Size = uint32(2 + extraWords*2)
	if len(data) >= int(inst.Size) {
		inst.Bytes = data[:inst.Size]
	}
	return nil
}

func decodeNEG(data []byte, opcode uint16, inst *Instruction) error {
	size := (opcode >> 6) & 0x3
	sizeStr := []string{"B", "W", "L", "?"}[size]
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	operand, extraWords, err := decodeAddressingMode(data, mode, reg)
	if err != nil {
		return err
	}
	inst.Mnemonic = "NEG." + sizeStr
	inst.Operands = operand
	inst.Size = uint32(2 + extraWords*2)
	if len(data) >= int(inst.Size) {
		inst.Bytes = data[:inst.Size]
	}
	return nil
}

func decodeNEGX(data []byte, opcode uint16, inst *Instruction) error {
	size := (opcode >> 6) & 0x3
	sizeStr := []string{"B", "W", "L", "?"}[size]
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	operand, extraWords, err := decodeAddressingMode(data, mode, reg)
	if err != nil {
		return err
	}
	inst.Mnemonic = "NEGX." + sizeStr
	inst.Operands = operand
	inst.Size = uint32(2 + extraWords*2)
	if len(data) >= int(inst.Size) {
		inst.Bytes = data[:inst.Size]
	}
	return nil
}

func decodeNOT(data []byte, opcode uint16, inst *Instruction) error {
	size := (opcode >> 6) & 0x3
	sizeStr := []string{"B", "W", "L", "?"}[size]
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	operand, extraWords, err := decodeAddressingMode(data, mode, reg)
	if err != nil {
		return err
	}
	inst.Mnemonic = "NOT." + sizeStr
	inst.Operands = operand
	inst.Size = uint32(2 + extraWords*2)
	if len(data) >= int(inst.Size) {
		inst.Bytes = data[:inst.Size]
	}
	return nil
}

func decodeTST(data []byte, opcode uint16, inst *Instruction) error {
	size := (opcode >> 6) & 0x3
	sizeStr := []string{"B", "W", "L", "?"}[size]
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	operand, extraWords, err := decodeAddressingMode(data, mode, reg)
	if err != nil {
		return err
	}
	inst.Mnemonic = "TST." + sizeStr
	inst.Operands = operand
	inst.Size = uint32(2 + extraWords*2)
	if len(data) >= int(inst.Size) {
		inst.Bytes = data[:inst.Size]
	}
	return nil
}
