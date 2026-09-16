# 📊 Benchmark Report: DMAA Layer 2 (Monolith vs. OKF Progressive Disclosure)

* **Provider**: `LMSTUDIO`
* **Model Tested**: `qwen/qwen3-coder-30b`
* **Execution Mode**: `Local On-Device Inference`
* **Hardware / Client**: `Apple M2 Pro (32 GB Unified Memory, macOS)`
* **Endpoint**: `http://localhost:1234/v1`
* **Temperature**: `0.10`
* **Date**: 2026-09-16 09:56:15

| Metric | Monolith Context Dump | OKF Progressive Disclosure | Delta |
| :--- | :--- | :--- | :--- |
| **Input Tokens (Prompt)** | `3055` tokens | `627` tokens | **-79.5%** |
| **Output Tokens (Generated)** | `346` tokens | `285` tokens | - |
| **Prefill Latency (TTFT)** | `10375.5 ms` | `2614.0 ms` | **4.0x faster** |
| **Total Turn Time** | `23.98 s` | `13.12 s` | - |
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
package security

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "fmt"
    "io"
)

// EncryptPayload encrypts sensitive customer data using AES-256-GCM
// Returns the encrypted payload with prepended 12-byte nonce
// and the required metadata header for encryption version
func EncryptPayload(plaintext []byte, key []byte) ([]byte, map[string]string, error) {
    // Validate key length
    if len(key) != 32 {
        return nil, nil, fmt.Errorf("encryption key must be 32 bytes (256 bits)")
    }

    // Create AES cipher
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to create AES cipher: %w", err)
    }

    // Create GCM mode
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to create GCM mode: %w", err)
    }

    // Generate 12-byte nonce
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, nil, fmt.Errorf("failed to generate nonce: %w", err)
    }

    // Encrypt the payload
    ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

    // Prepare metadata header
    headers := map[string]string{
        "X-OKF-Encryption-Version": "v2",
    }

    return ciphertext, headers, nil
}
```

### Run 2: OKF Progressive Disclosure

```go
package security

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "fmt"
    "io"
)

// EncryptPayload encrypts sensitive customer data using AES-256-GCM
// with a 96-bit random nonce and returns the ciphertext with the nonce prepended.
func EncryptPayload(plaintext []byte, key []byte) ([]byte, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, fmt.Errorf("failed to create cipher: %w", err)
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, fmt.Errorf("failed to create GCM: %w", err)
    }
    
    nonce := make([]byte, gcm.NonceSize()) // 12 bytes
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, fmt.Errorf("failed to generate nonce: %w", err)
    }
    
    ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
    return ciphertext, nil
}

// GetEncryptionHeaders returns the required metadata header for encrypted payloads.
func GetEncryptionHeaders() map[string]string {
    return map[string]string{
        "X-OKF-Encryption-Version": "v2",
    }
}
```
</details>
