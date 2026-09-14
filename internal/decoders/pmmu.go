package decoders

import (
	"encoding/binary"
	"fmt"
)

// This file decodes the 68851/68030 PMMU (paged memory management unit)
// coprocessor instruction set — the F-line CpId-0 coprocessor space,
// distinct from the FPU's CpId-1 space decoded in fpu.go (compare
// fpuWord1Base's 0xF200 there with this file's own bare 0xF000: PMMU's
// word1 has no coprocessor-ID bit to fold in at all). Covers PMOVE (every
// 68851/68030 register PMOVE can move, except BAD0-7/BAC0-7 — a genuinely
// different, numbered-register-family shape, left for a follow-up),
// PMOVEFD, PFLUSHA, PFLUSH, PFLUSHS, PFLUSHR, PLOADR, PLOADW, PTESTR, and
// PTESTW.
//
// Every one of these shares PMMU's bare "0xF000 | <ea>" word1 — word2
// alone selects which mnemonic, matching FPU's own approach of dispatching
// several instruction families out of one opcode-table pattern (compare
// decodeFPGeneric's word2-based dispatch in fpu.go).
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

// PFLUSH/PFLUSHS word2 shapes: top 6 bits (mask 0xFC00) select one of four
// variants — PFLUSH or PFLUSHS (extraBit 0x0400 apart), each either
// "FC,#mask" (no <ea>, word1 stays 0xF000 exactly) or "FC,#mask,<ea>"
// (word1 carries the <ea>, extraBit 0x0800 apart from the no-<ea> form).
// The mask (5 bits, 0-31) sits at bits 9-5; the function-code specifier
// (see fcSpecOperand) at bits 4-0. Cross-checked against m68kasm's
// newPFlushDef (cpu030_pmmu_ptest.go).
const (
	pflushClassMask = 0xFC00
	pflushBase      = 0x3000 // PFLUSH,  no <ea>
	pflushsBase     = 0x3400 // PFLUSHS, no <ea>
	pflushEABase    = 0x3800 // PFLUSH,  with <ea>
	pflushsEABase   = 0x3C00 // PFLUSHS, with <ea>
	pflushrWord2    = 0xA000 // PFLUSHR: fully-fixed word2, plain memory <ea>
)

// PLOADR/PLOADW word2: base literal (bit 9 the R/W direction — matching
// PTESTR/PTESTW's own R/W bit below) | the function-code specifier at
// bits 4-0. Cross-checked against m68kasm's newPLoadDef.
const (
	ploadMask  = 0xFFE0
	ploadrBase = 0x2200
	ploadwBase = 0x2000
)

// PTESTR/PTESTW word2: base literal (bit 9 = R/W direction) | function-code
// specifier (bits 4-0) | #level (bits 12-10) | optional result register An
// (bits 7-5). Bits 15-13 and 9-8 are the only bits none of those three
// fields ever touch (An's value 0-7 only reaches bits 7-5, never 9-8), so
// they're what identifies this word2 shape regardless of level/An/FC
// values — mask 0xE300. Cross-checked against m68kasm's newPTestDef.
const (
	ptestClassMask = 0xE300
	ptestrBase     = 0x8200
	ptestwBase     = 0x8000
)

// BAD0-BAD7/BAC0-BAC7 (breakpoint address/access registers) are the last
// PMOVE-movable registers, and genuinely different in shape from every
// other one: a register NUMBER (0-7) at bits 4-2 rather than a fixed
// selector, AND an INVERTED load/store direction bit relative to every
// other PMOVE register — bit 9 set means LOAD here (every other register's
// word2, per pmmuMoveWord2 above, uses 0x0200/bit9 for STORE). Confirmed
// against m68kasm's own newPmmuNumberedReg literals (cpu030_pmmu_badbac.go)
// before coding, given the inverted-bit convention is exactly the kind of
// easy-to-transpose detail that caused real bugs earlier in this sequence
// (see docs/design-fpu-mmu.md's FMOVEM/FMOVE.P entries).
//
//	BADn load  = 0x7200 | (n<<2)   BADn store = 0x7000 | (n<<2)
//	BACn load  = 0x7600 | (n<<2)   BACn store = 0x7400 | (n<<2)
const (
	badBacFamilyMask = 0xF000
	badBacFamilyBase = 0x7000
	badBacIsBAC      = 0x0400 // bit 10: 0 = BAD, 1 = BAC
	badBacIsLoad     = 0x0200 // bit 9: 1 = load (<ea>,REGn) — inverted, see above
)

