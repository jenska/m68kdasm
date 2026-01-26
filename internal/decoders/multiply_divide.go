package decoders

import "fmt"

func decodeMULU(data []byte, opcode uint16, inst *Instruction) error {
	dstReg := uint8((opcode >> 9) & 0x7)
	srcMode := uint8((opcode >> 3) & 0x7)
	srcReg := uint8(opcode & 0x7)
	srcStr, srcExtraWords, err := decodeAddressingMode(data, srcMode, srcReg)
	if err != nil {
		return err
	}
	inst.Mnemonic = "MULU"
	inst.Operands = fmt.Sprintf("%s, D%d", srcStr, dstReg)
	inst.Size = uint32(2 + srcExtraWords*2)
	if len(data) >= int(inst.Size) {
		inst.Bytes = data[:inst.Size]
	}
	return nil
}

func decodeMULS(data []byte, opcode uint16, inst *Instruction) error {
	dstReg := uint8((opcode >> 9) & 0x7)
	srcMode := uint8((opcode >> 3) & 0x7)
	srcReg := uint8(opcode & 0x7)
	srcStr, srcExtraWords, err := decodeAddressingMode(data, srcMode, srcReg)
	if err != nil {
		return err
	}
	inst.Mnemonic = "MULS"
	inst.Operands = fmt.Sprintf("%s, D%d", srcStr, dstReg)
	inst.Size = uint32(2 + srcExtraWords*2)
	if len(data) >= int(inst.Size) {
		inst.Bytes = data[:inst.Size]
	}
	return nil
}

func decodeDIVU(data []byte, opcode uint16, inst *Instruction) error {
	dstReg := uint8((opcode >> 9) & 0x7)
	srcMode := uint8((opcode >> 3) & 0x7)
	srcReg := uint8(opcode & 0x7)
	srcStr, srcExtraWords, err := decodeAddressingMode(data, srcMode, srcReg)
	if err != nil {
		return err
	}
	inst.Mnemonic = "DIVU"
	inst.Operands = fmt.Sprintf("%s, D%d", srcStr, dstReg)
	inst.Size = uint32(2 + srcExtraWords*2)
	if len(data) >= int(inst.Size) {
		inst.Bytes = data[:inst.Size]
	}
	return nil
}

func decodeDIVS(data []byte, opcode uint16, inst *Instruction) error {
	dstReg := uint8((opcode >> 9) & 0x7)
	srcMode := uint8((opcode >> 3) & 0x7)
	srcReg := uint8(opcode & 0x7)
	srcStr, srcExtraWords, err := decodeAddressingMode(data, srcMode, srcReg)
	if err != nil {
		return err
	}
	inst.Mnemonic = "DIVS"
	inst.Operands = fmt.Sprintf("%s, D%d", srcStr, dstReg)
	inst.Size = uint32(2 + srcExtraWords*2)
	if len(data) >= int(inst.Size) {
		inst.Bytes = data[:inst.Size]
	}
	return nil
}
