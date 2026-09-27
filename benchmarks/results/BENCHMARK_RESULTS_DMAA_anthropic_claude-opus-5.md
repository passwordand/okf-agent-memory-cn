# 📊 Benchmark Report: Dual-Memory Agent Architecture (DMAA)

> Comprehensive evaluation of **Push Working Memory** (Agent Action Grammar) and **Pull Knowledge Memory** (OKF Progressive Disclosure) against industry monolithic baselines.

* **Provider**: `ANTHROPIC`
* **Model Tested**: `claude-opus-5`
* **Execution Mode**: `Remote Cloud API`
* **Hardware / Client**: `Apple M2 Pro (32 GB Unified Memory, macOS)`
* **Endpoint**: `https://api.anthropic.com/v1`
* **Temperature**: `0.10`
* **Date**: 2026-09-19 13:40:30

---

## 🚀 Executive Summary

| Architecture Metric | Industry Monolith | OKF DMAA Stack | Performance Delta |
| :--- | :--- | :--- | :--- |
| **Layer 1: Push Steering (Prompt)** | `716` tokens | `290` tokens | **-59.5% overhead** |
| **Layer 2: Pull Retrieval (Prompt)** | `3055` tokens | `627` tokens | **-79.5% overhead** |
| **Total Prompt Context Overhead** | `3771` tokens | `917` tokens | **🔥 -75.7% context tax** |
| **Average Prefill Latency (TTFT)** | `7318.2 ms` | `7876.2 ms` | **⚡ 0.9x faster TTFT** |
| **Total Turn Duration** | `65.08 s` | `84.79 s` | **0.8x faster completion** |
| **Global Constraint Adherence** | `5/9` (55.6%) | `9/9` (100.0%) | **+44.4% Adherence (100% Deterministic)** |

---

## 🎯 Layer 1: Push Working Memory (AAG vs. Conversational Prose)

Measures steering efficiency, token tax, and rule compliance between conversational prose (`.cursorrules`) and Agent Action Grammar (`AGENTS.md`).

| Metric | Conversational Prose (.cursorrules) | Agent Action Grammar (AGENTS.md) | Delta |
| :--- | :--- | :--- | :--- |
| **Input Tokens (Prompt)** | `716` tokens | `290` tokens | **-59.5%** |
| **Output Tokens (Generated)** | `0` tokens | `514` tokens | - |
| **Prefill Latency (TTFT)** | `7162.4 ms` | `11878.7 ms` | **0.6x faster** |
| **Turn Duration** | `7.16 s` | `42.29 s` | - |
| **Rule Adherence Score** | `1/5` | `5/5` | +80.0% Consistent |


### Behavioral Checks Verified:
* **96-bit / 12-byte Nonce**: ✅ PASS
* **AES-256-GCM Cipher Mode**: ✅ PASS
* **Mermaid Diagram Syntax**: ✅ PASS
* **No Forged Human Verification**: ✅ PASS
* **X-OKF-Encryption-Version Header**: ✅ PASS

<details>
<summary><b>🔍 View Layer 1 Generated Responses & Internal Reasoning</b></summary>

### Run 1: Conversational Prose (.cursorrules)

> [!WARNING]
> **No output generated:** The model did not output a final response block.

### Run 2: Agent Action Grammar (AGENTS.md)

## Scope gate (read first)

Two AAG preconditions apply before any file write:

1. **Path required.** `ON edit(@path/)` with `first_visit` triggers `okf_search(for_path=@path/)`. I have no repo path or tooling in this session, so the code below is delivered **inline, unwritten**. Give me the target path (e.g. `@internal/crypto/payload/`) and I'll run the search before committing.
2. **`@internal/crypto/legacy` is out of bounds.** If this module is meant to replace or patch anything under that subsystem, governance is `hold` — I stop and you confirm. The code below assumes a **new** module, no legacy edits.

---

## 1. Data flow

