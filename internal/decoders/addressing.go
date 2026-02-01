package decoders

import (
	"encoding/binary"
	"fmt"
)

// decodeAddressingMode decodes an addressing mode and returns the formatted string
// Returns: (formatted string, extra words needed, error)
func decodeAddressingMode(data []byte, mode, reg uint8) (string, int, error) {
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
			return "", 0, fmt.Errorf("nicht genügend Daten für Displacement")
		}
		displacement := int16(binary.BigEndian.Uint16(data[:2]))
		return fmt.Sprintf("(%d,A%d)", displacement, reg), 1, nil

	case 6: // Address Register Indirect with Index
		if len(data) < 2 {
			return "", 0, fmt.Errorf("nicht genügend Daten für Index")
		}
		// Format: Xn (2-bit index scale, 4-bit index register, sign bit, size bit, 8-bit displacement)
		indexWord := binary.BigEndian.Uint16(data[:2])
		indexReg := (indexWord >> 12) & 0xF
		indexType := "D"
		if indexReg >= 8 {
			indexType = "A"
			indexReg -= 8
		}
		displacement := int8(indexWord & 0xFF)
		return fmt.Sprintf("(%d,A%d,%s%d.L)", displacement, reg, indexType, indexReg), 1, nil

	case 7:
		// Special cases based on register field
		switch reg {
		case 0: // Absolute Short Address
			if len(data) < 2 {
				return "", 0, fmt.Errorf("nicht genügend Daten für Absolute Short")
			}
			addr := int16(binary.BigEndian.Uint16(data[:2]))
			return fmt.Sprintf("$%04X", uint16(addr)), 1, nil

		case 1: // Absolute Long Address
			if len(data) < 4 {
				return "", 0, fmt.Errorf("nicht genügend Daten für Absolute Long")
			}
			addr := binary.BigEndian.Uint32(data[:4])
			return fmt.Sprintf("$%08X", addr), 2, nil

		case 2: // Program Counter with Displacement
			if len(data) < 2 {
				return "", 0, fmt.Errorf("nicht genügend Daten für PC Displacement")
			}
			displacement := int16(binary.BigEndian.Uint16(data[:2]))
			return fmt.Sprintf("(%d,PC)", displacement), 1, nil

		case 3: // Program Counter with Index
			if len(data) < 2 {
				return "", 0, fmt.Errorf("nicht genügend Daten für PC Index")
			}
			indexWord := binary.BigEndian.Uint16(data[:2])
			indexReg := (indexWord >> 12) & 0xF
			indexType := "D"
			if indexReg >= 8 {
				indexType = "A"
				indexReg -= 8
			}
			displacement := int8(indexWord & 0xFF)
			return fmt.Sprintf("(%d,PC,%s%d.L)", displacement, indexType, indexReg), 1, nil

		case 4: // Immediate Data
			if len(data) < 2 {
				return "", 0, fmt.Errorf("nicht genügend Daten für Immediate")
			}
			// Size is determined by context, but for now assume 16-bit
			value := binary.BigEndian.Uint16(data[:2])
			immStr := formatImmediate(uint32(value), 2)
			return fmt.Sprintf("#%s", immStr), 1, nil

		default:
			return "", 0, fmt.Errorf("unbekannte Adressierungsmode: %d.%d", mode, reg)
		}

	default:
		return "", 0, fmt.Errorf("unbekannte Adressierungsmode: %d", mode)
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
