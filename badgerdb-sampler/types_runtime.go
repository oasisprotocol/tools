package main

import "github.com/fxamacker/cbor/v2"

// =============================================================================
// Decoded Output Types (structured representations for JSON output)
// =============================================================================

// RuntimeMkvsKeyInfo represents a decoded runtime-mkvs key.
// See: _oasis-core/go/storage/mkvs/db/badger/badger.go:31-66
type RuntimeMkvsKeyInfo struct {
	KeyType       string `json:"key_type"`                 // "node", "write_log", "roots_metadata", "root_updated_nodes", "metadata", "unknown"
	RuntimeHeight uint64 `json:"runtime_height,omitempty"` // For write_log, roots_metadata, root_updated_nodes
	Hash          string `json:"hash,omitempty"`           // For node (hex, truncated)
}

// RuntimeMkvsNodeInfo represents a decoded runtime-mkvs value (node).
// See: _oasis-core/go/storage/mkvs/node/node.go:26-32 (prefixes), 294-309 (InternalNode), 531-537 (LeafNode)
type RuntimeMkvsNodeInfo struct {
	NodeType      string                   `json:"node_type"` // "leaf", "internal", "nil", "non_node", "unknown"
	NodeSize      int                      `json:"node_size"` // Total MKVS node size (renamed from value_size for clarity)
	NodeHex       string                   `json:"node_hex,omitempty"` // Raw node dump (hex, truncated)
	Leaf          *RuntimeMkvsLeafInfo     `json:"leaf,omitempty"`
	LeafError     string                   `json:"leaf_error,omitempty"` // Error decoding leaf node
	Internal      *RuntimeMkvsInternalInfo `json:"internal,omitempty"`
	InternalError string                   `json:"internal_error,omitempty"` // Error decoding internal node
	NodeError     string                   `json:"node_error,omitempty"` // Structural parsing error
}

// RuntimeMkvsLeafInfo represents a decoded MKVS LeafNode.
// See: _oasis-core/go/storage/mkvs/node/node.go:531-537
type RuntimeMkvsLeafInfo struct {
	Module     string                `json:"module"`
	KeySize    int                   `json:"key_size"`
	KeyHex     string                `json:"key_hex,omitempty"` // hex, truncated
	ValueSize  int                   `json:"value_size"`
	ValueHex   string                `json:"value_hex,omitempty"` // hex, truncated
	Value      *RuntimeLeafValueInfo `json:"value,omitempty"` // Decoded value (renamed from ValueDecoded)
	ValueError string                `json:"value_error,omitempty"` // Error decoding value
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
	KeyType       string `json:"key_type"`                  // "metadata", "block", "round_results", "unknown"
	RuntimeHeight uint64 `json:"runtime_height,omitempty"`  // For block, round_results (runtime block height)
	ExtraData     string `json:"extra,omitempty"`           // Unexpected extra bytes (hex)
}

// RuntimeHistoryValueInfo represents a decoded runtime-history value.
type RuntimeHistoryValueInfo struct {
	Metadata          *RuntimeHistoryMetadataInfo     `json:"metadata,omitempty"`
	MetadataError     string                          `json:"metadata_error,omitempty"` // Error decoding metadata
	Block             *RuntimeHistoryBlockInfo        `json:"block,omitempty"`
	BlockError        string                          `json:"block_error,omitempty"` // Error decoding block
	RoundResults      *RuntimeHistoryRoundResultsInfo `json:"round_results,omitempty"`
	RoundResultsError string                          `json:"round_results_error,omitempty"` // Error decoding round results
}

// RuntimeHistoryMetadataInfo represents decoded runtime history metadata.
// See: _oasis-core/go/runtime/history/db.go:34-44
type RuntimeHistoryMetadataInfo struct {
	Version              uint64 `json:"version"`
	RuntimeID            string `json:"runtime_id"`             // hex, truncated
	LastRuntimeHeight    uint64 `json:"last_runtime_height"`    // last runtime block height
	LastConsensusHeight  int64  `json:"last_consensus_height"`  // last consensus block height
}

