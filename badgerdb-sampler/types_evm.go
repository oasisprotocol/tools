package main

// EVMDataInfo represents decoded EVM storage data.
// See: _oasis-sdk/runtime-sdk/modules/evm/src/state.rs
type EVMDataInfo struct {
	StorageType   string `json:"storage_type"`              // "code", "storage", "block_hash", "confidential_storage"
	Address       string `json:"address,omitempty"`         // H160 (20 bytes) - contract address (0x-prefixed hex)
	StorageSlot   string `json:"storage_slot,omitempty"`    // H256 (32 bytes) - storage slot (0x-prefixed hex, truncated)
	StorageValue  string `json:"storage_value,omitempty"`   // H256 (32 bytes) - storage value (0x-prefixed hex)
	RuntimeHeight uint64 `json:"runtime_height,omitempty"`  // For block_hash type (runtime block height)
	BlockHash     string `json:"block_hash,omitempty"`      // H256 (32 bytes) - block hash (0x-prefixed hex, truncated)
	CodeSize      int    `json:"code_size,omitempty"`       // Size of contract bytecode
	CodeHex       string `json:"code_hex,omitempty"`        // Truncated bytecode (hex, no 0x prefix)
}

// EVMEventInfo represents a decoded EVM Log event.
// See: _oasis-sdk/runtime-sdk/modules/evm/src/lib.rs:253-263 (Event::Log)
// See: _oasis-sdk/client-sdk/go/modules/evm/types.go:56-61 (Event)
type EVMEventInfo struct {
	// Raw fields
	RawDump  string `json:"raw_dump,omitempty"`
	RawSize  int    `json:"raw_size"`
	RawError string `json:"raw_error,omitempty"`

	// Decoded fields
	Address        string   `json:"address"`                    // H160 contract address (0x-prefixed hex)
	TopicCount     int      `json:"topic_count"`                // Number of topics (0-4)
	Topics         []string `json:"topics,omitempty"`           // H256 topics array (0x-prefixed hex)
	EventSignature string   `json:"event_signature,omitempty"`  // Human-readable signature (if known)
	EventHash      string   `json:"event_hash,omitempty"`       // topic[0] keccak256 hash (0x-prefixed)
	DataSize       int      `json:"data_size"`                  // Size of non-indexed data
	DataHex        string   `json:"data_hex,omitempty"`         // Truncated raw data (0x-prefixed hex)
}

// EVMTxInputInfo represents decoded EVM transaction input artifacts.
// See: _oasis-core/go/runtime/transaction/transaction.go:126-140 (inputArtifacts)
type EVMTxInputInfo struct {
	// Raw fields
	RawDump  string `json:"raw_dump,omitempty"`
	RawSize  int    `json:"raw_size"`
	RawError string `json:"raw_error,omitempty"`

	// Decoded fields
	TxHash     string              `json:"tx_hash"`           // Transaction hash (0x-prefixed hex)
	BatchOrder uint32              `json:"batch_order"`       // Order within batch
	Method     string              `json:"method,omitempty"`  // SDK method (e.g., "evm.Call")
	EVMTx      *EVMTransactionInfo `json:"evm_tx,omitempty"`  // Decoded EVM transaction
}

// EVMTransactionInfo represents a decoded EVM transaction.
type EVMTransactionInfo struct {
	// Raw EVM transaction body bytes (CBOR)
	RawDump  string `json:"raw_dump,omitempty"`
	RawSize  int    `json:"raw_size"`
	RawError string `json:"raw_error,omitempty"`

	// Decoded EVM transaction fields
	Type     string `json:"type"`                  // "call" or "create"
	From     string `json:"from,omitempty"`        // 0x prefix
	To       string `json:"to,omitempty"`          // 0x prefix
	Value    string `json:"value,omitempty"`       // Wei as decimal string
	GasLimit uint64 `json:"gas_limit,omitempty"`
	GasPrice string `json:"gas_price,omitempty"`   // Wei as decimal string
	Nonce    uint64 `json:"nonce,omitempty"`
	DataSize int    `json:"data_size"`
	DataDump string `json:"data_dump,omitempty"`   // 0x-hex
}

// EVMTxOutputInfo represents decoded EVM transaction output artifacts.
// See: _oasis-core/go/runtime/transaction/transaction.go:142-150 (outputArtifacts)
type EVMTxOutputInfo struct {
	// Raw fields
	RawDump  string `json:"raw_dump,omitempty"`
	RawSize  int    `json:"raw_size"`
	RawError string `json:"raw_error,omitempty"`

	// Decoded execution status fields
	SuccessExecution bool   `json:"success"`
	ResultSize       int    `json:"result_size,omitempty"`
	ResultDump       string `json:"result_dump,omitempty"`
	ErrorExecution   string `json:"error,omitempty"` // Execution error (application data, not decoding error)
}
