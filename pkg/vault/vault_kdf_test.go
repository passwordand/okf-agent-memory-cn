package vault_test

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/okf-memory/okf-agent-memory/pkg/vault"
)

func TestDeriveVaultKey_Deterministic(t *testing.T) {
	password := "super-secret-password"
	// 16 bytes of entropy: "0123456789abcdef" -> Base32 unpadded is 26 chars
	rawSecret := []byte("0123456789abcdef")
	secretKey := vault.FormatSecretKey(rawSecret)

	key1, err := vault.DeriveVaultKey(password, secretKey)
	if err != nil {
		t.Fatalf("unexpected error deriving key1: %v", err)
	}

	const expectedHex = "0305ac72c04d5f7c0020da5c22b6e37d6de8fecde572e7106640e3dc76b056ee"
	if hex.EncodeToString(key1) != expectedHex {
		t.Fatalf("key derivation test vector mismatch:\nexpected: %s\ngot:      %s",
			expectedHex, hex.EncodeToString(key1))
	}

	if len(key1) != 32 {
		t.Fatalf("expected 32-byte key, got %d bytes", len(key1))
	}

	// Case-insensitivity and formatting (spaces, lower-case) test
	secretKeyVariant := "   " + strings.ToLower(secretKey) + "   "
	key2, err := vault.DeriveVaultKey(password, secretKeyVariant)
	if err != nil {
		t.Fatalf("unexpected error deriving key2: %v", err)
	}

	if !bytes.Equal(key1, key2) {
		t.Fatalf("expected key derivation to be insensitive to case/spacing, but keys differ:\nkey1: %s\nkey2: %s",
			hex.EncodeToString(key1), hex.EncodeToString(key2))
	}
}

func TestDeriveVaultKey_DifferentInputsProduceDifferentKeys(t *testing.T) {
	secretKey := vault.FormatSecretKey([]byte("0123456789abcdef"))

	keyA, err := vault.DeriveVaultKey("password-A", secretKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	keyB, err := vault.DeriveVaultKey("password-B", secretKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if bytes.Equal(keyA, keyB) {
		t.Fatalf("different passwords should produce different keys")
	}
}

func TestDeriveVaultKey_Validation(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		secretKey string
		wantErr   error
	}{
		{
			name:      "empty password",
			password:  "",
			secretKey: "MFRGGZDFMZTWQ2LKNNWG23TPOBYXE43U",
			wantErr:   vault.ErrEmptyPassword,
		},
		{
			name:      "empty secret key",
			password:  "password",
			secretKey: "",
			wantErr:   vault.ErrInvalidSecretKey,
		},
		{
			name:      "invalid base32 secret key",
			password:  "password",
			secretKey: "!!!invalid-base32!!!",
			wantErr:   vault.ErrInvalidSecretKey,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := vault.DeriveVaultKey(tc.password, tc.secretKey)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if tc.wantErr != nil && err != tc.wantErr {
				t.Fatalf("expected error %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestSecretKeyGenerationAndFormatting(t *testing.T) {
	secretKey, err := vault.GenerateSecretKey()
	if err != nil {
		t.Fatalf("failed to generate secret key: %v", err)
	}

	raw, err := vault.ParseSecretKey(secretKey)
	if err != nil {
		t.Fatalf("failed to parse generated secret key %q: %v", secretKey, err)
	}

	if len(raw) != 16 {
		t.Fatalf("expected 16 bytes (128-bit) secret key, got %d bytes", len(raw))
	}

	// Parsing with dashes and lowercase should work
	formatted := vault.FormatSecretKey(raw)
	parsedFormatted, err := vault.ParseSecretKey(formatted)
	if err != nil {
		t.Fatalf("failed to parse formatted key: %v", err)
	}

	if !bytes.Equal(raw, parsedFormatted) {
		t.Fatalf("parsed key does not match original raw bytes")
	}
}
