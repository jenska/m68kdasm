package m68kdasm

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/jenska/m68kasm"
)

func TestDisassembleRoundTrip(t *testing.T) {
	// Diese Testfälle definieren die gewünschte Syntax.
	// Der Assembler erzeugt daraus Maschinencode, und der Disassembler
	// muss exakt diesen String wiederherstellen.
	testCases := []string{
		"NOP",
		"RTS",
		"STOP #$2700",
		"TRAP #9",
		"TRAPV",
		"MOVEM.L D0/A0, (A1)",
		"CLR.B D0",
		"NEG.W D2",
		"NEGX.L D3",
		"NOT.W (A0)+",
		"TST.L D3",
		"JSR (A2)",
		"JMP (A0)",
		"LEA (A1), A7",
		"PEA (A1)",
		"MULU (A1), D0",
		"MULS (A1), D0",
		"DIVU (A2), D1",
		"DIVS (A2), D1",
		"BTST #3, D1",
		"BCHG D2, (A3)",
		"BCLR #7, (A1)",
		"BSET #0, (A0)+",
		"ABCD D1, D0",
		"SBCD D2, D3",
		"ADDI.W #$0100, D0",
		"SUBI.W #$0100, (A1)",
		"ANDI.B #16, D0",
		"ORI.W #$0100, (A0)",
		"EORI.W #$0100, D1",
		"CMPI.B #1, D2",
		"CMPA.L #$00000200, A0",
		"MOVEQ #$7F, D0",
		"MOVE.B D1, D0",
		"MOVE.W D0, D1",
		"MOVE.L D2, D3",
		"MOVEA.L #$00002140, A0",
		"MOVEA.L #$0001007C, A1",
		"OR.B D0, (A1)",
		"SUB.W D2, (A3)",
		"SUBA.W (A1), A2",
		"SUBA.L A0, A1",
		"CMP.B (A0), D2",
		"AND.W D1, D0",
		"ADD.W D1, D0",
		"EOR.W D0, (A1)",
		"LSL.W #1, D0",
		"MOVE.W A0, D1",
		"MOVE.W (A0), D1",
		"MOVE.W (A0)+, D1",
		"MOVE.W -(A0), D1",
		"MOVE.W (16,A0), D1",
		"MOVE.W (4,A0,D1.L), D1",
		"MOVE.W $00123456, D1",
		"MOVE.W (16,PC), D1",
		"MOVE.W (4,PC,D1.L), D1",
		"MOVE.W (4,PC,D1.W), D1",
		"MOVE.W #$1234, D1",
	}

	for _, source := range testCases {
		t.Run(source, func(t *testing.T) {
			// 1. Assemblieren (Source -> Bytes)
			bytes, err := m68kasm.AssembleString(source)
			if err != nil {
				t.Fatalf("Assembler-Fehler bei '%s': %v", source, err)
			}

			// 2. Disassemblieren (Bytes -> Instruction)
			instrs, err := DisassembleRange(bytes, 0)
			if err != nil {
				t.Fatalf("Disassembler-Fehler bei '%s': %v", source, err)
			}

			if len(instrs) != 1 {
				t.Fatalf("Erwartete 1 Instruktion, erhielt %d", len(instrs))
			}

			// 3. Vergleichen (Instruction.Assembly() == Source)
			got := instrs[0].Assembly()
			if got != source {
				t.Errorf("Mismatch!\nErwartet: '%s'\nErhalten: '%s'", source, got)
			}
		})
	}
}

func TestDecodeBranchInstructions(t *testing.T) {
	data := []byte{0x66, 0x02} // BNE.S to $0004 when starting at 0
	instrs, err := DisassembleRange(data, 0)
	if err != nil {
		t.Fatalf("Disassembler-Fehler: %v", err)
	}
	if len(instrs) != 1 {
		t.Fatalf("Erwartete 1 Instruktion, erhielt %d", len(instrs))
	}
	got := instrs[0].Assembly()
	want := "BNE.S $0004"
	if got != want {
		t.Errorf("Mismatch!\nErwartet: '%s'\nErhalten: '%s'", want, got)
	}
}

func TestDecodeAbsoluteShortAddressing(t *testing.T) {
	source := "MOVE.W $1234.W, D1"
	bytes, err := m68kasm.AssembleString(source)
	if err != nil {
		t.Fatalf("Assembler-Fehler bei '%s': %v", source, err)
	}
	instrs, err := DisassembleRange(bytes, 0)
	if err != nil {
		t.Fatalf("Disassembler-Fehler bei '%s': %v", source, err)
	}
	if len(instrs) != 1 {
		t.Fatalf("Erwartete 1 Instruktion, erhielt %d", len(instrs))
	}
	got := instrs[0].Assembly()
	want := "MOVE.W $1234, D1"
	if got != want {
		t.Errorf("Mismatch!\nErwartet: '%s'\nErhalten: '%s'", want, got)
	}
}

