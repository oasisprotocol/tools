package main

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/gogo/protobuf/proto"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmstore "github.com/tendermint/tendermint/proto/tendermint/store"
)

// decodeKeyConsensusBlockstore parses consensus-blockstore key and returns key type and decoded representation
func decodeKeyConsensusBlockstore(key []byte) (string, string) {
	if len(key) < 2 {
		return "unknown", fmt.Sprintf("%x", key)
	}

	// All keys start with 0x01 (dbVersion from Oasis BadgerDB wrapper)
	if key[0] != 0x01 {
		return "unknown", fmt.Sprintf("%x", key)
	}

	// Parse ASCII key after 0x01 prefix
	asciiKey := string(key[1:])

	// blockStore state key
	if asciiKey == "blockStore" {
		return "blockstore_state", "blockStore"
	}

	// Parse structured keys: "H:{height}", "P:{height}:{part}", "C:{height}", "SC:{height}", "BH:{hash}"
	if strings.HasPrefix(asciiKey, "H:") {
		return "block_meta", asciiKey
	}
	if strings.HasPrefix(asciiKey, "P:") {
		return "block_part", asciiKey
	}
	if strings.HasPrefix(asciiKey, "C:") {
		return "block_commit", asciiKey
	}
	if strings.HasPrefix(asciiKey, "SC:") {
		return "seen_commit", asciiKey
	}
	if strings.HasPrefix(asciiKey, "BH:") {
		return "block_hash", asciiKey
	}

	return "unknown", asciiKey
}

// decodeValueConsensusBlockstore attempts to decode protobuf value based on key type
// Returns: (decoded_string, timestamp_unix)
func decodeValueConsensusBlockstore(keyType string, value []byte) (string, int64) {
	switch keyType {
	case "blockstore_state":
		var state tmstore.BlockStoreState
		if err := proto.Unmarshal(value, &state); err == nil {
			return fmt.Sprintf("BlockStoreState{base: %d, height: %d}", state.Base, state.Height), 0
		}

	case "block_meta":
		var meta tmproto.BlockMeta
		if err := proto.Unmarshal(value, &meta); err == nil {
			// Extract timestamp
			tsUnix := int64(0)
			ts := ""
			if meta.Header.Time.Unix() > 0 {
				tsUnix = meta.Header.Time.Unix()
				ts = meta.Header.Time.Format(time.RFC3339)
			}

			// Format AppHash and ChainID
			appHash := ""
			if len(meta.Header.AppHash) >= 8 {
				appHash = fmt.Sprintf("%x", meta.Header.AppHash[:8])
			} else if len(meta.Header.AppHash) > 0 {
				appHash = fmt.Sprintf("%x", meta.Header.AppHash)
			}

			chainID := meta.Header.ChainID
			if len(chainID) > 20 {
				chainID = chainID[:20] + "..."
			}

			return fmt.Sprintf("BlockMeta{height: %d, time: %s, chain: %s, num_txs: %d, app_hash: %s}",
				meta.Header.Height, ts, chainID, meta.NumTxs, appHash), tsUnix
		}

	case "block_part":
		var part tmproto.Part
		if err := proto.Unmarshal(value, &part); err == nil {
			return fmt.Sprintf("Part{index: %d, bytes_size: %d, proof_total: %d}",
				part.Index, len(part.Bytes), part.Proof.Total), 0
		}

	case "block_commit":
		var commit tmproto.Commit
		if err := proto.Unmarshal(value, &commit); err == nil {
			return fmt.Sprintf("Commit{height: %d, round: %d, signatures: %d}",
				commit.Height, commit.Round, len(commit.Signatures)), 0
		}

	case "seen_commit":
		var commit tmproto.Commit
		if err := proto.Unmarshal(value, &commit); err == nil {
			return fmt.Sprintf("Commit{height: %d, round: %d, signatures: %d}",
				commit.Height, commit.Round, len(commit.Signatures)), 0
		}

	case "block_hash":
		// Block hash values are plain strings containing height numbers
		height, err := strconv.ParseInt(string(value), 10, 64)
		if err == nil {
			return fmt.Sprintf("Hash{height: %d}", height), 0
		}

	default:
		return fmt.Sprintf("Unknown type (size: %d bytes)", len(value)), 0
	}

	return fmt.Sprintf("Failed to decode blockstore (type: %s, size: %d bytes)", keyType, len(value)), 0
}

// decodeKeyConsensusEvidence parses consensus-evidence key and returns key type and decoded representation
func decodeKeyConsensusEvidence(key []byte) (string, string) {
	if len(key) < 2 {
		return "unknown", fmt.Sprintf("%x", key)
	}

	// All keys start with 0x01 (dbVersion from Oasis BadgerDB wrapper)
	if key[0] != 0x01 {
		return "unknown", fmt.Sprintf("%x", key)
	}

	// Evidence DB is typically empty or uses simple key patterns
	return fmt.Sprintf("type_%02x", key[1]), fmt.Sprintf("%x", key[1:])
}

