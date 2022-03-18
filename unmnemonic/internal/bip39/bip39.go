// Package bip39 implements the non-generation portions of BIP-39
// "Mnemonic code for generating deterministic keys".
package bip39

import (
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"math/big"

	"golang.org/x/crypto/pbkdf2"
)

const (
	expansionSize       = 64
	expansionIters      = 2048
	expansionSaltPrefix = "mnemonic"
)

// GetEntropyBits returns the amount of entropy in a given number of words, in bits.
func GetEntropyBits(l int) (int, error) {
	switch l {
	case 12:
		return 128, nil
	case 15:
		return 160, nil
	case 18:
		return 192, nil
	case 21:
		return 224, nil
	case 24:
		return 256, nil
	default:
		return 0, fmt.Errorf("bip39: invalid mnemonic, unexpected number of words: %d", l)
	}
}

// ValidateAndExpandMnemonic expands abbreviated 4-character prefixes to their
// full words, validates the mnemonic for correctness, and returns the full
// mnemonic suitable for seed derivation.
func ValidateAndExpandMnemonic(raw []byte) ([]byte, error) {
	splitRaw := bytes.Split(raw, []byte(" "))
	entropyBits, err := GetEntropyBits(len(splitRaw))
	if err != nil {
		return nil, err
	}

	// Note: This is not anything resembling constant time.  Users would
	// need to be out of their god damn minds to use this on a system
	// connected to any network, so whatever.

	entropy := big.NewInt(0)
	expandedWords := make([][]byte, 0, len(splitRaw))
	for _, prefix := range splitRaw {
		word, err := ExpandWord(string(prefix))
		if err != nil {
			return nil, err
		}

		expandedWords = append(expandedWords, []byte(word))

		// Store the 11 bits corresponding to the word as well.
		entropy = entropy.Lsh(entropy, 11)
		entropy = entropy.Or(entropy, big.NewInt(int64(englishWordLUT[word])))
	}

	// Use the accumulated bits to derive the initial entropy and
	// checksum.  Nothing fancy, just the checksum concatenated to
	// the entropy.
	checksumBits := uint(entropyBits) / 32
	bigChecksum := big.NewInt((1 << checksumBits) - 1)
	bigChecksum = bigChecksum.And(bigChecksum, entropy)
	checksum := bigChecksum.Uint64()

	entropy = entropy.Rsh(entropy, checksumBits)
	entropyBytes := make([]byte, entropyBits/8)
	entropyBytes = entropy.FillBytes(entropyBytes)

	// Validate the checksum, which is the first n-bits of the SHA256
	// digest of the entropy.
	entropyDigest := sha256.Sum256(entropyBytes)
	derivedChecksum := entropyDigest[0] >> (8 - checksumBits)
	if derivedChecksum != byte(checksum) {
		return nil, fmt.Errorf("bip39: checksum mismatch")
	}

	// Checksum ok, return the possibly expanded mnemonic.
	return bytes.Join(expandedWords, []byte(" ")), nil
}

// MnemonicToSeed converts from a mnemonic to a seed.  Note that the mnemonic
// should be validated and fixed-up with ValidateAndExpandMnemonic prior to
// being converted to a seed.
func MnemonicToSeed(passphrase, mnemonic []byte) []byte {
	salt := append([]byte(expansionSaltPrefix), passphrase...)
	return pbkdf2.Key(mnemonic, salt, expansionIters, expansionSize, sha512.New)
}
