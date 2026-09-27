package vault_test

import (
	"os/exec"
	"testing"
)

func TestWasmCompilation(t *testing.T) {
	cmd := exec.Command("go", "build", ".")
	cmd.Env = append(cmd.Environ(), "GOOS=js", "GOARCH=wasm")
	cmd.Dir = "."

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Wasm compilation (GOOS=js GOARCH=wasm) failed: %v\nOutput: %s", err, string(out))
	}
}
