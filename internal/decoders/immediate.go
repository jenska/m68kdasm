package decoders

import "fmt"

func decodeMOVEQ(data []byte, opcode uint16, inst *Instruction) error {
	dstReg := uint8((opcode >> 9) & 0x7)
	immediate := int8(opcode & 0xFF)
	inst.Mnemonic = "MOVEQ"
	immStr := formatImmediateForMOVEQ(int32(immediate))
	inst.Operands = fmt.Sprintf("#%s, D%d", immStr, dstReg)
	inst.Size = 2
	if len(data) >= 2 {
		inst.Bytes = data[:2]
	}
	return nil
}
