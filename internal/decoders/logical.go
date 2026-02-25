package decoders

import (
	"encoding/binary"
	"fmt"
)

func decodeAND(data []byte, opcode uint16, inst *Instruction) error {
	return decodeLogical("AND", data, opcode, inst)
}

func decodeOR(data []byte, opcode uint16, inst *Instruction) error {
	return decodeLogical("OR", data, opcode, inst)
}

func decodeEOR(data []byte, opcode uint16, inst *Instruction) error {
	return decodeLogical("EOR", data, opcode, inst)
}

func decodeANDI(data []byte, opcode uint16, inst *Instruction) error {
	return decodeLogicalI("ANDI", data, opcode, inst)
}

func decodeORI(data []byte, opcode uint16, inst *Instruction) error {
	return decodeLogicalI("ORI", data, opcode, inst)
}

func decodeEORI(data []byte, opcode uint16, inst *Instruction) error {
	return decodeLogicalI("EORI", data, opcode, inst)
}

func decodeLogicalI(mn string, data []byte, opcode uint16, inst *Instruction) error {
	size := (opcode >> 6) & 0x3
	sizeStr := []string{".B", ".W", ".L", "?"}[size]
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
	inst.Mnemonic = mn + sizeStr
	immStr := formatImmediate(immediate, 2)
	inst.Operands = fmt.Sprintf("#%s, %s", immStr, dstOperand)
	inst.Size = uint32(offset)
	if len(data) >= offset {
		inst.Bytes = data[:offset]
	}
	return nil
}

func decodeLogical(mn string, data []byte, opcode uint16, inst *Instruction) error {
	direction := (opcode >> 8) & 0x1
	size := (opcode >> 6) & 0x3
	sizeStr := []string{".B", ".W", ".L", "?"}[size]
	dstReg := uint8((opcode >> 9) & 0x7)
	srcMode := uint8((opcode >> 3) & 0x7)
	srcReg := uint8(opcode & 0x7)
	offset := 2
	srcStr, srcExtra, err := decodeAddressingMode(data[2:], srcMode, srcReg)
	if err != nil {
		return err
	}
	offset += srcExtra * 2
	inst.Mnemonic = mn + sizeStr
	if direction == 0 {
		inst.Operands = fmt.Sprintf("%s, D%d", srcStr, dstReg)
	} else {
		inst.Operands = fmt.Sprintf("D%d, %s", dstReg, srcStr)
	}
	inst.Size = uint32(offset)
	if len(data) >= offset {
		inst.Bytes = data[:offset]
	}
	return nil
}
