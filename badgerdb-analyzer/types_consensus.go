package main

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/fxamacker/cbor/v2"
)

// Consensus decode output types - structured representations of decoded consensus database entries.
// These types separate decoding logic from string formatting, enabling flexible output formats.

// =============================================================================
// Blockstore Types
// =============================================================================

// ConsensusBlockstoreKeyInfo represents a decoded consensus-blockstore key.
// See: tendermint/store/store.go for key formats (H:, P:, C:, SC:, BH:)
type ConsensusBlockstoreKeyInfo struct {
	// Raw fields
	KeyDump  string `json:"key_dump,omitempty"`
	KeySize  int    `json:"key_size"`
	KeyError string `json:"key_error,omitempty"`

	// Decoded fields
	KeyType         string `json:"key_type"`                   // "blockstore_state", "block_meta", "block_part", "block_commit", "seen_commit", "block_hash", "unknown"
	ConsensusHeight int64  `json:"consensus_height,omitempty"` // For H:, C:, SC:, P: keys
	PartIndex       int    `json:"part_index,omitempty"`       // For P: keys
	Hash            string `json:"hash,omitempty"`             // For BH: keys
}

// ConsensusBlockstoreValueInfo represents a decoded consensus-blockstore value.
// See: tendermint/proto/tendermint/store/types.proto (BlockStoreState)
// See: tendermint/proto/tendermint/types/types.proto (BlockMeta, Part, Commit)
type ConsensusBlockstoreValueInfo struct {
	// Raw fields
	RawDump  string `json:"raw_dump,omitempty"`
	RawSize  int    `json:"raw_size"`
	RawError string `json:"raw_error,omitempty"`

	// Decoded fields
	Timestamp  int64                     `json:"timestamp,omitempty"` // Unix timestamp from block meta
	State      *ConsensusBlockStoreState `json:"state,omitempty"`
	BlockMeta  *ConsensusBlockMetaInfo   `json:"block_meta,omitempty"`
	Part       *ConsensusPartInfo        `json:"part,omitempty"`
	Commit     *ConsensusCommitInfo      `json:"commit,omitempty"`
	HashHeight int64                     `json:"hash_height,omitempty"` // For block_hash type
}

// ConsensusBlockStoreState represents BlockStoreState from tendermint.
// See: tendermint/proto/tendermint/store/types.proto (BlockStoreState message)
type ConsensusBlockStoreState struct {
	Base            int64 `json:"base"`
	ConsensusHeight int64 `json:"consensus_height"`
}

// ConsensusBlockMetaInfo represents decoded block metadata.
// See: tendermint/proto/tendermint/types/types.proto (BlockMeta, Header messages)
type ConsensusBlockMetaInfo struct {
	ConsensusHeight int64  `json:"consensus_height"`
	Time            string `json:"time"`     // RFC3339 format
	ChainID         string `json:"chain_id"`
	NumTxs          int64  `json:"num_txs"`
	AppHash         string `json:"app_hash,omitempty"` // hex, truncated
}

// ConsensusPartInfo represents decoded block part.
// See: tendermint/proto/tendermint/types/types.proto (Part message)
type ConsensusPartInfo struct {
	Index      uint32 `json:"index"`
	PartSize   int    `json:"part_size"`
	ProofTotal int64  `json:"proof_total"`
}

// ConsensusCommitInfo represents decoded commit.
// See: tendermint/proto/tendermint/types/types.proto (Commit message)
type ConsensusCommitInfo struct {
	ConsensusHeight int64 `json:"consensus_height"`
	Round           int32 `json:"round"`      // Tendermint commit round (not runtime height)
	Signatures      int   `json:"signatures"`
}

// =============================================================================
// Evidence Types
// =============================================================================

// ConsensusEvidenceKeyInfo represents a decoded consensus-evidence key.
// Key format: dbVersion (0x01) + prefix (0x00=committed, 0x01=pending) + "HEIGHT_HEX/HASH_HEX"
// See: cometbft/evidence/pool.go for key formats (keyCommitted, keyPending)
type ConsensusEvidenceKeyInfo struct {
	// Raw fields
	KeyDump  string `json:"key_dump,omitempty"`
	KeySize  int    `json:"key_size"`
	KeyError string `json:"key_error,omitempty"`

	// Decoded fields
	KeyType         string `json:"key_type"`             // "committed", "pending", "type_XX"
	PrefixByte      byte   `json:"prefix_byte,omitempty"`
	ConsensusHeight int64  `json:"consensus_height,omitempty"` // Extracted from key suffix
	Hash            string `json:"hash,omitempty"`             // Evidence hash from key suffix
}

// ConsensusEvidenceValueInfo represents a decoded consensus-evidence value.
// Committed evidence stores Int64Value (height only), pending stores full Evidence protobuf.
// See: cometbft/evidence/pool.go (addPendingEvidence stores full evidence, markEvidenceAsCommitted stores height)
// See: tendermint/proto/tendermint/types/evidence.proto (DuplicateVoteEvidence, LightClientAttackEvidence)
type ConsensusEvidenceValueInfo struct {
	// Raw fields
	RawDump       string `json:"raw_dump,omitempty"`
	RawSize       int    `json:"raw_size"`
	RawError      string `json:"raw_error,omitempty"`
	SchemaVersion string `json:"schema_version,omitempty"` // "cb-v0.37", "tm-v0.34", "unknown"

	// Decoded fields
	EvidenceType     string `json:"evidence_type,omitempty"`      // "committed_marker", "duplicate_vote", "light_client_attack", "unknown"
	CommittedHeight  int64  `json:"committed_height,omitempty"`   // For committed evidence (Int64Value)
	VoteAHeight      int64  `json:"vote_a_height,omitempty"`      // For pending DuplicateVoteEvidence
	VoteBHeight      int64  `json:"vote_b_height,omitempty"`      // For pending DuplicateVoteEvidence
	TotalVotingPower int64  `json:"total_voting_power,omitempty"` // For pending evidence
	ValidatorPower   int64  `json:"validator_power,omitempty"`    // For pending evidence
	Timestamp        string `json:"timestamp,omitempty"`          // RFC3339 format (for pending evidence)
}

// =============================================================================
// MKVS Types
// =============================================================================

