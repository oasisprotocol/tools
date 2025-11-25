
package main

import (
	"encoding/binary"
	"fmt"

	"github.com/fxamacker/cbor/v2"
)

// decodeEVMData decodes EVM module storage data.
// See: _oasis-sdk/runtime-sdk/modules/evm/src/state.rs
func decodeEVMData(module string, key []byte, value []byte) *EVMDataInfo {
	// Module format: "evm:subtype" where subtype is extracted from key prefix
	// The full key after module name starts with the storage type prefix

	// Find where module name ends in the key
	moduleNameEnd := 0
	for i, b := range key {
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_' {
			moduleNameEnd = i + 1
		} else {
			break
		}
	}

	if moduleNameEnd >= len(key) {
		return nil // No data after module name
	}

	subKey := key[moduleNameEnd:]
	if len(subKey) < 1 {
		return nil
	}

	evmInfo := &EVMDataInfo{}
	storagePrefix := subKey[0]
	data := subKey[1:]

	switch storagePrefix {
	case 0x01: // CODES: evm + 0x01 + H160 (address)
		evmInfo.StorageType = "code"
		if len(data) >= 20 {
			evmInfo.Address = truncateHex0x(data[:20], TruncateLongSize)
			data = data[20:]
		}
		// Value is contract bytecode
		evmInfo.CodeSize = len(value)
		if len(value) > 0 {
			evmInfo.CodeHex = truncateHex(value, TruncateLongSize)
		}

	case 0x02: // STORAGES: evm + 0x02 + H160 (address) + H256 (slot)
		evmInfo.StorageType = "storage"
		if len(data) >= 20 {
			evmInfo.Address = truncateHex0x(data[:20], TruncateLongSize)
			data = data[20:]
			if len(data) >= 32 {
				evmInfo.StorageSlot = truncateHex0x(data[:32], TruncateHashSize)
			}
		}
		// Value is H256 storage value
		if len(value) == 32 {
			evmInfo.StorageValue = truncateHex0x(value, TruncateLongSize)
		}

	case 0x03: // BLOCK_HASHES: evm + 0x03 + RuntimeHeight (uint64 BE)
		evmInfo.StorageType = "block_hash"
		if len(data) >= 8 {
			evmInfo.RuntimeHeight = binary.BigEndian.Uint64(data[:8])
		}
		// Value is H256 block hash
		if len(value) == 32 {
			evmInfo.BlockHash = truncateHex0x(value, TruncateHashSize)
		}

	case 0x04: // CONFIDENTIAL_STORAGES: evm + 0x04 + H160 (address) + H256 (slot)
		evmInfo.StorageType = "confidential_storage"
		if len(data) >= 20 {
			evmInfo.Address = truncateHex0x(data[:20], TruncateLongSize)
			data = data[20:]
			if len(data) >= 32 {
				evmInfo.StorageSlot = truncateHex0x(data[:32], TruncateHashSize)
			}
		}
		// Value is encrypted - we can only show size
		if len(value) > 0 {
			evmInfo.StorageValue = fmt.Sprintf("<encrypted:%d bytes>", len(value))
		}

	default:
		return nil // Unknown EVM storage type
	}

	return evmInfo
}

// decodeEVMEvent decodes an EVM Log event from CBOR-encoded value.
// See: _oasis-sdk/runtime-sdk/modules/evm/src/lib.rs:253-263
// Returns (info, error) where error is a parsing error, not an execution error.
func decodeEVMEvent(value []byte) (*EVMEventInfo, string) {
	info := &EVMEventInfo{}

	// Value is CBOR: [event_code, {address, topics, data}]
	var eventWrapper []interface{}
	if err := cbor.Unmarshal(value, &eventWrapper); err != nil {
		return nil, err.Error()
	}
	if len(eventWrapper) < 2 {
		return nil, "invalid format"
	}

	eventData, ok := eventWrapper[1].(map[interface{}]interface{})
	if !ok {
		return nil, "invalid format"
	}

	// Extract address (H160)
	if addrBytes, ok := eventData["address"].([]byte); ok && len(addrBytes) == 20 {
		info.Address = truncateHex0x(addrBytes, TruncateLongSize)
	} else {
		// Invalid address, but continue decoding - return partial data with error
		return info, "invalid address (partial decode)"
	}

	// Extract topics (Vec<H256>)
	if topicsArray, ok := eventData["topics"].([]interface{}); ok {
		info.TopicCount = len(topicsArray)
		for i, topic := range topicsArray {
			if topicBytes, ok := topic.([]byte); ok && len(topicBytes) == 32 {
				topicHex := truncateHex0x(topicBytes, TruncateHashSize)
				info.Topics = append(info.Topics, topicHex)
				if i == 0 {
					info.EventHash = topicHex
					if sig, found := EVMEventSignatures[topicHex[:len(topicHex)]]; found {
						info.EventSignature = sig
					}
				}
			}
		}
	}

	// Extract data
	if dataBytes, ok := eventData["data"].([]byte); ok {
		info.DataSize = len(dataBytes)
		info.DataHex = truncateHex(dataBytes, TruncateLongSize)
	}

	return info, ""
}

