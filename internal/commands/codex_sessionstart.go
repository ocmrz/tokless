package commands

import (
	"fmt"
	"io"
	"os/exec"
)

// RunCodexSessionStart wraps context-mode's sessionstart hook.
func RunCodexSessionStart() int {
	cmd := exec.Command("context-mode", "hook", "codex", "sessionstart")
	stdin, _ := io.Pipe()
	cmd.Stdin = stdin
	_ = cmd.Start()
	stdin.Close()
	_ = cmd.Wait()

	fmt.Println(`{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"Use ctx_execute/ctx_search/ctx_batch_execute/ctx_execute_file for code analysis, ctx_fetch_and_index for web docs, ctx_index to store content. Prefer these over Bash/Read/Grep to keep context small. Run 'tokless doctor' to diagnose."}}`)
	return 0
}