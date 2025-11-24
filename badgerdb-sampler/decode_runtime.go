package main

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/fxamacker/cbor/v2"
)

// decodeKeyRuntimeMkvs parses runtime-mkvs key and returns structured info.
// Key format: [optional db prefix 0x01|0x05][type byte][data...]
// See: _oasis-core/go/storage/mkvs/db/badger/badger.go:31-66
func decodeKeyRuntimeMkvs(key []byte) RuntimeMkvsKeyInfo {
	info := RuntimeMkvsKeyInfo{}

	if len(key) < 1 {
		info.KeyType = "unknown"
		info.DecodeError = "key too short"
		return info
	}

	// Check for dbVersion prefix (0x01 or 0x05)
	prefixByte := key[0]
	var data []byte

	if prefixByte == 0x01 || prefixByte == 0x05 {
		info.DbPrefix = prefixByte
		if len(key) < 2 {
			info.KeyType = "unknown"
			info.DecodeError = "key too short after prefix"
			return info
		}
		prefixByte = key[1]
		data = key[2:]
	} else {
		data = key[1:]
	}

	switch prefixByte {
	case 0x00:
		info.KeyType = "node"
		info.Hash = truncateHex(data, 16)
	case 0x01:
		info.KeyType = "write_log"
		if len(data) >= 8 {
			info.Height = binary.BigEndian.Uint64(data[0:8])
		} else {
			info.DecodeError = "write_log data too short"
		}
	case 0x02:
		info.KeyType = "roots_metadata"
		if len(data) >= 8 {
			info.Height = binary.BigEndian.Uint64(data[0:8])
		} else {
			info.DecodeError = "roots_metadata data too short"
		}
	case 0x03:
		info.KeyType = "root_updated_nodes"
		if len(data) >= 8 {
			info.Height = binary.BigEndian.Uint64(data[0:8])
		} else {
			info.DecodeError = "root_updated_nodes data too short"
		}
	case 0x04:
		info.KeyType = "metadata"
	default:
		info.KeyType = fmt.Sprintf("unknown_%02x", prefixByte)
		info.DecodeError = fmt.Sprintf("unknown key prefix 0x%02x", prefixByte)
	}

	return info
}

// decodeValueRuntimeMkvs decodes runtime MKVS value and returns structured info.
// See: _oasis-core/go/storage/mkvs/node/node.go:26-32 (prefixes), 294-309 (InternalNode), 531-537 (LeafNode)
func decodeValueRuntimeMkvs(keyType string, value []byte) RuntimeMkvsNodeInfo {
	info := RuntimeMkvsNodeInfo{Size: len(value)}

	if len(value) == 0 {
		info.NodeType = "empty"
		return info
	}

	if keyType != "node" {
		info.NodeType = "non_node"
		return info
	}

	// Parse MKVS node: 0x00=leaf, 0x01=internal, 0x02=nil
	switch value[0] {
	case 0x00: // LeafNode: [2-byte keyLen LE][key][4-byte valueLen LE][value]
		info.NodeType = "leaf"
		data := value[1:]

		if len(data) < 2 {
			info.DecodeError = "key length missing"
			return info
		}

		keyLen := int(binary.LittleEndian.Uint16(data[0:2]))
		data = data[2:]

		if len(data) < keyLen {
			info.DecodeError = fmt.Sprintf("key truncated (expected %d bytes)", keyLen)
			return info
		}

		key := data[:keyLen]
		data = data[keyLen:]

		// Extract module name from key
		module := extractModuleName(key)

		leaf := &RuntimeMkvsLeafInfo{
			Module: module,
			KeyLen: keyLen,
			Key:    truncateHex(key, 32),
		}

		if len(data) < 4 {
			info.DecodeError = "value length missing"
			info.Leaf = leaf
			return info
		}

		valueLen := int(binary.LittleEndian.Uint32(data[0:4]))
		data = data[4:]
		leaf.ValueLen = valueLen

		if len(data) < valueLen {
			info.DecodeError = fmt.Sprintf("value truncated (expected %d bytes)", valueLen)
			info.Leaf = leaf
			return info
		}

		leafValue := data[:valueLen]
		leaf.DecodedValue = decodeLeafValue(module, key, leafValue)
		info.Leaf = leaf
		return info

	case 0x01: // InternalNode: [2-byte labelBits LE][label][leaf/nil marker][hashes]
		info.NodeType = "internal"
		data := value[1:]

		if len(data) < 2 {
			info.DecodeError = "label bits missing"
			return info
		}

		labelBits := binary.LittleEndian.Uint16(data[0:2])
		data = data[2:]

		labelBytes := (int(labelBits) + 7) / 8
		if len(data) < labelBytes+1 {
			info.DecodeError = "label truncated"
			info.Internal = &RuntimeMkvsInternalInfo{LabelBits: labelBits}
			return info
		}

		data = data[labelBytes:] // skip label

		internal := &RuntimeMkvsInternalInfo{LabelBits: labelBits}
		hasLeaf := data[0] == 0x00
		internal.HasLeaf = hasLeaf

		if !hasLeaf {
			data = data[1:] // skip nil marker
			// Extract child hashes
			if len(data) >= 32 {
				internal.LeftHash = truncateHex(data[:32], 16)
				data = data[32:]
			}
			if len(data) >= 32 {
				internal.RightHash = truncateHex(data[:32], 16)
			}
		}

		info.Internal = internal
		return info

	case 0x02: // NilNode
		info.NodeType = "nil"
		return info

	default:
		info.NodeType = "unknown"
		info.DecodeError = fmt.Sprintf("unknown prefix 0x%02x", value[0])
		return info
	}
}

