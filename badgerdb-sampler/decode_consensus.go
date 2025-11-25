package main

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/gogo/protobuf/proto"
)

// decodeKeyConsensusBlockstore parses consensus-blockstore key and returns structured info.
// See: tendermint/store/store.go for key formats (H:, P:, C:, SC:, BH:)
func decodeKeyConsensusBlockstore(key []byte) ConsensusBlockstoreKeyInfo {
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

	// Parse ASCII key after 0x01 prefix
	asciiKey := string(key[1:])
	info.RawKey = asciiKey

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

// decodeValueConsensusBlockstore decodes protobuf value and returns structured info.
// See: tendermint/proto/tendermint/store and tendermint/proto/tendermint/types
func decodeValueConsensusBlockstore(keyType string, value []byte) ConsensusBlockstoreValueInfo {
	info := ConsensusBlockstoreValueInfo{
		KeyType: keyType,
		Size:    len(value),
	}

	switch keyType {
	case "blockstore_state":
		var state tmBlockStoreState
		if err := proto.Unmarshal(value, &state); err == nil {
			info.State = &ConsensusBlockStoreState{
				Base:            state.Base,
				ConsensusHeight: state.Height,
			}
		} else {
			info.DecodeError = err.Error()
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

			// Format AppHash (truncate to 16 bytes = 32 hex chars)
			blockMeta.AppHash = truncateHex(meta.Header.AppHash, 32)

			// Truncate ChainID
			chainID := meta.Header.ChainID
			if len(chainID) > 20 {
				chainID = chainID[:20] + "..."
			}
			blockMeta.ChainID = chainID

			info.BlockMeta = blockMeta
		} else {
			info.DecodeError = err.Error()
		}

	case "block_part":
		var part tmPart
		if err := proto.Unmarshal(value, &part); err == nil {
			info.Part = &ConsensusPartInfo{
				Index:      part.Index,
				BytesSize:  len(part.Bytes),
				ProofTotal: part.Proof.Total,
			}
		} else {
			info.DecodeError = err.Error()
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
			info.DecodeError = err.Error()
		}

	case "block_hash":
		// Block hash values are plain strings containing height numbers
		if height, err := strconv.ParseInt(string(value), 10, 64); err == nil {
			info.HashHeight = height
		} else {
			info.DecodeError = err.Error()
		}

	default:
		info.DecodeError = fmt.Sprintf("unknown key type: %s", keyType)
	}

	return info
}

// decodeKeyConsensusEvidence parses consensus-evidence key and returns structured info.
func decodeKeyConsensusEvidence(key []byte) ConsensusEvidenceKeyInfo {
	info := ConsensusEvidenceKeyInfo{}

	if len(key) < 2 {
		info.KeyType = "unknown"
		info.RawKey = fmt.Sprintf("%x", key)
		return info
	}

	// All keys start with 0x01 (dbVersion from Oasis BadgerDB wrapper)
	if key[0] != 0x01 {
		info.KeyType = "unknown"
		info.RawKey = fmt.Sprintf("%x", key)
		return info
	}

	// Evidence DB is typically empty or uses simple key patterns
	info.KeyType = fmt.Sprintf("type_%02x", key[1])
	info.PrefixByte = key[1]
	info.RawKey = fmt.Sprintf("%x", key[1:])
	return info
}

// decodeValueConsensusEvidence decodes evidence value and returns structured info.
func decodeValueConsensusEvidence(keyType string, value []byte) ConsensusEvidenceValueInfo {
	info := ConsensusEvidenceValueInfo{
		Size: len(value),
	}

	if len(value) == 0 {
		return info
	}

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
		return info
	}

	// Try to decode as LightClientAttackEvidence
	var lca tmLightClientAttackEvidence
	if err := proto.Unmarshal(value, &lca); err == nil && lca.ConflictingBlock != nil {
		info.EvidenceType = "light_client_attack"
		info.TotalVotingPower = lca.TotalVotingPower
		if lca.Timestamp.Unix() > 0 {
			info.Timestamp = lca.Timestamp.Format(time.RFC3339)
		}
		return info
	}

	// If all parsing failed, show raw value
	info.EvidenceType = "unknown"
	info.RawValue = truncateHex(value, 64)
	info.DecodeError = "unable to decode evidence format"

	return info
}

