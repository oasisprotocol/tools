package main

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/gogo/protobuf/proto"
)

// decodeConsensusBlockstoreKey parses consensus-blockstore key and returns structured info.
// See: tendermint/store/store.go for key formats (H:, P:, C:, SC:, BH:)
func decodeConsensusBlockstoreKey(key []byte) ConsensusBlockstoreKeyInfo {
	info := ConsensusBlockstoreKeyInfo{}

	if len(key) < 2 {
		info.KeyType = "unknown"
		return info
	}

	// All keys start with 0x01 (dbVersion from Oasis BadgerDB wrapper)
	if key[0] != 0x01 {
		info.KeyType = "unknown"
		return info
	}

	info.KeySize = len(key)

	// Parse ASCII key after 0x01 prefix
	asciiKey := string(key[1:])
	info.KeyRaw = asciiKey

	// blockStore state key
	if asciiKey == "blockStore" {
		info.KeyType = "blockstore_state"
		return info
	}

	// Parse structured keys: "H:{height}", "P:{height}:{part}", "C:{height}", "SC:{height}", "BH:{hash}"
	if strings.HasPrefix(asciiKey, "H:") {
		info.KeyType = "block_meta"
		if h, err := strconv.ParseInt(asciiKey[2:], 10, 64); err == nil {
			info.ConsensusHeight = h
		}
		return info
	}
	if strings.HasPrefix(asciiKey, "P:") {
		info.KeyType = "block_part"
		// Format: P:{height}:{part}
		parts := strings.Split(asciiKey[2:], ":")
		if len(parts) >= 1 {
			if h, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
				info.ConsensusHeight = h
			}
		}
		if len(parts) >= 2 {
			if p, err := strconv.Atoi(parts[1]); err == nil {
				info.PartIndex = p
			}
		}
		return info
	}
	if strings.HasPrefix(asciiKey, "C:") {
		info.KeyType = "block_commit"
		if h, err := strconv.ParseInt(asciiKey[2:], 10, 64); err == nil {
			info.ConsensusHeight = h
		}
		return info
	}
	if strings.HasPrefix(asciiKey, "SC:") {
		info.KeyType = "seen_commit"
		if h, err := strconv.ParseInt(asciiKey[3:], 10, 64); err == nil {
			info.ConsensusHeight = h
		}
		return info
	}
	if strings.HasPrefix(asciiKey, "BH:") {
		info.KeyType = "block_hash"
		info.Hash = asciiKey[3:]
		return info
	}

	info.KeyType = "unknown"
	return info
}

// decodeConsensusBlockstoreValue decodes protobuf value and returns structured info.
// See: tendermint/proto/tendermint/store and tendermint/proto/tendermint/types
func decodeConsensusBlockstoreValue(keyType string, value []byte) ConsensusBlockstoreValueInfo {
	info := ConsensusBlockstoreValueInfo{}

	switch keyType {
	case "blockstore_state":
		var state tmBlockStoreState
		if err := proto.Unmarshal(value, &state); err == nil {
			info.State = &ConsensusBlockStoreState{
				Base:            state.Base,
				ConsensusHeight: state.Height,
			}
		} else {
			info.StateError = err.Error()
		}

	case "block_meta":
		var meta tmBlockMeta
		if err := proto.Unmarshal(value, &meta); err == nil {
			blockMeta := &ConsensusBlockMetaInfo{
				ConsensusHeight: meta.Header.Height,
				NumTxs:          meta.NumTxs,
			}

			// Extract timestamp
			if meta.Header.Time.Unix() > 0 {
				info.Timestamp = meta.Header.Time.Unix()
				blockMeta.Time = meta.Header.Time.Format(time.RFC3339)
			}

			// Format AppHash (truncate to 16 hex chars)
			blockMeta.AppHash = truncateHex(meta.Header.AppHash, TruncateHashSize)

			// Truncate ChainID
			chainID := meta.Header.ChainID
			if len(chainID) > 20 {
				chainID = chainID[:20] + "..."
			}
			blockMeta.ChainID = chainID

			info.BlockMeta = blockMeta
		} else {
			info.BlockMetaError = err.Error()
		}

	case "block_part":
		var part tmPart
		if err := proto.Unmarshal(value, &part); err == nil {
			info.Part = &ConsensusPartInfo{
				Index:      part.Index,
				PartSize:   len(part.Bytes),
				ProofTotal: part.Proof.Total,
			}
		} else {
			info.PartError = err.Error()
		}

	case "block_commit", "seen_commit":
		var commit tmCommit
		if err := proto.Unmarshal(value, &commit); err == nil {
			info.Commit = &ConsensusCommitInfo{
				ConsensusHeight: commit.Height,
				Round:           commit.Round,
				Signatures:      len(commit.Signatures),
			}
		} else {
			info.CommitError = err.Error()
		}

	case "block_hash":
		// Block hash values are plain strings containing height numbers
		if height, err := strconv.ParseInt(string(value), 10, 64); err == nil {
			info.HashHeight = height
		}
		// Note: No error field for block_hash as HashHeight is a simple int64 field
	}

	return info
}

