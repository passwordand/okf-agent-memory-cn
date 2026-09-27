package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/okf-memory/okf-agent-memory/pkg/sync"
	"github.com/okf-memory/okf-agent-memory/pkg/vault"
)

// VaultConfigFile stores local configuration for syncing an OKF bundle.
type VaultConfigFile struct {
	VaultID   string `json:"vault_id"`
	HubURL    string `json:"hub_url,omitempty"`
	AuthToken string `json:"auth_token,omitempty"`
}

const configFileName = ".okf-vault.json"

func resolveHubURL(flagURL string, cfg *VaultConfigFile) string {
	if flagURL != "" {
		return flagURL
	}
	if cfg != nil && cfg.HubURL != "" {
		return cfg.HubURL
	}
	return "http://127.0.0.1:8080"
}

func resolveToken(flagToken string, cfg *VaultConfigFile) string {
	if flagToken != "" {
		return flagToken
	}
	if env := os.Getenv("OKF_HUB_TOKEN"); env != "" {
		return env
	}
	if cfg != nil && cfg.AuthToken != "" {
		return cfg.AuthToken
	}
	return ""
}

func loadVaultConfig(dir string) (*VaultConfigFile, error) {
	cfgPath := filepath.Join(dir, configFileName)
	// #nosec G304 -- cfgPath is within user bundle root
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", cfgPath, err)
	}
	var cfg VaultConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", cfgPath, err)
	}
	return &cfg, nil
}

func saveVaultConfig(dir string, cfg *VaultConfigFile) error {
	cfgPath := filepath.Join(dir, configFileName)
	// #nosec G117 -- local vault configuration optionally stores user hub bearer token
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	// #nosec G703,G306 -- cfgPath is within user bundle root
	return os.WriteFile(cfgPath, data, 0o644)
}

func runHubInitVault(w io.Writer, dir, hubURL, authToken string) error {
	// Generate random 16-byte vault ID
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return fmt.Errorf("failed to generate random vault id: %w", err)
	}
	vaultID := "v_" + hex.EncodeToString(idBytes)

	secretKey, err := vault.GenerateSecretKey()
	if err != nil {
		return fmt.Errorf("failed to generate secret key: %w", err)
	}

	if hubURL == "" {
		hubURL = "http://127.0.0.1:8080"
	}

	cfg := &VaultConfigFile{
		VaultID:   vaultID,
		HubURL:    hubURL,
		AuthToken: authToken,
	}
	if err := saveVaultConfig(dir, cfg); err != nil {
		return fmt.Errorf("failed to save %s: %w", configFileName, err)
	}

	_, _ = fmt.Fprintf(w, `
================================================================================
                       OKF MEMORY HUB — EMERGENCY KIT
================================================================================
Vault ID:   %s
Secret Key: %s

CRITICAL WARNING:
Your knowledge bundle is encrypted client-side using zero-knowledge AES-256-GCM.
If you lose your master password and this Secret Key, your data stored on the
Hub CANNOT be recovered by anyone. Store this Emergency Kit in a safe place.
================================================================================
`, vaultID, secretKey)

	return nil
}

func executeHubPush(w io.Writer, dir string, client *sync.Client, vaultID, password, secretKey, message string) error {
	vaultKey, err := vault.DeriveVaultKey(password, secretKey)
	if err != nil {
		return fmt.Errorf("failed to derive vault key: %w", err)
	}

	engine := sync.NewEngine(client, vaultID, vaultKey, dir)
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "local-device"
	}
	author := vault.CommitAuthor{
		ClientID: hostname,
		Agent:    "okf-cli/" + Version,
	}

	res, err := engine.Push(context.Background(), author, message)
	if err != nil {
		return fmt.Errorf("push failed: %w", err)
	}

	_, _ = fmt.Fprintf(w, "Push completed: commit %s (uploaded %d, unchanged %d)\n", res.CommitHash, res.UploadedBlobs, res.UnchangedBlobs)
	return nil
}

func executeHubPull(w io.Writer, dir string, client *sync.Client, vaultID, password, secretKey string) error {
	vaultKey, err := vault.DeriveVaultKey(password, secretKey)
	if err != nil {
		return fmt.Errorf("failed to derive vault key: %w", err)
	}

	engine := sync.NewEngine(client, vaultID, vaultKey, dir)
	res, err := engine.Pull(context.Background())
	if err != nil {
		return fmt.Errorf("pull failed: %w", err)
	}

	_, _ = fmt.Fprintf(w, "Pull completed: commit %s (updated %d, deleted %d)\n", res.CommitHash, len(res.UpdatedFiles), len(res.DeletedFiles))
	return nil
}

func executeHubSync(w io.Writer, dir string, client *sync.Client, vaultID, password, secretKey, message string) error {
	vaultKey, err := vault.DeriveVaultKey(password, secretKey)
	if err != nil {
		return fmt.Errorf("failed to derive vault key: %w", err)
	}

	engine := sync.NewEngine(client, vaultID, vaultKey, dir)
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "local-device"
	}
	author := vault.CommitAuthor{
		ClientID: hostname,
		Agent:    "okf-cli/" + Version,
	}

	res, err := engine.Sync(context.Background(), author, message)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	if len(res.Conflicts) > 0 {
		for _, c := range res.Conflicts {
			forked := sync.ConflictLocalPath(c.Path)
			_, _ = fmt.Fprintf(w, "⚠️  Collision detected at %s. Local version saved as %s.\n", c.Path, forked)
		}
	}

	_, _ = fmt.Fprintf(w, "Sync completed: commit %s\n", res.CommitHash)
	return nil
}

