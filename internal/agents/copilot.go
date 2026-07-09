package agents

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/HoangP8/tokless/internal/core"
	"github.com/HoangP8/tokless/internal/util"
)

// ConfigureCopilotMcp upserts an MCP entry into Copilot CLI (~/.copilot/mcp-config.json)
// and VS Code user mcp.json. Returns whether either file changed.
func ConfigureCopilotMcp(toolID string) (changed bool, file string) {
	spawn := pickCopilotSpawn(toolID)
	cliChanged, cliFile := upsertCopilotCliMcp(toolID, spawn)
	vsChanged, vsFile := upsertVSCodeUserMcp(toolID, spawn)
	changed = cliChanged || vsChanged
	if cliChanged {
		file = cliFile
	} else if vsChanged {
		file = vsFile
	} else {
		file = util.CopilotPathsResolved().McpConfig
	}
	return changed, file
}

func pickCopilotSpawn(toolID string) util.McpSpawn {
	if toolID == "codegraph" {
		return util.WrapAutoIndex("copilot", util.PickMcpSpawn("codegraph", "serve", "--mcp"))
	}
	return util.PickMcpSpawn(toolID)
}

func upsertCopilotCliMcp(toolID string, spawn util.McpSpawn) (bool, string) {
	p := util.CopilotPathsResolved()
	_ = util.EnsureDir(p.Dir)
	raw, _ := util.ReadFileSafe(p.McpConfig)
	cfg := util.TryParseJsonc(raw)
	if cfg == nil {
		cfg = util.NewOrderedMap()
	}
	servers := getOrCreateMap(cfg, "mcpServers")
	desired := util.NewOrderedMap()
	desired.Set("type", "local")
	desired.Set("command", spawn.Command)
	desired.Set("args", toAnySlice(spawn.Args))
	desired.Set("tools", []any{"*"})
	if existing, ok := servers.Get(toolID); ok {
		if copilotCliMcpEqual(existing, desired) {
			return false, p.McpConfig
		}
	}
	servers.Set(toolID, desired)
	_ = util.WriteFile(p.McpConfig, util.StringifyJSON(cfg))
	return true, p.McpConfig
}

func upsertVSCodeUserMcp(toolID string, spawn util.McpSpawn) (bool, string) {
	f := util.VSCodeUserMcpPath()
	if f == "" {
		return false, ""
	}
	_ = util.EnsureDir(filepath.Dir(f))
	raw, _ := util.ReadFileSafe(f)
	cfg := util.TryParseJsonc(raw)
	if cfg == nil {
		cfg = util.NewOrderedMap()
	}
	servers := getOrCreateMap(cfg, "servers")
	desired := util.NewOrderedMap()
	desired.Set("type", "stdio")
	desired.Set("command", spawn.Command)
	desired.Set("args", toAnySlice(spawn.Args))
	if existing, ok := servers.Get(toolID); ok {
		if vscodeMcpEqual(existing, desired) {
			return false, f
		}
	}
	servers.Set(toolID, desired)
	_ = util.WriteFile(f, util.StringifyJSON(cfg))
	return true, f
}

func copilotCliMcpEqual(existing any, desired *util.OrderedMap) bool {
	em, ok := existing.(*util.OrderedMap)
	if !ok {
		return false
	}
	for _, key := range []string{"type", "command"} {
		a, _ := em.Get(key)
		b, _ := desired.Get(key)
		if jsonStr(a) != jsonStr(b) {
			return false
		}
	}
	argsA, _ := em.Get("args")
	argsB, _ := desired.Get("args")
	if jsonStr(orEmptyArr(argsA)) != jsonStr(orEmptyArr(argsB)) {
		return false
	}
	toolsA, _ := em.Get("tools")
	toolsB, _ := desired.Get("tools")
	return jsonStr(orEmptyArr(toolsA)) == jsonStr(orEmptyArr(toolsB))
}

func vscodeMcpEqual(existing any, desired *util.OrderedMap) bool {
	em, ok := existing.(*util.OrderedMap)
	if !ok {
		return false
	}
	for _, key := range []string{"type", "command"} {
		a, _ := em.Get(key)
		b, _ := desired.Get(key)
		if jsonStr(a) != jsonStr(b) {
			return false
		}
	}
	argsA, _ := em.Get("args")
	argsB, _ := desired.Get("args")
	return jsonStr(orEmptyArr(argsA)) == jsonStr(orEmptyArr(argsB))
}

// CopilotHasMcp reports whether both CLI and VS Code configs register the tool
// when the VS Code path is writable; CLI presence is always required.
func CopilotHasMcp(toolID string) bool {
	if !copilotCliHasMcp(toolID) {
		return false
	}
	f := util.VSCodeUserMcpPath()
	if f == "" {
		return true
	}
	return vscodeHasMcp(toolID)
}

func copilotCliHasMcp(toolID string) bool {
	p := util.CopilotPathsResolved()
	raw, ok := util.ReadFileSafe(p.McpConfig)
	if !ok {
		return false
	}
	cfg := util.TryParseJsonc(raw)
	if cfg == nil {
		return false
	}
	if s, ok := cfg.Get("mcpServers"); ok {
		if sm, ok := s.(*util.OrderedMap); ok {
			_, has := sm.Get(toolID)
			return has
		}
	}
	return false
}

