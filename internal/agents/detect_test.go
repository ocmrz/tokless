package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HoangP8/tokless/internal/util"
)

// restrictPath points PATH at an empty dir so no real CLI leaks in.
func restrictPath(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
}

func TestDetectAgentCLIWins(t *testing.T) {
	restrictPath(t)
	binDir := t.TempDir()
	bin := filepath.Join(binDir, "fakecli")
	if util.IsWin {
		bin += ".exe"
	}
	os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755)

	d := detectAgent("fakecli", t.TempDir(), []string{binDir}, nil)
	if !d.Installed || d.Source != "cli" {
		t.Fatalf("CLI in known dir should detect as cli, got %+v", d)
	}
}

func TestDetectAgentDesktopFallback(t *testing.T) {
	restrictPath(t)
	app := filepath.Join(t.TempDir(), "Fake.app")
	os.MkdirAll(app, 0o755)

	d := detectAgent("no-such-cli", t.TempDir(), nil, []string{app})
	if !d.Installed || d.Source != "desktop" {
		t.Fatalf("desktop app present should detect as desktop, got %+v", d)
	}
}

func TestDetectAgentDesktopMissing(t *testing.T) {
	restrictPath(t)
	d := detectAgent("no-such-cli", t.TempDir(), nil, []string{filepath.Join(t.TempDir(), "absent.app")})
	if d.Installed {
		t.Fatalf("nothing present should be not installed, got %+v", d)
	}
}

func TestDetectAgentBothSurfaces(t *testing.T) {
	restrictPath(t)
	binDir := t.TempDir()
	bin := filepath.Join(binDir, "fakecli")
	if util.IsWin {
		bin += ".exe"
	}
	os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755)
	app := filepath.Join(t.TempDir(), "Fake.app")
	os.MkdirAll(app, 0o755)

	d := detectAgent("fakecli", t.TempDir(), []string{binDir}, []string{app})
	if !d.Installed || d.Source != "cli+desktop" {
		t.Fatalf("both surfaces present should report cli+desktop, got %+v", d)
	}
}

// setGoos overrides the OS seam for desktop path resolution.
func setGoos(t *testing.T, goos string) {
	t.Helper()
	old := goosForDetect
	goosForDetect = goos
	t.Cleanup(func() { goosForDetect = old })
}

func TestOpencodeDesktopPathsPerOS(t *testing.T) {
	setGoos(t, "windows")
	t.Setenv("LOCALAPPDATA", `C:\Users\u\AppData\Local`)
	got := opencodeDesktopPaths()
	want := filepath.Join(`C:\Users\u\AppData\Local`, "Programs", "OpenCode", "OpenCode.exe")
	if len(got) != 1 || got[0] != want {
		t.Fatalf("windows: want [%s], got %v", want, got)
	}

	setGoos(t, "darwin")
	got = opencodeDesktopPaths()
	if len(got) != 1 || got[0] != "/Applications/OpenCode.app" {
		t.Fatalf("darwin: got %v", got)
	}

	setGoos(t, "linux")
	got = opencodeDesktopPaths()
	if len(got) != 1 || got[0] != "/usr/bin/ai.opencode.desktop" {
		t.Fatalf("linux: got %v", got)
	}
}

func TestClaudeDesktopPathsPerOS(t *testing.T) {
	setGoos(t, "windows")
	t.Setenv("LOCALAPPDATA", `C:\Users\u\AppData\Local`)
	t.Setenv("APPDATA", `C:\Users\u\AppData\Roaming`)
	got := claudeDesktopPaths()
	if len(got) != 2 ||
		got[0] != filepath.Join(`C:\Users\u\AppData\Local`, "AnthropicClaude", "claude.exe") ||
		got[1] != filepath.Join(`C:\Users\u\AppData\Roaming`, "Claude", "claude.exe") {
		t.Fatalf("windows: got %v", got)
	}

	setGoos(t, "darwin")
	got = claudeDesktopPaths()
	if len(got) != 1 || got[0] != "/Applications/Claude.app" {
		t.Fatalf("darwin: got %v", got)
	}

	// No Claude Desktop on Linux — must return nothing.
	setGoos(t, "linux")
	if got = claudeDesktopPaths(); len(got) != 0 {
		t.Fatalf("linux: expected no paths, got %v", got)
	}
}

