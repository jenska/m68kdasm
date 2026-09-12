package decoders

import (
	"encoding/binary"
	"fmt"
)

func decodeMOVEQ(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	dstReg := uint8((opcode >> 9) & 0x7)
	immediate := int8(opcode & 0xFF)
	immText := fmt.Sprintf("#%s", formatImmediateForMOVEQ(int32(immediate)))
	setInstruction(data, inst, 2, "MOVEQ", fmt.Sprintf("%s, D%d", immText, dstReg), immediateOperand(immText, uint32(uint8(immediate)), 1), registerOperand(RegisterKindData, dstReg))
	return nil
}

// decodeMOVE - Move data
// MOVE Format: 00ss ddd mmm rrr (source and destination can use all addressing modes)
// ss = size (01=Byte, 11=Word, 10=Long)
func decodeMOVE(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
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
	srcStr, offset, srcMeta, err := decodeEAWithSize(data, inst.Address, offset, srcMode, srcReg, sizeBytes, cpu)
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
	dstStr, offset, dstMeta, err := decodeEA(data, inst.Address, offset, dstMode, dstReg, cpu)
	if err != nil {
		return err
	}

	setInstruction(data, inst, offset, "MOVE."+sizeStr, fmt.Sprintf("%s, %s", srcStr, dstStr), srcMeta, dstMeta)

	return nil
}

func decodeMOVEM(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
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
		addrModeStr, offset, addrModeMeta, err = decodeEA(data, inst.Address, offset, mode, reg, cpu)
		if err != nil {
			return err
		}
	}
	regListText, registers := formatRegisterList(regListMask, direction == 0 && mode == 4)
	regListMeta := registerListOperand(regListText, registers)
	if direction == 0 {
		setInstruction(data, inst, offset, "MOVEM."+sizeStr, fmt.Sprintf("%s, %s", regListText, addrModeStr), regListMeta, addrModeMeta)
		return nil
	}
	setInstruction(data, inst, offset, "MOVEM."+sizeStr, fmt.Sprintf("%s, %s", addrModeStr, regListText), addrModeMeta, regListMeta)
	return nil
}

// decodeMOVEP - Move Peripheral Data. Format: 0000 ddd 1 dr sz 001 aaa,
// where dr=direction (0=memory->register,1=register->memory) and
// sz=size(0=Word,1=Long), followed by a 16-bit signed displacement applied
// to Ay (matching address-register-indirect-with-displacement addressing).
func decodeMOVEP(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	dReg := uint8((opcode >> 9) & 0x7)
	aReg := uint8(opcode & 0x7)
	registerToMemory := opcode&0x0080 != 0
	sizeStr := "W"
	if opcode&0x0040 != 0 {
		sizeStr = "L"
	}

	if err := requireLength(data, 4, "MOVEP displacement"); err != nil {
		return err
	}
	disp := int32(int16(binary.BigEndian.Uint16(data[2:4])))
	memText := fmt.Sprintf("(%d,A%d)", disp, aReg)
	memMeta := effectiveAddressOperand(memText, EffectiveAddress{
		Kind:         EAKindDisplacement,
		Register:     aReg,
		Base:         &Register{Kind: RegisterKindAddress, Number: aReg},
		Displacement: new(disp),
	})
	dMeta := registerOperand(RegisterKindData, dReg)

	var operandsText string
	var operands []Operand
	if registerToMemory {
		operandsText = fmt.Sprintf("%s, %s", dMeta.Text, memText)
		operands = []Operand{dMeta, memMeta}
	} else {
		operandsText = fmt.Sprintf("%s, %s", memText, dMeta.Text)
		operands = []Operand{memMeta, dMeta}
	}
	setInstruction(data, inst, 4, "MOVEP."+sizeStr, operandsText, operands...)
	return nil
}

