
package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/fxamacker/cbor/v2"
)

// decodeEVMData decodes EVM module storage data.
// See: _oasis-sdk/runtime-sdk/modules/evm/src/state.rs
func decodeEVMData(module string, key []byte, value []byte) *EVMDataInfo {
	info := &EVMDataInfo{}

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
		info.RawError = "no data after module name"
		return info
	}

	subKey := key[moduleNameEnd:]
	if len(subKey) < 1 {
		info.RawError = "insufficient subkey data"
		return info
	}

	storagePrefix := subKey[0]
	data := subKey[1:]

	switch storagePrefix {
	case 0x01: // CODES: evm + 0x01 + H160 (address)
		info.StorageType = "code"
		if len(data) >= 20 {
			info.Address = "0x" + hex.EncodeToString(data[:20])
			data = data[20:]
		}
		// Value is contract bytecode
		info.CodeSize = len(value)
		if len(value) > 0 {
			info.CodeHex = formatRawValue(value, TruncateLongLen)
		}

	case 0x02: // STORAGES: evm + 0x02 + H160 (address) + H256 (slot)
		info.StorageType = "storage"
		if len(data) >= 20 {
			info.Address = "0x" + hex.EncodeToString(data[:20])
			data = data[20:]
			if len(data) >= 32 {
				info.StorageSlot = formatRawValue(data[:32], TruncateHashLen)
			}
		}
		// Value is H256 storage value
		if len(value) == 32 {
			info.StorageValue = formatRawValue(value, TruncateLongLen)
		}

	case 0x03: // BLOCK_HASHES: evm + 0x03 + RuntimeHeight (uint64 BE)
		info.StorageType = "block_hash"
		if len(data) >= 8 {
			info.RuntimeHeight = binary.BigEndian.Uint64(data[:8])
		}
		// Value is H256 block hash
		if len(value) == 32 {
			info.BlockHash = formatRawValue(value, TruncateHashLen)
		}

	case 0x04: // CONFIDENTIAL_STORAGES: evm + 0x04 + H160 (address) + H256 (slot)
		info.StorageType = "confidential_storage"
		if len(data) >= 20 {
			info.Address = "0x" + hex.EncodeToString(data[:20])
			data = data[20:]
			if len(data) >= 32 {
				info.StorageSlot = formatRawValue(data[:32], TruncateHashLen)
			}
		}
		// Value is encrypted - we can only show size
		if len(value) > 0 {
			info.StorageValue = fmt.Sprintf("<encrypted:%d bytes>", len(value))
		}

	default:
		info.RawError = fmt.Sprintf("unknown EVM storage type: 0x%02x", storagePrefix)
		return info
	}

	return info
}

// decodeEVMEvent decodes an EVM Log event from CBOR-encoded value.
// See: _oasis-sdk/runtime-sdk/modules/evm/src/lib.rs:253-263
// Returns (info, error) where error is a parsing error, not an execution error.
func decodeEVMEvent(value []byte) *EVMEventInfo {
	info := &EVMEventInfo{}

	// Value is CBOR: [event_code, {address, topics, data}]
	var eventWrapper []interface{}
	if err := cbor.Unmarshal(value, &eventWrapper); err != nil {
		info.RawError = fmt.Sprintf("cbor unmarshal failed: %v", err)
		return info
	}
	if len(eventWrapper) < 2 {
		info.RawError = "invalid format: insufficient fields"
		return info
	}

	eventData, ok := eventWrapper[1].(map[interface{}]interface{})
	if !ok {
		info.RawError = "invalid format: element[1] not a map"
		return info
	}

	// Extract address (H160)
	if addrBytes, ok := eventData["address"].([]byte); ok && len(addrBytes) == 20 {
		info.Address = "0x" + hex.EncodeToString(addrBytes)
	} else {
		info.RawError = "invalid address"
		return info
	}

	// Extract topics (Vec<H256>)
	if topicsArray, ok := eventData["topics"].([]interface{}); ok {
		info.TopicCount = len(topicsArray)
		for i, topic := range topicsArray {
			if topicBytes, ok := topic.([]byte); ok && len(topicBytes) == 32 {
				topicHex := "0x" + hex.EncodeToString(topicBytes)
				info.Topics = append(info.Topics, topicHex)
				if i == 0 {
					info.EventHash = topicHex
					if sig, found := EVMEventSignatures[topicHex]; found {
						info.EventSignature = sig
					}
				}
			}
		}
	}

	// Extract data
	if dataBytes, ok := eventData["data"].([]byte); ok {
		info.DataSize = len(dataBytes)
		info.DataHex = formatRawValue(dataBytes, TruncateLongLen)
	}

	return info
}

