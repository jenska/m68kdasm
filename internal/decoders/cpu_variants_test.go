package decoders

import (
	"fmt"
	"testing"
)

func sameDecoder(a, b OpcodeDecoder) bool {
	return fmt.Sprintf("%p", a) == fmt.Sprintf("%p", b)
}

func TestMOVES_CPUGating(t *testing.T) {
	if d := FindDecoder(valMOVES, M68000, false, false); d != nil {
		t.Fatalf("MOVES should not decode on M68000, got %p", d)
	}
	for _, cpu := range []CPU{M68010, CPU32, M68020, M68030, M68040, M68060} {
		if d := FindDecoder(valMOVES, cpu, false, false); !sameDecoder(d, decodeMOVES) {
			t.Fatalf("MOVES should decode on %v, got %p", cpu, d)
		}
	}
}

func TestMOVEC_CPUGating(t *testing.T) {
	for _, op := range []uint16{valMOVECFromCtl, valMOVECToCtl} {
		if d := FindDecoder(op, M68000, false, false); d != nil {
			t.Fatalf("MOVEC (opcode %04X) should not decode on M68000, got %p", op, d)
		}
		if d := FindDecoder(op, M68010, false, false); d == nil {
			t.Fatalf("MOVEC (opcode %04X) should decode on M68010", op)
		}
	}
}

func TestRTD_CPUGating(t *testing.T) {
	if d := FindDecoder(valRTD, M68000, false, false); d != nil {
		t.Fatalf("RTD should not decode on M68000, got %p", d)
	}
	if d := FindDecoder(valRTD, CPU32, false, false); !sameDecoder(d, decodeRTD) {
		t.Fatalf("RTD should decode on CPU32, got %p", d)
	}
}

func TestBGND_CPU32Only(t *testing.T) {
	if d := FindDecoder(valBGND, CPU32, false, false); !sameDecoder(d, decodeBGND) {
		t.Fatalf("BGND should decode as BGND on CPU32, got %p", d)
	}
	for _, cpu := range []CPU{M68000, M68010, M68020, M68030, M68040, M68060} {
		d := FindDecoder(valBGND, cpu, false, false)
		if d == nil {
			t.Fatalf("opcode %04X should still fall back to the TST decoder on %v", valBGND, cpu)
		}
		if sameDecoder(d, decodeBGND) {
			t.Fatalf("opcode %04X should NOT decode as BGND on %v", valBGND, cpu)
		}
	}
}

func TestDecodeMOVEC(t *testing.T) {
	// MOVEC VBR,A0 : ctrl(VBR=0x801) -> A0
	data := []byte{0x4E, 0x7A, 0x88, 0x01}
	inst := &Instruction{Address: 0x1000, Opcode: valMOVECFromCtl}
	if err := decodeMOVECFromControl(data, valMOVECFromCtl, inst, M68010); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if inst.Mnemonic != "MOVEC" || inst.Operands != "VBR, A0" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}
	if inst.Size != 4 {
		t.Fatalf("expected size 4, got %d", inst.Size)
	}

	// MOVEC D1,USP : D1 -> ctrl(USP=0x800)
	data2 := []byte{0x4E, 0x7B, 0x18, 0x00}
	inst2 := &Instruction{Address: 0x1000, Opcode: valMOVECToCtl}
	if err := decodeMOVECToControl(data2, valMOVECToCtl, inst2, M68010); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if inst2.Mnemonic != "MOVEC" || inst2.Operands != "D1, USP" {
		t.Fatalf("got %q %q", inst2.Mnemonic, inst2.Operands)
	}
}

func TestDecodeMOVES(t *testing.T) {
	// MOVES.W D0,(A1) : opcode 0x0E51, ext 0x0000 (D0, register->memory)
	data := []byte{0x0E, 0x51, 0x00, 0x00}
	opcode := uint16(0x0E51)
	inst := &Instruction{Address: 0x1000, Opcode: opcode}
	if err := decodeMOVES(data, opcode, inst, M68010); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if inst.Mnemonic != "MOVES.W" || inst.Operands != "D0, (A1)" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}
	if inst.Size != 4 {
		t.Fatalf("expected size 4, got %d", inst.Size)
	}
}

