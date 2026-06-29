package tools

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/HoangP8/tokless/internal/agents"
	"github.com/HoangP8/tokless/internal/core"
	"github.com/HoangP8/tokless/internal/util"
)

func ctxEnsureInstalled(opts core.RunOpts) (bool, error) {
	if isTest() {
		return true, nil
	}
	if opts.DryRun {
		util.L.Sub("[dry-run] would: npm install -g context-mode@latest (cache-skew-resistant)")
		return true, nil
	}
	opts.Reportf("checking", 0.1)
	if util.Which("context-mode") != "" && !opts.Upgrade {
		opts.Reportf("already installed", 1)
		return true, nil
	}
	// context-mode needs Node 22+ (better-sqlite3 native prebuild).
	if !util.NodeAgeAlreadyChecked() {
		if maj := util.NodeMajor(); maj > 0 && maj < contextModeMinNode {
			util.L.Warn("Node.js v" + strconv.Itoa(maj) + " is too old for context-mode (need v" + strconv.Itoa(contextModeMinNode) + "+).")
			if util.Confirm("Upgrade Node.js now? (y/n)", true) {
				if !util.InstallNodeForTools() {
					util.L.Err("couldn't upgrade Node.js. context-mode needs v" + strconv.Itoa(contextModeMinNode) + "+.")
					util.L.Sub("Manual: https://nodejs.org/en/download")
					return false, nil
				}
			} else {
				util.L.Sub("Skipping. context-mode may fail to install.")
			}
		}
	}
	opts.Reportf("npm install -g", 0.4)
	v, ok, _ := util.NpmGlobalInstall("context-mode", "latest")
	if !ok {
		util.L.Err("context-mode install failed across all strategies (npm + tarball fallback).")
		util.L.Sub("Each attempt was logged above. Common causes: old Node, no build tools, or a registry mirror.")
		if hint := util.NodeTooOldHint(contextModeMinNode); hint != "" {
			util.L.Sub(hint)
		}
		return false, nil
	}
	opts.Reportf("ready", 1)
	util.L.Sub(util.C.Dim("context-mode @" + v + " installed"))
	return true, nil
}

const contextModeMinNode = 22

func pluginIsContextMode(entry string) bool {
	return entry == "context-mode" || strings.HasPrefix(entry, "context-mode@")
}

// setContextModePluginBare ensures the opencode.jsonc `plugin` array ends with
// the bare `context-mode` entry.
func setContextModePluginBare(cfg *util.OrderedMap) {
	plugins := getArr(cfg, "plugin")
	kept := make([]any, 0, len(plugins))
	for _, p := range plugins {
		if s, ok := p.(string); ok && pluginIsContextMode(s) {
			continue
		}
		kept = append(kept, p)
	}
	kept = append(kept, "context-mode")
	cfg.Set("plugin", kept)
	if mv, ok := cfg.Get("mcp"); ok {
		if mm, ok := mv.(*util.OrderedMap); ok {
			if _, has := mm.Get("context-mode"); has {
				mm.Delete("context-mode")
				if mm.Len() == 0 {
					cfg.Delete("mcp")
				}
			}
		}
	}
}

// --- Claude ---

func ctxWireClaude(opts core.RunOpts) (bool, error) {
	if opts.DryRun {
		util.L.Sub("[dry-run] would: claude mcp add context-mode -- context-mode")
		return true, nil
	}
	spawn := util.PickMcpSpawn("context-mode")
	if isTest() {
		cp := util.ClaudeCodePaths()
		_ = util.EnsureDir(cp.Dir)
		cfg := loadOrdered(cp.GlobalJSON)
		servers := getOrCreateMapT(cfg, "mcpServers")
		entry := util.NewOrderedMap()
		entry.Set("type", "stdio")
		entry.Set("command", spawn.Command)
		entry.Set("args", toAny(spawn.Args))
		servers.Set("context-mode", entry)
		_ = util.WriteFile(cp.GlobalJSON, util.StringifyJSON(cfg))
		agents.AllowClaudeMcpTool("context-mode")
		return true, nil
	}
	agents.ConfigureClaudeMcp("context-mode")
	util.L.Sub(util.C.Dim("tip: to enable slash commands, type inside Claude Code: /plugin marketplace add mksglu/context-mode && /plugin install context-mode@context-mode"))
	return true, nil
}

