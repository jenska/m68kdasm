package decoders

import "fmt"

func decodeCMP(data []byte, opcode uint16, inst *Instruction) error {
	sizeStr := getSizeString((opcode >> 6) & 0x3)
	dstReg := uint8((opcode >> 9) & 0x7)
	srcMode := uint8((opcode >> 3) & 0x7)
	srcReg := uint8(opcode & 0x7)

	srcStr, offset, err := decodeEA(data, 2, srcMode, srcReg)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, "CMP."+sizeStr, fmt.Sprintf("%s, D%d", srcStr, dstReg))
	return nil
}

func decodeCMPI(data []byte, opcode uint16, inst *Instruction) error {
	sizeStr, immSize, err := immediateSpec((opcode>>6)&0x3, false, "CMPI")
	if err != nil {
		return err
	}
	dstMode := uint8((opcode >> 3) & 0x7)
	dstReg := uint8(opcode & 0x7)
	immediate, offset, err := readImmediate(data, 2, immSize, "CMPI")
	if err != nil {
		return err
	}
	dstOperand, offset, err := decodeEA(data, offset, dstMode, dstReg)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, "CMPI."+sizeStr, fmt.Sprintf("#%s, %s", formatImmediate(immediate, immSize), dstOperand))
	return nil
}
