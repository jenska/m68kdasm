package decoders

import (
	"encoding/binary"
	"fmt"
)

func decodeBTST(data []byte, opcode uint16, inst *Instruction) error {
	return decodeBitset("BTST", data, opcode, inst)
}

func decodeBCHG(data []byte, opcode uint16, inst *Instruction) error {
	return decodeBitset("BCHG", data, opcode, inst)
}

func decodeBCLR(data []byte, opcode uint16, inst *Instruction) error {
	return decodeBitset("BCLR", data, opcode, inst)
}

func decodeBSET(data []byte, opcode uint16, inst *Instruction) error {
	return decodeBitset("BSET", data, opcode, inst)
}

func decodeBitset(mn string, data []byte, opcode uint16, inst *Instruction) error {
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	offset := 2
	var bitNumStr string
	if (opcode>>8)&0x1 == 1 {
		bitReg := uint8((opcode >> 9) & 0x7)
		bitNumStr = fmt.Sprintf("D%d", bitReg)
	} else {
		if len(data) < offset+2 {
			return fmt.Errorf("insufficient data for BSET")
		}
		bitNum := binary.BigEndian.Uint16(data[offset : offset+2])
		bitNumStr = fmt.Sprintf("#%d", bitNum&0xFF)
		offset += 2
	}
	operand, offset, err := decodeEA(data, offset, mode, reg)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, mn, fmt.Sprintf("%s, %s", bitNumStr, operand))
	return nil
}
