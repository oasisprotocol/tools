package main

import (
	"fmt"
	"strings"
)

// extractModuleName extracts ASCII module name from MKVS key
func extractModuleName(key []byte) string {
	if len(key) == 0 {
		return "<empty>"
	}

	// Skip leading 0x00 if present
	if key[0] == 0x00 && len(key) > 1 {
		key = key[1:]
	}

	// Find ASCII module name
	end := 0
	for i, b := range key {
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_' {
			end = i + 1
		} else {
			break
		}
	}

	if end > 0 {
		return string(key[:end])
	}
	return truncateHex(key, 16)
}

// formatCBOR formats decoded CBOR value concisely
func formatCBOR(v interface{}, rawLen int) string {
	switch val := v.(type) {
	case map[interface{}]interface{}:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, fmt.Sprintf("%v", k))
		}
		if len(keys) > 3 {
			return fmt.Sprintf("map{%s, +%d}", strings.Join(keys[:3], ","), len(keys)-3)
		}
		return fmt.Sprintf("map{%s}", strings.Join(keys, ","))
	case []interface{}:
		return fmt.Sprintf("array[%d]", len(val))
	case []byte:
		return fmt.Sprintf("bytes(%d)", len(val))
	case string:
		if len(val) > 20 {
			return fmt.Sprintf("%q...", val[:20])
		}
		return fmt.Sprintf("%q", val)
	case uint64:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case bool:
		return fmt.Sprintf("%v", val)
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%T(%d bytes)", v, rawLen)
	}
}

// isPrintableASCII checks if a string contains only printable ASCII characters
func isPrintableASCII(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r < 32 || r > 126 {
			return false
		}
	}
	return true
}

// truncateBytes returns first n bytes or all bytes if shorter
func truncateBytes(data []byte, n int) []byte {
	if len(data) <= n {
		return data
	}
	return data[:n]
}

// truncateHex converts bytes to hex and truncates if too long
func truncateHex(data []byte, maxLen int) string {
	hex := fmt.Sprintf("%x", data)
	if len(hex) > maxLen {
		return hex[:maxLen] + "..."
	}
	return hex
}
