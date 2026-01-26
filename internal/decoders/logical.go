package decoders

import (
	"encoding/binary"
	"fmt"
)

func decodeAND(data []byte, opcode uint16, inst *Instruction) error {
	direction := (opcode >> 8) & 0x1
	size := (opcode >> 6) & 0x3
	sizeStr := []string{"B", "W", "L", "?"}[size]
	dstReg := uint8((opcode >> 9) & 0x7)
	srcMode := uint8((opcode >> 3) & 0x7)
	srcReg := uint8(opcode & 0x7)
	offset := 2
	srcStr, srcExtra, err := decodeAddressingMode(data, srcMode, srcReg)
	if err != nil {
		return err
	}
	offset += srcExtra * 2
	inst.Mnemonic = "AND." + sizeStr
	if direction == 0 {
		inst.Operands = fmt.Sprintf("%s, D%d", srcStr, dstReg)
	} else {
		dstMode := uint8((opcode >> 6) & 0x7)
		dstStr, dstExtra, _ := decodeAddressingMode(data[offset:], dstMode, dstReg)
		offset += dstExtra * 2
		inst.Operands = fmt.Sprintf("D%d, %s", dstReg, dstStr)
	}
	inst.Size = uint32(offset)
	if len(data) >= offset {
		inst.Bytes = data[:offset]
	}
	return nil
}

func decodeOR(data []byte, opcode uint16, inst *Instruction) error {
	direction := (opcode >> 8) & 0x1
	size := (opcode >> 6) & 0x3
	sizeStr := []string{"B", "W", "L", "?"}[size]
	dstReg := uint8((opcode >> 9) & 0x7)
	srcMode := uint8((opcode >> 3) & 0x7)
	srcReg := uint8(opcode & 0x7)
	offset := 2
	srcStr, srcExtra, err := decodeAddressingMode(data, srcMode, srcReg)
	if err != nil {
		return err
	}
	offset += srcExtra * 2
	inst.Mnemonic = "OR." + sizeStr
	if direction == 0 {
		inst.Operands = fmt.Sprintf("%s, D%d", srcStr, dstReg)
	} else {
		dstMode := uint8((opcode >> 6) & 0x7)
		dstStr, dstExtra, _ := decodeAddressingMode(data[offset:], dstMode, dstReg)
		offset += dstExtra * 2
		inst.Operands = fmt.Sprintf("D%d, %s", dstReg, dstStr)
	}
	inst.Size = uint32(offset)
	if len(data) >= offset {
		inst.Bytes = data[:offset]
	}
	return nil
}

func decodeANDI(data []byte, opcode uint16, inst *Instruction) error {
	size := (opcode >> 6) & 0x3
	sizeStr := []string{"B", "W", "L", "?"}[size]
	dstMode := uint8((opcode >> 3) & 0x7)
	dstReg := uint8(opcode & 0x7)
	offset := 2
	immediate := uint32(0)
	if len(data) < offset+2 {
		return fmt.Errorf("insufficient data for ANDI")
	}
	immediate = uint32(binary.BigEndian.Uint16(data[offset : offset+2]))
	offset += 2
	dstOperand, extraWords, err := decodeAddressingMode(data[offset:], dstMode, dstReg)
	if err != nil {
		return err
	}
	offset += extraWords * 2
	inst.Mnemonic = "ANDI." + sizeStr
	immStr := formatImmediate(immediate, 2)
	inst.Operands = fmt.Sprintf("#%s, %s", immStr, dstOperand)
	inst.Size = uint32(offset)
	if len(data) >= offset {
		inst.Bytes = data[:offset]
	}
	return nil
}

func decodeORI(data []byte, opcode uint16, inst *Instruction) error {
	size := (opcode >> 6) & 0x3
	sizeStr := []string{"B", "W", "L", "?"}[size]
	dstMode := uint8((opcode >> 3) & 0x7)
	dstReg := uint8(opcode & 0x7)
	offset := 2
	if len(data) < offset+2 {
		return fmt.Errorf("insufficient data for ORI")
	}
	immediate := uint32(binary.BigEndian.Uint16(data[offset : offset+2]))
	offset += 2
	dstOperand, extraWords, err := decodeAddressingMode(data[offset:], dstMode, dstReg)
	if err != nil {
		return err
	}
	offset += extraWords * 2
	inst.Mnemonic = "ORI." + sizeStr
	immStr := formatImmediate(immediate, 2)
	inst.Operands = fmt.Sprintf("#%s, %s", immStr, dstOperand)
	inst.Size = uint32(offset)
	if len(data) >= offset {
		inst.Bytes = data[:offset]
	}
	return nil
}

func decodeEORI(data []byte, opcode uint16, inst *Instruction) error {
	size := (opcode >> 6) & 0x3
	sizeStr := []string{"B", "W", "L", "?"}[size]
	dstMode := uint8((opcode >> 3) & 0x7)
	dstReg := uint8(opcode & 0x7)
	offset := 2
	if len(data) < offset+2 {
		return fmt.Errorf("insufficient data for EORI")
	}
	immediate := uint32(binary.BigEndian.Uint16(data[offset : offset+2]))
	offset += 2
	dstOperand, extraWords, err := decodeAddressingMode(data[offset:], dstMode, dstReg)
	if err != nil {
		return err
	}
	offset += extraWords * 2
	inst.Mnemonic = "EORI." + sizeStr
	immStr := formatImmediate(immediate, 2)
	inst.Operands = fmt.Sprintf("#%s, %s", immStr, dstOperand)
	inst.Size = uint32(offset)
	if len(data) >= offset {
		inst.Bytes = data[:offset]
	}
	return nil
}
