package decoders

import (
	"encoding/binary"
	"fmt"
)

// This file decodes the PMMU's own conditional branch/set/trap family —
// PBcc/PDBcc/PScc/PTRAPcc — structurally the exact same shape already
// built twice before: once for the integer ISA's Bcc/DBcc/Scc/TRAPcc
// (branch.go), once for the FPU's own FBcc/FDBcc/FScc/FTRAPcc
// (fpu.go). The only real differences: 16 conditions instead of 32 (a
// 4-bit field, not 5), and no coprocessor-ID bit folded into word1 (PMMU's
// word1 is a bare 0xF0xx, unlike the FPU family's 0xF2xx — see
// fpuWord1Base's doc comment in fpu.go for why FPU needs that fold and
// PMMU never does).
//
// pmmuConditions names the PMMU's 16 condition codes, indexed by
// condition value (0-15). Copied verbatim from m68kasm's pmmuConditions
// (cpu030_pmmu_cond.go), itself decoded from GAS's m68k opcode table.
var pmmuConditions = [...]string{
	"BS", "BC", "LS", "LC", "SS", "SC", "AS", "AC",
	"WS", "WC", "IS", "IC", "GS", "GC", "CS", "CC",
}

func pmmuConditionName(cc uint16) string {
	if int(cc) < len(pmmuConditions) {
		return pmmuConditions[cc]
	}
	return fmt.Sprintf("?%d", cc)
}

// decodePBcc decodes both PBcc forms — word displacement (word1 =
// 0xF080|cc) and long displacement (word1 = 0xF0C0|cc, distinguished by
// bit 6, the one bit that differs between the two base literals) — using
// branchTarget (branch.go) for the target address, same as every other
// PC-relative branch family in this codebase. Unlike FBcc, there is no
// PMMU condition aliased to a NOP-like mnemonic (pmmuConditions has no
// "always false" entry the way the FPU's condition 0, "F", does).
func decodePBcc(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	cc := opcode & 0xF
	mnemonic := "PB" + pmmuConditionName(cc)
	long := opcode&0x0040 != 0

	if long {
		if err := requireLength(data, 6, mnemonic+".L displacement"); err != nil {
			return err
		}
		disp := int32(binary.BigEndian.Uint32(data[2:6]))
		target := branchTarget(inst.Address, disp)
		targetText := formatBranchTarget(target)
		setInstruction(data, inst, 6, mnemonic+".L", targetText, branchOperand(targetText, target))
		return nil
	}

	if err := requireLength(data, 4, mnemonic+".W displacement"); err != nil {
		return err
	}
	disp := int32(int16(binary.BigEndian.Uint16(data[2:4])))
	target := branchTarget(inst.Address, disp)
	targetText := formatBranchTarget(target)
	setInstruction(data, inst, 4, mnemonic+".W", targetText, branchOperand(targetText, target))
	return nil
}

// decodePDBcc mirrors decodeFDBcc (fpu.go): word1 only carries the Dn
// register (bits 2-0); the condition lives in a dedicated, fully-fixed
// word2 (0x0000-0x000F).
func decodePDBcc(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	if err := requireLength(data, 6, "PDBcc"); err != nil {
		return err
	}
	word2 := binary.BigEndian.Uint16(data[2:4])
	cc := word2 & 0xF
	mnemonic := "PDB" + pmmuConditionName(cc)
	reg := uint8(opcode & 0x7)
	disp := int32(int16(binary.BigEndian.Uint16(data[4:6])))
	target := branchTarget(inst.Address, disp)
	targetText := formatBranchTarget(target)
	regMeta := registerOperand(RegisterKindData, reg)
	setInstruction(data, inst, 6, mnemonic, regMeta.Text+", "+targetText, regMeta, branchOperand(targetText, target))
	return nil
}

// decodePScc mirrors decodeFScc (fpu.go): word1 = 0xF040|<ea> (data-
// alterable, byte-sized destination), word2 = the condition, fully fixed
// like PDBcc's. PDBcc and PTRAPcc's exact opcodes both occupy part of
// this pattern's own EA range (address-register-direct and mode-7/reg-2..4
// respectively) and so must precede this pattern in the opcode table —
// see their registrations in opcodetable.go.
func decodePScc(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	if err := requireLength(data, 4, "PScc"); err != nil {
		return err
	}
	word2 := binary.BigEndian.Uint16(data[2:4])
	cc := word2 & 0xF
	mnemonic := "PS" + pmmuConditionName(cc)
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	operand, offset, meta, err := decodeEAWithSize(data, inst.Address, 4, mode, reg, 1, cpu)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, mnemonic, operand, meta)
	return nil
}

// decodePTRAPccBare/Word/Long mirror decodeFTRAPccBare/Word/Long (fpu.go):
// each form is a fully-fixed word1 literal (0xF07C/0xF07A/0xF07B) rather
// than a shared mask, with the condition in word2.
func decodePTRAPccBare(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodePTRAPcc(data, inst, 0)
}

func decodePTRAPccWord(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodePTRAPcc(data, inst, 2)
}

func decodePTRAPccLong(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodePTRAPcc(data, inst, 4)
}

func decodePTRAPcc(data []byte, inst *Instruction, immSize int) error {
	if err := requireLength(data, 4, "PTRAPcc"); err != nil {
		return err
	}
	word2 := binary.BigEndian.Uint16(data[2:4])
	mnemonic := "PTRAP" + pmmuConditionName(word2&0xF)

	switch immSize {
	case 0:
		setInstruction(data, inst, 4, mnemonic, "")
	case 2:
		if err := requireLength(data, 6, mnemonic+".W operand"); err != nil {
			return err
		}
		imm := uint32(binary.BigEndian.Uint16(data[4:6]))
		immText := fmt.Sprintf("#%s", formatImmediate(imm, 2))
		setInstruction(data, inst, 6, mnemonic+".W", immText, immediateOperand(immText, imm, 2))
	case 4:
		if err := requireLength(data, 8, mnemonic+".L operand"); err != nil {
			return err
		}
		imm := binary.BigEndian.Uint32(data[4:8])
		immText := fmt.Sprintf("#%s", formatImmediate(imm, 4))
		setInstruction(data, inst, 8, mnemonic+".L", immText, immediateOperand(immText, imm, 4))
	}
	return nil
}
