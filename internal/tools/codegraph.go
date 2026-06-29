package tools

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/HoangP8/tokless/internal/agents"
	"github.com/HoangP8/tokless/internal/core"
	"github.com/HoangP8/tokless/internal/util"
)

func codegraphEnsureInstalled(opts core.RunOpts) (bool, error) {
	if isTest() {
		dest := filepath.Join(util.Home(), ".local", "bin")
		_ = os.MkdirAll(dest, 0o755)
		cgPath := filepath.Join(dest, "codegraph")
		_ = os.Remove(cgPath)
		_ = os.WriteFile(cgPath, []byte("#!/bin/sh\necho ok"), 0o755)
		sep := ":"
		if util.IsWin {
			sep = ";"
		}
		cur := os.Getenv("PATH")
		if !strings.Contains(sep+cur+sep, sep+dest+sep) {
			os.Setenv("PATH", dest+sep+cur)
		}
		return true, nil
	}
	opts.Reportf("checking", 0.1)
	if util.Which("codegraph") != "" && !opts.Upgrade {
		opts.Reportf("already installed", 1)
		return true, nil
	}
	if opts.DryRun {
		util.L.Sub("[dry-run] would install @colbymchenry/codegraph globally")
		return true, nil
	}
	opts.Reportf("npm install -g", 0.4)
	_, ok, _ := util.NpmGlobalInstall("@colbymchenry/codegraph", "latest")
	if ok {
		opts.Reportf("ready", 1)
	}
	return ok, nil
}

var (
	realInstallOnce sync.Once
	realInstallRes  bool
)

// codegraphRealInstall runs `codegraph install --target <agent>` per call.
func codegraphRealInstall(opts core.RunOpts, agent string) bool {
	if opts.DryRun {
		util.L.Sub("[dry-run] would run: codegraph install --yes")
		return true
	}
	help := util.Run(util.ResolveCodegraphBin(), []string{"install", "--help"}, util.RunOptions{Capture: true})
	hasYes := strings.Contains(help.Stdout, "--yes") || strings.Contains(help.Stderr, "--yes")
	hasTarget := strings.Contains(help.Stdout, "--target") || strings.Contains(help.Stderr, "--target")
	args := []string{"install"}
	if hasYes {
		args = append(args, "--yes")
	}
	if hasTarget {
		target := agent
		if target == "antigravity" {
			target = "gemini"
		}
		if target == "" {
			target = "all"
		}
		args = append(args, "--target", target)
	}
	return util.Run(util.ResolveCodegraphBin(), args, util.RunOptions{Capture: true}).Code == 0
}

// codegraphConfigureMcp writes the MCP entry tokless-side.
func codegraphConfigureMcp(agent string) bool {
	switch agent {
	case "claude":
		agents.ConfigureClaudeMcp("codegraph")
	case "opencode":
		agents.ConfigureOpenCodeMcp("codegraph")
	case "codex":
		agents.ConfigureCodexMcp("codegraph")
	case "antigravity":
		agents.ConfigureAntigravityMcp("codegraph")
	}
	return true
}

func codegraphVerify(agent string) bool {
	switch agent {
	case "claude":
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
				_, has := sm.Get("codegraph")
				return has
			}
		}
		return false
	case "opencode":
		op := util.OpenCodePathsResolved()
		raw, ok := util.ReadFileSafe(op.Config)
		if !ok {
			return false
		}
		cfg := util.TryParseJsonc(raw)
		if cfg == nil {
			return false
		}
		if m, ok := cfg.Get("mcp"); ok {
			if mm, ok := m.(*util.OrderedMap); ok {
				_, has := mm.Get("codegraph")
				return has
			}
		}
		return false
	case "codex":
		cx := util.CodexPathsResolved()
		raw, _ := util.ReadFileSafe(cx.Config)
		return strings.Contains(raw, "[mcp_servers.codegraph]")
	case "antigravity":
		agents.CleanupDeadIdeHooks()
		return agents.AntigravityMcpHas("codegraph") && agents.HasAntigravityCodegraphIndexHook()
	}
	return false
}

