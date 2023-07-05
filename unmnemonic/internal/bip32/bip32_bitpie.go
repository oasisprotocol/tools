package bip32

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"fmt"

	"github.com/oasisprotocol/curve25519-voi/primitives/ed25519"
	"github.com/tyler-smith/go-bip32"
)

func (n *Node) deriveBitpieChild(idx uint32) (*Node, error) {
	if idx >= HardenedIndexOffset {
		return nil, fmt.Errorf("bip32: bitpie derivation is non-hardened only")
	}

	aBytes, err := bitpieSeedToPublicKey(n.kL[:])
	if err != nil {
		return nil, err
	}

	var iBytes [4]byte
	binary.BigEndian.PutUint32(iBytes[:], idx)

	zMac := hmac.New(sha512.New, n.c[:])
	_, _ = zMac.Write([]byte("N"))
	_, _ = zMac.Write(aBytes)
	_, _ = zMac.Write(iBytes[:])
	h := zMac.Sum(nil)

	childNode := &Node{
		isBitpie: true,
	}
	copy(childNode.kL[:], h[:32])
	copy(childNode.c[:], h[32:])

	// Clear cofactor, truncate to 225-bits.
	childNode.kL[0] &= 248
	childNode.kL[29] &= 1
	childNode.kL[30] = 0
	childNode.kL[31] = 0

	// kL = [h] + [k_P_L]
	var carry uint16
	for i := 0; i < 32; i++ {
		tmp := uint16(childNode.kL[i]) + uint16(n.kL[i]) + carry
		childNode.kL[i] = byte(tmp & 0xff)
		carry = tmp >> 8
	}

	if carry != 0 {
		return nil, fmt.Errorf("bip32: bitpie child derivation overflows")
	}

	return childNode, nil
}

func bitpieSeedToPublicKey(seed []byte) (ed25519.PublicKey, error) {
	// bitpie claims:
	//
	// EdPublicKey XPub() {
	//   byte[] scalar = new byte[32];
	//   System.arraycopy(toBytes(),0,scalar,0,32);
	//   GroupElement point = ED_25519_CURVE_SPEC.getB().scalarMultiply(scalar);
	//   byte[] buf = point.toByteArray();
	//   byte[] xpub = new byte[64];
	//   System.arraycopy(buf,0,xpub,0,32);
	//   System.arraycopy(toBytes(),32,xpub,32,32);
	//   return new EdPublicKey(xpub);
	// }

	// What really happens is:
	priv := ed25519.NewKeyFromSeed(seed)
	return priv.Public().(ed25519.PublicKey), nil
}

func NewBitpieRoot(seed []byte) (*Node, error) {
	masterKey, _, err := newBitpieMasterKey(seed)
	if err != nil {
		return nil, err
	}

	// Per Bitpie, the master key is just used as is, and the chainCode
	// is all 0s.
	rootNode := &Node{
		isRoot: true,
		isBitpie: true,
	}
	copy(rootNode.kL[:], masterKey)

	return rootNode, nil
}

func newBitpieMasterKey(seed []byte) ([]byte, []byte, error) {
	// Use the ECDSA curve to derive the master private key of ROSE,
	// derive path `m/44'/474‘/0’`
	key, err := bip32.NewMasterKey(seed)
	if err != nil {
		return nil, nil, fmt.Errorf("bip32: failed to derive bitpie master key root: %w", err)
	}

	for _, idx := range []uint32{44, 474, 0} {
		key, err = key.NewChildKey(idx + HardenedIndexOffset)
		if err != nil {
			return nil, nil, fmt.Errorf("bip32: failed to derive bitpie master key child: %w", err)
		}
	}

	return key.Key, key.ChainCode, nil
}