// decodeValueConsensusEvidence attempts to decode evidence value
func decodeValueConsensusEvidence(keyType string, value []byte) string {
	if len(value) == 0 {
		return "Empty"
	}

	// Try to decode as DuplicateVoteEvidence
	var evidence tmproto.DuplicateVoteEvidence
	if err := proto.Unmarshal(value, &evidence); err == nil {
		return fmt.Sprintf("DuplicateVoteEvidence{vote_a_height: %d, vote_b_height: %d}",
			evidence.VoteA.Height, evidence.VoteB.Height)
	}

	return fmt.Sprintf("Evidence (type: %s, size: %d bytes)", keyType, len(value))
}

// decodeKeyConsensusMkvs parses consensus-mkvs key and returns key type and decoded representation
// FIXED: Handles both old format (no dbVersion prefix) and new format (0x01 or 0x05 dbVersion prefix)
// MKVS keys from oasis-core/go/storage/mkvs/db/badger/badger.go
func decodeKeyConsensusMkvs(key []byte) (string, string) {
	if len(key) < 1 {
		return "unknown", fmt.Sprintf("%x", key)
	}

	// Check if key has dbVersion prefix (0x01 or 0x05)
	// Old format: type byte directly at key[0]
	// New format: 0x01 or 0x05 at key[0], type byte at key[1]
	prefixByte := key[0]
	var data []byte

	if prefixByte == 0x01 || prefixByte == 0x05 {
		// New format with dbVersion prefix
		if len(key) < 2 {
			return "unknown", fmt.Sprintf("%x", key)
		}
		prefixByte = key[1]
		data = key[2:]
	} else {
		// Old format without dbVersion prefix (BadgerDB v2 MKVS)
		data = key[1:]
	}

	switch prefixByte {
	case 0x00:
		// nodeKeyFmt: hash.Hash (32 bytes)
		return "node", fmt.Sprintf("node{hash: %x....}", truncateBytes(data, 8))
	case 0x01:
		// writeLogKeyFmt: uint64(version) + TypedHash(new root) + TypedHash(old root)
		if len(data) >= 8 {
			version := binary.BigEndian.Uint64(data[0:8])
			if len(data) >= 8+33 { // version + first TypedHash (33 bytes in v3+)
				rootType := data[8]
				rootHash := data[9 : 9+32]
				return "write_log", fmt.Sprintf("write_log{v:%d, new_root_type:%d, hash: %x...}", version, rootType, truncateBytes(rootHash, 4))
			}
			return "write_log", fmt.Sprintf("write_log{v:%d}", version)
		}
		return "write_log", "write_log{<malformed>}"
	case 0x02:
		// rootsMetadataKeyFmt: uint64(version)
		if len(data) >= 8 {
			version := binary.BigEndian.Uint64(data[0:8])
			return "roots_metadata", fmt.Sprintf("roots_metadata{v:%d}", version)
		}
		return "roots_metadata", "roots_metadata{<malformed>}"
	case 0x03:
		// rootUpdatedNodesKeyFmt: uint64(version) + TypedHash(root)
		if len(data) >= 8 {
			version := binary.BigEndian.Uint64(data[0:8])
			if len(data) >= 8+33 { // version + TypedHash
				rootType := data[8]
				rootHash := data[9 : 9+32]
				return "root_updated_nodes", fmt.Sprintf("root_updated_nodes{v:%d, root_type:%d, hash:%x...}", version, rootType, truncateBytes(rootHash, 4))
			}
			return "root_updated_nodes", fmt.Sprintf("root_updated_nodes{v:%d}", version)
		}
		return "root_updated_nodes", "root_updated_nodes{<malformed>}"
	case 0x04:
		// metadataKeyFmt: no additional data
		return "metadata", "metadata"
	case 0x05:
		// multipartRestoreNodeLogKeyFmt: TypedHash (33 bytes)
		if len(data) >= 33 {
			rootType := data[0]
			rootHash := data[1:33]
			return "multipart_restore_log", fmt.Sprintf("multipart_restore_log{type:%d, hash:%x}", rootType, truncateBytes(rootHash, 8))
		}
		return "multipart_restore_log", fmt.Sprintf("multipart_restore_log{hash:%x}", truncateBytes(data, 8))
	case 0x06:
		// rootNodeKeyFmt: TypedHash (33 bytes)
		if len(data) >= 33 {
			rootType := data[0]
			rootHash := data[1:33]
			return "root_node", fmt.Sprintf("root_node{type:%d, hash:%x}", rootType, truncateBytes(rootHash, 8))
		}
		return "root_node", fmt.Sprintf("root_node{hash:%x}", truncateBytes(data, 8))
	default:
		return fmt.Sprintf("unknown_%02x", prefixByte), fmt.Sprintf("unknown_%02x:%x", prefixByte, truncateBytes(data, 8))
	}
}