// decodeKeyRuntimeHistory parses runtime-history key and returns structured info.
// See: _oasis-core/go/runtime/history/db.go:19-31
// Key formats:
//   - 0x01: metadata key (1 byte)
//   - 0x02 + uint64: block key (9 bytes) - round number in big-endian
//   - 0x03 + uint64: round_results key (9 bytes) - round number in big-endian
func decodeKeyRuntimeHistory(key []byte) RuntimeHistoryKeyInfo {
	info := RuntimeHistoryKeyInfo{}

	if len(key) == 0 {
		info.KeyType = "unknown"
		info.DecodeError = "empty key"
		return info
	}

	switch key[0] {
	case 0x01:
		info.KeyType = "metadata"
		if len(key) > 1 {
			info.ExtraData = fmt.Sprintf("%x", key[1:])
		}

	case 0x02:
		info.KeyType = "block"
		if len(key) == 9 {
			info.Height = binary.BigEndian.Uint64(key[1:9])
		} else {
			info.DecodeError = "block key wrong length"
		}

	case 0x03:
		info.KeyType = "round_results"
		if len(key) == 9 {
			info.Height = binary.BigEndian.Uint64(key[1:9])
		} else {
			info.DecodeError = "round_results key wrong length"
		}

	default:
		info.KeyType = "unknown"
		info.DecodeError = fmt.Sprintf("unknown key prefix 0x%02x", key[0])
	}

	return info
}

// decodeValueRuntimeHistory decodes CBOR-encoded runtime history value and returns structured info.
// See: _oasis-core/go/runtime/history/db.go:34-44 (metadata)
// See: _oasis-core/go/roothash/api/api.go:402-409 (AnnotatedBlock)
// See: _oasis-core/go/roothash/api/results.go:5-17 (RoundResults)
func decodeValueRuntimeHistory(keyType string, value []byte) RuntimeHistoryValueInfo {
	info := RuntimeHistoryValueInfo{
		KeyType: keyType,
		Size:    len(value),
	}

	if len(value) == 0 {
		info.DecodeError = "empty value"
		return info
	}

	switch keyType {
	case "metadata":
		var meta RuntimeHistoryMetadata
		if err := cbor.Unmarshal(value, &meta); err != nil {
			info.DecodeError = err.Error()
			return info
		}
		info.Metadata = &RuntimeHistoryMetadataInfo{
			Version:             meta.Version,
			RuntimeID:           truncateHex(meta.RuntimeID, 16),
			LastRound:           meta.LastRound,
			LastConsensusHeight: meta.LastConsensusHeight,
		}

	case "block":
		var block RuntimeHistoryAnnotatedBlock
		if err := cbor.Unmarshal(value, &block); err != nil {
			info.DecodeError = err.Error()
			return info
		}
		blockInfo := &RuntimeHistoryBlockInfo{
			ConsensusHeight: block.Height,
		}
		if block.Block == nil {
			blockInfo.BlockNil = true
		} else {
			h := block.Block.Header
			blockInfo.Round = h.Round
			blockInfo.Timestamp = time.Unix(int64(h.Timestamp), 0).UTC().Format(time.RFC3339)
			blockInfo.HeaderType = headerTypeName(h.HeaderType)
			blockInfo.StateRoot = truncateHex(h.StateRoot, 16)
		}
		info.Block = blockInfo

	case "round_results":
		var results RuntimeHistoryRoundResults
		if err := cbor.Unmarshal(value, &results); err != nil {
			info.DecodeError = err.Error()
			return info
		}
		info.RoundResults = &RuntimeHistoryRoundResultsInfo{
			MessageCount:        len(results.Messages),
			GoodComputeEntities: len(results.GoodComputeEntities),
			BadComputeEntities:  len(results.BadComputeEntities),
		}

	default:
		info.DecodeError = fmt.Sprintf("unknown key type: %s", keyType)
	}

	return info
}