// decodeEVMTxInput decodes EVM transaction input artifacts.
// Returns (info, error) where error is a parsing error.
func decodeEVMTxInput(key []byte, value []byte) (*EVMTxInputInfo, string) {
	info := &EVMTxInputInfo{
		TxInputSize: len(value),
		TxInputHex:  truncateHex(value, TruncateLongSize),
	}

	if len(key) >= 33 {
		info.TxHash = truncateHex0x(key[1:33], TruncateHashSize)
	}

	var ia cborRuntimeInputArtifacts
	if err := cbor.Unmarshal(value, &ia); err != nil {
		return nil, fmt.Sprintf("failed to unmarshal RuntimeInputArtifacts: %v", err)
	}
	info.BatchOrder = ia.BatchOrder

	// Try decoding as array format first (old runtime version)
	var callArray cborRuntimeCallArrayFormat
	if err := cbor.Unmarshal(ia.Input, &callArray); err == nil {
		info.Method = callArray.Method
		// Decode EVM call methods that have body structure
		if callArray.Method == "evm.Call" || callArray.Method == "evm.Create" || callArray.Method == "evm.SimulateCall" || callArray.Method == "evm.EstimateGas" {
			isCreate := callArray.Method == "evm.Create"
			info.EVMCall = decodeEVMCallBodyFromCBOR(callArray.Body, isCreate)
		}
		return info, ""
	}

	// Try decoding as map format (newer runtime version)
	var callMap cborRuntimeCallMapFormat
	if err := cbor.Unmarshal(ia.Input, &callMap); err == nil {
		info.Method = callMap.Method
		// Decode EVM call methods that have body structure
		if callMap.Method == "evm.Call" || callMap.Method == "evm.Create" || callMap.Method == "evm.SimulateCall" || callMap.Method == "evm.EstimateGas" {
			isCreate := callMap.Method == "evm.Create"
			info.EVMCall = decodeEVMCallBodyFromCBOR(callMap.Body, isCreate)
		}
		return info, ""
	}

	// Fallback: try generic array decode (for compatibility with other runtime versions)
	var callArrayGeneric []interface{}
	if err := cbor.Unmarshal(ia.Input, &callArrayGeneric); err == nil {
		// Array format: [format, method, body, ...] or [format, [method, body, ...]]
		if len(callArrayGeneric) < 2 {
			return info, fmt.Sprintf("array format: insufficient fields (len=%d), expected at least 2", len(callArrayGeneric))
		}

		// Extract method and body - handle both flat and nested array formats
		var method string
		var bodyBytes []byte

		if methodVal, ok := callArrayGeneric[1].(string); ok {
			// Flat format: [format, method, body, ...]
			method = methodVal
			if len(callArrayGeneric) >= 3 {
				bodyBytes, _ = callArrayGeneric[2].([]byte)
			}
		} else if nestedArray, ok := callArrayGeneric[1].([]interface{}); ok {
			// Nested format: [format, [method, body, ...]] or [format, [{method: "...", body: ...}]]
			if len(nestedArray) < 1 {
				return info, "nested array format: element[1] is empty array"
			}
			if methodVal, ok := nestedArray[0].(string); ok {
				// Nested array format: [format, [method, body, ...]]
				method = methodVal
				if len(nestedArray) >= 2 {
					bodyBytes, _ = nestedArray[1].([]byte)
				}
			} else if nestedMap, ok := nestedArray[0].(map[interface{}]interface{}); ok {
				// Nested map format: [format, [{method: "...", body: ...}]]
				if methodVal, ok := nestedMap["method"].(string); ok {
					method = methodVal
					if bodyVal, ok := nestedMap["body"].([]byte); ok {
						bodyBytes = bodyVal
					}
				} else {
					return info, "nested map format: method field missing or invalid"
				}
			} else {
				return info, fmt.Sprintf("nested array format: element[1][0] is not a valid method string or map (type: %T)", nestedArray[0])
			}
		} else {
			return info, fmt.Sprintf("array format: element[1] is not a valid method string or array (type: %T)", callArrayGeneric[1])
		}

		info.Method = method
		// Decode EVM call methods that have body structure
		if bodyBytes != nil && (method == "evm.Call" || method == "evm.Create" || method == "evm.SimulateCall" || method == "evm.EstimateGas") {
			info.EVMCall = decodeEVMCallBodyFromCBOR(bodyBytes, method == "evm.Create")
		}
		return info, ""
	}

	// Final fallback: try legacy map[interface{}]interface{} decode
	var call map[interface{}]interface{}
	if err := cbor.Unmarshal(ia.Input, &call); err == nil {
		if methodVal, ok := call["method"].(string); ok {
			info.Method = methodVal
			// Decode EVM call methods that have body structure
			if methodVal == "evm.Call" || methodVal == "evm.Create" || methodVal == "evm.SimulateCall" || methodVal == "evm.EstimateGas" {
				isCreate := methodVal == "evm.Create"
				// Extract body bytes and decode
				if bodyBytes, ok := call["body"].([]byte); ok {
					info.EVMCall = decodeEVMCallBodyFromCBOR(bodyBytes, isCreate)
				}
			}
			return info, ""
		}
		return info, "map format: method field missing or invalid"
	}

	// All decode attempts failed
	return info, "all formats failed"
}