func codegraphIndexed(dir string, _ core.RunOpts) bool {
	return util.Exists(filepath.Join(dir, ".codegraph"))
}

func codegraphIndexProject(dir string, opts core.RunOpts) (bool, error) {
	if isTest() {
		_ = os.MkdirAll(filepath.Join(dir, ".codegraph"), 0o755)
		return true, nil
	}
	if util.Which("codegraph") == "" {
		return false, nil
	}
	if opts.DryRun {
		util.L.Sub("[dry-run] would run codegraph in " + dir)
		return true, nil
	}
	hasIndex := util.Exists(filepath.Join(dir, ".codegraph"))
	var res util.ExecResult
	if hasIndex {
		res = util.Run(util.ResolveCodegraphBin(), []string{"sync"}, util.RunOptions{Cwd: dir, Capture: true})
		if res.Code == 0 {
			return true, nil
		}
	}
	res = util.Run(util.ResolveCodegraphBin(), []string{"init", "-i"}, util.RunOptions{Cwd: dir, Capture: true})
	if res.Code != 0 {
		res = util.Run(util.ResolveCodegraphBin(), []string{"init"}, util.RunOptions{Cwd: dir, Capture: true})
	}
	return res.Code == 0, nil
}

func codegraphWire(agent string) core.AgentFn {
	return func(opts core.RunOpts) (bool, error) {
		if isTest() {
			codegraphConfigureMcp(agent)
			if agent == "antigravity" {
				agents.InstallAntigravityCodegraphIndexHook()
				agents.InstallAntigravityCodegraphToolDefs()
				agents.CleanupDeadIdeHooks()
			}
			return codegraphVerify(agent), nil
		}
		if opts.DryRun {
			return codegraphRealInstall(opts, agent), nil
		}
		if ran := codegraphRealInstall(opts, agent); !ran {
			util.L.Debug("codegraph's own installer failed; writing MCP entry directly")
		}
		codegraphConfigureMcp(agent)
		writeCodegraphBlock(agent)
		unwireAutoIndex(agent)
		if agent == "antigravity" {
			agents.InstallAntigravityCodegraphIndexHook()
			agents.InstallAntigravityCodegraphToolDefs()
			agents.CleanupDeadIdeHooks()
		}
		return codegraphVerify(agent), nil
	}
}

// writeCodegraphBlock strips any existing CODEGRAPH_START/END block from the
// agent's instructions file and inserts the canonical block at the position
// defined by the order: caveman → codegraph → CONTEXT-MODE.
func writeCodegraphBlock(agent string) bool {
	var path string
	switch agent {
	case "claude":
		path = util.ClaudeCodePaths().Instructions
	case "opencode":
		path = util.OpenCodePathsResolved().Instructions
	case "codex":
		path = util.CodexPathsResolved().Instructions
	case "antigravity":
		path = util.AntigravityPathsResolved().Instructions
	default:
		return false
	}

	const block = util.CodegraphMarkerStart + "\n" + util.CodegraphAgentBlock + "\n" + util.CodegraphMarkerEnd

	raw, ok := util.ReadFileSafe(path)
	if !ok {
		_ = util.EnsureDir(filepath.Dir(path))
		return util.WriteFile(path, block+"\n") == nil
	}

	for {
		oi := strings.Index(raw, util.CodegraphMarkerStart)
		if oi < 0 {
			break
		}
		ci := strings.Index(raw[oi:], util.CodegraphMarkerEnd)
		if ci < 0 {
			break
		}
		end := oi + ci + len(util.CodegraphMarkerEnd)
		if oi > 0 && raw[oi-1] == '\n' {
			oi--
		}
		for end < len(raw) && raw[end] == '\n' {
			end++
			if end-oi > len(util.CodegraphMarkerStart)+len(util.CodegraphAgentBlock)+len(util.CodegraphMarkerEnd)+8 {
				break
			}
		}
		raw = raw[:oi] + raw[end:]
	}

	insertAt := -1
	if i := strings.Index(raw, "<!-- caveman-end -->"); i >= 0 {
		j := i + len("<!-- caveman-end -->")
		for j < len(raw) && raw[j] == '\n' {
			j++
		}
		insertAt = j
	} else if i := strings.Index(raw, "<!-- CONTEXT-MODE_START -->"); i >= 0 {
		if i > 0 {
			k := i
			for k > 0 && raw[k-1] == '\n' {
				k--
			}
			insertAt = k
		}
	}
	if insertAt < 0 {
		sep := "\n\n"
		if !strings.HasSuffix(raw, "\n") {
			sep = "\n\n"
		} else if strings.HasSuffix(raw, "\n\n") {
			sep = ""
		}
		next := strings.TrimRight(raw, "\n") + sep + block + "\n"
		if next == raw {
			return false
		}
		return util.WriteFile(path, next) == nil
	}

	var next string
	if insertAt > 0 && raw[insertAt-1] != '\n' {
		next = raw[:insertAt] + "\n" + block + "\n\n" + raw[insertAt:]
	} else if insertAt > 0 && raw[insertAt-1] == '\n' {
		next = raw[:insertAt] + block + "\n\n" + raw[insertAt:]
	} else {
		next = block + "\n\n" + raw[insertAt:]
	}
	if next == raw {
		return false
	}
	return util.WriteFile(path, next) == nil
}

