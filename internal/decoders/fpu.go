package decoders

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// This file decodes the complete FPU instruction set covered by
// docs/design-fpu-mmu.md's delivery sequence (steps 3-8.5), deliberately
// matching the scope of github.com/jenska/m68kasm's own equivalent
// milestones (internal/asm/instructions/cpu020_fpu.go, cpu020_fpu_movem.go,
// cpu020_fpu_movem_ctrl.go, cpu020_fpu_cond.go, cpu020_fpu_trans.go,
// cpu020_fpu_mathext.go, cpu020_fpu_sincos.go, cpu020_fpu_packed.go):
// FMOVE, FADD, FSUB, FMUL, FDIV, FCMP, FABS, FNEG, FSQRT, FTST, FNOP,
// FMOVEM (both the FP0-FP7 register-list form and the FPCR/FPSR/FPIAR
// control-register-list form), the full transcendental function set (FSIN,
// FCOS, FLOGN, ...), the math-extensions bucket (FGETEXP, FGETMAN, FSCALE,
// FMOD, FREM), FMOVECR (ROM constant load), FSINCOS (the one FPU
// instruction with two destination registers), the FPU's own conditional
// branch/set/trap family (FBcc/FDBcc/FScc/FTRAPcc, a distinct 32-condition
// space from the integer ISA's 16), FSAVE/FRESTORE (coprocessor
// state-frame save/restore — the instruction shell only, not frame
// contents, see decodeFSAVE's own doc comment), and all 7 FPU data formats
// including packed BCD's k-factor-driven store direction. Not implemented
// here (out of scope for this doc's FPU sequence — see PMMU steps 9-10):
// the 68851/68030 PMMU coprocessor.
//
// Every bit position below was read from m68kasm v1.5.0's verified encoder
// (internal/asm/encode.go's applyField/fpFormatCode/fpRMBit), itself
// cross-checked there against GNU binutils' opcodes/m68k-opc.c — not
// recalled from memory, per this project's own established practice (see
// docs/design-cpu-variants.md's "Explicitly deferred" section) of treating
// dense FPU/coprocessor bit-packing as too error-prone to free-hand.
//
// Word1 (the opcode word already matched by the opcodeBuckets[0xF] pattern):
//
//	1111 001 000 mmmrrr   (0xF200 | <ea>), mmm/rrr = <ea> mode/reg — used by
//	                       the <ea>-involving forms. For the pure
//	                       register-to-register form, mmm/rrr is unused and
//	                       always 0 (word1 == 0xF200 exactly); R/M in word2
//	                       is what actually distinguishes the two, not
//	                       word1's low bits.
//
// Word2 (the extension word, required for every form):
//
//	bit 14      R/M: 0 = register-to-register (bits 12-10 hold the source
//	            FPm register number), 1 = one operand is <ea> (bits 12-10
//	            hold that operand's data-format code instead).
//	bit 13      direction, meaningful only when R/M=1 and the opmode is
//	            FMOVE's (0x00 — the only opmode with a store direction):
//	            0 = load (<ea>,FPn), 1 = store (FPn,<ea>).
//	bits 12-10  format code (R/M=1) or source FPm register (R/M=0).
//	bits 9-7    destination FPn register (or, in the store direction, the
//	            source FPn register being written to <ea>).
//	bits 6-0    opmode, selecting FMOVE/FADD/.../FTST.
const (
	fpRMBit  = 0x4000
	fpDirBit = 0x2000 // FMOVE only: 0 = load <ea>,FPn, 1 = store FPn,<ea>
	fpOpMask = 0x007F
)

// fpFormatInfo describes one of the FPU's 7 <ea> data formats (word2 bits
// 12-10). Cross-checked against m68kasm's fpFormatCode (internal/asm/
// encode.go), itself decoded from GAS's per-size FADD opcode literals
// (faddl/fadds/faddx/faddp/faddw/faddd/faddb).
type fpFormatInfo struct {
	bytes  int
	suffix string
}

var fpFormats = [8]fpFormatInfo{
	0: {4, "L"},  // Long
	1: {4, "S"},  // Single
	2: {12, "X"}, // Extended
	3: {12, "P"}, // Packed BCD
	4: {2, "W"},  // Word
	5: {8, "D"},  // Double
	6: {1, "B"},  // Byte
	7: {0, ""},   // reserved
}

// fpOpInfo describes one word2 opmode (bits 6-0).
type fpOpInfo struct {
	mnemonic string
	hasDst   bool // false only for FTST: its <ea>/register form has no destination FPn
	canStore bool // true only for FMOVE: the only opmode with a FPn-to-<ea> store direction
}