// --- OpenCode ---

func ctxWireOpenCode(opts core.RunOpts) (bool, error) {
	if opts.DryRun {
		util.L.Sub("[dry-run] would add 'context-mode' to opencode.json plugin[]")
		return true, nil
	}
	op := util.OpenCodePathsResolved()
	_ = util.EnsureDir(op.Dir)
	cfg := loadOrdered(op.Config)
	setContextModePlugin(cfg)
	_ = util.WriteFile(op.Config, util.StringifyJSON(cfg))
	if isTest() {
		return true, nil
	}
	copyOpenCodeAgentsMd(op.Dir)
	cleanAllContextModeCache()
	runPostinstallInOpenCodeCache()
	return true, nil
}

func setContextModePlugin(cfg *util.OrderedMap) {
	plugins := getArr(cfg, "plugin")
	kept := make([]any, 0, len(plugins))
	for _, p := range plugins {
		if s, ok := p.(string); ok && pluginIsContextMode(s) {
			continue
		}
		kept = append(kept, p)
	}
	kept = append(kept, "context-mode")
	cfg.Set("plugin", kept)
	if mv, ok := cfg.Get("mcp"); ok {
		if mm, ok := mv.(*util.OrderedMap); ok {
			if _, has := mm.Get("context-mode"); has {
				mm.Delete("context-mode")
				if mm.Len() == 0 {
					cfg.Delete("mcp")
				}
			}
		}
	}
}

// cleanAllContextModeCache clears stale/dangling cached dirs so bare @latest refetches.
func cleanAllContextModeCache() {
	cacheRoot := filepath.Join(util.Home(), ".cache", "opencode", "packages")
	entries, err := os.ReadDir(cacheRoot)
	if err != nil {
		return
	}
	n := 0
	for _, e := range entries {
		d := e.Name()
		if d == "context-mode" || strings.HasPrefix(d, "context-mode@") {
			_ = os.RemoveAll(filepath.Join(cacheRoot, d))
			n++
		}
	}
	if n > 0 {
		util.L.Sub(util.C.Dim("cleaned " + strconv.Itoa(n) + " old context-mode cache dir(s)"))
	}
}

func runPostinstallInOpenCodeCache() {
	if util.Which("bun") == "" {
		return
	}
	cacheRoot := filepath.Join(util.Home(), ".cache", "opencode", "packages")
	entries, err := os.ReadDir(cacheRoot)
	if err != nil {
		return
	}
	for _, e := range entries {
		d := e.Name()
		if d != "context-mode" && !strings.HasPrefix(d, "context-mode@") {
			continue
		}
		pkgHost := filepath.Join(cacheRoot, d)
		if !util.Exists(filepath.Join(pkgHost, "node_modules", "context-mode")) {
			continue
		}
		r := util.Run(util.ResolveBunBinary(), []string{"pm", "trust", "context-mode"}, util.RunOptions{Cwd: pkgHost, Capture: true})
		if r.Code == 0 {
			util.L.Sub(util.C.Dim("healed OpenCode plugin cache (" + d + ")"))
		}
	}
}

func copyOpenCodeAgentsMd(opencodeDir string) {
	dest := filepath.Join(opencodeDir, "AGENTS.md")
	if util.Exists(dest) {
		return
	}
	if util.Which("npm") == "" {
		return
	}
	root := util.Run(util.ResolveNpmBinary(), []string{"root", "-g"}, util.RunOptions{Capture: true})
	if root.Code != 0 {
		return
	}
	src := filepath.Join(strings.TrimSpace(root.Stdout), "context-mode", "configs", "opencode", "AGENTS.md")
	if !util.Exists(src) {
		return
	}
	if b, err := os.ReadFile(src); err == nil {
		_ = util.WriteFile(dest, string(b))
	}
}

