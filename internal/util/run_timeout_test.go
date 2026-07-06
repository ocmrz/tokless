package util

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

func hangingCommand(t *testing.T) (string, []string) {
	t.Helper()
	if os.Getenv("TOKLESS_RUN_TIMEOUT_HELPER") == "1" {
		select {}
	}
	return os.Args[0], []string{"-test.run=TestRunCtxTimeoutHelper", "--"}
}

func TestRunCtxTimeoutHelper(t *testing.T) {
	if os.Getenv("TOKLESS_RUN_TIMEOUT_HELPER") != "1" {
		return
	}
	select {}
}

func TestRunCtxTimeoutKillsHangingSubprocess(t *testing.T) {
	if testing.Short() {
		t.Skip("uses real sleep subprocess")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	bin, args := hangingCommand(t)

	start := time.Now()
	r := Run(bin, args, RunOptions{Capture: true, Ctx: ctx, Env: []string{"TOKLESS_RUN_TIMEOUT_HELPER=1"}})
	elapsed := time.Since(start)

	if elapsed > 3*time.Second {
		t.Fatalf("Run did not honor Ctx timeout: elapsed=%v (want <3s)", elapsed)
	}
	if r.Code == 0 {
		t.Fatalf("hanging subprocess exited 0; expected killed. elapsed=%v", elapsed)
	}
}

func TestRunCtxTimeoutBoundedCaptureLogs(t *testing.T) {
	if testing.Short() {
		t.Skip("uses real sleep subprocess")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	bin, args := hangingCommand(t)

	done := make(chan struct{})
	var logs string
	go func() {
		logs, _ = CaptureLogs(func() error {
			Run(bin, args, RunOptions{Capture: true, Ctx: ctx, Env: []string{"TOKLESS_RUN_TIMEOUT_HELPER=1"}})
			return nil
		})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("CaptureLogs did not return after subprocess timeout")
	}
	if os.Stdout == nil {
		t.Fatal("os.Stdout nil after CaptureLogs")
	}
	_ = logs
}

func TestRunCtxTimeoutKillsChildHoldingPipe(t *testing.T) {
	if os.Getenv("TOKLESS_RUN_TIMEOUT_CHILD") == "1" {
		cmd := exec.Command(os.Args[0], "-test.run=TestRunCtxTimeoutHelper", "--")
		cmd.Env = append(os.Environ(), "TOKLESS_RUN_TIMEOUT_HELPER=1")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		select {}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	start := time.Now()
	r := Run(os.Args[0], []string{"-test.run=TestRunCtxTimeoutKillsChildHoldingPipe", "--"}, RunOptions{
		Capture: true,
		Ctx:     ctx,
		Env:     []string{"TOKLESS_RUN_TIMEOUT_CHILD=1"},
	})
	elapsed := time.Since(start)

	if elapsed > 3*time.Second {
		t.Fatalf("Run did not kill child holding pipe: elapsed=%v (want <3s)", elapsed)
	}
	if r.Code == 0 {
		t.Fatalf("subprocess tree exited 0; expected killed. elapsed=%v", elapsed)
	}
}