func TestDecodeRegressionRawOpcodes(t *testing.T) {
	testCases := []struct {
		name    string
		address uint32
		data    []byte
		want    string
	}{
		{
			name: "CMPA long immediate",
			data: []byte{0xB1, 0xFC, 0x00, 0x00, 0x02, 0x00},
			want: "CMPA.L #$00000200, A0",
		},
		{
			name: "MOVEA long immediate A0",
			data: []byte{0x20, 0x7C, 0x00, 0x00, 0x21, 0x40},
			want: "MOVEA.L #$00002140, A0",
		},
		{
			name: "MOVEA long immediate A1",
			data: []byte{0x22, 0x7C, 0x00, 0x01, 0x00, 0x7C},
			want: "MOVEA.L #$0001007C, A1",
		},
		{
			name: "BEQ short 08",
			data: []byte{0x67, 0x08},
			want: "BEQ.S $000A",
		},
		{
			name: "BEQ short 36",
			data: []byte{0x67, 0x36},
			want: "BEQ.S $0038",
		},
		{
			name: "BSR short 1A",
			data: []byte{0x61, 0x1A},
			want: "BSR.S $001C",
		},
		{
			name: "SUBA long register",
			data: []byte{0x93, 0xC8},
			want: "SUBA.L A0, A1",
		},
		{
			name: "MOVEM register list decode",
			data: []byte{0x4C, 0xDF, 0x0C, 0x04},
			want: "MOVEM.L (A7)+, D2/A2-A3",
		},
		{
			name: "SWAP D0",
			data: []byte{0x48, 0x40},
			want: "SWAP D0",
		},
		{
			name: "BEQ.S short branch 1",
			data: []byte{0x67, 0x06},
			want: "BEQ.S $0008",
		},
		{
			name: "BEQ.S short branch 2",
			data: []byte{0x67, 0x12},
			want: "BEQ.S $0014",
		},
		{
			name: "BEQ.S short branch 3",
			data: []byte{0x67, 0x02},
			want: "BEQ.S $0004",
		},
		{
			name: "BSR.W word branch",
			data: []byte{0x61, 0x00, 0xFE, 0xF0},
			want: "BSR.W $FFFFFEF4",
		},
		{
			name: "BRA.S mnemonic",
			data: []byte{0x60, 0x3C},
			want: "BRA.S $003E",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			instr, err := Decode(tc.data, tc.address)
			if err != nil {
				t.Fatalf("Decode-Fehler: %v", err)
			}
			if got := instr.Assembly(); got != tc.want {
				t.Errorf("Mismatch!\nErwartet: '%s'\nErhalten: '%s'", tc.want, got)
			}
		})
	}
}

func TestDecodeRegressionMaintainsInstructionAlignment(t *testing.T) {
	data := []byte{
		0x20, 0x7C, 0x00, 0x00, 0x21, 0x40, // MOVEA.L #$00002140, A0
		0x4E, 0x71, // NOP
		0xB1, 0xFC, 0x00, 0x00, 0x02, 0x00, // CMPA.L #$00000200, A0
		0x4E, 0x75, // RTS
	}

	instrs, err := DisassembleRange(data, 0)
	if err != nil {
		t.Fatalf("Disassembler-Fehler: %v", err)
	}

	want := []string{
		"MOVEA.L #$00002140, A0",
		"NOP",
		"CMPA.L #$00000200, A0",
		"RTS",
	}

	if len(instrs) != len(want) {
		t.Fatalf("Erwartete %d Instruktionen, erhielt %d", len(want), len(instrs))
	}

	for i, expected := range want {
		if got := instrs[i].Assembly(); got != expected {
			t.Fatalf("Instruktion %d mismatch!\nErwartet: '%s'\nErhalten: '%s'", i, expected, got)
		}
	}
}