// contextModeInstructionBody loads the upstream context-mode instruction
// template for the named agent.
func contextModeInstructionBody(agent string) string {
	var rel string
	switch agent {
	case "codex":
		rel = filepath.Join("configs", "codex", "AGENTS.md")
	case "antigravity":
		rel = filepath.Join("configs", "antigravity", "GEMINI.md")
	default:
		return ""
	}
	var b []byte
	if ctxDir := findContextModeDir(); ctxDir != "" {
		b, _ = os.ReadFile(filepath.Join(ctxDir, rel))
	}
	if len(b) == 0 {
		root := util.Run(util.ResolveNpmBinary(), []string{"root", "-g"}, util.RunOptions{Capture: true})
		if root.Code != 0 {
			return ""
		}
		b, _ = os.ReadFile(filepath.Join(strings.TrimSpace(root.Stdout), "context-mode", rel))
	}
	if len(b) == 0 {
		return ""
	}
	body := string(b)
	body = regexp.MustCompile(`(?m)^#{1,5}\s`).ReplaceAllStringFunc(body, func(s string) string {
		return "#" + s
	})
	body = strings.ReplaceAll(body,
		"Codex CLI hooks provide runtime enforcement when `[features].hooks = true`; these instructions remain mandatory model-side enforcement. Follow strictly.",
		"Follow the routing rules below.")
	body = strings.ReplaceAll(body,
		"Antigravity has NO hooks — these instructions are ONLY enforcement.",
		"")

	lines := strings.Split(body, "\n")
	var out []string
	skip := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "configs/"+agent) && (strings.Contains(trimmed, "config.toml") || strings.Contains(trimmed, "hooks.json")) {
			skip = true
			continue
		}
		if skip && (trimmed == "" || strings.HasPrefix(trimmed, "```")) {
			skip = false
			if trimmed == "" {
				continue
			}
		}
		if skip {
			continue
		}
		out = append(out, line)
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n")
}

// --- Codex ---

func ctxWireCodex(opts core.RunOpts) (bool, error) {
	if isTest() {
		return wireCodexManual(), nil
	}
	if util.Which("codex") == "" {
		util.L.Err("codex CLI not on PATH — install codex first.")
		return false, nil
	}
	return wireCodexManual(), nil
}

func wireCodexManual() bool {
	cx := util.CodexPathsResolved()
	_ = util.EnsureDir(cx.Dir)
	raw, _ := util.ReadFileSafe(cx.Config)
	raw = util.RemoveBlock(raw, "mcp_servers.context-mode")
	spawn := util.PickMcpSpawn("context-mode")
	block := util.NewTomlBlock("mcp_servers.context_mode")
	block.Set("command", spawn.Command)
	if len(spawn.Args) > 0 {
		block.Set("args", spawn.Args)
	}
	block.Set("enabled", true)
	block.Set("default_tools_approval_mode", "approve")
	_ = util.WriteFile(cx.Config, util.UpsertBlock(raw, block, false))

	cleanupCodexContextModeHooks()
	cleanupWorkspaceCodexContextModeMcp(cx.Dir)
	writeCodexAgentsMd()

	agents.InstallCodexContextModeHook()

	return true
}

// writeCodexAgentsMd writes the upstream context-mode Codex instruction
// template into `~/.codex/AGENTS.md`.
func writeCodexAgentsMd() {
	cx := util.CodexPathsResolved()
	instructionsPath := cx.Instructions
	if instructionsPath == "" {
		return
	}
	body := contextModeInstructionBody("codex")
	if body == "" {
		return
	}
	upsertCodexAgentsMdSection("<!-- CONTEXT-MODE_START -->", "<!-- CONTEXT-MODE_END -->", body)
}

func removeCodexAgentsMdSection(open, close string) {
	cx := util.CodexPathsResolved()
	path := cx.Instructions
	if path == "" {
		return
	}
	existing, ok := util.ReadFileSafe(path)
	if !ok {
		return
	}
	oi := strings.Index(existing, open)
	ci := strings.Index(existing, close)
	if oi < 0 || ci <= oi {
		return
	}
	ci += len(close)
	for oi > 0 && existing[oi-1] == '\n' {
		oi--
	}
	for ci < len(existing) && existing[ci] == '\n' {
		ci++
	}
	next := strings.TrimRight(existing[:oi]+existing[ci:], "\n")
	if strings.TrimSpace(next) == "" {
		_ = os.Remove(path)
		return
	}
	_ = util.WriteFile(path, next+"\n")
}

