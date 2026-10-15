package okf

import (
	"fmt"
	"os"
	"strings"
)

// IsHumanIdentity reports whether an actor or verifier identifier denotes a human:
// "human" alone or followed by ':' or '/', compared case-insensitively.
// It is the single definition used by the provenance guard and by the verified.by filter.
func IsHumanIdentity(id string) bool {
	lower := strings.ToLower(strings.TrimSpace(id))
	if lower == "human" {
		return true
	}
	return strings.HasPrefix(lower, "human:") || strings.HasPrefix(lower, "human/")
}

// ensureNoForgedHumanVerification rejects human verifications that a non-human actor adds to c.
// Verifications already recorded for the concept at fullPath may be preserved. A new concept
// inherits nothing from a file that already exists at its path.
//
// The actor is self-declared, so this guards callers that cannot choose it (the MCP server)
// and accidental or injected forgery; it does not authenticate a CLI user who passes --actor.
func ensureNoForgedHumanVerification(fullPath string, c *Concept, actor string, isNew bool) error {
	if IsHumanIdentity(actor) {
		return nil
	}
	var recorded map[Verified]bool
	if !isNew {
		recorded = recordedVerifications(fullPath)
	}
	for _, v := range c.Verified {
		if IsHumanIdentity(v.By) && !recorded[v] {
			return fmt.Errorf("actor %q cannot add human verification %q", actor, v.By)
		}
	}
	return nil
}

// recordedVerifications returns the verifications stored in the concept file at fullPath.
func recordedVerifications(fullPath string) map[Verified]bool {
	// #nosec G304 -- fullPath is resolved and confined to the bundle by resolveInBundle
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil
	}
	existing, err := ParseConcept("", string(data))
	if err != nil {
		return nil
	}
	recorded := make(map[Verified]bool, len(existing.Verified))
	for _, v := range existing.Verified {
		recorded[v] = true
	}
	return recorded
}