func cmdHub(args []string) {
	if len(args) < 1 {
		printHubUsage()
		os.Exit(1)
	}

	subcmd := args[0]
	subargs := args[1:]

	switch subcmd {
	case "init-vault":
		fs := flag.NewFlagSet("hub init-vault", flag.ExitOnError)
		hubURL := fs.String("hub", "", "Hub server URL (default from .okf-vault.json or http://127.0.0.1:8080)")
		authToken := fs.String("auth-token", "", "Optional Hub authentication Bearer token")
		_ = fs.Parse(subargs)

		dir := "."
		if fs.NArg() > 0 {
			dir = fs.Arg(0)
		}
		if err := runHubInitVault(os.Stdout, dir, *hubURL, *authToken); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "push":
		fs := flag.NewFlagSet("hub push", flag.ExitOnError)
		hubURL := fs.String("hub", "", "Hub server URL (default from .okf-vault.json or http://127.0.0.1:8080)")
		authToken := fs.String("auth-token", "", "Hub authentication Bearer token (or OKF_HUB_TOKEN env)")
		password := fs.String("password", "", "Master password")
		secretKey := fs.String("secret-key", "", "Secret key")
		msg := fs.String("message", "CLI push", "Commit message")
		_ = fs.Parse(subargs)

		dir := "."
		if fs.NArg() > 0 {
			dir = fs.Arg(0)
		}

		cfg, err := loadVaultConfig(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v (run 'okf hub init-vault' first)\n", err)
			os.Exit(1)
		}

		client := sync.NewClient(resolveHubURL(*hubURL, cfg), resolveToken(*authToken, cfg))
		if err := executeHubPush(os.Stdout, dir, client, cfg.VaultID, *password, *secretKey, *msg); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "pull":
		fs := flag.NewFlagSet("hub pull", flag.ExitOnError)
		hubURL := fs.String("hub", "", "Hub server URL (default from .okf-vault.json or http://127.0.0.1:8080)")
		authToken := fs.String("auth-token", "", "Hub authentication Bearer token (or OKF_HUB_TOKEN env)")
		password := fs.String("password", "", "Master password")
		secretKey := fs.String("secret-key", "", "Secret key")
		_ = fs.Parse(subargs)

		dir := "."
		if fs.NArg() > 0 {
			dir = fs.Arg(0)
		}

		cfg, err := loadVaultConfig(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v (run 'okf hub init-vault' first)\n", err)
			os.Exit(1)
		}

		client := sync.NewClient(resolveHubURL(*hubURL, cfg), resolveToken(*authToken, cfg))
		if err := executeHubPull(os.Stdout, dir, client, cfg.VaultID, *password, *secretKey); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "sync":
		fs := flag.NewFlagSet("hub sync", flag.ExitOnError)
		hubURL := fs.String("hub", "", "Hub server URL (default from .okf-vault.json or http://127.0.0.1:8080)")
		authToken := fs.String("auth-token", "", "Hub authentication Bearer token (or OKF_HUB_TOKEN env)")
		password := fs.String("password", "", "Master password")
		secretKey := fs.String("secret-key", "", "Secret key")
		msg := fs.String("message", "CLI sync", "Commit message")
		_ = fs.Parse(subargs)

		dir := "."
		if fs.NArg() > 0 {
			dir = fs.Arg(0)
		}

		cfg, err := loadVaultConfig(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v (run 'okf hub init-vault' first)\n", err)
			os.Exit(1)
		}

		client := sync.NewClient(resolveHubURL(*hubURL, cfg), resolveToken(*authToken, cfg))
		if err := executeHubSync(os.Stdout, dir, client, cfg.VaultID, *password, *secretKey, *msg); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "serve":
		fs := flag.NewFlagSet("hub serve", flag.ExitOnError)
		port := fs.Int("port", 8080, "Port to listen on")
		storage := fs.String("storage", "", "Path to storage directory (empty for in-memory)")
		_ = fs.Parse(subargs)

		srv := sync.NewServer(*storage)
		addr := fmt.Sprintf(":%d", *port)
		fmt.Printf("Starting OKF Memory Hub server on http://localhost%s ...\n", addr)
		httpServer := &http.Server{
			Addr:              addr,
			Handler:           srv.Handler(),
			ReadHeaderTimeout: 10 * time.Second,
		}
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown hub command '%s'\n\n", subcmd)
		printHubUsage()
		os.Exit(1)
	}
}

func printHubUsage() {
	fmt.Print(`OKF Memory Hub Commands:

Usage:
  okf hub <command> [arguments] [flags]

Commands:
  init-vault [bundle]      Initialize a new zero-knowledge vault and print Emergency Kit
  push [bundle]            Push local changes to the remote hub
  pull [bundle]            Pull latest remote changes into the local bundle
  sync [bundle]            Pull and push with automated conflict reconciliation
  serve [--port 8080]      Run embedded blind CAS server for self-hosting and testing

Flags (push, pull, sync, init-vault):
  -hub <url>               Hub server URL (default from .okf-vault.json or http://127.0.0.1:8080)
  -auth-token <token>      Hub authentication Bearer token (or OKF_HUB_TOKEN env)
  -password <pass>         Master password
  -secret-key <key>        Secret key
  -message <msg>           Commit message (push, sync)
`)
}
