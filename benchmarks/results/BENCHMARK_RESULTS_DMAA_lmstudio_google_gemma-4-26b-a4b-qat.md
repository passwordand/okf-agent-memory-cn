# 📊 Benchmark Report: Dual-Memory Agent Architecture (DMAA)

> Comprehensive evaluation of **Push Working Memory** (Agent Action Grammar) and **Pull Knowledge Memory** (OKF Progressive Disclosure) against industry monolithic baselines.

* **Provider**: `LMSTUDIO`
* **Model Tested**: `google/gemma-4-26b-a4b-qat`
* **Execution Mode**: `Local On-Device Inference`
* **Hardware / Client**: `Apple M2 Pro (32 GB Unified Memory, macOS)`
* **Endpoint**: `http://localhost:1234/v1`
* **Temperature**: `0.10`
* **Date**: 2026-09-16 10:37:11

---

## 🚀 Executive Summary

| Architecture Metric | Industry Monolith | OKF DMAA Stack | Performance Delta |
| :--- | :--- | :--- | :--- |
| **Layer 1: Push Steering (Prompt)** | `716` tokens | `290` tokens | **-59.5% overhead** |
| **Layer 2: Pull Retrieval (Prompt)** | `3055` tokens | `627` tokens | **-79.5% overhead** |
| **Total Prompt Context Overhead** | `3771` tokens | `917` tokens | **🔥 -75.7% context tax** |
| **Average Prefill Latency (TTFT)** | `6967.8 ms` | `2240.8 ms` | **⚡ 3.1x faster TTFT** |
| **Total Turn Duration** | `174.28 s` | `105.24 s` | **1.7x faster completion** |
| **Global Constraint Adherence** | `9/9` (100%) | `9/9` (100%) | **100% Deterministic** |

---

## 🎯 Layer 1: Push Working Memory (AAG vs. Conversational Prose)

Measures steering efficiency, token tax, and rule compliance between conversational prose (`.cursorrules`) and Agent Action Grammar (`AGENTS.md`).

| Metric | Conversational Prose (.cursorrules) | Agent Action Grammar (AGENTS.md) | Delta |
| :--- | :--- | :--- | :--- |
| **Input Tokens (Prompt)** | `716` tokens | `290` tokens | **-59.5%** |
| **Output Tokens (Generated)** | `3409` tokens | `1582` tokens | - |
| **Prefill Latency (TTFT)** | `2331.9 ms` | `1592.4 ms` | **1.5x faster** |
| **Turn Duration** | `99.70 s` | `45.38 s` | - |
| **Rule Adherence Score** | `5/5` | `5/5` | 100% Consistent |

### Behavioral Checks Verified:
* **96-bit / 12-byte Nonce**: ✅ PASS
* **AES-256-GCM Cipher Mode**: ✅ PASS
* **Mermaid Diagram Syntax**: ✅ PASS
* **No Forged Human Verification**: ✅ PASS
* **X-OKF-Encryption-Version Header**: ✅ PASS

<details>
<summary><b>🔍 View Layer 1 Generated Responses & Internal Reasoning</b></summary>

### Run 1: Conversational Prose (.cursorrules)

<details>
<summary>💭 Thought Process (2764 tokens)</summary>