```mermaid
flowchart TD
    A["Caller: sensitive customer payload"] --> B["Validate key (32 bytes / AES-256)"]
    B -->|"invalid length"| E1["Error: ErrInvalidKeySize"]
    B -->|"valid"| C["Generate 96-bit nonce (crypto/rand)"]
    C -->|"rand failure"| E2["Error: nonce generation failed"]
    C --> D["AES-256-GCM Seal(plaintext, nonce, AAD)"]
    D --> F["Ciphertext + 128-bit auth tag"]
    F --> G["Build envelope: nonce || ciphertext||tag"]
    G --> H["Attach header: 'X-OKF-Encryption-Version: v2'"]
    H --> I["Transport / storage"]
    I --> J["Decrypt: split nonce, verify tag, Open"]
    J -->|"tag mismatch"| E3["Error: authentication failed (discard)"]
    J -->|"ok"| K["Plaintext returned to caller"]
```

---

## 2. Implementation

```go
// Package payloadcrypto implements authenticated encryption for sensitive
// customer payloads using AES-256-GCM with a 96-bit random nonce.
//
// Wire format (Envelope.Sealed):
//
//	| nonce (12 bytes) | ciphertext (n bytes) | auth tag (16 bytes) |
//
// Every emitted envelope carries the header
// "X-OKF-Encryption-Version: v2".
package payloadcrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

const (
	// EncryptionVersionHeader is the mandatory version header name.
	EncryptionVersionHeader = "X-OKF-Encryption-Version"
	// EncryptionVersion is the current envelope version.
	EncryptionVersion = "v2"

	// KeySize is the AES-256 key length in bytes.
	KeySize = 32
	// NonceSize is the GCM nonce length in bytes (96 bits).
	NonceSize = 12
	// TagSize is the GCM authentication tag length in bytes (128 bits).
	TagSize = 16
)

var (
	ErrInvalidKeySize   = errors.New("payloadcrypto: key must be exactly 32 bytes (AES-256)")
	ErrEmptyPlaintext   = errors.New("payloadcrypto: plaintext must not be empty")
	ErrMalformedEnvelope = errors.New("payloadcrypto: envelope too short or malformed")
	ErrUnsupportedVersion = errors.New("payloadcrypto: unsupported encryption version")
	ErrAuthentication   = errors.New("payloadcrypto: authentication failed; payload rejected")
)

// Envelope is the transport-ready result of Encrypt.
type Envelope struct {
	// Headers always contains X-OKF-Encryption-Version: v2.
	Headers map[string]string
	// Sealed is nonce || ciphertext || tag.
	Sealed []byte
}

// Header returns the version header pair for callers wiring this into an
// existing HTTP/message header set.
func Header() (string, string) {
	return EncryptionVersionHeader, EncryptionVersion
}

// Encrypt seals plaintext under key using AES-256-GCM.
//
// key must be 32 bytes. aad (additional authenticated data) is authenticated
// but not encrypted; pass nil if unused. A fresh 96-bit nonce is drawn from
// crypto/rand for every call — never reuse a nonce with the same key.
func Encrypt(key, plaintext, aad []byte) (*Envelope, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKeySize
	}
	if len(plaintext) == 0 {
		return nil, ErrEmptyPlaintext
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("payloadcrypto: cipher init: %w", err)
	}

	aead, err := cipher.NewGCMWithNonceSize(block, NonceSize)
	if err != nil {
		return nil, fmt.Errorf("payloadcrypto: gcm init: %w", err)
	}

	// Allocate one buffer: nonce prefix + room for ciphertext and tag.
	out := make([]byte, NonceSize, NonceSize+len(plaintext)+aead.Overhead())
	nonce := out[:NonceSize]
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("payloadcrypto: nonce generation: %w", err)
	}

	// Seal appends ciphertext||tag directly after the nonce prefix.
	sealed := aead.Seal(out, nonce, plaintext, aad)

	return &Envelope{
		Headers: map[string]string{EncryptionVersionHeader: EncryptionVersion},
		Sealed:  sealed,
	}, nil
}

// Decrypt opens an envelope produced by Encrypt.
//
// version must be the value of the X-OKF-Encryption-Version header received
// alongside sealed. aad must match the value supplied at encryption time.
func Decrypt(key, sealed, aad []byte, version string) ([]byte, error) {
	if version != EncryptionVersion {
		return nil, fmt.Errorf("%w: got %q, want %q", ErrUnsupportedVersion, version, EncryptionVersion)
	}
	if len(key) != KeySize {
		return nil, ErrInvalidKeySize
	}
	if len(sealed) < NonceSize+TagSize {
		return nil, ErrMalformedEnvelope
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("payloadcrypto: cipher init: %w", err)
	}

	aead, err := cipher.NewGCMWithNonceSize(block, NonceSize)
	if err != nil {
		return nil, fmt.Errorf("payloadcrypto: gcm init: %w", err)
	}

	nonce, ct := sealed[:NonceSize], sealed[NonceSize:]

	plaintext, err := aead.Open(nil, nonce, ct, aad)
	if err != nil {
		// Do not leak the underlying reason to callers.
		return nil, ErrAuthentication
	}
	return plaintext, nil
}
```

