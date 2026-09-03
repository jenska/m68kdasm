package decoders

import (
	"encoding/binary"
	"fmt"
)

var sizeNames = [...]string{"B", "W", "L", "?"}

// DecodeUnknown builds the "DC.W" pseudo-instruction used when no decoder
// matches the opcode.
func DecodeUnknown(data []byte, address uint32, opcode uint16) *Instruction {
	text := fmt.Sprintf("$%04X", opcode)
	inst := &Instruction{Address: address, Opcode: opcode}
	setInstruction(data, inst, 2, "DC.W", text, immediateOperand(text, uint32(opcode), 2))
	return inst
}

// decodeNOP - No Operation (exact opcode: 0x4E71)
func decodeNOP(data []byte, opcode uint16, inst *Instruction) error {
	setInstruction(data, inst, 2, "NOP", "")
	return nil
}

// decodeRTS - Return from Subroutine (exact opcode: 0x4E75)
func decodeRTS(data []byte, opcode uint16, inst *Instruction) error {
	setInstruction(data, inst, 2, "RTS", "")
	return nil
}

// getSizeString converts a 68000 size field (bits 6-7) to string.
// Maps: 0=B (byte), 1=W (word), 2=L (long), 3=? (undefined)
func getSizeString(size uint16) string {
	return sizeNames[size&0x3]
}

// setInstructionSize sets the instruction's Size and Bytes fields
func setInstructionSize(data []byte, inst *Instruction, offset int) {
	inst.Size = uint32(offset)
	if len(data) >= offset {
		inst.Bytes = data[:offset]
	}
	inst.ExtensionWords = collectExtensionWords(inst.Bytes)
}

func setInstruction(data []byte, inst *Instruction, size int, mnemonic, operands string, structuredOperands ...Operand) {
	inst.Mnemonic = mnemonic
	inst.Operands = operands
	setInstructionSize(data, inst, size)
	populateMetadata(inst, mnemonic, structuredOperands)
}

// decodeUnaryEA decodes a single effective-address operand (JSR, JMP, PEA …).
func decodeUnaryEA(mnemonic string, data []byte, opcode uint16, inst *Instruction) error {
	mode := uint8((opcode >> 3) & 0x7)
	reg := uint8(opcode & 0x7)
	operand, offset, meta, err := decodeEA(data, inst.Address, 2, mode, reg)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, mnemonic, operand, meta)
	return nil
}

func decodeEA(data []byte, address uint32, offset int, mode, reg uint8) (string, int, Operand, error) {
	return decodeEAWithSize(data, address, offset, mode, reg, 2)
}

func decodeEAWithSize(data []byte, address uint32, offset int, mode, reg uint8, operandSize int) (string, int, Operand, error) {
	operand, extraWords, structured, err := decodeAddressingMode(data[offset:], mode, reg, operandSize)
	if err != nil {
		return "", offset, Operand{}, err
	}
	nextOffset := offset + extraWords*2
	return operand, nextOffset, resolveEffectiveAddress(address, nextOffset, structured), nil
}

func decodeDirectedBinaryOp(mnemonic string, data []byte, opcode uint16, inst *Instruction) error {
	direction := (opcode >> 8) & 0x1
	sizeBits := (opcode >> 6) & 0x3
	sizeStr := getSizeString(sizeBits)
	operandSize, err := operandSize(sizeBits, mnemonic)
	if err != nil {
		return err
	}
	dstReg := uint8((opcode >> 9) & 0x7)
	srcMode := uint8((opcode >> 3) & 0x7)
	srcReg := uint8(opcode & 0x7)

	srcOperand, offset, srcMeta, err := decodeEAWithSize(data, inst.Address, 2, srcMode, srcReg, operandSize)
	if err != nil {
		return err
	}

	dstMeta := registerOperand(RegisterKindData, dstReg)
	dstText := fmt.Sprintf("D%d", dstReg)
	text, first, second := srcOperand+", "+dstText, srcMeta, dstMeta
	if direction != 0 {
		text, first, second = dstText+", "+srcOperand, dstMeta, srcMeta
	}
	setInstruction(data, inst, offset, mnemonic+"."+sizeStr, text, first, second)
	return nil
}

// decodeAddressRegisterOp decodes the "<mnemonic>A" form where the destination
// is an address register and the size comes from the op-mode field
// (op-mode 3 = word, op-mode 7 = long).
func decodeAddressRegisterOp(mnemonic string, data []byte, opcode uint16, inst *Instruction) error {
	dstReg := uint8((opcode >> 9) & 0x7)
	srcMode := uint8((opcode >> 3) & 0x7)
	srcReg := uint8(opcode & 0x7)

	sizeStr, sizeBytes := "W", 2
	if (opcode>>6)&0x7 == 7 {
		sizeStr, sizeBytes = "L", 4
	}

	srcOperand, offset, srcMeta, err := decodeEAWithSize(data, inst.Address, 2, srcMode, srcReg, sizeBytes)
	if err != nil {
		return err
	}
	setInstruction(data, inst, offset, mnemonic+"A."+sizeStr, fmt.Sprintf("%s, A%d", srcOperand, dstReg), srcMeta, registerOperand(RegisterKindAddress, dstReg))
	return nil
}

