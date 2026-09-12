package decoders

// CPU identifies a target member of the 68k family. The zero value, M68000,
// preserves plain 68000 decoding behavior.
type CPU uint8

const (
	M68000 CPU = iota
	M68010
	CPU32
	M68020
	M68030
	M68040
	M68060
)

// cpuSet is a bitmask of CPUs an opcode pattern is valid on. The 68k family
// is not a strict "newer implies older" chain (CPU32 branches off the 68010
// core with its own extensions and a reduced addressing-mode set; 68040
// removes CALLM/RTM that 68020/68030 have), so availability is tracked
// explicitly per pattern rather than via an ordinal comparison.
type cpuSet uint8

const (
	cpu000 cpuSet = 1 << iota
	cpu010
	cpu32
	cpu020
	cpu030
	cpu040
	cpu060

	cpuAll     = cpu000 | cpu010 | cpu32 | cpu020 | cpu030 | cpu040 | cpu060
	cpu010up   = cpu010 | cpu32 | cpu020 | cpu030 | cpu040 | cpu060 // 68010 and later, not plain 68000
	cpu020up   = cpu020 | cpu030 | cpu040 | cpu060
	cpu020_030 = cpu020 | cpu030 // e.g. CALLM/RTM, removed starting with 68040
)

func cpuBit(cpu CPU) cpuSet {
	switch cpu {
	case M68000:
		return cpu000
	case M68010:
		return cpu010
	case CPU32:
		return cpu32
	case M68020:
		return cpu020
	case M68030:
		return cpu030
	case M68040:
		return cpu040
	case M68060:
		return cpu060
	default:
		return cpu000
	}
}