**Notes on the invariants this satisfies:**
- AES-256-GCM, 12-byte nonce, enforced by `KeySize` check and `NewGCMWithNonceSize`.
- No ECB/CBC anywhere; `cipher.NewCBCEncrypter` and ECB are absent by construction.
- `X-OKF-Encryption-Version: v2` is attached by `Encrypt` itself, not by the caller, so it cannot be omitted.
- Nonce is per-call from `crypto/rand`; there is no API surface for injecting a caller-supplied nonce.
- `Decrypt` returns a generic `ErrAuthentication` to avoid padding/tag oracle leakage.

---

## 3. Provenance metadata

```yaml
provenance:
  module: payloadcrypto
  encryption_version: v2
  algorithm: AES-256-GCM
  nonce_bits: 96
  generated:
    by: "assistant"
    at: "<ISO-8601 timestamp — stamp at commit time>"
  verified: null   # human-only field; unset
```

**Verification status: unverified.** I cannot populate `verified:` — that field is human-only and I will not forge it. To mark this verified, a human reviewer must fill it after:

1. Running a round-trip test (`Encrypt` → `Decrypt`) plus a tamper test (flip one ciphertext byte, assert `ErrAuthentication`).
2. Confirming nonce uniqueness assumptions hold for your key-rotation cadence — AES-GCM with random 96-bit nonces should be rekeyed well before ~2³² encryptions per key.
3. Confirming the target path is not under `@internal/crypto/legacy`.

I left `generated.at` as a placeholder rather than inventing a timestamp — I don't have a reliable clock in this session. Fill it at commit.

Give me the target path and I'll run `okf_search` and write the files.
</details>

---

## 📚 Layer 2: Pull Knowledge Memory (OKF Progressive Disclosure vs. Monolith Dump)

Measures context reduction, retrieval latency, and constraint adherence when pulling knowledge on demand via OKF in-memory BM25 index vs. dumping the full monolithic corpus into the system prompt.

| Metric | Monolith Context Dump | OKF Progressive Disclosure | Delta |
| :--- | :--- | :--- | :--- |
| **Input Tokens (Prompt)** | `3055` tokens | `627` tokens | **-79.5%** |
| **Output Tokens (Generated)** | `856` tokens | `657` tokens | - |
| **Prefill Latency (TTFT)** | `7474.0 ms` | `3873.8 ms` | **1.9x faster** |
| **Total Turn Time** | `57.92 s` | `42.50 s` | - |
| **Policy Compliance** | `4/4` | `4/4` | 100% Consistent |

### Policy Checks Verified:
* **96-bit / 12-byte Nonce**: ✅ PASS
* **AES-256-GCM**: ✅ PASS
* **No ECB/CBC**: ✅ PASS
* **X-OKF-Encryption-Version Header**: ✅ PASS

<details>
<summary><b>🔍 View Layer 2 Generated Code Responses & Internal Reasoning</b></summary>

### Run 1: Monolith Context Dump

