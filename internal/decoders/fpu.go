package decoders

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
)

// This file decodes the FPU "general instruction" family (68881/68882, or
// the 68040/68060's built-in FPU, which is opcode-compatible for this
// subset): FMOVE, FADD, FSUB, FMUL, FDIV, FCMP, FABS, FNEG, FSQRT, FTST, and
// FNOP. It is the first slice of docs/design-fpu-mmu.md's delivery
// sequence (step 3), deliberately matching the scope of
// github.com/jenska/m68kasm's own first FPU milestone (internal/asm/
// instructions/cpu020_fpu.go) — the transcendental function set, FMOVEM,
// FBcc/FDBcc/FScc/FTRAPcc, FSAVE/FRESTORE, and packed-BCD store forms are
// follow-ups, not implemented here.
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
}

func decodeFNOP(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	if err := requireLength(data, 4, "FNOP"); err != nil {
		return err
	}
	setInstruction(data, inst, 4, "FNOP", "")
	return nil
}

// decodeFPGeneric decodes the FPU general-instruction family — see the
// bit-layout notes in this file's header comment.
func decodeFPGeneric(data []byte, opcode uint16, inst *Instruction, cpu CPU) error {
	if err := requireLength(data, 4, "FPU generic instruction extension word"); err != nil {
		return err
	}
	word2 := binary.BigEndian.Uint16(data[2:4])

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
