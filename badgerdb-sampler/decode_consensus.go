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
			info.Height = h
		}
		return info
	}
	if strings.HasPrefix(asciiKey, "P:") {
		info.KeyType = "block_part"
		// Format: P:{height}:{part}
		parts := strings.Split(asciiKey[2:], ":")
		if len(parts) >= 1 {
			if h, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
				info.Height = h
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
			info.Height = h
		}
		return info
	}
	if strings.HasPrefix(asciiKey, "SC:") {
		info.KeyType = "seen_commit"
		if h, err := strconv.ParseInt(asciiKey[3:], 10, 64); err == nil {
			info.Height = h
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
		var state tmstore.BlockStoreState
		if err := proto.Unmarshal(value, &state); err == nil {
			info.State = &ConsensusBlockStoreState{
				Base:   state.Base,
				Height: state.Height,
			}
		} else {
			info.DecodeError = err.Error()
		}

	case "block_meta":
		var meta tmproto.BlockMeta
		if err := proto.Unmarshal(value, &meta); err == nil {
			blockMeta := &ConsensusBlockMetaInfo{
				Height: meta.Header.Height,
				NumTxs: meta.NumTxs,
			}

			// Extract timestamp
			if meta.Header.Time.Unix() > 0 {
				info.Timestamp = meta.Header.Time.Unix()
				blockMeta.Time = meta.Header.Time.Format(time.RFC3339)
			}

			// Format AppHash
			if len(meta.Header.AppHash) >= 8 {
				blockMeta.AppHash = fmt.Sprintf("%x", meta.Header.AppHash[:8])
			} else if len(meta.Header.AppHash) > 0 {
				blockMeta.AppHash = fmt.Sprintf("%x", meta.Header.AppHash)
			}

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
		var part tmproto.Part
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
		var commit tmproto.Commit
		if err := proto.Unmarshal(value, &commit); err == nil {
			info.Commit = &ConsensusCommitInfo{
				Height:     commit.Height,
				Round:      commit.Round,
				Signatures: len(commit.Signatures),
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

	// Try to decode as DuplicateVoteEvidence
	var evidence tmproto.DuplicateVoteEvidence
	if err := proto.Unmarshal(value, &evidence); err == nil {
		if evidence.VoteA != nil {
			info.VoteAHeight = evidence.VoteA.Height
		}
		if evidence.VoteB != nil {
			info.VoteBHeight = evidence.VoteB.Height
		}
	} else {
		info.DecodeError = err.Error()
	}

	return info
}

// decodeKeyConsensusMkvs parses consensus-mkvs key and returns structured info.
// Handles both old format (no dbVersion prefix) and new format (0x01 or 0x05 dbVersion prefix).
// See: _oasis-core/go/storage/mkvs/db/badger/badger.go:31-66
func decodeKeyConsensusMkvs(key []byte) ConsensusMkvsKeyInfo {
	info := ConsensusMkvsKeyInfo{}

	if len(key) < 1 {
		info.KeyType = "unknown"
		info.DecodeError = "key too short"
		return info
	}

	// Check if key has dbVersion prefix (0x01 or 0x05)
	prefixByte := key[0]
	var data []byte

	if prefixByte == 0x01 || prefixByte == 0x05 {
		info.DbPrefix = prefixByte
		if len(key) < 2 {
			info.KeyType = "unknown"
			info.DecodeError = "key too short after prefix"
			return info
		}
		prefixByte = key[1]
		data = key[2:]
	} else {
		data = key[1:]
	}

	switch prefixByte {
	case 0x00:
		info.KeyType = "node"
		info.Hash = truncateHex(data, 16)

	case 0x01:
		info.KeyType = "write_log"
		if len(data) >= 8 {
			info.Height = binary.BigEndian.Uint64(data[0:8])
			if len(data) >= 8+33 {
				info.RootType = data[8]
				info.Hash = truncateHex(data[9:9+32], 8)
			}
		} else {
			info.DecodeError = "write_log data too short"
		}

	case 0x02:
		info.KeyType = "roots_metadata"
		if len(data) >= 8 {
			info.Height = binary.BigEndian.Uint64(data[0:8])
		} else {
			info.DecodeError = "roots_metadata data too short"
		}

	case 0x03:
		info.KeyType = "root_updated_nodes"
		if len(data) >= 8 {
			info.Height = binary.BigEndian.Uint64(data[0:8])
			if len(data) >= 8+33 {
				info.RootType = data[8]
				info.Hash = truncateHex(data[9:9+32], 8)
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
		case "consensusParamsKey":
			info.KeyType = "consensus_params"
		case "validatorsKey":
			info.KeyType = "validators"
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
	return ConsensusStateValueInfo{
		KeyType: keyType,
		Size:    len(value),
	}
}
