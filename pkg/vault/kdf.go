// Package vault implements the core zero-knowledge encryption, envelope packaging,
// and tree/commit manifest data structures for OKF Memory Hub vaults.
package vault

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2id KDF parameters matching OKF Memory Hub specification.
const (
	// ArgonMemory is 64 MB (expressed in KiB for golang.org/x/crypto/argon2).
	ArgonMemory uint32 = 64 * 1024
	// ArgonIterations is the number of passes over the memory.
	ArgonIterations uint32 = 3
	// ArgonParallelism is the number of threads/lanes used.
	ArgonParallelism uint8 = 4
	// KeyLength is 32 bytes (256 bits) for AES-256.
	KeyLength uint32 = 32
	// SecretKeyBytes is 16 bytes (128 bits) of cryptographic entropy.
	SecretKeyBytes = 16
)

var (
	// ErrEmptyPassword indicates that the provided master password is empty.
	ErrEmptyPassword = errors.New("vault: password cannot be empty")
	// ErrInvalidSecretKey indicates that the provided secret key is malformed or invalid.
	ErrInvalidSecretKey = errors.New("vault: invalid secret key")
)

var b32Encoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// GenerateSecretKey generates a cryptographically secure 128-bit secret key
// formatted as a human-readable, dash-separated Base32 string.
func GenerateSecretKey() (string, error) {
	raw := make([]byte, SecretKeyBytes)
	if _, err := io.ReadFull(rand.Reader, raw); err != nil {
		return "", fmt.Errorf("vault: failed to generate secret key entropy: %w", err)
	}
	return FormatSecretKey(raw), nil
}

// FormatSecretKey formats 16 bytes of entropy as grouped Base32 chunks separated by dashes.
func FormatSecretKey(raw []byte) string {
	encoded := b32Encoding.EncodeToString(raw)
	// Group into blocks of 4 characters: XXXX-XXXX-XXXX-XXXX-XXXX-XXXX-XX
	var parts []string
	for len(encoded) > 4 {
		parts = append(parts, encoded[:4])
		encoded = encoded[4:]
	}
	if len(encoded) > 0 {
		parts = append(parts, encoded)
	}
	return strings.Join(parts, "-")
}

// ParseSecretKey cleans and decodes a dash-separated or raw Base32 secret key into 16 bytes.
func ParseSecretKey(s string) ([]byte, error) {
	cleaned := strings.ToUpper(strings.TrimSpace(s))
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	if cleaned == "" {
		return nil, ErrInvalidSecretKey
	}

	decoded, err := b32Encoding.DecodeString(cleaned)
	if err != nil {
		// Also try with standard padding in case user pasted padded string
		padded := strings.TrimRight(cleaned, "=")
		if decPadded, errPadded := b32Encoding.DecodeString(padded); errPadded == nil {
			decoded = decPadded
		} else {
			return nil, ErrInvalidSecretKey
		}
	}

	if len(decoded) != SecretKeyBytes {
		return nil, ErrInvalidSecretKey
	}

	return decoded, nil
}

// DeriveVaultKey derives a deterministic 256-bit (32-byte) AES key using Argon2id
// with parameters m=64MB, t=3, p=4, using the decoded 128-bit secretKey as salt.
func DeriveVaultKey(password, secretKey string) ([]byte, error) {
	if password == "" {
		return nil, ErrEmptyPassword
	}

	salt, err := ParseSecretKey(secretKey)
	if err != nil {
		return nil, ErrInvalidSecretKey
	}

	key := argon2.IDKey([]byte(password), salt, ArgonIterations, ArgonMemory, ArgonParallelism, KeyLength)
	return key, nil
}
