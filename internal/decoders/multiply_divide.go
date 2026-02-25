package decoders

import "fmt"

func decodeMULU(data []byte, opcode uint16, inst *Instruction) error {
	return decodeMulDiv("MULU", data, opcode, inst)
}

func decodeMULS(data []byte, opcode uint16, inst *Instruction) error {
	return decodeMulDiv("MULS", data, opcode, inst)
}

func decodeDIVU(data []byte, opcode uint16, inst *Instruction) error {
	return decodeMulDiv("DIVU", data, opcode, inst)
}

func decodeDIVS(data []byte, opcode uint16, inst *Instruction) error {
	return decodeMulDiv("DIVS", data, opcode, inst)
}

func decodeMulDiv(mn string, data []byte, opcode uint16, inst *Instruction) error {
	dstReg := uint8((opcode >> 9) & 0x7)
	srcMode := uint8((opcode >> 3) & 0x7)
	srcReg := uint8(opcode & 0x7)
	srcStr, srcExtraWords, err := decodeAddressingMode(data[2:], srcMode, srcReg)
	if err != nil {
		return err
	}
	inst.Mnemonic = mn
	inst.Operands = fmt.Sprintf("%s, D%d", srcStr, dstReg)
	inst.Size = uint32(2 + srcExtraWords*2)
	if len(data) >= int(inst.Size) {
		inst.Bytes = data[:inst.Size]
	}
	return nil
}