// decodeKeyConsensusMkvs parses consensus-mkvs key and returns structured info.
// Keys use keyformat encoding: [type_byte][data...]
// See: _oasis-core/go/storage/mkvs/db/badger/badger.go:31-66
func decodeKeyConsensusMkvs(key []byte) ConsensusMkvsKeyInfo {
	info := ConsensusMkvsKeyInfo{}

	if len(key) < 1 {
		info.KeyType = "unknown"
		info.DecodeError = "key too short"
		return info
	}

	// First byte is the key type prefix
	prefixByte := key[0]
	data := key[1:]

	switch prefixByte {
	case 0x00:
		info.KeyType = "node"
		info.Hash = truncateHex(data, 16)

	case 0x01:
		info.KeyType = "write_log"
		if len(data) >= 8 {
			info.ConsensusHeight = binary.BigEndian.Uint64(data[0:8])
			if len(data) >= 8+33 {
				info.RootType = data[8]
				info.Hash = truncateHex(data[9:9+32], 32)
			}
		} else {
			info.DecodeError = "write_log data too short"
		}

	case 0x02:
		info.KeyType = "roots_metadata"
		if len(data) >= 8 {
			info.ConsensusHeight = binary.BigEndian.Uint64(data[0:8])
		} else {
			info.DecodeError = "roots_metadata data too short"
		}

	case 0x03:
		info.KeyType = "root_updated_nodes"
		if len(data) >= 8 {
			info.ConsensusHeight = binary.BigEndian.Uint64(data[0:8])
			if len(data) >= 8+33 {
				info.RootType = data[8]
				info.Hash = truncateHex(data[9:9+32], 32)
			}
		} else {
			info.DecodeError = "root_updated_nodes data too short"
		}

	case 0x04:
		info.KeyType = "metadata"

	case 0x05:
		info.KeyType = "multipart_restore_log"
		if len(data) >= 33 {
			info.RootType = data[0]
			info.Hash = truncateHex(data[1:33], 16)
		} else {
			info.Hash = truncateHex(data, 16)
		}

	case 0x06:
		info.KeyType = "root_node"
		if len(data) >= 33 {
			info.RootType = data[0]
			info.Hash = truncateHex(data[1:33], 16)
		} else {
			info.Hash = truncateHex(data, 16)
		}

	default:
		info.KeyType = fmt.Sprintf("unknown_%02x", prefixByte)
		info.DecodeError = fmt.Sprintf("unknown key prefix 0x%02x", prefixByte)
	}

	return info
}

