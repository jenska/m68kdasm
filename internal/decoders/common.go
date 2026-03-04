package decoders

import "fmt"

// decodeNOP - No Operation (exact opcode: 0x4E71)
func decodeNOP(data []byte, opcode uint16, inst *Instruction) error {
	inst.Mnemonic = "NOP"
	inst.Size = 2
	if len(data) >= 2 {
		inst.Bytes = data[:2]
	}
	return nil
}

// decodeRTS - Return from Subroutine (exact opcode: 0x4E75)
func decodeRTS(data []byte, opcode uint16, inst *Instruction) error {
	inst.Mnemonic = "RTS"
	inst.Size = 2
	if len(data) >= 2 {
		inst.Bytes = data[:2]
	}
	return nil
}

// getSizeString converts 68000 size field (bits 6-7) to string
// Maps: 0=B (byte), 1=W (word), 2=L (long), 3=? (undefined)
func getSizeString(size uint16, options ...string) string {
	sizeMap := []string{"B", "W", "L", "?"}
	if int(size) < len(sizeMap) {
		return sizeMap[size]
	}
	return "?"
}

// buildDirectedOperands creates reversed operands based on direction bit
// direction == 0: src, dstReg format
// direction != 0: dstReg, src format
func buildDirectedOperands(direction uint16, src string, dstReg uint8) string {
	if direction == 0 {
		return fmt.Sprintf("%s, D%d", src, dstReg)
	}
	return fmt.Sprintf("D%d, %s", dstReg, src)
}

// setInstructionSize sets the instruction's Size and Bytes fields
func setInstructionSize(data []byte, inst *Instruction, offset int) {
	inst.Size = uint32(offset)
	if len(data) >= offset {
		inst.Bytes = data[:offset]
	}
}
