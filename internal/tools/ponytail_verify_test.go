package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HoangP8/tokless/internal/agents"
	"github.com/HoangP8/tokless/internal/core"
	"github.com/HoangP8/tokless/internal/util"
)

func init() {
	Register()
	agents.Register()
}

func lookupPonytail(t *testing.T) *core.ToolManifest {
	t.Helper()
	for _, tm := range core.ListTools() {
		if tm.ID == "ponytail" {
			return tm
		}
	}
	t.Fatal("ponytail not registered")
	return nil
}

// TestPonytailWireDryRun confirms dry-run path emits the expected hint without
// invoking any external command.
func TestPonytailWireDryRun(t *testing.T) {
	setupHome(t)
	tm := lookupPonytail(t)
	for _, agent := range []string{"claude", "opencode", "codex", "antigravity", "copilot"} {
		path := agentInstructionPath(t, agent)
		_ = util.EnsureDir(filepath.Dir(path))
		_ = util.WriteFile(path, "# Notes\n\nkeep me\n")

		if _, err := tm.WireFor[agent](core.RunOpts{DryRun: true}); err != nil {
			t.Fatalf("%s dry-run wire: %v", agent, err)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s read: %v", agent, err)
		}
		body := string(raw)
		if !strings.Contains(body, "## Build Discipline (ponytail)") {
			t.Errorf("%s: ponytail section missing after dry-run wire:\n%s", agent, body)
		}
	}
}

// TestPonytailUnwireRemovesSection confirms unwire drops the ponytail block
// from each agent's instruction body and leaves the rest of the body intact.
func TestPonytailUnwireRemovesSection(t *testing.T) {
	setupHome(t)
	tm := lookupPonytail(t)
	for _, agent := range []string{"claude", "opencode", "codex", "antigravity", "copilot"} {
		path := agentInstructionPath(t, agent)
		_ = util.EnsureDir(filepath.Dir(path))
		_ = util.WriteFile(path, "# Notes\n\nkeep me\n")

		_, _ = tm.WireFor[agent](core.RunOpts{DryRun: true})
		raw, _ := os.ReadFile(path)
		if !strings.Contains(string(raw), "## Build Discipline (ponytail)") {
			t.Fatalf("%s: section not written", agent)
		}
		_, _ = tm.UnwireFor[agent](core.RunOpts{DryRun: true})
		raw, _ = os.ReadFile(path)
		if strings.Contains(string(raw), "## Build Discipline (ponytail)") {
			t.Errorf("%s: section still present after unwire:\n%s", agent, raw)
		}
		if !strings.Contains(string(raw), "keep me") {
			t.Errorf("%s: user notes lost after unwire:\n%s", agent, raw)
		}
	}
}

// TestPonytailOpencodeRegistersPlugin confirms the opencode wire path writes
// a plugin entry into opencode.json when test mode short-circuits the install.
func TestPonytailOpencodeRegistersPlugin(t *testing.T) {
	setupHome(t)
	tm := lookupPonytail(t)
	path := agentInstructionPath(t, "opencode")
	_ = util.EnsureDir(filepath.Dir(path))
	_ = util.WriteFile(path, "# Notes\n\nkeep me\n")

	if _, err := tm.WireFor["opencode"](core.RunOpts{}); err != nil {
		t.Fatalf("opencode wire: %v", err)
	}
	if !ponytailOpencodeInstalled() {
		t.Errorf("opencode plugin path not registered in opencode.json")
	}
	if !HasOwner("opencode", "ponytail") {
		t.Errorf("ponytail not marked in AGENTS.md owner list")
	}
	if v := tm.VerifyFor["opencode"](); v == nil || !*v {
		t.Errorf("opencode verify should be true in test mode")
	}
}

