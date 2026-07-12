package sovereign_auth

import (
	"crypto/ed25519"
	"fmt"
	"time"

	"github.com/fxamacker/cbor/v2"
)

// PermissionToken represents the signed statement
type PermissionToken struct {
	Issuer   string         `cbor:"issuer"`
	Subject  string         `cbor:"subject"`
	Resource string         `cbor:"resource"`
	Action   string         `cbor:"action"`
	Expiry   *int64         `cbor:"expiry"` // Nullable in JS
	Metadata map[string]any `cbor:"metadata"`
}

// VerifyToken verifies a permission token
func VerifyToken(token *PermissionToken, signature []byte, issuerPubKey ed25519.PublicKey) error {
	// 1. Check Expiry
	if token.Expiry != nil {
		now := time.Now().UnixMilli() // JS uses Date.now() (milliseconds)
		if now > *token.Expiry {
			return fmt.Errorf("token expired")
		}
	}

	// 2. Check Issuer Identity matches Public Key
	if err := VerifyPublicKey(issuerPubKey, token.Issuer); err != nil {
		return fmt.Errorf("issuer key verification failed: %w", err)
	}

	// 3. Verify Signature over Canonical CBOR
	em, err := cbor.CanonicalEncOptions().EncMode()
	if err != nil {
		return err
	}

	encoded, err := em.Marshal(token)
	if err != nil {
		return fmt.Errorf("cbor encoding failed: %w", err)
	}

	if !ed25519.Verify(issuerPubKey, encoded, signature) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}