// decodeConsensusEvidenceKey parses consensus-evidence key and returns structured info.
func decodeConsensusEvidenceKey(key []byte) ConsensusEvidenceKeyInfo {
	info := ConsensusEvidenceKeyInfo{
		KeySize: len(key),
		KeyHex:  truncateHex(key, TruncateLongSize),
	}

	if len(key) < 2 {
		info.KeyType = "unknown"
		return info
	}

	// All keys start with 0x01 (dbVersion from Oasis BadgerDB wrapper)
	if key[0] != 0x01 {
		info.KeyType = "unknown"
		return info
	}

	// Evidence DB is typically empty or uses simple key patterns
	info.KeyType = fmt.Sprintf("type_%02x", key[1])
	info.PrefixByte = key[1]
	return info
}

// decodeConsensusEvidenceValue decodes evidence value and returns structured info.
// Returns (info, error) where error is a parsing error.
func decodeConsensusEvidenceValue(keyType string, value []byte) (*ConsensusEvidenceValueInfo, string) {
	info := &ConsensusEvidenceValueInfo{
		ValueSize: len(value),
		ValueHex:  truncateHex(value, TruncateLongSize),
	}

	if len(value) == 0 {
		return info, ""
	}

	// Recover from protobuf panics that occur with schema mismatches
	var panicErr string
	defer func() {
		if r := recover(); r != nil {
			panicErr = fmt.Sprintf("protobuf panic: %v", r)
		}
	}()

	// Try to decode as DuplicateVoteEvidence first
	var dve tmDuplicateVoteEvidence
	if err := proto.Unmarshal(value, &dve); err == nil && dve.VoteA != nil {
		info.EvidenceType = "duplicate_vote"
		info.VoteAHeight = dve.VoteA.Height
		if dve.VoteB != nil {
			info.VoteBHeight = dve.VoteB.Height
		}
		info.TotalVotingPower = dve.TotalVotingPower
		info.ValidatorPower = dve.ValidatorPower
		if dve.Timestamp.Unix() > 0 {
			info.Timestamp = dve.Timestamp.Format(time.RFC3339)
		}
		return info, panicErr
	}

	// Try to decode as LightClientAttackEvidence
	var lca tmLightClientAttackEvidence
	if err := proto.Unmarshal(value, &lca); err == nil && lca.ConflictingBlock != nil {
		info.EvidenceType = "light_client_attack"
		info.TotalVotingPower = lca.TotalVotingPower
		if lca.Timestamp.Unix() > 0 {
			info.Timestamp = lca.Timestamp.Format(time.RFC3339)
		}
		return info, panicErr
	}

	// If all parsing failed, show raw value
	info.EvidenceType = "unknown"
	if panicErr != "" {
		return info, panicErr
	}
	return info, "unable to decode evidence format"
}