// ConsensusMkvsKeyInfo represents a decoded consensus-mkvs key.
// See: _oasis-core/go/storage/mkvs/db/badger/badger.go:31-66
type ConsensusMkvsKeyInfo struct {
	// Raw fields
	KeyDump  string `json:"key_dump,omitempty"`
	KeySize  int    `json:"key_size"`
	KeyError string `json:"key_error,omitempty"`

	// Decoded fields
	KeyType         string `json:"key_type"`                   // "node", "write_log", "roots_metadata", "root_updated_nodes", "metadata", "multipart_restore_log", "root_node", "unknown"
	ConsensusHeight int64  `json:"consensus_height,omitempty"` // For write_log, roots_metadata, root_updated_nodes
	Hash            string `json:"hash,omitempty"`             // hex, truncated (partial key data)
	RootType        string `json:"root_type,omitempty"`
}

// ConsensusMkvsValueInfo represents a decoded consensus-mkvs value (node).
// See: _oasis-core/go/storage/mkvs/node/node.go:26-32 (prefixes), 294-309 (InternalNode), 531-537 (LeafNode)
type ConsensusMkvsValueInfo struct {
	// Raw fields
	RawDump  string `json:"raw_dump,omitempty"`
	RawSize  int    `json:"raw_size"`
	RawError string `json:"raw_error,omitempty"`

	// Node info
	NodeType string                     `json:"node_type"` // "leaf", "internal", "nil", "non_node", "unknown"
	Leaf     *ConsensusMkvsLeafInfo     `json:"leaf,omitempty"`
	Internal *ConsensusMkvsInternalInfo `json:"internal,omitempty"`
}

// ConsensusMkvsLeafInfo represents a decoded MKVS LeafNode for consensus.
// See: _oasis-core/go/storage/mkvs/node/node.go:531-537 (LeafNode)
// See: _oasis-core/go/consensus/tendermint/apps/*/state/state.go (module prefixes)
type ConsensusMkvsLeafInfo struct {
	// Leaf key (extracted from MKVS node)
	KeyDump string `json:"key_dump,omitempty"`
	KeySize int    `json:"key_size"`

	// Decoded key fields
	Module       string `json:"module"`
	KeyType      string `json:"key_type,omitempty"`
	OasisAddress string `json:"oasis_address,omitempty"` // bech32 oasis1... address if applicable

	// Leaf value raw data
	ValueDump  string `json:"value_dump,omitempty"`
	ValueSize  int    `json:"value_size"`
	ValueError string `json:"value_error,omitempty"`

	// Decoded value
	ValueType string      `json:"value_type,omitempty"`
	Value     interface{} `json:"value,omitempty"`
	CBOR      string      `json:"cbor,omitempty"` // CBOR type description or format hint
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
	// Raw fields
	KeyDump  string `json:"key_dump,omitempty"`
	KeySize  int    `json:"key_size"`
	KeyError string `json:"key_error,omitempty"`

	// Decoded fields
	KeyType         string `json:"key_type"`                   // "abci_responses", "consensus_params", "validators", "state", "genesis", "text_key", "binary", "unknown"
	ConsensusHeight int64  `json:"consensus_height,omitempty"` // For abci_responses (extracted from key)
	IsBinary        bool   `json:"is_binary,omitempty"`
}

// ConsensusStateValueInfo represents a decoded consensus-state value.
// See: tendermint/proto/tendermint/state/types.proto (various state types)
type ConsensusStateValueInfo struct {
	// Raw fields
	RawDump       string `json:"raw_dump,omitempty"`
	RawSize       int    `json:"raw_size"`
	RawError      string `json:"raw_error,omitempty"`
	SchemaVersion string `json:"schema_version,omitempty"` // "cometbft", "tendermint-v0.34", "unknown"

	// Decoded content - only ONE populated
	ABCIResponse        *ConsensusABCIResponseInfo `json:"abci_response,omitempty"`        // For abci_responses
	ConsensusParams     *tmConsensusParams         `json:"consensus_params,omitempty"`     // For consensus_params
	ConsensusValidators *tmValidatorSet            `json:"consensus_validators,omitempty"` // For validators
	ConsensusState      *tmState                   `json:"consensus_state,omitempty"`      // For state
}

// ConsensusABCIResponseInfo represents decoded ABCI response summary.
// See: tendermint/proto/tendermint/abci/types.proto (ResponseFinalizeBlock)
type ConsensusABCIResponseInfo struct {
	TxResultCount      int                        `json:"tx_result_count"`
	ValidatorUpdates   int                        `json:"validator_updates"`
	EventCount         int                        `json:"event_count"`
	TransactionSummary *ConsensusTransactionSummary `json:"transaction_summary,omitempty"`
	EventSummary       *ConsensusEventSummary       `json:"event_summary,omitempty"`
}

// ConsensusTransactionSummary summarizes decoded transactions from ABCI responses.
type ConsensusTransactionSummary struct {
	MethodCounts map[string]int            `json:"method_counts"`         // Count by method name
	Transactions []ConsensusTransactionInfo `json:"transactions,omitempty"` // Decoded transaction details
}

// ConsensusTransactionInfo represents a decoded Oasis consensus transaction.
// See: _oasis-core/go/consensus/api/transaction/transaction.go:42-54
type ConsensusTransactionInfo struct {
	Nonce     uint64        `json:"nonce"`
	Method    string        `json:"method"`
	Fee       *ConsensusFee `json:"fee,omitempty"`
	Signer    string        `json:"signer,omitempty"`      // hex-encoded public key
	TxHash    string        `json:"tx_hash,omitempty"`     // hex-encoded transaction hash
	BodyHex   string        `json:"body_hex,omitempty"`    // Truncated hex dump of raw body
	BodySize  int           `json:"body_size,omitempty"`   // Size of body in bytes
	Body      interface{}   `json:"body,omitempty"`        // Decoded body for known types
	BodyError string        `json:"body_error,omitempty"`  // Error decoding body
}

// ConsensusFee represents transaction fee.
// See: _oasis-core/go/consensus/api/transaction/gas.go:30-35
type ConsensusFee struct {
	Amount string `json:"amount"` // string representation of quantity.Quantity
	Gas    uint64 `json:"gas"`
}

