package app

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/devops-fj/herdr-handoff/internal/herdr"
)

func TestContextPane(t *testing.T) {
	raw := `{"invocation":{"pane":{"pane_id":"w2:p3"}}}`
	if got := contextPane(raw); got != "w2:p3" {
		t.Fatalf("got %q", got)
	}
}

func TestChooseAgentScopesToWorkspaceAndSearchesAll(t *testing.T) {
	agents := []herdr.Agent{
		{Name: "", Kind: "codex", Status: "idle", PaneID: "w1:p1", Workspace: "w1", CWD: "/repo/current"},
		{Name: "reviewer", Kind: "codex", Status: "idle", PaneID: "w2:p1", Workspace: "w2", CWD: "/repo/other"},
	}
	input := bufio.NewReader(strings.NewReader("/reviewer\n1\n"))
	var output bytes.Buffer
	got, err := chooseAgent(input, &output, "Target agent", agents, "", "", "w1")
	if err != nil {
		t.Fatal(err)
	}
	if got.PaneID != "w2:p1" {
		t.Fatalf("got %#v", got)
	}
	if !strings.Contains(output.String(), "1 agent(s) outside this view") || !strings.Contains(output.String(), `matches for "reviewer"`) {
		t.Fatalf("unexpected output: %s", output.String())
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

func TestPrioritizeTargetsPrefersNamedReadyAgentWithinScope(t *testing.T) {
	source := herdr.Agent{PaneID: "w1:p1", Workspace: "w1"}
	agents := []herdr.Agent{
		{Kind: "codex", Status: "working", PaneID: "w1:p2", Workspace: "w1"},
		{Name: "reviewer", Kind: "codex", Status: "idle", PaneID: "w1:p3", Workspace: "w1"},
		source,
	}
	got := prioritizeTargets(agents, source)
	if got[0].PaneID != "w1:p3" {
		t.Fatalf("unexpected order: %#v", got)
	}
}