func vscodeHasMcp(toolID string) bool {
	raw, ok := util.ReadFileSafe(util.VSCodeUserMcpPath())
	if !ok {
		return false
	}
	cfg := util.TryParseJsonc(raw)
	if cfg == nil {
		return false
	}
	if s, ok := cfg.Get("servers"); ok {
		if sm, ok := s.(*util.OrderedMap); ok {
			_, has := sm.Get(toolID)
			return has
		}
	}
	return false
}

// RemoveCopilotMcp deletes the tool from both Copilot CLI and VS Code MCP configs.
func RemoveCopilotMcp(toolID string) bool {
	removed := false
	p := util.CopilotPathsResolved()
	if raw, ok := util.ReadFileSafe(p.McpConfig); ok {
		if cfg := util.TryParseJsonc(raw); cfg != nil {
			if servers, ok := cfg.Get("mcpServers"); ok {
				if sm, ok := servers.(*util.OrderedMap); ok {
					if _, has := sm.Get(toolID); has {
						sm.Delete(toolID)
						_ = util.WriteFile(p.McpConfig, util.StringifyJSON(cfg))
						removed = true
					}
				}
			}
		}
	}
	if f := util.VSCodeUserMcpPath(); f != "" {
		if raw, ok := util.ReadFileSafe(f); ok {
			if cfg := util.TryParseJsonc(raw); cfg != nil {
				if servers, ok := cfg.Get("servers"); ok {
					if sm, ok := servers.(*util.OrderedMap); ok {
						if _, has := sm.Get(toolID); has {
							sm.Delete(toolID)
							_ = util.WriteFile(f, util.StringifyJSON(cfg))
							removed = true
						}
					}
				}
			}
		}
	}
	return removed
}

const (
	copilotRtkHookFile = "rtk-rewrite.json"
	copilotRtkMarker   = "rtk hook copilot"
)

func copilotRtkHookPath() string {
	return filepath.Join(util.CopilotPathsResolved().HooksDir, copilotRtkHookFile)
}

func copilotRtkHookCommand() string {
	if p := util.ResolveRtkBin(); p != "" {
		if strings.ContainsAny(p, " \t") {
			return "rtk hook copilot"
		}
		return p + " hook copilot"
	}
	return "rtk hook copilot"
}

// InstallCopilotRtkHook writes ~/.copilot/hooks/rtk-rewrite.json for both
// Copilot CLI and VS Code (which loads ~/.copilot/hooks by default).
func InstallCopilotRtkHook() {
	p := util.CopilotPathsResolved()
	_ = util.EnsureDir(p.HooksDir)
	command := copilotRtkHookCommand()

	hook := util.NewOrderedMap()
	hook.Set("type", "command")
	hook.Set("bash", command)
	hook.Set("powershell", command)
	hook.Set("cwd", ".")
	hook.Set("timeoutSec", 10)

	// Copilot CLI uses camelCase; VS Code maps it to PreToolUse.
	hooks := util.NewOrderedMap()
	hooks.Set("preToolUse", []any{hook})
	hooks.Set("PreToolUse", []any{hook})

	cfg := util.NewOrderedMap()
	cfg.Set("version", 1)
	cfg.Set("hooks", hooks)

	_ = util.WriteFile(copilotRtkHookPath(), util.StringifyJSON(cfg))
}

// RemoveCopilotRtkHook removes the tokless-managed RTK hook file.
func RemoveCopilotRtkHook() {
	path := copilotRtkHookPath()
	if !HasCopilotRtkHook() {
		return
	}
	_ = os.Remove(path)
}

// HasCopilotRtkHook reports whether the RTK rewrite hook is present.
func HasCopilotRtkHook() bool {
	raw, ok := util.ReadFileSafe(copilotRtkHookPath())
	if !ok {
		return false
	}
	return strings.Contains(raw, copilotRtkMarker)
}

func copilotKnownBinDirs() []string {
	dirs := []string{filepath.Join(util.Home(), ".local", "bin")}
	if util.IsWin {
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			dirs = append(dirs, filepath.Join(local, "GitHubCopilot", "bin"))
		}
	}
	return dirs
}

func copilotDesktopPaths() []string {
	switch goosForDetect {
	case "windows":
		var paths []string
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			paths = append(paths,
				filepath.Join(local, "Programs", "Microsoft VS Code", "Code.exe"),
				filepath.Join(local, "Programs", "Microsoft VS Code Insiders", "Code - Insiders.exe"),
			)
		}
		return paths
	case "darwin":
		return []string{"/Applications/Visual Studio Code.app", "/Applications/Visual Studio Code - Insiders.app"}
	default:
		return []string{"/usr/share/code", "/usr/bin/code"}
	}
}

var copilot = &core.AgentManifest{
	ID:        "copilot",
	Label:     "GitHub Copilot",
	Homepage:  "https://docs.github.com/en/copilot",
	CLIBin:    "copilot",
	ConfigDir: func() string { return util.CopilotPathsResolved().Dir },
	Detect: func() core.Detection {
		return detectAgent("copilot", util.CopilotPathsResolved().Dir, copilotKnownBinDirs(), copilotDesktopPaths())
	},
}
