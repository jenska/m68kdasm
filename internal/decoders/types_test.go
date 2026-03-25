package decoders

import (
	"fmt"
	"testing"
)

func TestFindDecoderMatchesOpcodeTable(t *testing.T) {
	for opcode := 0; opcode <= 0xFFFF; opcode++ {
		op := uint16(opcode)
		got := FindDecoder(op)
		want := findDecoderLinear(op)
		if fmt.Sprintf("%p", got) != fmt.Sprintf("%p", want) {
			t.Fatalf("decoder mismatch for opcode %04X: FindDecoder=%p OpcodeTable=%p", op, got, want)
		}
	}
}

func findDecoderLinear(opcode uint16) OpcodeDecoder {
	for _, pattern := range OpcodeTable {
		if (opcode & pattern.Mask) == pattern.Value {
			return pattern.Decoder
		}
	}
	return nil
}