func TestDecodeReturnsStructuredMetadataAndExtensionWords(t *testing.T) {
	data := []byte{0x20, 0x7C, 0x00, 0x00, 0x21, 0x40} // MOVEA.L #$00002140, A0

	inst, err := Decode(data, 0x2000)
	if err != nil {
		t.Fatalf("Decode-Fehler: %v", err)
	}

	if inst.Size != 6 {
		t.Fatalf("Erwartete Instruktionsgröße 6, erhielt %d", inst.Size)
	}
	if len(inst.ExtensionWords) != 2 || inst.ExtensionWords[0] != 0x0000 || inst.ExtensionWords[1] != 0x2140 {
		t.Fatalf("Unerwartete Extension-Wörter: %#v", inst.ExtensionWords)
	}
	if inst.Metadata.MnemonicBase != "MOVEA" || inst.Metadata.SizeSuffix != "L" {
		t.Fatalf("Unerwartete Metadaten für Mnemonic: %+v", inst.Metadata)
	}
	if len(inst.Metadata.Operands) != 2 {
		t.Fatalf("Erwartete 2 Operanden, erhielt %d", len(inst.Metadata.Operands))
	}
	src := inst.Metadata.Operands[0]
	if src.Kind != OperandKindEffectiveAddr || src.EffectiveAddress == nil || src.EffectiveAddress.Kind != EAKindImmediate {
		t.Fatalf("Quelloperand wurde nicht als Immediate-EA dekodiert: %+v", src)
	}
	if src.EffectiveAddress.Immediate == nil || src.EffectiveAddress.Immediate.Value != 0x2140 {
		t.Fatalf("Immediate-Metadaten fehlen: %+v", src.EffectiveAddress)
	}
	dst := inst.Metadata.Operands[1]
	if dst.Kind != OperandKindRegister || dst.Register == nil || dst.Register.Kind != RegisterKindAddress || dst.Register.Number != 0 {
		t.Fatalf("Zieloperand wurde nicht als A0 dekodiert: %+v", dst)
	}
	if len(inst.Metadata.ImmediateValues) != 1 || inst.Metadata.ImmediateValues[0].Value != 0x2140 {
		t.Fatalf("Unerwartete Immediate-Liste: %+v", inst.Metadata.ImmediateValues)
	}
}

func TestDecodeReaderAtSupportsStreamingDecode(t *testing.T) {
	data := []byte{0x20, 0x7C, 0x00, 0x00, 0x21, 0x40}

	inst, err := DecodeReaderAt(bytes.NewReader(data), 0)
	if err != nil {
		t.Fatalf("DecodeReaderAt-Fehler: %v", err)
	}
	if got := inst.Assembly(); got != "MOVEA.L #$00002140, A0" {
		t.Fatalf("Unerwartete Assembly: %s", got)
	}
}

func TestDecodeFuncSupportsCallbackReads(t *testing.T) {
	data := []byte{0x67, 0x08} // BEQ.S $000A at address 0
	var calls []uint32

	inst, err := DecodeFunc(func(address uint32, p []byte) (int, error) {
		calls = append(calls, address)
		if int(address) >= len(data) {
			return 0, io.EOF
		}
		n := copy(p, data[address:])
		if n < len(p) {
			return n, io.EOF
		}
		return n, nil
	}, 0)
	if err != nil {
		t.Fatalf("DecodeFunc-Fehler: %v", err)
	}
	if got := inst.Assembly(); got != "BEQ.S $000A" {
		t.Fatalf("Unerwartete Assembly: %s", got)
	}
	if len(calls) == 0 {
		t.Fatal("DecodeFunc hat den Callback nicht verwendet")
	}
}

func TestDecodePartialErrorsReportMissingBytes(t *testing.T) {
	_, err := Decode([]byte{0x4E, 0x72}, 0x1000) // STOP missing immediate word
	if err == nil {
		t.Fatal("Erwartete PartialDecodeError")
	}

	var partial *PartialDecodeError
	if !errors.As(err, &partial) {
		t.Fatalf("Erwartete PartialDecodeError, erhielt %T", err)
	}
	if partial.Missing != 2 || partial.Context != "STOP immediate" {
		t.Fatalf("Unerwartete PartialDecodeError-Daten: %+v", partial)
	}
	if got := partial.Error(); got != "need 2 more byte(s) for STOP immediate at address 00001000" {
		t.Fatalf("Unerwartete Fehlermeldung: %q", got)
	}
}

func TestDecodeSymbolizerFormatsResolvedAddresses(t *testing.T) {
	data := []byte{0x4E, 0xB9, 0x00, 0x00, 0x12, 0x34} // JSR $00001234

	inst, err := DecodeWithOptions(data, 0, DecodeOptions{
		Symbolizer: SymbolizeFunc(func(address uint32) (string, bool) {
			if address == 0x1234 {
				return "_bios_init", true
			}
			return "", false
		}),
	})
	if err != nil {
		t.Fatalf("DecodeWithOptions-Fehler: %v", err)
	}

	if got := inst.Assembly(); got != "JSR _bios_init" {
		t.Fatalf("Unerwartete symbolisierte Assembly: %s", got)
	}
	if inst.Metadata.Operands[0].Text != "$00001234" {
		t.Fatalf("Rohoperand wurde unerwartet überschrieben: %+v", inst.Metadata.Operands[0])
	}
}
