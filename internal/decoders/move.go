package decoders

import (
	"encoding/binary"
	"fmt"
)

func decodeMOVEQ(data []byte, opcode uint16, inst *Instruction) error {
	dstReg := uint8((opcode >> 9) & 0x7)
	immediate := int8(opcode & 0xFF)
	immText := fmt.Sprintf("#%s", formatImmediateForMOVEQ(int32(immediate)))
	setInstruction(data, inst, 2, "MOVEQ", fmt.Sprintf("%s, D%d", immText, dstReg), immediateOperand(immText, uint32(uint8(immediate)), 1), registerOperand(RegisterKindData, dstReg))
	return nil
}

// decodeMOVE - Move data
// MOVE Format: 00ss ddd mmm rrr (source and destination can use all addressing modes)
// ss = size (01=Byte, 11=Word, 10=Long)
func decodeMOVE(data []byte, opcode uint16, inst *Instruction) error {
	// Size: extract bits 12-13 (note: 00=reserved, 01=B, 11=W, 10=L)
	sizeField := (opcode >> 12) & 0x3

	var sizeStr string
	var sizeBytes int
	switch sizeField {
	case 1:
		sizeStr = "B"
		sizeBytes = 1
	case 3:
		sizeStr = "W"
		sizeBytes = 2
	case 2:
		sizeStr = "L"
		sizeBytes = 4
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
	srcStr, offset, srcMeta, err := decodeEAWithSize(data, offset, srcMode, srcReg, sizeBytes)
	if err != nil {
		return err
	}

	if dstMode == 1 {
		if sizeField == 1 {
			return fmt.Errorf("MOVEA does not support byte size")
		}
		setInstruction(data, inst, offset, "MOVEA."+sizeStr, fmt.Sprintf("%s, A%d", srcStr, dstReg), srcMeta, registerOperand(RegisterKindAddress, dstReg))
		return nil
	}

	// Decode destination addressing mode
	dstStr, offset, dstMeta, err := decodeEA(data, offset, dstMode, dstReg)
	if err != nil {
		return err
	}

	setInstruction(data, inst, offset, "MOVE."+sizeStr, fmt.Sprintf("%s, %s", srcStr, dstStr), srcMeta, dstMeta)

	return nil
}

func decodeMOVEM(data []byte, opcode uint16, inst *Instruction) error {
	if err := requireLength(data, 4, "MOVEM register list"); err != nil {
		return err
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
	var addrModeMeta Operand
	var err error
	if mode != 0 || reg != 0 {
		addrModeStr, offset, addrModeMeta, err = decodeEA(data, offset, mode, reg)
		if err != nil {
			return err
		}
	}
	regListText, registers := formatRegisterList(regListMask, direction)
	regListMeta := registerListOperand(regListText, registers)
	if direction == 0 {
		setInstruction(data, inst, offset, "MOVEM."+sizeStr, fmt.Sprintf("%s, %s", regListText, addrModeStr), regListMeta, addrModeMeta)
		return nil
	}
	setInstruction(data, inst, offset, "MOVEM."+sizeStr, fmt.Sprintf("%s, %s", addrModeStr, regListText), addrModeMeta, regListMeta)
	return nil
}
