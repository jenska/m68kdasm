package decoders

// This file decides, from the opcode word alone, whether an integer-unit
// instruction (lines 0-E) exists on a CPU, following the operation code map
// and the effective-address categories of the M68000 Family Programmer's
// Reference Manual. FindDecoder consults it before any pattern, so an
// encoding the CPU does not have renders as DC.W instead of as an
// instruction with an operand it cannot take. Line A and the coprocessor
// line F are left to their decoders.

// eaClass is a set of addressing modes, one bit per mode (and per mode-7
// register).
type eaClass uint16

const (
	eaDn eaClass = 1 << iota
	eaAn
	eaInd
	eaPostInc
	eaPreDec
	eaDisp
	eaIndex
	eaAbsW
	eaAbsL
	eaPCDisp
	eaPCIndex
	eaImm
)

const (
	eaAll             = eaDn | eaAn | eaInd | eaPostInc | eaPreDec | eaDisp | eaIndex | eaAbsW | eaAbsL | eaPCDisp | eaPCIndex | eaImm
	eaData            = eaAll &^ eaAn
	eaMemory          = eaAll &^ (eaDn | eaAn)
	eaControl         = eaInd | eaDisp | eaIndex | eaAbsW | eaAbsL | eaPCDisp | eaPCIndex
	eaControlAlt      = eaControl &^ (eaPCDisp | eaPCIndex)
	eaAlterable       = eaDn | eaAn | eaInd | eaPostInc | eaPreDec | eaDisp | eaIndex | eaAbsW | eaAbsL
	eaDataAlterable   = eaAlterable &^ eaAn
	eaMemoryAlterable = eaAlterable &^ (eaDn | eaAn)
)

func eaClassOf(mode, reg uint16) eaClass {
	if mode < 7 {
		return eaClass(1) << mode
	}
	switch reg {
	case 0:
		return eaAbsW
	case 1:
		return eaAbsL
	case 2:
		return eaPCDisp
	case 3:
		return eaPCIndex
	case 4:
		return eaImm
	}
	return 0
}

// eaIn reports whether the opcode's low six bits name an addressing mode in
// set.
func eaIn(op uint16, set eaClass) bool {
	c := eaClassOf((op>>3)&7, op&7)
	return c != 0 && c&set != 0
}

// cpuTier groups the integer ISA levels: the 68010 (and CPU32) add MOVEC,
// MOVES, RTD, BKPT and MOVE from CCR; CPU32 and the 68020+ share the
// "68020-era" subset CPU32 got (long branches, CHK2/CMP2, EXTB, TRAPcc,
// CHK.L, LINK.L, 32-bit MUL/DIV, wider TST/CMPI operands); bit fields,
// CAS/CAS2 and PACK/UNPK are 68020+ only.
type cpuTier struct {
	has010 bool // 68010 additions
	has020 bool // 68020-era additions CPU32 shares
	full20 bool // 68020+ only (not CPU32)
	is020  bool // exactly the 68020 (CALLM/RTM)
}

func tierOf(cpu CPU) cpuTier {
	switch cpu {
	case M68000:
		return cpuTier{}
	case M68010:
		return cpuTier{has010: true}
	case CPU32:
		return cpuTier{has010: true, has020: true}
	case M68020:
		return cpuTier{has010: true, has020: true, full20: true, is020: true}
	default:
		return cpuTier{has010: true, has020: true, full20: true}
	}
}

