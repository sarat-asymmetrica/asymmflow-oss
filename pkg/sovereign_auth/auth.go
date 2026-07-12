package sovereign_auth

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"

	"github.com/fxamacker/cbor/v2"
)

// Identity Constants
const (
	NodeIDLength   = 20 // 160 bits
	ChecksumLength = 4  // 32 bits
	IdentityPrefix = "asym1:"
)

// Identity represents a parsed sovereign identity
type Identity struct {
	IdentityString string
	NodeID         []byte
	Checksum       []byte
}

// ParseIdentityString parses and validates a sovereign identity string
func ParseIdentityString(idStr string) (*Identity, error) {
	if !strings.HasPrefix(idStr, IdentityPrefix) {
		return nil, fmt.Errorf("invalid identity prefix: expected %s", IdentityPrefix)
	}

	encoded := idStr[len(IdentityPrefix):]

	// Use standard base32 decoder (RFC 4648)
	// Note: Noble Base32 uses lowercase, Go's StdEncoding uses uppercase.
	// We need a custom decoder or simply upper-case the input.
	decoder := base32.StdEncoding.WithPadding(base32.NoPadding)
	identityBytes, err := decoder.DecodeString(strings.ToUpper(encoded))
	if err != nil {
		// Try Hex encoding if base32 fails? No, specification says Base32.
		// Let's try custom alphabet if Noble uses a non-standard one.
		// Checking `encoding.js`: 'abcdefghijklmnopqrstuvwxyz234567' (RFC 4648 but lowercase)
		// So standard decoder with upper-case input is correct.
		return nil, fmt.Errorf("base32 decode failed: %w", err)
	}

	if len(identityBytes) != NodeIDLength+ChecksumLength {
		return nil, fmt.Errorf("invalid identity length: got %d, want %d", len(identityBytes), NodeIDLength+ChecksumLength)
	}

	nodeID := identityBytes[:NodeIDLength]
	checksum := identityBytes[NodeIDLength:]

	// Verify Checksum
	hash := sha256.Sum256(nodeID)
	expectedChecksum := hash[:ChecksumLength]

	for i := 0; i < ChecksumLength; i++ {
		if checksum[i] != expectedChecksum[i] {
			return nil, errors.New("identity checksum verification failed")
		}
	}

	return &Identity{
		IdentityString: idStr,
		NodeID:         nodeID,
		Checksum:       checksum,
	}, nil
}

// VerifyPublicKey checks if a public key matches an identity string
func VerifyPublicKey(pubKey ed25519.PublicKey, idStr string) error {
	// 1. Parse Identity
	identity, err := ParseIdentityString(idStr)
	if err != nil {
		return err
	}

	// 2. Hash Public Key
	hash := sha256.Sum256(pubKey)
	derivedNodeID := hash[:NodeIDLength]

	// 3. Compare Node IDs
	if len(derivedNodeID) != len(identity.NodeID) {
		return errors.New("node ID length mismatch")
	}
	for i := range derivedNodeID {
		if derivedNodeID[i] != identity.NodeID[i] {
			return errors.New("public key does not match identity")
		}
	}

	return nil
}

// VerifySignature verifies a signature against a public key and data
func VerifySignature(pubKey ed25519.PublicKey, data []byte, signature []byte) bool {
	return ed25519.Verify(pubKey, data, signature)
}

// VerifyCBORSignature verifies a signature over a CBOR encoded structure
// This ensures we use the exact same canonical encoding as the JS side
func VerifyCBORSignature(pubKey ed25519.PublicKey, object any, signature []byte) (bool, error) {
	// Create Canonical Encoder
	em, err := cbor.CanonicalEncOptions().EncMode()
	if err != nil {
		return false, err
	}

	// Encode
	encoded, err := em.Marshal(object)
	if err != nil {
		return false, err
	}

	// Verify
	return ed25519.Verify(pubKey, encoded, signature), nil
}
