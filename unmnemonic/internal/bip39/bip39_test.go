package bip39

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

const passphraseTrezor = "TREZOR"

func TestShortMnemonic(t *testing.T) {
	if _, err := ValidateAndExpandMnemonic([]byte("abandon")); err == nil {
		t.Fatalf("failed to reject undersized mnemonic")
	} else {
		t.Logf("undersized mnemonic: %v", err)
	}
}

func TestBadChecksum(t *testing.T) {
	if _, err := ValidateAndExpandMnemonic([]byte("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon")); err == nil {
		t.Fatalf("failed to reject corrupted mnemonic")
	} else {
		t.Logf("corrupted mnemonic: %v", err)
	}
}

func TestAbbreviation(t *testing.T) {
	const expected = "letter advice cage absurd amount doctor acoustic avoid letter advice cage absurd amount doctor acoustic avoid letter advice cage absurd amount doctor acoustic bless"
	expanded, err := ValidateAndExpandMnemonic([]byte("lett advi cage absu amou doct acou avoi LETT ADVI CAGE ABSU AMOU DOCT ACOU AVOI letter advice cage absurd amount doctor acoustic bless"))
	if err != nil {
		t.Fatalf("failed to expand menemonic with abbreviations: %v", err)
	}
	if !bytes.Equal(expanded, []byte(expected)) {
		t.Fatalf("mnemonic mismatch: expected '%s', got '%s'", expected, expanded)
	}
}

func TestVectors(t *testing.T) {
	// Deserialize the test vectors.
	rawVectors, err := os.ReadFile("../testdata/bip39_vectors.json")
	if err != nil {
		t.Fatalf("failed to read test vectors: %v", err)
	}
	testVectors := make(map[string][][4]string)
	if err = json.Unmarshal(rawVectors, &testVectors); err != nil {
		t.Fatalf("failed to deserialize test vectors: %v", err)
	}

	// English motherfucker, do you speak it?
	englishTestVectors := testVectors["english"]
	if len(englishTestVectors) == 0 {
		t.Fatalf("failed to find english test vectors")
	}
	for i, vector := range englishTestVectors {
		t.Run(fmt.Sprintf("Vector %d", i), func(t *testing.T) {
			// entropy, mnemonic, seed, xprv(?)
			entropy, err := hex.DecodeString(vector[0])
			if err != nil {
				t.Fatalf("failed to deserialize entropy: %v", err)
			}
			mnemonic := []byte(vector[1])
			seed, err := hex.DecodeString(vector[2])
			if err != nil {
				t.Fatalf("failed to deserialize seed: %v", err)
			}

			_ = entropy // Unused since we don't do generation.

			derivedMnemonic, err := ValidateAndExpandMnemonic(mnemonic)
			if err != nil {
				t.Fatalf("failed to validate/expand mnemonic: %v", err)
			}
			if !bytes.Equal(mnemonic, derivedMnemonic) {
				t.Fatalf("mnemonic mismatch: expected '%s', got '%s'", mnemonic, derivedMnemonic)
			}
			derivedSeed := MnemonicToSeed([]byte(passphraseTrezor), derivedMnemonic)
			if !bytes.Equal(seed, derivedSeed) {
				t.Fatalf("seed mismatch: expected %02x, got %02x", seed, derivedSeed)
			}
		})
	}
}
