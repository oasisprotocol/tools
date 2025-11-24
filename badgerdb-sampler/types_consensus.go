package main

// Consensus decode output types - structured representations of decoded consensus database entries.
// These types separate decoding logic from string formatting, enabling flexible output formats.

// =============================================================================
// Blockstore Types
// =============================================================================

// ConsensusBlockstoreKeyInfo represents a decoded consensus-blockstore key.
// See: tendermint/store/store.go for key formats (H:, P:, C:, SC:, BH:)
type ConsensusBlockstoreKeyInfo struct {
	KeyType   string `json:"key_type"`            // "blockstore_state", "block_meta", "block_part", "block_commit", "seen_commit", "block_hash", "unknown"
	Height    int64  `json:"height,omitempty"`    // For H:, C:, SC:, P: keys
	PartIndex int    `json:"part_index,omitempty"` // For P: keys
	Hash      string `json:"hash,omitempty"`      // For BH: keys
	RawKey    string `json:"raw_key,omitempty"`   // Original ASCII key
}

// ConsensusBlockstoreValueInfo represents a decoded consensus-blockstore value.
// See: tendermint/proto/tendermint/store/types.proto (BlockStoreState)
// See: tendermint/proto/tendermint/types/types.proto (BlockMeta, Part, Commit)
type ConsensusBlockstoreValueInfo struct {
	KeyType     string                     `json:"key_type"`
	Size        int                        `json:"size"`
	Timestamp   int64                      `json:"timestamp,omitempty"` // Unix timestamp from block meta
	State       *ConsensusBlockStoreState  `json:"state,omitempty"`
	BlockMeta   *ConsensusBlockMetaInfo    `json:"block_meta,omitempty"`
	Part        *ConsensusPartInfo         `json:"part,omitempty"`
	Commit      *ConsensusCommitInfo       `json:"commit,omitempty"`
	HashHeight  int64                      `json:"hash_height,omitempty"` // For block_hash type
	DecodeError string                     `json:"decode_error,omitempty"`
}

// ConsensusBlockStoreState represents BlockStoreState from tendermint.
// See: tendermint/proto/tendermint/store/types.proto (BlockStoreState message)
type ConsensusBlockStoreState struct {
	Base   int64 `json:"base"`
	Height int64 `json:"height"`
}

// ConsensusBlockMetaInfo represents decoded block metadata.
// See: tendermint/proto/tendermint/types/types.proto (BlockMeta, Header messages)
type ConsensusBlockMetaInfo struct {
	Height  int64  `json:"height"`
	Time    string `json:"time"`     // RFC3339 format
	ChainID string `json:"chain_id"`
	NumTxs  int64  `json:"num_txs"`
	AppHash string `json:"app_hash,omitempty"` // hex, truncated
}

// ConsensusPartInfo represents decoded block part.
// See: tendermint/proto/tendermint/types/types.proto (Part message)
type ConsensusPartInfo struct {
	Index      uint32 `json:"index"`
	BytesSize  int    `json:"bytes_size"`
	ProofTotal int64  `json:"proof_total"`
}

// ConsensusCommitInfo represents decoded commit.
// See: tendermint/proto/tendermint/types/types.proto (Commit message)
type ConsensusCommitInfo struct {
	Height     int64 `json:"height"`
	Round      int32 `json:"round"`
	Signatures int   `json:"signatures"`
}

// =============================================================================
// Evidence Types
// =============================================================================

// ConsensusEvidenceKeyInfo represents a decoded consensus-evidence key.
// See: tendermint/store/evidence/pool.go for key formats
type ConsensusEvidenceKeyInfo struct {
	KeyType    string `json:"key_type"`
	PrefixByte byte   `json:"prefix_byte,omitempty"`
	RawKey     string `json:"raw_key,omitempty"` // hex
}