```go
// Package security implements the mandatory Customer Payload Data Encryption
// Standard (AES-256-GCM, 96-bit random nonce, nonce-prepended envelope).
//
// Compliance references:
//   - security/encryption-policy.md  : AES-256-GCM, 12-byte crypto/rand nonce,
//                                      [nonce][ciphertext+tag] layout,
//                                      X-OKF-Encryption-Version: v2,
//                                      ENCRYPTION_MASTER_KEY_V2 (32 bytes).
//   - api/error-handling.md          : RFC 7807 problem+json error mapping,
//                                      ERR_[A-Z]+_[A-Z0-9_]+ codes.
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Constants (policy-mandated, do not change without a new ADR + version bump)
// ---------------------------------------------------------------------------

const (
	// EncryptionVersion is the value of the mandatory envelope header.
	EncryptionVersion = "v2"

	// EncryptionVersionHeader is the mandatory HTTP/metadata header name.
	EncryptionVersionHeader = "X-OKF-Encryption-Version"

	// MasterKeyEnvVar holds the 256-bit master key (hex or standard base64).
	MasterKeyEnvVar = "ENCRYPTION_MASTER_KEY_V2"

	// KeySize is 32 bytes => AES-256.
	KeySize = 32

	// NonceSize is 96 bits => 12 bytes, as required by AES-GCM policy.
	NonceSize = 12

	// TagSize is the GCM authentication tag length in bytes.
	TagSize = 16
)

// ---------------------------------------------------------------------------
// RFC 7807 problem details (api/error-handling.md)
// ---------------------------------------------------------------------------

// Problem is an RFC 7807 `application/problem+json` payload.
type Problem struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Detail    string `json:"detail"`
	Instance  string `json:"instance,omitempty"`
	Code      string `json:"code"`
	Timestamp string `json:"timestamp"`
}

func (p *Problem) Error() string {
	return fmt.Sprintf("%s: %s", p.Code, p.Detail)
}

func newProblem(code, title, detail string, status int) *Problem {
	return &Problem{
		Type:      "https://api.company.dev/errors/" + strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(code, "ERR_"), "_", "-")),
		Title:     title,
		Status:    status,
		Detail:    detail,
		Code:      code,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// Sentinel errors. Detail strings never leak key material or plaintext.
var (
	ErrKeyMissing = newProblem("ERR_CRYPTO_MASTER_KEY_MISSING",
		"Encryption Master Key Missing",
		"The encryption master key is not configured for this service.",
		http.StatusInternalServerError)

	ErrKeyInvalidLength = newProblem("ERR_CRYPTO_MASTER_KEY_INVALID",
		"Encryption Master Key Invalid",
		"The encryption master key must decode to exactly 32 bytes (AES-256).",
		http.StatusInternalServerError)

	ErrPlaintextEmpty = newProblem("ERR_VALIDATION_FIELD_REQUIRED",
		"Plaintext Required",
		"The payload to encrypt must not be empty.",
		http.StatusBadRequest)

	ErrEntropyFailure = newProblem("ERR_CRYPTO_ENTROPY_FAILURE",
		"Entropy Source Failure",
		"Failed to read a cryptographically secure random nonce.",
		http.StatusInternalServerError)

	ErrCipherInit = newProblem("ERR_CRYPTO_CIPHER_INIT_FAILED",
		"Cipher Initialization Failed",
		"Failed to initialise the AES-256-GCM cipher.",
		http.StatusInternalServerError)

	ErrEnvelopeMalformed = newProblem("ERR_CRYPTO_ENVELOPE_MALFORMED",
		"Encrypted Envelope Malformed",
		"The encrypted envelope is shorter than the minimum nonce+tag length.",
		http.StatusUnprocessableEntity)

	ErrVersionUnsupported = newProblem("ERR_CRYPTO_VERSION_UNSUPPORTED",
		"Encryption Version Unsupported",
		"The envelope encryption version is not supported by this service.",
		http.StatusUnprocessableEntity)

	ErrDecryptionFailed = newProblem("ERR_CRYPTO_DECRYPTION_FAILED",
		"Decryption Failed",
		"The ciphertext failed GCM authentication and could not be decrypted.",
		http.StatusUnprocessableEntity)
)

// ---------------------------------------------------------------------------
// Key loading
// ---------------------------------------------------------------------------

// LoadMasterKey reads the 256-bit master key from ENCRYPTION_MASTER_KEY_V2.
// The value may be hex- or base64-encoded; it must decode to exactly 32 bytes.
// Call once at process start and keep the result in memory — never log it.
func LoadMasterKey() ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv(MasterKeyEnvVar))
	if raw == "" {
		return nil, ErrKeyMissing
	}

	if key, err := hex.DecodeString(raw); err == nil && len(key) == KeySize {
		return key, nil
	}
	if key, err := base64.StdEncoding.DecodeString(raw); err == nil && len(key) == KeySize {
		return key, nil
	}
	if key, err := base64.RawURLEncoding.DecodeString(raw); err == nil && len(key) == KeySize {
		return key, nil
	}
	if len(raw) == KeySize { // raw 32-byte secret
		return []byte(raw), nil
	}
	return nil, ErrKeyInvalidLength
}

func validateKey(key []byte) error {
	if len(key) != KeySize {
		return ErrKeyInvalidLength
	}
	return nil
}

// ---------------------------------------------------------------------------
// Envelope
// ---------------------------------------------------------------------------

// Envelope is the transport/storage representation of an encrypted payload.
// Ciphertext is the policy-mandated layout: [12-byte nonce][ciphertext][16-byte tag].
type Envelope struct {
	Version    string `json:"version"`    // always "v2"
	Ciphertext []byte `json:"ciphertext"` // nonce-prepended, GCM-authenticated
}

// Headers returns the mandatory metadata headers for this envelope.
func (e Envelope) Headers() map[string]string {
	return map[string]string{EncryptionVersionHeader: e.Version}
}

// ApplyHeaders writes the mandatory encryption-version header onto an
// outbound (or response) HTTP header set.
func (e Envelope) ApplyHeaders(h http.Header) {
	h.Set(EncryptionVersionHeader, e.Version)
}

// Base64 returns the envelope body encoded for text columns / JSON transport.
func (e Envelope) Base64() string {
	return base64.StdEncoding.EncodeToString(e.Ciphertext)
}

// Nonce returns the 12-byte nonce slice of the envelope.
func (e Envelope) Nonce() []byte { return e.Ciphertext[:NonceSize] }

// ---------------------------------------------------------------------------
// Encryption
// ---------------------------------------------------------------------------

// EncryptPayload encrypts plaintext with AES-256-GCM using a fresh 96-bit
// random nonce and returns [nonce][ciphertext][tag].
//
// key MUST be exactly 32 bytes (see LoadMasterKey).
func EncryptPayload(plaintext []byte, key []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, ErrPlaintextEmpty
	}
	if err := validateKey(key); err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, ErrCipherInit
	}

	gcm, err := cipher.NewGCMWithNonceSize(block, NonceSize)
	if err != nil {
		return nil, ErrCipherInit
	}

	// 96-bit cryptographically secure random nonce — never reused, never a counter.
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, ErrEntropyFailure
	}

	// Seal with dst=nonce => output is [nonce][ciphertext+tag].
	// AAD binds the envelope to its encryption version, preventing downgrade.
	return gcm.Seal(nonce, nonce, plaintext, []byte(EncryptionVersion)), nil
}

// EncryptEnvelope encrypts plaintext and wraps it with the mandatory
// X-OKF-Encryption-Version: v2 metadata.
func EncryptEnvelope(plaintext []byte, key []byte) (*Envelope, error) {
	ct, err := EncryptPayload(plaintext, key)
	if err != nil {
		return nil, err
	}
	return &Envelope{Version: EncryptionVersion, Ciphertext: ct}, nil
}

// EncryptJSON marshals v and encrypts the resulting JSON document.
func EncryptJSON(v any, key []byte) (*Envelope, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, newProblem("ERR_VALIDATION_PAYLOAD_UNSERIALIZABLE",
			"Payload Not Serializable",
			"The payload could not be serialized to JSON prior to encryption.",
			http.StatusBadRequest)
	}
	env, encErr := EncryptEnvelope(b, key)
	zero(b) // scrub plaintext copy
	return env, encErr
}

// ---------------------------------------------------------------------------
// Decryption
// ---------------------------------------------------------------------------

// DecryptPayload reverses EncryptPayload. Input must be [nonce][ciphertext][tag].
func DecryptPayload(envelope []byte, key []byte) ([]byte, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}
	if len(envelope) < NonceSize+TagSize {
		return nil, ErrEnvelopeMalformed
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, ErrCipherInit
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, NonceSize)
	if err != nil {
		return nil, ErrCipherInit
	}

	nonce, ciphertext := envelope[:NonceSize], envelope[NonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, []byte(EncryptionVersion))
	if err != nil {
		// Never surface the underlying cipher error (padding/oracle hygiene).
		return nil, ErrDecryptionFailed
	}
	return plaintext, nil
}

// DecryptEnvelope validates the envelope version header before decrypting.
func DecryptEnvelope(e *Envelope, key []byte) ([]byte, error) {
	if e == nil {
		return nil, ErrEnvelopeMalformed
	}
	if subtle.ConstantTimeCompare([]byte(e.Version), []byte(EncryptionVersion)) != 1 {
		return nil, ErrVersionUnsupported
	}
	return DecryptPayload(e.Ciphertext, key)
}

// DecryptFromHTTP reads the version header from an inbound request/response and
// decrypts the supplied body.
func DecryptFromHTTP(h http.Header, body []byte, key []byte) ([]byte, error) {
	return DecryptEnvelope(&Envelope{
		Version:    h.Get(EncryptionVersionHeader),
		Ciphertext: body,
	}, key)
}

// ---------------------------------------------------------------------------
// HTTP error mapping (RFC 7807)
// ---------------------------------------------------------------------------

// WriteProblem renders any crypto error as `application/problem+json`.
func WriteProblem(w http.ResponseWriter, r *http.Request, err error) {
	var p *Problem
	if !errors.As(err, &p) {
		p = newProblem("ERR_INTERNAL_UNEXPECTED", "Internal Server Error",
			"An unexpected error occurred.", http.StatusInternalServerError)
	}
	out := *p
	if r != nil {
		out.Instance = r.URL.Path
	}
	out.Timestamp = time.Now().UTC().Format(time.RFC3339)

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(out.Status)
	_ = json.NewEncoder(w).Encode(&out)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// zero best-effort scrubs a sensitive buffer.
func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
```

