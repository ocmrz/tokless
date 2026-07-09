package util

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestCopilotPathsResolvedDefault(t *testing.T) {
	home := t.TempDir()
	SetHomeOverride(home)
	defer SetHomeOverride("")
	t.Setenv("COPILOT_HOME", "")

	p := CopilotPathsResolved()
	if p.Dir != filepath.Join(home, ".copilot") {
		t.Fatalf("Dir: got %q want %q", p.Dir, filepath.Join(home, ".copilot"))
	}
	if p.McpConfig != filepath.Join(home, ".copilot", "mcp-config.json") {
		t.Fatalf("McpConfig: got %q", p.McpConfig)
	}
	if p.Instructions != filepath.Join(home, ".copilot", "copilot-instructions.md") {
		t.Fatalf("Instructions: got %q", p.Instructions)
	}
	if p.HooksDir != filepath.Join(home, ".copilot", "hooks") {
		t.Fatalf("HooksDir: got %q", p.HooksDir)
	}
	if p.SkillsDir != filepath.Join(home, ".agents", "skills") {
		t.Fatalf("SkillsDir: got %q", p.SkillsDir)
	}
}

func TestCopilotPathsResolvedCOPILOT_HOME(t *testing.T) {
	home := t.TempDir()
	SetHomeOverride(home)
	defer SetHomeOverride("")
	override := t.TempDir()
	t.Setenv("COPILOT_HOME", override)

	p := CopilotPathsResolved()
	if p.Dir != override {
		t.Fatalf("Dir: got %q want %q", p.Dir, override)
	}
	if p.McpConfig != filepath.Join(override, "mcp-config.json") {
		t.Fatalf("McpConfig: got %q", p.McpConfig)
	}
}

func TestVSCodeUserMcpPath(t *testing.T) {
	home := t.TempDir()
	SetHomeOverride(home)
	defer SetHomeOverride("")

	var want string
	switch runtime.GOOS {
	case "darwin":
		want = filepath.Join(home, "Library", "Application Support", "Code", "User", "mcp.json")
	case "windows":
		app := filepath.Join(home, "AppData", "Roaming")
		t.Setenv("APPDATA", app)
		want = filepath.Join(app, "Code", "User", "mcp.json")
	default:
		t.Setenv("XDG_CONFIG_HOME", "")
		want = filepath.Join(home, ".config", "Code", "User", "mcp.json")
	}
	if got := VSCodeUserMcpPath(); got != want {
		t.Fatalf("VSCodeUserMcpPath: got %q want %q", got, want)
	}
}
