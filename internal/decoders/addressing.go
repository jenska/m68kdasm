package decoders

import (
	"encoding/binary"
	"fmt"
)

// decodeAddressingMode decodes an addressing mode and returns the formatted string.
// operandSize is the logical operand width in bytes so immediate operands consume
// the right number of extension words.
// Returns: (formatted string, extra words needed, structured operand, error)
func decodeAddressingMode(data []byte, mode, reg uint8, operandSize int) (string, int, Operand, error) {
	switch mode {
	case 0: // Data Register Direct
		text := fmt.Sprintf("D%d", reg)
		return text, 0, effectiveAddressOperand(text, EffectiveAddress{
			Kind:     EAKindDataRegisterDirect,
			Mode:     mode,
			Register: reg,
			Base: &Register{
				Kind:   RegisterKindData,
				Number: reg,
			},
		}), nil

	case 1: // Address Register Direct
		text := fmt.Sprintf("A%d", reg)
		return text, 0, effectiveAddressOperand(text, EffectiveAddress{
			Kind:     EAKindAddressRegisterDirect,
			Mode:     mode,
			Register: reg,
			Base: &Register{
				Kind:   RegisterKindAddress,
				Number: reg,
			},
		}), nil

	case 2: // Address Register Indirect
		text := fmt.Sprintf("(A%d)", reg)
		return text, 0, effectiveAddressOperand(text, EffectiveAddress{
			Kind:     EAKindAddressIndirect,
			Mode:     mode,
			Register: reg,
			Base: &Register{
				Kind:   RegisterKindAddress,
				Number: reg,
			},
		}), nil

	case 3: // Address Register Indirect with Post-increment
		text := fmt.Sprintf("(A%d)+", reg)
		return text, 0, effectiveAddressOperand(text, EffectiveAddress{
			Kind:     EAKindPostIncrement,
			Mode:     mode,
			Register: reg,
			Base: &Register{
				Kind:   RegisterKindAddress,
				Number: reg,
			},
		}), nil

	case 4: // Address Register Indirect with Pre-decrement
		text := fmt.Sprintf("-(A%d)", reg)
		return text, 0, effectiveAddressOperand(text, EffectiveAddress{
			Kind:     EAKindPreDecrement,
			Mode:     mode,
			Register: reg,
			Base: &Register{
				Kind:   RegisterKindAddress,
				Number: reg,
			},
		}), nil

	case 5: // Address Register Indirect with Displacement
		if err := requireLength(data, 2, "displacement"); err != nil {
			return "", 0, Operand{}, err
		}
		displacement := int16(binary.BigEndian.Uint16(data[:2]))
		text := fmt.Sprintf("(%d,A%d)", displacement, reg)
		return text, 1, effectiveAddressOperand(text, EffectiveAddress{
			Kind:         EAKindDisplacement,
			Mode:         mode,
			Register:     reg,
			Base:         &Register{Kind: RegisterKindAddress, Number: reg},
			Displacement: int32Ptr(int32(displacement)),
		}), nil

	case 6: // Address Register Indirect with Index
		if err := requireLength(data, 2, "index extension word"); err != nil {
			return "", 0, Operand{}, err
		}
		indexWord := binary.BigEndian.Uint16(data[:2])
		indexType, indexReg, indexSize, displacement := decodeIndexWord(indexWord)
		text := fmt.Sprintf("(%d,A%d,%s%d.%c)", displacement, reg, indexType, indexReg, indexSize)
		return text, 1, effectiveAddressOperand(text, EffectiveAddress{
			Kind:         EAKindIndex,
			Mode:         mode,
			Register:     reg,
			Base:         &Register{Kind: RegisterKindAddress, Number: reg},
			Displacement: int32Ptr(int32(displacement)),
			Index: &IndexRegister{
				Register: Register{Kind: parseIndexRegisterKind(indexType), Number: indexReg},
				Size:     string(indexSize),
			},
		}), nil

	case 7:
		// Special cases based on register field
		switch reg {
		case 0: // Absolute Short Address
			if err := requireLength(data, 2, "absolute short address"); err != nil {
				return "", 0, Operand{}, err
			}
			addr := int16(binary.BigEndian.Uint16(data[:2]))
			absolute := uint32(uint16(addr))
			text := fmt.Sprintf("$%04X", uint16(addr))
			return text, 1, effectiveAddressOperand(text, EffectiveAddress{
				Kind:            EAKindAbsoluteShort,
				Mode:            mode,
				Register:        reg,
				AbsoluteAddress: uint32Ptr(absolute),
				ResolvedAddress: uint32Ptr(absolute),
			}), nil

		case 1: // Absolute Long Address
			if err := requireLength(data, 4, "absolute long address"); err != nil {
				return "", 0, Operand{}, err
			}
			addr := binary.BigEndian.Uint32(data[:4])
			text := fmt.Sprintf("$%08X", addr)
			return text, 2, effectiveAddressOperand(text, EffectiveAddress{
				Kind:            EAKindAbsoluteLong,
				Mode:            mode,
				Register:        reg,
				AbsoluteAddress: uint32Ptr(addr),
				ResolvedAddress: uint32Ptr(addr),
			}), nil

		case 2: // Program Counter with Displacement
			if err := requireLength(data, 2, "pc displacement"); err != nil {
				return "", 0, Operand{}, err
			}
			displacement := int16(binary.BigEndian.Uint16(data[:2]))
			text := fmt.Sprintf("(%d,PC)", displacement)
			return text, 1, effectiveAddressOperand(text, EffectiveAddress{
				Kind:         EAKindPCDisplacement,
				Mode:         mode,
				Register:     reg,
				Base:         &Register{Kind: RegisterKindPC},
				Displacement: int32Ptr(int32(displacement)),
			}), nil

		case 3: // Program Counter with Index
			if err := requireLength(data, 2, "pc index extension word"); err != nil {
				return "", 0, Operand{}, err
			}
			indexWord := binary.BigEndian.Uint16(data[:2])
			indexType, indexReg, indexSize, displacement := decodeIndexWord(indexWord)
			text := fmt.Sprintf("(%d,PC,%s%d.%c)", displacement, indexType, indexReg, indexSize)
			return text, 1, effectiveAddressOperand(text, EffectiveAddress{
				Kind:         EAKindPCIndex,
				Mode:         mode,
				Register:     reg,
				Base:         &Register{Kind: RegisterKindPC},
				Displacement: int32Ptr(int32(displacement)),
				Index: &IndexRegister{
					Register: Register{Kind: parseIndexRegisterKind(indexType), Number: indexReg},
					Size:     string(indexSize),
				},
			}), nil

		case 4: // Immediate Data
			switch operandSize {
			case 4:
				if err := requireLength(data, 4, "long immediate"); err != nil {
					return "", 0, Operand{}, err
				}
				value := binary.BigEndian.Uint32(data[:4])
				text := fmt.Sprintf("#%s", formatImmediate(value, operandSize))
				return text, 2, effectiveAddressOperand(text, EffectiveAddress{
					Kind:      EAKindImmediate,
					Mode:      mode,
					Register:  reg,
					Immediate: immediatePtr(value, operandSize),
				}), nil
			case 1:
				fallthrough
			case 2:
				if err := requireLength(data, 2, "immediate"); err != nil {
					return "", 0, Operand{}, err
				}
				value := uint32(binary.BigEndian.Uint16(data[:2]))
				if operandSize == 1 {
					value &= 0xFF
				}
				text := fmt.Sprintf("#%s", formatImmediate(value, operandSize))
				return text, 1, effectiveAddressOperand(text, EffectiveAddress{
					Kind:      EAKindImmediate,
					Mode:      mode,
					Register:  reg,
					Immediate: immediatePtr(value, operandSize),
				}), nil
			default:
				return "", 0, Operand{}, fmt.Errorf("unsupported immediate size: %d", operandSize)
			}

		default:
			return "", 0, Operand{}, fmt.Errorf("unknown addressing mode: %d.%d", mode, reg)
		}

	default:
		return "", 0, Operand{}, fmt.Errorf("unknown addressing mode: %d", mode)
	}
}

