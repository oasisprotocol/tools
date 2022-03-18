package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"github.com/dgraph-io/badger/v3"
	"github.com/dgraph-io/badger/v3/options"
	gogotypes "github.com/gogo/protobuf/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"github.com/tendermint/tendermint/types"
)

// Badger's read only mode is busted LOL.
// https://discuss.dgraph.io/t/read-only-log-truncate-required-to-run-db/16444

var (
	evidenceDir   string
	blockstoreDir string
)

func main() {
	flag.Parse()

	if err := doDump(); err != nil {
		fmt.Printf("evidence dump failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("dump completed\n")

	os.Exit(0)
}

func doDump() error {
	// Pull out the relevant heights from the evidence db.
	heights, err := getCommittedHeights()
	if err != nil {
		return err
	}

	fmt.Printf("\n heights in blockstore: %+v\n\n", heights)

	if err = dumpBlockEvidence(heights); err != nil {
		fmt.Printf("block store dump failed: %v\n", err)
		os.Exit(1)
	}

	return nil
}

func getCommittedHeights() ([]int64, error) {
	opts := badger.DefaultOptions(evidenceDir)
	opts = opts.WithCompression(options.Snappy)
	// opts = opts.WithReadOnly(true)
	opts = opts.WithLogger(nil)
	opts = opts.WithBlockCacheSize(64 * 1024 * 1024)

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("dump: failed to open db: %w", err)
	}
	defer db.Close()

	fmt.Printf("dumping evidence db:\n")

	var heights []int64
	if err = db.View(func(txn *badger.Txn) error {
		iterOpt := badger.DefaultIteratorOptions
		it := txn.NewIterator(iterOpt)
		defer it.Close()

		var i int
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()

			key := item.Key()
			value, _ := item.ValueCopy(nil)

			keyStr := hex.Dump(key)
			valueStr := hex.Dump(value)
			fmt.Printf("[%d]:\n key: %s\n value: %s\n", i, keyStr, valueStr)

			switch key[1] {
			case 0x01:
				var pb tmproto.Evidence
				if txErr := pb.Unmarshal(value); txErr != nil {
					return fmt.Errorf("dump: failed to deserialize evidence: %w", txErr)
				}
				fmt.Printf(" pending: %+v\n", pb)
			case 0x00:
				var height gogotypes.Int64Value
				if txErr := height.Unmarshal(value); txErr != nil {
					return fmt.Errorf("dump: failed to deserialize height: %w", txErr)
				}
				fmt.Printf(" committed: %+v\n", height.Value)
				heights = append(heights, height.Value)
			default:
				fmt.Printf("skipping unknown prefix: '%x'\n", key[1])
			}

			i++
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("dump: failed evidence view transaction: %w", err)
	}

	return heights, nil
}

func dumpBlockEvidence(heights []int64) error {
	opts := badger.DefaultOptions(blockstoreDir)
	opts = opts.WithCompression(options.Snappy)
	// opts = opts.WithReadOnly(true)
	opts = opts.WithLogger(nil)
	opts = opts.WithBlockCacheSize(64 * 1024 * 1024)

	db, err := badger.Open(opts)
	if err != nil {
		return fmt.Errorf("dump: failed to open db: %w", err)
	}
	defer db.Close()

	fmt.Printf("dumping blockstore db:\n")

	if err = db.View(func(txn *badger.Txn) error {
		for _, v := range heights {
			v = v + 1 // Evidence is in the next block.
			// Retreive the block metadata.
			metaKey := []byte(fmt.Sprintf("H:%d", v))
			key := append([]byte{0x01}, metaKey...)
			item, txErr := txn.Get(key)
			if txErr != nil {
				return fmt.Errorf("dump: block %d missing: %w", v, txErr)
			}
			b, _ := item.ValueCopy(nil)
			var pbbm tmproto.BlockMeta
			if txErr = pbbm.Unmarshal(b); txErr != nil {
				return fmt.Errorf("dump: failed to deserialize block %d meta: %w", v, txErr)
			}
			blockMeta, txErr := types.BlockMetaFromProto(&pbbm)
			if txErr != nil {
				return fmt.Errorf("dump: failed to convert block %d meta: %w", v, txErr)
			}

			fmt.Printf(" block %d:\n  meta: %+v\n", v, blockMeta)

			// Reassemble the block from parts. (Jesus fucking christ)
			var rawBlockProto []byte
			for i := 0; i < int(blockMeta.BlockID.PartSetHeader.Total); i++ {
				partKey := []byte(fmt.Sprintf("P:%d:%d", v, i))
				key = append([]byte{0x01}, partKey...)
				item, txErr = txn.Get(key)
				if txErr != nil {
					return fmt.Errorf("dump: block %d/%d missing: %w", v, i, txErr)
				}
				b, _ = item.ValueCopy(nil)

				var pbpart tmproto.Part
				if txErr = pbpart.Unmarshal(b); txErr != nil {
					return fmt.Errorf("dump: failed to deserialize block %d/%d: %w", v, i, txErr)
				}
				part, txErr := types.PartFromProto(&pbpart)
				if txErr != nil {
					return fmt.Errorf("dump: failed to convert block %d/%d: %w", v, i, txErr)
				}
				rawBlockProto = append(rawBlockProto, part.Bytes...)
			}

			var pbb tmproto.Block
			if txErr = pbb.Unmarshal(rawBlockProto); err != nil {
				return fmt.Errorf("dump: failed to deserialize block %d: %w", v, txErr)
			}

			block, txErr := types.BlockFromProto(&pbb)
			if txErr != nil {
				return fmt.Errorf("dump: failed to convert block %d: %w", v, txErr)
			}

			fmt.Printf(" body: %+v\n", block)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("dump: failed blockstore view transaction: %w", err)
	}

	return nil
}

func init() {
	flag.StringVar(&evidenceDir, "e", "evidence.badger.db", "the evidence badger db dir")
	flag.StringVar(&blockstoreDir, "b", "blockstore.badger.db", "the block badger db")
}