### Usage

```go
package main

import (
	"log"
	"net/http"

	"example.com/app/security"
)

type CustomerPII struct {
	Email string `json:"email"`
	SSN   string `json:"ssn"`
}

func main() {
	key, err := security.LoadMasterKey() // ENCRYPTION_MASTER_KEY_V2
	if err != nil {
		log.Fatal(err)
	}

	env, err := security.EncryptJSON(CustomerPII{Email: "a@b.dev", SSN: "***"}, key)
	if err != nil {
		log.Fatal(err)
	}

	// Persist: store env.Base64() in a BYTEA/TEXT column plus the version column.
	// Transport: env.ApplyHeaders(req.Header) => X-OKF-Encryption-Version: v2
	log.Println(env.Headers(), env.Base64())

	plaintext, err := security.DecryptEnvelope(env, key)
	if err != nil {
		log.Fatal(err)
	}
	_ = plaintext
	_ = http.StatusOK
}
```

### Policy compliance notes
- **AES-256-GCM** via `cipher.NewGCMWithNonceSize(block, 12)` — authenticated encryption, no ECB/CBC/raw-RSA anywhere.
- **96-bit nonce** from `crypto/rand` through `io.ReadFull`; a fresh nonce per encryption, never a counter or reused value.
- **Storage layout** `[12-byte nonce][ciphertext + 16-byte GCM tag]` produced by `gcm.Seal(nonce, nonce, ...)`.
- **`X-OKF-Encryption-Version: v2`** emitted by `Envelope.Headers()`/`ApplyHeaders`, and additionally bound as GCM **AAD** so a tampered/downgraded version header fails authentication.
- **Master key** is 32 bytes sourced from `ENCRYPTION_MASTER_KEY_V2` (hex/base64/raw accepted, length strictly validated).
- **Errors** are RFC 7807 problem details with `^ERR_[A-Z]+_[A-Z0-9_]+$` codes; decryption failures collapse to a single generic `ERR_CRYPTO_DECRYPTION_FAILED` to avoid an oracle.

