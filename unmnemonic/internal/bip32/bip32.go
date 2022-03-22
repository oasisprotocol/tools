// Package bip32 implements the various screwed up legacy Oasis variants of
// BIP32-Ed25519.
package bip32

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/oasisprotocol/curve25519-voi/curve"
	"github.com/oasisprotocol/curve25519-voi/curve/scalar"
	"github.com/oasisprotocol/curve25519-voi/primitives/ed25519"

	"github.com/oasisprotocol/tools/unmnemonic/internal/slip10"
)

const (
	// HardenedIndexOffset is the offset added to derivation indexes to indicate
	// that the hardened formula should be used.
	HardenedIndexOffset = 1 << 31

	hardenedSuffix = "'"
)

var (
	// ErrDivisibleByBaseOrder is the error returned when the derived child
	// is a multiple of the scalar group order.
	ErrDivisibleByBaseOrder = errors.New("bip32: kL divisible by basepoint order")

	scalarZero scalar.Scalar
)

// Node is a HKD derivation node.
type Node struct {
	kL [32]byte
	kR [32]byte
	c  [32]byte

	isRoot bool
}

// GetLedgerPrivateKey returns the Oasis network private key associated
// with a node, derived using the legacy Ledger method.
//
// Just use Ristretto, but using kL as the RFC 8032 seeds is marginally
// less bad than BIP-Ed25519.
func (n *Node) GetLedgerPrivateKey() ed25519.PrivateKey {
	return ed25519.NewKeyFromSeed(n.kL[:])
}

// GetExtendedPrivateKey returns the Oasis network private key associated
// with a node, in the BIP-Ed25519 extended format.
func (n *Node) GetExtendedPrivateKey() []byte {
	var r []byte
	r = append([]byte{}, n.kL[:]...)
	r = append(r, n.kR[:]...)
	return r
}

// DeriveChild derives a sub-key with the provided index.
func (n *Node) DeriveChild(idx uint32) (*Node, error) {
	// BIP32-Ed25519 is a steaming pile of shit, and really shouldn't be
	// used at all, which is why we don't use it.  Unfortunately legacy
	// applications exist.
	//
	// * Ledger uses k_L as the RFC 8032 seed, and only uses hardened paths.

	isHardened := idx >= HardenedIndexOffset

	var aBytes []byte
	if !isHardened {
		// Non-hardened derivation requires feeding A_P into the HMAC
		// calls, so we need to derive it.
		pk, err := ScalarToPublicKey(n.kL[:])
		if err != nil {
			return nil, err
		}

		aBytes = pk
	}

	var iBytes [4]byte
	binary.LittleEndian.PutUint32(iBytes[:], idx)

	zMac := hmac.New(sha512.New, n.c[:])
	switch isHardened {
	case true:
		// Z = FcP (0x00 || k_P || i), i >= 2^31
		_, _ = zMac.Write([]byte{0x00})
		_, _ = zMac.Write(n.kL[:])
		_, _ = zMac.Write(n.kR[:])
	case false:
		// Z = FcP (0x02 || A_P || i), i < 2^31
		_, _ = zMac.Write([]byte{0x02})
		_, _ = zMac.Write(aBytes)
	}
	_, _ = zMac.Write(iBytes[:])
	z := zMac.Sum(nil)

	cMac := hmac.New(sha512.New, n.c[:])
	switch isHardened {
	case true:
		// ci = FcP (0x01 || k_P || i), i >= 2^31
		_, _ = cMac.Write([]byte{0x01})
		_, _ = cMac.Write(n.kL[:])
		_, _ = cMac.Write(n.kR[:])
	case false:
		// ci = FcP (0x03 || A_P || i), i < 2^31
		_, _ = cMac.Write([]byte{0x03})
		_, _ = cMac.Write(aBytes)
	}
	_, _ = cMac.Write(iBytes[:])
	c := cMac.Sum(nil)
	var childNode Node
	copy(childNode.c[:], c[32:]) // where the output of F is truncated to the right 32 bytes.

	// ZL, ZR = Z[:28], Z[32:]
	copy(childNode.kL[:], z[:28]) // left 28-bytes
	copy(childNode.kR[:], z[32:]) // right 32-bytes

	// Sigh, deriving kL and kR needs to be done explicitly not modulo
	// the group order.

	// kL = 8[ZL] + [k_P_L]
	var carry uint16
	for i := 0; i < 32; i++ {
		tmp := 8*uint16(childNode.kL[i]) + uint16(n.kL[i]) + carry
		childNode.kL[i] = byte(tmp & 0xff)
		carry = tmp >> 8
	}

	// kR = [ZR] + [k_P_R] mod 2^256
	carry = 0
	for i := 0; i < 32; i++ {
		tmp := uint16(childNode.kR[i]) + uint16(n.kR[i]) + carry
		childNode.kR[i] = byte(tmp & 0xff)
		carry = tmp >> 8
	}

	// If kL is divisible by the base order n, discard the child.
	//
	// This is implemented by doing a wide reduction modulo the group
	// order and checking if the result is zero.
	var klWide [scalar.ScalarWideSize]byte
	copy(klWide[:], childNode.kL[:])
	var kL scalar.Scalar
	if _, err := kL.SetBytesModOrderWide(klWide[:]); err != nil {
		return nil, fmt.Errorf("bip32: failed to deserialize kL (wide): %w", err)
	}
	if kL.Equal(&scalarZero) == 1 {
		return nil, ErrDivisibleByBaseOrder
	}

	return &childNode, nil
}

