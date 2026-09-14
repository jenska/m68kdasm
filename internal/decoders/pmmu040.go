package decoders

import "fmt"

// This file decodes the 68040's own PMMU instructions — a simplified,
// re-encoded interface distinct from the 68030/68851 two-word coprocessor
// forms decoded in pmmu.go: every instruction here is a single fixed
// 16-bit word, not a 0xF000-prefixed word1+word2 pair, and each one's sole
// operand (when it has one) is always a plain address register — no <ea>
// mode field, no coprocessor command word, no function-code specifier.
//
// PFLUSHA/PFLUSHAN/PFLUSHN/PFLUSH share a mnemonic with their existing
// 68030/68851 two-word counterparts in pmmu.go (decodePMMUGeneral/
// decodePFLUSH) — real 68040 binaries can use either encoding, and since
// the two forms' word1 values never overlap (0xF000-prefixed vs.
// 0xF500-0xF56F here), both opcode-table patterns just coexist; whichever
// bytes actually appear determines which one matches.
//
// Cross-checked against m68kasm v1.5.0's encoder
// (internal/asm/instructions/cpu040_pmmu.go), itself cross-checked there
// against GNU binutils' opcodes/m68k-opc.c:
//
//	{"pflusha",  one(0xf518), one(0xfff8), "",   m68040up },
//	{"pflushan", one(0xf510), one(0xfff8), "",   m68040up },
//	{"pflushn",  one(0xf500), one(0xfff8), "as", m68040up },
//	{"pflush",   one(0xf508), one(0xfff8), "as", m68040up },
//	{"ptestr",   one(0xf568), one(0xfff8), "as", m68040 },
//	{"ptestw",   one(0xf548), one(0xfff8), "as", m68040 },

func decodePFLUSHA040(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodePMMU040NoOperand(data, inst, "PFLUSHA")
}

func decodePFLUSHAN040(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodePMMU040NoOperand(data, inst, "PFLUSHAN")
}

func decodePFLUSHN040(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodePMMU040AnOnly(data, opcode, inst, "PFLUSHN")
}

func decodePFLUSH040(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodePMMU040AnOnly(data, opcode, inst, "PFLUSH")
}

func decodePTESTR040(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodePMMU040AnOnly(data, opcode, inst, "PTESTR")
}

func decodePTESTW040(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodePMMU040AnOnly(data, opcode, inst, "PTESTW")
}

// decodePMMU040AnOnly decodes the shared shape of every 68040 PMMU
// instruction that takes an operand: a plain address register, encoded
// directly in bits 2-0 with no addressing-mode field at all (m68kasm's
// FSrcRegOnly — the assembler accepts both "(An)" and bare "An" as input
// spellings for this operand, since the register just names a memory
// address either way; rendered here as "(An)", the indirect-addressing
// spelling, since the operand's meaning is always "the address in An").
func decodePMMU040AnOnly(data []byte, opcode uint16, inst *Instruction, mnemonic string) error {
	reg := uint8(opcode & 0x7)
	text := fmt.Sprintf("(A%d)", reg)
	setInstruction(data, inst, 2, mnemonic, text, addrIndirectOperand(EAKindAddressIndirect, reg, text))
	return nil
}

// decodePMMU040NoOperand decodes the shared shape of PFLUSHA/PFLUSHAN: no
// operand at all, just the fixed opcode word.
func decodePMMU040NoOperand(data []byte, inst *Instruction, mnemonic string) error {
	setInstruction(data, inst, 2, mnemonic, "")
	return nil
}
