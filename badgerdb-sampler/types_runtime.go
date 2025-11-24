package main

import "github.com/fxamacker/cbor/v2"

// Runtime decode output types - structured representations of decoded runtime database entries.
// These types separate decoding logic from string formatting, enabling flexible output formats.

// =============================================================================
// CBOR Deserialization Types (for unmarshaling from database)
// =============================================================================

// RuntimeHistoryMetadata represents the metadata stored in runtime history DB.
// See: _oasis-core/go/runtime/history/db.go:34-44 (dbMetadata)
type RuntimeHistoryMetadata struct {
	RuntimeID           []byte `cbor:"runtime_id"`
	Version             uint64 `cbor:"version"`
	LastConsensusHeight int64  `cbor:"last_consensus_height"`
	LastRound           uint64 `cbor:"last_round"`
}

// RuntimeHistoryAnnotatedBlock represents an annotated block in runtime history.
// See: _oasis-core/go/roothash/api/api.go:401-409 (AnnotatedBlock)
type RuntimeHistoryAnnotatedBlock struct {
	Height int64                `cbor:"consensus_height"`
	Block  *RuntimeHistoryBlock `cbor:"block"`
}

// RuntimeHistoryBlock represents a runtime block.
// See: _oasis-core/go/roothash/api/block/block.go:7-12 (Block)
type RuntimeHistoryBlock struct {
	Header RuntimeHistoryBlockHeader `cbor:"header"`
}

// RuntimeHistoryBlockHeader represents a runtime block header.
// See: _oasis-core/go/roothash/api/block/header.go:69-99 (Header)
type RuntimeHistoryBlockHeader struct {
	Version        uint16 `cbor:"version"`
	Namespace      []byte `cbor:"namespace"`
	Round          uint64 `cbor:"round"`
	Timestamp      uint64 `cbor:"timestamp"` // POSIX time (Unix seconds)
	HeaderType     uint8  `cbor:"header_type"`
	PreviousHash   []byte `cbor:"previous_hash"`
	IORoot         []byte `cbor:"io_root"`
	StateRoot      []byte `cbor:"state_root"`
	MessagesHash   []byte `cbor:"messages_hash"`
	InMessagesHash []byte `cbor:"in_msgs_hash"`
}

// RuntimeHistoryRoundResults represents round results in runtime history.
// See: _oasis-core/go/roothash/api/results.go:5-17 (RoundResults)
type RuntimeHistoryRoundResults struct {
	Messages            []RuntimeHistoryMessageEvent `cbor:"messages,omitempty"`
	GoodComputeEntities [][]byte                     `cbor:"good_compute_entities,omitempty"`
	BadComputeEntities  [][]byte                     `cbor:"bad_compute_entities,omitempty"`
}

// RuntimeHistoryMessageEvent represents a message event.
// See: _oasis-core/go/roothash/api/api.go:492-499 (MessageEvent)
type RuntimeHistoryMessageEvent struct {
	Module string          `cbor:"module,omitempty"`
	Code   uint32          `cbor:"code,omitempty"`
	Index  uint32          `cbor:"index,omitempty"`
	Result cbor.RawMessage `cbor:"result,omitempty"`
}

// RuntimeInputArtifacts represents input transaction artifacts stored in IO tree.
// See: _oasis-core/go/runtime/transaction/transaction.go:129-140 (inputArtifacts)
type RuntimeInputArtifacts struct {
	_          struct{} `cbor:",toarray"`
	Input      []byte
	BatchOrder uint32
}

// RuntimeOutputArtifacts represents output transaction artifacts stored in IO tree.
// See: _oasis-core/go/runtime/transaction/transaction.go:145-150 (outputArtifacts)
type RuntimeOutputArtifacts struct {
	_      struct{} `cbor:",toarray"`
	Output []byte
}

// =============================================================================
// Decoded Output Types (structured representations for JSON output)
// =============================================================================

// RuntimeMkvsKeyInfo represents a decoded runtime-mkvs key.
// See: _oasis-core/go/storage/mkvs/db/badger/badger.go:31-66
type RuntimeMkvsKeyInfo struct {
	KeyType     string `json:"key_type"`             // "node", "write_log", "roots_metadata", "root_updated_nodes", "metadata", "unknown"
	Height      uint64 `json:"height,omitempty"`     // For write_log, roots_metadata, root_updated_nodes
	Hash        string `json:"hash,omitempty"`       // For node (hex, truncated)
	DbPrefix    byte   `json:"db_prefix,omitempty"`  // 0x01 or 0x05 if present
	DecodeError string `json:"decode_error,omitempty"`
}

// RuntimeMkvsNodeInfo represents a decoded runtime-mkvs value (node).
// See: _oasis-core/go/storage/mkvs/node/node.go:26-32 (prefixes), 294-309 (InternalNode), 531-537 (LeafNode)
type RuntimeMkvsNodeInfo struct {
	NodeType    string                   `json:"node_type"` // "leaf", "internal", "nil", "non_node", "unknown"
	Size        int                      `json:"size"`
	Leaf        *RuntimeMkvsLeafInfo     `json:"leaf,omitempty"`
	Internal    *RuntimeMkvsInternalInfo `json:"internal,omitempty"`
	DecodeError string                   `json:"decode_error,omitempty"`
}

