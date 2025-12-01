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
func decodeConsensusBlockstoreKey(key []byte) *ConsensusBlockstoreKeyInfo {
	info := &ConsensusBlockstoreKeyInfo{
		KeyDump: formatRawValue(key, TruncateLongLen),
		KeySize: len(key),
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

	// Parse ASCII key after 0x01 prefix
	asciiKey := string(key[1:])

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
func decodeConsensusBlockstoreValue(keyType string, value []byte) *ConsensusBlockstoreValueInfo {
	info := &ConsensusBlockstoreValueInfo{
		RawDump: formatRawValue(value, TruncateLongLen),
		RawSize: len(value),
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
			info.RawError = err.Error()
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
			blockMeta.AppHash = formatRawValue(meta.Header.AppHash, TruncateHashLen)

			// Truncate ChainID
			chainID := meta.Header.ChainID
			if len(chainID) > 20 {
				chainID = chainID[:20] + "..."
			}
			blockMeta.ChainID = chainID

			info.BlockMeta = blockMeta
		} else {
			info.RawError = err.Error()
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
			info.RawError = err.Error()
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
			info.RawError = err.Error()
		}

	case "block_hash":
		// Block hash values are plain strings containing height numbers
		if height, err := strconv.ParseInt(string(value), 10, 64); err == nil {
			info.HashHeight = height
		}
	}

	return info
}

// decodeConsensusEvidenceKey parses consensus-evidence key and returns structured info.
// Key format: 0x01 (dbVersion) + 0x00/0x01 (committed/pending) + "HEIGHT_HEX/HASH_HEX"
// See: cometbft/evidence/pool.go (keyCommitted, keyPending functions)
func decodeConsensusEvidenceKey(key []byte) *ConsensusEvidenceKeyInfo {
	info := &ConsensusEvidenceKeyInfo{
		KeyDump: formatRawValue(key, TruncateLongLen),
		KeySize: len(key),
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

	// Second byte is the evidence state prefix
	prefixByte := key[1]
	info.PrefixByte = prefixByte

	switch prefixByte {
	case 0x00:
		info.KeyType = "committed"
	case 0x01:
		info.KeyType = "pending"
	default:
		info.KeyType = fmt.Sprintf("type_%02x", prefixByte)
		return info
	}

	// Parse key suffix: "HEIGHT_HEX/HASH_HEX"
	if len(key) > 2 {
		keySuffix := string(key[2:])
		parts := strings.Split(keySuffix, "/")
		if len(parts) == 2 {
			// Parse height from hex string (big-endian padded)
			if h, err := strconv.ParseInt(parts[0], 16, 64); err == nil {
				info.ConsensusHeight = h
			}
			// Store evidence hash
			info.Hash = parts[1]
		}
	}

	return info
}

// decodeConsensusEvidenceValue decodes evidence value and returns structured info.
// Committed evidence stores only Int64Value (height), pending evidence stores full Evidence protobuf.
// See: cometbft/evidence/pool.go (addPendingEvidence, markEvidenceAsCommitted)
func decodeConsensusEvidenceValue(keyType string, value []byte) (info *ConsensusEvidenceValueInfo) {
	info = &ConsensusEvidenceValueInfo{
		RawDump: formatRawValue(value, TruncateLongLen),
		RawSize: len(value),
	}

	if len(value) == 0 {
		return info
	}

	// Recover from protobuf panics that occur with schema mismatches
	defer func() {
		if r := recover(); r != nil {
			info.RawError = fmt.Sprintf("protobuf panic: %v", r)
		}
	}()

	// Committed evidence only stores the block height as Int64Value
	if keyType == "committed" {
		var heightValue tmInt64Value
		if err := proto.Unmarshal(value, &heightValue); err == nil {
			info.EvidenceType = "committed_marker"
			info.CommittedHeight = heightValue.Value
			return info
		} else {
			info.RawError = fmt.Sprintf("failed to decode committed evidence Int64Value: %v", err)
			info.EvidenceType = "unknown"
			return info
		}
	}

	// Pending evidence stores full evidence protobuf - try CometBFT DuplicateVoteEvidence first
	cbSuccess := false
	var cbDve cbDuplicateVoteEvidence
	func() {
		defer func() {
			recover() // Silently catch panics from CometBFT unmarshal
		}()
		if err := proto.Unmarshal(value, &cbDve); err == nil && cbDve.VoteA != nil && cbDve.VoteB != nil {
			cbSuccess = true
		}
	}()

	if cbSuccess {
		info.SchemaVersion = "cb-v0.37"
		info.EvidenceType = "duplicate_vote"
		info.VoteAHeight = cbDve.VoteA.Height
		info.VoteBHeight = cbDve.VoteB.Height
		info.TotalVotingPower = cbDve.TotalVotingPower
		info.ValidatorPower = cbDve.ValidatorPower
		if cbDve.Timestamp.Unix() > 0 {
			info.Timestamp = cbDve.Timestamp.Format(time.RFC3339)
		}
		return info
	}

	// Fall back to Tendermint v0.34 DuplicateVoteEvidence
	var dve tmDuplicateVoteEvidence
	if err := proto.Unmarshal(value, &dve); err == nil && dve.VoteA != nil {
		info.SchemaVersion = "tm-v0.34"
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

	// Try to decode as LightClientAttackEvidence (same for both versions)
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
	if info.RawError == "" {
		info.RawError = "failed to decode as any known evidence type"
	}
	return info
}

// decodeConsensusMkvsKey parses consensus-mkvs key and returns structured info.
// Keys use keyformat encoding: [type_byte][data...]
// See: _oasis-core/go/storage/mkvs/db/badger/badger.go:31-66
func decodeConsensusMkvsKey(key []byte) *ConsensusMkvsKeyInfo {
	info := &ConsensusMkvsKeyInfo{
		KeyDump: formatRawValue(key, TruncateLongLen),
		KeySize: len(key),
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
		info.Hash = formatRawValue(data, TruncateHashLen)

	case 0x01:
		info.KeyType = "write_log"
		if len(data) >= 8 {
			info.ConsensusHeight = int64(binary.BigEndian.Uint64(data[0:8]))
			if len(data) >= 8+33 {
				info.RootType = fmt.Sprintf("%d", data[8])
				info.Hash = formatRawValue(data[9:9+32], TruncateHashLen)
			}
		}

	case 0x02:
		info.KeyType = "roots_metadata"
		if len(data) >= 8 {
			info.ConsensusHeight = int64(binary.BigEndian.Uint64(data[0:8]))
		}

	case 0x03:
		info.KeyType = "root_updated_nodes"
		if len(data) >= 8 {
			info.ConsensusHeight = int64(binary.BigEndian.Uint64(data[0:8]))
			if len(data) >= 8+33 {
				info.RootType = fmt.Sprintf("%d", data[8])
				info.Hash = formatRawValue(data[9:9+32], TruncateHashLen)
			}
		}

	case 0x04:
		info.KeyType = "metadata"

	case 0x05:
		info.KeyType = "multipart_restore_log"
		if len(data) >= 33 {
			info.RootType = fmt.Sprintf("%d", data[0])
			info.Hash = formatRawValue(data[1:33], TruncateHashLen)
		} else {
			info.Hash = formatRawValue(data, TruncateHashLen)
		}

	case 0x06:
		info.KeyType = "root_node"
		if len(data) >= 33 {
			info.RootType = fmt.Sprintf("%d", data[0])
			info.Hash = formatRawValue(data[1:33], TruncateHashLen)
		} else {
			info.Hash = formatRawValue(data, TruncateHashLen)
		}

	default:
		info.KeyType = fmt.Sprintf("unknown_%02x", prefixByte)
	}

	return info
}

// decodeConsensusMkvsValue decodes consensus MKVS value and returns structured info.
// See: _oasis-core/go/storage/mkvs/node/node.go:26-32 (prefixes), 294-309 (InternalNode), 531-537 (LeafNode)
func decodeConsensusMkvsValue(keyType string, value []byte) *ConsensusMkvsValueInfo {
	info := &ConsensusMkvsValueInfo{
		RawDump: formatRawValue(value, TruncateLongLen),
		RawSize: len(value),
	}

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

		// Try pre-v21.1.0 format (skip 8-byte Version field)
		if len(data) >= 10 {
			testKeySize := int(binary.LittleEndian.Uint16(data[8:10]))
			if testKeySize > 0 && testKeySize <= 10000 && len(data) >= 10+testKeySize+4 {
				data = data[8:]
			}
		}

		if len(data) < 2 {
			info.RawError = "key length missing"
			return info
		}

		keySize := int(binary.LittleEndian.Uint16(data[0:2]))
		data = data[2:]

		if len(data) < keySize {
			info.RawError = fmt.Sprintf("key truncated")
			return info
		}

		key := data[:keySize]
		data = data[keySize:]

		// Decode consensus module key prefix
		module := decodeConsensusModulePrefix(key)

		leaf := &ConsensusMkvsLeafInfo{
			KeyDump: formatRawValue(key, TruncateLongLen),
			KeySize: keySize,
			Module:  module,
		}

		// Extract Oasis address for staking-related modules
		// Staking keys have format: module_prefix (1 byte) + 21-byte Oasis address
		if keySize == 22 {
			switch key[0] {
			case 0x50, 0x53, 0x54: // staking accounts, delegations, debonding_delegations
				leaf.OasisAddress = (*(*OasisAddress)(key[1:22])).String()
			}
		}
		// Entity/node keys also contain addresses
		if keySize >= 22 {
			switch key[0] {
			case 0x10, 0x11: // registry entities, nodes
				leaf.OasisAddress = (*(*OasisAddress)(key[1:22])).String()
			}
		}

		if len(data) < 4 {
			info.RawError = "value length missing"
			info.Leaf = leaf
			return info
		}

		valueSize := int(binary.LittleEndian.Uint32(data[0:4]))
		data = data[4:]
		leaf.ValueSize = valueSize

		if len(data) < valueSize {
			info.RawError = fmt.Sprintf("value truncated")
			info.Leaf = leaf
			return info
		}

		leafValue := data[:valueSize]
		leaf.ValueDump = formatRawValue(leafValue, TruncateLongLen)

		// Deterministic format lookup based on key prefix
		format, exists := GetConsensusMKVSFormat(key)
		if !exists {
			// Unknown key prefix - try CBOR as fallback
			var decoded interface{}
			if err := cbor.Unmarshal(leafValue, &decoded); err == nil {
				leaf.Value = formatCBORDetailed(decoded)
				leaf.ValueType = "cbor"
			} else {
				leaf.ValueError = fmt.Sprintf("unknown key prefix 0x%02x: %s", key[0], err.Error())
				leaf.ValueType = "unknown"
			}
		} else {
			// Known format - decode deterministically
			switch format.Format {
			case "cbor":
				var decoded interface{}
				if err := cbor.Unmarshal(leafValue, &decoded); err == nil {
					leaf.Value = formatCBORDetailed(decoded)
					leaf.ValueType = "cbor"
					leaf.CBOR = format.Type // Store expected type
				} else {
					leaf.ValueError = err.Error()
					leaf.ValueType = "cbor_error"
				}

			case "binary":
				// Raw binary data
				leaf.ValueType = "raw_binary"
				switch len(leafValue) {
				case 32:
					leaf.CBOR = fmt.Sprintf("%s (32-byte hash)", format.Description)
				case 64:
					leaf.CBOR = fmt.Sprintf("%s (64-byte signature)", format.Description)
				case 65:
					leaf.CBOR = fmt.Sprintf("%s (65-byte pubkey)", format.Description)
				default:
					leaf.CBOR = fmt.Sprintf("%s (%d bytes)", format.Description, len(leafValue))
				}

			case "empty":
				// Index entry with empty value
				leaf.ValueType = "empty"
				leaf.CBOR = fmt.Sprintf("index entry: %s", format.Description)
			}
		}

		info.Leaf = leaf
		return info

	case 0x01: // InternalNode
		info.NodeType = "internal"
		data := value[1:]

		// Try pre-v21.1.0 format (skip 8-byte Version field)
		if len(data) >= 10 {
			testLabelBits := binary.LittleEndian.Uint16(data[8:10])
			if testLabelBits <= 2048 && len(data) >= 10+(int(testLabelBits)+7)/8+1 {
				data = data[8:]
			}
		}

		if len(data) < 2 {
			info.RawError = "label bits missing"
			return info
		}

		labelBits := binary.LittleEndian.Uint16(data[0:2])
		data = data[2:]

		labelBytes := (int(labelBits) + 7) / 8
		if len(data) < labelBytes+1 {
			info.RawError = "label truncated"
			info.Internal = &ConsensusMkvsInternalInfo{LabelBits: labelBits}
			return info
		}

		data = data[labelBytes:] // skip label

		internal := &ConsensusMkvsInternalInfo{LabelBits: labelBits}

		// Check for embedded leaf node or nil marker
		if len(data) < 1 {
			info.RawError = "missing leaf/nil marker"
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
				info.RawError = "embedded leaf key length missing"
				info.Internal = internal
				return info
			}
			keyLen := binary.LittleEndian.Uint16(data[0:2])
			data = data[2:]
			if len(data) < int(keyLen)+4 {
				info.RawError = "embedded leaf truncated"
				info.Internal = internal
				return info
			}
			data = data[keyLen:] // skip key
			valueLen := binary.LittleEndian.Uint32(data[0:4])
			data = data[4:]
			if len(data) < int(valueLen) {
				info.RawError = "embedded leaf value truncated"
				info.Internal = internal
				return info
			}
			data = data[valueLen:] // skip value
		} else {
			info.RawError = fmt.Sprintf("unexpected marker 0x%02x", data[0])
			info.Internal = internal
			return info
		}

		// Read left and right hashes
		if len(data) >= 32 {
			internal.LeftHash = formatRawValue(data[:32], TruncateHashLen)
			data = data[32:]
		}
		if len(data) >= 32 {
			internal.RightHash = formatRawValue(data[:32], TruncateHashLen)
		}

		info.Internal = internal
		return info

	case 0x02: // NilNode
		info.NodeType = "nil"
		return info

	default:
		info.NodeType = "unknown"
		info.RawError = fmt.Sprintf("unknown prefix 0x%02x", value[0])
		return info
	}
}

// decodeConsensusModulePrefix decodes the consensus module key prefix.
//
// Prefix ranges based on Oasis Core v22.2.13:
//   0x10-0x19: Registry module
//   0x20-0x29: Roothash module
//   0x40-0x46: Beacon module
//   0x50-0x59: Staking module
//   0x60-0x63: Scheduler module
//   0x70:      Keymanager module
//   0x80-0x85: Governance module
//   0xF1:      Consensus parameters
//
// Source files in Oasis Core repository:
//   go/consensus/tendermint/apps/registry/state/state.go (0x10-0x19)
//   go/consensus/tendermint/apps/roothash/state/state.go (0x20-0x29)
//   go/consensus/tendermint/apps/beacon/state/state.go (0x40-0x43, 0x45)
//   go/consensus/tendermint/apps/beacon/state/state_vrf.go (0x46)
//   go/consensus/tendermint/apps/staking/state/state.go (0x50-0x59)
//   go/consensus/tendermint/apps/scheduler/state/state.go (0x60-0x63)
//   go/consensus/tendermint/apps/keymanager/state/state.go (0x70)
//   go/consensus/tendermint/apps/governance/state/state.go (0x80-0x85)
//   go/consensus/tendermint/abci/state/state.go (0xF1)
//
func decodeConsensusModulePrefix(key []byte) string {
	if len(key) == 0 {
		return "<empty>"
	}

	switch key[0] {
	// Registry module (0x10-0x19)
	// Source: _oasis-core/go/consensus/tendermint/apps/registry/state/state.go
	case 0x10:
		return "registry/entities"
	case 0x11:
		return "registry/nodes"
	case 0x12:
		return "registry/node_by_entity"
	case 0x13:
		return "registry/runtimes"
	case 0x14:
		return "registry/node_by_consensus_address"
	case 0x15:
		return "registry/node_status"
	case 0x16:
		return "registry/params"
	case 0x17:
		return "registry/key_map"
	case 0x18:
		return "registry/suspended_runtimes"
	case 0x19:
		return "registry/runtime_by_entity"

	// Roothash module (0x20-0x29)
	// Source: _oasis-core/go/consensus/tendermint/apps/roothash/state/state.go
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

	// Beacon module (0x40-0x46)
	// Source: _oasis-core/go/consensus/tendermint/apps/beacon/state/*.go
	case 0x40:
		return "beacon/epoch_current"
	case 0x41:
		return "beacon/epoch_future"
	case 0x42:
		return "beacon/beacon"
	case 0x43:
		return "beacon/params"
	case 0x44:
		return "beacon/pvss_state_deprecated"
	case 0x45:
		return "beacon/epoch_pending_mock"
	case 0x46:
		return "beacon/vrf_state"

	// Staking module (0x50-0x59)
	// Source: _oasis-core/go/consensus/tendermint/apps/staking/state/state.go
	case 0x50:
		return "staking/accounts"
	case 0x51:
		return "staking/total_supply"
	case 0x52:
		return "staking/common_pool"
	case 0x53:
		return "staking/delegations"
	case 0x54:
		return "staking/debonding_delegations"
	case 0x55:
		return "staking/debonding_queue"
	case 0x56:
		return "staking/params"
	case 0x57:
		return "staking/last_block_fees"
	case 0x58:
		return "staking/epoch_signing"
	case 0x59:
		return "staking/governance_deposits"

	// Scheduler module (0x60-0x63)
	// Source: _oasis-core/go/consensus/tendermint/apps/scheduler/state/state.go
	case 0x60:
		return "scheduler/committees"
	case 0x61:
		return "scheduler/validators_current"
	case 0x62:
		return "scheduler/validators_pending"
	case 0x63:
		return "scheduler/params"

	// Keymanager module (0x70)
	// Source: _oasis-core/go/consensus/tendermint/apps/keymanager/state/state.go
	case 0x70:
		return "keymanager/status"

	// Governance module (0x80-0x85)
	// Source: _oasis-core/go/consensus/tendermint/apps/governance/state/state.go
	case 0x80:
		return "governance/next_proposal_id"
	case 0x81:
		return "governance/proposals"
	case 0x82:
		return "governance/active_proposals"
	case 0x83:
		return "governance/votes"
	case 0x84:
		return "governance/pending_upgrades"
	case 0x85:
		return "governance/params"

	// Consensus parameters (0xF1)
	// Source: _oasis-core/go/consensus/tendermint/abci/state/state.go
	case 0xF1:
		return "consensus/params"

	default:
		if key[0] >= 'a' && key[0] <= 'z' {
			return extractModuleName(key)
		}
		return fmt.Sprintf("unknown:0x%02x", key[0])
	}
}

// decodeConsensusStateKey parses consensus-state key and returns structured info.
func decodeConsensusStateKey(key []byte) *ConsensusStateKeyInfo {
	info := &ConsensusStateKeyInfo{
		KeyDump: formatRawValue(key, TruncateLongLen),
		KeySize: len(key),
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

	// Parse ASCII key after 0x01 prefix
	decoded := string(key[1:])

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
func decodeConsensusStateValue(keyType string, value []byte) (info *ConsensusStateValueInfo) {
	info = &ConsensusStateValueInfo{
		RawDump: formatRawValue(value, TruncateLongLen),
		RawSize: len(value),
	}

	if len(value) == 0 {
		return info
	}

	// Recover from protobuf panics that occur with schema mismatches
	defer func() {
		if r := recover(); r != nil {
			info.RawError = fmt.Sprintf("protobuf panic: %v", r)
		}
	}()

	switch keyType {
	case "abci_responses":
		// Try CometBFT FinalizeBlock format first
		cbSuccess := false
		var cbResp cbResponseFinalizeBlock
		func() {
			defer func() {
				recover() // Silently catch panics from CometBFT unmarshal
			}()
			if err := proto.Unmarshal(value, &cbResp); err == nil && (len(cbResp.TxResults) > 0 || len(cbResp.Events) > 0 || len(cbResp.ValidatorUpdates) > 0) {
				cbSuccess = true
			}
		}()

		if cbSuccess {
			info.SchemaVersion = "cb-v0.37"
			abciResponseInfo := &ConsensusABCIResponseInfo{}

			// Count tx results
			if cbResp.TxResults != nil {
				abciResponseInfo.TxResultCount = len(cbResp.TxResults)
			}

			// Count validator updates
			if cbResp.ValidatorUpdates != nil {
				abciResponseInfo.ValidatorUpdates = len(cbResp.ValidatorUpdates)
			}

			// Count event types from all sources
			eventSummary := &ConsensusEventSummary{
				EventTypeCounts: make(map[string]int),
			}

			// Decode transactions and collect events from TxResults
			txSummary := &ConsensusTransactionSummary{
				MethodCounts: make(map[string]int),
			}

			if cbResp.TxResults != nil {
				for _, txResult := range cbResp.TxResults {
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

			// Events from root Events field (BeginBlock-style events in CometBFT)
			if cbResp.Events != nil {
				for _, event := range cbResp.Events {
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

			abciResponseInfo.EventCount = len(cbResp.Events)

			if len(txSummary.MethodCounts) > 0 {
				abciResponseInfo.TransactionSummary = txSummary
			}

			if len(eventSummary.EventTypeCounts) > 0 {
				abciResponseInfo.EventSummary = eventSummary
			}

			info.ABCIResponse = abciResponseInfo

		} else {
			// Fall back to Tendermint v0.34 format
			var resp tmABCIResponses
			if err := proto.Unmarshal(value, &resp); err == nil {
				// Validate that unmarshal actually decoded meaningful data
				if resp.DeliverTxs == nil && resp.BeginBlock == nil && resp.EndBlock == nil {
					info.RawError = "protobuf unmarshal succeeded but all fields are nil (schema mismatch)"
				} else {
					info.SchemaVersion = "tm-v0.34"
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
				info.RawError = fmt.Sprintf("failed to decode as CometBFT or Tendermint v0.34 schema: %v", err)
			}
		}

	case "consensus_params":
		var params tmConsensusParams
		if err := proto.Unmarshal(value, &params); err == nil {
			// Fields are non-nullable structs, so always initialized
			info.ConsensusParams = &params
		} else {
			info.RawError = fmt.Sprintf("failed to decode consensus params: %v", err)
		}

	case "validators":
		var valSet tmValidatorSet
		if err := proto.Unmarshal(value, &valSet); err == nil {
			// Validate that unmarshal decoded meaningful data
			if len(valSet.Validators) == 0 && valSet.Proposer == nil && valSet.TotalVotingPower == 0 {
				info.RawError = "protobuf unmarshal succeeded but all fields are empty/nil (schema mismatch)"
			} else {
				info.ConsensusValidators = &valSet
			}
		} else {
			info.RawError = fmt.Sprintf("failed to decode validator set: %v", err)
		}

	case "state":
		var state tmState
		if err := proto.Unmarshal(value, &state); err == nil {
			// Validate that unmarshal decoded meaningful data
			if state.ChainID == "" && state.LastBlockHeight == 0 && state.InitialHeight == 0 {
				info.RawError = "protobuf unmarshal succeeded but all fields are empty/zero (schema mismatch)"
			} else {
				info.ConsensusState = &state
			}
		} else {
			info.RawError = fmt.Sprintf("failed to decode state: %v", err)
		}

	case "genesis":
		if value[0] != '{' {
			info.RawError = "genesis document not in expected JSON format"
		}
		// Genesis is JSON, RawDump shows hex preview (already set above)
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
	info.Signer = formatRawValue(signedTx.Signature.PublicKey[:], TruncateLongLen)

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
		info.BodyHex = formatRawValue(tx.Body, TruncateLongLen)
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
		info.BodyHex = formatRawValue(cborData, TruncateLongLen)
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