// ConsensusEventSummary summarizes decoded events from ABCI responses.
type ConsensusEventSummary struct {
	EventTypeCounts map[string]int    `json:"event_type_counts"`  // Count by event type
	Events          []ConsensusEventInfo `json:"events,omitempty"` // Sample of decoded events
}

// ConsensusEventInfo represents a decoded consensus event (unified type for all event kinds).
type ConsensusEventInfo struct {
	EventType string      `json:"event_type"`           // "staking.transfer", "staking.burn", etc.
	EventKind string      `json:"event_kind"`           // "transfer", "burn", "add_escrow", "reclaim_escrow"
	BodyHex   string      `json:"body_hex,omitempty"`   // Truncated hex dump of raw body
	BodySize  int         `json:"body_size,omitempty"`  // Size of body in bytes
	Body      interface{} `json:"body,omitempty"`       // Decoded body for known types
	BodyError string      `json:"body_error,omitempty"` // Error decoding body
}


// =============================================================================
// Consensus CBOR Deserialization Types
// =============================================================================

// QuantityBytes wraps big.Int and implements encoding.BinaryUnmarshaler to automatically
// decode CBOR byte strings (big-endian) into arbitrary-precision unsigned integers.
// See: _oasis-core/go/common/quantity/quantity.go:28-50 (MarshalBinary/UnmarshalBinary)
type QuantityBytes big.Int

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
// Decodes a byte slice (big-endian) into a big.Int.
func (q *QuantityBytes) UnmarshalBinary(data []byte) error {
	if q == nil {
		return nil
	}
	(*big.Int)(q).SetBytes(data)
	return nil
}

// String returns the decimal string representation of the quantity.
func (q *QuantityBytes) String() string {
	if q == nil {
		return "0"
	}
	return (*big.Int)(q).String()
}

// OasisAddress represents a 21-byte Oasis address with automatic bech32 encoding.
// See: _oasis-core/go/common/crypto/address/address.go (ADDRESS_SIZE = 21)
type OasisAddress [21]byte

// MarshalJSON implements json.Marshaler to automatically encode as bech32 string.
func (a OasisAddress) MarshalJSON() ([]byte, error) {
	return json.Marshal(bech32Encode("oasis", a[:]))
}

// String returns the bech32-encoded address (e.g., "oasis1...").
func (a OasisAddress) String() string {
	return bech32Encode("oasis", a[:])
}

// cborConsensusSignedTransaction represents the signature envelope.
// See: _oasis-core/go/common/crypto/signature/signature.go:415-421
type cborConsensusSignedTransaction struct {
	Blob      []byte                   `json:"untrusted_raw_value"`
	Signature cborConsensusSignature   `json:"signature"`
}

// cborConsensusInnerTransaction represents an unsigned consensus transaction.
// See: _oasis-core/go/consensus/api/transaction/transaction.go:43-54
type cborConsensusInnerTransaction struct {
	Nonce  uint64             `json:"nonce"`
	Fee    *cborConsensusFee  `json:"fee"`
	Method string             `json:"method"`
	Body   cbor.RawMessage    `json:"body"`
}

// cborConsensusFee represents transaction fee.
// See: _oasis-core/go/consensus/api/transaction/gas.go:30-35
type cborConsensusFee struct {
	Amount string `json:"amount"` // string representation of quantity.Quantity
	Gas    uint64 `json:"gas"`
}

// cborConsensusSignature represents a signature with public key.
// See: _oasis-core/go/common/crypto/signature/signature.go:313-318
type cborConsensusSignature struct {
	PublicKey [32]byte `json:"public_key"` // ED25519 public key
	Signature [64]byte `json:"signature"`  // ED25519 signature
}

// cborConsensusTransfer represents a staking transfer transaction body.
// See: _oasis-core/go/staking/api/api.go:342-345
type cborConsensusTransfer struct {
	To     OasisAddress   `json:"to"`     // Oasis address (bech32-encoded in JSON)
	Amount *QuantityBytes `json:"amount"` // QuantityBytes automatically unmarshals from CBOR byte string
}

// cborConsensusBurn represents a staking burn transaction body.
// See: _oasis-core/go/staking/api/api.go:368-370
type cborConsensusBurn struct {
	Amount *QuantityBytes `json:"amount"` // QuantityBytes automatically unmarshals from CBOR byte string
}

// cborConsensusAddEscrow represents an add escrow transaction body.
// See: _oasis-core/go/staking/api/api.go:403-406
type cborConsensusAddEscrow struct {
	Account OasisAddress   `json:"account"` // Oasis address (bech32-encoded in JSON)
	Amount  *QuantityBytes `json:"amount"`  // QuantityBytes automatically unmarshals from CBOR byte string
}

// cborConsensusReclaimEscrow represents a reclaim escrow transaction body.
// See: _oasis-core/go/staking/api/api.go:444-447
type cborConsensusReclaimEscrow struct {
	Account OasisAddress   `json:"account"` // Oasis address (bech32-encoded in JSON)
	Shares  *QuantityBytes `json:"shares"`  // QuantityBytes automatically unmarshals from CBOR byte string
}

// cborConsensusRegisterEntity represents a registry entity registration transaction body.
// See: _oasis-core/go/registry/api/api.go
type cborConsensusRegisterEntity struct {
	Signature  []byte `json:"signature,omitempty"`
	Descriptor []byte `json:"descriptor,omitempty"` // CBOR-encoded entity descriptor
}

// cborConsensusRegisterNode represents a registry node registration transaction body.
// See: _oasis-core/go/registry/api/api.go
type cborConsensusRegisterNode struct {
	Signature  []byte `json:"signature,omitempty"`
	Descriptor []byte `json:"descriptor,omitempty"` // CBOR-encoded node descriptor
}

// cborConsensusExecutorCommit represents a roothash executor commit transaction body.
// See: _oasis-core/go/roothash/api/commitment/executor.go
type cborConsensusExecutorCommit struct {
	ID      []byte `json:"runtime_id,omitempty"`  // Runtime ID
	Commits []byte `json:"commits,omitempty"`     // CBOR-encoded commits
}

// cborConsensusSubmitProposal represents a governance proposal submission transaction body.
// See: _oasis-core/go/governance/api/api.go
type cborConsensusSubmitProposal struct {
	Content []byte         `json:"content,omitempty"` // CBOR-encoded proposal content
	Deposit *QuantityBytes `json:"deposit,omitempty"` // QuantityBytes automatically unmarshals from CBOR byte string
}

