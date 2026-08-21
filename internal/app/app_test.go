package app

import (
	"testing"

	"github.com/devops-fj/herdr-handoff/internal/herdr"
)

func TestContextPane(t *testing.T) {
	raw := `{"invocation":{"pane":{"pane_id":"w2:p3"}}}`
	if got := contextPane(raw); got != "w2:p3" {
		t.Fatalf("got %q", got)
	}
}

func TestPrioritizeTargetsKeepsSameWorkspaceFirst(t *testing.T) {
	source := herdr.Agent{PaneID: "w2:p1", Workspace: "w2", CWD: "/repo"}
	agents := []herdr.Agent{
		{PaneID: "w1:p1", Workspace: "w1"},
		{PaneID: "w3:p1", Workspace: "w3", CWD: "/repo"},
		{PaneID: "w2:p2", Workspace: "w2"},
		source,
	}
	got := prioritizeTargets(agents, source)
	if got[0].PaneID != "w2:p2" || got[1].PaneID != "w3:p1" {
		t.Fatalf("unexpected order: %#v", got)
	}
}