// upsertCodexAgentsMdSection merges a marked block into ~/.codex/AGENTS.md.
func upsertCodexAgentsMdSection(open, close, body string) {
	cx := util.CodexPathsResolved()
	instructionsPath := cx.Instructions
	if instructionsPath == "" {
		return
	}
	existing := ""
	if raw, ok := util.ReadFileSafe(instructionsPath); ok {
		existing = raw
	}
	oi := strings.Index(existing, open)
	ci := strings.Index(existing, close)
	if oi >= 0 && ci > oi {
		ci += len(close)
		for ci < len(existing) && existing[ci] == '\n' {
			ci++
		}
		r := existing[:oi] + open + "\n" + body + "\n" + close + "\n" + existing[ci:]
		_ = util.WriteFile(instructionsPath, r)
		return
	}
	if existing != "" && !strings.HasSuffix(existing, "\n") {
		existing += "\n"
	}
	if existing != "" && !strings.HasSuffix(existing, "\n\n") {
		existing += "\n"
	}
	_ = util.WriteFile(instructionsPath, existing+open+"\n"+body+"\n"+close+"\n")
}

// removeCodexContextModeHooks removes context-mode hook entries, keeping
// unrelated hooks such as rtk/codegraph/user hooks intact.
func removeCodexContextModeHooks(existing *util.OrderedMap) *util.OrderedMap {
	out := existing
	if out == nil {
		out = util.NewOrderedMap()
	}
	hooks := getOrCreateMapT(out, "hooks")
	for _, event := range []string{"PreToolUse", "PostToolUse", "UserPromptSubmit", "SessionStart", "PreCompact", "Stop", "PermissionRequest"} {
		var arr []any
		if v, ok := hooks.Get(event); ok {
			if a, ok := v.([]any); ok {
				arr = a
			}
		}
		var filtered []any
		for _, entry := range arr {
			if !isOursForEvent(entry, event) {
				filtered = append(filtered, entry)
			}
		}
		if len(filtered) == 0 {
			hooks.Delete(event)
		} else {
			hooks.Set(event, filtered)
		}
	}
	if hooks.Len() == 0 {
		out.Delete("hooks")
	} else {
		out.Set("hooks", hooks)
	}
	return out
}

func cleanupCodexContextModeHooks() {
	cx := util.CodexPathsResolved()
	cleanupCodexContextModeHooksInDir(cx.Dir)
	if cwd, err := os.Getwd(); err == nil {
		projectCodex := filepath.Join(cwd, ".codex")
		if projectCodex != cx.Dir {
			cleanupCodexContextModeHooksInDir(projectCodex)
		}
	}
}

func cleanupWorkspaceCodexContextModeMcp(activeCodexDir string) {
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	projectCodex := filepath.Join(cwd, ".codex")
	if projectCodex == activeCodexDir {
		return
	}
	configPath := filepath.Join(projectCodex, "config.toml")
	raw, ok := util.ReadFileSafe(configPath)
	if !ok {
		return
	}
	next := util.RemoveBlock(raw, "mcp_servers.context-mode")
	next = util.RemoveBlock(next, "mcp_servers.context_mode")
	if next != raw {
		_ = util.WriteFile(configPath, next)
	}
}

func cleanupCodexContextModeHooksInDir(dir string) {
	hooksPath := filepath.Join(dir, "hooks.json")
	if !util.Exists(hooksPath) {
		return
	}
	raw, _ := util.ReadFileSafe(hooksPath)
	next := removeCodexContextModeHooks(loadOrdered(hooksPath))
	if next.Len() == 0 {
		_ = os.Remove(hooksPath)
		return
	}
	if s := util.StringifyJSON(next); s != raw {
		_ = util.WriteFile(hooksPath, s)
	}
}

func isOursForEvent(entry any, event string) bool {
	em, ok := entry.(*util.OrderedMap)
	if !ok {
		return false
	}
	hv, ok := em.Get("hooks")
	if !ok {
		return false
	}
	arr, ok := hv.([]any)
	if !ok {
		return false
	}
	prefix := "context-mode hook codex " + strings.ToLower(event)
	legacyPrefix := "tokless context-mode-hook codex " + strings.ToLower(event)
	altPrefix := "tokless codex-sessionstart"
	for _, h := range arr {
		hm, ok := h.(*util.OrderedMap)
		if !ok {
			continue
		}
		if cmd, ok := hm.Get("command"); ok {
			if s, ok := cmd.(string); ok && (strings.HasPrefix(s, prefix) || strings.HasPrefix(s, legacyPrefix) || strings.HasPrefix(s, altPrefix)) {
				return true
			}
		}
	}
	return false
}

