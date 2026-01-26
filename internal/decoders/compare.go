package decoders

import (
	"encoding/binary"
	"fmt"
)

func decodeCMP(data []byte, opcode uint16, inst *Instruction) error {
	size := (opcode >> 6) & 0x3
	sizeStr := []string{"B", "W", "L", "?"}[size]
	dstReg := uint8((opcode >> 9) & 0x7)
	srcMode := uint8((opcode >> 3) & 0x7)
	srcReg := uint8(opcode & 0x7)
	offset := 2
	srcStr, srcExtraWords, err := decodeAddressingMode(data, srcMode, srcReg)
	if err != nil {
		return err
	}
	offset += srcExtraWords * 2
	inst.Mnemonic = "CMP." + sizeStr
	inst.Operands = fmt.Sprintf("%s, D%d", srcStr, dstReg)
	inst.Size = uint32(offset)
	if len(data) >= offset {
		inst.Bytes = data[:offset]
	}
	return nil
}

func decodeCMPI(data []byte, opcode uint16, inst *Instruction) error {
	size := (opcode >> 6) & 0x3
	sizeStr := []string{"B", "W", "L", "?"}[size]
	dstMode := uint8((opcode >> 3) & 0x7)
	dstReg := uint8(opcode & 0x7)
	offset := 2
	if len(data) < offset+2 {
		return fmt.Errorf("insufficient data for CMPI")
	}
	immediate := uint32(binary.BigEndian.Uint16(data[offset : offset+2]))
	offset += 2
	dstOperand, extraWords, err := decodeAddressingMode(data[offset:], dstMode, dstReg)
	if err != nil {
		return err
	}
	offset += extraWords * 2
	inst.Mnemonic = "CMPI." + sizeStr
	immStr := formatImmediate(immediate, 2)
	inst.Operands = fmt.Sprintf("#%s, %s", immStr, dstOperand)
	inst.Size = uint32(offset)
	if len(data) >= offset {
		inst.Bytes = data[:offset]
	}
	return nil
}
