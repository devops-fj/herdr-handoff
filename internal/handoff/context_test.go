package handoff

import (
	"strings"
	"testing"

	"github.com/devops-fj/herdr-handoff/internal/herdr"
)

func TestRedact(t *testing.T) {
	input := "TOKEN=abc123 password: hunter2 Authorization: Bearer topsecret ghp_abcdefghijklmnopqrstuvwxyz AKIA1234567890ABCDEF"
	got := Redact(input)
	for _, secret := range []string{"abc123", "hunter2", "topsecret", "ghp_", "AKIA"} {
		if strings.Contains(got, secret) {
			t.Fatalf("secret %q remains in %q", secret, got)
		}
	}
}

func TestBuildPrompt(t *testing.T) {
	bundle := Bundle{
		Source: herdr.Agent{Kind: "codex", PaneID: "w1:p1", Status: "done", CWD: "/tmp/project"},
		Target: herdr.Agent{Name: "reviewer", Kind: "claude", PaneID: "w1:p2"},
		Note:   "Review the implementation.",
		Git:    GitContext{Root: "/tmp/project", Branch: "feature", Status: " M main.go"},
	}
	prompt := BuildPrompt(bundle)
	for _, want := range []string{"# Agent handoff", "Review the implementation.", "w1:p1", "feature", "Working tree status"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestTruncate(t *testing.T) {
	got := truncate(strings.Repeat("x", 100), 50)
	if len(got) != 50 || !strings.Contains(got, "truncated") {
		t.Fatalf("unexpected truncation: %q (%d)", got, len(got))
	}
}

func TestBuildPromptUsesFenceLongerThanContent(t *testing.T) {
	bundle := Bundle{
		Source:     herdr.Agent{PaneID: "w1:p1"},
		Target:     herdr.Agent{PaneID: "w1:p2"},
		Transcript: "message with ```nested fence``` inside",
	}
	prompt := BuildPrompt(bundle)
	if !strings.Contains(prompt, "````text") {
		t.Fatalf("expected a four-backtick fence:\n%s", prompt)
	}
}