// decodePMMUGeneral decodes every PMMU instruction sharing the bare
// "0xF000 | <ea>" word1 shape: PMOVE, PMOVEFD, PFLUSHA, PFLUSH, PFLUSHS,
// PFLUSHR, PLOADR, PLOADW, PTESTR, and PTESTW.
func decodePMMUGeneral(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	if err := requireLength(data, 4, "PMMU coprocessor command word"); err != nil {
		return err
	}
	word2 := binary.BigEndian.Uint16(data[2:4])

	switch {
	case word2 == pmmuFlushAWord2:
		setInstruction(data, inst, 4, "PFLUSHA", "")
		return nil
	case word2 == pflushrWord2:
		return decodePFLUSHR(data, opcode, inst, cpu)
	case word2&pflushClassMask == pflushBase:
		return decodePFLUSH(data, opcode, inst, cpu, "PFLUSH", word2, false)
	case word2&pflushClassMask == pflushsBase:
		return decodePFLUSH(data, opcode, inst, cpu, "PFLUSHS", word2, false)
	case word2&pflushClassMask == pflushEABase:
		return decodePFLUSH(data, opcode, inst, cpu, "PFLUSH", word2, true)
	case word2&pflushClassMask == pflushsEABase:
		return decodePFLUSH(data, opcode, inst, cpu, "PFLUSHS", word2, true)
	case word2&ploadMask == ploadrBase:
		return decodePLOAD(data, opcode, inst, cpu, "PLOADR", word2)
	case word2&ploadMask == ploadwBase:
		return decodePLOAD(data, opcode, inst, cpu, "PLOADW", word2)
	case word2&ptestClassMask == ptestrBase:
		return decodePTEST(data, opcode, inst, cpu, "PTESTR", word2)
	case word2&ptestClassMask == ptestwBase:
		return decodePTEST(data, opcode, inst, cpu, "PTESTW", word2)
	case word2&badBacFamilyMask == badBacFamilyBase:
		return decodeBADBAC(data, opcode, inst, cpu, word2)
	}

	info, ok := pmmuMoveWord2[word2]
	if !ok {
		return fmt.Errorf("unknown PMMU coprocessor command word: $%04X", word2)
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

// fcSpecOperand decodes PFLUSH/PLOAD/PTEST's "function code specifier"
// operand (bits 4-0 of word2): bit 4 set = immediate (bits 2-0, 0-7); bit 3
// set = Dn register (bits 2-0); neither set = named SFC(0)/DFC(1). Cross-
// checked against m68kasm's FFCSpecWord (encode.go).
func fcSpecOperand(v uint16) Operand {
	switch {
	case v&0x10 != 0:
		val := uint32(v & 0x7)
		text := fmt.Sprintf("#%d", val)
		return immediateOperand(text, val, 1)
	case v&0x08 != 0:
		return registerOperand(RegisterKindData, uint8(v&0x7))
	default:
		switch v & 0x7 {
		case 0:
			return Operand{Text: "SFC", Kind: OperandKindRegister}
		case 1:
			return Operand{Text: "DFC", Kind: OperandKindRegister}
		default:
			return Operand{Text: fmt.Sprintf("FC%d", v&0x7), Kind: OperandKindRegister}
		}
	}
}

func decodePFLUSH(data []byte, opcode uint16, inst *Instruction, cpu CPU, mnemonic string, word2 uint16, withEA bool) error {
	fc := fcSpecOperand(word2 & 0x1F)
	mask := uint32((word2 >> 5) & 0x1F)
	maskText := fmt.Sprintf("#%d", mask)
	maskMeta := immediateOperand(maskText, mask, 1)

	if !withEA {
		setInstruction(data, inst, 4, mnemonic, fc.Text+", "+maskText, fc, maskMeta)
		return nil
	}

	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	eaText, offset, eaMeta, err := decodeEA(data, inst.Address, 4, mode, reg, cpu)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, mnemonic, fc.Text+", "+maskText+", "+eaText, fc, maskMeta, eaMeta)
	return nil
}

func decodePFLUSHR(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	eaText, offset, eaMeta, err := decodeEA(data, inst.Address, 4, mode, reg, cpu)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, "PFLUSHR", eaText, eaMeta)
	return nil
}

