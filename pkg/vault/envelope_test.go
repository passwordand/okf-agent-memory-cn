package vault_test

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/okf-memory/okf-agent-memory/pkg/vault"
)

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("failed to generate random key: %v", err)
	}

	plaintext := []byte("# Markdown Title\n\nThis is a zero-knowledge confidential note.")

	envelopeBytes, err := vault.EncryptPayload(plaintext, key)
	if err != nil {
		t.Fatalf("unexpected encryption error: %v", err)
	}

	// Verify binary layout: 1 byte version + 12 bytes nonce + len(plaintext) ciphertext + 16 bytes tag
	expectedLen := 1 + 12 + len(plaintext) + 16
	if len(envelopeBytes) != expectedLen {
		t.Fatalf("expected envelope length %d, got %d", expectedLen, len(envelopeBytes))
	}

	if envelopeBytes[0] != vault.EnvelopeVersionV1 {
		t.Fatalf("expected version 0x01, got 0x%02x", envelopeBytes[0])
	}

	decrypted, err := vault.DecryptPayload(envelopeBytes, key)
	if err != nil {
		t.Fatalf("unexpected decryption error: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("decrypted text does not match original plaintext:\ngot:  %s\nwant: %s",
			string(decrypted), string(plaintext))
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	keyA := make([]byte, 32)
	keyB := make([]byte, 32)
	keyA[0] = 1
	keyB[0] = 2

	plaintext := []byte("Top secret agent memory")
	envelopeBytes, err := vault.EncryptPayload(plaintext, keyA)
	if err != nil {
		t.Fatalf("encryption error: %v", err)
	}

	_, err = vault.DecryptPayload(envelopeBytes, keyB)
	if err == nil {
		t.Fatalf("expected decryption error with wrong key, got nil")
	}

	if !errors.Is(err, vault.ErrAuthFailed) {
		t.Fatalf("expected ErrAuthFailed, got: %v", err)
	}
}

func TestTamperDetection(t *testing.T) {
	key := make([]byte, 32)
	plaintext := []byte("Tamper resistance test payload")

	envelopeBytes, err := vault.EncryptPayload(plaintext, key)
	if err != nil {
		t.Fatalf("encryption error: %v", err)
	}

	// 1. Tamper with Version (byte 0)
	tamperedVersion := bytes.Clone(envelopeBytes)
	tamperedVersion[0] = 0x99
	_, err = vault.DecryptPayload(tamperedVersion, key)
	if err == nil || (!errors.Is(err, vault.ErrUnsupportedVersion) && !errors.Is(err, vault.ErrAuthFailed)) {
		t.Fatalf("expected error on tampered version, got: %v", err)
	}

	// 2. Tamper with Nonce (bytes 1..12)
	tamperedNonce := bytes.Clone(envelopeBytes)
	tamperedNonce[5] ^= 0x01
	_, err = vault.DecryptPayload(tamperedNonce, key)
	if !errors.Is(err, vault.ErrAuthFailed) {
		t.Fatalf("expected ErrAuthFailed on tampered nonce, got: %v", err)
	}

	// 3. Tamper with Ciphertext body
	tamperedCipher := bytes.Clone(envelopeBytes)
	tamperedCipher[15] ^= 0x01
	_, err = vault.DecryptPayload(tamperedCipher, key)
	if !errors.Is(err, vault.ErrAuthFailed) {
		t.Fatalf("expected ErrAuthFailed on tampered ciphertext, got: %v", err)
	}

	// 4. Tamper with Auth Tag (last byte)
	tamperedTag := bytes.Clone(envelopeBytes)
	tamperedTag[len(tamperedTag)-1] ^= 0x01
	_, err = vault.DecryptPayload(tamperedTag, key)
	if !errors.Is(err, vault.ErrAuthFailed) {
		t.Fatalf("expected ErrAuthFailed on tampered tag, got: %v", err)
	}
}

func TestPayloadLengths(t *testing.T) {
	key := make([]byte, 32)
	sizes := []int{0, 1, 15, 16, 17, 1024, 65536}

	for _, size := range sizes {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i % 256)
		}

		envelopeBytes, err := vault.EncryptPayload(data, key)
		if err != nil {
			t.Fatalf("failed to encrypt payload of size %d: %v", size, err)
		}

		decrypted, err := vault.DecryptPayload(envelopeBytes, key)
		if err != nil {
			t.Fatalf("failed to decrypt payload of size %d: %v", size, err)
		}

		if !bytes.Equal(decrypted, data) {
			t.Fatalf("mismatch for payload size %d", size)
		}
	}
}

func TestInvalidInputs(t *testing.T) {
	key := make([]byte, 32)

	// Truncated envelope (< 29 bytes)
	shortBytes := make([]byte, 28)
	_, err := vault.DecryptPayload(shortBytes, key)
	if !errors.Is(err, vault.ErrInvalidEnvelope) {
		t.Fatalf("expected ErrInvalidEnvelope, got: %v", err)
	}

	// Invalid key length (!= 32)
	_, err = vault.EncryptPayload([]byte("hello"), make([]byte, 16))
	if !errors.Is(err, vault.ErrInvalidKeyLength) {
		t.Fatalf("expected ErrInvalidKeyLength on Encrypt, got: %v", err)
	}

	_, err = vault.DecryptPayload(make([]byte, 35), make([]byte, 16))
	if !errors.Is(err, vault.ErrInvalidKeyLength) {
		t.Fatalf("expected ErrInvalidKeyLength on Decrypt, got: %v", err)
	}
}

func TestHashBlobAndHashPlaintext(t *testing.T) {
	content := []byte("Hello world test content")

	hash1 := vault.HashPlaintext(content)
	expectedHex := sha256.Sum256(content)
	if hash1 != hex.EncodeToString(expectedHex[:]) {
		t.Fatalf("expected plaintext hash %s, got %s", hex.EncodeToString(expectedHex[:]), hash1)
	}

	blobHash := vault.HashBlob(content)
	if blobHash != hex.EncodeToString(expectedHex[:]) {
		t.Fatalf("expected blob hash %s, got %s", hex.EncodeToString(expectedHex[:]), blobHash)
	}
}

func TestParseEnvelopeStruct(t *testing.T) {
	key := make([]byte, 32)
	plaintext := []byte("Structured envelope test")

	envelopeBytes, err := vault.EncryptPayload(plaintext, key)
	if err != nil {
		t.Fatalf("encryption error: %v", err)
	}

	env, err := vault.ParseEnvelope(envelopeBytes)
	if err != nil {
		t.Fatalf("unexpected ParseEnvelope error: %v", err)
	}

	if env.Version != vault.EnvelopeVersionV1 {
		t.Fatalf("expected version 0x01, got 0x%02x", env.Version)
	}
	if len(env.Nonce) != 12 {
		t.Fatalf("expected 12-byte nonce, got %d", len(env.Nonce))
	}
	if len(env.Ciphertext) != len(plaintext) {
		t.Fatalf("expected %d-byte ciphertext, got %d", len(plaintext), len(env.Ciphertext))
	}
	if len(env.Tag) != 16 {
		t.Fatalf("expected 16-byte tag, got %d", len(env.Tag))
	}

	// Check roundtrip reconstruction via env.Bytes()
	if !bytes.Equal(env.Bytes(), envelopeBytes) {
		t.Fatalf("env.Bytes() does not match original envelope bytes")
	}
}