// cborConsensusCastVote represents a governance vote transaction body.
// See: _oasis-core/go/governance/api/api.go
type cborConsensusCastVote struct {
	ProposalID uint64 `json:"proposal_id"`
	Vote       uint8  `json:"vote"` // 0=invalid, 1=yes, 2=no, 3=abstain
}

// cborConsensusTransferEvent represents a staking transfer event.
// See: _oasis-core/go/staking/api/api.go:223-229
type cborConsensusTransferEvent struct {
	From   OasisAddress   `json:"from"`   // Oasis address (bech32-encoded in JSON)
	To     OasisAddress   `json:"to"`     // Oasis address (bech32-encoded in JSON)
	Amount *QuantityBytes `json:"amount"` // QuantityBytes automatically unmarshals from CBOR byte string
}

// cborConsensusBurnEvent represents a staking burn event.
// See: _oasis-core/go/staking/api/api.go:236-240
type cborConsensusBurnEvent struct {
	Owner  OasisAddress   `json:"owner"`  // Oasis address (bech32-encoded in JSON)
	Amount *QuantityBytes `json:"amount"` // QuantityBytes automatically unmarshals from CBOR byte string
}

// cborConsensusAddEscrowEvent represents an add escrow event.
// See: _oasis-core/go/staking/api/api.go:266-273
type cborConsensusAddEscrowEvent struct {
	Owner     OasisAddress   `json:"owner"`      // Oasis address (bech32-encoded in JSON)
	Escrow    OasisAddress   `json:"escrow"`     // Oasis address (bech32-encoded in JSON)
	Amount    *QuantityBytes `json:"amount"`     // QuantityBytes automatically unmarshals from CBOR byte string
	NewShares *QuantityBytes `json:"new_shares"` // QuantityBytes automatically unmarshals from CBOR byte string
}

// cborConsensusReclaimEscrowEvent represents a reclaim escrow event.
// See: _oasis-core/go/staking/api/api.go:313-320
type cborConsensusReclaimEscrowEvent struct {
	Owner  OasisAddress   `json:"owner"`  // Oasis address (bech32-encoded in JSON)
	Escrow OasisAddress   `json:"escrow"` // Oasis address (bech32-encoded in JSON)
	Amount *QuantityBytes `json:"amount"` // QuantityBytes automatically unmarshals from CBOR byte string
	Shares *QuantityBytes `json:"shares"` // QuantityBytes automatically unmarshals from CBOR byte string
}


// =============================================================================
// Tendermint Protobuf Deserialization Types
// =============================================================================
// Source: github.com/tendermint/tendermint v0.34.21
// These types are used only for protobuf deserialization - no tendermint business logic is needed.
// For proto.Unmarshal() to work types need to implement proto.Message interface.

// tmBlockStoreState represents BlockStoreState from tendermint/proto/tendermint/store/types.proto
type tmBlockStoreState struct {
	Base   int64 `protobuf:"varint,1,opt,name=base,proto3"`
	Height int64 `protobuf:"varint,2,opt,name=height,proto3"`
}
func (m *tmBlockStoreState) Reset()         { *m = tmBlockStoreState{} }
func (m *tmBlockStoreState) String() string { return "" }
func (*tmBlockStoreState) ProtoMessage()    {}

// tmBlockMeta represents BlockMeta from tendermint/proto/tendermint/types/types.proto
type tmBlockMeta struct {
	BlockID   tmBlockID `protobuf:"bytes,1,opt,name=block_id,json=blockId,proto3"`
	BlockSize int64     `protobuf:"varint,2,opt,name=block_size,json=blockSize,proto3"`
	Header    tmHeader  `protobuf:"bytes,3,opt,name=header,proto3"`
	NumTxs    int64     `protobuf:"varint,4,opt,name=num_txs,json=numTxs,proto3"`
}
func (m *tmBlockMeta) Reset()         { *m = tmBlockMeta{} }
func (m *tmBlockMeta) String() string { return "" }
func (*tmBlockMeta) ProtoMessage()    {}

// tmBlockID from tendermint/proto/tendermint/types/types.proto
type tmBlockID struct {
	Hash          []byte          `protobuf:"bytes,1,opt,name=hash,proto3"`
	PartSetHeader tmPartSetHeader `protobuf:"bytes,2,opt,name=part_set_header,json=partSetHeader,proto3"`
}


// tmPartSetHeader from tendermint/proto/tendermint/types/types.proto
type tmPartSetHeader struct {
	Total uint32 `protobuf:"varint,1,opt,name=total,proto3"`
	Hash  []byte `protobuf:"bytes,2,opt,name=hash,proto3"`
}


// tmHeader from tendermint/proto/tendermint/types/types.proto
type tmHeader struct {
	Version            tmConsensus `protobuf:"bytes,1,opt,name=version,proto3"`
	ChainID            string      `protobuf:"bytes,2,opt,name=chain_id,json=chainId,proto3"`
	Height             int64       `protobuf:"varint,3,opt,name=height,proto3"`
	Time               time.Time   `protobuf:"bytes,4,opt,name=time,proto3,stdtime"`
	LastBlockID        tmBlockID   `protobuf:"bytes,5,opt,name=last_block_id,json=lastBlockId,proto3"`
	LastCommitHash     []byte      `protobuf:"bytes,6,opt,name=last_commit_hash,json=lastCommitHash,proto3"`
	DataHash           []byte      `protobuf:"bytes,7,opt,name=data_hash,json=dataHash,proto3"`
	ValidatorsHash     []byte      `protobuf:"bytes,8,opt,name=validators_hash,json=validatorsHash,proto3"`
	NextValidatorsHash []byte      `protobuf:"bytes,9,opt,name=next_validators_hash,json=nextValidatorsHash,proto3"`
	ConsensusHash      []byte      `protobuf:"bytes,10,opt,name=consensus_hash,json=consensusHash,proto3"`
	AppHash            []byte      `protobuf:"bytes,11,opt,name=app_hash,json=appHash,proto3"`
	LastResultsHash    []byte      `protobuf:"bytes,12,opt,name=last_results_hash,json=lastResultsHash,proto3"`
	EvidenceHash       []byte      `protobuf:"bytes,13,opt,name=evidence_hash,json=evidenceHash,proto3"`
	ProposerAddress    []byte      `protobuf:"bytes,14,opt,name=proposer_address,json=proposerAddress,proto3"`
}


