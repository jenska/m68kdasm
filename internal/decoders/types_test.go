package decoders

import (
	"fmt"
	"testing"
)

var allTestCPUs = []CPU{M68000, M68010, CPU32, M68020, M68030, M68040, M68060}

func TestFindDecoderMatchesOpcodeTable(t *testing.T) {
	for _, cpu := range allTestCPUs {
		for opcode := 0; opcode <= 0xFFFF; opcode++ {
			op := uint16(opcode)
			got := FindDecoder(op, cpu)
			want := findDecoderLinear(op, cpu)
			if fmt.Sprintf("%p", got) != fmt.Sprintf("%p", want) {
				t.Fatalf("decoder mismatch for opcode %04X on cpu %v: FindDecoder=%p OpcodeTable=%p", op, cpu, got, want)
			}
		}
	}
}

func findDecoderLinear(opcode uint16, cpu CPU) OpcodeDecoder {
	bit := cpuBit(cpu)
	for _, pattern := range OpcodeTable {
		if (opcode&pattern.Mask) == pattern.Value && pattern.CPUs&bit != 0 {
			return pattern.Decoder
		}
	}
	return nil
}
