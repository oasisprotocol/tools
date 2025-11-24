package main

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/fxamacker/cbor/v2"
)

// Runtime history CBOR types (from oasis-core/go/runtime/history/db.go)

// RuntimeHistoryMetadata represents the metadata stored in runtime history DB
type RuntimeHistoryMetadata struct {
	RuntimeID           []byte `cbor:"runtime_id"`
	Version             uint64 `cbor:"version"`
	LastConsensusHeight int64  `cbor:"last_consensus_height"`
	LastRound           uint64 `cbor:"last_round"`
}

// RuntimeHistoryAnnotatedBlock represents an annotated block in runtime history
type RuntimeHistoryAnnotatedBlock struct {
	Height int64                `cbor:"consensus_height"`
	Block  *RuntimeHistoryBlock `cbor:"block"`
}

// RuntimeHistoryBlock represents a runtime block
type RuntimeHistoryBlock struct {
	Header RuntimeHistoryBlockHeader `cbor:"header"`
}

// RuntimeHistoryBlockHeader represents a runtime block header
type RuntimeHistoryBlockHeader struct {
	Version        uint16 `cbor:"version"`
	Namespace      []byte `cbor:"namespace"`
	Round          uint64 `cbor:"round"`
	Timestamp      uint64 `cbor:"timestamp"`
	HeaderType     uint8  `cbor:"header_type"`
	PreviousHash   []byte `cbor:"previous_hash"`
	IORoot         []byte `cbor:"io_root"`
	StateRoot      []byte `cbor:"state_root"`
	MessagesHash   []byte `cbor:"messages_hash"`
	InMessagesHash []byte `cbor:"in_msgs_hash"`
}

// RuntimeHistoryRoundResults represents round results in runtime history
type RuntimeHistoryRoundResults struct {
	Messages            []RuntimeHistoryMessageEvent `cbor:"messages,omitempty"`
	GoodComputeEntities [][]byte                     `cbor:"good_compute_entities,omitempty"`
	BadComputeEntities  [][]byte                     `cbor:"bad_compute_entities,omitempty"`
}

// RuntimeHistoryMessageEvent represents a message event
type RuntimeHistoryMessageEvent struct {
	Module string          `cbor:"module,omitempty"`
	Code   uint32          `cbor:"code,omitempty"`
	Index  uint32          `cbor:"index,omitempty"`
	Result cbor.RawMessage `cbor:"result,omitempty"`
}

// RuntimeInputArtifacts represents input transaction artifacts stored in IO tree
// From oasis-core/go/runtime/transaction/transaction.go
type RuntimeInputArtifacts struct {
	_          struct{} `cbor:",toarray"`
	Input      []byte
	BatchOrder uint32
}

// RuntimeOutputArtifacts represents output transaction artifacts stored in IO tree
// From oasis-core/go/runtime/transaction/transaction.go
type RuntimeOutputArtifacts struct {
	_      struct{} `cbor:",toarray"`
	Output []byte
}

// decodeKeyRuntimeMkvs parses runtime-mkvs key
// Key format: [prefix byte][type byte][data...]
func decodeKeyRuntimeMkvs(key []byte) (string, string) {
	if len(key) < 1 {
		return "unknown", fmt.Sprintf("%x", key)
	}

	// Check for dbVersion prefix (0x01 or 0x05)
	prefixByte := key[0]
	var data []byte

	if prefixByte == 0x01 || prefixByte == 0x05 {
		if len(key) < 2 {
			return "unknown", fmt.Sprintf("%x", key)
		}
		prefixByte = key[1]
		data = key[2:]
	} else {
		data = key[1:]
	}

	switch prefixByte {
	case 0x00:
		return "node", fmt.Sprintf("node{hash: %s}", truncateHex(data, 16))
	case 0x01:
		if len(data) >= 8 {
			height := binary.BigEndian.Uint64(data[0:8])
			return "write_log", fmt.Sprintf("write_log{height:%d}", height)
		}
		return "write_log", "write_log{<malformed>}"
	case 0x02:
		if len(data) >= 8 {
			height := binary.BigEndian.Uint64(data[0:8])
			return "roots_metadata", fmt.Sprintf("roots_metadata{height:%d}", height)
		}
		return "roots_metadata", "roots_metadata{<malformed>}"
	case 0x03:
		if len(data) >= 8 {
			height := binary.BigEndian.Uint64(data[0:8])
			return "root_updated_nodes", fmt.Sprintf("root_updated_nodes{height:%d}", height)
		}
		return "root_updated_nodes", "root_updated_nodes{<malformed>}"
	case 0x04:
		return "metadata", "metadata"
	default:
		return fmt.Sprintf("unknown_%02x", prefixByte), fmt.Sprintf("%x", key)
	}
}