// fpGeneralOps maps word2's opmode to the instruction it selects.
// Cross-checked against m68kasm's cpu020_fpu.go registerInstrDef calls
// (newFPBinaryDef/newFPMonadicDef's opBase arguments).
var fpGeneralOps = map[uint16]fpOpInfo{
	0x00: {"FMOVE", true, true},
	0x22: {"FADD", true, false},
	0x28: {"FSUB", true, false},
	0x23: {"FMUL", true, false},
	0x20: {"FDIV", true, false},
	0x38: {"FCMP", true, false},
	0x18: {"FABS", true, false},
	0x1A: {"FNEG", true, false},
	0x04: {"FSQRT", true, false},
	0x3A: {"FTST", false, false},

	// Transcendental function set — a discrete 68881/68882 executes these
	// natively; a 68040/68060's integrated FPU traps and emulates them in
	// software, which is not disassembler-visible (the opcode and its
	// decode are identical either way), so these are gated by the same
	// DecodeOptions.FPU flag as everything else above rather than a
	// separate "full FPU" capability — see m68kasm's requireFPUFull for
	// the encoder-side distinction this project deliberately doesn't
	// mirror on the decode side. Every one of these is the same monadic
	// "<ea>,FPn"/"FPm,FPn" shape FABS/FNEG/FSQRT already use (hasDst:
	// true, canStore: false). opBase values copied verbatim from
	// m68kasm's cpu020_fpu_trans.go, itself decoded from GAS's m68k-opc.c.
	0x0E: {"FSIN", true, false},
	0x1D: {"FCOS", true, false},
	0x0F: {"FTAN", true, false},
	0x0A: {"FATAN", true, false},
	0x0C: {"FASIN", true, false},
	0x1C: {"FACOS", true, false},
	0x0D: {"FATANH", true, false},
	0x02: {"FSINH", true, false},
	0x19: {"FCOSH", true, false},
	0x09: {"FTANH", true, false},
	0x10: {"FETOX", true, false},
	0x08: {"FETOXM1", true, false},
	0x14: {"FLOGN", true, false},
	0x06: {"FLOGNP1", true, false},
	0x15: {"FLOG10", true, false},
	0x16: {"FLOG2", true, false},
	0x11: {"FTWOTOX", true, false},
	0x12: {"FTENTOX", true, false},

	// "Math extensions" — FGETEXP/FGETMAN are monadic (same shape as the
	// transcendental set above); FSCALE/FMOD/FREM are genuinely binary
	// (two FPn operands, e.g. "FSCALE FPm,FPn" scales FPn by FPm — the
	// same {hasDst: true, canStore: false} shape FADD/FSUB/etc. already
	// use, not FABS's). Also gated by DecodeOptions.FPU rather than a
	// separate "full FPU" flag, same reasoning as the transcendental set.
	// opBase values copied verbatim from m68kasm's cpu020_fpu_mathext.go.
	0x1E: {"FGETEXP", true, false},
	0x1F: {"FGETMAN", true, false},
	0x26: {"FSCALE", true, false},
	0x21: {"FMOD", true, false},
	0x25: {"FREM", true, false},
}

// fpConditions names the FPU's 32 condition codes, indexed by condition
// value (0-31) — a distinct, wider space from the integer ISA's 16
// (branchCondNames/standardCondNames in branch.go). One table serves all
// four condition-driven FPU families (FBcc/FDBcc/FScc/FTRAPcc): each
// mnemonic is this suffix prefixed with the family's own letter(s).
// Copied verbatim from m68kasm's fpConditions (cpu020_fpu_cond.go), itself
// decoded from GAS's m68k opcode table.
var fpConditions = [...]string{
	"F", "EQ", "OGT", "OGE", "OLT", "OLE", "OGL", "OR",
	"UN", "UEQ", "UGT", "UGE", "ULT", "ULE", "NE", "T",
	"SF", "SEQ", "GT", "GE", "LT", "LE", "GL", "GLE",
	"NGLE", "NGL", "NLE", "NLT", "NGE", "NGT", "SNE", "ST",
}

func fpConditionName(cc uint16) string {
	if int(cc) < len(fpConditions) {
		return fpConditions[cc]
	}
	return fmt.Sprintf("?%d", cc)
}