// decodeConsensusMkvsKey parses consensus-mkvs key and returns structured info.
// Keys use keyformat encoding: [type_byte][data...]
// See: _oasis-core/go/storage/mkvs/db/badger/badger.go:31-66
func decodeConsensusMkvsKey(key []byte) ConsensusMkvsKeyInfo {
	info := ConsensusMkvsKeyInfo{
		KeySize: len(key),
		KeyHex:  truncateHex(key, TruncateLongSize),
	}

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
			info.ConsensusHeight = binary.BigEndian.Uint64(data[0:8])
			if len(data) >= 8+33 {
				info.RootType = data[8]
				info.Hash = truncateHex(data[9:9+32], TruncateHashSize)
			}
		}

	case 0x02:
		info.KeyType = "roots_metadata"
		if len(data) >= 8 {
			info.ConsensusHeight = binary.BigEndian.Uint64(data[0:8])
		}

	case 0x03:
		info.KeyType = "root_updated_nodes"
		if len(data) >= 8 {
			info.ConsensusHeight = binary.BigEndian.Uint64(data[0:8])
			if len(data) >= 8+33 {
				info.RootType = data[8]
				info.Hash = truncateHex(data[9:9+32], TruncateHashSize)
			}
		}

	case 0x04:
		info.KeyType = "metadata"

	case 0x05:
		info.KeyType = "multipart_restore_log"
		if len(data) >= 33 {
			info.RootType = data[0]
			info.Hash = truncateHex(data[1:33], TruncateHashSize)
		} else {
			info.Hash = truncateHex(data, TruncateHashSize)
		}

	case 0x06:
		info.KeyType = "root_node"
		if len(data) >= 33 {
			info.RootType = data[0]
			info.Hash = truncateHex(data[1:33], TruncateHashSize)
		} else {
			info.Hash = truncateHex(data, TruncateHashSize)
		}

	default:
		info.KeyType = fmt.Sprintf("unknown_%02x", prefixByte)
	}

	return info
}

