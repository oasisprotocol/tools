package main

import (
	"math/big"

	"github.com/fxamacker/cbor/v2"
)

// =============================================================================
// Decoded Output Types (structured representations for JSON output)
// =============================================================================

// RuntimeMkvsKeyInfo represents a decoded runtime-mkvs key.
// See: _oasis-core/go/storage/mkvs/db/badger/badger.go:31-66
type RuntimeMkvsKeyInfo struct {
	// Raw fields
	KeyDump  string `json:"key_dump,omitempty"`
	KeySize  int    `json:"key_size"`
	KeyError string `json:"key_error,omitempty"`

	// Decoded fields
	KeyType       string `json:"key_type"`                 // "node", "write_log", "roots_metadata", "root_updated_nodes", "metadata", "unknown"
	RuntimeHeight uint64 `json:"runtime_height,omitempty"` // For write_log, roots_metadata, root_updated_nodes
	Hash          string `json:"hash,omitempty"`           // For node (hex, truncated)
}

// RuntimeMkvsValueInfo represents a decoded runtime-mkvs value (node).
// See: _oasis-core/go/storage/mkvs/node/node.go:26-32 (prefixes), 294-309 (InternalNode), 531-537 (LeafNode)
type RuntimeMkvsValueInfo struct {
	// Raw fields
	RawDump  string `json:"raw_dump,omitempty"`
	RawSize  int    `json:"raw_size"`
	RawError string `json:"raw_error,omitempty"`

	// Node info
	NodeType string                   `json:"node_type"` // "leaf", "internal", "nil", "non_node", "unknown"
	Leaf     *RuntimeMkvsLeafInfo     `json:"leaf,omitempty"`
	Internal *RuntimeMkvsInternalInfo `json:"internal,omitempty"`
}

// RuntimeMkvsLeafInfo represents a decoded MKVS LeafNode.
// See: _oasis-core/go/storage/mkvs/node/node.go:531-537
type RuntimeMkvsLeafInfo struct {
	// Leaf key (extracted from MKVS node)
	KeyDump string `json:"key_dump,omitempty"`
	KeySize int    `json:"key_size"`

	// Decoded key fields
	Module  string `json:"module"`
	KeyType string `json:"key_type,omitempty"`

	// Nested value (has its own raw representation)
	Value *RuntimeLeafValueInfo `json:"value,omitempty"`
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
	// Raw fields
	KeyDump  string `json:"key_dump,omitempty"`
	KeySize  int    `json:"key_size"`
	KeyError string `json:"key_error,omitempty"`

	// Decoded fields
	KeyType       string `json:"key_type"`                 // "metadata", "block", "round_results", "unknown"
	RuntimeHeight uint64 `json:"runtime_height,omitempty"` // For block, round_results (runtime block height)
	ExtraData     string `json:"extra,omitempty"`          // Unexpected extra bytes (hex)
}

