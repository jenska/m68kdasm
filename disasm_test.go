package m68kdasm

import (
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
		"MOVEQ #$7F, D0",
		"MOVE.B D1, D0",
		"MOVE.W D0, D1",
		"MOVE.L D2, D3",
		"OR.B D0, (A1)",
		"SUB.W D2, (A3)",
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
