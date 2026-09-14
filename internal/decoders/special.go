package decoders

import (
	"encoding/binary"
	"fmt"
)

// Miscellaneous system/control instructions that don't belong to one of the
// arithmetic/logical/move families: no-operand instructions, LEA/PEA/SWAP,
// UNLK/LINK, EXT/EXTB, CHK, EXG, and the 68020+ CALLM/RTM module calls.

var (
	decodeNOP     = noOperand("NOP")
	decodeRTS     = noOperand("RTS")
	decodeRESET   = noOperand("RESET")
	decodeRTE     = noOperand("RTE")
	decodeRTR     = noOperand("RTR")
	decodeTRAPV   = noOperand("TRAPV")
	decodeILLEGAL = noOperand("ILLEGAL") // must precede TST in the opcode table (see opcodetable.go)
	decodeBGND    = noOperand("BGND")    // CPU32 only, must precede ILLEGAL/TST

	decodeSWAP = singleRegisterOperand("SWAP", RegisterKindData)
	decodeUNLK = singleRegisterOperand("UNLK", RegisterKindAddress)
	decodeEXTB = singleRegisterOperand("EXTB.L", RegisterKindData) // 68020+
)

// decodeRTD - Return and Deallocate Parameters (68010+, exact opcode 0x4E74)
func decodeRTD(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	if err := requireLength(data, 4, "RTD displacement"); err != nil {
		return err
	}
	disp := int16(binary.BigEndian.Uint16(data[2:4]))
	immText := fmt.Sprintf("#%s", formatImmediate(uint32(uint16(disp)), 2))
	setInstruction(data, inst, 4, "RTD", immText, immediateOperand(immText, uint32(uint16(disp)), 2))
	return nil
}

func decodeLEA(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	regX := uint8((opcode >> 9) & 0x7)
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	operand, offset, meta, err := decodeEA(data, inst.Address, 2, mode, reg, cpu)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, "LEA", fmt.Sprintf("%s, A%d", operand, regX), meta, registerOperand(RegisterKindAddress, regX))
	return nil
}

func decodePEA(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeUnaryEA("PEA", data, opcode, inst, cpu)
}

func decodeSTOP(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	if err := requireLength(data, 4, "STOP immediate"); err != nil {
		return err
	}
	immediate := binary.BigEndian.Uint16(data[2:4])
	immText := fmt.Sprintf("#%s", formatImmediate(uint32(immediate), 2))
	setInstruction(data, inst, 4, "STOP", immText, immediateOperand(immText, uint32(immediate), 2))
	return nil
}

func decodeTRAP(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	vector := opcode & 0xF
	immText := fmt.Sprintf("#%d", vector)
	setInstruction(data, inst, 2, "TRAP", immText, immediateOperand(immText, uint32(vector), 1))
	return nil
}

// decodeLINK - Link and Allocate. Word-displacement form (0x4E50-0x4E57) is
// available on every CPU; the 32-bit-displacement form (0x4808-0x480F) is
// 68020 and later.
func decodeLINK(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeLinkGeneric(data, opcode, inst, 2)
}

func decodeLINKLong(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeLinkGeneric(data, opcode, inst, 4)
}

func decodeLinkGeneric(data []byte, opcode uint16, inst *Instruction, dispSize int) error {
	regMeta := registerOperand(RegisterKindAddress, uint8(opcode&0x7))
	size := 2 + dispSize

	if err := requireLength(data, size, "LINK displacement"); err != nil {
		return err
	}
	var disp uint32
	if dispSize == 4 {
		disp = binary.BigEndian.Uint32(data[2:size])
	} else {
		disp = uint32(binary.BigEndian.Uint16(data[2:size]))
	}
	sizeStr := "W"
	if dispSize == 4 {
		sizeStr = "L"
	}
	immText := fmt.Sprintf("#%s", formatImmediate(disp, dispSize))
	setInstruction(data, inst, size, "LINK."+sizeStr, fmt.Sprintf("%s, %s", regMeta.Text, immText), regMeta, immediateOperand(immText, disp, dispSize))
	return nil
}

// decodeEXT - Sign Extend. Opmode (bits 8-6) selects B->W (0x4880 family) or
// W->L (0x48C0 family); EXTB.L (byte->long, 0x49C0 family) is 68020+.
func decodeEXT(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	sizeStr := "W"
	if opcode&0x0040 != 0 {
		sizeStr = "L"
	}
	meta := registerOperand(RegisterKindData, uint8(opcode&0x7))
	setInstruction(data, inst, 2, "EXT."+sizeStr, meta.Text, meta)
	return nil
}