// decodeFBcc decodes both FBcc forms — word displacement (word1 =
// 0xF280|cc) and long displacement (word1 = 0xF2C0|cc, distinguished from
// the word form by bit 6, the one bit that differs between the two base
// literals) — using branchTarget (branch.go) for the target address, same
// as every other PC-relative branch family.
//
// FBcc never inlines an 8-bit displacement in the opcode itself (unlike
// integer Bcc) — every form needs a full extension word — so cc==0
// ("F", branch never) with a zero word displacement is bit-for-bit
// identical to what real hardware and every 68k assembler spell "FNOP",
// and is rendered that way here rather than as "FBF.W $addr".
func decodeFBcc(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	cc := opcode & 0x1F
	mnemonic := "FB" + fpConditionName(cc)
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
	if cc == 0 && disp == 0 {
		setInstruction(data, inst, 4, "FNOP", "")
		return nil
	}
	target := branchTarget(inst.Address, disp)
	targetText := formatBranchTarget(target)
	setInstruction(data, inst, 4, mnemonic+".W", targetText, branchOperand(targetText, target))
	return nil
}

// decodeFDBcc mirrors decodeDBcc (branch.go), except the condition lives in
// a dedicated, fully-fixed word2 (0x0000-0x001F) rather than in the opcode
// word's own bits — word1 only carries the Dn register (bits 2-0).
func decodeFDBcc(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	if err := requireLength(data, 6, "FDBcc"); err != nil {
		return err
	}
	word2 := binary.BigEndian.Uint16(data[2:4])
	cc := word2 & 0x1F
	mnemonic := "FDB" + fpConditionName(cc)
	reg := uint8(opcode & 0x7)
	disp := int32(int16(binary.BigEndian.Uint16(data[4:6])))
	target := branchTarget(inst.Address, disp)
	targetText := formatBranchTarget(target)
	regMeta := registerOperand(RegisterKindData, reg)
	setInstruction(data, inst, 6, mnemonic, regMeta.Text+", "+targetText, regMeta, branchOperand(targetText, target))
	return nil
}

// decodeFScc mirrors decodeScc (branch.go): word1 = 0xF240|<ea> (data-
// alterable, byte-sized destination), word2 = the condition, fully fixed
// like FDBcc's. FDBcc and FTRAPcc's exact opcodes both occupy part of this
// pattern's own EA range (address-register-direct and mode-7/reg-2..4
// respectively) and so must precede this pattern in the opcode table — see
// their registrations in opcodetable.go, mirroring integer DBcc/TRAPcc's
// identical precedence requirement ahead of Scc.
func decodeFScc(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	if err := requireLength(data, 4, "FScc"); err != nil {
		return err
	}
	word2 := binary.BigEndian.Uint16(data[2:4])
	cc := word2 & 0x1F
	mnemonic := "FS" + fpConditionName(cc)
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	operand, offset, meta, err := decodeEAWithSize(data, inst.Address, 4, mode, reg, 1, cpu)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, mnemonic, operand, meta)
	return nil
}

// decodeFTRAPccBare/Word/Long mirror decodeTRAPcc's three-form shape
// (branch.go), but each form is a fully-fixed word1 literal (0xF27C/0xF27A/
// 0xF27B) rather than a shared mask with the operand selector in the
// opcode's low bits — the condition lives in word2, like FDBcc/FScc.
func decodeFTRAPccBare(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeFTRAPcc(data, inst, 0)
}

func decodeFTRAPccWord(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeFTRAPcc(data, inst, 2)
}

func decodeFTRAPccLong(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeFTRAPcc(data, inst, 4)
}

