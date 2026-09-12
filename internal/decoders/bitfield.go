package decoders

import (
	"encoding/binary"
	"fmt"
)

// Bitfield instructions (68020+): BFTST, BFEXTU, BFCHG, BFEXTS, BFCLR,
// BFFFO, BFSET, BFINS. Opcode: 1110 oooo 11 mmm rrr (oooo selects the
// instruction), followed by an extension word:
//
//	bit15    reserved (0)
//	bit14    offset type: 0 = 5-bit immediate in bits 10-6, 1 = register in bits 10-8
//	bits13-11 reserved (0)
//	bits10-6 offset (immediate value, or register number when bit14=1)
//	bit5     width type: 0 = 5-bit immediate in bits 4-0, 1 = register in bits 2-0
//	bits4-0  width (immediate value — 0 means 32 — or register number when bit5=1)
//
// BFEXTU/BFEXTS/BFFFO additionally use bits14-12 of the extension word as a
// destination data register; BFINS uses the same field as its source
// register (operand order reversed: "BFINS Dn,<ea>{offset:width}").

type bitfieldSpec struct {
	offsetIsReg bool
	offsetReg   uint8
	offsetImm   uint8
	widthIsReg  bool
	widthReg    uint8
	widthImm    uint8 // 0 means 32
}

func parseBitfieldExt(ext uint16) bitfieldSpec {
	var spec bitfieldSpec
	spec.offsetIsReg = ext&0x0800 != 0
	if spec.offsetIsReg {
		// Register number occupies bits 10-8 (the top 3 bits of the 5-bit
		// offset field); bits 7-6 are unused in this case.
		spec.offsetReg = uint8((ext >> 8) & 0x7)
	} else {
		spec.offsetImm = uint8((ext >> 6) & 0x1F)
	}
	spec.widthIsReg = ext&0x0020 != 0
	if spec.widthIsReg {
		spec.widthReg = uint8(ext & 0x7)
	} else {
		spec.widthImm = uint8(ext & 0x1F)
	}
	return spec
}

func (s bitfieldSpec) offsetText() string {
	if s.offsetIsReg {
		return fmt.Sprintf("D%d", s.offsetReg)
	}
	return fmt.Sprintf("%d", s.offsetImm)
}

func (s bitfieldSpec) widthText() string {
	if s.widthIsReg {
		return fmt.Sprintf("D%d", s.widthReg)
	}
	w := s.widthImm
	if w == 0 {
		w = 32
	}
	return fmt.Sprintf("%d", w)
}

func decodeBFTST(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeBitfield("BFTST", bitfieldNoReg, data, opcode, inst, cpu)
}
func decodeBFCHG(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeBitfield("BFCHG", bitfieldNoReg, data, opcode, inst, cpu)
}
func decodeBFCLR(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeBitfield("BFCLR", bitfieldNoReg, data, opcode, inst, cpu)
}
func decodeBFSET(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeBitfield("BFSET", bitfieldNoReg, data, opcode, inst, cpu)
}
func decodeBFEXTU(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeBitfield("BFEXTU", bitfieldDestReg, data, opcode, inst, cpu)
}
func decodeBFEXTS(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeBitfield("BFEXTS", bitfieldDestReg, data, opcode, inst, cpu)
}
func decodeBFFFO(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeBitfield("BFFFO", bitfieldDestReg, data, opcode, inst, cpu)
}
func decodeBFINS(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeBitfield("BFINS", bitfieldSrcReg, data, opcode, inst, cpu)
}

type bitfieldRegRole int

const (
	bitfieldNoReg bitfieldRegRole = iota
	bitfieldDestReg
	bitfieldSrcReg
)

func decodeBitfield(mnemonic string, role bitfieldRegRole, data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)

	if err := requireLength(data, 4, mnemonic+" bitfield extension word"); err != nil {
		return err
	}
	ext := binary.BigEndian.Uint16(data[2:4])
	spec := parseBitfieldExt(ext)

	eaOperand, offset, eaMeta, err := decodeEAWithSize(data, inst.Address, 4, mode, reg, 4, cpu)
	if err != nil {
		return err
	}

	bitfieldText := fmt.Sprintf("%s{%s:%s}", eaOperand, spec.offsetText(), spec.widthText())

	var operandsText string
	var operands []Operand
	switch role {
	case bitfieldDestReg:
		destMeta := registerOperand(RegisterKindData, uint8((ext>>12)&0x7))
		operandsText = fmt.Sprintf("%s, %s", bitfieldText, destMeta.Text)
		operands = []Operand{eaMeta, destMeta}
	case bitfieldSrcReg:
		srcMeta := registerOperand(RegisterKindData, uint8((ext>>12)&0x7))
		operandsText = fmt.Sprintf("%s, %s", srcMeta.Text, bitfieldText)
		operands = []Operand{srcMeta, eaMeta}
	default:
		operandsText = bitfieldText
		operands = []Operand{eaMeta}
	}

	setInstruction(data, inst, offset, mnemonic, operandsText, operands...)
	return nil
}
