package decoders

import (
	"encoding/binary"
	"fmt"
)

func decodeMOVEQ(data []byte, opcode uint16, inst *Instruction) error {
	dstReg := uint8((opcode >> 9) & 0x7)
	immediate := int8(opcode & 0xFF)
	inst.Mnemonic = "MOVEQ"
	immStr := formatImmediateForMOVEQ(int32(immediate))
	inst.Operands = fmt.Sprintf("#%s, D%d", immStr, dstReg)
	inst.Size = 2
	if len(data) >= 2 {
		inst.Bytes = data[:2]
	}
	return nil
}

// decodeMOVE - Move data
// MOVE Format: 00ss ddd mmm rrr (source and destination can use all addressing modes)
// ss = size (01=Byte, 11=Word, 10=Long)
func decodeMOVE(data []byte, opcode uint16, inst *Instruction) error {
	// Size: extract bits 12-13 (note: 00=reserved, 01=B, 11=W, 10=L)
	sizeField := (opcode >> 12) & 0x3

	var sizeStr string
	switch sizeField {
	case 1:
		sizeStr = "B"
	case 3:
		sizeStr = "W"
	case 2:
		sizeStr = "L"
	default:
		return fmt.Errorf("unbekannte MOVE-Größe: %d", sizeField)
	}

	// Destination: bits 9-11 (register), bits 6-8 (mode)
	dstReg := uint8((opcode >> 9) & 0x7)
	dstMode := uint8((opcode >> 6) & 0x7)

	// Source: bits 0-5 (register and mode)
	srcReg := uint8(opcode & 0x7)
	srcMode := uint8((opcode >> 3) & 0x7)

	offset := 2

	// Decode source addressing mode
	srcStr, srcExtraWords, err := decodeAddressingMode(data[2:], srcMode, srcReg)
	if err != nil {
		return err
	}
	offset += srcExtraWords * 2

	// Decode destination addressing mode
	dstStr, dstExtraWords, err := decodeAddressingMode(data[offset:], dstMode, dstReg)
	if err != nil {
		return err
	}
	offset += dstExtraWords * 2

	inst.Mnemonic = "MOVE." + sizeStr
	inst.Operands = fmt.Sprintf("%s, %s", srcStr, dstStr)
	inst.Size = uint32(offset)
	if len(data) >= offset {
		inst.Bytes = data[:offset]
	}

	return nil
}

func decodeMOVEM(data []byte, opcode uint16, inst *Instruction) error {
	if len(data) < 4 {
		return fmt.Errorf("insufficient data for MOVEM")
	}
	direction := (opcode >> 10) & 0x1
	sizeStr := "W"
	if opcode&0x0040 != 0 {
		sizeStr = "L"
	}
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	regListMask := binary.BigEndian.Uint16(data[2:4])
	offset := 4
	var addrModeStr string
	var extraWords int
	var err error
	if mode != 0 || reg != 0 {
		addrModeStr, extraWords, err = decodeAddressingMode(data[offset:], mode, reg)
		if err != nil {
			return err
		}
		offset += extraWords * 2
	}
	inst.Mnemonic = "MOVEM." + sizeStr
	regList := formatRegisterList(regListMask, direction)
	if direction == 0 {
		inst.Operands = fmt.Sprintf("%s, %s", regList, addrModeStr)
	} else {
		inst.Operands = fmt.Sprintf("%s, %s", addrModeStr, regList)
	}
	inst.Size = uint32(offset)
	if len(data) >= offset {
		inst.Bytes = data[:offset]
	}
	return nil
}