func decodeFTRAPcc(data []byte, inst *Instruction, immSize int) error {
	if err := requireLength(data, 4, "FTRAPcc"); err != nil {
		return err
	}
	word2 := binary.BigEndian.Uint16(data[2:4])
	mnemonic := "FTRAP" + fpConditionName(word2&0x1F)

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

// decodeFPGeneric decodes the FPU general-instruction family — see the
// bit-layout notes in this file's header comment.
func decodeFPGeneric(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	if err := requireLength(data, 4, "FPU generic instruction extension word"); err != nil {
		return err
	}
	word2 := binary.BigEndian.Uint16(data[2:4])

	// word2 bit 15 is always 0 for the FADD/FMOVE/.../FTST family decoded
	// below (confirmed from m68kasm's applyField: none of FFPFormat/
	// FFPDstReg7/FFPSrcReg7/FFPSrcReg10 ever set it) and always 1 for
	// FMOVEM's word2 literals (0xD000/0xD800/0xE000/0xF000/0xF800) — real
	// hardware uses word2's high bits to select which "coprocessor command
	// word" shape applies, since word1 (0xF200 | <ea>) is identical for
	// both instruction families.
	if word2&0x8000 != 0 {
		return decodeFMOVEM(data, opcode, inst, cpu, word2)
	}

	// FMOVECR's word2 (0x5C00 | 7-bit ROM index | dst FPn<<7) sets R/M
	// (bit14) with format-code bits 12-10 = 111 — a value fpFormats marks
	// reserved for every other instruction, since real hardware format
	// codes only span 0-6. That makes FMOVECR's top-6-bits pattern
	// (0xFC00 mask, 0x5C00 value) uniquely identifiable before the format
	// code is ever interpreted as one. Cross-checked against m68kasm's
	// defFMOVECR (cpu020_fpu_cond.go).
	if word2&0xFC00 == 0x5C00 {
		return decodeFMOVECR(data, inst, word2)
	}

	// FSINCOS's opmode literal is 0x30, but unlike every fpGeneralOps
	// entry, bits 2-0 of that literal aren't fixed opmode bits — they're
	// the cosine result register (FSincosRegCos0 in m68kasm), so the full
	// 7-bit opmode field varies (0x30-0x37) depending on which FPn holds
	// the result. Only bits 6-3 (mask 0x78) are the actual fixed selector;
	// checking the full fpOpMask here would miss every case where the
	// cosine register's low bits happen to be nonzero. Its word2 also
	// packs a second destination register that fpOpInfo's single-dst model
	// has no field for, so this needs its own decode path rather than a
	// fpGeneralOps table entry either way. Cross-checked against
	// m68kasm's defFSINCOS (cpu020_fpu_sincos.go).
	if word2&0x78 == 0x30 {
		return decodeFSINCOS(data, opcode, inst, cpu, word2)
	}

	// FMOVE.P's store direction (FPn,<ea>{k}, packed BCD) is a genuinely
	// separate word2 shape, not a `.P`-suffixed row of the ordinary
	// FMOVE-store family: real hardware has no room for a format-code
	// field here (a store destination is always packed, implied by the
	// mnemonic), so bits 13-10 that would otherwise be FFPFormat's R/M+
	// format bits are repurposed for the k-factor instead. Detected via
	// bits 15,14,13,11,10 (mask 0xEC00, matching base value 0x6C00 for
	// both the static and dynamic k-factor sub-forms — bit 12, the only
	// difference between them, is deliberately excluded from the mask).
	// This must run before the generic fpGeneralOps dispatch below:
	// without it, word2&fpOpMask reads as 0x00 (FMOVE's own key) with
	// R/M=1 and the store-direction bit set, so decodeFPStore would
	// silently misdecode it as "FMOVE.P <ea>,FPn" — reading the k-factor
	// bits as if they were a destination FPn register — rather than
	// erroring or falling through, since format code 3 (Packed) is
	// otherwise a perfectly valid <ea> format for every OTHER store.
	// Cross-checked against m68kasm's cpu020_fpu_packed.go.
	if word2&0xEC00 == 0x6C00 {
		return decodeFMOVEPStore(data, opcode, inst, cpu, word2)
	}

	op, ok := fpGeneralOps[word2&fpOpMask]
	if !ok {
		return fmt.Errorf("unknown FPU opmode: $%02X", word2&fpOpMask)
	}

	if word2&fpRMBit == 0 {
		return decodeFPRegToReg(data, inst, op, word2)
	}
	if op.canStore && word2&fpDirBit != 0 {
		return decodeFPStore(data, opcode, inst, cpu, word2)
	}
	return decodeFPLoad(data, opcode, inst, cpu, op, word2)
}

func decodeFPRegToReg(data []byte, inst *Instruction, op fpOpInfo, word2 uint16) error {
	srcMeta := registerOperand(RegisterKindFP, uint8((word2>>10)&0x7))
	mnemonic := op.mnemonic + ".X"
	if !op.hasDst {
		setInstruction(data, inst, 4, mnemonic, srcMeta.Text, srcMeta)
		return nil
	}
	dstMeta := registerOperand(RegisterKindFP, uint8((word2>>7)&0x7))
	setInstruction(data, inst, 4, mnemonic, srcMeta.Text+", "+dstMeta.Text, srcMeta, dstMeta)
	return nil
}

func decodeFPLoad(data []byte, opcode uint16, inst *Instruction, cpu CPU, op fpOpInfo, word2 uint16) error {
	fmtCode := (word2 >> 10) & 0x7
	info := fpFormats[fmtCode]
	if info.bytes == 0 {
		return fmt.Errorf("reserved FPU data format code: %d", fmtCode)
	}

	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	srcOperand, offset, srcMeta, err := decodeFPEAWithSize(data, inst.Address, 4, mode, reg, fmtCode, cpu)
	if err != nil {
		return err
	}

	mnemonic := op.mnemonic + "." + info.suffix
	if !op.hasDst {
		setInstruction(data, inst, offset, mnemonic, srcOperand, srcMeta)
		return nil
	}
	dstMeta := registerOperand(RegisterKindFP, uint8((word2>>7)&0x7))
	setInstruction(data, inst, offset, mnemonic, srcOperand+", "+dstMeta.Text, srcMeta, dstMeta)
	return nil
}

// decodeFPStore decodes FMOVE's FPn,<ea> store direction — the only
// opmode/direction pair where the FPU register is the source and <ea> is
// the destination.
func decodeFPStore(data []byte, opcode uint16, inst *Instruction, cpu CPU, word2 uint16) error {
	fmtCode := (word2 >> 10) & 0x7
	info := fpFormats[fmtCode]
	if info.bytes == 0 {
		return fmt.Errorf("reserved FPU data format code: %d", fmtCode)
	}

	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	dstOperand, offset, dstMeta, err := decodeFPEAWithSize(data, inst.Address, 4, mode, reg, fmtCode, cpu)
	if err != nil {
		return err
	}

	srcMeta := registerOperand(RegisterKindFP, uint8((word2>>7)&0x7))
	mnemonic := "FMOVE." + info.suffix
	setInstruction(data, inst, offset, mnemonic, srcMeta.Text+", "+dstOperand, srcMeta, dstMeta)
	return nil
}

// decodeFMOVECR decodes "FMOVECR #<romIndex>,FPn": loads one of the FPU's
// built-in ROM constants (pi, e, log10(2), ...) into FPn. word1 carries no
// <ea> field (FMOVECR never reads memory); word2 is 0x5C00 | the 7-bit ROM
// index (bits 6-0) | the destination FPn (bits 9-7). Only the numeric
// "#<n>" form is decoded — GAS additionally accepts named aliases
// (fp_pi, fp_e, ...) for specific indices at assembly time, which have no
// decode-time meaning (the numeric index is what's actually encoded either
// way, and m68kasm itself only implements the numeric form — see
// cpu020_fpu_cond.go's own doc comment).
func decodeFMOVECR(data []byte, inst *Instruction, word2 uint16) error {
	romIndex := uint32(word2 & 0x7F)
	dstMeta := registerOperand(RegisterKindFP, uint8((word2>>7)&0x7))
	immText := fmt.Sprintf("#%s", formatImmediate(romIndex, 1))
	setInstruction(data, inst, 4, "FMOVECR", immText+", "+dstMeta.Text, immediateOperand(immText, romIndex, 1), dstMeta)
	return nil
}

// decodeFSINCOS decodes "FSINCOS <ea>,FPc:FPs" / "FSINCOS FPm,FPc:FPs":
// computes sine and cosine simultaneously from one input, writing two
// different FPn registers — the only FPU transcendental with two outputs.
// word2's bits 9-7 hold the sine result (FSincosRegSin7 — the same bit
// position every other instruction's single destination uses) and bits 2-0
// hold the cosine result (FSincosRegCos0). Motorola's own mnemonic syntax
// writes cosine first, "FPc:FPs" — cross-checked against m68kasm's
// defFSINCOS (cpu020_fpu_sincos.go).
func decodeFSINCOS(data []byte, opcode uint16, inst *Instruction, cpu CPU, word2 uint16) error {
	sinReg := uint8((word2 >> 7) & 0x7)
	cosReg := uint8(word2 & 0x7)
	dstList := []string{fmt.Sprintf("FP%d", cosReg), fmt.Sprintf("FP%d", sinReg)}
	dstText := dstList[0] + ":" + dstList[1]
	dstMeta := registerListOperand(dstText, dstList)

	if word2&fpRMBit == 0 {
		srcMeta := registerOperand(RegisterKindFP, uint8((word2>>10)&0x7))
		setInstruction(data, inst, 4, "FSINCOS.X", srcMeta.Text+", "+dstText, srcMeta, dstMeta)
		return nil
	}

	fmtCode := (word2 >> 10) & 0x7
	info := fpFormats[fmtCode]
	if info.bytes == 0 {
		return fmt.Errorf("reserved FPU data format code: %d", fmtCode)
	}
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	srcOperand, offset, srcMeta, err := decodeFPEAWithSize(data, inst.Address, 4, mode, reg, fmtCode, cpu)
	if err != nil {
		return err
	}
	mnemonic := "FSINCOS." + info.suffix
	setInstruction(data, inst, offset, mnemonic, srcOperand+", "+dstText, srcMeta, dstMeta)
	return nil
}

// decodeFMOVEPStore decodes "FMOVE.P FPn,<ea>{k}": stores FPn to memory as
// packed BCD, converting to a k-factor-controlled number of mantissa
// digits. word2 = 0x6C00 (static k-factor, a 7-bit two's-complement value
// at bits 6-0) or 0x7C00 (dynamic, bit 12 set and a Dn register number at
// bits 6-4 instead) | source FPn at bits 9-7. The k-factor renders as a
// "{...}" suffix directly appended to the destination <ea> text — GAS's
// own syntax, e.g. "FMOVE.P FP3,BUFFER{#-5}" or "...{D2}" — not a separate
// operand.
func decodeFMOVEPStore(data []byte, opcode uint16, inst *Instruction, cpu CPU, word2 uint16) error {
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	dstOperand, offset, dstMeta, err := decodeFPEAWithSize(data, inst.Address, 4, mode, reg, 3, cpu)
	if err != nil {
		return err
	}

	var kFactorText string
	if word2&0x1000 != 0 {
		kReg := uint8((word2 >> 4) & 0x7)
		kFactorText = fmt.Sprintf("{D%d}", kReg)
	} else {
		kVal := int32(word2 & 0x7F)
		if kVal >= 64 {
			kVal -= 128
		}
		kFactorText = fmt.Sprintf("{#%d}", kVal)
	}

	srcMeta := registerOperand(RegisterKindFP, uint8((word2>>7)&0x7))
	setInstruction(data, inst, offset, "FMOVE.P", srcMeta.Text+", "+dstOperand+kFactorText, srcMeta, dstMeta)
	return nil
}

// FMOVEM has two independent register-list forms sharing one mnemonic:
// FPn (FP0-FP7, an 8-bit mask) and the FPCR/FPSR/FPIAR control registers (a
// 3-bit mask). Word2's top 3 bits (mask 0xE000) select which of four
// classes applies — this is the only reliable dispatch key: a naive "top
// nibble" (bits 15-12) works for the FPn forms (their register/mask fields
// never touch bit 12) but NOT for the control-register forms, where the
// 3-bit mask sits at bits 12-10 and can legitimately set bit 12 itself
// (whenever FPCR, mask bit 2, is included) — e.g. "FMOVEM FPCR/FPSR/FPIAR,
// (A0)" encodes word2 as $BC00, whose naive top nibble ($B) looks nothing
// like the $A000 base literal it actually is. Cross-checked against
// m68kasm's word2 literals in cpu020_fpu_movem.go/cpu020_fpu_movem_ctrl.go.
const (
	fpMovemClassMask  = 0xE000
	fpMovemCtrlLoad   = 0x8000 // <ea>,FPCR/FPSR/FPIAR-list
	fpMovemCtrlStore  = 0xA000 // FPCR/FPSR/FPIAR-list,<ea>
	fpMovemFPnLoad    = 0xC000 // <ea>,FPn-list (word2 top nibble 0xD)
	fpMovemFPnStore   = 0xE000 // FPn-list,<ea> (word2 top nibble 0xE or 0xF)
	fpMovemPredecBit  = 0x1000 // FPn-store only: 0 = -(An) (reversed list), 1 = general memory
	fpMovemDynBit     = 0x0800 // FPn forms only: dynamic Dn-specified list instead of a static mask
	fpMovemCtrlSelect = 0x1C00 // control-register forms: 3-bit mask at bits 12-10
)

func decodeFMOVEM(data []byte, opcode uint16, inst *Instruction, cpu CPU, word2 uint16) error {
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)

	switch word2 & fpMovemClassMask {
	case fpMovemCtrlLoad:
		eaText, offset, eaMeta, err := decodeEA(data, inst.Address, 4, mode, reg, cpu)
		if err != nil {
			return err
		}
		listText, names := formatFPCtrlRegList(uint8((word2 & fpMovemCtrlSelect) >> 10))
		listMeta := registerListOperand(listText, names)
		setInstruction(data, inst, offset, "FMOVEM.L", eaText+", "+listText, eaMeta, listMeta)
		return nil

	case fpMovemCtrlStore:
		eaText, offset, eaMeta, err := decodeEA(data, inst.Address, 4, mode, reg, cpu)
		if err != nil {
			return err
		}
		listText, names := formatFPCtrlRegList(uint8((word2 & fpMovemCtrlSelect) >> 10))
		listMeta := registerListOperand(listText, names)
		setInstruction(data, inst, offset, "FMOVEM.L", listText+", "+eaText, listMeta, eaMeta)
		return nil

	case fpMovemFPnLoad:
		eaText, offset, eaMeta, err := decodeEA(data, inst.Address, 4, mode, reg, cpu)
		if err != nil {
			return err
		}
		if word2&fpMovemDynBit != 0 {
			dMeta := registerOperand(RegisterKindData, uint8((word2>>4)&0x7))
			setInstruction(data, inst, offset, "FMOVEM.X", eaText+", "+dMeta.Text, eaMeta, dMeta)
			return nil
		}
		listText, regs := formatFPRegisterList(word2&0xFF, false)
		listMeta := registerListOperand(listText, regs)
		setInstruction(data, inst, offset, "FMOVEM.X", eaText+", "+listText, eaMeta, listMeta)
		return nil

	case fpMovemFPnStore:
		eaText, offset, eaMeta, err := decodeEA(data, inst.Address, 4, mode, reg, cpu)
		if err != nil {
			return err
		}
		predecrement := word2&fpMovemPredecBit == 0
		if word2&fpMovemDynBit != 0 {
			dMeta := registerOperand(RegisterKindData, uint8((word2>>4)&0x7))
			setInstruction(data, inst, offset, "FMOVEM.X", dMeta.Text+", "+eaText, dMeta, eaMeta)
			return nil
		}
		listText, regs := formatFPRegisterList(word2&0xFF, predecrement)
		listMeta := registerListOperand(listText, regs)
		setInstruction(data, inst, offset, "FMOVEM.X", listText+", "+eaText, listMeta, eaMeta)
		return nil

	default:
		return fmt.Errorf("unrecognized FPU coprocessor command word: $%04X", word2)
	}
}