// RuntimeHistoryBlockInfo represents a decoded runtime history block.
// See: _oasis-core/go/roothash/api/api.go:402-409 (AnnotatedBlock)
// See: _oasis-core/go/roothash/api/block/header.go:69-99 (Header)
type RuntimeHistoryBlockInfo struct {
	ConsensusHeight int64  `json:"consensus_height"`         // consensus block height
	RuntimeHeight   uint64 `json:"runtime_height"`           // runtime block height (formerly "round")
	Timestamp       string `json:"timestamp"`                // RFC3339 format
	HeaderType      string `json:"header_type"`
	StateRoot       string `json:"state_root,omitempty"`     // hex, truncated
	BlockNil        bool   `json:"block_nil,omitempty"`      // true if block was nil
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
// Note: ValueHex and ValueSize are stored in parent RuntimeMkvsLeafInfo to avoid duplication
type RuntimeLeafValueInfo struct {
	ValueType        string           `json:"value_type,omitempty"`          // Computed classification
	CBOR             interface{}      `json:"cbor,omitempty"`                // For io_event and cbor types - decoded CBOR data
	CBORError        string           `json:"cbor_error,omitempty"`          // Error decoding CBOR
	EVM              *EVMDataInfo     `json:"evm,omitempty"`                 // For EVM-specific storage data
	EVMError         string           `json:"evm_error,omitempty"`           // Error decoding EVM storage
	EVMEvent         *EVMEventInfo    `json:"evm_event,omitempty"`           // For EVM event data
	EVMEventError    string           `json:"evm_event_error,omitempty"`     // Error decoding EVM event
	EVMTxInput       *EVMTxInputInfo  `json:"evm_tx_input,omitempty"`        // For EVM transaction input artifacts
	EVMTxInputError  string           `json:"evm_tx_input_error,omitempty"`  // Error decoding EVM tx input
	EVMTxOutput      *EVMTxOutputInfo `json:"evm_tx_output,omitempty"`       // For EVM transaction output artifacts
	EVMTxOutputError string           `json:"evm_tx_output_error,omitempty"` // Error decoding EVM tx output
}

// =============================================================================
// Runtime CBOR Deserialization Types
// =============================================================================

// cborRuntimeHistoryMetadata represents the metadata stored in runtime history DB.
// See: _oasis-core/go/runtime/history/db.go:34-44 (dbMetadata)
type cborRuntimeHistoryMetadata struct {
	RuntimeID           []byte `cbor:"runtime_id"`
	Version             uint64 `cbor:"version"`
	LastConsensusHeight int64  `cbor:"last_consensus_height"`
	LastRound           uint64 `cbor:"last_round"`
}

// cborRuntimeHistoryAnnotatedBlock represents an annotated block in runtime history.
// See: _oasis-core/go/roothash/api/api.go:401-409 (AnnotatedBlock)
type cborRuntimeHistoryAnnotatedBlock struct {
	Height int64                `cbor:"consensus_height"`
	Block  *cborRuntimeHistoryBlock `cbor:"block"`
}

// cborRuntimeHistoryBlock represents a runtime block.
// See: _oasis-core/go/roothash/api/block/block.go:7-12 (Block)
type cborRuntimeHistoryBlock struct {
	Header cborRuntimeHistoryBlockHeader `cbor:"header"`
}

// cborRuntimeHistoryBlockHeader represents a runtime block header.
// See: _oasis-core/go/roothash/api/block/header.go:69-99 (Header)
type cborRuntimeHistoryBlockHeader struct {
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

// cborRuntimeHistoryRoundResults represents round results in runtime history.
// See: _oasis-core/go/roothash/api/results.go:5-17 (RoundResults)
type cborRuntimeHistoryRoundResults struct {
	Messages            []cborRuntimeHistoryMessageEvent `cbor:"messages,omitempty"`
	GoodComputeEntities [][]byte                     `cbor:"good_compute_entities,omitempty"`
	BadComputeEntities  [][]byte                     `cbor:"bad_compute_entities,omitempty"`
}

// cborRuntimeHistoryMessageEvent represents a message event.
// See: _oasis-core/go/roothash/api/api.go:492-499 (MessageEvent)
type cborRuntimeHistoryMessageEvent struct {
	Module string          `cbor:"module,omitempty"`
	Code   uint32          `cbor:"code,omitempty"`
	Index  uint32          `cbor:"index,omitempty"`
	Result cbor.RawMessage `cbor:"result,omitempty"`
}

// cborRuntimeInputArtifacts represents input transaction artifacts stored in IO tree.
// See: _oasis-core/go/runtime/transaction/transaction.go:129-140 (inputArtifacts)
type cborRuntimeInputArtifacts struct {
	_          struct{} `cbor:",toarray"`
	Input      []byte
	BatchOrder uint32
}

// cborRuntimeOutputArtifacts represents output transaction artifacts stored in IO tree.
// See: _oasis-core/go/runtime/transaction/transaction.go:145-150 (outputArtifacts)
type cborRuntimeOutputArtifacts struct {
	_      struct{} `cbor:",toarray"`
	Output []byte
}

// cborRuntimeCallArrayFormat represents runtime Call structure encoded as CBOR array (old format).
// See: _oasis-sdk/runtime-sdk/src/types/transaction.rs:124-141 (Call)
// Used in older runtime versions before map-based encoding
type cborRuntimeCallArrayFormat struct {
	_        struct{}        `cbor:",toarray"`
	Format   uint8           // CallFormat: 0=Plain, 1=EncryptedX25519DeoxysII
	Method   string          // Method name
	Body     cbor.RawMessage // Method body (CBOR-encoded)
	ReadOnly bool            // Read-only flag
}

// cborRuntimeCallMapFormat represents runtime Call structure encoded as CBOR map (new format).
// See: _oasis-sdk/runtime-sdk/src/types/transaction.rs:124-141 (Call)
// Used in newer runtime versions
type cborRuntimeCallMapFormat struct {
	Format   uint8           `cbor:"format,omitempty"`
	Method   string          `cbor:"method,omitempty"`
	Body     cbor.RawMessage `cbor:"body"`
	ReadOnly bool            `cbor:"ro,omitempty"`
}
