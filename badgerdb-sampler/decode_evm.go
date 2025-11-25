
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
			evmInfo.Address = fmt.Sprintf("0x%x", data[:20])
			data = data[20:]
		}
		// Value is contract bytecode
		evmInfo.BytecodeSize = len(value)
		if len(value) > 0 {
			previewLen := 32
			if len(value) < previewLen {
				previewLen = len(value)
			}
			evmInfo.BytecodeRaw = fmt.Sprintf("0x%x", value[:previewLen])
		}

	case 0x02: // STORAGES: evm + 0x02 + H160 (address) + H256 (slot)
		evmInfo.StorageType = "storage"
		if len(data) >= 20 {
			evmInfo.Address = fmt.Sprintf("0x%x", data[:20])
			data = data[20:]
			if len(data) >= 32 {
				evmInfo.StorageSlot = fmt.Sprintf("0x%x", data[:32])
			}
		}
		// Value is H256 storage value
		if len(value) == 32 {
			evmInfo.StorageValue = fmt.Sprintf("0x%x", value)
		}

	case 0x03: // BLOCK_HASHES: evm + 0x03 + RuntimeHeight (uint64 BE)
		evmInfo.StorageType = "block_hash"
		if len(data) >= 8 {
			evmInfo.RuntimeHeight = binary.BigEndian.Uint64(data[:8])
		}
		// Value is H256 block hash
		if len(value) == 32 {
			evmInfo.BlockHash = fmt.Sprintf("0x%x", value)
		}

	case 0x04: // CONFIDENTIAL_STORAGES: evm + 0x04 + H160 (address) + H256 (slot)
		evmInfo.StorageType = "confidential_storage"
		if len(data) >= 20 {
			evmInfo.Address = fmt.Sprintf("0x%x", data[:20])
			data = data[20:]
			if len(data) >= 32 {
				evmInfo.StorageSlot = fmt.Sprintf("0x%x", data[:32])
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
func decodeEVMEvent(value []byte) *EVMEventInfo {
	info := &EVMEventInfo{}

	// Value is CBOR: [event_code, {address, topics, data}]
	var eventWrapper []interface{}
	if err := cbor.Unmarshal(value, &eventWrapper); err != nil {
		info.DecodeError = err.Error()
		return info
	}
	if len(eventWrapper) < 2 {
		info.DecodeError = "invalid format"
		return info
	}

	eventData, ok := eventWrapper[1].(map[interface{}]interface{})
	if !ok {
		info.DecodeError = "invalid format"
		return info
	}

	// Extract address (H160)
	if addrBytes, ok := eventData["address"].([]byte); ok && len(addrBytes) == 20 {
		info.Address = fmt.Sprintf("0x%x", addrBytes)
	} else {
		// Continue decoding even with invalid address - topics and data may still be useful
		info.DecodeError = "invalid address (continuing decode)"
	}

	// Extract topics (Vec<H256>)
	if topicsArray, ok := eventData["topics"].([]interface{}); ok {
		info.TopicCount = len(topicsArray)
		for i, topic := range topicsArray {
			if topicBytes, ok := topic.([]byte); ok && len(topicBytes) == 32 {
				topicHex := fmt.Sprintf("%x", topicBytes)
				info.Topics = append(info.Topics, "0x"+topicHex)
				if i == 0 {
					info.EventHash = "0x" + topicHex
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
		info.DataRaw = truncateHex0x(dataBytes, 128) // 64 bytes = 128 hex chars
	}

	return info
}

// decodeEVMTxInput decodes EVM transaction input artifacts.
func decodeEVMTxInput(key []byte, value []byte) *EVMTxInputInfo {
	info := &EVMTxInputInfo{InputSize: len(value)}

	if len(key) >= 33 {
		info.TxHash = fmt.Sprintf("0x%x", key[1:33])
	}

	var ia RuntimeInputArtifacts
	if err := cbor.Unmarshal(value, &ia); err != nil {
		info.DecodeError = err.Error()
		return info
	}
	info.BatchOrder = ia.BatchOrder

	var call map[interface{}]interface{}
	if err := cbor.Unmarshal(ia.Input, &call); err == nil {
		if methodVal, ok := call["method"].(string); ok {
			info.Method = methodVal
			// Decode EVM call methods that have body structure
			if methodVal == "evm.Call" || methodVal == "evm.Create" || methodVal == "evm.SimulateCall" || methodVal == "evm.EstimateGas" {
				isCreate := methodVal == "evm.Create"
				info.EVMCall = decodeEVMCallBody(call, isCreate)
			}
		} else {
			info.DecodeError = "method field missing or invalid in call structure"
		}
	} else {
		info.DecodeError = fmt.Sprintf("failed to unmarshal call structure: %v", err)
	}

	return info
}

// decodeEVMCallBody decodes EVM call/create body from SDK Call structure.
func decodeEVMCallBody(call map[interface{}]interface{}, isCreate bool) *EVMCallInfo {
	info := &EVMCallInfo{}
	if isCreate {
		info.Type = "create"
	} else {
		info.Type = "call"
	}

	bodyBytes, ok := call["body"].([]byte)
	if !ok {
		return info
	}

	var bodyMap map[interface{}]interface{}
	if err := cbor.Unmarshal(bodyBytes, &bodyMap); err != nil {
		return info
	}

	// Extract address (calls only)
	if !isCreate {
		if addrBytes, ok := bodyMap["address"].([]byte); ok && len(addrBytes) == 20 {
			info.Address = fmt.Sprintf("0x%x", addrBytes)
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
		info.DataRaw = truncateHex0x(dataBytes, 64) // 32 bytes = 64 hex chars
	}

	return info
}

// decodeEVMTxOutput decodes EVM transaction output artifacts.
func decodeEVMTxOutput(value []byte) *EVMTxOutputInfo {
	info := &EVMTxOutputInfo{OutputSize: len(value)}

	var oa RuntimeOutputArtifacts
	if err := cbor.Unmarshal(value, &oa); err != nil {
		info.DecodeError = err.Error()
		return info
	}

	if len(oa.Output) == 0 {
		info.Success = true
		return info
	}

	// Try to decode as CBOR CallResult
	var result map[interface{}]interface{}
	if err := cbor.Unmarshal(oa.Output, &result); err != nil {
		// Raw bytes - success
		info.Success = true
		info.ResultSize = len(oa.Output)
		info.ResultRaw = truncateHex0x(oa.Output, 64) // 32 bytes = 64 hex chars
		return info
	}

	// Check success/failure
	if okVal, exists := result["ok"]; exists {
		info.Success = true
		if okBytes, ok := okVal.([]byte); ok {
			info.ResultSize = len(okBytes)
			info.ResultRaw = truncateHex0x(okBytes, 64)
		}
	} else if failVal, exists := result["fail"]; exists {
		info.Success = false
		if failMap, ok := failVal.(map[interface{}]interface{}); ok {
			if msgVal, ok := failMap["message"].(string); ok {
				info.Error = msgVal
			} else {
				info.Error = fmt.Sprintf("%v", failMap)
			}
		}
	}

	return info
}