// formatFPCtrlRegList renders FMOVEM's 3-bit FPCR/FPSR/FPIAR selector mask
// (bit 0 = FPIAR, bit 1 = FPSR, bit 2 = FPCR — GAS's own bit assignment,
// see m68kasm's fpCtrlRegisterBit). Printed high-bit-first (FPCR/FPSR/
// FPIAR when all three are set), matching the order m68kasm's own test
// source consistently writes them in (cpu020_fpu_movem_ctrl_test.go),
// there being no numeric register ordering to fall back on the way
// formatRegisterRange has for D0-D7/FP0-FP7.
func formatFPCtrlRegList(mask uint8) (string, []string) {
	var names []string
	if mask&0x4 != 0 {
		names = append(names, "FPCR")
	}
	if mask&0x2 != 0 {
		names = append(names, "FPSR")
	}
	if mask&0x1 != 0 {
		names = append(names, "FPIAR")
	}
	return strings.Join(names, "/"), names
}

// formatFPRegisterList mirrors formatRegisterList (special.go) for FMOVEM's
// 8-bit FP0-FP7 mask: reverse selects the bit-reflected order integer
// MOVEM's -(An) predecrement form also uses (see formatRegisterList's own
// "reverse" parameter), here over 8 bits instead of 16 since there is no
// second (address-register) half to a FPn list.
func formatFPRegisterList(mask uint16, reverse bool) (string, []string) {
	bitFor := func(listIndex int) uint {
		if reverse {
			return uint(7 - listIndex)
		}
		return uint(listIndex)
	}
	var registers []string
	for i := range 8 {
		if mask&(1<<bitFor(i)) != 0 {
			registers = append(registers, fmt.Sprintf("FP%d", i))
		}
	}
	return formatRegisterRange(registers), registers
}