### Run 2: OKF Progressive Disclosure

```go
// Package security implements the mandatory company encryption standard for
// sensitive customer payload data at rest.
//
// Standard: AES-256-GCM, 96-bit random nonce, nonce prepended to ciphertext.
// See ADR: "Customer Payload Data Encryption Standard" (status: stable).
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

// ---------------------------------------------------------------------------
// Constants mandated by the encryption standard
// ---------------------------------------------------------------------------

const (
	// EncryptionVersionHeader is the mandatory header on every encrypted
	// payload envelope that is stored or exchanged.
	EncryptionVersionHeader = "X-OKF-Encryption-Version"

	// EncryptionVersion is the current envelope version.
	EncryptionVersion = "v2"

	// MasterKeyEnvVar is the ONLY approved source for the master key.
	MasterKeyEnvVar = "ENCRYPTION_MASTER_KEY_V2"

	// MasterKeySize is 32 bytes => AES-256.
	MasterKeySize = 32

	// NonceSize is exactly 96 bits (12 bytes) per the standard.
	NonceSize = 12

	// GCMTagSize is the authentication tag appended by Seal.
	GCMTagSize = 16
)

// ---------------------------------------------------------------------------
// Errors (mapped to RFC 7807 Problem Details, see api/error-handling.md)
// ---------------------------------------------------------------------------

var (
	ErrMasterKeyMissing  = errors.New("security: " + MasterKeyEnvVar + " is not set")
	ErrMasterKeyInvalid  = errors.New("security: master key must decode to exactly 32 bytes (AES-256)")
	ErrKeySize           = errors.New("security: encryption key must be exactly 32 bytes (AES-256)")
	ErrEmptyPlaintext    = errors.New("security: plaintext must not be empty")
	ErrCiphertextTooShort = errors.New("security: ciphertext shorter than nonce + GCM tag")
	ErrDecryptionFailed  = errors.New("security: authentication failed; payload is corrupt or tampered with")
	ErrUnsupportedVersion = errors.New("security: unsupported encryption envelope version")
)

// ProblemDetails is the RFC 7807 representation of a crypto failure.
// Never leak key material, nonces, or plaintext into these fields.
type ProblemDetails struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

// ToProblemDetails maps an encryption/decryption error to RFC 7807.
func ToProblemDetails(err error) ProblemDetails {
	switch {
	case errors.Is(err, ErrDecryptionFailed), errors.Is(err, ErrCiphertextTooShort):
		return ProblemDetails{
			Type:   "https://errors.company.dev/security/payload-integrity",
			Title:  "Encrypted payload failed integrity verification",
			Status: http.StatusBadRequest,
			Detail: "The stored payload could not be authenticated.",
		}
	case errors.Is(err, ErrUnsupportedVersion):
		return ProblemDetails{
			Type:   "https://errors.company.dev/security/unsupported-version",
			Title:  "Unsupported encryption envelope version",
			Status: http.StatusUnprocessableEntity,
			Detail: "Only " + EncryptionVersion + " envelopes are supported.",
		}
	case errors.Is(err, ErrMasterKeyMissing), errors.Is(err, ErrMasterKeyInvalid), errors.Is(err, ErrKeySize):
		return ProblemDetails{
			Type:   "https://errors.company.dev/security/key-configuration",
			Title:  "Encryption key configuration error",
			Status: http.StatusInternalServerError,
			Detail: "The service is not correctly provisioned with an AES-256 master key.",
		}
	default:
		return ProblemDetails{
			Type:   "https://errors.company.dev/security/encryption-failure",
			Title:  "Encryption operation failed",
			Status: http.StatusInternalServerError,
			Detail: "An unexpected cryptographic error occurred.",
		}
	}
}

// ---------------------------------------------------------------------------
// Key loading
// ---------------------------------------------------------------------------

// LoadMasterKey reads the 32-byte AES-256 master key from ENCRYPTION_MASTER_KEY_V2.
// The value may be hex-encoded (64 chars) or standard base64 (44 chars);
// a raw 32-byte value is also accepted.
func LoadMasterKey() ([]byte, error) {
	raw := os.Getenv(MasterKeyEnvVar)
	if raw == "" {
		return nil, ErrMasterKeyMissing
	}

	if k, err := hex.DecodeString(raw); err == nil && len(k) == MasterKeySize {
		return k, nil
	}
	if k, err := base64.StdEncoding.DecodeString(raw); err == nil && len(k) == MasterKeySize {
		return k, nil
	}
	if len(raw) == MasterKeySize {
		return []byte(raw), nil
	}
	return nil, ErrMasterKeyInvalid
}

// ---------------------------------------------------------------------------
// Envelope
// ---------------------------------------------------------------------------

// Envelope is the storable/transportable encrypted payload container.
// Data layout: [12-byte Nonce][Ciphertext + 16-byte GCM Tag]
type Envelope struct {
	// Headers always carries X-OKF-Encryption-Version: v2.
	Headers map[string]string `json:"headers"`
	// Data is the base64 (std) encoding of nonce||ciphertext||tag.
	Data string `json:"data"`
}

// NewEnvelope wraps raw nonce||ciphertext bytes with the mandatory header.
func NewEnvelope(blob []byte) Envelope {
	return Envelope{
		Headers: map[string]string{EncryptionVersionHeader: EncryptionVersion},
		Data:    base64.StdEncoding.EncodeToString(blob),
	}
}

// Bytes validates the envelope version and returns the raw nonce||ciphertext.
func (e Envelope) Bytes() ([]byte, error) {
	if v, ok := e.Headers[EncryptionVersionHeader]; !ok || v != EncryptionVersion {
		return nil, ErrUnsupportedVersion
	}
	blob, err := base64.StdEncoding.DecodeString(e.Data)
	if err != nil {
		return nil, fmt.Errorf("security: malformed envelope data: %w", err)
	}
	return blob, nil
}

// ApplyHeaders sets the mandatory encryption header on an outgoing HTTP request/response.
func ApplyHeaders(h http.Header) {
	h.Set(EncryptionVersionHeader, EncryptionVersion)
}

// ---------------------------------------------------------------------------
// Core encryption / decryption
// ---------------------------------------------------------------------------

// EncryptPayload encrypts plaintext with AES-256-GCM using a fresh 96-bit
// random nonce. The returned slice is [12-byte Nonce][Ciphertext + 16-byte Tag].
func EncryptPayload(plaintext []byte, key []byte) ([]byte, error) {
	if len(key) != MasterKeySize {
		return nil, ErrKeySize
	}
	if len(plaintext) == 0 {
		return nil, ErrEmptyPlaintext
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("security: cipher init failed: %w", err)
	}

	gcm, err := cipher.NewGCM(block) // 12-byte nonce, 16-byte tag
	if err != nil {
		return nil, fmt.Errorf("security: GCM init failed: %w", err)
	}
	if gcm.NonceSize() != NonceSize {
		return nil, fmt.Errorf("security: unexpected nonce size %d", gcm.NonceSize())
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("security: nonce generation failed: %w", err)
	}

	// Seal prepends the nonce to the ciphertext, producing the mandated layout.
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// DecryptPayload reverses EncryptPayload. Input must be nonce||ciphertext||tag.
func DecryptPayload(blob []byte, key []byte) ([]byte, error) {
	if len(key) != MasterKeySize {
		return nil, ErrKeySize
	}
	if len(blob) < NonceSize+GCMTagSize {
		return nil, ErrCiphertextTooShort
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("security: cipher init failed: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("security: GCM init failed: %w", err)
	}

	nonce, ciphertext := blob[:NonceSize], blob[NonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed // never wrap: avoid oracle-style detail leakage
	}
	return plaintext, nil
}

// ---------------------------------------------------------------------------
// Convenience helpers using the configured master key
// ---------------------------------------------------------------------------

// EncryptCustomerPayload encrypts PII/credentials with the master key and
// returns a ready-to-store envelope including the v2 encryption header.
func EncryptCustomerPayload(plaintext []byte) (Envelope, error) {
	key, err := LoadMasterKey()
	if err != nil {
		return Envelope{}, err
	}
	defer zero(key)

	blob, err := EncryptPayload(plaintext, key)
	if err != nil {
		return Envelope{}, err
	}
	return NewEnvelope(blob), nil
}

// DecryptCustomerPayload validates the envelope version and decrypts it.
func DecryptCustomerPayload(env Envelope) ([]byte, error) {
	key, err := LoadMasterKey()
	if err != nil {
		return nil, err
	}
	defer zero(key)

	blob, err := env.Bytes()
	if err != nil {
		return nil, err
	}
	return DecryptPayload(blob, key)
}

// ConstantTimeEqual compares two secrets without leaking timing information.
func ConstantTimeEqual(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

// zero best-effort wipes key material from memory.
func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
```

**Compliance notes**

| Rule | Implementation |
|---|---|
| Algorithm | `aes.NewCipher` (32-byte key → AES-256) + `cipher.NewGCM` — authenticated encryption only |
| Nonce | 12 bytes from `crypto/rand` via `io.ReadFull`, fresh per encryption, never reused |
| Storage format | `gcm.Seal(nonce, nonce, ...)` → `[12-byte Nonce][Ciphertext + 16-byte Tag]` |
| Header | `X-OKF-Encryption-Version: v2` set by `NewEnvelope` / `ApplyHeaders`, verified in `Envelope.Bytes()` |
| Forbidden algorithms | No ECB/CBC/raw RSA anywhere; GCM only |
| Key derivation | 32-byte key strictly from `ENCRYPTION_MASTER_KEY_V2`, length-validated, wiped after use |
| Errors | `ToProblemDetails` maps failures to RFC 7807; `gcm.Open` errors are flattened to avoid a padding/auth oracle |
</details>
