package m68kdasm

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"testing"

	asm "github.com/jenska/m68kasm"
	"github.com/jenska/m68kdasm/internal/decoders"
)

// These tests hold the disassembler to the assembler: every instruction it
// decodes must assemble back, with github.com/jenska/m68kasm for the same
// CPU, to exactly the bytes it was decoded from; and every opcode word the
// CPU has must decode to an instruction rather than DC.W.

type roundTripConfig struct {
	name   string
	opts   DecodeOptions
	target asm.Target
}

var roundTripConfigs = []roundTripConfig{
	{"68000", DecodeOptions{CPU: M68000}, asm.Target{CPU: asm.CPU68000}},
	{"68010", DecodeOptions{CPU: M68010}, asm.Target{CPU: asm.CPU68010}},
	{"cpu32", DecodeOptions{CPU: CPU32}, asm.Target{CPU: asm.CPU32}},
	{"68020+fpu+mmu", DecodeOptions{CPU: M68020, FPU: true, MMU: true}, asm.Target{CPU: asm.CPU68020, Features: asm.FeatFPU | asm.FeatFPUFull | asm.FeatPMMU}},
	{"68030+fpu+mmu", DecodeOptions{CPU: M68030, FPU: true, MMU: true}, asm.Target{CPU: asm.CPU68030, Features: asm.FeatFPU | asm.FeatFPUFull | asm.FeatPMMU}},
	// The 68040/68060 trap and software-emulate the transcendental FPU
	// instructions; the disassembler still decodes them, so the assembler
	// side enables the full FPU set.
	{"68040+fpu+mmu", DecodeOptions{CPU: M68040, FPU: true, MMU: true}, asm.Target{CPU: asm.CPU68040, Features: asm.FeatFPU | asm.FeatFPUFull | asm.FeatPMMU}},
	{"68060+fpu+mmu", DecodeOptions{CPU: M68060, FPU: true, MMU: true}, asm.Target{CPU: asm.CPU68060, Features: asm.FeatFPU | asm.FeatFPUFull | asm.FeatPMMU}},
}

const roundTripBase = 0x1000

// Extension words placed after the opcode: zeros are a valid extension for
// every instruction; the second pattern exercises non-zero fields.
var roundTripExtensions = [][]byte{
	make([]byte, 14),
	{0x00, 0x12, 0x00, 0x34, 0x00, 0x56, 0x00, 0x78, 0x00, 0x9A, 0x00, 0xBC, 0x00, 0xDE},
}

type roundTripFailures map[string][]string

func (f roundTripFailures) add(kind, mnemonic, detail string) {
	key := kind + " " + mnemonic
	f[key] = append(f[key], detail)
}

func (f roundTripFailures) report(t *testing.T, config string) {
	t.Helper()
	keys := make([]string, 0, len(f))
	for k := range f {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		examples := f[k]
		shown := examples
		if len(shown) > 3 {
			shown = shown[:3]
		}
		t.Errorf("%s: %s (%d): %s", config, k, len(examples), strings.Join(shown, "; "))
	}
}

func checkRoundTrip(t *testing.T, cfg roundTripConfig, code []byte, failures roundTripFailures) bool {
	t.Helper()
	inst, err := DecodeWithOptions(code, roundTripBase, cfg.opts)
	if err != nil {
		failures.add("DECODE-ERROR", fmt.Sprintf("%04x", code[:2]), err.Error())
		return false
	}
	if strings.HasPrefix(inst.Mnemonic, "DC.") {
		return false
	}
	text := inst.Assembly()
	mnemonic, _, _ := strings.Cut(inst.Mnemonic, ".")
	got, err := asm.AssembleStringWithOptions(fmt.Sprintf("\tORG $%X\n\t%s\n", roundTripBase, text), asm.ParseOptions{Target: cfg.target})
	switch {
	case err != nil:
		msg, _, _ := strings.Cut(err.Error(), "\n")
		failures.add("REJECTED", mnemonic, fmt.Sprintf("%q from % x: %s", text, inst.Bytes, msg))
	case !bytes.Equal(got, inst.Bytes):
		failures.add("MISMATCH", mnemonic, fmt.Sprintf("%q from % x assembles to % x", text, inst.Bytes, got))
	}
	return true
}

// TestRoundTripFirstWord decodes every opcode word of every CPU.
func TestRoundTripFirstWord(t *testing.T) {
	for _, cfg := range roundTripConfigs {
		t.Run(cfg.name, func(t *testing.T) {
			failures := roundTripFailures{}
			for op := 0; op < 0x10000; op++ {
				decodedAny := false
				for _, ext := range roundTripExtensions {
					code := append([]byte{byte(op >> 8), byte(op)}, ext...)
					if checkRoundTrip(t, cfg, code, failures) {
						decodedAny = true
					}
				}
				line := op >> 12
				if !decodedAny && line != 0xA && line != 0xF && decoders.ValidIntegerOpcode(uint16(op), cfg.opts.CPU) {
					failures.add("NOT-DECODED", fmt.Sprintf("line %X", line), fmt.Sprintf("%04x", op))
				}
			}
			failures.report(t, cfg.name)
		})
	}
}

// secondWordOpcodes are opcode words whose instruction is selected or
// completed by an extension word: FPU and PMMU command words, the 68020
// long multiply/divide, bit fields, MOVEC/MOVES, CAS/CAS2 and CHK2/CMP2.
var secondWordOpcodes = []uint16{
	0xF200, 0xF210, 0xF000, 0xF010,
	0x4C00, 0x4C10, 0x4C40, 0x4C50,
	0xE8C0, 0xE9D0, 0xEAD0, 0xEFC0,
	0x4E7A, 0x4E7B, 0x0E10, 0x0E90,
	0x0AD0, 0x0CFC, 0x00D0, 0x04D0,
}

// TestRoundTripSecondWord combines those opcodes with every second word.
func TestRoundTripSecondWord(t *testing.T) {
	if testing.Short() {
		t.Skip("sweeps every extension word")
	}
	for _, cfg := range roundTripConfigs[3:] {
		t.Run(cfg.name, func(t *testing.T) {
			failures := roundTripFailures{}
			for _, op := range secondWordOpcodes {
				for w := 0; w < 0x10000; w++ {
					code := []byte{byte(op >> 8), byte(op), byte(w >> 8), byte(w), 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
					checkRoundTrip(t, cfg, code, failures)
				}
			}
			failures.report(t, cfg.name)
		})
	}
}
