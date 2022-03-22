package main

import (
	"bytes"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/oasisprotocol/curve25519-voi/primitives/ed25519"

	"github.com/oasisprotocol/tools/unmnemonic/internal/address"
	"github.com/oasisprotocol/tools/unmnemonic/internal/bip32"
	"github.com/oasisprotocol/tools/unmnemonic/internal/bip39"
	"github.com/oasisprotocol/tools/unmnemonic/internal/slip10"
)

const (
	algoLedger  = "Ledger"
	algoAdr0008 = "ADR-0008"

	maxAccountKeyNumber = uint32(0x7fffffff)
)

func perror(err error) {
	fmt.Printf("err: %v\n", err)
	os.Exit(1)
}

func main() {
	// unmnemonic is explicitly interactive because people will probably
	// splatter their mnemonic into their shell history otherwise.
	if err := doInteractive(); err != nil {
		perror(err)
	}
}

func doInteractive() error {
	// Display splash screen and warning.
	fmt.Printf("\n")
	fmt.Printf("  unmnemonic - Recover Oasis Network signing keys from mnemonics\n")
	fmt.Printf("\n")

	// Figure out the derivation scheme.
	var algo string
	if err := survey.AskOne(&survey.Select{
		Message: "Which algorithm does your wallet use",
		Options: []string{algoAdr0008, algoLedger},
	}, &algo); err != nil {
		return err
	}

	if algo == algoLedger {
		fmt.Printf(" WARNING:\n")
		fmt.Printf("\n")
		fmt.Printf("  Entering your Ledger device mnemonic into any non-Leger device\n")
		fmt.Printf("  can COMPROMISE THE SECURITY OF ALL ACCOUNTS TIED TO THE MNEMONIC.\n")
		fmt.Printf("  Use of this tool is STRONGLY DISCOURAGED.\n")
		fmt.Printf("\n")

		// Make sure the user knows what they are getting into.
		var ok bool
		if err := survey.AskOne(&survey.Confirm{
			Message: "Have you read and understand the warning",
		}, &ok); err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("user abort")
		}
	}

	// Deal with mnemonic entry.
	var s string
	if err := survey.AskOne(&survey.Input{
		Message: "How many words is your mnemonic",
		Default: "24",
	}, &s, survey.WithValidator(isMnemonicLength)); err != nil {
		return err
	}

	var mnemonic []byte
	mnemonicLength, _ := strconv.ParseUint(s, 10, 32)
	for {
		words := make([]string, 0, int(mnemonicLength))
		for i := 1; i <= int(mnemonicLength); i++ {
			if err := survey.AskOne(&survey.Password{
				Message: fmt.Sprintf("Enter word %d", i),
			}, &s, survey.WithValidator(isMnemonicWord)); err != nil {
				return err
			}
			words = append(words, s)
		}

		var err error
		if mnemonic, err = bip39.ValidateAndExpandMnemonic([]byte(strings.Join(words, " "))); err != nil {
			fmt.Printf(" Invalid mnemonic: %v\n", err)
			continue
		}

		break
	}

	// Read the index(es).
	var indexes []uint32
	if err := survey.AskOne(&survey.Input{
		Message: "Wallet index(es) (comma separated)",
		Default: "0",
	}, &s, survey.WithValidator(isCommaSeparatedUint32List)); err != nil {
		return err
	}
	for _, v := range strings.Split(s, ",") {
		idx, _ := strconv.ParseUint(strings.TrimSpace(v), 10, 32)
		indexes = append(indexes, uint32(idx))
	}

	// Do the derivation.
	var (
		seed  = bip39.MnemonicToSeed(nil, mnemonic)
		infos []*walletInfo
		err   error
	)
	switch algo {
	case algoLedger:
		infos, err = deriveLedger(seed, indexes)
	case algoAdr0008:
		infos, err = deriveAdr0008(seed, indexes)
	default:
		return fmt.Errorf("unknown algorithm")
	}
	if err != nil {
		return err
	}
	for _, v := range infos {
		fmt.Printf(" Index[%d]: %s\n", v.index, v.address)
	}

	// Figure out if the user wants to write out the keys
	var ok bool
	if err = survey.AskOne(&survey.Confirm{
		Message: "Write the keys to disk",
	}, &ok); err != nil {
		return err
	}
	if !ok {
		// Welp, all done.
		os.Exit(0)
	}

	// Figure out the output directory.
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	if err = survey.AskOne(&survey.Input{
		Message: "Output directory",
		Default: filepath.Join(wd, "wallet-export-"+time.Now().Format("2006-01-02")),
	}, &s); err != nil {
		return err
	}
	if err = os.MkdirAll(s, 0o700); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write out each wallet to disk.
	for _, info := range infos {
		fn := fmt.Sprintf("%s.private.pem", info.address)
		b, err := encodeEd25519PrivateToPEMBuf(info.privateKey)
		if err != nil {
			return fmt.Errorf("failed to encode private key to PEM: %w", err)
		}
		if err = os.WriteFile(filepath.Join(s, fn), b, 0o600); err != nil {
			return fmt.Errorf("failed to write private key to file: %w", err)
		}
		fmt.Printf(" Index[%d]: %s - done\n", info.index, fn)
	}

	fmt.Printf("Done writing wallet keys to disk, goodbye.\n")

	return nil
}

