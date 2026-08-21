package herdr

import (
	"context"
	"strings"
	"testing"
)

type fakeCommander struct {
	output string
	name   string
	args   []string
}

func (f *fakeCommander) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.name = name
	f.args = append([]string(nil), args...)
	return []byte(f.output), nil
}

func TestAgentsParsesAndSortsFocusedFirst(t *testing.T) {
	fake := &fakeCommander{output: `{"result":{"agents":[{"agent":"codex","agent_status":"idle","pane_id":"w1:p2"},{"agent":"claude","agent_status":"working","focused":true,"pane_id":"w1:p1"}]}}`}
	client := &Client{Command: fake}
	agents, err := client.Agents()
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 2 || agents[0].PaneID != "w1:p1" {
		t.Fatalf("unexpected agents: %#v", agents)
	}
}

func TestPromptUsesArgvWithoutShell(t *testing.T) {
	fake := &fakeCommander{output: `{}`}
	client := &Client{Command: fake}
	prompt := "review $(touch /tmp/never) `whoami`"
	if err := client.Prompt("w1:p2", prompt); err != nil {
		t.Fatal(err)
	}
	if fake.name != "herdr" || strings.Join(fake.args, "|") != "agent|prompt|w1:p2|"+prompt {
		t.Fatalf("unexpected invocation: %q %#v", fake.name, fake.args)
	}
}

func TestExtractText(t *testing.T) {
	got := extractText([]byte(`{"result":{"output":"hello"}}`))
	if got != "hello" {
		t.Fatalf("got %q", got)
	}
}

func TestAgentLabelIncludesCompactProjectPath(t *testing.T) {
	agent := Agent{Name: "reviewer", Status: "idle", TabID: "w1:t2", PaneID: "w1:p3", CWD: "/code/opsPlatform/categraf"}
	got := agent.Label()
	if got != "reviewer  [idle]  w1:t2  w1:p3  opsPlatform/categraf" {
		t.Fatalf("got %q", got)
	}
}