// decodeEVMCallBodyFromCBOR decodes EVM call/create body from raw CBOR bytes.
func decodeEVMCallBodyFromCBOR(bodyBytes []byte, isCreate bool) *EVMCallInfo {
	info := &EVMCallInfo{}
	if isCreate {
		info.Type = "create"
	} else {
		info.Type = "call"
	}

	var bodyMap map[interface{}]interface{}
	if err := cbor.Unmarshal(bodyBytes, &bodyMap); err != nil {
		return info
	}

	// Extract address (calls only)
	if !isCreate {
		if addrBytes, ok := bodyMap["address"].([]byte); ok && len(addrBytes) == 20 {
			info.Address = truncateHex0x(addrBytes, TruncateLongSize)
		}
	}

	// Extract value
	if valueBytes, ok := bodyMap["value"].([]byte); ok {
		info.Value = formatU256(valueBytes)
	}

	// Extract data/init_code
	dataKey := "data"
	if isCreate {
		dataKey = "init_code"
	}
	if dataBytes, ok := bodyMap[dataKey].([]byte); ok {
		info.DataSize = len(dataBytes)
		info.DataHex = truncateHex(dataBytes, TruncateLongSize)
	}

	return info
}

// decodeEVMTxOutput decodes EVM transaction output artifacts.
// Returns (info, error) where error is a parsing error.
// Execution errors (transaction failures) are stored in info.Error field.
func decodeEVMTxOutput(value []byte) (*EVMTxOutputInfo, string) {
	info := &EVMTxOutputInfo{
		TxOutputSize: len(value),
		TxOutputHex:  truncateHex(value, TruncateLongSize),
	}

	var oa cborRuntimeOutputArtifacts
	if err := cbor.Unmarshal(value, &oa); err != nil {
		return nil, err.Error()
	}

	if len(oa.Output) == 0 {
		info.Success = true
		return info, ""
	}

	// Try to decode as CBOR CallResult
	var result map[interface{}]interface{}
	if err := cbor.Unmarshal(oa.Output, &result); err != nil {
		// Raw bytes - success
		info.Success = true
		info.ResultSize = len(oa.Output)
		info.ResultHex = truncateHex(oa.Output, TruncateLongSize)
		return info, ""
	}

	// Check success/failure
	if okVal, exists := result["ok"]; exists {
		info.Success = true
		if okBytes, ok := okVal.([]byte); ok {
			info.ResultSize = len(okBytes)
			info.ResultHex = truncateHex(okBytes, TruncateLongSize)
		}
	} else if failVal, exists := result["fail"]; exists {
		info.Success = false
		if failMap, ok := failVal.(map[interface{}]interface{}); ok {
			if msgVal, ok := failMap["message"].(string); ok {
				info.Error = msgVal // Execution error, keep in struct
			} else {
				info.Error = fmt.Sprintf("%v", failMap)
			}
		}
	}

	return info, ""
}