// headerTypeName converts header type byte to string
func headerTypeName(headerType uint8) string {
	switch headerType {
	case 0:
		return "Invalid"
	case 1:
		return "Normal"
	case 2:
		return "RoundFailed"
	case 3:
		return "EpochTransition"
	case 4:
		return "Suspended"
	default:
		return fmt.Sprintf("Unknown(%d)", headerType)
	}
}

// decodeLeafValue decodes MKVS leaf value and returns structured info.
// See: _oasis-core/go/runtime/transaction/transaction.go:129-150 (artifacts)
func decodeLeafValue(module string, key []byte, value []byte) *RuntimeLeafValueInfo {
	info := &RuntimeLeafValueInfo{}

	// Check for EVM module data
	if len(module) >= 3 && module[:3] == "evm" {
		evmInfo := decodeEVMData(module, key, value)
		if evmInfo != nil {
			info.EVM = evmInfo
			// Set value_type based on EVM storage type
			switch evmInfo.StorageType {
			case "code":
				info.ValueType = "evm_code"
			case "storage", "confidential_storage":
				info.ValueType = "evm_storage"
			case "block_hash":
				info.ValueType = "evm_block_hash"
			default:
				info.ValueType = "evm_unknown"
			}
			return info
		}
	}

	// Check for IO transaction artifacts
	if len(module) > 5 && module[:5] == "io_tx" {
		// Determine artifact kind from key
		if len(key) >= 34 {
			kind := key[33]
			if kind == 1 {
				// Input artifact
				var ia RuntimeInputArtifacts
				if err := cbor.Unmarshal(value, &ia); err == nil {
					info.ValueType = "io_input"
					info.InputSize = len(ia.Input)
					info.BatchOrder = ia.BatchOrder
					return info
				}
			} else if kind == 2 {
				// Output artifact
				var oa RuntimeOutputArtifacts
				if err := cbor.Unmarshal(value, &oa); err == nil {
					info.ValueType = "io_output"
					info.OutputSize = len(oa.Output)
					return info
				}
			}
		}
	}

	// Check for IO event tags
	if len(module) > 8 && module[:8] == "io_event" {
		var decoded interface{}
		if err := cbor.Unmarshal(value, &decoded); err == nil {
			info.ValueType = "io_event"
			info.DecodedValue = formatCBOR(decoded, len(value))
			return info
		}
		info.ValueType = "binary"
		info.BinarySize = len(value)
		return info
	}

	// Try CBOR decode for regular state keys
	var decoded interface{}
	if err := cbor.Unmarshal(value, &decoded); err == nil {
		info.ValueType = "cbor"
		info.DecodedValue = formatCBOR(decoded, len(value))
		return info
	}

	info.ValueType = "binary"
	info.BinarySize = len(value)
	return info
}

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
			evmInfo.Address = fmt.Sprintf("%x", data[:20])
			data = data[20:]
		}
		// Value is contract bytecode
		evmInfo.CodeSize = len(value)
		if len(value) > 0 {
			previewLen := 32
			if len(value) < previewLen {
				previewLen = len(value)
			}
			evmInfo.CodePreview = fmt.Sprintf("%x", value[:previewLen])
		}

	case 0x02: // STORAGES: evm + 0x02 + H160 (address) + H256 (slot)
		evmInfo.StorageType = "storage"
		if len(data) >= 20 {
			evmInfo.Address = fmt.Sprintf("%x", data[:20])
			data = data[20:]
			if len(data) >= 32 {
				evmInfo.StorageSlot = fmt.Sprintf("%x", data[:32])
			}
		}
		// Value is H256 storage value
		if len(value) == 32 {
			evmInfo.StorageValue = fmt.Sprintf("%x", value)
		}

	case 0x03: // BLOCK_HASHES: evm + 0x03 + Round (uint64 BE)
		evmInfo.StorageType = "block_hash"
		if len(data) >= 8 {
			evmInfo.Round = binary.BigEndian.Uint64(data[:8])
		}
		// Value is H256 block hash
		if len(value) == 32 {
			evmInfo.BlockHash = fmt.Sprintf("%x", value)
		}

	case 0x04: // CONFIDENTIAL_STORAGES: evm + 0x04 + H160 (address) + H256 (slot)
		evmInfo.StorageType = "confidential_storage"
		if len(data) >= 20 {
			evmInfo.Address = fmt.Sprintf("%x", data[:20])
			data = data[20:]
			if len(data) >= 32 {
				evmInfo.StorageSlot = fmt.Sprintf("%x", data[:32])
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
