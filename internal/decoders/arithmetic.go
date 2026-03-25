package decoders

import "fmt"

// decodeADD - Add (generisch für alle Adressierungsmodi)
// ADD Format: 1101 ddd ooo sss rrr
func decodeADD(data []byte, opcode uint16, inst *Instruction) error {
	return decodeDirectedBinaryOp("ADD", data, opcode, inst)
}

// decodeSUB - Subtract (generisch für alle Adressierungsmodi)
// SUB Format: 1001 ddd ooo sss rrr
func decodeSUB(data []byte, opcode uint16, inst *Instruction) error {
	return decodeDirectedBinaryOp("SUB", data, opcode, inst)
}

// decodeADDI - Add Immediate
// Format: 0000 0110 sz 000 mmm rrr (sz: 00=Byte, 01=Word, 10=Long)
func decodeADDI(data []byte, opcode uint16, inst *Instruction) error {
	return decodeImmediateBinaryOp("ADDI", data, opcode, inst, true)
}

// decodeSUBI - Subtract Immediate
// Format: 0000 0100 sz 000 mmm rrr
func decodeSUBI(data []byte, opcode uint16, inst *Instruction) error {
	return decodeImmediateBinaryOp("SUBI", data, opcode, inst, true)
}

func decodeImmediateBinaryOp(mnemonic string, data []byte, opcode uint16, inst *Instruction, longImmediate bool) error {
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

	dstOperand, offset, err := decodeEA(data, offset, dstMode, dstReg)
	if err != nil {
		return err
	}

	setInstruction(data, inst, offset, fmt.Sprintf("%s.%s", mnemonic, sizeStr), fmt.Sprintf("#%s, %s", formatImmediate(immediate, immSize), dstOperand))
	return nil
}
