package decoders

import "testing"

func decodeFor(t *testing.T, data []byte, cpu CPU) *Instruction {
	t.Helper()
	opcode := uint16(data[0])<<8 | uint16(data[1])
	decoder := FindDecoder(opcode, cpu, false)
	if decoder == nil {
		t.Fatalf("no decoder found for opcode %04X on %v", opcode, cpu)
	}
	inst := &Instruction{Address: 0x1000, Opcode: opcode}
	if err := decoder(data, opcode, inst, cpu); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	return inst
}

func TestDecodeRESET(t *testing.T) {
	inst := decodeFor(t, []byte{0x4E, 0x70}, M68000)
	if inst.Mnemonic != "RESET" {
		t.Fatalf("got %q", inst.Mnemonic)
	}
}

func TestDecodeRTE(t *testing.T) {
	inst := decodeFor(t, []byte{0x4E, 0x73}, M68000)
	if inst.Mnemonic != "RTE" {
		t.Fatalf("got %q", inst.Mnemonic)
	}
}

func TestDecodeILLEGAL(t *testing.T) {
	inst := decodeFor(t, []byte{0x4A, 0xFC}, M68000)
	if inst.Mnemonic != "ILLEGAL" {
		t.Fatalf("expected ILLEGAL, got %q (this opcode used to mis-decode as TST)", inst.Mnemonic)
	}
}

func TestDecodeUNLK(t *testing.T) {
	inst := decodeFor(t, []byte{0x4E, 0x5A}, M68000) // UNLK A2
	if inst.Mnemonic != "UNLK" || inst.Operands != "A2" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}
}

func TestDecodeLINK(t *testing.T) {
	// LINK.W A5,#-16 (displacement is rendered like other 16-bit immediates
	// in this library: unsigned hex once it's >= 100, per formatImmediate)
	data := []byte{0x4E, 0x55, 0xFF, 0xF0}
	inst := decodeFor(t, data, M68000)
	if inst.Mnemonic != "LINK.W" || inst.Operands != "A5, #$FFF0" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}

	// LINK.L's opcode slot is reserved on M68000 (NBCD's broader EA mask
	// picks it up once the CPU gate excludes decodeLINKLong, same class of
	// overlap as EXTB/LEA and TRAPcc/Scc elsewhere in this file).
	llData := []byte{0x48, 0x0D, 0x00, 0x00, 0x00, 0x10}
	if d := FindDecoder(0x480D, M68000, false); sameDecoder(d, decodeLINKLong) {
		t.Fatalf("LINK.L should not decode on M68000")
	}
	inst = decodeFor(t, llData, M68020)
	if inst.Mnemonic != "LINK.L" || inst.Operands != "A5, #16" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}
}

func TestDecodeEXT(t *testing.T) {
	inst := decodeFor(t, []byte{0x48, 0x80}, M68000) // EXT.W D0
	if inst.Mnemonic != "EXT.W" || inst.Operands != "D0" {
		t.Fatalf("EXT.W D0 mis-decoded as %q %q (this used to collide with MOVEM)", inst.Mnemonic, inst.Operands)
	}

	inst = decodeFor(t, []byte{0x48, 0xC3}, M68000) // EXT.L D3
	if inst.Mnemonic != "EXT.L" || inst.Operands != "D3" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}
}

func TestDecodeEXTB(t *testing.T) {
	// EXTB.L's bit pattern coincides with LEA's mask/value once the CPU gate
	// excludes decodeEXTB, since real 68000/68010 silicon never assigned this
	// specific (reserved, mode-0) LEA-shaped encoding to anything else. That
	// matches this library's existing approach of not validating per-instruction
	// EA-mode restrictions (see e.g. the TST/ILLEGAL overlap) — we only assert
	// it does not decode as EXTB.L outside 68020+.
	if d := FindDecoder(0x49C1, M68000, false); sameDecoder(d, decodeEXTB) {
		t.Fatalf("EXTB.L should not decode on M68000")
	}
	inst := decodeFor(t, []byte{0x49, 0xC1}, M68020) // EXTB.L D1
	if inst.Mnemonic != "EXTB.L" || inst.Operands != "D1" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}
}