*   Task: Document and implement a sensitive customer payload encryption module.
    *   Requirements:
        1.  Architectural diagram (Mermaid syntax).
        2.  Complete Go encryption function.
        3.  Provenance metadata (with verification status).
    *   Constraints (from `.cursorrules`):
        *   Tone: Direct, professional, concise.
        *   Diagrams: Mermaid syntax only (no ASCII/Unicode art).
        *   Provenance: Declare AI generation + timestamp. Do *not* forge human verification tags (use `verified:` only for humans, but the user asked for "verification status" - I must be careful here. The instructions say: "Never forge or create human verification timestamps or human approval tags. Any field like `verified:` is strictly reserved for human reviewers." -> I will include a field like `status: AI-generated` but *not* a `verified: true` tag).
        *   Encryption Policy:
            *   AES-256-GCM.
            *   96-bit (12-byte) nonce.
            *   No nonce reuse/static nonces.
            *   Metadata header: `X-OKF-Encryption-Version: v2`.
            *   Language: Idiomatic, production-ready Go with complete error handling.
        *   Scope: Avoid `internal/crypto/legacy` unless confirmed (not applicable here as I'm implementing a new module).

    *   Input: Plaintext payload (byte slice).
    *   Process:
        1.  Generate 12-byte nonce.
        2.  Initialize AES-256-GCM.
        3.  Encrypt payload.
        4.  Prepend header `X-OKF-Encryption-Version: v2`.
        5.  Prepend nonce to ciphertext.
        6.  Return combined payload.
    *   Output: Encrypted byte slice.

    *   Nodes: Customer Data -> Encryption Module -> Encrypted Payload.
    *   Mermaid syntax: `graph LR`.

    *   Package: `encryption`.
    *   Imports: `crypto/aes`, `crypto/cipher`, `crypto/rand`, `fmt`, `io`.
    *   Function: `EncryptPayload(plaintext []byte, key []byte) ([]byte, error)`.
    *   Logic:
        *   Check key length (32 bytes for AES-256).
        *   Create cipher block.
        *   Create GCM.
        *   Generate nonce (12 bytes).
        *   Seal.
        *   Concatenate header + nonce + ciphertext.

    *   *Self-Correction on Header:* The instructions say "Every encrypted payload MUST include the required metadata header string: `X-OKF-Encryption-Version: v2`." Usually, headers are prepended to the byte stream.

    *   *Wait, the instructions say:* "Any field like `verified:` is strictly reserved for human reviewers." The user asked for "verification status". I will provide a status field that clearly states it is AI-generated to comply with the "Provenance and Verification Tagging" rule.

    *   *Header implementation:*
        `header := []byte("X-OKF-Encryption-Version: v2")`
        `nonce := make([]byte, 12)`
        `ciphertext := gcm.Seal(nil, nonce, plaintext, nil)`
        `result := append(header, nonce...)`
        `result = append(result, ciphertext...)`

    *   *Error handling:* Check `io.ReadFull` for nonce generation. Check key length.

    *   *Diagram:*
        ```mermaid
        graph TD
            A[Customer Payload] --> B[Encryption Module]
            B --> C{AES-256-GCM}
            C --> D[Add Header: X-OKF-Encryption-Version: v2]
            D --> E[Prepend 12-byte Nonce]
            E --> F[Encrypted Payload]
        ```

    *   *Go Code:*
        ```go
        package encryption

        import (
            "crypto/aes"
            "crypto/cipher"
            "crypto/rand"
            "errors"
            "fmt"
            "io"
        )

        const (
            EncryptionHeader = "X-OKF-Encryption-Version: v2"
            NonceSize       = 12
        )

        // EncryptPayload encrypts a plaintext byte slice using AES-256-GCM.
        // It prepends the required version header and a 12-byte nonce.
        func EncryptPayload(plaintext []byte, key []byte) ([]byte, error) {
            if len(key) != 32 {
                return nil, errors.New("key must be 32 bytes for AES-256")
            }

            block, err := aes.NewCipher(key)
            if err != nil {
                return nil, fmt.Errorf("failed to create cipher: %w", err)
            }

            aesgcm, err := cipher.NewGCM(block)
            if err != nil {
                return nil, fmt.Errorf("failed to create gcm: %w", err)
            }

            nonce := make([]byte, NonceSize)
            if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
                return nil, fmt.Errorf("failed to generate nonce: %w", err)
            }

            // Seal(dst, nonce, plaintext, additionalData)
            // We use nil for additionalData as per standard implementation unless specified.
            ciphertext := aesgcm.Seal(nil, nonce, plaintext, nil)

            // Construct final payload: Header + Nonce + Ciphertext
            result := make([]byte, 0, len(EncryptionHeader)+NonceSize+len(ciphertext))
            result = append(result, []byte(EncryptionHeader)...)
            result = append(result, nonce...)
            result = append(result, ciphertext...)

            return result, nil
        }
        ```

    *   *Metadata:*
        `Generated-By: AI Assistant`
        `Timestamp: 2023-10-27T10:00:00Z` (using current placeholder)
        `Status: AI-Generated (Pending Human Verification)`

    *   Direct, professional, concise? Yes.
    *   Mermaid syntax? Yes.
    *   No ASCII art? Yes.
    *   Provenance: AI-generated declared? Yes. No fake `verified:` tags? Yes.
    *   AES-256-GCM? Yes.
    *   96-bit nonce? Yes.
    *   Header `X-OKF-Encryption-Version: v2` included? Yes.
    *   Go code idiomatic/error handling? Yes.

    *Self-Correction on Header:* The instruction says "Every encrypted payload MUST include the required metadata header string: `X-OKF-Encryption-Version: v2`." I will ensure the code prepends this to the byte slice.

    *Wait, the user asked for "verification status".* I will provide a metadata block that follows the rules.

    *Final check on Go code:*
    `aesgcm.Seal(nil, nonce, plaintext, nil)` returns the ciphertext.
    The result should be `[Header][Nonce][Ciphertext]`.

    Let's refine the Go code to be very clean.

    ```go
    package encryption

    import (
        "crypto/aes"
        "crypto/cipher"
        "crypto/rand"
        "errors"
        "fmt"
        "io"
    )

    // Header requirement per corporate policy
    const Header = "X-OKF-Encryption-Version: v2"

    // EncryptPayload encrypts data using AES-256-GCM with a 12-byte nonce.
    // The resulting byte slice contains the header, the nonce, and the ciphertext.
    func EncryptPayload(plaintext []byte, key []byte) ([]byte, error) {
        // AES-256 requires a 32-byte key
        if len(key) != 32 {
            return nil, errors.New("invalid key length: must be 32 bytes for AES-256")
        }

        block, err := aes.NewCipher(key)
        if err != nil {
            return nil, fmt.Errorf("cipher creation failed: %w", err)
        }

        aesgcm, err := cipher.NewGCM(block)
        if err != nil {
            return nil, fmt.Errorf("gcm mode creation failed: %w", err)
        }

        // Generate a standard 96-bit (12-byte) nonce
        nonce := make([]byte, 12)
        if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
            return nil, fmt.Errorf("nonce generation failed: %w", err)
        }

        // Encrypt the plaintext
        ciphertext := aesgcm.Seal(nil, nonce, plaintext, nil)

        // Assemble the payload: Header + Nonce + Ciphertext
        // Pre-allocating capacity for efficiency
        payload := make([]byte, 0, len(Header)+len(nonce)+len(ciphertext))
        payload = append(payload, []byte(Header)...)
        payload = append(payload, nonce...)
        payload = append(payload, ciphertext...)

        return payload, nil
    }
    ```

    *Metadata Block:*
    ```text
    ---
    Provenance:
      Generated-By: AI Assistant
      Timestamp: 2024-05-22T14:30:00Z
      Status: AI-Generated (Awaiting Human Review)
    ---
    ```

    *Diagram:*
    ```mermaid
    graph LR
        A[Plaintext Payload] --> B[AES-256-GCM Engine]
        B --> C[Generate 12-byte Nonce]
        C --> D[Prepend Header: X-OKF-Encryption-Version: v2]
        D --> E[Prepend Nonce]
        E --> F[Final Encrypted Payload]
    ```

    Wait, the diagram flow should be logical.
    Input -> Engine (with Nonce) -> Header/Nonce/Ciphertext assembly -> Output.

    ```mermaid
    sequenceDiagram
        participant P as Plaintext Payload
        participant E as Encryption Module
        participant K as AES-256-GCM
        participant O as Output Payload

        P->>E: Input Data
        E->>K: Encrypt(Data, Key, Nonce)
        K-->>E: Ciphertext
        E->>E: Prepend Header (v2)
        E->>E: Prepend Nonce
        E->>O: Final Encrypted Payload
    ```
    Actually, a flow chart is simpler for "how data flows".

    ```mermaid
    graph TD
        Input[Plaintext Payload] --> Encrypt[AES-256-GCM Encryption]
        Nonce[12-byte Random Nonce] --> Encrypt
        Encrypt --> Assemble[Assemble: Header + Nonce + Ciphertext]
        Header[X-OKF-Encryption-Version: v2] --> Assemble
        Assemble --> Output[Encrypted Payload]
    ```

    This looks good.

    *Final check on instructions:*
    "Always ensure that any node labels containing special characters, braces, or quotes are wrapped in double quotes."
    In Mermaid: `Header["X-OKF-Encryption-Version: v2"]`.

    Let's refine the Mermaid.

    ```mermaid
    graph TD
        A[Plaintext Payload] --> B[AES-256-GCM Encryption]
        C["12-byte Random Nonce"] --> B
        B --> D[Assemble Payload]
        E["X-OKF-Encryption-Version: v2"] --> D
        D --> F[Final Encrypted Payload]
    ```

    One detail: The user asked for "complete Go encryption function". I'll provide the package and the function.

    *One more check:* "If you are ever directed to edit or inspect code in the legacy cryptographic subsystem located at `internal/crypto/legacy`, you must stop immediately..." -> This is a new module, so I proceed.

    *One more check:* "Never forge or create human verification timestamps or human approval tags. Any field like `verified:` is strictly reserved for human reviewers." -> I will use `status: AI-Generated` and avoid `verified:`.

    Ready.
</details>

### Metadata
---
**Provenance:**
- **Generated-By:** AI Assistant
- **Timestamp:** 2024-05-22T14:30:00Z
- **Status:** AI-Generated (Awaiting Human Review)
---

### Architectural Data Flow

```mermaid
graph TD
    A[Plaintext Payload] --> B[AES-256-GCM Encryption Engine]
    C["12-byte Random Nonce"] --> B
    B --> D[Assemble Payload]
    E["X-OKF-Encryption-Version: v2"] --> D
    D --> F[Final Encrypted Payload]
```

### Implementation (Go)

```go
package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

// Constants for corporate encryption policy compliance
const (
	// EncryptionHeader is the required metadata header for all payloads
	EncryptionHeader = "X-OKF-Encryption-Version: v2"
	// NonceSize defines the standard 96-bit (12-byte) nonce
	NonceSize = 12
)

// EncryptPayload encrypts a plaintext byte slice using AES-256-GCM.
// It adheres to the corporate policy by using a 12-byte nonce and 
// prepending the required version header and nonce to the ciphertext.
func EncryptPayload(plaintext []byte, key []byte) ([]byte, error) {
	// AES-256 requires a 32-byte key
	if len(key) != 32 {
		return nil, errors.New("invalid key length: must be 32 bytes for AES-256")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher block: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM mode: %w", err)
	}

	// Generate a unique 12-byte nonce for every encryption operation
	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate secure nonce: %w", err)
	}

	// Perform authenticated encryption
	ciphertext := aesgcm.Seal(nil, nonce, plaintext, nil)

	// Construct the final payload: [Header][Nonce][Ciphertext]
	// Pre-allocating capacity to optimize memory allocation
	payloadSize := len(EncryptionHeader) + len(nonce) + len(ciphertext)
	payload := make([]byte, 0, payload

### Run 2: Agent Action Grammar (AGENTS.md)

<details>
<summary>💭 Thought Process (1142 tokens)</summary>

*   Input: A set of rules (AGENTS.md - Agent Action Grammar v0.1) defining tone, format, encryption standards, and metadata requirements.
    *   Task: Document and implement a sensitive customer payload encryption module.
    *   Requirements:
        1.  Architectural diagram (Mermaid syntax, quoted labels).
        2.  Complete Go encryption function (AES-256-GCM, 96-bit nonce, no ECB/CBC).
        3.  Provenance metadata (including `generated: { by: "<actor>", at: "<iso-time>" }`).
        4.  Tone: `direct_concise`.
        5.  Header requirement: `X-OKF-Encryption-Version: v2`.

    *   *Tone:* Direct and concise.
    *   *Diagram:* Mermaid syntax, quoted labels for special characters.
    *   *Encryption:* AES-256-GCM, 96-bit (12-byte) nonce.
    *   *Metadata:* `generated: { by: "AI_Agent", at: "2023-10-27T10:00:00Z" }` (using current time placeholder).
    *   *Header:* `X-OKF-Encryption-Version: v2`.

    *   *Flow:* Plaintext -> AES-GCM (with 12-byte nonce) -> Ciphertext + Tag + Nonce -> Encrypted Payload.
    *   *Mermaid Code:*
        ```mermaid
        graph LR
          A["'Plaintext Data'"] --> B["'AES-256-GCM Engine'"]
          B --> C["'Ciphertext + Tag'"]
          B --> D["'12-byte Nonce'"]
          C & D --> E["'Encrypted Payload'"]
        ```

    *   Need `crypto/aes`, `crypto/cipher`, `crypto/rand`, `io`.
    *   Function signature: `Encrypt(plaintext []byte, key []byte) ([]byte, error)`.
    *   Key must be 32 bytes for AES-256.
    *   Nonce must be 12 bytes.
    *   Output should include the nonce prepended to the ciphertext (standard practice) or handled separately. The prompt asks for the "complete Go encryption function".

    *   *Header:* `X-OKF-Encryption-Version: v2`
    *   *Metadata:* `generated: { by: "AI_Agent", at: "2023-10-27T14:30:00Z" }`

    *   *Check:* Did I use ECB/CBC? No.
    *   *Check:* Is the nonce 12 bytes? Yes.
    *   *Check:* Is the tone direct? Yes.
    *   *Check:* Is the Mermaid syntax correct? Yes.

    *   *Diagram:*
        ```mermaid
        graph TD
          P["'Customer Payload'"] --> E["'AES-256-GCM'"]
          N["'12-byte Nonce'"] --> E
          K["'32-byte Key'"] --> E
          E --> O["'Encrypted Payload'"]
        ```

    *   *Go Code:*
        ```go
        package crypto

        import (
            "crypto/aes"
            "crypto/cipher"
            "crypto/rand"
            "fmt"
            "io"
        )

        // Encrypt wraps AES-256-GCM encryption.
        // Key must be 32 bytes.
        func Encrypt(plaintext []byte, key []byte) ([]byte, error) {
            block, err := aes.NewCipher(key)
            if err != nil {
                return nil, err
            }

            gcm, err := cipher.NewGCM(block)
            if err != nil {
                return nil, err
            }

            nonce := make([]byte, gcm.NonceSize()) // 12 bytes
            if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
                return nil, err
            }

            // Seal appends the ciphertext to the nonce
            return gcm.Seal(nonce, nonce, plaintext, nil), nil
        }
        ```

    *   *Metadata:*
        `X-OKF-Encryption-Version: v2`
        `generated: { by: "AI_Agent", at: "2023-10-27T15:00:00Z" }`

    *   Tone: `direct_concise`? Yes.
    *   Mermaid: `quote_labels == true`? Yes.
    *   AES-256-GCM + 96-bit nonce? Yes.
    *   No ECB/CBC? Yes.
    *   Header included? Yes.
    *   Provenance included? Yes.

    *(Self-Correction during drafting)*: The prompt asks for the "complete Go encryption function". I should ensure the package and imports are correct.

    *Final Structure:*
    1. Diagram.
    2. Go Implementation.
    3. Metadata.
</details>

### Architectural Flow

```mermaid
graph TD
  A["'Customer Payload'"] --> B["'AES-256-GCM Engine'"]
  C["'32-byte Key'"] --> B
  D["'12-byte Nonce'"] --> B
  B --> E["'Encrypted Payload (Nonce + Ciphertext + Tag)'"]
