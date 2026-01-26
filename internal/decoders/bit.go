package decoders

import (
	"encoding/binary"
	"fmt"
)

func decodeBTST(data []byte, opcode uint16, inst *Instruction) error {
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	offset := 2
	var bitNumStr string
	if (opcode>>8)&0x1 == 1 {
		bitReg := uint8((opcode >> 9) & 0x7)
		bitNumStr = fmt.Sprintf("D%d", bitReg)
	} else {
		if len(data) < offset+2 {
			return fmt.Errorf("insufficient data for BTST")
		}
		bitNum := binary.BigEndian.Uint16(data[offset : offset+2])
		bitNumStr = fmt.Sprintf("#%d", bitNum&0xFF)
		offset += 2
	}
	operand, extraWords, err := decodeAddressingMode(data[offset:], mode, reg)
	if err != nil {
		return err
	}
	offset += extraWords * 2
	inst.Mnemonic = "BTST"
	inst.Operands = fmt.Sprintf("%s, %s", bitNumStr, operand)
	inst.Size = uint32(offset)
	if len(data) >= offset {
		inst.Bytes = data[:offset]
	}
	return nil
}

func decodeBCHG(data []byte, opcode uint16, inst *Instruction) error {
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	offset := 2
	var bitNumStr string
	if (opcode>>8)&0x1 == 1 {
		bitReg := uint8((opcode >> 9) & 0x7)
		bitNumStr = fmt.Sprintf("D%d", bitReg)
	} else {
		if len(data) < offset+2 {
			return fmt.Errorf("insufficient data for BCHG")
		}
		bitNum := binary.BigEndian.Uint16(data[offset : offset+2])
		bitNumStr = fmt.Sprintf("#%d", bitNum&0xFF)
		offset += 2
	}
	operand, extraWords, err := decodeAddressingMode(data[offset:], mode, reg)
	if err != nil {
		return err
	}
	offset += extraWords * 2
	inst.Mnemonic = "BCHG"
	inst.Operands = fmt.Sprintf("%s, %s", bitNumStr, operand)
	inst.Size = uint32(offset)
	if len(data) >= offset {
		inst.Bytes = data[:offset]
	}
	return nil
}

func decodeBCLR(data []byte, opcode uint16, inst *Instruction) error {
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	offset := 2
	var bitNumStr string
	if (opcode>>8)&0x1 == 1 {
		bitReg := uint8((opcode >> 9) & 0x7)
		bitNumStr = fmt.Sprintf("D%d", bitReg)
	} else {
		if len(data) < offset+2 {
			return fmt.Errorf("insufficient data for BCLR")
		}
		bitNum := binary.BigEndian.Uint16(data[offset : offset+2])
		bitNumStr = fmt.Sprintf("#%d", bitNum&0xFF)
		offset += 2
	}
	operand, extraWords, err := decodeAddressingMode(data[offset:], mode, reg)
	if err != nil {
		return err
	}
	offset += extraWords * 2
	inst.Mnemonic = "BCLR"
	inst.Operands = fmt.Sprintf("%s, %s", bitNumStr, operand)
	inst.Size = uint32(offset)
	if len(data) >= offset {
		inst.Bytes = data[:offset]
	}
	return nil
}

func decodeBSET(data []byte, opcode uint16, inst *Instruction) error {
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
	operand, extraWords, err := decodeAddressingMode(data[offset:], mode, reg)
	if err != nil {
		return err
	}
	offset += extraWords * 2
	inst.Mnemonic = "BSET"
	inst.Operands = fmt.Sprintf("%s, %s", bitNumStr, operand)
	inst.Size = uint32(offset)
	if len(data) >= offset {
		inst.Bytes = data[:offset]
	}
	return nil
}