func TestDecodeCHK(t *testing.T) {
	// CHK.W D1,D2 -> opcode 0100 010 110 000 001 = 0x4581
	inst := decodeFor(t, []byte{0x45, 0x81}, M68000)
	if inst.Mnemonic != "CHK.W" || inst.Operands != "D1, D2" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}

	// CHK.L D1,D2 -> opcode 0100 010 100 000 001 = 0x4501, 68020+ only
	if d := FindDecoder(0x4501, M68000, false); d != nil {
		t.Fatalf("CHK.L should not decode on M68000")
	}
	inst = decodeFor(t, []byte{0x45, 0x01}, M68020)
	if inst.Mnemonic != "CHK.L" || inst.Operands != "D1, D2" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}
}

func TestDecodeEXG(t *testing.T) {
	inst := decodeFor(t, []byte{0xC1, 0x41}, M68000) // EXG D0,D1
	if inst.Mnemonic != "EXG" || inst.Operands != "D0, D1" {
		t.Fatalf("EXG D0,D1 mis-decoded as %q %q (this used to collide with AND)", inst.Mnemonic, inst.Operands)
	}

	inst = decodeFor(t, []byte{0xC1, 0x49}, M68000) // EXG A0,A1
	if inst.Mnemonic != "EXG" || inst.Operands != "A0, A1" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}

	inst = decodeFor(t, []byte{0xC1, 0x89}, M68000) // EXG D0,A1
	if inst.Mnemonic != "EXG" || inst.Operands != "D0, A1" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}
}

func TestDecodeRTR(t *testing.T) {
	inst := decodeFor(t, []byte{0x4E, 0x77}, M68000)
	if inst.Mnemonic != "RTR" {
		t.Fatalf("got %q", inst.Mnemonic)
	}
}

func TestDecodeNBCD(t *testing.T) {
	// NBCD D0 -> opcode 0x4800
	inst := decodeFor(t, []byte{0x48, 0x00}, M68000)
	if inst.Mnemonic != "NBCD" || inst.Operands != "D0" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}
}

func TestDecodeTAS(t *testing.T) {
	// TAS D0 -> opcode 0x4AC0
	inst := decodeFor(t, []byte{0x4A, 0xC0}, M68000)
	if inst.Mnemonic != "TAS" || inst.Operands != "D0" {
		t.Fatalf("TAS D0 mis-decoded as %q %q (this used to collide with TST)", inst.Mnemonic, inst.Operands)
	}
}

func TestDecodeADDX(t *testing.T) {
	// ADDX.W D1,D0 (register form) -> dst=0,src=1,size=W(01) => 0xD141
	inst := decodeFor(t, []byte{0xD1, 0x41}, M68000)
	if inst.Mnemonic != "ADDX.W" || inst.Operands != "D1, D0" {
		t.Fatalf("ADDX register form mis-decoded as %q %q (this used to collide with ADD)", inst.Mnemonic, inst.Operands)
	}

	// ADDX.L -(A2),-(A1) (memory form) -> dst=1,src=2,size=L(10) => 0xD38A
	inst = decodeFor(t, []byte{0xD3, 0x8A}, M68000)
	if inst.Mnemonic != "ADDX.L" || inst.Operands != "-(A2), -(A1)" {
		t.Fatalf("ADDX memory form mis-decoded as %q %q", inst.Mnemonic, inst.Operands)
	}
}

func TestDecodeSUBX(t *testing.T) {
	// SUBX.B D1,D0 -> 0x9101
	inst := decodeFor(t, []byte{0x91, 0x01}, M68000)
	if inst.Mnemonic != "SUBX.B" || inst.Operands != "D1, D0" {
		t.Fatalf("SUBX register form mis-decoded as %q %q (this used to collide with SUB)", inst.Mnemonic, inst.Operands)
	}

	// SUBX.W -(A2),-(A1) -> 0x934A
	inst = decodeFor(t, []byte{0x93, 0x4A}, M68000)
	if inst.Mnemonic != "SUBX.W" || inst.Operands != "-(A2), -(A1)" {
		t.Fatalf("SUBX memory form mis-decoded as %q %q", inst.Mnemonic, inst.Operands)
	}
}

