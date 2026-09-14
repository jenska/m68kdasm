package decoders

import (
	"fmt"
	"testing"
)

var allTestCPUs = []CPU{M68000, M68010, CPU32, M68020, M68030, M68040, M68060}

func TestFindDecoderMatchesOpcodeTable(t *testing.T) {
	for _, cpu := range allTestCPUs {
		for _, fpu := range []bool{false, true} {
			for opcode := 0; opcode <= 0xFFFF; opcode++ {
				op := uint16(opcode)
				got := FindDecoder(op, cpu, fpu)
				want := findDecoderLinear(op, cpu, fpu)
				if fmt.Sprintf("%p", got) != fmt.Sprintf("%p", want) {
					t.Fatalf("decoder mismatch for opcode %04X on cpu %v fpu %v: FindDecoder=%p OpcodeTable=%p", op, cpu, fpu, got, want)
				}
			}
		}
	}
}

func findDecoderLinear(opcode uint16, cpu CPU, fpu bool) OpcodeDecoder {
	bit := cpuBit(cpu)
	for _, pattern := range OpcodeTable {
		if (opcode & pattern.Mask) != pattern.Value {
			continue
		}
		if pattern.CPUs&bit == 0 {
			continue
		}
		if pattern.RequiresFPU && !fpu {
			continue
		}
		return pattern.Decoder
	}
	return nil
}
