package manifest

import (
	"os"
	"strings"
	"testing"
)

func TestPaneExecutableIsRelativeToPluginRoot(t *testing.T) {
	data, err := os.ReadFile("../../herdr-plugin.toml")
	if err != nil {
		t.Fatalf("read plugin manifest: %v", err)
	}

	manifest := string(data)
	if !strings.Contains(manifest, `command = ["./bin/herdr-handoff", "open"]`) {
		t.Fatal("pane command must resolve herdr-handoff relative to the plugin root")
	}
	if strings.Contains(manifest, `command = ["bin/herdr-handoff", "open"]`) {
		t.Fatal("pane command without ./ is resolved through PATH by Herdr")
	}
}
