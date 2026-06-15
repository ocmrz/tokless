package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// RunRtkHook handles the transparent command rewriting for Antigravity's PreToolUse hook.
func RunRtkHook() int {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return 0 // recover -> exit 0, no output
	}
	if len(input) == 0 {
		return 0
	}

	var req struct {
		ToolCall struct {
			Name string                 `json:"name"`
			Args map[string]interface{} `json:"args"`
		} `json:"toolCall"`
	}

	if err := json.Unmarshal(input, &req); err != nil {
		return 0
	}

	if req.ToolCall.Name != "run_command" {
		return 0
	}

	cmdLineObj, ok := req.ToolCall.Args["CommandLine"]
	if !ok {
		return 0
	}

	cmdLine, ok := cmdLineObj.(string)
	if !ok || cmdLine == "" {
		return 0
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "rtk", "rewrite", cmdLine)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	_ = cmd.Run()

	newCmd := strings.TrimSpace(stdout.String())
	if newCmd == "" || strings.HasPrefix(newCmd, "No rewrite") || newCmd == cmdLine {
		return 0
	}

	req.ToolCall.Args["CommandLine"] = newCmd

	var resp struct {
		Decision  string                 `json:"decision"`
		Overwrite map[string]interface{} `json:"overwrite"`
	}
	resp.Decision = "modify"
	resp.Overwrite = req.ToolCall.Args

	out, err := json.Marshal(resp)
	if err == nil {
		fmt.Println(string(out))
	}
	return 0
}