// --- unwire ---

func ctxUnwireClaude(opts core.RunOpts) (bool, error) {
	if opts.DryRun {
		return true, nil
	}
	agents.RemoveClaudeMcp("context-mode")
	return true, nil
}

func ctxUnwireOpenCode(opts core.RunOpts) (bool, error) {
	if opts.DryRun {
		return true, nil
	}
	op := util.OpenCodePathsResolved()
	raw, ok := util.ReadFileSafe(op.Config)
	if !ok {
		return true, nil
	}
	cfg := util.TryParseJsonc(raw)
	if cfg == nil {
		cfg = util.NewOrderedMap()
	}
	if pv, ok := cfg.Get("plugin"); ok {
		if arr, ok := pv.([]any); ok {
			var kept []any
			for _, p := range arr {
				if s, ok := p.(string); ok && pluginIsContextMode(s) {
					continue
				}
				kept = append(kept, p)
			}
			if len(kept) == 0 {
				cfg.Delete("plugin")
			} else {
				cfg.Set("plugin", kept)
			}
		}
	}
	_ = util.WriteFile(op.Config, util.StringifyJSON(cfg))
	return true, nil
}

func ctxUnwireCodex(opts core.RunOpts) (bool, error) {
	if opts.DryRun {
		return true, nil
	}
	cx := util.CodexPathsResolved()
	if raw, ok := util.ReadFileSafe(cx.Config); ok {
		next := util.RemoveBlock(raw, "mcp_servers.context-mode")
		next = util.RemoveBlock(next, "mcp_servers.context_mode")
		if next != raw {
			_ = util.WriteFile(cx.Config, next)
		}
	}
	agents.RemoveCodexContextModeHook()
	cleanupCodexContextModeHooks()
	return true, nil
}

// --- Antigravity (MCP + persistent GEMINI.md routing + one minimal hook) ---

const ctxGeminiMarker = "context-mode — MANDATORY routing rules"

func antigravityRoutingBody() string {
	return contextModeInstructionBody("antigravity")
}

func geminiMdPath() string {
	return filepath.Join(util.Home(), ".gemini", "GEMINI.md")
}

func upsertGeminiMdSection(open, close, content string) {
	p := geminiMdPath()
	existing, ok := util.ReadFileSafe(p)
	if ok {
		oi := strings.Index(existing, open)
		ci := strings.Index(existing, close)
		if oi >= 0 && ci > oi {
			ci += len(close)
			for ci < len(existing) && existing[ci] == '\n' {
				ci++
			}
			r := existing[:oi] + content + "\n" + existing[ci:]
			_ = util.WriteFile(p, strings.TrimRight(r, "\n")+"\n")
			return
		}
		existing = strings.TrimRight(existing, "\n")
		if existing != "" {
			existing += "\n\n"
		}
		_ = util.WriteFile(p, existing+content+"\n")
		return
	}
	_ = util.WriteFile(p, content+"\n")
}

func removeGeminiMdSection(open, close string) {
	p := geminiMdPath()
	existing, ok := util.ReadFileSafe(p)
	if !ok {
		return
	}
	oi := strings.Index(existing, open)
	ci := strings.Index(existing, close)
	if oi < 0 || ci <= oi {
		return
	}
	ci += len(close)
	for oi > 0 && existing[oi-1] == '\n' {
		oi--
	}
	for ci < len(existing) && existing[ci] == '\n' {
		ci++
	}
	c := existing[:oi] + existing[ci:]
	c = strings.TrimRight(c, "\n")
	if strings.TrimSpace(c) == "" {
		_ = os.Remove(p)
		return
	}
	_ = util.WriteFile(p, c+"\n")
}

