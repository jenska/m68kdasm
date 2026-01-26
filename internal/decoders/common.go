package decoders

// decodeNOP - No Operation (exact opcode: 0x4E71)
func decodeNOP(data []byte, opcode uint16, inst *Instruction) error {
	inst.Mnemonic = "NOP"
	inst.Size = 2
	if len(data) >= 2 {
		inst.Bytes = data[:2]
	}
	return nil
}

// decodeRTS - Return from Subroutine (exact opcode: 0x4E75)
func decodeRTS(data []byte, opcode uint16, inst *Instruction) error {
	inst.Mnemonic = "RTS"
	inst.Size = 2
	if len(data) >= 2 {
		inst.Bytes = data[:2]
	}
	return nil
}