// decodeEVMTxInput decodes EVM transaction input artifacts.
func decodeEVMTxInput(key []byte, value []byte) *EVMTxInputInfo {
	info := &EVMTxInputInfo{}

	if len(key) >= 33 {
		info.TxHash = formatRawValue(key[1:33], TruncateHashLen)
	}

	var ia cborRuntimeInputArtifacts
	if err := cbor.Unmarshal(value, &ia); err != nil {
		info.RawError = fmt.Sprintf("failed to unmarshal RuntimeInputArtifacts: %v", err)
		return info
	}
	info.BatchOrder = ia.BatchOrder

	// Attempt 0: Variant wrapper format [version, [variant_tag, data]]
	var wrapperArray []interface{}
	if err := cbor.Unmarshal(ia.Input, &wrapperArray); err == nil && len(wrapperArray) >= 2 {
		if nestedArray, ok := wrapperArray[1].([]interface{}); ok && len(nestedArray) >= 2 {
			// Extract the actual transaction data (second element of nested array)
			if txDataBytes, err := cbor.Marshal(nestedArray[1]); err == nil {
				// Recursively decode the unwrapped transaction
				ia.Input = txDataBytes
				return decodeEVMTxInput(key, value)
			}
		}
	}

	// Try decoding as array format first (old runtime version)
	var callArray cborRuntimeCallArrayFormat
	if err := cbor.Unmarshal(ia.Input, &callArray); err == nil {
		info.Method = callArray.Method
		// Decode EVM call methods that have body structure
		if callArray.Method == "evm.Call" || callArray.Method == "evm.Create" || callArray.Method == "evm.SimulateCall" || callArray.Method == "evm.EstimateGas" {
			isCreate := callArray.Method == "evm.Create"
			info.EVMTx = decodeEVMTransactionFromCBOR(callArray.Body, isCreate)
		}
		return info
	}

	// Try decoding as map format (newer runtime version)
	var callMap cborRuntimeCallMapFormat
	if err := cbor.Unmarshal(ia.Input, &callMap); err == nil {
		info.Method = callMap.Method
		// Decode EVM call methods that have body structure
		if callMap.Method == "evm.Call" || callMap.Method == "evm.Create" || callMap.Method == "evm.SimulateCall" || callMap.Method == "evm.EstimateGas" {
			isCreate := callMap.Method == "evm.Create"
			info.EVMTx = decodeEVMTransactionFromCBOR(callMap.Body, isCreate)
		}
		return info
	}

	// Fallback: try generic array decode (for compatibility with other runtime versions)
	var callArrayGeneric []interface{}
	if err := cbor.Unmarshal(ia.Input, &callArrayGeneric); err == nil {
		// Array format: [format, method, body, ...] or [format, [method, body, ...]]
		if len(callArrayGeneric) < 2 {
			info.RawError = "array format: insufficient fields"
			return info
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
				info.RawError = "nested array format: empty"
				return info
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
					// Fallback A: Re-marshal map and try decoding as cborRuntimeCallMapFormat
					if marshaled, err := cbor.Marshal(nestedMap); err == nil {
						var callMap cborRuntimeCallMapFormat
						if err := cbor.Unmarshal(marshaled, &callMap); err == nil {
							method = callMap.Method
							bodyBytes = callMap.Body
						} else {
							info.RawError = "nested map format: method missing"
							return info
						}
					} else {
						info.RawError = "nested map format: method missing"
						return info
					}
				}
			} else if nestedBytes, ok := nestedArray[0].([]byte); ok {
				// Fallback B: Nested bytes - unwrap and decode
				var callMap cborRuntimeCallMapFormat
				if err := cbor.Unmarshal(nestedBytes, &callMap); err == nil {
					method = callMap.Method
					bodyBytes = callMap.Body
				} else {
					// Try array format as second attempt
					var callArray cborRuntimeCallArrayFormat
					if err := cbor.Unmarshal(nestedBytes, &callArray); err == nil {
						method = callArray.Method
						bodyBytes = callArray.Body
					} else {
						info.RawError = "nested array format: unable to decode nested bytes"
						return info
					}
				}
			} else {
				info.RawError = "nested array format: invalid element type"
				return info
			}
		} else {
			info.RawError = "array format: invalid element[1] type"
			return info
		}

		info.Method = method
		// Decode EVM call methods that have body structure
		if bodyBytes != nil && (method == "evm.Call" || method == "evm.Create" || method == "evm.SimulateCall" || method == "evm.EstimateGas") {
			info.EVMTx = decodeEVMTransactionFromCBOR(bodyBytes, method == "evm.Create")
		}
		return info
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
					info.EVMTx = decodeEVMTransactionFromCBOR(bodyBytes, isCreate)
				}
			}
			return info
		}
		info.RawError = "map format: method missing"
		return info
	}

	// All decode attempts failed
	info.RawError = "all formats failed"
	return info
}