// ValidIntegerOpcode reports whether op is an integer-unit opcode word that
// exists on cpu. Opcodes on lines A and F always report true; their
// decoders decide.
func ValidIntegerOpcode(op uint16, cpu CPU) bool {
	t := tierOf(cpu)
	size := (op >> 6) & 3
	mode := (op >> 3) & 7
	switch op >> 12 {
	case 0x0:
		return validLine0(op, t)
	case 0x1, 0x2, 0x3:
		dstMode := (op >> 6) & 7
		dstReg := (op >> 9) & 7
		if !eaIn(op, eaAll) || (op>>12 == 1 && eaIn(op, eaAn)) {
			return false
		}
		if dstMode == 1 {
			return op>>12 != 1 // MOVEA.W/L
		}
		c := eaClassOf(dstMode, dstReg)
		return c != 0 && c&eaDataAlterable != 0
	case 0x4:
		return validLine4(op, t)
	case 0x5:
		if size == 3 {
			switch {
			case mode == 1:
				return true // DBcc
			case mode == 7 && op&7 >= 2 && op&7 <= 4:
				return t.has020 // TRAPcc (.W/.L/no operand)
			}
			return eaIn(op, eaDataAlterable) // Scc
		}
		if mode == 1 {
			return size != 0 // ADDQ/SUBQ to An: word or long
		}
		return eaIn(op, eaAlterable)
	case 0x6:
		if op&0xFF == 0xFF {
			return t.has020 // 32-bit displacement; an odd 8-bit one on the 68000/010
		}
		return true
	case 0x7:
		return op&0x0100 == 0
	case 0x8, 0xC:
		return validLine8C(op, t)
	case 0x9, 0xB, 0xD:
		opmode := (op >> 6) & 7
		if opmode == 3 || opmode == 7 { // SUBA/CMPA/ADDA
			return eaIn(op, eaAll)
		}
		if opmode < 3 { // SUB/CMP/ADD <ea>,Dn
			if opmode == 0 && eaIn(op, eaAn) {
				return false
			}
			return eaIn(op, eaAll)
		}
		if op>>12 == 0xB {
			if mode == 1 {
				return true // CMPM
			}
			return eaIn(op, eaDataAlterable) // EOR
		}
		if mode <= 1 {
			return true // SUBX/ADDX
		}
		return eaIn(op, eaMemoryAlterable)
	case 0xE:
		if size != 3 {
			return true // register shifts and rotates
		}
		if op&0x0800 == 0 {
			return eaIn(op, eaMemoryAlterable) // memory shifts and rotates
		}
		if !t.full20 {
			return false
		}
		// Bit fields: BFTST/BFEXTU/BFEXTS/BFFFO read, the rest write.
		switch (op >> 8) & 7 {
		case 0, 1, 3, 5:
			return eaIn(op, eaDn|eaControl)
		default:
			return eaIn(op, eaDn|eaControlAlt)
		}
	}
	return true // lines A and F
}

func validLine0(op uint16, t cpuTier) bool {
	size := (op >> 6) & 3
	mode := (op >> 3) & 7
	switch op {
	case 0x003C, 0x007C, 0x023C, 0x027C, 0x0A3C, 0x0A7C: // ORI/ANDI/EORI to CCR/SR
		return true
	case 0x0CFC, 0x0EFC: // CAS2.W/.L
		return t.full20
	}
	if op&0x0100 != 0 {
		if mode == 1 {
			return true // MOVEP
		}
		if size == 0 { // BTST Dn,<ea>
			return eaIn(op, eaData)
		}
		return eaIn(op, eaDataAlterable)
	}
	field := (op >> 9) & 7
	if field == 4 { // static bit: bits 7-6 select BTST/BCHG/BCLR/BSET
		if size == 0 {
			return eaIn(op, eaData&^eaImm)
		}
		return eaIn(op, eaDataAlterable)
	}
	if size == 3 {
		switch field {
		case 0, 1, 2: // CHK2/CMP2 .B/.W/.L
			return t.has020 && eaIn(op, eaControl)
		case 3: // RTM / CALLM
			if !t.is020 {
				return false
			}
			if mode <= 1 {
				return true
			}
			return eaIn(op, eaControl)
		case 5, 6, 7: // CAS .B/.W/.L
			return t.full20 && eaIn(op, eaMemoryAlterable)
		}
		return false
	}
	switch field {
	case 0, 1, 2, 3, 5: // ORI ANDI SUBI ADDI EORI
		return eaIn(op, eaDataAlterable)
	case 6: // CMPI
		if t.has020 {
			return eaIn(op, eaData&^eaImm)
		}
		return eaIn(op, eaDataAlterable)
	case 7: // MOVES
		return t.has010 && eaIn(op, eaMemoryAlterable)
	}
	return false
}

