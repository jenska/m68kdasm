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
		"ADD.W D1, D0",
		"MOVE.W D0, D1",
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