// decodeFPEAWithSize decodes one <ea> operand of an FPU instruction. The
// integer data formats (Long/Word/Byte) reuse the standard integer <ea>
// decoder unchanged (its immediate-mode handling already covers 1/2/4-byte
// immediates). The floating formats (Single/Double/Extended) need their own
// immediate handling for mode 7/reg 4, since a real FPU immediate in those
// formats is an IEEE-754 (or 68881 extended-precision) encoded constant, not
// a raw integer — see decodeFPFloatImmediate. Packed BCD immediates are not
// decoded to text (m68kasm's own encoder does not support generating them
// either — see validateFPUOperand's "packed BCD immediate literals are not
// supported" in cpu020_fpu.go); they render as a raw hex literal so the
// instruction still decodes to the correct length.
func decodeFPEAWithSize(data []byte, address uint32, offset int, mode, reg uint8, fmtCode uint16, cpu CPU) (string, int, Operand, error) {
	info := fpFormats[fmtCode]
	if mode == 7 && reg == 4 {
		switch fmtCode {
		case 1, 2, 5: // Single, Extended, Double
			return decodeFPFloatImmediate(data, offset, fmtCode)
		case 3: // Packed BCD
			return decodeFPRawImmediate(data, offset, info.bytes)
		}
	}
	return decodeEAWithSize(data, address, offset, mode, reg, info.bytes, cpu)
}