// decodeValueConsensusMkvs decodes consensus MKVS value and returns structured info.
// See: _oasis-core/go/storage/mkvs/node/node.go:26-32 (prefixes), 294-309 (InternalNode), 531-537 (LeafNode)
func decodeValueConsensusMkvs(keyType string, value []byte) ConsensusMkvsNodeInfo {
	info := ConsensusMkvsNodeInfo{Size: len(value)}

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
	case 0x00: // LeafNode
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

		// Decode consensus module key prefix
		module := decodeConsensusModulePrefix(key)

		leaf := &ConsensusMkvsLeafInfo{
			Module: module,
			KeyLen: keyLen,
			Key:    truncateHex(key, 32),
		}

		// Extract Oasis address for staking-related modules
		// Staking keys have format: module_prefix (1 byte) + 21-byte Oasis address
		if keyLen == 22 {
			switch key[0] {
			case 0x34, 0x35, 0x36, 0x37: // staking accounts, delegations, debonding, allowances
				leaf.OasisAddress = bech32Encode("oasis", key[1:22])
			}
		}
		// Entity/node keys also contain addresses
		if keyLen >= 22 {
			switch key[0] {
			case 0x40, 0x41: // registry entities, nodes
				leaf.OasisAddress = bech32Encode("oasis", key[1:22])
			}
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

		// Try CBOR decode
		var decoded interface{}
		if err := cbor.Unmarshal(leafValue, &decoded); err == nil {
			leaf.DecodedValue = formatCBOR(decoded, valueLen)
		} else {
			leaf.DecodedValue = fmt.Sprintf("binary(%d bytes)", valueLen)
		}

		info.Leaf = leaf
		return info

	case 0x01: // InternalNode
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
			info.Internal = &ConsensusMkvsInternalInfo{LabelBits: labelBits}
			return info
		}

		data = data[labelBytes:] // skip label

		internal := &ConsensusMkvsInternalInfo{LabelBits: labelBits}
		hasLeaf := data[0] == 0x00
		internal.HasLeaf = hasLeaf

		if !hasLeaf {
			data = data[1:] // skip nil marker
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

// decodeKeyConsensusState parses consensus-state key and returns structured info.
func decodeKeyConsensusState(key []byte) ConsensusStateKeyInfo {
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

	// Parse ASCII key after 0x01 prefix
	decoded := string(key[1:])

	// Check if it's printable ASCII
	if !isPrintableASCII(decoded) {
		info.KeyType = "binary"
		info.IsBinary = true
		return info
	}

	info.DecodedKey = decoded

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

// decodeValueConsensusState decodes state value and returns structured info.
func decodeValueConsensusState(keyType string, value []byte) ConsensusStateValueInfo {
	info := ConsensusStateValueInfo{
		KeyType: keyType,
		Size:    len(value),
	}

	if len(value) == 0 {
		return info
	}

	switch keyType {
	case "abci_responses":
		// Try to decode ABCI responses (ResponseFinalizeBlock in newer Tendermint)
		var resp tmABCIResponses
		if err := proto.Unmarshal(value, &resp); err == nil {
			abciInfo := &ABCIResponseInfo{}

			// Count deliver_tx results (transaction results)
			if resp.DeliverTxs != nil {
				abciInfo.TxResultCount = len(resp.DeliverTxs)
			}

			// Count events from end_block
			if resp.EndBlock != nil {
				abciInfo.EventCount = len(resp.EndBlock.Events)
				if resp.EndBlock.ValidatorUpdates != nil {
					abciInfo.ValidatorUpdates = len(resp.EndBlock.ValidatorUpdates)
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
					// Count events
					for _, event := range txResult.Events {
						eventSummary.EventTypeCounts[event.Type]++
					}

					// Decode transaction if present
					if len(txResult.Data) > 0 {
						decodedTx := decodeCBORTransaction(txResult.Data)
						if decodedTx.DecodeError == "" {
							txSummary.MethodCounts[decodedTx.Method]++
							// Only include first few transactions for sample
							if len(txSummary.Transactions) < 5 {
								txSummary.Transactions = append(txSummary.Transactions, decodedTx)
							}
						}
					}
				}
			}

			if len(txSummary.MethodCounts) > 0 {
				abciInfo.TransactionSummary = txSummary
			}

			// Events from BeginBlock
			if resp.BeginBlock != nil {
				for _, event := range resp.BeginBlock.Events {
					eventSummary.EventTypeCounts[event.Type]++
				}
			}

			// Events from EndBlock
			if resp.EndBlock != nil {
				for _, event := range resp.EndBlock.Events {
					eventSummary.EventTypeCounts[event.Type]++
				}
			}

			if len(eventSummary.EventTypeCounts) > 0 {
				abciInfo.EventSummary = eventSummary
			}

			info.ABCIInfo = abciInfo
		} else {
			info.DecodeError = fmt.Sprintf("failed to decode ABCI responses: %v", err)
		}

	case "consensus_params":
		info.RawValue = truncateHex(value, 32)
		var params tmConsensusParams
		if err := proto.Unmarshal(value, &params); err == nil {
			info.ConsensusParams = &params
		} else {
			info.DecodeError = fmt.Sprintf("failed to decode consensus params: %v", err)
		}

	case "validators":
		info.RawValue = truncateHex(value, 32)
		var valSet tmValidatorSet
		if err := proto.Unmarshal(value, &valSet); err == nil {
			info.ConsensusValidators = &valSet
		} else {
			info.DecodeError = fmt.Sprintf("failed to decode validator set: %v", err)
		}

	case "state":
		info.RawValue = truncateHex(value, 32)
		var state tmState
		if err := proto.Unmarshal(value, &state); err == nil {
			info.ConsensusState = &state
		} else {
			info.DecodeError = fmt.Sprintf("failed to decode state: %v", err)
		}

	case "genesis":
		info.RawValue = truncateHex(value, 64)
		if value[0] != '{' {
			info.DecodeError = "genesis document not in expected JSON format"
		}
		// Genesis is JSON, RawValue shows hex preview

	default:
		info.RawValue = truncateHex(value, 32)
	}

	return info
}

// decodeCBORTransaction decodes a raw CBOR-encoded transaction bytes into ConsensusDecodedTransaction.
// See: _oasis-core/go/consensus/api/transaction/transaction.go:42-54
// See: _oasis-core/go/common/crypto/signature/signature.go:415-421
// See: _oasis-core/go/staking/api/api.go for transaction body types
func decodeCBORTransaction(rawTx []byte) ConsensusDecodedTransaction {
	decoded := ConsensusDecodedTransaction{}

	// Decode SignedTransaction envelope
	var signedTx ConsensusSignedTransaction
	if err := cbor.Unmarshal(rawTx, &signedTx); err != nil {
		decoded.DecodeError = fmt.Sprintf("failed to decode SignedTransaction: %v", err)
		return decoded
	}

	// Extract signer public key
	decoded.Signer = fmt.Sprintf("%x", signedTx.Signature.PublicKey[:])

	// Decode inner Transaction from the blob
	var tx struct {
		Nonce  uint64          `cbor:"nonce"`
		Fee    *ConsensusFee   `cbor:"fee"`
		Method string          `cbor:"method"`
		Body   cbor.RawMessage `cbor:"body"`
	}
	if err := cbor.Unmarshal(signedTx.Blob, &tx); err != nil {
		decoded.DecodeError = fmt.Sprintf("failed to decode Transaction: %v", err)
		return decoded
	}

	decoded.Nonce = tx.Nonce
	decoded.Method = tx.Method
	decoded.Fee = tx.Fee

	// Decode body based on method
	if len(tx.Body) > 0 {
		decoded.BodyPreview = truncateHex(tx.Body, 32)

		switch tx.Method {
		case "staking.Transfer":
			var transfer ConsensusTransfer
			if err := cbor.Unmarshal(tx.Body, &transfer); err == nil {
				transfer.Amount = quantityBytesToString(tx.Body, "amount")
				decoded.DecodedBody = transfer
			}

		case "staking.Burn":
			var burn ConsensusBurn
			if err := cbor.Unmarshal(tx.Body, &burn); err == nil {
				burn.Amount = quantityBytesToString(tx.Body, "amount")
				decoded.DecodedBody = burn
			}

		case "staking.AddEscrow":
			var escrow ConsensusAddEscrow
			if err := cbor.Unmarshal(tx.Body, &escrow); err == nil {
				escrow.Amount = quantityBytesToString(tx.Body, "amount")
				decoded.DecodedBody = escrow
			}

		case "staking.ReclaimEscrow":
			var reclaim ConsensusReclaimEscrow
			if err := cbor.Unmarshal(tx.Body, &reclaim); err == nil {
				reclaim.Shares = quantityBytesToString(tx.Body, "shares")
				decoded.DecodedBody = reclaim
			}

		case "registry.RegisterEntity":
			var regEntity ConsensusRegisterEntity
			if err := cbor.Unmarshal(tx.Body, &regEntity); err == nil {
				decoded.DecodedBody = regEntity
			}

		case "registry.RegisterNode":
			var regNode ConsensusRegisterNode
			if err := cbor.Unmarshal(tx.Body, &regNode); err == nil {
				decoded.DecodedBody = regNode
			}

		case "roothash.ExecutorCommit":
			var execCommit ConsensusExecutorCommit
			if err := cbor.Unmarshal(tx.Body, &execCommit); err == nil {
				decoded.DecodedBody = execCommit
			}

		case "governance.SubmitProposal":
			var proposal ConsensusSubmitProposal
			if err := cbor.Unmarshal(tx.Body, &proposal); err == nil {
				proposal.Deposit = quantityBytesToString(tx.Body, "deposit")
				decoded.DecodedBody = proposal
			}

		case "governance.CastVote":
			var vote ConsensusCastVote
			if err := cbor.Unmarshal(tx.Body, &vote); err == nil {
				decoded.DecodedBody = vote
			}
		}
	}

	return decoded
}
