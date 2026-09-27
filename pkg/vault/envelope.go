package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

// Envelope binary format constants:
// [Version: 1 Byte][Nonce: 12 Bytes][Ciphertext: n Bytes][Auth Tag: 16 Bytes]
const (
	// EnvelopeVersionV1 is the binary envelope format version 1.
	EnvelopeVersionV1 byte = 0x01
	// NonceLength is 12 bytes (96 bits) for AES-GCM.
	NonceLength = 12
	// TagLength is 16 bytes (128 bits) for AES-GCM authentication tag.
	TagLength = 16
	// MinEnvelopeLength is 1 byte version + 12 bytes nonce + 0 bytes ciphertext + 16 bytes tag.
	MinEnvelopeLength = 1 + NonceLength + TagLength
)

var (
	// ErrAuthFailed indicates the ciphertext authentication tag was invalid or tampered with.
	ErrAuthFailed = errors.New("vault: authentication failed / ciphertext tampered or wrong key")
	// ErrInvalidKeyLength indicates the provided encryption key is not exactly 32 bytes (256 bits).
	ErrInvalidKeyLength = errors.New("vault: key must be 32 bytes for AES-256")
	// ErrInvalidEnvelope indicates the envelope binary data is too short or malformed.
	ErrInvalidEnvelope = errors.New("vault: envelope too short or malformed")
	// ErrUnsupportedVersion indicates the envelope version byte is not recognized.
	ErrUnsupportedVersion = errors.New("vault: unsupported envelope version")
)

// Envelope represents an unpacked AES-256-GCM encrypted envelope.
type Envelope struct {
	Version    byte
	Nonce      []byte // 12 bytes
	Ciphertext []byte // len(data) bytes
	Tag        []byte // 16 bytes
}

// Bytes serializes the envelope into its canonical binary layout:
// [Version: 1B][Nonce: 12B][Ciphertext: nB][Tag: 16B].
func (e *Envelope) Bytes() []byte {
	out := make([]byte, 1+len(e.Nonce)+len(e.Ciphertext)+len(e.Tag))
	out[0] = e.Version
	copy(out[1:1+len(e.Nonce)], e.Nonce)
	copy(out[1+len(e.Nonce):1+len(e.Nonce)+len(e.Ciphertext)], e.Ciphertext)
	copy(out[1+len(e.Nonce)+len(e.Ciphertext):], e.Tag)
	return out
}

// ParseEnvelope parses a binary envelope stream into its structured components.
func ParseEnvelope(data []byte) (*Envelope, error) {
	if len(data) < MinEnvelopeLength {
		return nil, ErrInvalidEnvelope
	}

	version := data[0]
	nonce := make([]byte, NonceLength)
	copy(nonce, data[1:1+NonceLength])

	cipherLen := len(data) - 1 - NonceLength - TagLength
	ciphertext := make([]byte, cipherLen)
	copy(ciphertext, data[1+NonceLength:1+NonceLength+cipherLen])

	tag := make([]byte, TagLength)
	copy(tag, data[1+NonceLength+cipherLen:])

	return &Envelope{
		Version:    version,
		Nonce:      nonce,
		Ciphertext: ciphertext,
		Tag:        tag,
	}, nil
}

// EncryptPayload encrypts arbitrary plaintext bytes using AES-256-GCM and wraps
// it into the binary envelope format: [Version: 1B][Nonce: 12B][Ciphertext: nB][Tag: 16B].
func EncryptPayload(plaintext, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKeyLength
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("vault: failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("vault: failed to create GCM: %w", err)
	}

	nonce := make([]byte, NonceLength)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("vault: failed to generate nonce: %w", err)
	}

	// Authenticate the 1-byte version header in the GCM additional data
	additionalData := []byte{EnvelopeVersionV1}
	sealed := gcm.Seal(nil, nonce, plaintext, additionalData)

	// Format: [0x01][12B Nonce][Sealed: Ciphertext + 16B Tag]
	result := make([]byte, 1+NonceLength+len(sealed))
	result[0] = EnvelopeVersionV1
	copy(result[1:1+NonceLength], nonce)
	copy(result[1+NonceLength:], sealed)

	return result, nil
}

// DecryptPayload decrypts a binary envelope [Version: 1B][Nonce: 12B][Ciphertext: nB][Tag: 16B]
// using the provided 256-bit key. It returns ErrAuthFailed if the ciphertext was tampered with
// or if the key is incorrect.
func DecryptPayload(envelopeBytes, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKeyLength
	}

	if len(envelopeBytes) < MinEnvelopeLength {
		return nil, ErrInvalidEnvelope
	}

	version := envelopeBytes[0]
	if version != EnvelopeVersionV1 {
		return nil, ErrUnsupportedVersion
	}

	nonce := envelopeBytes[1 : 1+NonceLength]
	sealed := envelopeBytes[1+NonceLength:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("vault: failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("vault: failed to create GCM: %w", err)
	}

	additionalData := []byte{version}
	plaintext, err := gcm.Open(nil, nonce, sealed, additionalData)
	if err != nil {
		return nil, ErrAuthFailed
	}

	return plaintext, nil
}

// HashBlob computes the deterministic SHA-256 hex digest of an encrypted binary envelope.
// This is the CAS (Content-Addressed Storage) key under which the blob is stored.
func HashBlob(envelopeBytes []byte) string {
	sum := sha256.Sum256(envelopeBytes)
	return hex.EncodeToString(sum[:])
}

// HashPlaintext computes the deterministic SHA-256 hex digest of plaintext content.
// This is stored in the Tree manifest for fast, local change detection without re-encryption.
func HashPlaintext(plaintext []byte) string {
	sum := sha256.Sum256(plaintext)
	return hex.EncodeToString(sum[:])
}
