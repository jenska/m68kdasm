package decoders

import (
	"encoding/binary"
	"fmt"
)

// This file decodes the 68851/68030 PMMU (paged memory management unit)
// coprocessor instruction set — the F-line CpId-0 coprocessor space,
// distinct from the FPU's CpId-1 space decoded in fpu.go (compare
// fpuWord1Base's 0xF200 there with this file's own bare 0xF000: PMMU's
// word1 has no coprocessor-ID bit to fold in at all). This first slice
// covers PMOVE (every 68851/68030 register PMOVE can move, except
// BAD0-7/BAC0-7 — a genuinely different, numbered-register-family shape,
// left for a follow-up), PMOVEFD, and PFLUSHA.
//
// Every bit position below was read from m68kasm v1.5.0's verified encoder
// (internal/asm/instructions/cpu030_pmmu.go, cpu030_pmmu_access.go,
// cpu030_pmmu_xlate.go, cpu030_pmmu_pmovefd.go), the same "read the real
// encoder, don't recall from memory" practice used throughout fpu.go — see
// docs/design-fpu-mmu.md.
//
// PMOVE/PMOVEFD word1 is 0xF000 | <ea> (mode/reg in bits 5-0), identical
// for both mnemonics — word2 alone determines which mnemonic, which
// register, which direction (PMOVE only; PMOVEFD is load-only), and the
// operand size. Unlike FPU's word2 layouts, none of these registers share
// a uniform bit-field scheme (TC/DRP/SRP/CRP/CAL/VAL/SCC/AC follow a
// "0x4000|selector<<10" pattern for load, "0x4200|selector<<10" for store,
// but TT0/TT1/MMUSR/PCSR each have their own unrelated base literal), so —
// mirroring m68kasm's own newPmmuFixedReg, which takes explicit word2
// literals per register rather than deriving them from one formula — this
// decoder uses a flat word2-to-register lookup table rather than bit-field
// extraction.
type pmmuMoveReg struct {
	mnemonic string // "PMOVE" or "PMOVEFD"
	reg      string
	size     int
	sizeStr  string
	isStore  bool
}

var pmmuMoveWord2 = map[uint16]pmmuMoveReg{
	// PMOVE: TC/DRP/SRP/CRP (Long), CAL/VAL/SCC (Byte), AC (Word),
	// TT0/TT1 (Long), MMUSR (Word) — load word2 | 0x0200 = store word2.
	0x4000: {"PMOVE", "TC", 4, "L", false},
	0x4200: {"PMOVE", "TC", 4, "L", true},
	0x4400: {"PMOVE", "DRP", 4, "L", false},
	0x4600: {"PMOVE", "DRP", 4, "L", true},
	0x4800: {"PMOVE", "SRP", 4, "L", false},
	0x4A00: {"PMOVE", "SRP", 4, "L", true},
	0x4C00: {"PMOVE", "CRP", 4, "L", false},
	0x4E00: {"PMOVE", "CRP", 4, "L", true},
	0x5000: {"PMOVE", "CAL", 1, "B", false},
	0x5200: {"PMOVE", "CAL", 1, "B", true},
	0x5400: {"PMOVE", "VAL", 1, "B", false},
	0x5600: {"PMOVE", "VAL", 1, "B", true},
	0x5800: {"PMOVE", "SCC", 1, "B", false},
	0x5A00: {"PMOVE", "SCC", 1, "B", true},
	0x5C00: {"PMOVE", "AC", 2, "W", false},
	0x5E00: {"PMOVE", "AC", 2, "W", true},
	// PCSR is store-only — GAS's own opcode table has no load-direction row.
	0x6600: {"PMOVE", "PCSR", 2, "W", true},
	0x0800: {"PMOVE", "TT0", 4, "L", false},
	0x0A00: {"PMOVE", "TT0", 4, "L", true},
	0x0C00: {"PMOVE", "TT1", 4, "L", false},
	0x0E00: {"PMOVE", "TT1", 4, "L", true},
	0x6000: {"PMOVE", "MMUSR", 2, "W", false},
	0x6200: {"PMOVE", "MMUSR", 2, "W", true},

	// PMOVEFD ("Function code lookup Disabled"): load-only, each word2 the
	// matching PMOVE load literal plus 0x0100 — only the registers whose
	// load affects how function-code lookups happen have a PMOVEFD form
	// (no CAL/VAL/SCC/AC/MMUSR/PCSR counterpart exists).
	0x4100: {"PMOVEFD", "TC", 4, "L", false},
	0x4500: {"PMOVEFD", "DRP", 4, "L", false},
	0x4900: {"PMOVEFD", "SRP", 4, "L", false},
	0x4D00: {"PMOVEFD", "CRP", 4, "L", false},
	0x0900: {"PMOVEFD", "TT0", 4, "L", false},
	0x0D00: {"PMOVEFD", "TT1", 4, "L", false},
}

// pmmuFlushAWord2 is PFLUSHA's word2 — a fixed, no-operand literal that
// happens to share PMOVE/PMOVEFD's word1 shape (0xF000, i.e. an empty
// <ea> field), the same way FNOP shares the FPU general instruction
// family's word1 shape. Checked before the pmmuMoveWord2 table lookup.
const pmmuFlushAWord2 = 0x2400

// decodePMOVEFamily decodes PMOVE, PMOVEFD, and PFLUSHA — every
// instruction sharing PMMU's bare "0xF000 | <ea>" word1 shape.
func decodePMOVEFamily(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	if err := requireLength(data, 4, "PMOVE/PMOVEFD/PFLUSHA extension word"); err != nil {
		return err
	}
	word2 := binary.BigEndian.Uint16(data[2:4])

	if word2 == pmmuFlushAWord2 {
		setInstruction(data, inst, 4, "PFLUSHA", "")
		return nil
	}

	info, ok := pmmuMoveWord2[word2]
	if !ok {
		return fmt.Errorf("unknown PMOVE/PMOVEFD coprocessor command word: $%04X", word2)
	}

	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	eaText, offset, eaMeta, err := decodeEAWithSize(data, inst.Address, 4, mode, reg, info.size, cpu)
	if err != nil {
		return err
	}

	regMeta := Operand{Text: info.reg, Kind: OperandKindRegister}
	mnemonic := info.mnemonic + "." + info.sizeStr
	if info.isStore {
		setInstruction(data, inst, offset, mnemonic, regMeta.Text+", "+eaText, regMeta, eaMeta)
		return nil
	}
	setInstruction(data, inst, offset, mnemonic, eaText+", "+regMeta.Text, eaMeta, regMeta)
	return nil
}