// RuntimeHistoryValueInfo represents a decoded runtime-history value.
type RuntimeHistoryValueInfo struct {
	// Raw fields
	RawDump  string `json:"raw_dump,omitempty"`
	RawSize  int    `json:"raw_size"`
	RawError string `json:"raw_error,omitempty"`

	// Decoded content - only ONE populated
	Metadata     *RuntimeHistoryMetadataInfo     `json:"metadata,omitempty"`
	Block        *RuntimeHistoryBlockInfo        `json:"block,omitempty"`
	RoundResults *RuntimeHistoryRoundResultsInfo `json:"round_results,omitempty"`
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
type RuntimeLeafValueInfo struct {
	// Raw leaf value bytes (use Value* prefix for leaf-specific fields)
	ValueDump  string `json:"value_dump,omitempty"`
	ValueSize  int    `json:"value_size"`
	ValueError string `json:"value_error,omitempty"`

	// Classification
	ValueType string `json:"value_type,omitempty"`

	// Decoded content - only ONE populated
	CBOR                          interface{}                         `json:"cbor,omitempty"`
	EVM                           *EVMDataInfo                        `json:"evm,omitempty"`
	EVMEvent                      *EVMEventInfo                       `json:"evm_event,omitempty"`
	EVMTxInput                    *EVMTxInputInfo                     `json:"evm_tx_input,omitempty"`
	EVMTxOutput                   *EVMTxOutputInfo                    `json:"evm_tx_output,omitempty"`
	RuntimeConsensusEvent *RuntimeConsensusEventInfo `json:"runtime_consensus_accounts_event,omitempty"`
}

// RuntimeConsensusEventError represents consensus error.
// See: _oasis-sdk/runtime-sdk/src/modules/consensus_accounts/types.rs:207-215
type RuntimeConsensusEventError struct {
	Module string `json:"module,omitempty"`
	Code   uint32 `json:"code,omitempty"`
}

// RuntimeConsensusEventInfo is decoded consensus_accounts event.
// See: _oasis-sdk/runtime-sdk/src/modules/consensus_accounts/mod.rs:105-157
type RuntimeConsensusEventInfo struct {
	EventType     string                               `json:"event_type"`
	From          string                               `json:"from,omitempty"`
	To            string                               `json:"to,omitempty"`
	Nonce         uint64                               `json:"nonce,omitempty"`
	Amount        string                               `json:"amount,omitempty"`
	Shares        string                               `json:"shares,omitempty"`
	DebondEndTime uint64                               `json:"debond_end_time,omitempty"`
	Error         *RuntimeConsensusEventError `json:"error,omitempty"`
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

// =============================================================================
// Runtime Event CBOR Deserialization Types
// =============================================================================

// RuntimeBaseUnits represents token::BaseUnits encoded as CBOR array [amount_bytes, denomination_bytes].
// See: _oasis-sdk/runtime-sdk/src/types/token.rs:90
//   pub struct BaseUnits(pub u128, pub Denomination);
type RuntimeBaseUnits struct {
	_            struct{} `cbor:",toarray"`
	Amount       []byte   // u128 as big-endian bytes
	Denomination []byte   // Denomination as bytes (empty for native token)
}

func (b *RuntimeBaseUnits) String() string {
	if len(b.Amount) == 0 {
		return "0"
	}
	return new(big.Int).SetBytes(b.Amount).String()
}

// RuntimeAddress represents a 21-byte runtime address with bech32 encoding.
// See: _oasis-sdk/runtime-sdk/src/types/address.rs:79-80
//   pub struct Address([u8; ADDRESS_SIZE]); // ADDRESS_SIZE = 21
// CBOR: Encodes as ByteString
// Display: Bech32 with HRP "oasis"
type RuntimeAddress [21]byte

func (a RuntimeAddress) String() string {
	return bech32Encode("oasis", a[:])
}

// cborRuntimeConsensusError represents error details from consensus layer.
// See: _oasis-sdk/runtime-sdk/src/modules/consensus_accounts/types.rs:207-215
type cborRuntimeConsensusError struct {
	Module string `cbor:"module,omitempty"`
	Code   uint32 `cbor:"code,omitempty"`
}

// Deposit/Withdraw/Delegate events (codes 1-3).
// See: _oasis-sdk/runtime-sdk/src/modules/consensus_accounts/mod.rs:109-137
type cborRuntimeConsensusTransferEvent struct {
	_      struct{} `cbor:",toarray"`
	From   RuntimeAddress
	Nonce  uint64
	To     RuntimeAddress
	Amount RuntimeBaseUnits
	Error  *cborRuntimeConsensusError `cbor:"error,omitempty"`
}

// UndelegateStart event (code 4).
// See: _oasis-sdk/runtime-sdk/src/modules/consensus_accounts/mod.rs:139-148
type cborRuntimeConsensusUndelegateStartEvent struct {
	_             struct{} `cbor:",toarray"`
	From          RuntimeAddress
	Nonce         uint64
	To            RuntimeAddress
	Shares        []byte
	DebondEndTime uint64
	Error         *cborRuntimeConsensusError `cbor:"error,omitempty"`
}

// UndelegateDone event (code 5).
// See: _oasis-sdk/runtime-sdk/src/modules/consensus_accounts/mod.rs:150-156
type cborRuntimeConsensusUndelegateDoneEvent struct {
	_      struct{} `cbor:",toarray"`
	From   RuntimeAddress
	To     RuntimeAddress
	Shares []byte
	Amount RuntimeBaseUnits
}