// RuntimeMkvsLeafInfo represents a decoded MKVS LeafNode.
// See: _oasis-core/go/storage/mkvs/node/node.go:531-537
type RuntimeMkvsLeafInfo struct {
	Module       string                  `json:"module"`
	KeyLen       int                     `json:"key_len"`
	Key          string                  `json:"key,omitempty"` // hex, truncated
	ValueLen     int                     `json:"value_len"`
	DecodedValue *RuntimeLeafValueInfo   `json:"decoded_value,omitempty"`
}

// RuntimeMkvsInternalInfo represents a decoded MKVS InternalNode.
// See: _oasis-core/go/storage/mkvs/node/node.go:294-309
type RuntimeMkvsInternalInfo struct {
	LabelBits uint16 `json:"label_bits"`
	HasLeaf   bool   `json:"has_leaf"`
	LeftHash  string `json:"left_hash,omitempty"`  // hex, truncated
	RightHash string `json:"right_hash,omitempty"` // hex, truncated
}

// RuntimeHistoryKeyInfo represents a decoded runtime-history key.
// See: _oasis-core/go/runtime/history/db.go:19-31
type RuntimeHistoryKeyInfo struct {
	KeyType     string `json:"key_type"`             // "metadata", "block", "round_results", "unknown"
	Height      uint64 `json:"height,omitempty"`     // For block, round_results (block height/round)
	ExtraData   string `json:"extra,omitempty"`      // Unexpected extra bytes (hex)
	DecodeError string `json:"decode_error,omitempty"`
}

// RuntimeHistoryValueInfo represents a decoded runtime-history value.
type RuntimeHistoryValueInfo struct {
	KeyType      string                         `json:"key_type"`
	Size         int                            `json:"size"`
	Metadata     *RuntimeHistoryMetadataInfo    `json:"metadata,omitempty"`
	Block        *RuntimeHistoryBlockInfo       `json:"block,omitempty"`
	RoundResults *RuntimeHistoryRoundResultsInfo `json:"round_results,omitempty"`
	DecodeError  string                         `json:"decode_error,omitempty"`
}

// RuntimeHistoryMetadataInfo represents decoded runtime history metadata.
// See: _oasis-core/go/runtime/history/db.go:34-44
type RuntimeHistoryMetadataInfo struct {
	Version             uint64 `json:"version"`
	RuntimeID           string `json:"runtime_id"` // hex, truncated
	LastRound           uint64 `json:"last_round"`
	LastConsensusHeight int64  `json:"last_consensus_height"`
}

// RuntimeHistoryBlockInfo represents a decoded runtime history block.
// See: _oasis-core/go/roothash/api/api.go:402-409 (AnnotatedBlock)
// See: _oasis-core/go/roothash/api/block/header.go:69-99 (Header)
type RuntimeHistoryBlockInfo struct {
	ConsensusHeight int64  `json:"consensus_height"`
	Round           uint64 `json:"round"`
	Timestamp       string `json:"timestamp"`  // RFC3339 format
	HeaderType      string `json:"header_type"`
	StateRoot       string `json:"state_root,omitempty"` // hex, truncated
	BlockNil        bool   `json:"block_nil,omitempty"`  // true if block was nil
}

// RuntimeHistoryRoundResultsInfo represents decoded runtime history round results.
// See: _oasis-core/go/roothash/api/results.go:5-17
type RuntimeHistoryRoundResultsInfo struct {
	MessageCount        int `json:"message_count"`
	GoodComputeEntities int `json:"good_compute_entities"`
	BadComputeEntities  int `json:"bad_compute_entities"`
}

// RuntimeLeafValueInfo represents a decoded MKVS leaf node value.
// See: _oasis-core/go/runtime/transaction/transaction.go:129-150 (artifacts)
type RuntimeLeafValueInfo struct {
	ValueType    string      `json:"value_type"` // "io_input", "io_output", "io_event", "evm_code", "evm_storage", "evm_block_hash", "cbor", "binary"
	InputSize    int         `json:"input_size,omitempty"`
	BatchOrder   uint32      `json:"batch_order,omitempty"`
	OutputSize   int         `json:"output_size,omitempty"`
	DecodedValue interface{} `json:"decoded_value,omitempty"` // For CBOR/event values
	BinarySize   int         `json:"binary_size,omitempty"`   // For binary values
	EVM          *EVMDataInfo `json:"evm,omitempty"`           // For EVM-specific data
}

// EVMDataInfo represents decoded EVM storage data.
// See: _oasis-sdk/runtime-sdk/modules/evm/src/state.rs
type EVMDataInfo struct {
	StorageType string `json:"storage_type"` // "code", "storage", "block_hash", "confidential_storage"
	Address     string `json:"address,omitempty"` // H160 (20 bytes) - contract address (hex)
	StorageSlot string `json:"storage_slot,omitempty"` // H256 (32 bytes) - storage slot (hex)
	StorageValue string `json:"storage_value,omitempty"` // H256 (32 bytes) - storage value (hex)
	Round       uint64 `json:"round,omitempty"` // For block_hash type
	BlockHash   string `json:"block_hash,omitempty"` // H256 (32 bytes) - block hash (hex)
	CodeSize    int    `json:"code_size,omitempty"` // Size of contract bytecode
	CodePreview string `json:"code_preview,omitempty"` // First few bytes of bytecode (hex)
}
