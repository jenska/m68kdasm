package decoders

import (
	"encoding/binary"
	"fmt"
)

// decodeAddressingMode decodes an addressing mode and returns the formatted string.
// operandSize is the logical operand width in bytes so immediate operands consume
// the right number of extension words.
// Returns: (formatted string, extra words needed, error)
func decodeAddressingMode(data []byte, mode, reg uint8, operandSize int) (string, int, error) {
	switch mode {
	case 0: // Data Register Direct
		return fmt.Sprintf("D%d", reg), 0, nil

	case 1: // Address Register Direct
		return fmt.Sprintf("A%d", reg), 0, nil

	case 2: // Address Register Indirect
		return fmt.Sprintf("(A%d)", reg), 0, nil

	case 3: // Address Register Indirect with Post-increment
		return fmt.Sprintf("(A%d)+", reg), 0, nil

	case 4: // Address Register Indirect with Pre-decrement
		return fmt.Sprintf("-(A%d)", reg), 0, nil

	case 5: // Address Register Indirect with Displacement
		if len(data) < 2 {
			return "", 0, fmt.Errorf("insufficient data for displacement")
		}
		displacement := int16(binary.BigEndian.Uint16(data[:2]))
		return fmt.Sprintf("(%d,A%d)", displacement, reg), 1, nil

	case 6: // Address Register Indirect with Index
		if len(data) < 2 {
			return "", 0, fmt.Errorf("insufficient data for index")
		}
		indexWord := binary.BigEndian.Uint16(data[:2])
		indexType, indexReg, indexSize, displacement := decodeIndexWord(indexWord)
		return fmt.Sprintf("(%d,A%d,%s%d.%c)", displacement, reg, indexType, indexReg, indexSize), 1, nil

	case 7:
		// Special cases based on register field
		switch reg {
		case 0: // Absolute Short Address
			if len(data) < 2 {
				return "", 0, fmt.Errorf("insufficient data for absolute short")
			}
			addr := int16(binary.BigEndian.Uint16(data[:2]))
			return fmt.Sprintf("$%04X", uint16(addr)), 1, nil

		case 1: // Absolute Long Address
			if len(data) < 4 {
				return "", 0, fmt.Errorf("insufficient data for absolute long")
			}
			addr := binary.BigEndian.Uint32(data[:4])
			return fmt.Sprintf("$%08X", addr), 2, nil

		case 2: // Program Counter with Displacement
			if len(data) < 2 {
				return "", 0, fmt.Errorf("insufficient data for PC displacement")
			}
			displacement := int16(binary.BigEndian.Uint16(data[:2]))
			return fmt.Sprintf("(%d,PC)", displacement), 1, nil

		case 3: // Program Counter with Index
			if len(data) < 2 {
				return "", 0, fmt.Errorf("insufficient data for PC index")
			}
			indexWord := binary.BigEndian.Uint16(data[:2])
			indexType, indexReg, indexSize, displacement := decodeIndexWord(indexWord)
			return fmt.Sprintf("(%d,PC,%s%d.%c)", displacement, indexType, indexReg, indexSize), 1, nil

		case 4: // Immediate Data
			switch operandSize {
			case 4:
				if len(data) < 4 {
					return "", 0, fmt.Errorf("insufficient data for long immediate")
				}
				value := binary.BigEndian.Uint32(data[:4])
				return fmt.Sprintf("#%s", formatImmediate(value, operandSize)), 2, nil
			case 1:
				fallthrough
			case 2:
				if len(data) < 2 {
					return "", 0, fmt.Errorf("insufficient data for immediate")
				}
				value := uint32(binary.BigEndian.Uint16(data[:2]))
				if operandSize == 1 {
					value &= 0xFF
				}
				return fmt.Sprintf("#%s", formatImmediate(value, operandSize)), 1, nil
			default:
				return "", 0, fmt.Errorf("unsupported immediate size: %d", operandSize)
			}

		default:
			return "", 0, fmt.Errorf("unknown addressing mode: %d.%d", mode, reg)
		}

	default:
		return "", 0, fmt.Errorf("unknown addressing mode: %d", mode)
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
