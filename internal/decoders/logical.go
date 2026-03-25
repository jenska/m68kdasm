package decoders

import "fmt"

func decodeAND(data []byte, opcode uint16, inst *Instruction) error {
	return decodeLogical("AND", data, opcode, inst)
}

func decodeOR(data []byte, opcode uint16, inst *Instruction) error {
	return decodeLogical("OR", data, opcode, inst)
}

func decodeEOR(data []byte, opcode uint16, inst *Instruction) error {
	return decodeLogical("EOR", data, opcode, inst)
}

func decodeANDI(data []byte, opcode uint16, inst *Instruction) error {
	return decodeLogicalI("ANDI", data, opcode, inst)
}

func decodeORI(data []byte, opcode uint16, inst *Instruction) error {
	return decodeLogicalI("ORI", data, opcode, inst)
}

func decodeEORI(data []byte, opcode uint16, inst *Instruction) error {
	return decodeLogicalI("EORI", data, opcode, inst)
}

func decodeLogicalI(mn string, data []byte, opcode uint16, inst *Instruction) error {
	sizeStr, immSize, err := immediateSpec((opcode>>6)&0x3, false, mn)
	if err != nil {
		return err
	}
	dstMode := uint8((opcode >> 3) & 0x7)
	dstReg := uint8(opcode & 0x7)

	immediate, offset, err := readImmediate(data, 2, immSize, mn)
	if err != nil {
		return err
	}
	dstOperand, offset, err := decodeEA(data, offset, dstMode, dstReg)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, mn+"."+sizeStr, fmt.Sprintf("#%s, %s", formatImmediate(immediate, immSize), dstOperand))
	return nil
}

func decodeLogical(mn string, data []byte, opcode uint16, inst *Instruction) error {
	return decodeDirectedBinaryOp(mn, data, opcode, inst)
}