func deriveLedger(seed []byte, indexes []uint32) ([]*walletInfo, error) {
	root, err := bip32.NewLedgerRoot(seed)
	if err != nil {
		return nil, fmt.Errorf("failed to derive BIP32-Ed25519 root: %w", err)
	}
	wallet, err := root.DerivePath("44'/474'/0'/0'")
	if err != nil {
		return nil, fmt.Errorf("failed to derive BIP32-Ed25519 wallet base: %w", err)
	}

	infos := make([]*walletInfo, 0, len(indexes))
	for _, index := range indexes {
		child, err := wallet.DeriveChild(index + bip32.HardenedIndexOffset)
		if err != nil {
			return nil, fmt.Errorf("failed to derive key for index %d: %w", index, err)
		}
		privateKey := child.GetLedgerPrivateKey()
		address, err := address.FromPublicKey(privateKey.Public())
		if err != nil {
			return nil, fmt.Errorf("failed to derive address for index %d: %w", index, err)
		}
		infos = append(infos, &walletInfo{
			index:      index,
			privateKey: privateKey,
			address:    address,
		})
	}

	return infos, nil
}

func deriveAdr0008(seed []byte, indexes []uint32) ([]*walletInfo, error) {
	// Derive the master secret.
	secret, chainCode, err := slip10.NewMasterKey(seed)
	if err != nil {
		return nil, fmt.Errorf("failed to derive SLIP-10 master key: %w", err)
	}

	// All wallets are in the path `//m/44'/474'/index'`, so descent to
	// the common sub-root.
	for _, index := range []uint32{44, 474} {
		secret, chainCode, err = slip10.NewChildKey(secret, chainCode, index+bip32.HardenedIndexOffset)
		if err != nil {
			return nil, fmt.Errorf("failed to derive SLIP-10 sub-key (%d): %w", index, err)
		}
	}
	infos := make([]*walletInfo, 0, len(indexes))
	for _, index := range indexes {
		child, _, err := slip10.NewChildKey(secret, chainCode, index+bip32.HardenedIndexOffset)
		if err != nil {
			return nil, fmt.Errorf("failed to derive key for index %d: %w", index, err)
		}
		privateKey := ed25519.NewKeyFromSeed(child[:])
		address, err := address.FromPublicKey(privateKey.Public())
		if err != nil {
			return nil, fmt.Errorf("failed to derive address for index %d: %w", index, err)
		}
		infos = append(infos, &walletInfo{
			index:      index,
			privateKey: privateKey,
			address:    address,
		})
	}

	return infos, nil
}

type walletInfo struct {
	index      uint32
	privateKey ed25519.PrivateKey
	address    string
}

func isMnemonicLength(val interface{}) error {
	s := val.(string)
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid number: '%s'", s)
	}
	if _, err = bip39.GetEntropyBits(int(v)); err != nil {
		return fmt.Errorf("invalid mnemonic length")
	}
	return nil
}

func isCommaSeparatedUint32List(val interface{}) error {
	s := val.(string)
	spl := strings.Split(s, ",")
	for _, v := range spl {
		v = strings.TrimSpace(v)
		i, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid index: '%s'", v)
		}
		if i > uint64(maxAccountKeyNumber) {
			return fmt.Errorf("invalid index (out of range): '%s'", v)
		}
	}
	return nil
}

func isMnemonicWord(val interface{}) error {
	s := val.(string)
	_, err := bip39.ExpandWord(s)
	return err
}

func encodeEd25519PrivateToPEMBuf(k ed25519.PrivateKey) ([]byte, error) {
	blk := &pem.Block{
		Type:  "ED25519 PRIVATE KEY",
		Bytes: k[:],
	}

	var buf bytes.Buffer
	if err := pem.Encode(&buf, blk); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
