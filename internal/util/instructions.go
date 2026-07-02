package util

import (
	_ "embed"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"time"
)

// McpInstructions probes an MCP server and returns result.instructions.
func McpInstructions(spawn McpSpawn) (string, bool) {
	if os.Getenv("TOKLESS_TEST") == "1" {
		return "", false
	}
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"tokless","version":"0"}}}` + "\n"
	cmd := exec.Command(spawn.Command, spawn.Args...)
	cmd.Stdin = strings.NewReader(initReq)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", false
	}
	if err := cmd.Start(); err != nil {
		return "", false
	}
	defer cmd.Process.Kill()

	type result struct {
		data []byte
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		buf := make([]byte, 0, 64*1024)
		tmp := make([]byte, 4096)
		for {
			n, err := stdout.Read(tmp)
			if n > 0 {
				buf = append(buf, tmp[:n]...)
				if idx := strings.IndexByte(string(buf), '\n'); idx >= 0 {
					ch <- result{data: buf[:idx], err: nil}
					return
				}
			}
			if err != nil {
				ch <- result{data: buf, err: err}
				return
			}
		}
	}()
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	select {
	case r := <-ch:
		cmd.Wait()
		if len(r.data) == 0 {
			return "", false
		}
		var resp struct {
			Result struct {
				Instructions string `json:"instructions"`
			} `json:"result"`
		}
		if err := json.Unmarshal(r.data, &resp); err != nil {
			return "", false
		}
		if resp.Result.Instructions == "" {
			return "", false
		}
		return resp.Result.Instructions, true
	case <-timer.C:
		return "", false
	}
}

// ToklessOwners is render order: meta rules first, then tools.
var ToklessOwners = []string{
	"principles",
	"caveman",
	"ponytail",
	"codegraph",
	"context-mode",
}

// SectionsByOwner maps each owner to its heading marker.
var SectionsByOwner = map[string]string{
	"principles":   "## Principles",
	"caveman":      "## Response Style (caveman)",
	"ponytail":     "## Build Discipline (ponytail)",
	"codegraph":    "## Code Index (codegraph)",
	"context-mode": "## Context Tools (context-mode)",
}

var legacySectionsByOwner = map[string][]string{
	"principles":   {"## 1. Principles", "## Principles (craft) →", "## Principles (craft)"},
	"caveman":      {"## 2. Response Style", "## Response Style", "## Style", "## Caveman Style", "## Caveman", "## Voice (caveman)", "## Response Style (caveman)"},
	"ponytail":     {"## 3. Build Discipline", "## Build Discipline", "## Build Less", "## Ponytail", "## Ponytail: Build Less", "## Reuse Ladder (ponytail)", "## Lazy Ladder (ponytail)", "## Build Discipline (ponytail)"},
	"codegraph":    {"## 4. Code Search", "## Codegraph", "## Codegraph — MUST USE FOR CODE", "## Code Index (codegraph)"},
	"context-mode": {"## 5. Context Control", "## Context Tools", "## Context Tools — MUST USE FOR DATA", "## Context Tools (context-mode)"},
}

func SectionPresent(body, owner string) bool {
	for _, marker := range SectionMarkers(owner) {
		if strings.Contains(body, marker) {
			return true
		}
	}
	return false
}

func SectionMarkers(owner string) []string {
	marker, ok := SectionsByOwner[owner]
	if !ok {
		return nil
	}
	markers := []string{marker}
	markers = append(markers, legacySectionsByOwner[owner]...)
	return markers
}

//go:embed agent_instructions.md
var agentInstructionsTemplate string

func instructionIndexSection() string {
	body := strings.TrimRight(agentInstructionsTemplate, "\n")
	idx := strings.Index(body, "\n## ")
	if idx < 0 {
		return body
	}
	return body[:idx]
}

func instructionSection(owner string) string {
	marker := SectionsByOwner[owner]
	if marker == "" {
		return ""
	}
	body := strings.TrimRight(agentInstructionsTemplate, "\n")
	start := strings.Index(body, marker)
	if start < 0 {
		return ""
	}
	if start > 0 {
		start = strings.LastIndex(body[:start], "\n") + 1
	}
	rest := body[start:]
	if idx := strings.Index(rest[1:], "\n## "); idx >= 0 {
		return strings.TrimRight(rest[:idx+1], "\n")
	}
	return strings.TrimRight(rest, "\n")
}

// ToklessAgentBody renders the full markdown body for the given owners.
func ToklessAgentBody(owners []string) string {
	var b strings.Builder

	if len(owners) >= 2 {
		b.WriteString(instructionIndexSection())
		b.WriteString("\n\n")
	}
	if len(owners) > 0 {
		b.WriteString(instructionSection("principles"))
		b.WriteString("\n\n")
	}
	if hasOwner(owners, "caveman") {
		b.WriteString(instructionSection("caveman"))
		b.WriteString("\n\n")
	}
	if hasOwner(owners, "ponytail") {
		b.WriteString(instructionSection("ponytail"))
		b.WriteString("\n\n")
	}
	if hasOwner(owners, "codegraph") {
		b.WriteString(instructionSection("codegraph"))
		b.WriteString("\n\n")
	}
	if hasOwner(owners, "context-mode") {
		b.WriteString(instructionSection("context-mode"))
		b.WriteString("\n\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// ToklessBody returns the rendered body. Convenience over ToklessAgentBody.
func ToklessBody(owners []string) string { return ToklessAgentBody(owners) }

// TokenizeBody infers active owners from section headings present in body.
func TokenizeBody(body string) []string {
	var out []string
	for _, owner := range ToklessOwners {
		if SectionPresent(body, owner) {
			out = append(out, owner)
		}
	}
	return out
}

func hasOwner(owners []string, want string) bool {
	for _, o := range owners {
		if o == want {
			return true
		}
	}
	return false
}