func operandSize(size uint16, mnemonic string) (int, error) {
	switch size {
	case 0:
		return 1, nil
	case 1:
		return 2, nil
	case 2:
		return 4, nil
	default:
		return 0, fmt.Errorf("unknown %s size: %d", mnemonic, size)
	}
}

func immediateSpec(size uint16, longImmediate bool, mnemonic string) (string, int, error) {
	switch size {
	case 0:
		return "B", 2, nil
	case 1:
		return "W", 2, nil
	case 2:
		if longImmediate {
			return "L", 4, nil
		}
		return "L", 2, nil
	default:
		return "", 0, fmt.Errorf("unknown %s size: %d", mnemonic, size)
	}
}

func readImmediate(data []byte, offset, size int, mnemonic string) (uint32, int, error) {
	if err := requireLength(data, offset+size, fmt.Sprintf("%s immediate", mnemonic)); err != nil {
		return 0, offset, err
	}

	switch size {
	case 2:
		return uint32(binary.BigEndian.Uint16(data[offset : offset+2])), offset + 2, nil
	case 4:
		return binary.BigEndian.Uint32(data[offset : offset+4]), offset + 4, nil
	default:
		return 0, offset, fmt.Errorf("unsupported immediate size for %s: %d", mnemonic, size)
	}
}

type NeedMoreError struct {
	Missing int
	Context string
}

func (e *NeedMoreError) Error() string {
	return fmt.Sprintf("need %d more byte(s) for %s", e.Missing, e.Context)
}

func requireLength(data []byte, need int, context string) error {
	if len(data) >= need {
		return nil
	}
	return &NeedMoreError{
		Missing: need - len(data),
		Context: context,
	}
}

func collectExtensionWords(data []byte) []uint16 {
	if len(data) <= 2 {
		return nil
	}
	extBytes := data[2:]
	words := make([]uint16, 0, len(extBytes)/2)
	for i := 0; i+1 < len(extBytes); i += 2 {
		words = append(words, binary.BigEndian.Uint16(extBytes[i:i+2]))
	}
	return words
}

func registerOperand(kind RegisterKind, number uint8) Operand {
	return Operand{
		Text: fmt.Sprintf("%s%d", registerPrefix(kind), number),
		Kind: OperandKindRegister,
		Register: &Register{
			Kind:   kind,
			Number: number,
		},
	}
}

func immediateOperand(text string, value uint32, size int) Operand {
	imm := ImmediateValue{
		Value:  value,
		Signed: signedImmediateValue(value, size),
		Size:   uint8(size),
	}
	return Operand{
		Text:      text,
		Kind:      OperandKindImmediate,
		Immediate: &imm,
	}
}

// addrIndirectOperand builds an address-register indirect operand (optionally
// with pre-decrement or post-increment) as used by ABCD/SBCD and CMPM.
func addrIndirectOperand(kind EffectiveAddressKind, reg uint8, text string) Operand {
	return effectiveAddressOperand(text, EffectiveAddress{
		Kind:     kind,
		Base:     &Register{Kind: RegisterKindAddress, Number: reg},
		Register: reg,
	})
}

func effectiveAddressOperand(text string, ea EffectiveAddress) Operand {
	return Operand{
		Text:             text,
		Kind:             OperandKindEffectiveAddr,
		EffectiveAddress: &ea,
	}
}

func registerListOperand(text string, registers []string) Operand {
	return Operand{
		Text:         text,
		Kind:         OperandKindRegisterList,
		RegisterList: append([]string(nil), registers...),
	}
}

func branchOperand(text string, target uint32) Operand {
	return Operand{
		Text:         text,
		Kind:         OperandKindBranchTarget,
		BranchTarget: uint32Ptr(target),
	}
}

func uint32Ptr(v uint32) *uint32 {
	return &v
}

func int32Ptr(v int32) *int32 {
	return &v
}

func immediatePtr(value uint32, size int) *ImmediateValue {
	return &ImmediateValue{
		Value:  value,
		Signed: signedImmediateValue(value, size),
		Size:   uint8(size),
	}
}

func signedImmediateValue(value uint32, size int) int32 {
	switch size {
	case 1:
		return int32(int8(value))
	case 2:
		return int32(int16(value))
	case 4:
		return int32(value)
	default:
		return int32(value)
	}
}

func resolveEffectiveAddress(address uint32, nextOffset int, operand Operand) Operand {
	if operand.EffectiveAddress == nil {
		return operand
	}

	ea := *operand.EffectiveAddress
	switch ea.Kind {
	case EAKindPCDisplacement, EAKindPCIndex:
		if ea.Displacement != nil {
			target := uint32(int32(address) + int32(nextOffset) + *ea.Displacement)
			ea.ResolvedAddress = uint32Ptr(target)
		}
	}

	operand.EffectiveAddress = &ea
	return operand
}

func registerPrefix(kind RegisterKind) string {
	switch kind {
	case RegisterKindAddress:
		return "A"
	case RegisterKindPC:
		return "PC"
	default:
		return "D"
	}
}