// ConsensusEvidenceValueInfo represents a decoded consensus-evidence value.
// See: tendermint/proto/tendermint/types/evidence.proto (DuplicateVoteEvidence)
type ConsensusEvidenceValueInfo struct {
	Size        int    `json:"size"`
	VoteAHeight int64  `json:"vote_a_height,omitempty"`
	VoteBHeight int64  `json:"vote_b_height,omitempty"`
	DecodeError string `json:"decode_error,omitempty"`
}

// =============================================================================
// MKVS Types
// =============================================================================

// ConsensusMkvsKeyInfo represents a decoded consensus-mkvs key.
// See: _oasis-core/go/storage/mkvs/db/badger/badger.go:31-66
type ConsensusMkvsKeyInfo struct {
	KeyType     string `json:"key_type"`             // "node", "write_log", "roots_metadata", "root_updated_nodes", "metadata", "multipart_restore_log", "root_node", "unknown"
	Height      uint64 `json:"height,omitempty"`     // For write_log, roots_metadata, root_updated_nodes
	Hash        string `json:"hash,omitempty"`       // hex, truncated
	RootType    byte   `json:"root_type,omitempty"`
	DbPrefix    byte   `json:"db_prefix,omitempty"`  // 0x01 or 0x05 if present
	DecodeError string `json:"decode_error,omitempty"`
}

// ConsensusMkvsNodeInfo represents a decoded consensus-mkvs value (node).
// See: _oasis-core/go/storage/mkvs/node/node.go:26-32 (prefixes), 294-309 (InternalNode), 531-537 (LeafNode)
type ConsensusMkvsNodeInfo struct {
	NodeType    string                     `json:"node_type"` // "leaf", "internal", "nil", "non_node", "unknown"
	Size        int                        `json:"size"`
	Leaf        *ConsensusMkvsLeafInfo     `json:"leaf,omitempty"`
	Internal    *ConsensusMkvsInternalInfo `json:"internal,omitempty"`
	DecodeError string                     `json:"decode_error,omitempty"`
}

// ConsensusMkvsLeafInfo represents a decoded MKVS LeafNode for consensus.
// See: _oasis-core/go/storage/mkvs/node/node.go:531-537 (LeafNode)
// See: _oasis-core/go/consensus/tendermint/apps/*/state/state.go (module prefixes)
type ConsensusMkvsLeafInfo struct {
	Module       string      `json:"module"`
	KeyLen       int         `json:"key_len"`
	Key          string      `json:"key,omitempty"` // hex, truncated
	ValueLen     int         `json:"value_len"`
	DecodedValue interface{} `json:"decoded_value,omitempty"` // CBOR decoded or size info
}

// ConsensusMkvsInternalInfo represents a decoded MKVS InternalNode.
// See: _oasis-core/go/storage/mkvs/node/node.go:294-309 (InternalNode)
type ConsensusMkvsInternalInfo struct {
	LabelBits uint16 `json:"label_bits"`
	HasLeaf   bool   `json:"has_leaf"`
	LeftHash  string `json:"left_hash,omitempty"`  // hex, truncated
	RightHash string `json:"right_hash,omitempty"` // hex, truncated
}

// =============================================================================
// State Types
// =============================================================================

// ConsensusStateKeyInfo represents a decoded consensus-state key.
// See: tendermint/state/store.go for key formats (abciResponsesKey, validatorsKey, etc.)
type ConsensusStateKeyInfo struct {
	KeyType    string `json:"key_type"` // "abci_responses", "consensus_params", "validators", "state", "genesis", "text_key", "binary", "unknown"
	DecodedKey string `json:"decoded_key,omitempty"`
	IsBinary   bool   `json:"is_binary,omitempty"`
}

// ConsensusStateValueInfo represents a decoded consensus-state value.
// See: tendermint/proto/tendermint/state/types.proto (various state types)
// Note: Tendermint state values are complex protobuf - we just report size.
type ConsensusStateValueInfo struct {
	KeyType string `json:"key_type"`
	Size    int    `json:"size"`
}
