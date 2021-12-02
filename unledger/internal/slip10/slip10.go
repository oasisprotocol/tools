// Package slip10 implements the SLIP-0010 master key derivation scheme
// for Ed25519.
package slip10

import (
	"crypto/hmac"
	"crypto/sha512"
	"fmt"
)

const (
	// SeedMinSize is the minimum seed byte sequence size in bytes.
	SeedMinSize = 16

	// SeedMaxSize is the maximum seed byte sequence size in bytes.
	SeedMaxSize = 64

	// ChainCodeSize is the size of a SLIP-0010 chain code in bytes.
	ChainCodeSize = 32

	// MasterSecretSize is the size of a SLIP-0010 master secret in bytes.
	MasterSecretSize = 32
)

// CurveConstant is the SLIP-0010 curve constant.
var CurveConstant = []byte("ed25519 seed")

// ChainCode is a SLIP-0010 chain code.
type ChainCode [ChainCodeSize]byte

// MasterSecret is a SLIP-0010 master secret.
type MasterSecret [MasterSecretSize]byte

// NewMasterKey derives a master key and chain code from a seed byte sequence.
func NewMasterKey(seed []byte) (MasterSecret, ChainCode, error) {
	// Let S be a seed byte sequence of 128 to 512 bits in length.
	if sLen := len(seed); sLen < SeedMinSize || sLen > SeedMaxSize {
		return MasterSecret{}, ChainCode{}, fmt.Errorf("slip10: invalid seed")
	}

	// 1. Calculate I = HMAC-SHA512(Key = Curve, Data = S)
	mac := hmac.New(sha512.New, CurveConstant)
	_, _ = mac.Write(seed)
	I := mac.Sum(nil)

	// 2. Split I into two 32-byte sequences, IL and IR.
	// 3. Use parse256(IL) as master secret key, and IR as master chain code.
	IL, IR := I[:32], I[32:]
	var (
		masterSecret MasterSecret
		chainCode    ChainCode
	)
	copy(masterSecret[:], IL)
	copy(chainCode[:], IR)

	return masterSecret, chainCode, nil
}