func unwireAutoIndex(agent string) {
	switch agent {
	case "claude":
		unwireClaudeAutoIndex()
	case "codex":
		unwireCodexAutoIndex()
	case "opencode":
		unwireOpencodeAutoIndex()
	case "antigravity":
		unwireGeminiAutoIndex()
	}
}

var codegraph = &core.ToolManifest{
	ID:           "codegraph",
	Label:        "CodeGraph",
	Description:  "MCP server that lets agents query a code knowledge graph instead of reading raw files.",
	Homepage:     "https://github.com/colbymchenry/codegraph",
	InstallHint:  "npm i -g @colbymchenry/codegraph",
	Channel:      core.ChannelNpm,
	Install:      codegraphEnsureInstalled,
	IndexProject: codegraphIndexProject,
	Indexed:      codegraphIndexed,
	IndexReady:   func() bool { return isTest() || util.Which("codegraph") != "" },
	WireFor: map[string]core.AgentFn{
		"claude":      codegraphWire("claude"),
		"opencode":    codegraphWire("opencode"),
		"codex":       codegraphWire("codex"),
		"antigravity": codegraphWire("antigravity"),
	},
	UnwireFor: map[string]core.AgentFn{
		"claude": func(core.RunOpts) (bool, error) {
			agents.RemoveClaudeMcp("codegraph")
			unwireAutoIndex("claude")
			return true, nil
		},
		"opencode": func(core.RunOpts) (bool, error) {
			agents.RemoveOpenCodeMcp("codegraph")
			unwireAutoIndex("opencode")
			return true, nil
		},
		"codex": func(core.RunOpts) (bool, error) {
			cx := util.CodexPathsResolved()
			raw, ok := util.ReadFileSafe(cx.Config)
			if ok {
				next := util.RemoveBlock(raw, "mcp_servers.codegraph")
				if next != raw {
					_ = util.WriteFile(cx.Config, next)
				}
			}
			unwireAutoIndex("codex")
			return true, nil
		},
		"antigravity": func(core.RunOpts) (bool, error) {
			agents.RemoveAntigravityMcp("codegraph")
			unwireAutoIndex("antigravity")
			agents.CleanupDeadIdeHooks()
			agents.RemoveAntigravityCodegraphToolDefs()
			return true, nil
		},
	},
	VerifyFor: map[string]core.VerifyFn{
		"claude":      func() *bool { return core.BoolPtr(codegraphVerify("claude")) },
		"opencode":    func() *bool { return core.BoolPtr(codegraphVerify("opencode")) },
		"codex":       func() *bool { return core.BoolPtr(codegraphVerify("codex")) },
		"antigravity": func() *bool { return core.BoolPtr(codegraphVerify("antigravity")) },
	},
}