// tmConsensus from tendermint/proto/tendermint/types/types.proto
type tmConsensus struct {
	Block uint64 `protobuf:"varint,1,opt,name=block,proto3"`
	App   uint64 `protobuf:"varint,2,opt,name=app,proto3"`
}


// tmPart from tendermint/proto/tendermint/types/types.proto
type tmPart struct {
	Index uint32  `protobuf:"varint,1,opt,name=index,proto3"`
	Bytes []byte  `protobuf:"bytes,2,opt,name=bytes,proto3"`
	Proof tmProof `protobuf:"bytes,3,opt,name=proof,proto3"`
}
func (m *tmPart) Reset()         { *m = tmPart{} }
func (m *tmPart) String() string { return "" }
func (*tmPart) ProtoMessage()    {}


// tmProof from tendermint/proto/tendermint/types/types.proto
type tmProof struct {
	Total    int64    `protobuf:"varint,1,opt,name=total,proto3"`
	Index    int64    `protobuf:"varint,2,opt,name=index,proto3"`
	LeafHash []byte   `protobuf:"bytes,3,opt,name=leaf_hash,json=leafHash,proto3"`
	Aunts    [][]byte `protobuf:"bytes,4,rep,name=aunts,proto3"`
}


// tmCommit from tendermint/proto/tendermint/types/types.proto
type tmCommit struct {
	Height     int64         `protobuf:"varint,1,opt,name=height,proto3"`
	Round      int32         `protobuf:"varint,2,opt,name=round,proto3"`
	BlockID    tmBlockID     `protobuf:"bytes,3,opt,name=block_id,json=blockId,proto3"`
	Signatures []tmCommitSig `protobuf:"bytes,4,rep,name=signatures,proto3"`
}
func (m *tmCommit) Reset()         { *m = tmCommit{} }
func (m *tmCommit) String() string { return "" }
func (*tmCommit) ProtoMessage()    {}


// tmCommitSig from tendermint/proto/tendermint/types/types.proto
type tmCommitSig struct {
	BlockIDFlag      int32     `protobuf:"varint,1,opt,name=block_id_flag,json=blockIdFlag,proto3"`
	ValidatorAddress []byte    `protobuf:"bytes,2,opt,name=validator_address,json=validatorAddress,proto3"`
	Timestamp        time.Time `protobuf:"bytes,3,opt,name=timestamp,proto3,stdtime"`
	Signature        []byte    `protobuf:"bytes,4,opt,name=signature,proto3"`
}


// tmInt64Value from gogoproto/protobuf/wrappers.proto
// Used by Tendermint to store committed evidence (just the height)
type tmInt64Value struct {
	Value int64 `protobuf:"varint,1,opt,name=value,proto3"`
}
func (m *tmInt64Value) Reset()         { *m = tmInt64Value{} }
func (m *tmInt64Value) String() string { return "" }
func (*tmInt64Value) ProtoMessage()    {}


// tmDuplicateVoteEvidence from tendermint/proto/tendermint/types/evidence.proto
type tmDuplicateVoteEvidence struct {
	VoteA            *tmVote   `protobuf:"bytes,1,opt,name=vote_a,json=voteA,proto3"`
	VoteB            *tmVote   `protobuf:"bytes,2,opt,name=vote_b,json=voteB,proto3"`
	TotalVotingPower int64     `protobuf:"varint,3,opt,name=total_voting_power,json=totalVotingPower,proto3"`
	ValidatorPower   int64     `protobuf:"varint,4,opt,name=validator_power,json=validatorPower,proto3"`
	Timestamp        time.Time `protobuf:"bytes,5,opt,name=timestamp,proto3,stdtime"`
}
func (m *tmDuplicateVoteEvidence) Reset()         { *m = tmDuplicateVoteEvidence{} }
func (m *tmDuplicateVoteEvidence) String() string { return "" }
func (*tmDuplicateVoteEvidence) ProtoMessage()    {}


// tmVote from tendermint/proto/tendermint/types/types.proto
type tmVote struct {
	Type             int32     `protobuf:"varint,1,opt,name=type,proto3"`
	Height           int64     `protobuf:"varint,2,opt,name=height,proto3"`
	Round            int32     `protobuf:"varint,3,opt,name=round,proto3"`
	BlockID          tmBlockID `protobuf:"bytes,4,opt,name=block_id,json=blockId,proto3"`
	Timestamp        time.Time `protobuf:"bytes,5,opt,name=timestamp,proto3,stdtime"`
	ValidatorAddress []byte    `protobuf:"bytes,6,opt,name=validator_address,json=validatorAddress,proto3"`
	ValidatorIndex   int32     `protobuf:"varint,7,opt,name=validator_index,json=validatorIndex,proto3"`
	Signature        []byte    `protobuf:"bytes,8,opt,name=signature,proto3"`
}


// tmLightClientAttackEvidence from tendermint/proto/tendermint/types/evidence.proto
type tmLightClientAttackEvidence struct {
	ConflictingBlock    *tmLightBlock   `protobuf:"bytes,1,opt,name=conflicting_block,json=conflictingBlock,proto3"`
	CommonHeight        int64           `protobuf:"varint,2,opt,name=common_height,json=commonHeight,proto3"`
	ByzantineValidators []*tmValidator  `protobuf:"bytes,3,rep,name=byzantine_validators,json=byzantineValidators,proto3"`
	TotalVotingPower    int64           `protobuf:"varint,4,opt,name=total_voting_power,json=totalVotingPower,proto3"`
	Timestamp           time.Time       `protobuf:"bytes,5,opt,name=timestamp,proto3,stdtime"`
}
func (m *tmLightClientAttackEvidence) Reset()         { *m = tmLightClientAttackEvidence{} }
func (m *tmLightClientAttackEvidence) String() string { return "" }
func (*tmLightClientAttackEvidence) ProtoMessage()    {}


// tmLightBlock from tendermint/proto/tendermint/types/types.proto
type tmLightBlock struct {
	SignedHeader *tmSignedHeader `protobuf:"bytes,1,opt,name=signed_header,json=signedHeader,proto3"`
	ValidatorSet *tmValidatorSet `protobuf:"bytes,2,opt,name=validator_set,json=validatorSet,proto3"`
}