// decodeValueConsensusMkvs decodes consensus MKVS value with inline node parsing
func decodeValueConsensusMkvs(keyType string, value []byte) string {
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

		// Decode consensus module key prefix (roothash, staking, registry, etc.)
		// Consensus modules use single-byte prefixes from oasis-core/go/consensus/tendermint/apps/*/state/state.go
		var module string
		if len(key) == 0 {
			module = "<empty>"
		} else {
			switch key[0] {
			// Roothash module (0x20-0x29)
			case 0x20:
				module = "roothash/runtime_state"
			case 0x21:
				module = "roothash/params"
			case 0x22:
				module = "roothash/round_timeout"
			case 0x24:
				module = "roothash/evidence"
			case 0x25:
				module = "roothash/state_root"
			case 0x26:
				module = "roothash/io_root"
			case 0x27:
				module = "roothash/last_round_results"
			case 0x28:
				module = "roothash/incoming_msg_queue_meta"
			case 0x29:
				module = "roothash/incoming_msg_queue"
			// Staking module (0x30-0x3F)
			case 0x30:
				module = "staking/total_supply"
			case 0x31:
				module = "staking/common_pool"
			case 0x32:
				module = "staking/last_block_fees"
			case 0x33:
				module = "staking/governance_deposits"
			case 0x34:
				module = "staking/accounts"
			case 0x35:
				module = "staking/delegations"
			case 0x36:
				module = "staking/debonding_delegations"
			case 0x37:
				module = "staking/allowances"
			case 0x38:
				module = "staking/params"
			// Registry module (0x40-0x4F)
			case 0x40:
				module = "registry/entities"
			case 0x41:
				module = "registry/nodes"
			case 0x42:
				module = "registry/node_by_consensus"
			case 0x43:
				module = "registry/runtimes"
			case 0x44:
				module = "registry/suspended_runtimes"
			case 0x45:
				module = "registry/params"
			case 0x46:
				module = "registry/node_status"
			// Scheduler module (0x50-0x5F)
			case 0x50:
				module = "scheduler/params"
			case 0x51:
				module = "scheduler/committees"
			case 0x52:
				module = "scheduler/validators"
			// Governance module (0x60-0x6F)
			case 0x60:
				module = "governance/params"
			case 0x61:
				module = "governance/proposals"
			case 0x62:
				module = "governance/active_proposals"
			case 0x63:
				module = "governance/votes"
			case 0x64:
				module = "governance/pending_upgrades"
			// Beacon module (0x70-0x7F)
			case 0x70:
				module = "beacon/params"
			case 0x71:
				module = "beacon/future_epoch"
			case 0x72:
				module = "beacon/epoch"
			case 0x73:
				module = "beacon/pvss_state"
			// Keymanager module (0x80-0x8F)
			case 0x80:
				module = "keymanager/status"
			case 0x81:
				module = "keymanager/params"
			// Consensus parameters
			case 0xF1:
				module = "consensus/params"
			default:
				if key[0] >= 'a' && key[0] <= 'z' {
					module = extractModuleName(key)
				} else {
					module = fmt.Sprintf("0x%02x", key[0])
				}
			}
		}

		if len(data) < 4 {
			return fmt.Sprintf("LeafNode{module=%s, <no value>}", module)
		}

		valueLen := int(binary.LittleEndian.Uint32(data[0:4]))
		data = data[4:]
		if len(data) < valueLen {
			return fmt.Sprintf("LeafNode{module=%s, valueLen=%d, <truncated>}", module, valueLen)
		}

		leafValue := data[:valueLen]

		// Try CBOR decode
		var decoded interface{}
		if err := cbor.Unmarshal(leafValue, &decoded); err == nil {
			return fmt.Sprintf("LeafNode{module=%s, value=%s}", module, formatCBOR(decoded, valueLen))
		}

		return fmt.Sprintf("LeafNode{module=%s, value=binary(%d bytes)}", module, valueLen)

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

// decodeKeyConsensusState parses consensus-state key and returns key type and decoded representation
func decodeKeyConsensusState(key []byte) (string, string) {
	if len(key) < 2 {
		return "unknown", fmt.Sprintf("%x", key)
	}

	// All keys start with 0x01 (dbVersion from Oasis BadgerDB wrapper)
	if key[0] != 0x01 {
		return "unknown", fmt.Sprintf("%x", key)
	}

	// Parse ASCII key after 0x01 prefix
	decoded := string(key[1:])

	// Check if it's printable ASCII
	if !isPrintableASCII(decoded) {
		return "binary", fmt.Sprintf("%x", key)
	}

	// Extract key type prefix before colon
	colonIdx := strings.IndexByte(decoded, ':')
	if colonIdx != -1 {
		prefix := decoded[:colonIdx]
		switch prefix {
		case "abciResponsesKey":
			return "abci_responses", decoded
		case "consensusParamsKey":
			return "consensus_params", decoded
		case "validatorsKey":
			return "validators", decoded
		case "stateKey":
			return "state", decoded
		case "genesisDoc":
			return "genesis", decoded
		default:
			return prefix, decoded
		}
	}

	return "text_key", decoded
}

// decodeValueConsensusState decodes state value
func decodeValueConsensusState(keyType string, value []byte) string {
	if len(value) == 0 {
		return "Empty"
	}
	return fmt.Sprintf("Tendermint %s data (size: %d bytes)", keyType, len(value))
}