func TestDecodeRTD(t *testing.T) {
	data := []byte{0x4E, 0x74, 0x00, 0x08}
	inst := &Instruction{Address: 0x1000, Opcode: valRTD}
	if err := decodeRTD(data, valRTD, inst, M68010); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if inst.Mnemonic != "RTD" || inst.Operands != "#8" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}
	if inst.Size != 4 {
		t.Fatalf("expected size 4, got %d", inst.Size)
	}
}

func TestDecodeBGND(t *testing.T) {
	data := []byte{0x4A, 0xFA}
	inst := &Instruction{Address: 0x1000, Opcode: valBGND}
	if err := decodeBGND(data, valBGND, inst, CPU32); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if inst.Mnemonic != "BGND" || inst.Operands != "" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}
	if inst.Size != 2 {
		t.Fatalf("expected size 2, got %d", inst.Size)
	}
}

func TestDecodeMULLong(t *testing.T) {
	// MULU.L D0,D2 : opcode 0x4C00 (mode0,reg0), ext 0x0002 (dl=2)
	inst := decodeFor(t, []byte{0x4C, 0x00, 0x00, 0x02}, M68020)
	if inst.Mnemonic != "MULU.L" || inst.Operands != "D0, D2" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}
	if d := FindDecoder(0x4C00, M68000, false, false); sameDecoder(d, decodeMULLong) {
		t.Fatalf("MULU.L should not decode on M68000")
	}

	// MULS.L D0,D3:D2 (wide, signed): opcode 0x4C00, ext 0xB402
	inst = decodeFor(t, []byte{0x4C, 0x00, 0xB4, 0x02}, M68020)
	if inst.Mnemonic != "MULS.L" || inst.Operands != "D0, D3:D2" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}
}

func TestDecodeDIVLong(t *testing.T) {
	// DIVU.L D1,D0 : opcode 0x4C41 (mode0,reg1), ext 0x0000
	inst := decodeFor(t, []byte{0x4C, 0x41, 0x00, 0x00}, M68020)
	if inst.Mnemonic != "DIVU.L" || inst.Operands != "D1, D0" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}
}

func TestDecodePACK(t *testing.T) {
	// PACK D1,D0,#$1234 : opcode 0x8141
	inst := decodeFor(t, []byte{0x81, 0x41, 0x12, 0x34}, M68020)
	if inst.Mnemonic != "PACK" || inst.Operands != "D1, D0, #$1234" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}
	if d := FindDecoder(0x8141, M68000, false, false); sameDecoder(d, decodePACK) {
		t.Fatalf("PACK should not decode on M68000")
	}
}

func TestDecodeCALLM_RTM(t *testing.T) {
	// CALLM #4,(A0) : opcode 0x06D0 (mode2,reg0), ext 0x0004
	inst := decodeFor(t, []byte{0x06, 0xD0, 0x00, 0x04}, M68020)
	if inst.Mnemonic != "CALLM" || inst.Operands != "#4, (A0)" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}

	// RTM D3 : opcode 0x06C3 — must take priority over CALLM
	inst = decodeFor(t, []byte{0x06, 0xC3}, M68020)
	if inst.Mnemonic != "RTM" || inst.Operands != "D3" {
		t.Fatalf("got %q %q (RTM must precede CALLM in the opcode table)", inst.Mnemonic, inst.Operands)
	}

	// RTM A2 : opcode 0x06CA
	inst = decodeFor(t, []byte{0x06, 0xCA}, M68020)
	if inst.Mnemonic != "RTM" || inst.Operands != "A2" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}

	if d := FindDecoder(0x06C3, M68040, false, false); sameDecoder(d, decodeRTM) {
		t.Fatalf("RTM should not decode on M68040")
	}
}