// tmSignedHeader from tendermint/proto/tendermint/types/types.proto
type tmSignedHeader struct {
	Header *tmHeader `protobuf:"bytes,1,opt,name=header,proto3"`
	Commit *tmCommit `protobuf:"bytes,2,opt,name=commit,proto3"`
}


// tmValidatorSet from tendermint/proto/tendermint/types/validator.proto
type tmValidatorSet struct {
	Validators       []*tmValidator `protobuf:"bytes,1,rep,name=validators,proto3"`
	Proposer         *tmValidator   `protobuf:"bytes,2,opt,name=proposer,proto3"`
	TotalVotingPower int64          `protobuf:"varint,3,opt,name=total_voting_power,json=totalVotingPower,proto3"`
}
func (m *tmValidatorSet) Reset()         { *m = tmValidatorSet{} }
func (m *tmValidatorSet) String() string { return "" }
func (*tmValidatorSet) ProtoMessage()    {}


// tmValidator from tendermint/proto/tendermint/types/validator.proto
type tmValidator struct {
	Address          []byte `protobuf:"bytes,1,opt,name=address,proto3"`
	PubKey           []byte `protobuf:"bytes,2,opt,name=pub_key,json=pubKey,proto3"`
	VotingPower      int64  `protobuf:"varint,3,opt,name=voting_power,json=votingPower,proto3"`
	ProposerPriority int64  `protobuf:"varint,4,opt,name=proposer_priority,json=proposerPriority,proto3"`
}


// tmABCIResponses from tendermint/proto/tendermint/state/types.proto
type tmABCIResponses struct {
	DeliverTxs []*tmResponseDeliverTx `protobuf:"bytes,1,rep,name=deliver_txs,json=deliverTxs,proto3"`
	EndBlock   *tmResponseEndBlock    `protobuf:"bytes,2,opt,name=end_block,json=endBlock,proto3"`
	BeginBlock *tmResponseBeginBlock  `protobuf:"bytes,3,opt,name=begin_block,json=beginBlock,proto3"`
}
func (m *tmABCIResponses) Reset()         { *m = tmABCIResponses{} }
func (m *tmABCIResponses) String() string { return "" }
func (*tmABCIResponses) ProtoMessage()    {}

// tmResponseDeliverTx from tendermint/proto/tendermint/abci/types.proto
type tmResponseDeliverTx struct {
	Code      uint32    `protobuf:"varint,1,opt,name=code,proto3"`
	Data      []byte    `protobuf:"bytes,2,opt,name=data,proto3"`
	Log       string    `protobuf:"bytes,3,opt,name=log,proto3"`
	Info      string    `protobuf:"bytes,4,opt,name=info,proto3"`
	GasWanted int64     `protobuf:"varint,5,opt,name=gas_wanted,json=gasWanted,proto3"`
	GasUsed   int64     `protobuf:"varint,6,opt,name=gas_used,json=gasUsed,proto3"`
	Events    []tmEvent `protobuf:"bytes,7,rep,name=events,proto3"`
	Codespace string    `protobuf:"bytes,8,opt,name=codespace,proto3"`
}

// tmResponseBeginBlock from tendermint/proto/tendermint/abci/types.proto
type tmResponseBeginBlock struct {
	Events []tmEvent `protobuf:"bytes,1,rep,name=events,proto3"`
}


// tmResponseEndBlock from tendermint/proto/tendermint/abci/types.proto
type tmResponseEndBlock struct {
	ValidatorUpdates      []tmValidatorUpdate  `protobuf:"bytes,1,rep,name=validator_updates,json=validatorUpdates,proto3"`
	ConsensusParamUpdates *tmConsensusParams   `protobuf:"bytes,2,opt,name=consensus_param_updates,json=consensusParamUpdates,proto3"`
	Events                []tmEvent            `protobuf:"bytes,3,rep,name=events,proto3"`
}


// tmEvent from tendermint/proto/tendermint/abci/types.proto
type tmEvent struct {
	Type       string              `protobuf:"bytes,1,opt,name=type,proto3"`
	Attributes []tmEventAttribute  `protobuf:"bytes,2,rep,name=attributes,proto3"`
}


// tmEventAttribute from tendermint/proto/tendermint/abci/types.proto
type tmEventAttribute struct {
	Key   []byte `protobuf:"bytes,1,opt,name=key,proto3"`
	Value []byte `protobuf:"bytes,2,opt,name=value,proto3"`
	Index bool   `protobuf:"varint,3,opt,name=index,proto3"`
}


// tmValidatorUpdate from tendermint/proto/tendermint/abci/types.proto
type tmValidatorUpdate struct {
	PubKey []byte `protobuf:"bytes,1,opt,name=pub_key,json=pubKey,proto3"`
	Power  int64  `protobuf:"varint,2,opt,name=power,proto3"`
}


// tmConsensusParams from tendermint/proto/tendermint/types/params.proto
type tmConsensusParams struct {
	Block     tmBlockParams     `protobuf:"bytes,1,opt,name=block,proto3"`
	Evidence  tmEvidenceParams  `protobuf:"bytes,2,opt,name=evidence,proto3"`
	Validator tmValidatorParams `protobuf:"bytes,3,opt,name=validator,proto3"`
	Version   tmVersionParams   `protobuf:"bytes,4,opt,name=version,proto3"`
}
func (m *tmConsensusParams) Reset()         { *m = tmConsensusParams{} }
func (m *tmConsensusParams) String() string { return "" }
func (*tmConsensusParams) ProtoMessage()    {}


// tmBlockParams from tendermint/proto/tendermint/types/params.proto
type tmBlockParams struct {
	MaxBytes int64 `protobuf:"varint,1,opt,name=max_bytes,json=maxBytes,proto3"`
	MaxGas   int64 `protobuf:"varint,2,opt,name=max_gas,json=maxGas,proto3"`
}


// tmEvidenceParams from tendermint/proto/tendermint/types/params.proto
type tmEvidenceParams struct {
	MaxAgeNumBlocks int64 `protobuf:"varint,1,opt,name=max_age_num_blocks,json=maxAgeNumBlocks,proto3"`
	MaxAgeDuration  int64 `protobuf:"varint,2,opt,name=max_age_duration,json=maxAgeDuration,proto3"`
	MaxBytes        int64 `protobuf:"varint,3,opt,name=max_bytes,json=maxBytes,proto3"`
}


