package main

import (
	"fmt"
	"strings"
)

// extractModuleName extracts module name from MKVS leaf key and returns module:subtype description
func extractModuleName(key []byte) string {
	if len(key) == 0 {
		return "<empty>"
	}

	// Skip leading 0x00 if present
	if key[0] == 0x00 && len(key) > 1 {
		key = key[1:]
	}

	// Check for IO Tree prefixes
	switch key[0] {
	case 'T': // Transaction artifact prefix (0x54)
		// Key format: 'T' + tx_hash (32 bytes) + artifact_kind (1 byte)
		if len(key) >= 34 {
			kind := key[33]
			kindStr := "unknown"
			if kind == 1 {
				kindStr = "input"
			} else if kind == 2 {
				kindStr = "output"
			}
			return fmt.Sprintf("io_tx:%s (hash=%s)", kindStr, truncateHex(key[1:33], 16))
		}
		return fmt.Sprintf("io_tx (hash=%s)", truncateHex(key[1:], 16))

	case 'E': // Event tag prefix (0x45)
		// Key format: 'E' + tag_key (variable, module name) + tx_hash (32 bytes)
		if len(key) > 33 {
			tagKey := key[1 : len(key)-32]
			// Extract module name from tag key
			tagModule := ""
			for _, b := range tagKey {
				if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_' {
					tagModule += string(b)
				} else {
					break
				}
			}
			if tagModule != "" {
				return fmt.Sprintf("io_event:%s", tagModule)
			}
		}
		return "io_event"
	}

	// Regular state key: module_name + sub_prefix + data
	end := 0
	for i, b := range key {
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_' {
			end = i + 1
		} else {
			break
		}
	}

	if end > 0 {
		moduleName := string(key[:end])
		subKey := key[end:]
		return describeModuleKey(moduleName, subKey)
	}
	return truncateHex(key, 16)
}

// describeModuleKey returns module name with sub-key type description
func describeModuleKey(module string, subKey []byte) string {
	if len(subKey) == 0 {
		return module
	}

	prefix := subKey[0]
	var subType string

	switch module {
	case "evm":
		switch prefix {
		case 0x01:
			subType = "code"
		case 0x02:
			subType = "storage"
		case 0x03:
			subType = "block_hash"
		case 0x04:
			subType = "confidential_storage"
		}
	case "accounts":
		switch prefix {
		case 0x01:
			subType = "account"
		case 0x02:
			subType = "balance"
		case 0x03:
			subType = "total_supply"
		}
	case "contracts":
		switch prefix {
		case 0x01:
			subType = "next_code_id"
		case 0x02:
			subType = "next_instance_id"
		case 0x03:
			subType = "code_info"
		case 0x04:
			subType = "instance_info"
		case 0x05:
			subType = "instance_state"
		case 0xFF:
			subType = "code"
		}
	case "core":
		switch prefix {
		case 0x01:
			subType = "metadata"
		case 0x02:
			subType = "message_handlers"
		case 0x03:
			subType = "last_epoch"
		case 0x04:
			subType = "dynamic_min_gas_price"
		}
	case "consensus_accounts":
		switch prefix {
		case 0x01:
			subType = "delegations"
		case 0x02:
			subType = "undelegations"
		case 0x03:
			subType = "undelegation_queue"
		case 0x04:
			subType = "receipts"
		}
	}

	if subType != "" {
		return fmt.Sprintf("%s:%s", module, subType)
	}
	return module
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

// truncateHex converts bytes to hex and truncates if too long
func truncateHex(data []byte, maxLen int) string {
	hex := fmt.Sprintf("%x", data)
	if len(hex) > maxLen {
		return hex[:maxLen] + "..."
	}
	return hex
}