func TestAntigravityDesktopPathsPerOS(t *testing.T) {
	setGoos(t, "windows")
	t.Setenv("LOCALAPPDATA", `C:\Users\u\AppData\Local`)
	got := antigravityDesktopPaths()
	if len(got) != 2 ||
		got[0] != filepath.Join(`C:\Users\u\AppData\Local`, "Programs", "Antigravity", "Antigravity.exe") ||
		got[1] != filepath.Join(`C:\Users\u\AppData\Local`, "Programs", "Antigravity IDE", "Antigravity IDE.exe") {
		t.Fatalf("windows: got %v", got)
	}

	setGoos(t, "darwin")
	got = antigravityDesktopPaths()
	if len(got) != 2 || got[0] != "/Applications/Antigravity.app" || got[1] != "/Applications/Antigravity IDE.app" {
		t.Fatalf("darwin: got %v", got)
	}

	setGoos(t, "linux")
	got = antigravityDesktopPaths()
	if len(got) != 4 || got[0] != "/opt/antigravity" || got[3] != "/usr/local/bin/antigravity-ide" {
		t.Fatalf("linux: got %v", got)
	}
}

func TestConfigureAntigravityMcpMergeAndRemove(t *testing.T) {
	home := t.TempDir()
	util.SetHomeOverride(home)
	defer util.SetHomeOverride("")

	p := util.AntigravityPathsResolved()
	_ = os.MkdirAll(p.Dir, 0o755)
	_ = os.WriteFile(p.McpConfig, []byte(`{"mcpServers":{"user-server":{"command":"keepme"}}}`), 0o644)
	_ = os.WriteFile(p.Settings, []byte(`{"mcpServers":{"codegraph":{"command":"old"}},"permissions":{"allow":["mcp(codegraph/*)"]}}`), 0o644)

	changed, _ := ConfigureAntigravityMcp("codegraph")
	if !changed {
		t.Fatal("expected first configure to write")
	}
	if changed, _ := ConfigureAntigravityMcp("codegraph"); changed {
		t.Fatal("second configure must be idempotent")
	}
	ConfigureAntigravityMcp("context-mode")

	raw, _ := os.ReadFile(p.McpConfigCLI)
	for _, want := range []string{`"codegraph"`, `"serve"`, `"--mcp"`, `"context-mode"`} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("mcp_config.json missing %s:\n%s", want, raw)
		}
	}
	if !AntigravityMcpHas("codegraph") || !AntigravityMcpHas("context-mode") {
		t.Fatal("AntigravityMcpHas should see both tools")
	}

	rawLegacy, _ := os.ReadFile(p.McpConfig)
	if !strings.Contains(string(rawLegacy), `"user-server"`) || !strings.Contains(string(rawLegacy), `"keepme"`) || strings.Contains(string(rawLegacy), `"codegraph"`) {
		t.Fatalf("legacy config not preserved/cleaned:\n%s", rawLegacy)
	}
	rawLegacySettings, _ := os.ReadFile(p.Settings)
	if strings.Contains(string(rawLegacySettings), `"codegraph"`) || strings.Contains(string(rawLegacySettings), `mcp(codegraph/*)`) {
		t.Fatalf("legacy settings should not contain codegraph:\n%s", rawLegacySettings)
	}
	rawCliSettings, _ := os.ReadFile(filepath.Join(home, ".gemini", "antigravity-cli", "settings.json"))
	if !strings.Contains(string(rawCliSettings), `mcp(codegraph/*)`) {
		t.Fatalf("CLI settings should allow codegraph MCP:\n%s", rawCliSettings)
	}

	RemoveAntigravityMcp("codegraph")
	if AntigravityMcpHas("codegraph") {
		t.Fatal("codegraph should be removed")
	}
	for _, f := range []string{p.McpConfig, p.McpConfigCLI, p.Settings} {
		raw, _ = os.ReadFile(f)
		if strings.Contains(string(raw), `"codegraph"`) {
			t.Fatalf("codegraph not removed from %s", f)
		}
	}
	raw, _ = os.ReadFile(p.McpConfig)
	if !strings.Contains(string(raw), `"user-server"`) {
		t.Fatalf("remove clobbered unrelated entries:\n%s", raw)
	}
	raw, _ = os.ReadFile(p.McpConfigCLI)
	if !strings.Contains(string(raw), `"context-mode"`) {
		t.Fatalf("remove clobbered canonical entries:\n%s", raw)
	}
}