// decodeValueRuntimeMkvs decodes runtime MKVS value with inline node parsing
func decodeValueRuntimeMkvs(keyType string, value []byte) string {
	if len(value) == 0 {
		return "Empty"
	}

	if keyType != "node" {
		return fmt.Sprintf("MKVS %s (size: %d bytes)", keyType, len(value))
	}

	// Parse MKVS node: 0x00=leaf, 0x01=internal, 0x02=nil
	switch value[0] {
	case 0x00: // LeafNode: [2-byte keyLen LE][key][4-byte valueLen LE][value]
		data := value[1:]
		if len(data) < 2 {
			return "LeafNode{<malformed>}"
		}

		keyLen := int(binary.LittleEndian.Uint16(data[0:2]))
		data = data[2:]
		if len(data) < keyLen {
			return fmt.Sprintf("LeafNode{keyLen=%d, <truncated>}", keyLen)
		}

		key := data[:keyLen]
		data = data[keyLen:]

		// Extract module name from key (runtime-specific: evm, contracts, accounts, etc.)
		module := extractModuleName(key)

		if len(data) < 4 {
			return fmt.Sprintf("LeafNode{module=%s, <no value>}", module)
		}

		valueLen := int(binary.LittleEndian.Uint32(data[0:4]))
		data = data[4:]
		if len(data) < valueLen {
			return fmt.Sprintf("LeafNode{module=%s, valueLen=%d, <truncated>}", module, valueLen)
		}

		leafValue := data[:valueLen]

		// Try to decode based on module type
		valueDesc := decodeLeafValue(module, key, leafValue)
		return fmt.Sprintf("LeafNode{module=%s, value=%s}", module, valueDesc)

	case 0x01: // InternalNode: [2-byte labelBits LE][label][leaf/nil marker][hashes]
		data := value[1:]
		if len(data) < 2 {
			return "InternalNode{<malformed>}"
		}

		labelBits := binary.LittleEndian.Uint16(data[0:2])
		data = data[2:]

		labelBytes := (int(labelBits) + 7) / 8
		if len(data) < labelBytes+1 {
			return fmt.Sprintf("InternalNode{label=%d bits, <truncated>}", labelBits)
		}

		data = data[labelBytes:] // skip label

		hasLeaf := data[0] == 0x00
		if hasLeaf {
			return fmt.Sprintf("InternalNode{label=%d bits, has_leaf=true}", labelBits)
		}

		data = data[1:] // skip nil marker

		// Extract child hashes
		var left, right string
		if len(data) >= 32 {
			left = truncateHex(data[:32], 16)
			data = data[32:]
		}
		if len(data) >= 32 {
			right = truncateHex(data[:32], 16)
		}

		return fmt.Sprintf("InternalNode{label=%d bits, left=%s, right=%s}", labelBits, left, right)

	case 0x02: // NilNode
		return "NilNode{}"

	default:
		return fmt.Sprintf("Unknown node prefix 0x%02x (size: %d)", value[0], len(value))
	}
}

// decodeKeyRuntimeHistory parses runtime-history key using binary prefix format
// Key formats from oasis-core/go/runtime/history/db.go:
//   - 0x01: metadata key (1 byte)
//   - 0x02 + uint64: block key (9 bytes) - round number in big-endian
//   - 0x03 + uint64: round_results key (9 bytes) - round number in big-endian
func decodeKeyRuntimeHistory(key []byte) (string, string) {
	if len(key) == 0 {
		return "unknown", ""
	}

	switch key[0] {
	case 0x01:
		// Metadata key - just the prefix byte
		if len(key) == 1 {
			return "metadata", "metadata"
		}
		// Unexpected extra data after metadata prefix
		return "metadata", fmt.Sprintf("metadata (extra: %x)", key[1:])

	case 0x02:
		// Block key - prefix + 8-byte round number
		if len(key) == 9 {
			round := binary.BigEndian.Uint64(key[1:9])
			return "block", fmt.Sprintf("round:%d", round)
		}
		return "block", fmt.Sprintf("block (malformed, len=%d)", len(key))

	case 0x03:
		// Round results key - prefix + 8-byte round number
		if len(key) == 9 {
			round := binary.BigEndian.Uint64(key[1:9])
			return "round_results", fmt.Sprintf("round:%d", round)
		}
		return "round_results", fmt.Sprintf("round_results (malformed, len=%d)", len(key))

	default:
		return "unknown", fmt.Sprintf("%x", key)
	}
}

