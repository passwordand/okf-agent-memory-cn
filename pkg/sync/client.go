// Package sync implements the client synchronization engine, remote hub communication,
// and conflict reconciliation protocols for OKF memory bundles.
package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var (
	// ErrBlobNotFound indicates that the requested CAS blob was not found on the hub.
	ErrBlobNotFound = errors.New("sync: blob not found")
	// ErrUnauthorized indicates invalid or missing authentication token.
	ErrUnauthorized = errors.New("sync: unauthorized")
)

// HeadConflictError is returned when a commit CAS (Compare-and-Swap) fails due to a head mismatch (HTTP 409).
type HeadConflictError struct {
	CurrentHead string `json:"current_head"`
	ErrorMsg    string `json:"error"`
}

func (e *HeadConflictError) Error() string {
	return fmt.Sprintf("sync: head conflict, server head is %q", e.CurrentHead)
}

// HeadResponse models the JSON payload from GET /api/v1/vaults/{id}/head.
type HeadResponse struct {
	VaultID    string `json:"vault_id"`
	HeadCommit string `json:"head_commit"`
	UpdatedAt  string `json:"updated_at"`
}

// CommitResponse models the JSON payload from POST /api/v1/vaults/{id}/commit.
type CommitResponse struct {
	Status string `json:"status"`
	Head   string `json:"head"`
}

// Client communicates with the blind CAS storage server of OKF Memory Hub.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// NewClient creates a configured Client instance.
func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	url := fmt.Sprintf("%s%s", c.BaseURL, path)
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("sync: failed to create request: %w", err)
	}

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	return req, nil
}

// GetHead retrieves the current HeadCommit of the specified vault.
func (c *Client) GetHead(ctx context.Context, vaultID string) (*HeadResponse, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/api/v1/vaults/%s/head", vaultID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sync: GetHead request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("sync: GetHead returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var head HeadResponse
	if err := json.NewDecoder(resp.Body).Decode(&head); err != nil {
		return nil, fmt.Errorf("sync: failed to decode GetHead response: %w", err)
	}

	return &head, nil
}

// Commit attempts to atomically advance the head pointer of the vault.
// Returns *HeadConflictError if the expected previous head does not match the server state.
func (c *Client) Commit(ctx context.Context, vaultID, newCommitHash string, expectedHead *string) (*CommitResponse, error) {
	reqPayload := struct {
		NewCommitHash        string  `json:"new_commit_hash"`
		ExpectedPreviousHead *string `json:"expected_previous_head"`
	}{
		NewCommitHash:        newCommitHash,
		ExpectedPreviousHead: expectedHead,
	}

	bodyBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("sync: failed to marshal commit request: %w", err)
	}

	req, err := c.newRequest(ctx, http.MethodPost, fmt.Sprintf("/api/v1/vaults/%s/commit", vaultID), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sync: Commit request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusConflict {
		var conflict HeadConflictError
		if err := json.NewDecoder(resp.Body).Decode(&conflict); err != nil {
			return nil, &HeadConflictError{ErrorMsg: "head_mismatch"}
		}
		return nil, &conflict
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrUnauthorized
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("sync: Commit returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var commitResp CommitResponse
	if err := json.NewDecoder(resp.Body).Decode(&commitResp); err != nil {
		return nil, fmt.Errorf("sync: failed to decode Commit response: %w", err)
	}

	return &commitResp, nil
}

// CheckMissingBlobs sends a batch of candidate blob hashes and receives the subset that does not exist in CAS.
func (c *Client) CheckMissingBlobs(ctx context.Context, vaultID string, hashes []string) ([]string, error) {
	reqPayload := struct {
		Hashes []string `json:"hashes"`
	}{
		Hashes: hashes,
	}

	bodyBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("sync: failed to marshal CheckMissing request: %w", err)
	}

	req, err := c.newRequest(ctx, http.MethodPost, fmt.Sprintf("/api/v1/vaults/%s/blobs/check-missing", vaultID), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sync: CheckMissing request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("sync: CheckMissing returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Missing []string `json:"missing"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("sync: failed to decode CheckMissing response: %w", err)
	}

	return result.Missing, nil
}

// PutBlob uploads an encrypted binary envelope to CAS storage at /api/v1/vaults/{id}/blobs/{hash}.
func (c *Client) PutBlob(ctx context.Context, vaultID, hash string, data []byte) error {
	req, err := c.newRequest(ctx, http.MethodPut, fmt.Sprintf("/api/v1/vaults/%s/blobs/%s", vaultID, hash), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("sync: PutBlob request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sync: PutBlob returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetBlob downloads an encrypted binary envelope from CAS storage at /api/v1/vaults/{id}/blobs/{hash}.
func (c *Client) GetBlob(ctx context.Context, vaultID, hash string) ([]byte, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/api/v1/vaults/%s/blobs/%s", vaultID, hash), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sync: GetBlob request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrBlobNotFound
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("sync: GetBlob returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("sync: failed to read blob response: %w", err)
	}

	return data, nil
}