func TestDecodeCHK2CMP2(t *testing.T) {
	// CHK2.W (A1),D3 : opcode 0x02D1 (mode2,reg1), ext 0x3800 (bit11=1 -> CHK2, reg=3)
	inst := decodeFor(t, []byte{0x02, 0xD1, 0x38, 0x00}, M68020)
	if inst.Mnemonic != "CHK2.W" || inst.Operands != "(A1), D3" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}

	// Same opcode, ext bit11=0 -> CMP2.W
	inst = decodeFor(t, []byte{0x02, 0xD1, 0x30, 0x00}, M68020)
	if inst.Mnemonic != "CMP2.W" || inst.Operands != "(A1), D3" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}
}

func TestDecodeCAS(t *testing.T) {
	// CAS.W D1,D2,(A3) : opcode 0x0CD3 (mode2,reg3), ext 0x000A (dc=1,du=2)
	inst := decodeFor(t, []byte{0x0C, 0xD3, 0x00, 0x0A}, M68020)
	if inst.Mnemonic != "CAS.W" || inst.Operands != "D1, D2, (A3)" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}
	if d := FindDecoder(0x0CD3, M68000, false, false); sameDecoder(d, decodeCAS) {
		t.Fatalf("CAS should not decode on M68000")
	}
}

func TestDecodeBitfield(t *testing.T) {
	// BFTST D0{4:8} : opcode 0xE0C0 (mode0,reg0), ext 0x0108
	inst := decodeFor(t, []byte{0xE0, 0xC0, 0x01, 0x08}, M68020)
	if inst.Mnemonic != "BFTST" || inst.Operands != "D0{4:8}" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}

	// BFEXTU D0{D1:D2},D3 : opcode 0xE1C0, ext 0x3922
	inst = decodeFor(t, []byte{0xE1, 0xC0, 0x39, 0x22}, M68020)
	if inst.Mnemonic != "BFEXTU" || inst.Operands != "D0{D1:D2}, D3" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}

	// BFINS D3,D0{4:8} : opcode 0xE7C0, ext 0x3108
	inst = decodeFor(t, []byte{0xE7, 0xC0, 0x31, 0x08}, M68020)
	if inst.Mnemonic != "BFINS" || inst.Operands != "D3, D0{4:8}" {
		t.Fatalf("got %q %q", inst.Mnemonic, inst.Operands)
	}

	if d := FindDecoder(0xE0C0, M68000, false, false); sameDecoder(d, decodeBFTST) {
		t.Fatalf("BFTST should not decode on M68000")
	}
}

// These tests confirm the bitset architecture actually delivers 68030/68040/
// 68060 support "for free" from the 68020 additions: every 68020-tagged
// opcode should also decode on 68030/68040/68060, except CALLM/RTM which
// real hardware dropped starting with the 68040.
func TestCPU020Opcodes_InheritedBy030_040_060(t *testing.T) {
	opcodes := []uint16{
		valBFTST, valBFEXTU, valBFCHG, valBFEXTS, valBFCLR, valBFFFO, valBFSET, valBFINS,
		valCASB, valCASW, valCASL,
		valCHK2B, valCHK2W, valCHK2L,
		valMULLong, valDIVLong,
		valPACK, valUNPK,
		valLINKL, valEXTB, valCHKL,
		valTRAPcc,
	}
	for _, cpu := range []CPU{M68030, M68040, M68060} {
		for _, op := range opcodes {
			if FindDecoder(op, cpu, false, false) == nil {
				t.Errorf("opcode %04X should decode on %v (inherited from 68020 tagging)", op, cpu)
			}
		}
	}
}

func TestCALLM_RTM_RemovedStarting68040(t *testing.T) {
	for _, op := range []uint16{valCALLM, valRTMDn, valRTMAn} {
		if FindDecoder(op, M68030, false, false) == nil {
			t.Errorf("opcode %04X should still decode on M68030", op)
		}
		for _, cpu := range []CPU{M68040, M68060} {
			d := FindDecoder(op, cpu, false, false)
			if sameDecoder(d, decodeCALLM) || sameDecoder(d, decodeRTM) {
				t.Errorf("opcode %04X should not decode as CALLM/RTM on %v", op, cpu)
			}
		}
	}
}