// decodeMOVES - Move Address Space (68010+)
// Format: 0000 1110 ss mmm rrr, extension word: [A/D][Rn:3][dr:1][reserved:11]
func decodeMOVES(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	if err := requireLength(data, 4, "MOVES extension word"); err != nil {
		return err
	}

	sizeBits := (opcode >> 6) & 0x3
	sizeStr := getSizeString(sizeBits)
	sizeBytes, err := operandSize(sizeBits, "MOVES")
	if err != nil {
		return err
	}

	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)

	ext := binary.BigEndian.Uint16(data[2:4])
	kind := RegisterKindData
	if ext&0x8000 != 0 {
		kind = RegisterKindAddress
	}
	regNum := uint8((ext >> 12) & 0x7)
	regMeta := registerOperand(kind, regNum)
	registerToMemory := ext&0x0800 == 0

	eaOperand, offset, eaMeta, err := decodeEAWithSize(data, inst.Address, 4, mode, reg, sizeBytes, cpu)
	if err != nil {
		return err
	}

	var operands string
	var first, second Operand
	if registerToMemory {
		operands = fmt.Sprintf("%s, %s", regMeta.Text, eaOperand)
		first, second = regMeta, eaMeta
	} else {
		operands = fmt.Sprintf("%s, %s", eaOperand, regMeta.Text)
		first, second = eaMeta, regMeta
	}
	setInstruction(data, inst, offset, "MOVES."+sizeStr, operands, first, second)
	return nil
}

// controlRegisterNames maps MOVEC control-register codes to their names.
// Not exhaustively gated by CPU (e.g. CACR is 68020+, PCR is 68060-only):
// this is a display lookup, not a validity check, so an unrecognized or
// CPU-inappropriate code just falls back to its raw hex form.
var controlRegisterNames = map[uint16]string{
	0x000: "SFC",
	0x001: "DFC",
	0x800: "USP",
	0x801: "VBR",
	0x002: "CACR",  // 68020 and later
	0x802: "CAAR",  // 68020/68030 only
	0x803: "MSP",   // 68020 and later
	0x804: "ISP",   // 68020 and later
	0x003: "TC",    // 68040 MMU
	0x004: "ITT0",  // 68040/68060
	0x005: "ITT1",  // 68040/68060
	0x006: "DTT0",  // 68040/68060
	0x007: "DTT1",  // 68040/68060
	0x805: "MMUSR", // 68040
	0x806: "URP",   // 68040/68060
	0x807: "SRP",   // 68040/68060
	0x808: "PCR",   // 68060
}

func controlRegisterName(code uint16) string {
	if name, ok := controlRegisterNames[code]; ok {
		return name
	}
	return fmt.Sprintf("$%03X", code)
}

// decodeMOVEC - Move Control Register (68010+). Two opcodes share one
// extension-word layout: bit15=register kind, bits14-12=Rn, bits11-0=control
// register code.
func decodeMOVECFromControl(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeMOVEC(data, inst, true)
}

func decodeMOVECToControl(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeMOVEC(data, inst, false)
}

func decodeMOVEC(data []byte, inst *Instruction, fromControl bool) error {
	if err := requireLength(data, 4, "MOVEC control register"); err != nil {
		return err
	}
	ext := binary.BigEndian.Uint16(data[2:4])

	kind := RegisterKindData
	if ext&0x8000 != 0 {
		kind = RegisterKindAddress
	}
	regMeta := registerOperand(kind, uint8((ext>>12)&0x7))
	ccMeta := Operand{Text: controlRegisterName(ext & 0x0FFF), Kind: OperandKindRegister}

	var operands string
	var first, second Operand
	if fromControl {
		operands = fmt.Sprintf("%s, %s", ccMeta.Text, regMeta.Text)
		first, second = ccMeta, regMeta
	} else {
		operands = fmt.Sprintf("%s, %s", regMeta.Text, ccMeta.Text)
		first, second = regMeta, ccMeta
	}
	setInstruction(data, inst, 4, "MOVEC", operands, first, second)
	return nil
}
