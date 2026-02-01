package decoders

import (
	"fmt"
)

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
