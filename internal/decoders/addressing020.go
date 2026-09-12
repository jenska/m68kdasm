package decoders

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// This file implements the 68020+ "full" extension word addressing modes:
// memory indirect (pre-indexed and post-indexed), base/index register
// suppression, scaled index registers, and 0/16/32-bit base and outer
// displacements. It is used from decodeAddressingMode (addressing.go) for
// modes 6 (An + index) and 7/3 (PC + index) whenever the extension word's
// "full format" bit (bit 8) is set and the target CPU supports it.
//
// Full extension word layout:
//
//	15    D/A (index register type)
//	14-12 index register number
//	11    index size (0=sign-extended word, 1=long)
//	10-9  scale (00=*1, 01=*2, 10=*4, 11=*8)
//	8     1 = full format
//	7     BS: base register suppress
//	6     IS: index suppress
//	5-4   base displacement size (00=reserved/treated as null, 01=null, 10=word, 11=long)
//	3     reserved (0)
//	2-0   I/IS: index/indirect selection (see classifyIIS)
func cpuHasFullExtWords(cpu CPU) bool {
	switch cpu {
	case M68020, M68030, M68040, M68060:
		return true
	default:
		return false
	}
}

// iisResult describes the memory-indirection behavior selected by the I/IS
// field (bits 2-0 of the full extension word), per the Motorola 68020+
// Programmer's Reference Manual.
type iisResult struct {
	indirect   bool // memory indirection: dereference the intermediate pointer
	preIndexed bool // only meaningful when indirect && an index is present
	outerBytes int  // 0, 2, or 4
	reserved   bool
}

func classifyIIS(iis uint8, indexPresent bool) iisResult {
	if !indexPresent {
		switch iis {
		case 0:
			return iisResult{}
		case 1:
			return iisResult{indirect: true}
		case 2:
			return iisResult{indirect: true, outerBytes: 2}
		case 3:
			return iisResult{indirect: true, outerBytes: 4}
		default:
			return iisResult{reserved: true}
		}
	}
	switch iis {
	case 0:
		return iisResult{}
	case 1:
		return iisResult{indirect: true, preIndexed: true}
	case 2:
		return iisResult{indirect: true, preIndexed: true, outerBytes: 2}
	case 3:
		return iisResult{indirect: true, preIndexed: true, outerBytes: 4}
	case 5:
		return iisResult{indirect: true}
	case 6:
		return iisResult{indirect: true, outerBytes: 2}
	case 7:
		return iisResult{indirect: true, outerBytes: 4}
	default: // 4 and any other value
		return iisResult{reserved: true}
	}
}

