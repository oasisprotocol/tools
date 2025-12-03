package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"unicode"
)

const (
	// TruncateHashLen is the default hex character limit for hash truncation
	TruncateHashLen = 16
	// TruncateLongLen is the default hex character limit for long data truncation
	TruncateLongLen = 1000
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
			return fmt.Sprintf("io_tx:%s", kindStr)
		}
		return "io_tx"

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
	return truncateHex(key, TruncateHashLen)
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

// formatCBORDetailed formats decoded CBOR value with detailed structure
func formatCBORDetailed(v interface{}) interface{} {
	switch val := v.(type) {
	case map[interface{}]interface{}:
		result := make(map[string]interface{})
		for k, v := range val {
			keyStr := fmt.Sprintf("%v", k)
			result[keyStr] = formatCBORDetailed(v)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, elem := range val {
			result[i] = formatCBORDetailed(elem)
		}
		return result
	case []byte:
		// Format as hex string with size info
		if len(val) <= 32 {
			return truncateHex0x(val, 64)
		}
		return truncateHex0x(val, 64) + fmt.Sprintf(" (%d bytes)", len(val))
	case string:
		return val
	case uint64:
		return val
	case int64:
		return val
	case bool:
		return val
	case nil:
		return nil
	default:
		return fmt.Sprintf("%v", val)
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

// formatRawValue formats data as ASCII or 0x-prefixed hex with truncation
// Returns ASCII string if printable, otherwise 0x-prefixed hex
func formatRawValue(data []byte, maxLen int) string {
	if isPrintableASCII(string(data)) {
		if len(data) > maxLen {
			return string(data[:maxLen]) + "..."
		}
		return string(data)
	}
	// Hex formatting with 0x prefix
	hexStr := hex.EncodeToString(data)
	if len(hexStr) > maxLen {
		return "0x" + hexStr[:maxLen] + "..."
	}
	return "0x" + hexStr
}

// truncateHex converts bytes to hex and truncates if too long.
func truncateHex(data []byte, maxLen int) string {
	hex := fmt.Sprintf("%x", data)
	if len(hex) > maxLen {
		return hex[:maxLen] + "..."
	}
	return hex
}

// truncateHex0x converts bytes to 0x-prefixed hex and truncates if too long.
func truncateHex0x(data []byte, maxLen int) string {
	hex := fmt.Sprintf("%x", data)
	if len(hex) > maxLen {
		return "0x" + hex[:maxLen] + "..."
	}
	return "0x" + hex
}

// formatU256 converts big-endian U256 bytes to decimal string.
func formatU256(b []byte) string {
	if len(b) == 0 {
		return "0"
	}
	n := new(big.Int).SetBytes(b)
	return n.String()
}

// bech32Charset is the character set for bech32 encoding
const bech32Charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"

// bech32Encode encodes data with bech32 using the given human-readable part
func bech32Encode(hrp string, data []byte) string {
	// Convert 8-bit data to 5-bit groups
	var values []int
	acc := 0
	bits := 0
	for _, b := range data {
		acc = (acc << 8) | int(b)
		bits += 8
		for bits >= 5 {
			bits -= 5
			values = append(values, (acc>>bits)&31)
		}
	}
	if bits > 0 {
		values = append(values, (acc<<(5-bits))&31)
	}

	// Create checksum
	checksum := bech32CreateChecksum(hrp, values)

	// Build result efficiently using strings.Builder
	var result strings.Builder
	result.Grow(len(hrp) + 1 + len(values) + len(checksum))
	result.WriteString(hrp)
	result.WriteString("1")
	for _, v := range values {
		result.WriteByte(bech32Charset[v])
	}
	for _, v := range checksum {
		result.WriteByte(bech32Charset[v])
	}
	return result.String()
}

// bech32Polymod computes the bech32 checksum polynomial
func bech32Polymod(values []int) int {
	gen := []int{0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3}
	chk := 1
	for _, v := range values {
		b := chk >> 25
		chk = ((chk & 0x1ffffff) << 5) ^ v
		for i := 0; i < 5; i++ {
			if (b>>i)&1 == 1 {
				chk ^= gen[i]
			}
		}
	}
	return chk
}

// bech32HRPExpand expands the human-readable part for checksum calculation
func bech32HRPExpand(hrp string) []int {
	result := make([]int, 0, len(hrp)*2+1)
	for _, c := range hrp {
		result = append(result, int(c)>>5)
	}
	result = append(result, 0)
	for _, c := range hrp {
		result = append(result, int(c)&31)
	}
	return result
}

// bech32CreateChecksum creates the bech32 checksum
func bech32CreateChecksum(hrp string, data []int) []int {
	values := append(bech32HRPExpand(hrp), data...)
	values = append(values, 0, 0, 0, 0, 0, 0)
	polymod := bech32Polymod(values) ^ 1
	checksum := make([]int, 6)
	for i := 0; i < 6; i++ {
		checksum[i] = (polymod >> (5 * (5 - i))) & 31
	}
	return checksum
}

// extractErrors recursively extracts all non-empty fields ending with "Error" from a value.
func extractErrors(v interface{}, prefix string, maxDepth int) []string {
	if maxDepth <= 0 || v == nil {
		return []string{}
	}

	val := reflect.ValueOf(v)
	if !val.IsValid() {
		return []string{}
	}

	// Dereference pointers
	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return []string{}
		}
		val = val.Elem()
	}

	results := []string{}
	switch val.Kind() {
	case reflect.Struct:
		typ := val.Type()
		for i := 0; i < val.NumField(); i++ {
			field, fieldType := val.Field(i), typ.Field(i)
			if !field.CanInterface() {
				continue
			}
			jsonName := getJSONFieldName(fieldType)
			// Collect error if field ends with "Error" and is non-empty string
			if strings.HasSuffix(fieldType.Name, "Error") && field.Kind() == reflect.String {
				if errStr := field.String(); errStr != "" {
					results = append(results, fmt.Sprintf("%s.%s: %s", prefix, jsonName, errStr))
				}
			}
			results = append(results, extractErrors(field.Interface(), prefix+"."+jsonName, maxDepth-1)...)
		}
	case reflect.Interface:
		if !val.IsNil() {
			results = extractErrors(val.Interface(), prefix, maxDepth-1)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < val.Len(); i++ {
			if elem := val.Index(i); elem.CanInterface() {
				results = append(results, extractErrors(elem.Interface(), prefix+"[]", maxDepth-1)...)
			}
		}
	}
	return results
}

// getJSONFieldName extracts JSON field name from struct tag, falls back to snake_case
func getJSONFieldName(field reflect.StructField) string {
	if tag := field.Tag.Get("json"); tag != "" {
		if name := strings.Split(tag, ",")[0]; name != "" {
			return name
		}
	}
	return toSnakeCase(field.Name)
}

// toSnakeCase converts PascalCase to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			result.WriteRune('_')
		}
		result.WriteRune(unicode.ToLower(r))
	}
	return result.String()
}

// formatApproxSize returns approximate size string for error messages.
// Rounds to first significant digit: 23→~20, 234→~200, 2345→~2K, 23456→~20K
func formatApproxSize(size int) string {
	if size < 10 {
		return fmt.Sprintf("~%d", size)
	} else if size < 100 {
		return fmt.Sprintf("~%d", (size/10)*10)
	} else if size < 1000 {
		return fmt.Sprintf("~%d", (size/100)*100)
	} else if size < 10000 {
		return fmt.Sprintf("~%dK", size/1000)
	} else if size < 100000 {
		return fmt.Sprintf("~%dK", (size/10000)*10)
	} else if size < 1000000 {
		return fmt.Sprintf("~%dK", (size/100000)*100)
	} else {
		return fmt.Sprintf("~%dM", size/1000000)
	}
}