// DerivePath derives the node associated with a path.
func (n *Node) DerivePath(path string) (*Node, error) {
	if !n.isRoot {
		return nil, fmt.Errorf("bip32: base node is not the root")
	}

	splitPath := strings.Split(path, "/")
	indices := make([]uint32, 0, len(splitPath))
	for _, pathEntry := range splitPath {
		isHardened := strings.HasSuffix(pathEntry, hardenedSuffix)
		s := strings.TrimSuffix(pathEntry, hardenedSuffix)
		i, err := strconv.ParseUint(s, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("bip32: path component '%s': %w", pathEntry, err)
		}
		if i >= HardenedIndexOffset {
			return nil, fmt.Errorf("bip32: path component '%s': out of range", pathEntry)
		}
		if isHardened {
			i += HardenedIndexOffset
		}
		indices = append(indices, uint32(i))
	}
	if len(indices) == 0 {
		return nil, fmt.Errorf("bip32: rejecting 0-length path")
	}
	if len(indices) > 1048576 {
		return nil, fmt.Errorf("bip32: path length over permitted maximum")
	}

	var err error
	ret := n
	for _, idx := range indices {
		ret, err = ret.DeriveChild(idx)
		if err != nil {
			return nil, fmt.Errorf("bip32: failed to derive child %d': %w", idx, err)
		}
	}

	return ret, nil
}

// NewLedgerRoot returns the root (master) node, corresponding to the
// provided seed, using the fucked up Ledger BIP32-Ed25519 variant.
func NewLedgerRoot(seed []byte) (*Node, error) {
	sTmp := append([]byte{}, seed...) // Copy

	var (
		n   Node
		err error
	)

	// The example code I have been told is the Ledger app's way of
	// doing things, does this once and only once, regardless of how
	// many attempts it takes to find an acceptable k.
	//
	// This disagrees with BIP32-Ed25519, which specifies
	// `c = SHA256(0x01 || k)`, with the k that is accepted.
	//
	// I will take the test code for it's word for now, because the
	// test cases exercise the re-derivation path.
	mac := hmac.New(sha256.New, slip10.CurveConstant)
	_, _ = mac.Write([]byte{0x01})
	_, _ = mac.Write(seed)
	c := mac.Sum(nil)
	copy(n.c[:], c)

	// BIP32-Ed25519: If the third highest bit of the last byte of kL is
	// not zero, discard k.
	for {
		var (
			kL *slip10.Secret
			kR *slip10.ChainCode
		)
		if kL, kR, err = slip10.NewMasterKey(sTmp); err != nil {
			return nil, err
		}
		copy(n.kL[:], kL[:])
		copy(n.kR[:], kR[:])

		if n.kL[31]&0x20 != 0x20 { // ~0b00100000
			break
		}

		// The ledger app iterateively calls the SLIP-10 master secret
		// derivation with kL | kR, if k would be discarded.
		sTmp = append([]byte{}, n.kL[:]...)
		sTmp = append(sTmp, n.kR[:]...)
	}

	// BIP32-Ed25519 requires that the scalar clamping is applied
	// to kL (Master secret) as in vanilla Ed25519pure.
	clampScalar(n.kL[:])

	n.isRoot = true

	return &n, nil
}

// NewRoot returns the root (master) node, corresponding to the provided
// seed.
func NewRoot(seed []byte) (*Node, error) {
	// k' = H512(k)
	kPrime := sha512.Sum512(seed)

	var n Node
	copy(n.kL[:], kPrime[:32])
	copy(n.kR[:], kPrime[32:])

	// If the third highest bit of the last byte of kL is not zero,
	// discard k'.
	if n.kL[31]&0x20 == 0x20 { // ~0b00100000
		return nil, fmt.Errorf("bip32: invalid k")
	}

	// BIP32-Ed25519 requires that the scalar clamping is applied
	// to kL (Master secret) as in vanilla Ed25519pure.
	clampScalar(n.kL[:])

	// Derive c = H256(0x01 | k).
	h := sha256.New()
	_, _ = h.Write([]byte{0x01})
	_, _ = h.Write(seed)
	c := h.Sum(nil)
	copy(n.c[:], c)

	n.isRoot = true

	return &n, nil
}

// ScalarToPublicKey converts a scalar to a public key.
func ScalarToPublicKey(rawScalar []byte) (ed25519.PublicKey, error) {
	if l := len(rawScalar); l != scalar.ScalarSize {
		return nil, fmt.Errorf("bip32: invalid scalar lenght: %v", l)
	}

	var s scalar.Scalar
	if _, err := s.SetBits(clampScalar(append([]byte{}, rawScalar...))); err != nil {
		return nil, fmt.Errorf("bip32: failed to deserialize scalar: %w", err)
	}

	var (
		A           curve.EdwardsPoint
		aCompressed curve.CompressedEdwardsY
	)
	aCompressed.SetEdwardsPoint(A.MulBasepoint(curve.ED25519_BASEPOINT_TABLE, &s))

	return ed25519.PublicKey(aCompressed[:]), nil
}

func clampScalar(s []byte) []byte {
	s[0] &= 248
	s[31] &= 127
	s[31] |= 64
	return s
}
