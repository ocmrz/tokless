package commands

import (
	"testing"

	"github.com/HoangP8/tokless/internal/core"
)

func TestToolNeedsNode(t *testing.T) {
	cases := []struct {
		name string
		tool core.ToolManifest
		want bool
	}{
		{name: "explicit", tool: core.ToolManifest{NeedsNode: true}, want: true},
		{name: "npm channel", tool: core.ToolManifest{Channel: core.ChannelNpm}, want: true},
		{name: "min node", tool: core.ToolManifest{MinNodeMajor: 20}, want: true},
		{name: "github only", tool: core.ToolManifest{Channel: core.ChannelGitHub}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := toolNeedsNode(&tc.tool); got != tc.want {
				t.Fatalf("toolNeedsNode() = %v, want %v", got, tc.want)
			}
		})
	}
}