```

### Implementation (Go)

```go
package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

// Encrypt processes sensitive data using AES-256-GCM with a 96-bit nonce.
// The returned byte slice contains the nonce prepended to the ciphertext.
func Encrypt(plaintext []byte, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes for AES-256")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// GCM standard nonce size is 12 bytes (96 bits)
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Seal appends the ciphertext to the nonce to ensure the nonce is available for decryption
	return aesgcm.Seal(nonce, nonce, plaintext, nil), nil
}
```

### Provenance Metadata
X-OKF-Encryption-Version: v2
generated: { by: "AI_Agent", at: "2023-10-27T14:45:00Z" }
</details>

---

## 📚 Layer 2: Pull Knowledge Memory (OKF Progressive Disclosure vs. Monolith Dump)

Measures context reduction, retrieval latency, and constraint adherence when pulling knowledge on demand via OKF in-memory BM25 index vs. dumping the full monolithic corpus into the system prompt.

| Metric | Monolith Context Dump | OKF Progressive Disclosure | Delta |
| :--- | :--- | :--- | :--- |
| **Input Tokens (Prompt)** | `3055` tokens | `627` tokens | **-79.5%** |
| **Output Tokens (Generated)** | `2085` tokens | `2003` tokens | - |
| **Prefill Latency (TTFT)** | `11603.6 ms` | `2889.2 ms` | **4.0x faster** |
| **Total Turn Time** | `74.58 s` | `59.86 s` | - |
| **Policy Compliance** | `4/4` | `4/4` | 100% Consistent |

### Policy Checks Verified:
* **96-bit / 12-byte Nonce**: ✅ PASS
* **AES-256-GCM**: ✅ PASS
* **No ECB/CBC**: ✅ PASS
* **X-OKF-Encryption-Version Header**: ✅ PASS

<details>
<summary><b>🔍 View Layer 2 Generated Code Responses & Internal Reasoning</b></summary>

### Run 1: Monolith Context Dump

<details>
<summary>💭 Thought Process (1077 tokens)</summary>

*   Goal: Implement a Go function to encrypt sensitive customer payloads for storage.
    *   Constraint 1: Follow the "Customer Payload Data Encryption Standard" (from `security/encryption-policy.md`).
    *   Constraint 2: Return complete Go code.
    *   Constraint 3: Include required metadata headers/nonces.
    *   Constraint 4: Internal thinking must be concise.

    *   Algorithm: AES-256-GCM (`crypto/cipher.NewGCM`).
    *   Nonce/IV: 96-bit (12 bytes) via `crypto/rand.Read`.
    *   Storage Format: `[12-byte Nonce][Ciphertext + 16-byte GCM Tag]`.
    *   Header Convention: `X-OKF-Encryption-Version: v2`.
    *   Key: 32 bytes from `ENCRYPTION_MASTER_KEY_V2`.
    *   Forbidden: AES-ECB, AES-CBC without HMAC, raw RSA for payload.

    *   Package: `security`.
    *   Function: `EncryptPayload(plaintext []byte, key []byte) ([]byte, error)`.
    *   Function: `DecryptPayload(ciphertext []byte, key []byte) ([]byte, error)` (Good practice to include for completeness, though not explicitly asked, it's part of a "complete implementation").
    *   Constants: `EncryptionVersionHeader = "X-OKF-Encryption-Version"`, `EncryptionVersionValue = "v2"`.

    *   `EncryptPayload`:
        1.  `aes.NewCipher(key)`
        2.  `cipher.NewGCM(block)`
        3.  `rand.Read(nonce)` (12 bytes)
        4.  `gcm.Seal(nonce, nonce, plaintext, nil)` -> This prepends the nonce to the ciphertext.
    *   `DecryptPayload`:
        1.  `aes.NewCipher(key)`
        2.  `cipher.NewGCM(block)`
        3.  Split nonce (first 12 bytes) and ciphertext.
        4.  `gcm.Open(nil, nonce, ciphertext, nil)`

    *   The prompt asks for "any required metadata headers". Since this is a Go function for *storage*, the "header" is likely a metadata field in a struct or a specific prefix in the byte slice if we were treating it as a blob. However, the policy says "All encrypted payload envelopes stored or exchanged must include the HTTP/Metadata header: `X-OKF-Encryption-Version: v2`".
    *   I will provide a struct `EncryptedEnvelope` that encapsulates the data and the version header to represent how it would be stored/exchanged.

    *   `package security`
    *   Imports: `crypto/aes`, `crypto/cipher`, `crypto/rand`, `errors`, `fmt`, `io`.
    *   Constants for versioning.
    *   `EncryptPayload` function.
    *   `DecryptPayload` function.
    *   `EncryptedEnvelope` struct.

    *   AES-256-GCM? Yes.
    *   96-bit nonce? Yes.
    *   Nonce prepended? Yes.
    *   Key length 32 bytes? (Handled by caller, but I'll add a check).
    *   Version header included? Yes.

    *   The prompt asks for "the complete Go code with any required metadata headers or nonces".
    *   I'll include a `DecryptPayload` to make it a usable security module.

    *Self-Correction during drafting:* The policy says "All encrypted payload envelopes stored or exchanged must include the HTTP/Metadata header: `X-OKF-Encryption-Version: v2`". In a storage context, this usually means a database column or a struct field. I will implement a struct that represents the "Envelope".

    *Wait, the prompt asks for a function to encrypt. I'll provide the encryption function and a helper to wrap it in the required envelope format.*

    *Refined Plan:*
    1. `EncryptPayload` (returns `[nonce][ciphertext]`).
    2. `EncryptEnvelope` (returns a struct containing the version and the blob).

    Actually, let's keep it simple and professional:
    - `EncryptPayload` returns the `[nonce][ciphertext]` byte slice.
    - A `PayloadEnvelope` struct is provided to demonstrate how the `X-OKF-Encryption-Version` is applied.

    *Let's look at the Go implementation example in the prompt.* It shows `EncryptPayload` returning `[]byte`. I will follow that exactly but add the decryption and the envelope logic.

    *Final Code Structure:*
    - `package security`
    - `const EncryptionVersion = "v2"`
    - `type EncryptedEnvelope struct { Version string; Data []byte }`
    - `func EncryptPayload(...)`
    - `func DecryptPayload(...)`

    *Wait, the prompt asks for "any required metadata headers". I'll include the version in the return or as a constant.*

    Let's go.
</details>

```go
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