// TestPonytailUnwireOpencodeDropsPlugin confirms unwire removes the plugin
// entry from opencode.json.
func TestPonytailUnwireOpencodeDropsPlugin(t *testing.T) {
	setupHome(t)
	tm := lookupPonytail(t)
	path := agentInstructionPath(t, "opencode")
	_ = util.EnsureDir(filepath.Dir(path))
	_ = util.WriteFile(path, "# Notes\n\nkeep me\n")

	_, _ = tm.WireFor["opencode"](core.RunOpts{})
	if !ponytailOpencodeInstalled() {
		t.Fatalf("opencode plugin not registered after wire")
	}
	_, _ = tm.UnwireFor["opencode"](core.RunOpts{DryRun: true})
	if ponytailOpencodeInstalled() {
		t.Errorf("ponytail plugin entry still present in opencode.json after unwire")
	}
	if v := tm.VerifyFor["opencode"](); v == nil || *v {
		t.Errorf("opencode verify should be false after unwire in test mode")
	}
}

// TestPonytailVersionDirs exercises ponytailVersionDirs and the marker write
// path so verify() can later report installed version.
func TestPonytailVersionDirs(t *testing.T) {
	setupHome(t)
	dirs := util.PonytailVersionDirsForTest()
	if len(dirs) == 0 {
		t.Fatal("ponytailVersionDirs returned empty")
	}
	for _, d := range dirs {
		if !strings.Contains(d, "ponytail") {
			t.Errorf("dir %q does not mention ponytail", d)
		}
	}
}

// TestPonytailGatherVersions confirms the version tracker surfaces ponytail.
func TestPonytailGatherVersions(t *testing.T) {
	info := util.GatherVersions()
	v, ok := info["ponytail"]
	if !ok {
		t.Fatal("ponytail missing from GatherVersions")
	}
	if v.Channel != "github" {
		t.Errorf("ponytail channel = %q, want github", v.Channel)
	}
}

// TestPonytailCodexWireDryRunIsInstructionOnly pins the Codex baseline:
// AGENTS.md only, no fake hook, no npx skills install.
func TestPonytailCodexWireDryRunIsInstructionOnly(t *testing.T) {
	setupHome(t)
	tm := lookupPonytail(t)
	path := agentInstructionPath(t, "codex")
	_ = util.EnsureDir(filepath.Dir(path))
	_ = util.WriteFile(path, "")

	captured, err := util.CaptureLogs(func() error {
		_, err := tm.WireFor["codex"](core.RunOpts{DryRun: true})
		return err
	})
	if err != nil {
		t.Fatalf("codex dry-run wire: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read codex AGENTS.md: %v", err)
	}
	if !strings.Contains(string(raw), "## Build Discipline (ponytail)") {
		t.Errorf("ponytail section not written to codex AGENTS.md:\n%s", raw)
	}
	for _, bad := range []string{"codex plugin install", "npx skills add"} {
		if strings.Contains(captured, bad) {
			t.Errorf("codex dry-run must not mention %q; got:\n%s", bad, captured)
		}
	}
	for _, want := range []string{"codex plugin marketplace add", "AGENTS.md", "/plugins", "/hooks"} {
		if !strings.Contains(captured, want) {
			t.Errorf("codex wire hint missing %q; got:\n%s", want, captured)
		}
	}
}

func TestPonytailCodexAndAgyVerifyOwnerBaseline(t *testing.T) {
	setupHome(t)
	tm := lookupPonytail(t)

	if v := tm.VerifyFor["codex"](); v == nil || *v {
		t.Fatalf("codex verify should start false")
	}
	if v := tm.VerifyFor["antigravity"](); v == nil || *v {
		t.Fatalf("antigravity verify should start false")
	}

	WriteOwner("codex", "ponytail")
	WriteOwner("antigravity", "ponytail")

	if v := tm.VerifyFor["codex"](); v == nil || !*v {
		t.Fatalf("codex verify should accept AGENTS.md ponytail baseline")
	}
	if v := tm.VerifyFor["antigravity"](); v == nil || !*v {
		t.Fatalf("antigravity verify should accept GEMINI.md ponytail baseline")
	}
}