// tmValidatorParams from tendermint/proto/tendermint/types/params.proto
type tmValidatorParams struct {
	PubKeyTypes []string `protobuf:"bytes,1,rep,name=pub_key_types,json=pubKeyTypes,proto3"`
}


// tmVersionParams from tendermint/proto/tendermint/types/params.proto
type tmVersionParams struct {
	AppVersion uint64 `protobuf:"varint,1,opt,name=app_version,json=appVersion,proto3"`
}


// tmState from tendermint/proto/tendermint/state/types.proto
type tmState struct {
	Version         uint64 `protobuf:"varint,1,opt,name=version,proto3"`
	ChainID         string `protobuf:"bytes,2,opt,name=chain_id,json=chainId,proto3"`
	InitialHeight   int64  `protobuf:"varint,3,opt,name=initial_height,json=initialHeight,proto3"`
	LastBlockHeight int64  `protobuf:"varint,4,opt,name=last_block_height,json=lastBlockHeight,proto3"`
}
func (m *tmState) Reset()         { *m = tmState{} }
func (m *tmState) String() string { return "" }
func (*tmState) ProtoMessage()    {}


// =============================================================================
// CometBFT Protobuf Deserialization Types
// =============================================================================
// Source: github.com/cometbft/cometbft v0.37.x / v0.38.x
// https://github.com/cometbft/cometbft/blob/main/proto/cometbft/abci/v1/types.proto
// https://github.com/cometbft/cometbft/blob/main/proto/cometbft/types/v1/evidence.proto
// For proto.Unmarshal() to work types need to implement proto.Message interface.

// Type aliases for cb* types identical to tm* types
// Source: cometbft/proto/cometbft/abci/v1/types.proto
type cbEvent = tmEvent
type cbEventAttribute = tmEventAttribute
type cbValidatorUpdate = tmValidatorUpdate
type cbConsensusParams = tmConsensusParams
type cbBlockParams = tmBlockParams
type cbEvidenceParams = tmEvidenceParams
type cbValidatorParams = tmValidatorParams
type cbVersionParams = tmVersionParams
type cbBlockID = tmBlockID
type cbPartSetHeader = tmPartSetHeader
type cbInt64Value = tmInt64Value

// cbResponseFinalizeBlock from CometBFT ABCI++ (v0.37+)
// Source: cometbft/proto/cometbft/abci/v1/types.proto
type cbResponseFinalizeBlock struct {
	Events                []cbEvent           `protobuf:"bytes,1,rep,name=events,proto3"`
	TxResults             []*cbExecTxResult   `protobuf:"bytes,2,rep,name=tx_results,json=txResults,proto3"`
	ValidatorUpdates      []cbValidatorUpdate `protobuf:"bytes,3,rep,name=validator_updates,json=validatorUpdates,proto3"`
	ConsensusParamUpdates *cbConsensusParams  `protobuf:"bytes,4,opt,name=consensus_param_updates,json=consensusParamUpdates,proto3"`
	AppHash               []byte              `protobuf:"bytes,5,opt,name=app_hash,json=appHash,proto3"`
}
func (m *cbResponseFinalizeBlock) Reset()         { *m = cbResponseFinalizeBlock{} }
func (m *cbResponseFinalizeBlock) String() string { return "" }
func (*cbResponseFinalizeBlock) ProtoMessage()    {}

// cbExecTxResult from CometBFT ABCI++ (v0.37+)
// Source: cometbft/proto/cometbft/abci/v1/types.proto
type cbExecTxResult struct {
	Code      uint32    `protobuf:"varint,1,opt,name=code,proto3"`
	Data      []byte    `protobuf:"bytes,2,opt,name=data,proto3"`
	Log       string    `protobuf:"bytes,3,opt,name=log,proto3"`
	Info      string    `protobuf:"bytes,4,opt,name=info,proto3"`
	GasWanted int64     `protobuf:"varint,5,opt,name=gas_wanted,json=gasWanted,proto3"`
	GasUsed   int64     `protobuf:"varint,6,opt,name=gas_used,json=gasUsed,proto3"`
	Events    []cbEvent `protobuf:"bytes,7,rep,name=events,proto3"`
	Codespace string    `protobuf:"bytes,8,opt,name=codespace,proto3"`
}

// cbDuplicateVoteEvidence from CometBFT
// Source: cometbft/proto/cometbft/types/v1/evidence.proto
type cbDuplicateVoteEvidence struct {
	VoteA            *cbVote   `protobuf:"bytes,1,opt,name=vote_a,json=voteA,proto3"`
	VoteB            *cbVote   `protobuf:"bytes,2,opt,name=vote_b,json=voteB,proto3"`
	TotalVotingPower int64     `protobuf:"varint,3,opt,name=total_voting_power,json=totalVotingPower,proto3"`
	ValidatorPower   int64     `protobuf:"varint,4,opt,name=validator_power,json=validatorPower,proto3"`
	Timestamp        time.Time `protobuf:"bytes,5,opt,name=timestamp,proto3,stdtime"`
}
func (m *cbDuplicateVoteEvidence) Reset()         { *m = cbDuplicateVoteEvidence{} }
func (m *cbDuplicateVoteEvidence) String() string { return "" }
func (*cbDuplicateVoteEvidence) ProtoMessage()    {}

// cbVote from CometBFT
// Source: cometbft/proto/cometbft/types/v1/types.proto
type cbVote struct {
	Type             int32     `protobuf:"varint,1,opt,name=type,proto3"`
	Height           int64     `protobuf:"varint,2,opt,name=height,proto3"`
	Round            int32     `protobuf:"varint,3,opt,name=round,proto3"`
	BlockID          *cbBlockID `protobuf:"bytes,4,opt,name=block_id,json=blockId,proto3"`
	Timestamp        time.Time `protobuf:"bytes,5,opt,name=timestamp,proto3,stdtime"`
	ValidatorAddress []byte    `protobuf:"bytes,6,opt,name=validator_address,json=validatorAddress,proto3"`
	ValidatorIndex   int32     `protobuf:"varint,7,opt,name=validator_index,json=validatorIndex,proto3"`
	Signature        []byte    `protobuf:"bytes,8,opt,name=signature,proto3"`
}


