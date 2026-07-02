package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"github.com/HoangP8/tokless/internal/util"
)

func TestHelpListsPonytailTool(t *testing.T) {
	help := helpText()
	if !strings.Contains(help, "ponytail") {
		t.Fatalf("help missing ponytail:\n%s", help)
	}
	if strings.Contains(help, "upgrade the 4 tools") {
		t.Fatalf("help still says 4 tools:\n%s", help)
	}
	if strings.Contains(help, "principles") {
		t.Fatalf("help still lists principles:\n%s", help)
	}
}

func TestDefaultRunChecksAndUpdatesToklessInTestMode(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	t.Setenv("TOKLESS_TEST", "1")
	t.Setenv("TOKLESS_TEST_LATEST", "0.1.0")
	util.SetHomeOverride(tmp)
	t.Cleanup(func() { util.SetHomeOverride("") })
	oldVersion := util.Version
	util.Version = "0.0.0"
	t.Cleanup(func() { util.Version = oldVersion })
	if err := os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() int {
		oldArgs := os.Args
		os.Args = []string{"tokless", "--tools", "rtk", "--agents", "claude"}
		defer func() { os.Args = oldArgs }()
		return run()
	})
	for _, want := range []string{"Tools", "RTK", "Agents", "Equipped Claude Code", "Tokless", "tokless v0.0.0 → v0.1.0 updated"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
	for _, nope := range []string{"global token-saver", "updating tokless", "update?", "Principles"} {
		if strings.Contains(out, nope) {
			t.Fatalf("output contains %q:\n%s", nope, out)
		}
	}
	if strings.Index(out, "Tokless") > strings.Index(out, "Tools") {
		t.Fatalf("self-update did not run before tool flow:\n%s", out)
	}
}

func TestInstallerRunDefersToklessUpdateUntilEnd(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	t.Setenv("TOKLESS_TEST", "1")
	t.Setenv("TOKLESS_TEST_LATEST", "0.1.0")
	t.Setenv("TOKLESS_INSTALLER_RUN", "1")
	util.SetHomeOverride(tmp)
	t.Cleanup(func() { util.SetHomeOverride("") })
	oldVersion := util.Version
	util.Version = "0.0.0"
	t.Cleanup(func() { util.Version = oldVersion })
	if err := os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() int {
		oldArgs := os.Args
		os.Args = []string{"tokless", "--tools", "rtk", "--agents", "claude"}
		defer func() { os.Args = oldArgs }()
		return run()
	})
	if strings.Index(out, "Equipped Claude Code") > strings.Index(out, "tokless v0.0.0 → v0.1.0 updated") {
		t.Fatalf("installer run moved self-update before agent flow:\n%s", out)
	}
}

func captureStdout(t *testing.T, fn func() int) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	code := fn()
	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	r.Close()
	if code != 0 {
		t.Fatalf("run exit %d:\n%s", code, out)
	}
	return string(out)
}