// decodeCHK - Check Register Against Bounds. Opmode 110 is the word form
// (all CPUs); opmode 100 is the long form, 68020 and later.
func decodeCHK(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	sizeStr := "W"
	sizeBytes := 2
	if opcode&0x0080 == 0 {
		sizeStr, sizeBytes = "L", 4
	}
	dstReg := uint8((opcode >> 9) & 0x7)
	srcMode := uint8((opcode >> 3) & 0x7)
	srcReg := uint8(opcode & 0x7)

	srcOperand, offset, srcMeta, err := decodeEAWithSize(data, inst.Address, 2, srcMode, srcReg, sizeBytes, cpu)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, "CHK."+sizeStr, fmt.Sprintf("%s, D%d", srcOperand, dstReg), srcMeta, registerOperand(RegisterKindData, dstReg))
	return nil
}

// decodeEXG - Exchange Registers. The 5-bit opmode field (bits 7-3)
// distinguishes Dx,Dy / Ax,Ay / Dx,Ay.
func decodeEXG(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	rx := uint8((opcode >> 9) & 0x7)
	ry := uint8(opcode & 0x7)
	opmode := (opcode >> 3) & 0x1F

	xKind, yKind := RegisterKindData, RegisterKindData
	switch opmode {
	case 0x08: // Dx,Dy
	case 0x09: // Ax,Ay
		xKind, yKind = RegisterKindAddress, RegisterKindAddress
	case 0x11: // Dx,Ay
		yKind = RegisterKindAddress
	default:
		return fmt.Errorf("unknown EXG opmode: %05b", opmode)
	}

	xMeta := registerOperand(xKind, rx)
	yMeta := registerOperand(yKind, ry)
	setInstruction(data, inst, 2, "EXG", fmt.Sprintf("%s, %s", xMeta.Text, yMeta.Text), xMeta, yMeta)
	return nil
}

// decodeCALLM - Call Module (68020/68030 only). Format: 0000 0110 11 mmm
// rrr, followed by a word extension whose low byte is the argument count.
func decodeCALLM(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)

	if err := requireLength(data, 4, "CALLM argument count"); err != nil {
		return err
	}
	argCount := binary.BigEndian.Uint16(data[2:4]) & 0xFF
	argText := fmt.Sprintf("#%d", argCount)

	operand, offset, meta, err := decodeEAWithSize(data, inst.Address, 4, mode, reg, 2, cpu)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, "CALLM", fmt.Sprintf("%s, %s", argText, operand), immediateOperand(argText, uint32(argCount), 1), meta)
	return nil
}

// decodeRTM - Return from Module (68020/68030 only). Reuses CALLM's
// register-direct/address-direct EA slots (0x06C0-0x06CF), so it must be
// registered before CALLM in the opcode table.
func decodeRTM(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	reg := uint8(opcode & 0x7)
	kind := RegisterKindData
	if opcode&0x8 != 0 {
		kind = RegisterKindAddress
	}
	meta := registerOperand(kind, reg)
	setInstruction(data, inst, 2, "RTM", meta.Text, meta)
	return nil
}

// formatRegisterList expands a MOVEM register-list mask. When reverse is set the
// mask is read most-significant-bit first (the -(An) predecrement encoding).
func formatRegisterList(regListMask uint16, reverse bool) (string, []string) {
	bitFor := func(listIndex int) uint {
		if reverse {
			return uint(15 - listIndex)
		}
		return uint(listIndex)
	}

	var registers []string
	for i := range 8 {
		if regListMask&(1<<bitFor(i)) != 0 {
			registers = append(registers, fmt.Sprintf("D%d", i))
		}
	}
	for i := range 8 {
		if regListMask&(1<<bitFor(i+8)) != 0 {
			registers = append(registers, fmt.Sprintf("A%d", i))
		}
	}
	return formatRegisterRange(registers), registers
}

func formatRegisterRange(registers []string) string {
	if len(registers) == 0 {
		return ""
	}
	result := ""
	i := 0
	for i < len(registers) {
		if result != "" {
			result += "/"
		}
		start := registers[i]
		end := start
		j := i + 1
		for j < len(registers) {
			prevNum := extractRegNum(registers[j-1])
			currNum := extractRegNum(registers[j])
			if prevNum >= 0 && currNum >= 0 && currNum == prevNum+1 {
				end = registers[j]
				j++
			} else {
				break
			}
		}
		if start == end {
			result += start
		} else {
			result += start + "-" + end
		}
		i = j
	}
	return result
}

// extractRegNum returns the trailing digit(s) of a register name (e.g. 3
// for "D3", "A3", or "FP3" — any single-letter or multi-letter prefix), or
// -1 if regName doesn't end in a single-digit 0-7 register number.
func extractRegNum(regName string) int {
	if len(regName) < 2 {
		return -1
	}
	num := regName[len(regName)-1] - '0'
	if num <= 7 {
		return int(num)
	}
	return -1
}