func ctxWireAntigravity(opts core.RunOpts) (bool, error) {
	if opts.DryRun {
		util.L.Sub("[dry-run] would add context-mode MCP and GEMINI.md routing for antigravity")
		return true, nil
	}
	agents.ConfigureAntigravityMcp("context-mode")
	agents.CleanupLegacyAntigravityContextMode()
	agents.RemoveAntigravityEntry("command(echo)")

	body := antigravityRoutingBody()
	if body != "" {
		section := "<!-- CONTEXT-MODE_START -->\n" + body + "\n<!-- CONTEXT-MODE_END -->"
		upsertGeminiMdSection("<!-- CONTEXT-MODE_START -->", "<!-- CONTEXT-MODE_END -->", section)
	}

	return ctxVerifyAntigravity(), nil
}

// findContextModeDir finds the context-mode npm package root directory.
func findContextModeDir() string {
	if util.Which("npm") != "" {
		root := util.Run(util.ResolveNpmBinary(), []string{"root", "-g"}, util.RunOptions{Capture: true})
		if root.Code == 0 {
			ctxRoot := strings.TrimSpace(root.Stdout)
			ctxPkg := filepath.Join(ctxRoot, "context-mode")
			if util.Exists(filepath.Join(ctxPkg, "configs", "antigravity", "GEMINI.md")) {
				return ctxPkg
			}
		}
	}
	if util.Which("node") != "" {
		script := "try{const p=require.resolve('context-mode/package.json');process.stdout.write(require('path').dirname(p))}catch(e){}"
		r := util.Run(util.ResolveNodeBinary(), []string{"-e", script}, util.RunOptions{Capture: true})
		if r.Code == 0 && strings.TrimSpace(r.Stdout) != "" {
			ctxRoot := strings.TrimSpace(r.Stdout)
			if util.Exists(filepath.Join(ctxRoot, "configs", "antigravity", "GEMINI.md")) {
				return ctxRoot
			}
		}
	}
	if cm := util.Which("context-mode"); cm != "" {
		script := "const{realpathSync}=require('fs');const{dirname,join}=require('path');const r=realpathSync(process.argv[1]);process.stdout.write(join(dirname(dirname(r)),'context-mode'))"
		r := util.Run(util.ResolveNodeBinary(), []string{"-e", script, cm}, util.RunOptions{Capture: true})
		if r.Code == 0 && strings.TrimSpace(r.Stdout) != "" {
			ctxRoot := strings.TrimSpace(r.Stdout)
			if util.Exists(filepath.Join(ctxRoot, "configs", "antigravity", "GEMINI.md")) {
				return ctxRoot
			}
		}
	}
	return ""
}

func ctxUnwireAntigravity(opts core.RunOpts) (bool, error) {
	if opts.DryRun {
		util.L.Sub("[dry-run] would remove context-mode MCP and GEMINI.md routing")
		return true, nil
	}
	agents.RemoveAntigravityMcp("context-mode")
	agents.CleanupLegacyAntigravityContextMode()
	removeGeminiMdSection("<!-- CONTEXT-MODE_START -->", "<!-- CONTEXT-MODE_END -->")
	if cwd, err := os.Getwd(); err == nil {
		dest := filepath.Join(cwd, "GEMINI.md")
		if raw, ok := util.ReadFileSafe(dest); ok && strings.Contains(raw, ctxGeminiMarker) {
			_ = os.Remove(dest)
		}
	}
	return true, nil
}

func ctxVerifyAntigravity() bool {
	return agents.AntigravityMcpHas("context-mode")
}

// --- verify ---

func ctxVerifyClaude() bool {
	cp := util.ClaudeCodePaths()
	raw, ok := util.ReadFileSafe(cp.GlobalJSON)
	if !ok {
		return false
	}
	cfg := util.TryParseJsonc(raw)
	if cfg == nil {
		return false
	}
	if s, ok := cfg.Get("mcpServers"); ok {
		if sm, ok := s.(*util.OrderedMap); ok {
			_, has := sm.Get("context-mode")
			return has
		}
	}
	return false
}

