package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/HoangP8/tokless/internal/util"
)

var codexPermAllowlist = map[string]bool{
	"rtk": true, "tokless": true, "git": true, "cd": true, "ls": true,
	"node": true, "npm": true, "npx": true,
	"context-mode": true, "codegraph": true,
	"cat": true, "head": true, "tail": true,
	"grep": true, "find": true, "pwd": true,
	"which": true, "echo": true, "true": true, "false": true,
	"bash": true,
}

func RunCodexPermHook() int {
	input, err := io.ReadAll(os.Stdin)
	if err != nil || len(input) == 0 {
		return 0
	}
	var req struct {
		ToolName  string `json:"tool_name"`
		ToolInput struct {
			Command string `json:"command"`
		} `json:"tool_input"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return 0
	}
	if !codexPermAllow(req.ToolName, req.ToolInput.Command) {
		return 0
	}
	resp := struct {
		HookSpecificOutput struct {
			HookEventName string `json:"hookEventName"`
			Decision      struct {
				Behavior string `json:"behavior"`
			} `json:"decision"`
		} `json:"hookSpecificOutput"`
	}{}
	resp.HookSpecificOutput.HookEventName = "PermissionRequest"
	resp.HookSpecificOutput.Decision.Behavior = "allow"
	if out, err := json.Marshal(resp); err == nil {
		fmt.Println(string(out))
	}
	return 0
}

func codexPermAllow(toolName, command string) bool {
	if toolName == "apply_patch" {
		return true
	}
	if strings.HasPrefix(toolName, "ctx_") || strings.HasPrefix(toolName, "codegraph_") {
		return true
	}
	if toolName != "Bash" {
		return false
	}
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return false
	}
	tok := firstToken(cmd)
	tok = stripPath(tok)
	if tok == "bash" || tok == "sh" {
		return bashInnerScriptAllAllowed(cmd)
	}
	return codexPermAllowlist[tok]
}

func firstToken(s string) string {
	if sp := strings.IndexAny(s, " \t"); sp >= 0 {
		return s[:sp]
	}
	return s
}

func stripPath(tok string) string {
	if idx := strings.LastIndexByte(tok, '/'); idx >= 0 {
		tok = tok[idx+1:]
	}
	if util.IsWin {
		if idx := strings.LastIndexByte(tok, '\\'); idx >= 0 {
			tok = tok[idx+1:]
		}
		tok = strings.TrimSuffix(strings.TrimSuffix(tok, ".exe"), ".cmd")
		tok = strings.TrimSuffix(tok, ".bat")
	}
	return tok
}

func bashInnerScriptAllAllowed(cmd string) bool {
	rest := cmd
	for _, flag := range []string{"-lc", "-c"} {
		if idx := strings.Index(rest, flag); idx >= 0 {
			rest = strings.TrimSpace(rest[idx+len(flag):])
			break
		}
	}
	if rest == "" {
		return false
	}
	if (rest[0] == '"' && rest[len(rest)-1] == '"') ||
		(rest[0] == '\'' && rest[len(rest)-1] == '\'') {
		rest = rest[1 : len(rest)-1]
	}
	segments := splitScript(rest)
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		tok := stripPath(firstToken(seg))
		if !codexPermAllowlist[tok] {
			return false
		}
	}
	return true
}

func splitScript(s string) []string {
	s = strings.ReplaceAll(s, "||", "\x00")
	s = strings.ReplaceAll(s, "&&", "\x00")
	s = strings.ReplaceAll(s, "|", "\x00")
	s = strings.ReplaceAll(s, ";", "\x00")
	return strings.Split(s, "\x00")
}