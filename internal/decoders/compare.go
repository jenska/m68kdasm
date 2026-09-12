package decoders

import (
	"encoding/binary"
	"fmt"
)

func decodeCMP(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	opmode := (opcode >> 6) & 0x7
	if (opcode & 0xF138) == 0xB108 {
		return decodeCMPM(data, opcode, inst)
	}
	if opmode == 3 || opmode == 7 {
		return decodeAddressRegisterOp("CMP", data, opcode, inst, cpu)
	}
	if opmode >= 4 && opmode <= 6 {
		return decodeEOR(data, opcode, inst, cpu)
	}

	sizeStr := getSizeString(opmode)
	sizeBytes, err := operandSize(opmode, "CMP")
	if err != nil {
		return err
	}
	dstReg := uint8((opcode >> 9) & 0x7)
	srcMode := uint8((opcode >> 3) & 0x7)
	srcReg := uint8(opcode & 0x7)

	srcStr, offset, srcMeta, err := decodeEAWithSize(data, inst.Address, 2, srcMode, srcReg, sizeBytes, cpu)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, "CMP."+sizeStr, fmt.Sprintf("%s, D%d", srcStr, dstReg), srcMeta, registerOperand(RegisterKindData, dstReg))
	return nil
}

func decodeCMPM(data []byte, opcode uint16, inst *Instruction) error {
	sizeBits := (opcode >> 6) & 0x3
	sizeStr := getSizeString(sizeBits)
	srcReg := uint8(opcode & 0x7)
	dstReg := uint8((opcode >> 9) & 0x7)
	srcText := fmt.Sprintf("(A%d)+", srcReg)
	dstText := fmt.Sprintf("(A%d)+", dstReg)
	setInstruction(data, inst, 2, "CMPM."+sizeStr, fmt.Sprintf("%s, %s", srcText, dstText),
		addrIndirectOperand(EAKindPostIncrement, srcReg, srcText),
		addrIndirectOperand(EAKindPostIncrement, dstReg, dstText))
	return nil
}

func decodeCMPI(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeImmediateBinaryOp("CMPI", data, opcode, inst, false, cpu)
}

// decodeCHK2CMP2 - Check/Compare Register Against Bounds (68020+).
// Format: 0000 0ss0 11 mmm rrr (ss: 00=B,01=W,10=L), followed by an
// extension word: bit15=register kind(0=Dn,1=An), bits14-12=Rn,
// bit11=0(CMP2)/1(CHK2).
func decodeCHK2CMP2(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	sizeBits := (opcode >> 9) & 0x3
	sizeStr := getSizeString(sizeBits)
	sizeBytes, err := operandSize(sizeBits, "CHK2/CMP2")
	if err != nil {
		return err
	}
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)

	boundsOperand, offset, boundsMeta, err := decodeEAWithSize(data, inst.Address, 2, mode, reg, sizeBytes, cpu)
	if err != nil {
		return err
	}

	if err := requireLength(data, offset+2, "CHK2/CMP2 extension word"); err != nil {
		return err
	}
	ext := binary.BigEndian.Uint16(data[offset : offset+2])
	regKind := RegisterKindData
	if ext&0x8000 != 0 {
		regKind = RegisterKindAddress
	}
	mnemonic := "CMP2"
	if ext&0x0800 != 0 {
		mnemonic = "CHK2"
	}
	regMeta := registerOperand(regKind, uint8((ext>>12)&0x7))
	setInstruction(data, inst, offset+2, mnemonic+"."+sizeStr, fmt.Sprintf("%s, %s", boundsOperand, regMeta.Text), boundsMeta, regMeta)
	return nil
}

// decodeCAS - Compare and Swap (68020+, single-operand form only; CAS2 is
// deliberately not implemented — its extension-word encoding packs two
// independent register triples across two extension words and wasn't
// confidently enough recalled to encode correctly without a primary-source
// cross-check).
// Format: 0000 1ss 011 mmm rrr (ss: 01=B,10=W,11=L), followed by an
// extension word: bits5-3=Dc (compare register), bits2-0=Du (update register).
func decodeCAS(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	var sizeStr string
	var sizeBytes int
	switch (opcode >> 9) & 0x3 {
	case 1:
		sizeStr, sizeBytes = "B", 1
	case 2:
		sizeStr, sizeBytes = "W", 2
	case 3:
		sizeStr, sizeBytes = "L", 4
	default:
		return fmt.Errorf("unknown CAS size selector: %d", (opcode>>9)&0x3)
	}
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)

	if err := requireLength(data, 4, "CAS extension word"); err != nil {
		return err
	}
	ext := binary.BigEndian.Uint16(data[2:4])
	dcMeta := registerOperand(RegisterKindData, uint8((ext>>3)&0x7))
	duMeta := registerOperand(RegisterKindData, uint8(ext&0x7))

	eaOperand, offset, eaMeta, err := decodeEAWithSize(data, inst.Address, 4, mode, reg, sizeBytes, cpu)
	if err != nil {
		return err
	}

	setInstruction(data, inst, offset, "CAS."+sizeStr, fmt.Sprintf("%s, %s, %s", dcMeta.Text, duMeta.Text, eaOperand), dcMeta, duMeta, eaMeta)
	return nil
}