// decodeEVMTransactionFromCBOR decodes EVM transaction from raw CBOR bytes.
// See: _oasis-sdk/runtime-sdk/modules/evm/src/types.rs for transaction structure
func decodeEVMTransactionFromCBOR(bodyBytes []byte, isCreate bool) *EVMTransactionInfo {
	info := &EVMTransactionInfo{}

	if isCreate {
		info.Type = "create"
	} else {
		info.Type = "call"
	}

	// Try to unmarshal as map
	var bodyMap map[interface{}]interface{}
	if err := cbor.Unmarshal(bodyBytes, &bodyMap); err != nil {
		info.RawError = fmt.Sprintf("cbor unmarshal failed: %v", err)
		return info
	}

	// Extract from address (20 bytes)
	if fromBytes, ok := bodyMap["from"].([]byte); ok && len(fromBytes) == 20 {
		info.From = "0x" + hex.EncodeToString(fromBytes)
	}

	// Extract to address (20 bytes, calls only)
	if !isCreate {
		if toBytes, ok := bodyMap["address"].([]byte); ok && len(toBytes) == 20 {
			info.To = "0x" + hex.EncodeToString(toBytes)
		} else if toBytes, ok := bodyMap["to"].([]byte); ok && len(toBytes) == 20 {
			info.To = "0x" + hex.EncodeToString(toBytes)
		}
	}

	// Extract value (U256)
	if valueBytes, ok := bodyMap["value"].([]byte); ok {
		info.Value = formatU256(valueBytes)
	}

	// Extract gas_limit (u64)
	if gasLimit, ok := bodyMap["gas_limit"].(uint64); ok {
		info.GasLimit = gasLimit
	} else if gasLimit, ok := bodyMap["gas"].(uint64); ok {
		info.GasLimit = gasLimit
	}

	// Extract gas_price (U256)
	if gasPriceBytes, ok := bodyMap["gas_price"].([]byte); ok {
		info.GasPrice = formatU256(gasPriceBytes)
	}

	// Extract nonce (u64)
	if nonce, ok := bodyMap["nonce"].(uint64); ok {
		info.Nonce = nonce
	}

	// Extract data/init_code
	dataKey := "data"
	if isCreate {
		dataKey = "init_code"
	}
	if dataBytes, ok := bodyMap[dataKey].([]byte); ok {
		info.DataSize = len(dataBytes)
		info.DataDump = formatRawValue(dataBytes, TruncateLongLen)
	}

	return info
}

// decodeEVMTxOutput decodes EVM transaction output artifacts.
// Execution errors (transaction failures) are stored in info.ErrorExecution field.
func decodeEVMTxOutput(value []byte) *EVMTxOutputInfo {
	info := &EVMTxOutputInfo{}

	var oa cborRuntimeOutputArtifacts
	if err := cbor.Unmarshal(value, &oa); err != nil {
		info.RawError = fmt.Sprintf("failed to unmarshal RuntimeOutputArtifacts: %v", err)
		return info
	}

	if len(oa.Output) == 0 {
		info.SuccessExecution = true
		return info
	}

	// Try to decode as CBOR CallResult
	var result map[interface{}]interface{}
	if err := cbor.Unmarshal(oa.Output, &result); err != nil {
		// Raw bytes - success
		info.SuccessExecution = true
		info.ResultSize = len(oa.Output)
		info.ResultDump = formatRawValue(oa.Output, TruncateLongLen)
		return info
	}

	// Check success/failure
	if okVal, exists := result["ok"]; exists {
		info.SuccessExecution = true
		if okBytes, ok := okVal.([]byte); ok {
			info.ResultSize = len(okBytes)
			info.ResultDump = formatRawValue(okBytes, TruncateLongLen)
		}
	} else if failVal, exists := result["fail"]; exists {
		info.SuccessExecution = false
		if failMap, ok := failVal.(map[interface{}]interface{}); ok {
			if msgVal, ok := failMap["message"].(string); ok {
				info.ErrorExecution = msgVal // Execution error, keep in struct
			} else {
				info.ErrorExecution = fmt.Sprintf("%v", failMap)
			}
		}
	}

	return info
}