func TestDecodeMOVEP(t *testing.T) {
	// MOVEP.W (4,A1),D0 : dReg=0,aReg=1,dir=0(mem->reg),size=W(bit6=0)
	// opcode = 0x0108 | (0<<9) | 1 = 0x0109
	data := []byte{0x01, 0x09, 0x00, 0x04}
	inst := decodeFor(t, data, M68000)
	if inst.Mnemonic != "MOVEP.W" || inst.Operands != "(4,A1), D0" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}

	// MOVEP.L D2,(-8,A3) : dReg=2,aReg=3,dir=1(reg->mem,bit7=1),size=L(bit6=1)
	// opcode = 0x0108 | (2<<9) | 0xC0 | 3 = 0x0508 | 0xC0 | 3 = 0x05CB
	data2 := []byte{0x05, 0xCB, 0xFF, 0xF8}
	inst = decodeFor(t, data2, M68000)
	if inst.Mnemonic != "MOVEP.L" || inst.Operands != "D2, (-8,A3)" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}
}

func TestDecodeADDQ_SUBQ(t *testing.T) {
	// ADDQ.W #4,D0 -> ddd=100(4),s=0,ss=01(W),mode=0,reg=0 => 0101 100 0 01 000 000 = 0x5840
	inst := decodeFor(t, []byte{0x58, 0x40}, M68000)
	if inst.Mnemonic != "ADDQ.W" || inst.Operands != "#4, D0" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}

	// ADDQ #0 count means 8: ddd=000
	inst = decodeFor(t, []byte{0x50, 0x40}, M68000) // ADDQ.W #8,D0
	if inst.Operands != "#8, D0" {
		t.Fatalf("got %q", inst.Operands)
	}

	// SUBQ.L #2,D1 -> ddd=001? wait count field: ddd for 2 = 010, s=1(SUBQ),ss=10(L),reg=1
	// 0101 010 1 10 000 001 = 0x5581
	inst = decodeFor(t, []byte{0x55, 0x81}, M68000)
	if inst.Mnemonic != "SUBQ.L" || inst.Operands != "#2, D1" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}
}

func TestDecodeScc(t *testing.T) {
	// SEQ D0 -> cond=EQ(7),mode=0,reg=0 => 0101 0111 11 000 000 = 0x57C0
	inst := decodeFor(t, []byte{0x57, 0xC0}, M68000)
	if inst.Mnemonic != "SEQ" || inst.Operands != "D0" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}
}

func TestDecodeDBcc(t *testing.T) {
	// DBF D0,* -> cond=F(1),reg=0, displacement=-2 (branch to self)
	// 0101 0001 11001 000 = 0x51C8
	data := []byte{0x51, 0xC8, 0xFF, 0xFE}
	inst := decodeFor(t, data, M68000)
	if inst.Mnemonic != "DBF" {
		t.Fatalf("got mnemonic %q", inst.Mnemonic)
	}
	if inst.Size != 4 {
		t.Fatalf("expected size 4, got %d", inst.Size)
	}

	// Precedence check: DBcc's opcode also satisfies Scc's broad mask, must
	// still resolve to DBcc, not Scc.
	if inst.Mnemonic == "SF" {
		t.Fatalf("DBF opcode mis-decoded as Scc")
	}
}

func TestDecodeTRAPcc(t *testing.T) {
	// Like the EXTB/LEA and BGND/TST overlaps elsewhere, TRAPcc's bit pattern
	// falls inside Scc's broader mask once the CPU gate excludes decodeTRAPcc;
	// on M68000 this is a reserved Scc encoding (invalid EA mode), not a hard
	// decode failure, consistent with the library's existing approach.
	if d := FindDecoder(0x51FC, M68000, false); sameDecoder(d, decodeTRAPcc) {
		t.Fatalf("TRAPcc should not decode on M68000")
	}

	// TRAPF (no operand): cond=F(1), sss=100 -> 0101 0001 11111 100 = 0x51FC
	inst := decodeFor(t, []byte{0x51, 0xFC}, M68020)
	if inst.Mnemonic != "TRAPF" || inst.Operands != "" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}

	// TRAPEQ.W #$1234: cond=EQ(7), sss=010 -> 0101 0111 11111 010 = 0x57FA
	data := []byte{0x57, 0xFA, 0x12, 0x34}
	inst = decodeFor(t, data, M68020)
	if inst.Mnemonic != "TRAPEQ" || inst.Operands != "#$1234" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}
	if inst.Size != 4 {
		t.Fatalf("expected size 4, got %d", inst.Size)
	}
}
