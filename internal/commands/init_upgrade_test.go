package commands_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HoangP8/tokless/internal/commands"
	"github.com/HoangP8/tokless/internal/core"
	"github.com/HoangP8/tokless/internal/util"
)

func TestInstallerRunPropagatesUpgradeToInstallAndWire(t *testing.T) {
	t.Setenv("TOKLESS_TEST", "1")
	t.Setenv("TOKLESS_INSTALLER_RUN", "1")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(os.Getenv("HOME"), ".config"))
	util.SetHomeOverride(os.Getenv("HOME"))
	defer util.SetHomeOverride("")

	toolID := "tokless-test-upgrade-tool"
	agentID := "tokless-test-upgrade-agent"

	installSeen := false
	wireSeen := false

	core.RegisterAgent(&core.AgentManifest{
		ID:       agentID,
		Label:    "Upgrade Probe Agent",
		Homepage: "https://example.com",
		Detect: func() core.Detection {
			return core.Detection{Installed: true, Source: "test"}
		},
	})
	core.RegisterTool(&core.ToolManifest{
		ID:           toolID,
		Label:        "Upgrade Probe Tool",
		Description:  "test",
		Homepage:     "https://example.com",
		InstallHint:  "test",
		NotTrackable: true,
		Install: func(opts core.RunOpts) (bool, error) {
			installSeen = opts.Upgrade
			return true, nil
		},
		WireFor: map[string]core.AgentFn{
			agentID: func(opts core.RunOpts) (bool, error) {
				wireSeen = opts.Upgrade
				return true, nil
			},
		},
		VerifyFor: map[string]core.VerifyFn{
			agentID: func() *bool { return core.BoolPtr(true) },
		},
	})

	if code := commands.RunInit(commands.InitOptions{Tools: []string{toolID}, Agents: []string{agentID}}); code != 0 {
		t.Fatalf("RunInit returned %d", code)
	}
	if !installSeen {
		t.Fatal("installer env did not propagate Upgrade=true into tool install")
	}
	if !wireSeen {
		t.Fatal("installer env did not propagate Upgrade=true into tool wire")
	}
}