// decodeFullExtension decodes a 68020+ full-format extension word (and any
// base/outer displacement words that follow it) for an An-relative (isPC
// false, baseReg = An number) or PC-relative (isPC true) operand. data must
// start at the extension word. Returns the formatted text, the number of
// 16-bit words consumed starting at data[0] (extension word plus
// displacements), and the structured operand.
func decodeFullExtension(data []byte, mode, baseReg uint8, isPC bool) (string, int, Operand, error) {
	if err := requireLength(data, 2, "full extension word"); err != nil {
		return "", 0, Operand{}, err
	}
	extWord := binary.BigEndian.Uint16(data[:2])
	consumed := 1

	indexSuppress := extWord&0x0040 != 0
	baseSuppress := extWord&0x0080 != 0

	var index *IndexRegister
	if !indexSuppress {
		kind := RegisterKindData
		if extWord&0x8000 != 0 {
			kind = RegisterKindAddress
		}
		num := uint8((extWord >> 12) & 0x7)
		sizeCh := "W"
		if extWord&0x0800 != 0 {
			sizeCh = "L"
		}
		scale := uint8(1) << ((extWord >> 9) & 0x3)
		index = &IndexRegister{Register: Register{Kind: kind, Number: num}, Size: sizeCh, Scale: scale}
	}

	var bdWords int
	switch (extWord >> 4) & 0x3 {
	case 2:
		bdWords = 1
	case 3:
		bdWords = 2
	default: // 0 (reserved) and 1 (null) both contribute no displacement word
		bdWords = 0
	}
	if err := requireLength(data, (consumed+bdWords)*2, "base displacement"); err != nil {
		return "", 0, Operand{}, err
	}
	var bd int32
	switch bdWords {
	case 1:
		bd = int32(int16(binary.BigEndian.Uint16(data[consumed*2:])))
	case 2:
		bd = int32(binary.BigEndian.Uint32(data[consumed*2:]))
	}
	consumed += bdWords

	cls := classifyIIS(uint8(extWord&0x7), index != nil)
	if cls.reserved {
		return "", 0, Operand{}, fmt.Errorf("reserved I/IS field in full extension word: %d", extWord&0x7)
	}

	if err := requireLength(data, (consumed*2)+cls.outerBytes, "outer displacement"); err != nil {
		return "", 0, Operand{}, err
	}
	var od int32
	switch cls.outerBytes {
	case 2:
		od = int32(int16(binary.BigEndian.Uint16(data[consumed*2:])))
		consumed++
	case 4:
		od = int32(binary.BigEndian.Uint32(data[consumed*2:]))
		consumed += 2
	}

	var base *Register
	if !baseSuppress {
		if isPC {
			base = &Register{Kind: RegisterKindPC}
		} else {
			base = &Register{Kind: RegisterKindAddress, Number: baseReg}
		}
	}

	// Matches the brief-mode PC-relative convention: the displayed base
	// displacement is the raw encoded value, unadjusted.
	text := formatFullExtension(base, bd, bdWords > 0, index, cls, od, cls.outerBytes > 0)

	kind := EAKindIndex
	if isPC {
		kind = EAKindPCIndex
	}
	if cls.indirect {
		kind = EAKindMemoryIndirect
	}

	ea := EffectiveAddress{
		Kind:     kind,
		Mode:     mode,
		Register: baseReg,
		Base:     base,
		Index:    index,
	}
	if bdWords > 0 {
		d := bd
		ea.Displacement = &d
	}
	if cls.indirect {
		ea.PreIndexed = cls.preIndexed
		if cls.outerBytes > 0 {
			o := od
			ea.OuterDisplacement = &o
		}
	}

	return text, consumed, effectiveAddressOperand(text, ea), nil
}

func formatFullExtension(base *Register, bd int32, hasBD bool, index *IndexRegister, cls iisResult, od int32, hasOD bool) string {
	baseText := ""
	if base != nil {
		if base.Kind == RegisterKindPC {
			baseText = "PC"
		} else {
			baseText = fmt.Sprintf("A%d", base.Number)
		}
	}
	indexText := ""
	if index != nil {
		indexText = fmt.Sprintf("%s%d.%s", registerPrefix(index.Register.Kind), index.Register.Number, index.Size)
		if index.Scale != 1 {
			indexText += fmt.Sprintf("*%d", index.Scale)
		}
	}
	bdText := ""
	if hasBD {
		bdText = fmt.Sprintf("%d", bd)
	}
	odText := ""
	if hasOD {
		odText = fmt.Sprintf("%d", od)
	}

	if !cls.indirect {
		return "(" + joinNonEmpty(bdText, baseText, indexText) + ")"
	}

	if cls.preIndexed || index == nil {
		inner := "[" + joinNonEmpty(bdText, baseText, indexText) + "]"
		return "(" + joinNonEmpty(inner, odText) + ")"
	}

	// Post-indexed: the index applies after dereferencing, so it sits
	// outside the bracketed [bd,An] group.
	inner := "[" + joinNonEmpty(bdText, baseText) + "]"
	return "(" + joinNonEmpty(inner, indexText, odText) + ")"
}

func joinNonEmpty(items ...string) string {
	nonEmpty := make([]string, 0, len(items))
	for _, it := range items {
		if it != "" {
			nonEmpty = append(nonEmpty, it)
		}
	}
	return strings.Join(nonEmpty, ",")
}