func TestRemoveAntigravityCodegraphToolDefs(t *testing.T) {
	home := t.TempDir()
	util.SetHomeOverride(home)
	defer util.SetHomeOverride("")

	gemini := filepath.Join(home, ".gemini")
	for _, variant := range []string{"antigravity-cli", "antigravity-ide"} {
		toolDir := filepath.Join(gemini, variant, "mcp", "codegraph")
		if err := os.MkdirAll(toolDir, 0o755); err != nil {
			t.Fatal(err)
		}
		for _, file := range codegraphToolDefFiles {
			if err := os.WriteFile(filepath.Join(toolDir, file), []byte("{}"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(toolDir, "user.json"), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	RemoveAntigravityCodegraphToolDefs()

	for _, variant := range []string{"antigravity-cli", "antigravity-ide"} {
		toolDir := filepath.Join(gemini, variant, "mcp", "codegraph")
		for _, file := range codegraphToolDefFiles {
			if _, err := os.Stat(filepath.Join(toolDir, file)); !os.IsNotExist(err) {
				t.Fatalf("%s still exists", filepath.Join(toolDir, file))
			}
		}
		if _, err := os.Stat(filepath.Join(toolDir, "user.json")); err != nil {
			t.Fatalf("unrelated file removed: %v", err)
		}
	}
}

func TestCleanupLegacyAntigravityContextMode(t *testing.T) {
	home := t.TempDir()
	util.SetHomeOverride(home)
	defer util.SetHomeOverride("")

	gemini := filepath.Join(home, ".gemini")
	hooksDir := filepath.Join(gemini, "config")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	hooksFile := filepath.Join(hooksDir, "hooks.json")
	original := `{"rtk":{"PreToolUse":[{"matcher":"","hooks":[{"type":"command","command":"tokless rtk-hook agy"}]}]},"ctx":{"PreToolUse":[{"matcher":"","hooks":[{"type":"command","command":"context-mode hook gemini-cli beforetool"}]}]},"tokless-context-mode":{"PreToolUse":[{"matcher":"","hooks":[]}]}}`
	if err := os.WriteFile(hooksFile, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	CleanupLegacyAntigravityContextMode()

	got, err := os.ReadFile(hooksFile)
	if err != nil {
		t.Fatalf("hooks.json missing after cleanup: %v", err)
	}
	gs := string(got)
	if strings.Contains(gs, `"ctx"`) || strings.Contains(gs, `"tokless-context-mode"`) {
		t.Fatalf("legacy groups still present:\n%s", gs)
	}
	if !strings.Contains(gs, `"rtk"`) || !strings.Contains(gs, "rtk-hook agy") {
		t.Fatalf("unrelated rtk group was clobbered:\n%s", gs)
	}
	if strings.Contains(gs, "context-mode hook gemini-cli") {
		t.Fatalf("legacy context-mode hook command still present:\n%s", gs)
	}

	CleanupLegacyAntigravityContextMode()
	got2, _ := os.ReadFile(hooksFile)
	if string(got2) != gs {
		t.Fatalf("second pass mutated hooks.json:\n%s\nvs\n%s", got2, gs)
	}
}

// TestCleanupLegacyAntigravityContextModeNoLegacyIsNoop ensures the migration
// is a clean no-op on a hooks.json that never had a legacy group.
func TestCleanupLegacyAntigravityContextModeNoLegacyIsNoop(t *testing.T) {
	home := t.TempDir()
	util.SetHomeOverride(home)
	defer util.SetHomeOverride("")

	hooksFile := filepath.Join(home, ".gemini", "config", "hooks.json")
	if err := os.MkdirAll(filepath.Dir(hooksFile), 0o755); err != nil {
		t.Fatal(err)
	}
	original := `{"rtk":{"PreToolUse":[]}}`
	if err := os.WriteFile(hooksFile, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	CleanupLegacyAntigravityContextMode()
	got, _ := os.ReadFile(hooksFile)
	if string(got) != original {
		t.Fatalf("unexpected mutation:\nbefore:\n%s\nafter:\n%s", original, got)
	}
}

func TestAgyKnownBinDirsPerOS(t *testing.T) {
	setGoos(t, "windows")
	t.Setenv("LOCALAPPDATA", `C:\Users\u\AppData\Local`)
	got := agyKnownBinDirs()
	if len(got) != 1 || got[0] != filepath.Join(`C:\Users\u\AppData\Local`, "agy", "bin") {
		t.Fatalf("windows: got %v", got)
	}

	home := t.TempDir()
	util.SetHomeOverride(home)
	defer util.SetHomeOverride("")
	for _, goos := range []string{"darwin", "linux"} {
		setGoos(t, goos)
		got = agyKnownBinDirs()
		if len(got) != 1 || got[0] != filepath.Join(home, ".local", "bin") {
			t.Fatalf("%s: got %v", goos, got)
		}
	}
}

func TestConfigureCopilotMcpMergeAndRemove(t *testing.T) {
	home := t.TempDir()
	util.SetHomeOverride(home)
	defer util.SetHomeOverride("")
	t.Setenv("COPILOT_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))

	p := util.CopilotPathsResolved()
	_ = os.MkdirAll(p.Dir, 0o755)
	_ = os.WriteFile(p.McpConfig, []byte(`{"mcpServers":{"user-server":{"type":"local","command":"keepme","args":[],"tools":["*"]}}}`), 0o644)

	changed, _ := ConfigureCopilotMcp("codegraph")
	if !changed {
		t.Fatal("expected first configure to write")
	}
	if changed, _ := ConfigureCopilotMcp("codegraph"); changed {
		t.Fatal("second configure must be idempotent")
	}
	ConfigureCopilotMcp("context-mode")

	raw, _ := os.ReadFile(p.McpConfig)
	for _, want := range []string{`"codegraph"`, `"context-mode"`, `"user-server"`, `"mcpServers"`} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("CLI mcp-config.json missing %s:\n%s", want, raw)
		}
	}
	vsRaw, _ := os.ReadFile(util.VSCodeUserMcpPath())
	for _, want := range []string{`"codegraph"`, `"context-mode"`, `"servers"`} {
		if !strings.Contains(string(vsRaw), want) {
			t.Fatalf("VS Code mcp.json missing %s:\n%s", want, vsRaw)
		}
	}
	if !CopilotHasMcp("codegraph") || !CopilotHasMcp("context-mode") {
		t.Fatal("CopilotHasMcp should see both tools")
	}

	RemoveCopilotMcp("codegraph")
	if CopilotHasMcp("codegraph") {
		t.Fatal("codegraph should be removed")
	}
	raw, _ = os.ReadFile(p.McpConfig)
	if strings.Contains(string(raw), `"codegraph"`) || !strings.Contains(string(raw), `"user-server"`) || !strings.Contains(string(raw), `"context-mode"`) {
		t.Fatalf("CLI remove incorrect:\n%s", raw)
	}
	vsRaw, _ = os.ReadFile(util.VSCodeUserMcpPath())
	if strings.Contains(string(vsRaw), `"codegraph"`) || !strings.Contains(string(vsRaw), `"context-mode"`) {
		t.Fatalf("VS Code remove incorrect:\n%s", vsRaw)
	}
}

func TestInstallCopilotRtkHook(t *testing.T) {
	home := t.TempDir()
	util.SetHomeOverride(home)
	defer util.SetHomeOverride("")
	t.Setenv("COPILOT_HOME", "")

	InstallCopilotRtkHook()
	if !HasCopilotRtkHook() {
		t.Fatal("expected RTK hook after install")
	}
	raw, _ := os.ReadFile(copilotRtkHookPath())
	for _, want := range []string{"preToolUse", "PreToolUse", "rtk hook copilot", `"version": 1`} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("hook file missing %q:\n%s", want, raw)
		}
	}
	RemoveCopilotRtkHook()
	if HasCopilotRtkHook() {
		t.Fatal("hook should be gone after remove")
	}
}