// formatImmediate formats an immediate value intelligently (decimal vs hex)
func formatImmediate(value uint32, size int) string {
	// Small values and common patterns use decimal
	if value < 100 {
		return fmt.Sprintf("%d", value)
	}
	// Otherwise use hex
	switch size {
	case 1:
		return fmt.Sprintf("$%02X", byte(value))
	case 2:
		return fmt.Sprintf("$%04X", uint16(value))
	case 4:
		return fmt.Sprintf("$%08X", value)
	default:
		return fmt.Sprintf("$%X", value)
	}
}

// formatImmediateForMOVEQ formats a signed 8-bit immediate for MOVEQ
func formatImmediateForMOVEQ(value int32) string {
	if value < 0 {
		return fmt.Sprintf("-$%X", -value)
	}
	if value < 100 {
		return fmt.Sprintf("%d", value)
	}
	return fmt.Sprintf("$%X", value)
}

// decodeIndexWord extracts index register, type, size, and displacement from index word
func decodeIndexWord(indexWord uint16) (indexType string, indexReg, indexSize uint8, displacement int8) {
	indexType = "D"
	if (indexWord>>15)&0x1 == 1 {
		indexType = "A"
	}
	indexReg = uint8((indexWord >> 12) & 0x7)
	if (indexWord>>11)&0x1 == 1 {
		indexSize = 'L'
	} else {
		indexSize = 'W'
	}
	displacement = int8(indexWord & 0xFF)
	return
}

func parseIndexRegisterKind(indexType string) RegisterKind {
	if indexType == "A" {
		return RegisterKindAddress
	}
	return RegisterKindData
}
