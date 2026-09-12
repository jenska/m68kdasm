package decoders

func decodeAND(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeDirectedBinaryOp("AND", data, opcode, inst, cpu)
}

func decodeOR(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeDirectedBinaryOp("OR", data, opcode, inst, cpu)
}

func decodeEOR(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeDirectedBinaryOp("EOR", data, opcode, inst, cpu)
}

func decodeANDI(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeImmediateBinaryOp("ANDI", data, opcode, inst, false, cpu)
}

func decodeORI(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeImmediateBinaryOp("ORI", data, opcode, inst, false, cpu)
}

func decodeEORI(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeImmediateBinaryOp("EORI", data, opcode, inst, false, cpu)
}