func decodePLOAD(data []byte, opcode uint16, inst *Instruction, cpu CPU, mnemonic string, word2 uint16) error {
	fc := fcSpecOperand(word2 & 0x1F)
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	eaText, offset, eaMeta, err := decodeEA(data, inst.Address, 4, mode, reg, cpu)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, mnemonic, fc.Text+", "+eaText, fc, eaMeta)
	return nil
}

// decodePTEST renders PTESTR/PTESTW's optional trailing result-register
// operand (bits 7-5) only when nonzero. Real hardware has no separate
// "An present" flag distinct from the register value itself, so "...,
// #level" and "...,#level,A0" are bit-identical — an inherent, not
// decoder-side, ambiguity (m68kasm's own encoder produces the same bytes
// for both source spellings). Showing the trailing operand only when it
// differs from the always-valid A0 default is this decoder's one
// necessary choice between the two equally-valid textual spellings.
func decodePTEST(data []byte, opcode uint16, inst *Instruction, cpu CPU, mnemonic string, word2 uint16) error {
	fc := fcSpecOperand(word2 & 0x1F)
	level := uint32((word2 >> 10) & 0x7)
	levelText := fmt.Sprintf("#%d", level)
	levelMeta := immediateOperand(levelText, level, 1)

	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	eaText, offset, eaMeta, err := decodeEA(data, inst.Address, 4, mode, reg, cpu)
	if err != nil {
		return err
	}

	anReg := uint8((word2 >> 5) & 0x7)
	if anReg == 0 {
		operands := fc.Text + ", " + eaText + ", " + levelText
		setInstruction(data, inst, offset, mnemonic, operands, fc, eaMeta, levelMeta)
		return nil
	}

	anMeta := registerOperand(RegisterKindAddress, anReg)
	operands := fc.Text + ", " + eaText + ", " + levelText + ", " + anMeta.Text
	setInstruction(data, inst, offset, mnemonic, operands, fc, eaMeta, levelMeta, anMeta)
	return nil
}

// decodeBADBAC decodes PMOVE's BAD0-BAD7/BAC0-BAC7 forms — see the
// badBacFamilyMask const block above for the bit layout.
func decodeBADBAC(data []byte, opcode uint16, inst *Instruction, cpu CPU, word2 uint16) error {
	name := "BAD"
	if word2&badBacIsBAC != 0 {
		name = "BAC"
	}
	regNum := (word2 >> 2) & 0x7
	regMeta := Operand{Text: fmt.Sprintf("%s%d", name, regNum), Kind: OperandKindRegister}

	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	eaText, offset, eaMeta, err := decodeEAWithSize(data, inst.Address, 4, mode, reg, 4, cpu)
	if err != nil {
		return err
	}

	if word2&badBacIsLoad != 0 {
		setInstruction(data, inst, offset, "PMOVE.L", eaText+", "+regMeta.Text, eaMeta, regMeta)
		return nil
	}
	setInstruction(data, inst, offset, "PMOVE.L", regMeta.Text+", "+eaText, regMeta, eaMeta)
	return nil
}

// decodePSAVE and decodePRESTORE decode the PMMU coprocessor state-frame
// save/restore instructions — structurally identical to fpu.go's
// decodeFSaveRestore (FSAVE/FRESTORE), except PMMU's word1 needs no
// coprocessor-ID adjustment (0xF100/0xF140 are used exactly as GAS's own
// table lists them, unlike FPU's 0xF100/0xF140-that-become-0xF300/0xF340).
// As with FSAVE/FRESTORE, frame contents are out of scope — only the
// instruction shell (mnemonic + <ea>) is decoded, per
// docs/design-fpu-mmu.md's Non-goals.
func decodePSAVE(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodePMMUSaveRestore(data, opcode, inst, cpu, "PSAVE")
}

func decodePRESTORE(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodePMMUSaveRestore(data, opcode, inst, cpu, "PRESTORE")
}

func decodePMMUSaveRestore(data []byte, opcode uint16, inst *Instruction, cpu CPU, mnemonic string) error {
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	eaText, offset, eaMeta, err := decodeEA(data, inst.Address, 2, mode, reg, cpu)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, mnemonic, eaText, eaMeta)
	return nil
}
