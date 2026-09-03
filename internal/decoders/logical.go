package decoders

func decodeAND(data []byte, opcode uint16, inst *Instruction) error {
	return decodeDirectedBinaryOp("AND", data, opcode, inst)
}

func decodeOR(data []byte, opcode uint16, inst *Instruction) error {
	return decodeDirectedBinaryOp("OR", data, opcode, inst)
}

func decodeEOR(data []byte, opcode uint16, inst *Instruction) error {
	return decodeDirectedBinaryOp("EOR", data, opcode, inst)
}

func decodeANDI(data []byte, opcode uint16, inst *Instruction) error {
	return decodeImmediateBinaryOp("ANDI", data, opcode, inst, false)
}

func decodeORI(data []byte, opcode uint16, inst *Instruction) error {
	return decodeImmediateBinaryOp("ORI", data, opcode, inst, false)
}

func decodeEORI(data []byte, opcode uint16, inst *Instruction) error {
	return decodeImmediateBinaryOp("EORI", data, opcode, inst, false)
}