func validLine4(op uint16, t cpuTier) bool {
	size := (op >> 6) & 3
	mode := (op >> 3) & 7
	if op&0x0100 != 0 {
		switch size {
		case 3:
			if op == 0x49C0|op&7 && mode == 0 {
				return t.has020 // EXTB.L
			}
			return eaIn(op, eaControl) // LEA
		case 2:
			return eaIn(op, eaData) // CHK.W
		case 0:
			return t.has020 && eaIn(op, eaData) // CHK.L
		}
		return false
	}
	switch (op >> 8) & 0xF {
	case 0x0: // NEGX / MOVE from SR
		return eaIn(op, eaDataAlterable)
	case 0x2: // CLR / MOVE from CCR (68010+)
		if size == 3 {
			return t.has010 && eaIn(op, eaDataAlterable)
		}
		return eaIn(op, eaDataAlterable)
	case 0x4, 0x6: // NEG, NOT / MOVE to CCR, MOVE to SR
		if size == 3 {
			return eaIn(op, eaData)
		}
		return eaIn(op, eaDataAlterable)
	case 0x8:
		switch size {
		case 0:
			if mode == 1 {
				return t.has020 // LINK.L
			}
			return eaIn(op, eaDataAlterable) // NBCD
		case 1:
			switch mode {
			case 0:
				return true // SWAP
			case 1:
				return t.has010 // BKPT
			}
			return eaIn(op, eaControl) // PEA
		default:
			if mode == 0 {
				return true // EXT.W/EXT.L
			}
			return eaIn(op, eaControlAlt|eaPreDec) // MOVEM registers to memory
		}
	case 0xA:
		switch {
		case op == 0x4AFC:
			return true // ILLEGAL
		case op == 0x4AFA:
			return t.has020 && !t.full20 // BGND, CPU32 only
		case size == 3:
			return eaIn(op, eaDataAlterable) // TAS
		case t.has020:
			if size == 0 && eaIn(op, eaAn) {
				return false
			}
			return eaIn(op, eaAll) // TST
		}
		return eaIn(op, eaDataAlterable) // TST
	case 0xC:
		if size >= 2 {
			return eaIn(op, eaControl|eaPostInc) // MOVEM memory to registers
		}
		return t.has020 && eaIn(op, eaData) // MULU.L/MULS.L, DIVU.L/DIVS.L
	case 0xE:
		switch {
		case op >= 0x4E40 && op <= 0x4E6F: // TRAP, LINK, UNLK, MOVE USP
			return true
		case op == 0x4E70, op == 0x4E71, op == 0x4E72, op == 0x4E73, op == 0x4E75, op == 0x4E76, op == 0x4E77:
			return true
		case op == 0x4E74, op == 0x4E7A, op == 0x4E7B:
			return t.has010 // RTD, MOVEC
		case size == 2, size == 3:
			return eaIn(op, eaControl) // JSR, JMP
		}
	}
	return false
}

func validLine8C(op uint16, t cpuTier) bool {
	mode := (op >> 3) & 7
	opmode := (op >> 6) & 7
	line := op >> 12
	if opmode == 3 || opmode == 7 { // DIVU/DIVS, MULU/MULS
		return eaIn(op, eaData)
	}
	if opmode >= 4 && mode <= 1 {
		switch {
		case opmode == 4:
			return true // SBCD / ABCD
		case line == 0xC:
			// EXG Dx,Dy (01000), Ax,Ay (01001), Dx,Ay (10001)
			return opmode == 5 || (opmode == 6 && mode == 1)
		}
		return t.full20 // PACK / UNPK
	}
	if opmode >= 4 {
		return eaIn(op, eaMemoryAlterable) // OR/AND Dn,<ea>
	}
	return eaIn(op, eaData) // OR/AND <ea>,Dn
}

// saveRestoreEAValid checks a coprocessor state-frame save (control
// alterable or predecrement) or restore (control or postincrement) operand.
func saveRestoreEAValid(op uint16, save bool) bool {
	if save {
		return eaIn(op, eaControlAlt|eaPreDec)
	}
	return eaIn(op, eaControl|eaPostInc)
}
