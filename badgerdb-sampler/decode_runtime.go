package main

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/fxamacker/cbor/v2"
)

// decodeRuntimeMkvsKey parses runtime-mkvs key and returns structured info.
// Keys use keyformat encoding: [type_byte][data...]
// See: _oasis-core/go/storage/mkvs/db/badger/badger.go:31-66
func decodeRuntimeMkvsKey(key []byte) RuntimeMkvsKeyInfo {
	info := RuntimeMkvsKeyInfo{}

	if len(key) < 1 {
		info.KeyType = "unknown"
		return info
	}

	// First byte is the key type prefix
	prefixByte := key[0]
	data := key[1:]

	switch prefixByte {
	case 0x00:
		info.KeyType = "node"
		info.Hash = truncateHex(data, TruncateHashSize)
	case 0x01:
		info.KeyType = "write_log"
		if len(data) >= 8 {
			info.RuntimeHeight = binary.BigEndian.Uint64(data[0:8])
		}
	case 0x02:
		info.KeyType = "roots_metadata"
		if len(data) >= 8 {
			info.RuntimeHeight = binary.BigEndian.Uint64(data[0:8])
		}
	case 0x03:
		info.KeyType = "root_updated_nodes"
		if len(data) >= 8 {
			info.RuntimeHeight = binary.BigEndian.Uint64(data[0:8])
		}
	case 0x04:
		info.KeyType = "metadata"
	default:
		info.KeyType = fmt.Sprintf("unknown_%02x", prefixByte)
	}

	return info
}

// decodeRuntimeMkvsValue decodes runtime MKVS value and returns structured info.
// See: _oasis-core/go/storage/mkvs/node/node.go:26-32 (prefixes), 294-309 (InternalNode), 531-537 (LeafNode)
func decodeRuntimeMkvsValue(keyType string, value []byte) RuntimeMkvsNodeInfo {
	info := RuntimeMkvsNodeInfo{NodeSize: len(value)}

	if len(value) == 0 {
		info.NodeType = "empty"
		return info
	}

	if keyType != "node" {
		info.NodeType = "non_node"
		return info
	}

	// Add raw node hex dump
	info.NodeHex = truncateHex(value, TruncateLongSize)

	// Parse MKVS node: 0x00=leaf, 0x01=internal, 0x02=nil
	switch value[0] {
	case 0x00: // LeafNode: [2-byte keySize LE][key][4-byte valueSize LE][value]
		info.NodeType = "leaf"
		data := value[1:]

		if len(data) < 2 {
			info.LeafError = "key length missing"
			return info
		}

		keySize := int(binary.LittleEndian.Uint16(data[0:2]))
		data = data[2:]

		if len(data) < keySize {
			info.LeafError = fmt.Sprintf("key truncated (expected %d bytes)", keySize)
			return info
		}

		key := data[:keySize]
		data = data[keySize:]

		// Extract module name from key
		module := extractModuleName(key)

		leaf := &RuntimeMkvsLeafInfo{
			Module:  module,
			KeySize: keySize,
			KeyHex:  truncateHex(key, TruncateLongSize),
		}

		if len(data) < 4 {
			info.LeafError = "value length missing"
			info.Leaf = leaf
			return info
		}

		valueSize := int(binary.LittleEndian.Uint32(data[0:4]))
		data = data[4:]
		leaf.ValueSize = valueSize

		if len(data) < valueSize {
			info.LeafError = fmt.Sprintf("value truncated (expected %d bytes)", valueSize)
			info.Leaf = leaf
			return info
		}

		leafValue := data[:valueSize]
		leaf.ValueHex = truncateHex(leafValue, TruncateLongSize)
		leaf.Value = decodeRuntimeLeafValue(module, key, leafValue)
		info.Leaf = leaf
		return info

	case 0x01: // InternalNode: [2-byte labelBits LE][label][leaf/nil marker][hashes]
		info.NodeType = "internal"
		data := value[1:]

		if len(data) < 2 {
			info.InternalError = "label bits missing"
			return info
		}

		labelBits := binary.LittleEndian.Uint16(data[0:2])
		data = data[2:]

		labelBytes := (int(labelBits) + 7) / 8
		if len(data) < labelBytes+1 {
			info.InternalError = "label truncated"
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
				internal.LeftHash = truncateHex(data[:32], TruncateHashSize)
				data = data[32:]
			}
			if len(data) >= 32 {
				internal.RightHash = truncateHex(data[:32], TruncateHashSize)
			}
		}

		info.Internal = internal
		return info

	case 0x02: // NilNode
		info.NodeType = "nil"
		return info

	default:
		info.NodeType = "unknown"
		info.NodeError = fmt.Sprintf("unknown prefix 0x%02x", value[0])
		return info
	}
}