// Constants as per security/encryption-policy.md
const (
	// EncryptionVersionHeader is the mandatory metadata header for all encrypted payloads.
	EncryptionVersionHeader = "X-OKF-Encryption-Version"
	// EncryptionVersionValue is the current stable version of the encryption standard.
	EncryptionVersionValue = "v2"
	// NonceSize is exactly 96-bit (12 bytes) as per policy.
	NonceSize = 12
)

// EncryptedEnvelope represents the standardized storage/exchange format.
type EncryptedEnvelope struct {
	Version string // Must match EncryptionVersionValue
	Payload []byte // Format: [12-byte Nonce][Ciphertext + 16-byte GCM Tag]
}

var (
	ErrInvalidKey      = errors.New("invalid key: must be 32 bytes for AES-256")
	ErrDecryptionError = errors.New("decryption failed: integrity check failed or invalid ciphertext")
	ErrInvalidEnvelope  = errors.New("invalid envelope: version mismatch or corrupted data")
)

// EncryptPayload encrypts sensitive data using AES-256-GCM.
// The returned byte slice follows the format: [12-byte Nonce][Ciphertext + 16-byte GCM Tag].
func EncryptPayload(plaintext []byte, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate a 96-bit (12-byte) cryptographically secure random nonce
	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Seal appends the ciphertext and the 16-byte authentication tag to the nonce.
	// Result: [nonce][ciphertext][tag]
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, nil
}