// =============================================================================
// Consensus MKVS Format Mappings
// =============================================================================

// ConsensusMKVSFormat describes the value format for a consensus MKVS key prefix.
type ConsensusMKVSFormat struct {
	Format      string // "cbor", "binary", "empty"
	Type        string // Go type name or description
	Description string // Human-readable description
}

// consensusMKVSFormats maps consensus MKVS key prefixes to their value formats.
// Extracted from _oasis-core/go/consensus/tendermint/apps/*/state/state.go
// This provides deterministic decoding without heuristics.
var consensusMKVSFormats = map[byte]ConsensusMKVSFormat{
	// Registry Module (0x10-0x19)
	// See: _oasis-core/go/consensus/tendermint/apps/registry/state/state.go
	0x10: {Format: "cbor", Type: "entity.SignedEntity", Description: "signed entity"},
	0x11: {Format: "cbor", Type: "node.MultiSignedNode", Description: "signed node"},
	0x12: {Format: "empty", Type: "index", Description: "node by entity index"},
	0x13: {Format: "cbor", Type: "registry.Runtime", Description: "runtime"},
	0x14: {Format: "binary", Type: "signature.PublicKey", Description: "node by consensus address"},
	0x15: {Format: "cbor", Type: "registry.NodeStatus", Description: "node status"},
	0x16: {Format: "cbor", Type: "registry.ConsensusParameters", Description: "registry parameters"},
	0x17: {Format: "binary", Type: "signature.PublicKey", Description: "key map"},
	0x18: {Format: "cbor", Type: "registry.Runtime", Description: "suspended runtime"},
	0x19: {Format: "empty", Type: "index", Description: "runtime by entity index"},

	// Roothash Module (0x20-0x29)
	// See: _oasis-core/go/consensus/tendermint/apps/roothash/state/state.go
	0x20: {Format: "cbor", Type: "roothash.RuntimeState", Description: "runtime state"},
	0x21: {Format: "cbor", Type: "roothash.ConsensusParameters", Description: "roothash parameters"},
	0x22: {Format: "binary", Type: "common.Namespace", Description: "round timeout queue"},
	0x24: {Format: "empty", Type: "index", Description: "evidence"},
	0x25: {Format: "binary", Type: "hash.Hash", Description: "state root"},
	0x26: {Format: "binary", Type: "hash.Hash", Description: "io root"},
	0x27: {Format: "cbor", Type: "roothash.RoundResults", Description: "last round results"},
	0x28: {Format: "cbor", Type: "message.IncomingMessageQueueMeta", Description: "incoming message queue meta"},
	0x29: {Format: "cbor", Type: "message.IncomingMessage", Description: "incoming message queue"},

	// Beacon Module (0x40-0x45)
	// See: _oasis-core/go/consensus/tendermint/apps/beacon/state/state.go
	0x40: {Format: "cbor", Type: "beacon.EpochTimeState", Description: "epoch current"},
	0x41: {Format: "cbor", Type: "beacon.EpochTimeState", Description: "epoch future"},
	0x42: {Format: "binary", Type: "beacon-32bytes", Description: "beacon value"},
	0x43: {Format: "cbor", Type: "beacon.ConsensusParameters", Description: "beacon parameters"},
	0x45: {Format: "cbor", Type: "beacon.EpochTime", Description: "pending mock epoch"},

	// Staking Module (0x50-0x59)
	// See: _oasis-core/go/consensus/tendermint/apps/staking/state/state.go
	0x50: {Format: "cbor", Type: "staking.Account", Description: "account"},
	0x51: {Format: "cbor", Type: "quantity.Quantity", Description: "total supply"},
	0x52: {Format: "cbor", Type: "quantity.Quantity", Description: "common pool"},
	0x53: {Format: "cbor", Type: "staking.Delegation", Description: "delegation"},
	0x54: {Format: "cbor", Type: "staking.DebondingDelegation", Description: "debonding delegation"},
	0x55: {Format: "empty", Type: "index", Description: "debonding queue"},
	0x56: {Format: "cbor", Type: "staking.ConsensusParameters", Description: "staking parameters"},
	0x57: {Format: "cbor", Type: "quantity.Quantity", Description: "last block fees"},
	0x58: {Format: "cbor", Type: "EpochSigning", Description: "epoch signing"},
	0x59: {Format: "cbor", Type: "quantity.Quantity", Description: "governance deposits"},

	// Scheduler Module (0x60-0x63)
	// See: _oasis-core/go/consensus/tendermint/apps/scheduler/state/state.go
	0x60: {Format: "cbor", Type: "api.Committee", Description: "committee"},
	0x61: {Format: "cbor", Type: "map[signature.PublicKey]int64", Description: "current validators"},
	0x62: {Format: "cbor", Type: "map[signature.PublicKey]int64", Description: "pending validators"},
	0x63: {Format: "cbor", Type: "api.ConsensusParameters", Description: "scheduler parameters"},

	// Keymanager Module (0x70)
	// See: _oasis-core/go/consensus/tendermint/apps/keymanager/state/state.go
	0x70: {Format: "cbor", Type: "api.Status", Description: "keymanager status"},

	// Governance Module (0x80-0x85)
	// See: _oasis-core/go/consensus/tendermint/apps/governance/state/state.go
	0x80: {Format: "cbor", Type: "uint64", Description: "next proposal identifier"},
	0x81: {Format: "cbor", Type: "governance.Proposal", Description: "proposals"},
	0x82: {Format: "empty", Type: "index", Description: "active proposals"},
	0x83: {Format: "cbor", Type: "governance.Vote", Description: "votes"},
	0x84: {Format: "empty", Type: "index", Description: "pending upgrades"},
	0x85: {Format: "cbor", Type: "governance.ConsensusParameters", Description: "governance parameters"},
}

// GetConsensusMKVSFormat returns the format metadata for a consensus MKVS key.
// Returns (format, true) if the key prefix is known, (empty, false) otherwise.
func GetConsensusMKVSFormat(key []byte) (ConsensusMKVSFormat, bool) {
	if len(key) == 0 {
		return ConsensusMKVSFormat{}, false
	}
	format, exists := consensusMKVSFormats[key[0]]
	return format, exists
}
