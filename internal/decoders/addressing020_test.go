package decoders

import "testing"

func TestFullExtension_NoIndirect_ScaledIndex(t *testing.T) {
	// D/A=D,reg=1,size=L,scale=*4,full=1,BS=0,IS=0,bdsize=word(10),I/IS=0
	// ext word = 0x1D20, bd word = 0x0008
	data := []byte{0x1D, 0x20, 0x00, 0x08}
	text, consumed, op, err := decodeFullExtension(data, 6, 0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "(8,A0,D1.L*4)" {
		t.Fatalf("got %q", text)
	}
	if consumed != 2 {
		t.Fatalf("expected 2 words consumed, got %d", consumed)
	}
	if op.EffectiveAddress.Kind != EAKindIndex {
		t.Fatalf("expected EAKindIndex, got %v", op.EffectiveAddress.Kind)
	}
	if op.EffectiveAddress.Index == nil || op.EffectiveAddress.Index.Scale != 4 {
		t.Fatalf("expected index scale 4, got %+v", op.EffectiveAddress.Index)
	}
}

func TestFullExtension_BaseSuppressed(t *testing.T) {
	// D/A=D,reg=1,size=W,scale=*1,full=1,BS=1,IS=0,bdsize=null(01),I/IS=0
	// ext word = 0x1190
	data := []byte{0x11, 0x90}
	text, consumed, op, err := decodeFullExtension(data, 6, 0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "(D1.W)" {
		t.Fatalf("got %q", text)
	}
	if consumed != 1 {
		t.Fatalf("expected 1 word consumed, got %d", consumed)
	}
	if op.EffectiveAddress.Base != nil {
		t.Fatalf("expected suppressed base, got %+v", op.EffectiveAddress.Base)
	}
}

func TestFullExtension_PCRelative_DisplacementConvention(t *testing.T) {
	// Same bit layout as the scaled-index case above but used as PC-relative;
	// mirrors brief-mode PC ops in displaying the raw encoded value.
	data := []byte{0x1D, 0x20, 0x00, 0x04} // bd = 4 (raw)
	text, _, _, err := decodeFullExtension(data, 7, 0, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "(4,PC,D1.L*4)" {
		t.Fatalf("got %q", text)
	}
}

func TestFullExtension_PreIndexed(t *testing.T) {
	// D/A=D,reg=0,size=W,scale=*1,full=1,BS=0,IS=0,bdsize=null,I/IS=2(preindexed,word outer)
	// ext word = 0x0112, outer word = 0x000C (12)
	data := []byte{0x01, 0x12, 0x00, 0x0C}
	text, consumed, op, err := decodeFullExtension(data, 6, 0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "([A0,D0.W],12)" {
		t.Fatalf("got %q", text)
	}
	if consumed != 2 {
		t.Fatalf("expected 2 words consumed, got %d", consumed)
	}
	if op.EffectiveAddress.Kind != EAKindMemoryIndirect || !op.EffectiveAddress.PreIndexed {
		t.Fatalf("expected pre-indexed memory indirect, got %+v", op.EffectiveAddress)
	}
	if op.EffectiveAddress.OuterDisplacement == nil || *op.EffectiveAddress.OuterDisplacement != 12 {
		t.Fatalf("expected outer displacement 12, got %+v", op.EffectiveAddress.OuterDisplacement)
	}
}

func TestFullExtension_PostIndexed(t *testing.T) {
	// Same as pre-indexed but I/IS=6 (postindexed, word outer), outer = -4
	// ext word = 0x0116, outer word = 0xFFFC (-4)
	data := []byte{0x01, 0x16, 0xFF, 0xFC}
	text, consumed, op, err := decodeFullExtension(data, 6, 0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "([A0],D0.W,-4)" {
		t.Fatalf("got %q", text)
	}
	if consumed != 2 {
		t.Fatalf("expected 2 words consumed, got %d", consumed)
	}
	if op.EffectiveAddress.Kind != EAKindMemoryIndirect || op.EffectiveAddress.PreIndexed {
		t.Fatalf("expected post-indexed memory indirect, got %+v", op.EffectiveAddress)
	}
}

func TestFullExtension_MemoryIndirectNoIndex(t *testing.T) {
	// IS=1 (index suppressed), BS=0, bdsize=null, I/IS=2 (word outer)
	// ext word = 0x0152, outer word = 0x0064 (100)
	data := []byte{0x01, 0x52, 0x00, 0x64}
	text, consumed, op, err := decodeFullExtension(data, 6, 0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "([A0],100)" {
		t.Fatalf("got %q", text)
	}
	if consumed != 2 {
		t.Fatalf("expected 2 words consumed, got %d", consumed)
	}
	if op.EffectiveAddress.Index != nil {
		t.Fatalf("expected no index, got %+v", op.EffectiveAddress.Index)
	}
}

func TestFullExtension_ReservedIIS(t *testing.T) {
	// I/IS=4 is reserved regardless of index presence.
	data := []byte{0x01, 0x14}
	if _, _, _, err := decodeFullExtension(data, 6, 0, false); err == nil {
		t.Fatalf("expected error for reserved I/IS field")
	}
}

func TestDecodeAddressingMode_FullFormatGatedByCPU(t *testing.T) {
	// Bytes shaped as a full-format extension word (bit8 set): on 68000/010/
	// CPU32 this must fall back to brief-mode interpretation (bit 8 ignored,
	// low byte used as an 8-bit displacement), matching real silicon; on
	// 68020+ it must use the full decode.
	data := []byte{0x1D, 0x20, 0x00, 0x08}

	briefText, briefWords, _, err := decodeAddressingMode(data, 6, 0, 2, M68000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if briefWords != 1 {
		t.Fatalf("brief mode should consume exactly 1 extension word, got %d", briefWords)
	}

	fullText, fullWords, _, err := decodeAddressingMode(data, 6, 0, 2, M68020)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fullWords != 2 {
		t.Fatalf("full mode should consume 2 words (ext + bd), got %d", fullWords)
	}
	if briefText == fullText {
		t.Fatalf("expected brief and full decodes of the same bytes to differ, both got %q", briefText)
	}

	for _, cpu := range []CPU{M68010, CPU32} {
		text, words, _, err := decodeAddressingMode(data, 6, 0, 2, cpu)
		if err != nil {
			t.Fatalf("unexpected error on %v: %v", cpu, err)
		}
		if words != 1 || text != briefText {
			t.Fatalf("%v should decode identically to M68000 (brief-only), got %q/%d vs %q/%d", cpu, text, words, briefText, briefWords)
		}
	}
}