func ctxVerifyOpenCode() bool {
	op := util.OpenCodePathsResolved()
	raw, ok := util.ReadFileSafe(op.Config)
	if !ok {
		return false
	}
	cfg := util.TryParseJsonc(raw)
	if cfg == nil {
		return false
	}
	hasPlugin := false
	if pv, ok := cfg.Get("plugin"); ok {
		if arr, ok := pv.([]any); ok {
			for _, p := range arr {
				if s, ok := p.(string); ok && pluginIsContextMode(s) {
					hasPlugin = true
					break
				}
			}
		}
	}
	hasLegacy := false
	if mv, ok := cfg.Get("mcp"); ok {
		if mm, ok := mv.(*util.OrderedMap); ok {
			_, hasLegacy = mm.Get("context-mode")
		}
	}
	return hasPlugin && !hasLegacy
}

func ctxVerifyCodex() bool {
	cx := util.CodexPathsResolved()
	raw, ok := util.ReadFileSafe(cx.Config)
	if !ok {
		return false
	}
	if !strings.Contains(raw, "[mcp_servers.context_mode]") {
		return false
	}
	if raw, ok := util.ReadFileSafe(cx.Instructions); !ok || !strings.Contains(raw, "<!-- CONTEXT-MODE_START -->") {
		return false
	}
	if !agents.HasCodexContextModeHook() {
		return false
	}
	if util.Exists(filepath.Join(cx.Dir, "hooks.json")) {
		data := loadOrdered(filepath.Join(cx.Dir, "hooks.json"))
		if hv, ok := data.Get("hooks"); ok {
			if hm, ok := hv.(*util.OrderedMap); ok {
				for _, ev := range []string{"PreCompact", "Stop", "SessionStart", "PostToolUse", "UserPromptSubmit", "PermissionRequest"} {
					if hasCodexHookEntry(hm, ev, "") {
						return false
					}
				}
			}
		}
	}
	return true
}

// hasCodexHookEntry returns true when hooks.json contains one of context-mode's entries.
func hasCodexHookEntry(hooks *util.OrderedMap, event, _ string) bool {
	v, ok := hooks.Get(event)
	if !ok {
		return false
	}
	arr, ok := v.([]any)
	if !ok {
		return false
	}
	for _, entry := range arr {
		if isOursForEvent(entry, event) {
			return true
		}
	}
	return false
}

var contextMode = &core.ToolManifest{
	ID:           "context-mode",
	Label:        "Context-Mode",
	Description:  "Routes long context off-thread to a sandbox, keeping the agent's window small.",
	Homepage:     "https://github.com/mksglu/context-mode",
	InstallHint:  "npm i -g context-mode",
	Channel:      core.ChannelNpm,
	MinNodeMajor: contextModeMinNode,
	Install:      ctxEnsureInstalled,
	WireFor: map[string]core.AgentFn{
		"claude":      ctxWireClaude,
		"opencode":    ctxWireOpenCode,
		"codex":       ctxWireCodex,
		"antigravity": ctxWireAntigravity,
	},
	UnwireFor: map[string]core.AgentFn{
		"claude":      ctxUnwireClaude,
		"opencode":    ctxUnwireOpenCode,
		"codex":       ctxUnwireCodex,
		"antigravity": ctxUnwireAntigravity,
	},
	VerifyFor: map[string]core.VerifyFn{
		"claude":      func() *bool { return core.BoolPtr(ctxVerifyClaude()) },
		"opencode":    func() *bool { return core.BoolPtr(ctxVerifyOpenCode()) },
		"codex":       func() *bool { return core.BoolPtr(ctxVerifyCodex()) },
		"antigravity": func() *bool { return core.BoolPtr(ctxVerifyAntigravity()) },
	},
}

// Register wires all tools into the core registry, in canonical order.
// MD-block write order is determined by this sequence:
// caveman → codegraph → context-mode → rtk.
func Register() {
	core.RegisterTool(rtk)
	core.RegisterTool(caveman)
	core.RegisterTool(codegraph)
	core.RegisterTool(contextMode)
}

// helpers shared in tools package

func loadOrdered(path string) *util.OrderedMap {
	if raw, ok := util.ReadFileSafe(path); ok {
		if m := util.TryParseJsonc(raw); m != nil {
			return m
		}
	}
	return util.NewOrderedMap()
}

func getArr(m *util.OrderedMap, key string) []any {
	if v, ok := m.Get(key); ok {
		if a, ok := v.([]any); ok {
			return a
		}
	}
	return []any{}
}

func toAny(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}