// decodeConsensusMkvsValue decodes consensus MKVS value and returns structured info.
// See: _oasis-core/go/storage/mkvs/node/node.go:26-32 (prefixes), 294-309 (InternalNode), 531-537 (LeafNode)
func decodeConsensusMkvsValue(keyType string, value []byte) ConsensusMkvsValueInfo {
	info := ConsensusMkvsValueInfo{
		NodeSize: len(value),
	}

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
	case 0x00: // LeafNode
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

		// Decode consensus module key prefix
		module := decodeConsensusModulePrefix(key)

		leaf := &ConsensusMkvsLeafInfo{
			Module:  module,
			KeySize: keySize,
			KeyHex:  truncateHex(key, TruncateLongSize),
		}

		// Extract Oasis address for staking-related modules
		// Staking keys have format: module_prefix (1 byte) + 21-byte Oasis address
		if keySize == 22 {
			switch key[0] {
			case 0x34, 0x35, 0x36, 0x37: // staking accounts, delegations, debonding, allowances
				leaf.OasisAddress = (*(*OasisAddress)(key[1:22])).String()
			}
		}
		// Entity/node keys also contain addresses
		if keySize >= 22 {
			switch key[0] {
			case 0x40, 0x41: // registry entities, nodes
				leaf.OasisAddress = (*(*OasisAddress)(key[1:22])).String()
			}
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

		// Try CBOR decode
		var decoded interface{}
		if err := cbor.Unmarshal(leafValue, &decoded); err == nil {
			leaf.ValueFormatted = formatCBOR(decoded, valueSize)
		}

		info.Leaf = leaf
		return info

	case 0x01: // InternalNode
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
			info.Internal = &ConsensusMkvsInternalInfo{LabelBits: labelBits}
			return info
		}

		data = data[labelBytes:] // skip label

		internal := &ConsensusMkvsInternalInfo{LabelBits: labelBits}

		// Check for embedded leaf node or nil marker
		if len(data) < 1 {
			info.InternalError = "missing leaf/nil marker"
			info.Internal = internal
			return info
		}

		if data[0] == 0x02 { // NilNode marker - no embedded leaf
			internal.HasLeaf = false
			data = data[1:] // skip nil marker
		} else if data[0] == 0x00 { // LeafNode prefix - embedded leaf present
			internal.HasLeaf = true
			// Skip embedded leaf: prefix(1) + keyLen(2) + key + valueLen(4) + value
			data = data[1:] // skip prefix
			if len(data) < 2 {
				info.InternalError = "embedded leaf key length missing"
				info.Internal = internal
				return info
			}
			keyLen := binary.LittleEndian.Uint16(data[0:2])
			data = data[2:]
			if len(data) < int(keyLen)+4 {
				info.InternalError = "embedded leaf truncated"
				info.Internal = internal
				return info
			}
			data = data[keyLen:] // skip key
			valueLen := binary.LittleEndian.Uint32(data[0:4])
			data = data[4:]
			if len(data) < int(valueLen) {
				info.InternalError = "embedded leaf value truncated"
				info.Internal = internal
				return info
			}
			data = data[valueLen:] // skip value
		} else {
			info.InternalError = fmt.Sprintf("unexpected marker 0x%02x (expected 0x00 or 0x02)", data[0])
			info.Internal = internal
			return info
		}

		// Read left and right hashes
		if len(data) >= 32 {
			internal.LeftHash = truncateHex(data[:32], TruncateHashSize)
			data = data[32:]
		}
		if len(data) >= 32 {
			internal.RightHash = truncateHex(data[:32], TruncateHashSize)
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

// decodeConsensusModulePrefix decodes the consensus module key prefix.
// See: _oasis-core/go/consensus/tendermint/apps/*/state/state.go
func decodeConsensusModulePrefix(key []byte) string {
	if len(key) == 0 {
		return "<empty>"
	}

	switch key[0] {
	// Roothash module (0x20-0x29)
	case 0x20:
		return "roothash/runtime_state"
	case 0x21:
		return "roothash/params"
	case 0x22:
		return "roothash/round_timeout"
	case 0x24:
		return "roothash/evidence"
	case 0x25:
		return "roothash/state_root"
	case 0x26:
		return "roothash/io_root"
	case 0x27:
		return "roothash/last_round_results"
	case 0x28:
		return "roothash/incoming_msg_queue_meta"
	case 0x29:
		return "roothash/incoming_msg_queue"
	// Staking module (0x30-0x3F)
	case 0x30:
		return "staking/total_supply"
	case 0x31:
		return "staking/common_pool"
	case 0x32:
		return "staking/last_block_fees"
	case 0x33:
		return "staking/governance_deposits"
	case 0x34:
		return "staking/accounts"
	case 0x35:
		return "staking/delegations"
	case 0x36:
		return "staking/debonding_delegations"
	case 0x37:
		return "staking/allowances"
	case 0x38:
		return "staking/params"
	// Registry module (0x40-0x4F)
	case 0x40:
		return "registry/entities"
	case 0x41:
		return "registry/nodes"
	case 0x42:
		return "registry/node_by_consensus"
	case 0x43:
		return "registry/runtimes"
	case 0x44:
		return "registry/suspended_runtimes"
	case 0x45:
		return "registry/params"
	case 0x46:
		return "registry/node_status"
	// Scheduler module (0x50-0x5F)
	case 0x50:
		return "scheduler/params"
	case 0x51:
		return "scheduler/committees"
	case 0x52:
		return "scheduler/validators"
	// Governance module (0x60-0x6F)
	case 0x60:
		return "governance/params"
	case 0x61:
		return "governance/proposals"
	case 0x62:
		return "governance/active_proposals"
	case 0x63:
		return "governance/votes"
	case 0x64:
		return "governance/pending_upgrades"
	// Beacon module (0x70-0x7F)
	case 0x70:
		return "beacon/params"
	case 0x71:
		return "beacon/future_epoch"
	case 0x72:
		return "beacon/epoch"
	case 0x73:
		return "beacon/pvss_state"
	// Keymanager module (0x80-0x8F)
	case 0x80:
		return "keymanager/status"
	case 0x81:
		return "keymanager/params"
	// Consensus parameters
	case 0xF1:
		return "consensus/params"
	default:
		if key[0] >= 'a' && key[0] <= 'z' {
			return extractModuleName(key)
		}
		return fmt.Sprintf("0x%02x", key[0])
	}
}

// decodeConsensusStateKey parses consensus-state key and returns structured info.
func decodeConsensusStateKey(key []byte) ConsensusStateKeyInfo {
	info := ConsensusStateKeyInfo{}

	if len(key) < 2 {
		info.KeyType = "unknown"
		return info
	}

	// All keys start with 0x01 (dbVersion from Oasis BadgerDB wrapper)
	if key[0] != 0x01 {
		info.KeyType = "unknown"
		return info
	}

	info.KeySize = len(key)

	// Parse ASCII key after 0x01 prefix
	decoded := string(key[1:])
	info.KeyHex = truncateHex(key, TruncateLongSize)

	// Check if it's printable ASCII
	if !isPrintableASCII(decoded) {
		info.KeyType = "binary"
		info.IsBinary = true
		return info
	}

	// Extract key type prefix before colon
	colonIdx := strings.IndexByte(decoded, ':')
	if colonIdx != -1 {
		prefix := decoded[:colonIdx]
		switch prefix {
		case "abciResponsesKey":
			info.KeyType = "abci_responses"
			// Extract height from key like "abciResponsesKey:10000000"
			if colonIdx < len(decoded)-1 {
				if h, err := strconv.ParseInt(decoded[colonIdx+1:], 10, 64); err == nil {
					info.ConsensusHeight = h
				}
			}
		case "consensusParamsKey":
			info.KeyType = "consensus_params"
			// Extract height if present
			if colonIdx < len(decoded)-1 {
				if h, err := strconv.ParseInt(decoded[colonIdx+1:], 10, 64); err == nil {
					info.ConsensusHeight = h
				}
			}
		case "validatorsKey":
			info.KeyType = "validators"
			// Extract height if present
			if colonIdx < len(decoded)-1 {
				if h, err := strconv.ParseInt(decoded[colonIdx+1:], 10, 64); err == nil {
					info.ConsensusHeight = h
				}
			}
		case "stateKey":
			info.KeyType = "state"
		case "genesisDoc":
			info.KeyType = "genesis"
		default:
			info.KeyType = prefix
		}
		return info
	}

	info.KeyType = "text_key"
	return info
}

// decodeConsensusStateValue decodes state value and returns structured info.
func decodeConsensusStateValue(keyType string, value []byte) ConsensusStateValueInfo {
	info := ConsensusStateValueInfo{
		ValueSize: len(value),
		ValueHex:  truncateHex(value, TruncateLongSize),
	}

	if len(value) == 0 {
		return info
	}

	// Add panic recovery for all protobuf unmarshaling operations
	defer func() {
		if r := recover(); r != nil {
			// Protobuf panic occurred (likely schema mismatch)
			errMsg := fmt.Sprintf("protobuf panic: %v", r)
			// Set appropriate error field based on key type
			switch keyType {
			case "abci_responses":
				if info.ABCIResponseError == "" {
					info.ABCIResponseError = errMsg
				}
			case "consensus_params":
				if info.ConsensusParamsError == "" {
					info.ConsensusParamsError = errMsg
				}
			case "validators":
				if info.ValidatorsError == "" {
					info.ValidatorsError = errMsg
				}
			case "state", "genesis":
				if info.StateError == "" {
					info.StateError = errMsg
				}
			}
		}
	}()

	switch keyType {
	case "abci_responses":
		// Try to decode ABCI responses (ResponseFinalizeBlock in newer Tendermint)
		var resp tmABCIResponses
		if err := proto.Unmarshal(value, &resp); err == nil {
			// Validate that unmarshal actually decoded meaningful data
			if resp.DeliverTxs == nil && resp.BeginBlock == nil && resp.EndBlock == nil {
				info.ABCIResponseError = "protobuf unmarshal succeeded but all fields are nil (schema mismatch)"
			} else {
				abciResponseInfo := &ConsensusABCIResponseInfo{}

				// Count deliver_tx results (transaction results)
				if resp.DeliverTxs != nil {
					abciResponseInfo.TxResultCount = len(resp.DeliverTxs)
				}

				// Count events from end_block
				if resp.EndBlock != nil {
					abciResponseInfo.EventCount = len(resp.EndBlock.Events)
					if resp.EndBlock.ValidatorUpdates != nil {
						abciResponseInfo.ValidatorUpdates = len(resp.EndBlock.ValidatorUpdates)
					}
				}

				// Count event types from all sources
				eventSummary := &ConsensusEventSummary{
					EventTypeCounts: make(map[string]int),
				}

				// Decode transactions and collect events from DeliverTxs
				txSummary := &ConsensusTransactionSummary{
					MethodCounts: make(map[string]int),
				}

				if resp.DeliverTxs != nil {
					for _, txResult := range resp.DeliverTxs {
						// Count and decode events
						for _, event := range txResult.Events {
							eventSummary.EventTypeCounts[event.Type]++

							// Decode event if sample limit not reached
							if decodedEvents, err := decodeConsensusEvent(event); err == nil {
								for _, decodedEvent := range decodedEvents {
									if len(eventSummary.Events) < 10 {
										eventSummary.Events = append(eventSummary.Events, decodedEvent)
									}
								}
							}
						}

						// Decode transaction if present
						if len(txResult.Data) > 0 {
							if decodedTx, err := decodeConsensusTransaction(txResult.Data); err == nil {
								txSummary.MethodCounts[decodedTx.Method]++
								// Only include first few transactions for sample
								if len(txSummary.Transactions) < 10 {
									txSummary.Transactions = append(txSummary.Transactions, decodedTx)
								}
							}
						}
					}
				}

				// Events from BeginBlock
				if resp.BeginBlock != nil {
					for _, event := range resp.BeginBlock.Events {
						eventSummary.EventTypeCounts[event.Type]++

						// Decode event if sample limit not reached
						if decodedEvents, err := decodeConsensusEvent(event); err == nil {
							for _, decodedEvent := range decodedEvents {
								if len(eventSummary.Events) < 10 {
									eventSummary.Events = append(eventSummary.Events, decodedEvent)
								}
							}
						}
					}
				}

				// Events from EndBlock
				if resp.EndBlock != nil {
					for _, event := range resp.EndBlock.Events {
						eventSummary.EventTypeCounts[event.Type]++

						// Decode event if sample limit not reached
						if decodedEvents, err := decodeConsensusEvent(event); err == nil {
							for _, decodedEvent := range decodedEvents {
								if len(eventSummary.Events) < 10 {
									eventSummary.Events = append(eventSummary.Events, decodedEvent)
								}
							}
						}
					}
				}


				if len(txSummary.MethodCounts) > 0 {
					abciResponseInfo.TransactionSummary = txSummary
				}

				if len(eventSummary.EventTypeCounts) > 0 {
					abciResponseInfo.EventSummary = eventSummary
				}

				info.ABCIResponse = abciResponseInfo
			}
		} else {
			info.ABCIResponseError = fmt.Sprintf("failed to decode ABCI responses: %v", err)
		}

	case "consensus_params":
		var params tmConsensusParams
		if err := proto.Unmarshal(value, &params); err == nil {
			// Validate that unmarshal decoded meaningful data
			if params.Block == nil && params.Evidence == nil && params.Validator == nil && params.Version == nil {
				info.ConsensusParamsError = "protobuf unmarshal succeeded but all fields are nil (schema mismatch)"
			} else {
				info.ConsensusParams = &params
			}
		} else {
			info.ConsensusParamsError = fmt.Sprintf("failed to decode consensus params: %v", err)
		}

	case "validators":
		var valSet tmValidatorSet
		if err := proto.Unmarshal(value, &valSet); err == nil {
			// Validate that unmarshal decoded meaningful data
			if len(valSet.Validators) == 0 && valSet.Proposer == nil && valSet.TotalVotingPower == 0 {
				info.ValidatorsError = "protobuf unmarshal succeeded but all fields are empty/nil (schema mismatch)"
			} else {
				info.ConsensusValidators = &valSet
			}
		} else {
			info.ValidatorsError = fmt.Sprintf("failed to decode validator set: %v", err)
		}

	case "state":
		var state tmState
		if err := proto.Unmarshal(value, &state); err == nil {
			// Validate that unmarshal decoded meaningful data
			if state.ChainID == "" && state.LastBlockHeight == 0 && state.InitialHeight == 0 {
				info.StateError = "protobuf unmarshal succeeded but all fields are empty/zero (schema mismatch)"
			} else {
				info.ConsensusState = &state
			}
		} else {
			info.StateError = fmt.Sprintf("failed to decode state: %v", err)
		}

	case "genesis":
		if value[0] != '{' {
			info.StateError = "genesis document not in expected JSON format"
		}
		// Genesis is JSON, ValueRaw shows hex preview (already set above)
	}

	return info
}

// decodeConsensusTransaction decodes a raw CBOR-encoded transaction bytes into ConsensusTransactionInfo.
// Returns (decoded, error) where error is a parsing error.
// See: _oasis-core/go/consensus/api/transaction/transaction.go:42-54
// See: _oasis-core/go/common/crypto/signature/signature.go:415-421
// See: _oasis-core/go/staking/api/api.go for transaction body types
func decodeConsensusTransaction(rawTx []byte) (ConsensusTransactionInfo, error) {
	info := ConsensusTransactionInfo{}

	// Decode SignedTransaction envelope
	var signedTx cborConsensusSignedTransaction
	if err := cbor.Unmarshal(rawTx, &signedTx); err != nil {
		return info, fmt.Errorf("failed to decode SignedTransaction: %w", err)
	}

	// Extract signer public key
	info.Signer = truncateHex(signedTx.Signature.PublicKey[:], TruncateLongSize)

	// Decode inner Transaction from the blob
	var tx cborConsensusInnerTransaction
	if err := cbor.Unmarshal(signedTx.Blob, &tx); err != nil {
		return info, fmt.Errorf("failed to decode Transaction: %w", err)
	}

	info.Nonce = tx.Nonce
	info.Method = tx.Method
	if tx.Fee != nil {
		info.Fee = &ConsensusFee{
			Amount: tx.Fee.Amount,
			Gas:    tx.Fee.Gas,
		}
	}

	// Decode body based on method
	if len(tx.Body) > 0 {
		info.BodyHex = truncateHex(tx.Body, TruncateLongSize)
		info.BodySize = len(tx.Body)

		switch tx.Method {
		case "staking.Transfer":
			var transfer cborConsensusTransfer
			if err := cbor.Unmarshal(tx.Body, &transfer); err == nil {
				info.Body = transfer
			} else {
				info.BodyError = fmt.Sprintf("failed to decode transfer body: %v", err)
			}

		case "staking.Burn":
			var burn cborConsensusBurn
			if err := cbor.Unmarshal(tx.Body, &burn); err == nil {
				info.Body = burn
			} else {
				info.BodyError = fmt.Sprintf("failed to decode burn body: %v", err)
			}

		case "staking.AddEscrow":
			var escrow cborConsensusAddEscrow
			if err := cbor.Unmarshal(tx.Body, &escrow); err == nil {
				info.Body = escrow
			} else {
				info.BodyError = fmt.Sprintf("failed to decode add_escrow body: %v", err)
			}

		case "staking.ReclaimEscrow":
			var reclaim cborConsensusReclaimEscrow
			if err := cbor.Unmarshal(tx.Body, &reclaim); err == nil {
				info.Body = reclaim
			} else {
				info.BodyError = fmt.Sprintf("failed to decode reclaim_escrow body: %v", err)
			}

		case "registry.RegisterEntity":
			var regEntity cborConsensusRegisterEntity
			if err := cbor.Unmarshal(tx.Body, &regEntity); err == nil {
				info.Body = regEntity
			} else {
				info.BodyError = fmt.Sprintf("failed to decode register_entity body: %v", err)
			}

		case "registry.RegisterNode":
			var regNode cborConsensusRegisterNode
			if err := cbor.Unmarshal(tx.Body, &regNode); err == nil {
				info.Body = regNode
			} else {
				info.BodyError = fmt.Sprintf("failed to decode register_node body: %v", err)
			}

		case "roothash.ExecutorCommit":
			var execCommit cborConsensusExecutorCommit
			if err := cbor.Unmarshal(tx.Body, &execCommit); err == nil {
				info.Body = execCommit
			} else {
				info.BodyError = fmt.Sprintf("failed to decode executor_commit body: %v", err)
			}

		case "governance.SubmitProposal":
			var proposal cborConsensusSubmitProposal
			if err := cbor.Unmarshal(tx.Body, &proposal); err == nil {
				info.Body = proposal
			} else {
				info.BodyError = fmt.Sprintf("failed to decode submit_proposal body: %v", err)
			}

		case "governance.CastVote":
			var vote cborConsensusCastVote
			if err := cbor.Unmarshal(tx.Body, &vote); err == nil {
				info.Body = vote
			} else {
				info.BodyError = fmt.Sprintf("failed to decode cast_vote body: %v", err)
			}
		}
	}

	return info, nil
}

// decodeConsensusEvent decodes a Tendermint event into ConsensusEventInfo.
// Events use base64-encoded CBOR-marshaled bodies in the Value field.
// Returns all decoded attributes. On success returns (results, nil).
// See: _oasis-core/go/consensus/api/events/events.go (TypedAttribute pattern)
// See: _oasis-core/go/staking/api/api.go (event type definitions)
func decodeConsensusEvent(event tmEvent) ([]ConsensusEventInfo, error) {
	if len(event.Attributes) == 0 {
		return nil, fmt.Errorf("event has no attributes")
	}

	var results []ConsensusEventInfo

	// Process all attributes
	for _, attr := range event.Attributes {
		info := ConsensusEventInfo{
			EventType: event.Type,
			EventKind: string(attr.Key),
		}

		// Base64 decode the attribute value
		cborData, err := base64.StdEncoding.DecodeString(string(attr.Value))
		if err != nil {
			info.BodyError = fmt.Sprintf("base64 decode failed: %v", err)
			results = append(results, info)
			continue
		}

		// Populate raw body fields
		info.BodyHex = truncateHex(cborData, TruncateLongSize)
		info.BodySize = len(cborData)

		// Decode body based on event kind
		switch info.EventKind {
		case "transfer":
			var transfer cborConsensusTransferEvent
			if err := cbor.Unmarshal(cborData, &transfer); err == nil {
				info.Body = transfer
			} else {
				info.BodyError = fmt.Sprintf("cbor unmarshal failed: %v", err)
			}

		case "burn":
			var burn cborConsensusBurnEvent
			if err := cbor.Unmarshal(cborData, &burn); err == nil {
				info.Body = burn
			} else {
				info.BodyError = fmt.Sprintf("cbor unmarshal failed: %v", err)
			}

		case "add_escrow":
			var addEscrow cborConsensusAddEscrowEvent
			if err := cbor.Unmarshal(cborData, &addEscrow); err == nil {
				info.Body = addEscrow
			} else {
				info.BodyError = fmt.Sprintf("cbor unmarshal failed: %v", err)
			}

		case "reclaim_escrow":
			var reclaimEscrow cborConsensusReclaimEscrowEvent
			if err := cbor.Unmarshal(cborData, &reclaimEscrow); err == nil {
				info.Body = reclaimEscrow
			} else {
				info.BodyError = fmt.Sprintf("cbor unmarshal failed: %v", err)
			}
		}

		results = append(results, info)
	}

	return results, nil
}