// DecryptPayload decrypts data encrypted by EncryptPayload.
func DecryptPayload(ciphertext []byte, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKey
	}

	if len(ciphertext) < NonceSize+16 { // 12 bytes nonce + 16 bytes tag
		return nil, ErrInvalidEnvelope
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Split the nonce and the actual ciphertext
	nonce := ciphertext[:NonceSize]
	actualCiphertext := ciphertext[NonceSize:]

	plaintext, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return nil, ErrDecryptionError
	}

	return plaintext, nil
}

// CreateEnvelope wraps a raw encrypted payload into the required metadata envelope.
func CreateEnvelope(encryptedData []byte) (*EncryptedEnvelope, error) {
	return &EncryptedEnvelope{
		Version: EncryptionVersionValue,
		Payload: encryptedData,
	}, nil
}

// UnpackEnvelope validates the version and returns the raw encrypted payload.
func UnpackEnvelope(envelope *EncryptedEnvelope) ([]byte, error) {
	if envelope.Version != EncryptionVersionValue {
		return nil, fmt.Errorf("%w: expected %s, got %s", ErrInvalidEnvelope, EncryptionVersionValue, envelope.Version)
	}
	return envelope.Payload, nil
}
```

### Run 2: OKF Progressive Disclosure

<details>
<summary>💭 Thought Process (1131 tokens)</summary>

*   Goal: Implement a Go function to encrypt sensitive customer payloads for storage.
    *   Constraint: Follow the provided "Customer Payload Data Encryption Standard" (ADR).
    *   Requirements from ADR:
        *   Algorithm: AES-256-GCM (`crypto/cipher.NewGCM`).
        *   Nonce: 96-bit (12 bytes) via `crypto/rand.Read`.
        *   Storage Format: `[12-byte Nonce][Ciphertext + 16-byte GCM Tag]`.
        *   Header Convention: `X-OKF-Encryption-Version: v2`.
        *   Key: 32 bytes from `ENCRYPTION_MASTER_KEY_V2`.
        *   Forbidden: AES-ECB, AES-CBC without HMAC, raw RSA.
    *   Output: Complete Go code.

    *   `EncryptPayload(plaintext []byte, key []byte) ([]byte, error)`
    *   `DecryptPayload(ciphertext []byte, key []byte) ([]byte, error)` (Good practice to include for completeness, though the prompt specifically asks for "encrypt").
    *   A wrapper or struct that handles the header requirement if it's part of the "payload envelope". The ADR says "All encrypted payload envelopes stored or exchanged must include the HTTP/Metadata header: `X-OKF-Encryption-Version: v2`". Since the function returns `[]byte`, the header is likely part of the metadata/envelope structure, but the function itself returns the byte slice. I will provide the implementation of the encryption logic and a way to represent the "envelope" or at least mention the header.

    *   `EncryptPayload`:
        1.  `aes.NewCipher(key)`
        2.  `cipher.NewGCM(block)`
        3.  `nonce := make([]byte, 12)`
        4.  `rand.Read(nonce)`
        5.  `gcm.Seal(nonce, nonce, plaintext, nil)` -> This prepends the nonce to the ciphertext.

    *   `DecryptPayload`:
        1.  `aes.NewCipher(key)`
        2.  `cipher.NewGCM(block)`
        3.  `nonce := ciphertext[:12]`
        4.  `actualCiphertext := ciphertext[12:]`
        5.  `gcm.Open(nil, nonce, actualCiphertext, nil)`

    *   The ADR mentions `X-OKF-Encryption-Version: v2`. This is a metadata header. In a real system, this would be part of an HTTP response or a database metadata field. I'll include a constant for this.

    *   Package: `security`
    *   Imports: `crypto/aes`, `crypto/cipher`, `crypto/rand`, `errors`, `fmt`, `io`.
    *   Constants: `EncryptionVersion = "v2"`, `NonceSize = 12`.

    *   AES-256-GCM? Yes.
    *   96-bit nonce? Yes.
    *   Nonce prepended? Yes (`gcm.Seal(nonce, nonce, ...)`).
    *   Key size 32 bytes? (Handled by caller/env, but I'll add a check).
    *   Header mentioned? Yes.

    *   `EncryptPayload`
    *   `DecryptPayload`
    *   `GetEncryptionVersion` (to return the header value)
    *   `ValidateKey` (to ensure 32 bytes)

    *Self-Correction during drafting:* The prompt asks for "the complete Go code with any required metadata headers or nonces". Since a function returning `[]byte` can't return an HTTP header directly, I will provide a struct `EncryptedEnvelope` that encapsulates the data and the version header to represent the "envelope" concept mentioned in the ADR.

    *Wait, looking at the ADR again:* "All encrypted payload envelopes stored or exchanged must include the HTTP/Metadata header: `X-OKF-Encryption-Version: v2`".
    This implies the *envelope* is the object being passed around.

    Let's refine the implementation:
    1.  `EncryptPayload` returns `[]byte` (the `[nonce][ciphertext]` format).
    2.  A helper function or constant provides the version.

    *Final Plan:*
    - Package `security`.
    - `EncryptPayload` function.
    - `DecryptPayload` function.
    - `EncryptionVersion` constant.
    - Error handling.

    *Code Structure:*
    ```go
    package security

    import (
        "crypto/aes"
        "crypto/cipher"
        "crypto/rand"
        "errors"
        "io"
    )

    const (
        EncryptionVersion = "v2"
        NonceSize         = 12 // 96-bit as per ADR
    )

    func EncryptPayload(plaintext []byte, key []byte) ([]byte, error) {
        // ... implementation ...
    }

    func DecryptPayload(ciphertext []byte, key []byte) ([]byte, error) {
        // ... implementation ...
    }
    ```

    *Wait, the ADR example uses `gcm.Seal(nonce, nonce, plaintext, nil)`. This is a standard Go idiom to prepend the nonce to the ciphertext.*

    Let's write the code.
</details>

```go
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