func decodeFPFloatImmediate(data []byte, offset int, fmtCode uint16) (string, int, Operand, error) {
	info := fpFormats[fmtCode]
	if err := requireLength(data, offset+info.bytes, "FPU float immediate"); err != nil {
		return "", offset, Operand{}, err
	}
	raw := append([]byte(nil), data[offset:offset+info.bytes]...)

	var value float64
	switch fmtCode {
	case 1: // Single
		value = float64(math.Float32frombits(binary.BigEndian.Uint32(raw)))
	case 5: // Double
		value = math.Float64frombits(binary.BigEndian.Uint64(raw))
	case 2: // Extended
		value = decodeExtendedReal(raw)
	}

	text := "#" + strconv.FormatFloat(value, 'g', -1, 64)
	nextOffset := offset + info.bytes
	return text, nextOffset, Operand{
		Text:      text,
		Kind:      OperandKindImmediate,
		Immediate: &ImmediateValue{Size: uint8(info.bytes), RawBytes: raw},
	}, nil
}

func decodeFPRawImmediate(data []byte, offset, size int) (string, int, Operand, error) {
	if err := requireLength(data, offset+size, "FPU packed-BCD immediate"); err != nil {
		return "", offset, Operand{}, err
	}
	raw := append([]byte(nil), data[offset:offset+size]...)
	text := "#$" + hex.EncodeToString(raw)
	nextOffset := offset + size
	return text, nextOffset, Operand{
		Text:      text,
		Kind:      OperandKindImmediate,
		Immediate: &ImmediateValue{Size: uint8(size), RawBytes: raw},
	}, nil
}

