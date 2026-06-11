package util

import (
	"path/filepath"
	"strings"
)

// McpSpawn is the command shape written into an agent's MCP config entry.
type McpSpawn struct {
	Command string
	Args    []string
}

var pkgForBin = map[string]string{
	"context-mode": "context-mode",
	"codegraph":    "@colbymchenry/codegraph",
}

// PickMcpSpawn prefers a real binary on PATH, else falls back to npx --no-install.
func PickMcpSpawn(bin string, extraArgs ...string) McpSpawn {
	if extraArgs == nil {
		extraArgs = []string{}
	}
	if Which(bin) != "" {
		return wrapCmdShim(McpSpawn{Command: bin, Args: extraArgs})
	}
	pkg, ok := pkgForBin[bin]
	if !ok {
		pkg = bin
	}
	args := append([]string{"--no-install", pkg}, extraArgs...)
	return wrapCmdShim(McpSpawn{Command: "npx", Args: args})
}

func wrapCmdShim(s McpSpawn) McpSpawn {
	if !IsWin {
		return s
	}
	p := Which(s.Command)
	ext := strings.ToLower(filepath.Ext(p))
	if ext != ".cmd" && ext != ".bat" {
		return s
	}
	return McpSpawn{Command: "cmd", Args: append([]string{"/c", s.Command}, s.Args...)}
}
