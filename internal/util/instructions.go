package util

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"time"
)

// McpInstructions probes an MCP server via the JSON-RPC initialize handshake
// and returns the result.instructions field.
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

const (
	CodegraphMarkerStart = "<!-- CODEGRAPH_START -->"
	CodegraphMarkerEnd   = "<!-- CODEGRAPH_END -->"
)

const CodegraphAgentBlock = "## CodeGraph\n\n" +
	"In repositories indexed by CodeGraph (a `.codegraph/` directory exists at the repo root), reach for it BEFORE grep/find or reading files when you need to understand or locate code:\n\n" +
	"- **MCP tool** (when available): `codegraph_explore` answers most code questions in one call — the relevant symbols' verbatim source plus the call paths between them, including dynamic-dispatch hops grep can't follow. Name a file or symbol in the query to read its current line-numbered source. If it's listed but deferred, load it by name via tool search.\n" +
	"- **Shell** (always works): `codegraph explore \"<symbol names or question>\"` prints the same output.\n\n" +
	"If there is no `.codegraph/` directory, skip CodeGraph entirely — indexing is the user's decision."