// Package address implements the v0 staking account address derivation.
package address

import (
	"crypto"
	"crypto/sha512"
	"fmt"

	"github.com/btcsuite/btcutil/bech32"
	"github.com/oasisprotocol/curve25519-voi/primitives/ed25519"
)

// FromPublicKey returns the Oasis v0 staking address corresponding to the
// provided Ed25519 public key.
func FromPublicKey(pk crypto.PublicKey) (string, error) {
	const addrVersion = 0

	edPk := pk.(ed25519.PublicKey)

	h := sha512.New512_256()
	_, _ = h.Write([]byte("oasis-core/address: staking"))
	_, _ = h.Write([]byte{addrVersion})
	_, _ = h.Write(edPk[:])
	digest := h.Sum(nil)

	addr := append([]byte{addrVersion}, digest[:20]...)

	converted, err := bech32.ConvertBits(addr, 8, 5, true)
	if err != nil {
		return "", fmt.Errorf("address: failed to convert bits for bech32: %w", err)
	}
	return bech32.Encode("oasis", converted)
}
