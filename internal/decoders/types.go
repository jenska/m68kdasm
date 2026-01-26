package decoders

// Instruction represents a single disassembled instruction.
// This mirrors the type from m68kdasm to avoid circular imports.
type Instruction struct {
	Address  uint32
	Opcode   uint16
	Mnemonic string
	Operands string
	Size     uint32 // Size in bytes
	Bytes    []byte // Raw instruction data
}

// OpcodeDecoder is the type for decoder functions
type OpcodeDecoder func(data []byte, opcode uint16, inst *Instruction) error

// OpcodePattern defines a pattern for opcode recognition
type OpcodePattern struct {
	Mask    uint16        // Bit mask for recognition
	Value   uint16        // Expected value after masking
	Decoder OpcodeDecoder // Decoder function
}

// OpcodeTable is the jump table for opcode decoding
var OpcodeTable = []OpcodePattern{
	// Exact matches (highest priority)
	{Mask: 0xFFFF, Value: 0x4E71, Decoder: decodeNOP},
	{Mask: 0xFFFF, Value: 0x4E75, Decoder: decodeRTS},

	// Pattern matches (with bit masks)
	{Mask: 0xFB80, Value: 0x4880, Decoder: decodeMOVEM}, // MOVEM Reg→Mem
	{Mask: 0xFB80, Value: 0x4C80, Decoder: decodeMOVEM}, // MOVEM Mem→Reg

	{Mask: 0xFF00, Value: 0x4200, Decoder: decodeCLR},  // CLR
	{Mask: 0xFF00, Value: 0x4400, Decoder: decodeNEG},  // NEG
	{Mask: 0xFF00, Value: 0x4000, Decoder: decodeNEGX}, // NEGX
	{Mask: 0xFF00, Value: 0x4600, Decoder: decodeNOT},  // NOT
	{Mask: 0xFF00, Value: 0x4A00, Decoder: decodeTST},  // TST

	{Mask: 0xF100, Value: 0x6000, Decoder: decodeBRA}, // BRA (and conditional branches)
	{Mask: 0xF1C0, Value: 0x4E80, Decoder: decodeJSR}, // JSR
	{Mask: 0xF1C0, Value: 0x4EC0, Decoder: decodeJMP}, // JMP
	{Mask: 0xF1C0, Value: 0x41C0, Decoder: decodeLEA}, // LEA
	{Mask: 0xFFC0, Value: 0x4840, Decoder: decodePEA}, // PEA

	// MUL/DIV instructions (must come before generic AND/OR patterns)
	{Mask: 0xF1C0, Value: 0xC0C0, Decoder: decodeMULU}, // MULU
	{Mask: 0xF1C0, Value: 0xC1C0, Decoder: decodeMULS}, // MULS
	{Mask: 0xF1C0, Value: 0x80C0, Decoder: decodeDIVU}, // DIVU
	{Mask: 0xF1C0, Value: 0x81C0, Decoder: decodeDIVS}, // DIVS

	// Bit operations (must come before generic patterns)
	{Mask: 0xF1C0, Value: 0x0800, Decoder: decodeBTST}, // BTST
	{Mask: 0xF1C0, Value: 0x0840, Decoder: decodeBCHG}, // BCHG
	{Mask: 0xF1C0, Value: 0x0880, Decoder: decodeBCLR}, // BCLR
	{Mask: 0xF1C0, Value: 0x08C0, Decoder: decodeBSET}, // BSET

	// BCD (Binary Coded Decimal) instructions (must come before generic AND/OR)
	{Mask: 0xF1F0, Value: 0xC100, Decoder: decodeABCD}, // ABCD
	{Mask: 0xF1F0, Value: 0x8100, Decoder: decodeSBCD}, // SBCD

	// Immediate instructions (must come before generic register instructions)
	{Mask: 0xFF00, Value: 0x0600, Decoder: decodeADDI},  // ADDI
	{Mask: 0xFF00, Value: 0x0400, Decoder: decodeSUBI},  // SUBI
	{Mask: 0xFF00, Value: 0x0200, Decoder: decodeANDI},  // ANDI
	{Mask: 0xFF00, Value: 0x0000, Decoder: decodeORI},   // ORI
	{Mask: 0xFF00, Value: 0x0A00, Decoder: decodeEORI},  // EORI
	{Mask: 0xFF00, Value: 0x0C00, Decoder: decodeCMPI},  // CMPI
	{Mask: 0xF100, Value: 0x7000, Decoder: decodeMOVEQ}, // MOVEQ

	{Mask: 0xF000, Value: 0x1000, Decoder: decodeMOVE}, // MOVE.B
	{Mask: 0xF000, Value: 0x2000, Decoder: decodeMOVE}, // MOVE.L
	{Mask: 0xF000, Value: 0x3000, Decoder: decodeMOVE}, // MOVE.W

	{Mask: 0xF000, Value: 0x8000, Decoder: decodeOR},  // OR
	{Mask: 0xF000, Value: 0x9000, Decoder: decodeSUB}, // SUB
	{Mask: 0xF000, Value: 0xB000, Decoder: decodeCMP}, // CMP
	{Mask: 0xF000, Value: 0xC000, Decoder: decodeAND}, // AND
	{Mask: 0xF000, Value: 0xD000, Decoder: decodeADD}, // ADD

	// Shift/Rotate instructions (0xE000-0xE7FF)
	{Mask: 0xF000, Value: 0xE000, Decoder: decodeShiftRotate}, // All ASL/ASR/LSL/LSR/ROL/ROR/ROXL/ROXR
}