// decodeValueRuntimeHistory decodes CBOR-encoded runtime history value
func decodeValueRuntimeHistory(keyType string, value []byte) string {
	if len(value) == 0 {
		return "Empty"
	}

	switch keyType {
	case "metadata":
		var meta RuntimeHistoryMetadata
		if err := cbor.Unmarshal(value, &meta); err != nil {
			return fmt.Sprintf("Failed to decode metadata: %v (size: %d)", err, len(value))
		}
		runtimeID := truncateHex(meta.RuntimeID, 16)
		return fmt.Sprintf("Metadata{version: %d, runtime_id: %s..., last_round: %d, last_consensus_height: %d}",
			meta.Version, runtimeID, meta.LastRound, meta.LastConsensusHeight)

	case "block":
		var block RuntimeHistoryAnnotatedBlock
		if err := cbor.Unmarshal(value, &block); err != nil {
			return fmt.Sprintf("Failed to decode block: %v (size: %d)", err, len(value))
		}
		if block.Block == nil {
			return fmt.Sprintf("AnnotatedBlock{consensus_height: %d, block: nil}", block.Height)
		}
		h := block.Block.Header
		// Format timestamp as human-readable
		ts := time.Unix(int64(h.Timestamp), 0).UTC().Format(time.RFC3339)
		headerType := headerTypeName(h.HeaderType)
		stateRoot := truncateHex(h.StateRoot, 16)
		return fmt.Sprintf("AnnotatedBlock{consensus_height: %d, round: %d, timestamp: %s, header_type: %s, state_root: %s...}",
			block.Height, h.Round, ts, headerType, stateRoot)

	case "round_results":
		var results RuntimeHistoryRoundResults
		if err := cbor.Unmarshal(value, &results); err != nil {
			return fmt.Sprintf("Failed to decode round_results: %v (size: %d)", err, len(value))
		}
		return fmt.Sprintf("RoundResults{messages: %d, good_entities: %d, bad_entities: %d}",
			len(results.Messages), len(results.GoodComputeEntities), len(results.BadComputeEntities))

	default:
		return fmt.Sprintf("Runtime %s data (size: %d bytes)", keyType, len(value))
	}
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

// decodeLeafValue decodes MKVS leaf value based on module type
func decodeLeafValue(module string, key []byte, value []byte) string {
	// Check for IO transaction artifacts
	if len(module) > 5 && module[:5] == "io_tx" {
		// Determine artifact kind from key
		if len(key) >= 34 {
			kind := key[33]
			if kind == 1 {
				// Input artifact
				var ia RuntimeInputArtifacts
				if err := cbor.Unmarshal(value, &ia); err == nil {
					return fmt.Sprintf("RuntimeInputArtifacts{input_size=%d, batch_order=%d}", len(ia.Input), ia.BatchOrder)
				}
			} else if kind == 2 {
				// Output artifact
				var oa RuntimeOutputArtifacts
				if err := cbor.Unmarshal(value, &oa); err == nil {
					return fmt.Sprintf("RuntimeOutputArtifacts{output_size=%d}", len(oa.Output))
				}
			}
		}
	}

	// Check for IO event tags
	if len(module) > 8 && module[:8] == "io_event" {
		// Event tag value is typically CBOR
		var decoded interface{}
		if err := cbor.Unmarshal(value, &decoded); err == nil {
			return fmt.Sprintf("event_value=%s", formatCBOR(decoded, len(value)))
		}
		return fmt.Sprintf("event_value=binary(%d bytes)", len(value))
	}

	// Try CBOR decode for regular state keys
	var decoded interface{}
	if err := cbor.Unmarshal(value, &decoded); err == nil {
		return formatCBOR(decoded, len(value))
	}

	return fmt.Sprintf("binary(%d bytes)", len(value))
}
