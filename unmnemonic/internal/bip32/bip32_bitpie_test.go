package bip32

import (
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/oasisprotocol/curve25519-voi/primitives/ed25519"

	"github.com/oasisprotocol/tools/unmnemonic/internal/address"
	"github.com/oasisprotocol/tools/unmnemonic/internal/bip39"
)

func testKnownAnswerBitpie(t *testing.T) {
	// From the test vector and intermediaries provided by Bitpie...
	const mnemonicStr = "cross enable vendor service pulse account ceiling omit trial myself front misery"
	mnemonicBytes, err := bip39.ValidateAndExpandMnemonic([]byte(mnemonicStr))
	if err != nil {
		t.Fatalf("ValidateAndExpandMnemonic: %v", err)
	}
	seed := bip39.MnemonicToSeed(nil, mnemonicBytes)

	t.Logf("seed: %02x", seed)

	// Derive the root node
	root, err := NewBitpieRoot(seed)
	if err != nil {
		t.Fatalf("NewBitpieRoot: %v", err)
	}
	debugDumpNode(t, "root", root)
	assertNodeEqualsHex(
		t,
		"fe333947e1dce3fcfa377dce4099f2972eadc25b1b6a4d5f60878969cd657bc0",
		"0000000000000000000000000000000000000000000000000000000000000000",
		"0000000000000000000000000000000000000000000000000000000000000000",
		root,
	)

	// Derive m/0
	child, err := root.DeriveChild(0)
	if err != nil {
		t.Fatalf("DeriveChild(0) (m/0): %v", err)
	}
	pk, err := bitpieSeedToPublicKey(child.kL[:])
	if err != nil {
		t.Fatalf("DeriveChild(0) to public: %v", err)
	}
	debugDumpNode(t, "child-m/0", child)
	assertNodeEqualsHex(
		t,
		"0ef310a16dcc1b68ac50b2b9c4890d076ab4128945db2d51e9f049ffb8667bc0",
		"0000000000000000000000000000000000000000000000000000000000000000",
		"5d294b5eba1b20afa30f4420992fef8ba741abaab5f02458884accd31afc0b62",
		child,
	)
	assertBytesEqualsHex(
		t,
		"child pub (m/0)",
		"93262aee1b625f793b7631f7879195e93b9bb89e2dd09cd5d639e00cb738462a",
		pk,
	)

	// Derive m/0/0
	child, err = child.DeriveChild(0)
	if err != nil {
		t.Fatalf("DeriveChild(0) (m/0/0): %v", err)
	}
	pk, err = bitpieSeedToPublicKey(child.kL[:])
	if err != nil {
		t.Fatalf("DeriveChild(0) to public: %v", err)
	}
	debugDumpNode(t, "child-m/0/0", child)
	assertNodeEqualsHex(
		t,
		"4e2e5cd369441263322fb2a95cba2cd14377662f623317c740548d3793687bc0",
		"0000000000000000000000000000000000000000000000000000000000000000",
		"5fb00636aaca0fd18aff232c6dee96a38c80170258fc0f5c3ecc7d16c8555f85",
		child,
	)
	assertBytesEqualsHex(
		t,
		"child pub (m/0/0)",
		"afa004d2863641f69a6ea725cb7abca70d6069c476ec3ed119c6dc6c72fa4e79",
		pk,
	)
}

func testKnownAnswerBitpieConsistency(t *testing.T) {
	// With the same mnemonic, we have the final values generated by @matevz,
	// and the intermediaries/final values from the example.  Check to see if
	// they are consistent.

	// From the wallet:
	// Private key: Ti5c02lEEmMyL7KpXLos0UN3Zi9iMxfHQFSNN5Noe8CvoATShjZB9ppupyXLerynDWBpxHbsPtEZxtxscvpOeQ==
	// Address:     oasis1qp8d9kuduq0zutuatjsgltpugxvl38cuaq3gzkmn

	const expectedAddr = "oasis1qp8d9kuduq0zutuatjsgltpugxvl38cuaq3gzkmn"

	t.Run("Exported", func(t *testing.T) {
		rawPriv, err := base64.StdEncoding.DecodeString(
			"Ti5c02lEEmMyL7KpXLos0UN3Zi9iMxfHQFSNN5Noe8CvoATShjZB9ppupyXLerynDWBpxHbsPtEZxtxscvpOeQ==",
		)
		if err != nil {
			t.Fatalf("base64.StdEncoding.DecodeString(): %v", err)
		}

		// Trailing 32-bytes is the public key.
		pubKey := ed25519.PublicKey(rawPriv[32:])
		addr, err := address.FromPublicKey(pubKey)
		if err != nil {
			t.Fatalf("address.FromPublicKey(): %v", err)
		}
		if addr != expectedAddr {
			t.Fatalf("address mismatch, got '%v' expected '%v'", addr, expectedAddr)
		}

		// Leading 32-bytes is the RFC 8032 seed.
		derivedPrivKey := ed25519.NewKeyFromSeed(rawPriv[:32])
		derivedAddr, err := address.FromPublicKey(derivedPrivKey.Public())
		if err != nil {
			t.Fatalf("address.FromPublicKey(): %v", err)
		}
		if derivedAddr != expectedAddr {
			t.Fatalf("exported: derived address mismatch, got '%v' expected '%v'", derivedAddr, expectedAddr)
		}
	})

	t.Run("Example", func(t *testing.T) {
		rawPriv, err := hex.DecodeString(
			"4e2e5cd369441263322fb2a95cba2cd14377662f623317c740548d3793687bc0afa004d2863641f69a6ea725cb7abca70d6069c476ec3ed119c6dc6c72fa4e79",
		)
		if err != nil {
			t.Fatalf("hex.DecodeString(): %v", err)
		}

		// Trailing 32-bytes is the public key.
		pubKey := ed25519.PublicKey(rawPriv[32:])
		addr, err := address.FromPublicKey(pubKey)
		if err != nil {
			t.Fatalf("address.FromPublicKey(): %v", err)
		}
		if addr != expectedAddr {
			t.Fatalf("address mismatch, got '%v' expected '%v'", addr, expectedAddr)
		}

		// Since this matches, the leading 32-bytes must be the seed, since
		// I doubt they can reverse SHA-512.  And... yep.  Ok, so that's how
		// export works.
		derivedPrivKey := ed25519.NewKeyFromSeed(rawPriv[:32])
		derivedAddr, err := address.FromPublicKey(derivedPrivKey.Public())
		if err != nil {
			t.Fatalf("address.FromPublicKey(): %v", err)
		}
		if derivedAddr != expectedAddr {
			t.Fatalf("exported: derived address mismatch, got '%v' expected '%v'", derivedAddr, expectedAddr)
		}
	})
}
