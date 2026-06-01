package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Non-interactive (no TTY in tests) + explicit flags: selective uninstall must
// remove only the chosen tool from the chosen agent, preserving the rest.
func TestUninstallSelectiveFlags(t *testing.T) {
	t.Setenv("TOKLESS_TEST", "1")
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))
	oc := filepath.Join(dir, ".config", "opencode")
	os.MkdirAll(oc, 0o755)
	// fake an opencode install (config dir present) + wired caveman + codegraph
	os.WriteFile(filepath.Join(oc, "opencode.json"),
		[]byte(`{"plugin":["./plugins/caveman/plugin.js","context-mode"],"mcp":{"caveman-shrink":{"type":"local"},"codegraph":{"type":"local"}}}`), 0o644)
	os.WriteFile(filepath.Join(oc, "AGENTS.md"), []byte("<!-- caveman-begin -->\nx\n<!-- caveman-end -->\n"), 0o644)

	code := RunUninstall(InitOptions{Agents: []string{"opencode"}, Tools: []string{"caveman"}})
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	b, _ := os.ReadFile(filepath.Join(oc, "opencode.json"))
	s := string(b)
	if strings.Contains(s, "caveman") {
		t.Fatal("caveman not fully removed from opencode.json")
	}
	if !strings.Contains(s, "codegraph") || !strings.Contains(s, "context-mode") {
		t.Fatal("non-selected tools were wrongly removed")
	}
	if amd, _ := os.ReadFile(filepath.Join(oc, "AGENTS.md")); strings.Contains(string(amd), "caveman-begin") {
		t.Fatal("caveman ruleset not removed")
	}
}