// decodeExtendedReal converts the 68881/68882's 96-bit ("double extended")
// real format back to a float64. raw is 12 bytes, big-endian: word0 is a
// 1-bit sign plus a 15-bit biased (bias 16383) exponent, word1 is reserved
// (ignored), and the remaining 8 bytes are a 64-bit mantissa with an
// EXPLICIT leading integer bit — bit-for-bit the x87 80-bit extended format
// with an extra reserved word after the exponent. This is the exact inverse
// of m68kasm's encodeExtendedReal (internal/asm/encode.go), which this
// project's fpu_test.go round-trips against via m68kasm.AssembleString to
// confirm the two agree, rather than trusting the derivation alone.
//
// Only handles finite values, matching encodeExtendedReal's own scope (the
// lexer that feeds it can't produce infinities/NaNs either).
func decodeExtendedReal(raw []byte) float64 {
	word0 := binary.BigEndian.Uint16(raw[0:2])
	mantissa := binary.BigEndian.Uint64(raw[4:12])
	sign := word0&0x8000 != 0
	exponent := int(word0 & 0x7FFF)

	if mantissa == 0 && exponent == 0 {
		if sign {
			return math.Copysign(0, -1)
		}
		return 0
	}

	value := math.Ldexp(float64(mantissa), exponent-16383-63)
	if sign {
		value = -value
	}
	return value
}

// decodeFSAVE and decodeFRESTORE decode the FPU coprocessor state-frame
// save/restore instructions. Unlike every other instruction in this file,
// these are single-word opcodes (word1 = fpuWord1Base | 0x0100 for FSAVE,
// | 0x0140 for FRESTORE, with the <ea> mode/reg in bits 5-0 as usual) —
// there is no coprocessor command word2, since the operand is purely a
// destination/source address, not a register or format selector.
//
// The frame body FSAVE writes (or FRESTORE reads) at runtime is a
// hardware/implementation-defined state dump with no disassembly-time
// meaning — like MOVEM's own <ea> operand not needing to know what values
// live in memory, decoding the instruction shell (mnemonic + <ea>) is the
// whole job; interpreting frame contents is out of scope (see the
// Non-goals section of docs/design-fpu-mmu.md).
//
// Cross-checked against m68kasm's newFSaveRestoreDef (cpu020_fpu_cond.go),
// including that file's own note that this codebase's word1 literals were
// originally shipped missing fpuWord1Base's coprocessor-ID bit (0xF100/
// 0xF140 instead of 0xF300/0xF340) — a bug class already found and fixed
// there once; this decoder is written against the corrected values.
func decodeFSAVE(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeFSaveRestore(data, opcode, inst, cpu, "FSAVE")
}

func decodeFRESTORE(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	return decodeFSaveRestore(data, opcode, inst, cpu, "FRESTORE")
}

func decodeFSaveRestore(data []byte, opcode uint16, inst *Instruction, cpu CPU, mnemonic string) error {
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	eaText, offset, eaMeta, err := decodeEA(data, inst.Address, 2, mode, reg, cpu)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, mnemonic, eaText, eaMeta)
	return nil
}
