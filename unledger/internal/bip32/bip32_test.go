package bip32

import (
	"encoding/hex"
	"testing"

	"github.com/oasisprotocol/curve25519-voi/primitives/ed25519"

	"github.com/oasisprotocol/tools/unledger/internal/bip39"
)

func TestKnownAnswer(t *testing.T) {
	// All values taken from the python code I was told implements what
	// a real ledger device will.
	//
	// To the author of the python's credit, it does dump intermediaries.

	// BIP39 Mnemonic to seed
	const mnemonicStr = "equip will roof matter pink blind book anxiety banner elbow sun young"
	mnemonicBytes, err := bip39.ValidateAndExpandMnemonic([]byte(mnemonicStr))
	if err != nil {
		t.Fatalf("ValidateAndExpandMnemonic: %v", err)
	}
	seed := bip39.MnemonicToSeed(nil, mnemonicBytes)
	assertBytesEqualsHex(
		t,
		"seed",
		"ed2f664e65b5ef0dd907ae15a2788cfc98e41970bc9fcb46f5900f6919862075e721f37212304a56505dab99b001cc8907ef093b7c5016a46b50c01cc3ec1cac",
		seed,
	)

	// Derive the root node
	root, err := NewRoot(seed)
	if err != nil {
		t.Fatalf("NewRoot: %v", err)
	}
	assertNodeEqualsHex(
		t,
		"281a231bf49b85f6490a21e1a9b526989d6a920e477772c1cf822e1c844b1257",
		"d5a2f9749e739aba4c7256209669cdfc1f8872713c0f62556ab0d25746e2905e",
		"04e5250fc6937d6d6848507a24661866bd402327d63e6da8637e55024bb4227b",
		root,
	)
	debugDumpNode(t, "root", root)

	// Incremental derivation
	t.Run("Incremental", func(t *testing.T) {
		// Derive each of the children manually (44'/474'/5'/0'/3')

		child, err := root.DeriveChild(HardenedIndexOffset + 44)
		if err != nil {
			t.Fatalf("DeriveChild(44'): %v", err)
		}
		assertNodeEqualsHex(
			t,
			"c0d7bdabf85770d87a5c2668620ae2e9f0c19135d15b2838601c7f13894b1257",
			"4f616a0a223bb44c3ea1f97242a46d3f70ac5ee7158aaf9491a05686197708a4",
			"83dc6af53af18e27fa51e7ddecedf2acc70360f4293ccc93af374fef93652e4d",
			child,
		)
		debugDumpNode(t, "child-44h", child)

		child, err = child.DeriveChild(HardenedIndexOffset + 474)
		if err != nil {
			t.Fatalf("DeriveChild(474'): %v", err)
		}
		assertNodeEqualsHex(
			t,
			"a87af482371ff302152591aa2dc34a4329a48c0e174fd512b63a38588a4b1257",
			"9e4e5ace531ea11cadf8d9f044141f08b71af40af33747c78446bba223f5370a",
			"d8641f9ed8752fc75b061a10f8cf496e48d31bcdc86346208ae3295ae146a162",
			child,
		)
		debugDumpNode(t, "child-474h", child)

		child, err = child.DeriveChild(HardenedIndexOffset + 5)
		if err != nil {
			t.Fatalf("DeriveChild(5'): %v", err)
		}
		assertNodeEqualsHex(
			t,
			"308122616df2c326ec1dcf3022cb7dd5b0c7e44864ba5f5c65211439904b1257",
			"3ffdced9554ebce172156bc07b8214a0875cb415edd53438ed3278447bb494a3",
			"e50e9114c26dd7966c5fe0beef7c6630885f622295dde3474d0b2a90b34deb4d",
			child,
		)
		debugDumpNode(t, "child-5h", child)

		child, err = child.DeriveChild(HardenedIndexOffset + 0)
		if err != nil {
			t.Fatalf("DeriveChild(0'): %v", err)
		}
		assertNodeEqualsHex(
			t,
			"38057f8f6486e8488d5c9a92278e33aad5f5f9b9165c0cac9a435725934b1257",
			"1be17ce5330bbd80798174ba619965f3e159bb75cf60b84aee27ee27fb4e4635",
			"b4c1045a54573211c0df8e4da36456ea0425f22db81e4de5d8236a1d0f855eaa",
			child,
		)
		debugDumpNode(t, "child-0h", child)

		child, err = child.DeriveChild(HardenedIndexOffset + 3)
		if err != nil {
			t.Fatalf("DeriveChild(3'): %v", err)
		}
		assertNodeEqualsHex(
			t,
			"58438fbaae6d0192420b1793a80b5579ae5ca30cbe51e4746cee81c4944b1257",
			"70dfacd37c00539d51fc0cdf7f4f457b0efa46241f69cb249d2837ecc5508c65",
			"94aaa7974cad3d44b04d8f17ff06920251ebe2a9eac0ecfa6e86d29e0063c91d",
			child,
		)
		debugDumpNode(t, "child-3h", child)
	})

	// Convenience derivation
	t.Run("DerivePath", func(t *testing.T) {
		child, err := root.DerivePath("44'/474'/5'/0'/3'")
		if err != nil {
			t.Fatalf("DerivePath(44'/474'/5'/0'/3'): %v", err)
		}
		assertNodeEqualsHex(
			t,
			"58438fbaae6d0192420b1793a80b5579ae5ca30cbe51e4746cee81c4944b1257",
			"70dfacd37c00539d51fc0cdf7f4f457b0efa46241f69cb249d2837ecc5508c65",
			"94aaa7974cad3d44b04d8f17ff06920251ebe2a9eac0ecfa6e86d29e0063c91d",
			child,
		)
		debugDumpNode(t, "child-oneshot", child)

		privKey := child.GetOasisPrivateKey()
		pubKey := (privKey.Public()).(ed25519.PublicKey)
		assertBytesEqualsHex(
			t,
			"public key",
			"aba52c0dcb80c2fe96ed4c3741af40c573a0500c0d73acda22795c37cb0f1739",
			pubKey[:],
		)
	})
}

func assertBytesEqualsHex(t *testing.T, descr, expected string, actual []byte) {
	aStr := hex.EncodeToString(actual)
	if expected != aStr {
		t.Fatalf("%s mismatch, expected %s, got %s", descr, expected, aStr)
	}
}

func assertNodeEqualsHex(t *testing.T, expectedKl, expectedKr, expectedC string, n *Node) {
	kLStr := hex.EncodeToString(n.kL[:])
	if expectedKl != kLStr {
		t.Errorf("kL mismatch, expected %s, got %s", expectedKl, kLStr)
	}

	kRStr := hex.EncodeToString(n.kR[:])
	if expectedKr != kRStr {
		t.Errorf("kR mismatch, expected %s, got %s", expectedKr, kRStr)
	}

	cStr := hex.EncodeToString(n.c[:])
	if expectedC != cStr {
		t.Errorf("c mismatch, expected %s, got %s", expectedC, cStr)
	}

	if t.Failed() {
		t.FailNow()
	}
}

func debugDumpNode(t *testing.T, descr string, n *Node) {
	t.Logf("Node(%s):\nkL: %02x\nkR: %02x\nc: %02x", descr, n.kL[:], n.kR[:], n.c[:])
}