// decodeRuntimeHistoryKey parses runtime-history key and returns structured info.
// See: _oasis-core/go/runtime/history/db.go:19-31
// Key formats:
//   - 0x01: metadata key (1 byte)
//   - 0x02 + uint64: block key (9 bytes) - round number in big-endian
//   - 0x03 + uint64: round_results key (9 bytes) - round number in big-endian
func decodeRuntimeHistoryKey(key []byte) RuntimeHistoryKeyInfo {
	info := RuntimeHistoryKeyInfo{}

	if len(key) == 0 {
		info.KeyType = "unknown"
		return info
	}

	switch key[0] {
	case 0x01:
		info.KeyType = "metadata"
		if len(key) > 1 {
			info.ExtraData = truncateHex(key[1:], TruncateLongSize)
		}

	case 0x02:
		info.KeyType = "block"
		if len(key) == 9 {
			info.RuntimeHeight = binary.BigEndian.Uint64(key[1:9])
		}

	case 0x03:
		info.KeyType = "round_results"
		if len(key) == 9 {
			info.RuntimeHeight = binary.BigEndian.Uint64(key[1:9])
		}

	default:
		info.KeyType = "unknown"
	}

	return info
}

// decodeRuntimeHistoryValue decodes CBOR-encoded runtime history value and returns structured info.
// See: _oasis-core/go/runtime/history/db.go:34-44 (metadata)
// See: _oasis-core/go/roothash/api/api.go:402-409 (AnnotatedBlock)
// See: _oasis-core/go/roothash/api/results.go:5-17 (RoundResults)
func decodeRuntimeHistoryValue(keyType string, value []byte) RuntimeHistoryValueInfo {
	info := RuntimeHistoryValueInfo{}

	if len(value) == 0 {
		// General error when we don't know which child yet
		switch keyType {
		case "metadata":
			info.MetadataError = "empty value"
		case "block":
			info.BlockError = "empty value"
		case "round_results":
			info.RoundResultsError = "empty value"
		}
		return info
	}

	switch keyType {
	case "metadata":
		var meta cborRuntimeHistoryMetadata
		if err := cbor.Unmarshal(value, &meta); err != nil {
			info.MetadataError = err.Error()
			return info
		}
		info.Metadata = &RuntimeHistoryMetadataInfo{
			Version:             meta.Version,
			RuntimeID:           truncateHex(meta.RuntimeID, TruncateLongSize),
			LastRuntimeHeight:   meta.LastRound,
			LastConsensusHeight: meta.LastConsensusHeight,
		}

	case "block":
		var block cborRuntimeHistoryAnnotatedBlock
		if err := cbor.Unmarshal(value, &block); err != nil {
			info.BlockError = err.Error()
			return info
		}
		blockInfo := &RuntimeHistoryBlockInfo{
			ConsensusHeight: block.Height,
		}
		if block.Block == nil {
			blockInfo.BlockNil = true
		} else {
			h := block.Block.Header
			blockInfo.RuntimeHeight = h.Round
			blockInfo.Timestamp = time.Unix(int64(h.Timestamp), 0).UTC().Format(time.RFC3339)
			blockInfo.HeaderType = headerTypeName(h.HeaderType)
			blockInfo.StateRoot = truncateHex(h.StateRoot, TruncateHashSize)
		}
		info.Block = blockInfo

	case "round_results":
		var results cborRuntimeHistoryRoundResults
		if err := cbor.Unmarshal(value, &results); err != nil {
			info.RoundResultsError = err.Error()
			return info
		}
		info.RoundResults = &RuntimeHistoryRoundResultsInfo{
			MessageCount:        len(results.Messages),
			GoodComputeEntities: len(results.GoodComputeEntities),
			BadComputeEntities:  len(results.BadComputeEntities),
		}
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

// decodeRuntimeLeafValue decodes MKVS leaf value and returns structured info with valueType classification.
// See: _oasis-core/go/runtime/transaction/transaction.go:129-150 (artifacts)
func decodeRuntimeLeafValue(module string, key []byte, value []byte) *RuntimeLeafValueInfo {
	info := &RuntimeLeafValueInfo{}
	info.ValueType = "binary" // default

	// Check for EVM module data
	if len(module) >= 3 && module[:3] == "evm" {
		evmInfo := decodeEVMData(module, key, value)
		if evmInfo != nil {
			info.EVM = evmInfo
			// No error return from decodeEVMData (returns nil on failure)
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
				info.EVMTxInput, info.EVMTxInputError = decodeEVMTxInput(key, value)
				info.ValueType = "io_input"
				return info
			} else if kind == 2 {
				// Output artifact
				info.EVMTxOutput, info.EVMTxOutputError = decodeEVMTxOutput(value)
				info.ValueType = "io_output"
				return info
			}
		}
	}

	// Check for IO event tags
	if len(module) > 8 && module[:8] == "io_event" {
		// Special handling for EVM events
		if len(module) >= 12 && module[9:12] == "evm" {
			info.EVMEvent, info.EVMEventError = decodeEVMEvent(value)
			info.ValueType = "evm_event"
			return info
		}

		// Generic event decoding for non-EVM events
		var decoded interface{}
		if err := cbor.Unmarshal(value, &decoded); err == nil {
			// Use detailed formatter to expand arrays and maps
			info.CBOR = formatCBORDetailed(decoded)
			info.ValueType = "io_event"
		} else {
			info.CBORError = err.Error()
		}
		return info
	}

	// Try CBOR decode for regular state keys
	var decoded interface{}
	if err := cbor.Unmarshal(value, &decoded); err == nil {
		info.CBOR = formatCBOR(decoded, len(value))
		info.ValueType = "cbor"
	} else {
		info.CBORError = err.Error()
	}

	return info
}