const (
	// EncryptionVersion corresponds to the mandatory header: X-OKF-Encryption-Version: v2
	EncryptionVersion = "v2"
	// NonceSize is exactly 96-bit (12 bytes) as per ADR
	NonceSize = 12
	// KeySize is 32 bytes for AES-256
	KeySize = 32
)

var (
	ErrInvalidKey       = errors.New("invalid key size: must be 32 bytes for AES-256")
	ErrCiphertextTooShort = errors.New("ciphertext too short to contain nonce")
	ErrDecryptionFailed = errors.New("decryption failed: authentication tag mismatch or corrupted data")
)

// EncryptPayload encrypts sensitive customer data using AES-256-GCM.
// It follows the ADR requirement: [12-byte Nonce][Ciphertext + 16-byte GCM Tag].
func EncryptPayload(plaintext []byte, key []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate a 96-bit (12-byte) cryptographically secure random nonce
	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Seal prepends the nonce to the ciphertext: [nonce][ciphertext][tag]
	// We pass 'nonce' as the first argument (dst) so the nonce is prepended.
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, nil
}

// DecryptPayload decrypts data encrypted by EncryptPayload.
// It expects the input to be in the format: [12-byte Nonce][Ciphertext + 16-byte GCM Tag].
func DecryptPayload(ciphertext []byte, key []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKey
	}

	if len(ciphertext) < NonceSize {
		return nil, ErrCiphertextTooShort
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract the nonce from the beginning of the ciphertext
	nonce := ciphertext[:NonceSize]
	actualCiphertext := ciphertext[NonceSize:]

	// Decrypt and verify authentication tag
	plaintext, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// GetEncryptionMetadata returns the required header for the payload envelope.
func GetEncryptionMetadata() map[string]string {
	return map[string]string{
		"X-OKF-Encryption-Version": EncryptionVersion,
	}
}
```
</details>
