package m68kdasm

import "testing"

func TestCPUOptionDefaultsToM68000(t *testing.T) {
	if CPU(0) != M68000 {
		t.Fatalf("expected zero value of CPU to be M68000")
	}
}

func TestMOVES_RequiresCPU010OrLater(t *testing.T) {
	// MOVES.W D0,(A1)
	data := []byte{0x0E, 0x51, 0x00, 0x00}

	inst, err := DecodeWithOptions(data, 0x1000, DecodeOptions{})
	if err != nil {
		t.Fatalf("unexpected error decoding with default (M68000) options: %v", err)
	}
	if inst.Mnemonic != "DC.W" {
		t.Fatalf("expected MOVES to be unrecognized on M68000, got %q", inst.Assembly())
	}

	inst, err = DecodeWithOptions(data, 0x1000, DecodeOptions{CPU: M68010})
	if err != nil {
		t.Fatalf("unexpected error decoding with M68010: %v", err)
	}
	if inst.Assembly() != "MOVES.W D0, (A1)" {
		t.Fatalf("got %q", inst.Assembly())
	}
}

func TestBGND_OnlyOnCPU32(t *testing.T) {
	data := []byte{0x4A, 0xFA}

	inst, err := DecodeWithOptions(data, 0x1000, DecodeOptions{CPU: CPU32})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inst.Assembly() != "BGND" {
		t.Fatalf("got %q", inst.Assembly())
	}

	// On every other CPU the same bytes still fall through to the legacy TST
	// family match (mode 7/reg 2 = PC displacement, hence the extra word),
	// just not as BGND.
	inst, err = DecodeWithOptions([]byte{0x4A, 0xFA, 0x00, 0x00}, 0x1000, DecodeOptions{CPU: M68020})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inst.Assembly() == "BGND" {
		t.Fatalf("BGND should not decode on M68020")
	}
}

func TestRTD_RequiresCPU010OrLater(t *testing.T) {
	data := []byte{0x4E, 0x74, 0x00, 0x08}

	inst, err := DecodeWithOptions(data, 0x1000, DecodeOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inst.Mnemonic != "DC.W" {
		t.Fatalf("expected RTD to be unrecognized on M68000, got %q", inst.Assembly())
	}

	inst, err = DecodeWithOptions(data, 0x1000, DecodeOptions{CPU: M68010})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inst.Assembly() != "RTD #8" {
		t.Fatalf("got %q", inst.Assembly())
	}
}

func TestMOVEC_RequiresCPU010OrLater(t *testing.T) {
	data := []byte{0x4E, 0x7A, 0x88, 0x01} // MOVEC VBR,A0

	inst, err := DecodeWithOptions(data, 0x1000, DecodeOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inst.Mnemonic != "DC.W" {
		t.Fatalf("expected MOVEC to be unrecognized on M68000, got %q", inst.Assembly())
	}

	inst, err = DecodeWithOptions(data, 0x1000, DecodeOptions{CPU: CPU32})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inst.Assembly() != "MOVEC VBR, A0" {
		t.Fatalf("got %q", inst.Assembly())
	}
}
