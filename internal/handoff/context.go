package handoff

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/devops-fj/herdr-handoff/internal/herdr"
)

const (
	defaultTranscriptLimit = 24_000
	defaultDiffLimit       = 36_000
)

type Options struct {
	IncludeTranscript bool
	IncludeGit        bool
	TranscriptLines   int
	TranscriptLimit   int
	DiffLimit         int
	Note              string
}

type Bundle struct {
	Source     herdr.Agent
	Target     herdr.Agent
	Note       string
	Transcript string
	Git        GitContext
	Warnings   []string
}

type GitContext struct {
	Root     string
	Branch   string
	Status   string
	DiffStat string
	Diff     string
}

type AgentReader interface {
	Read(target string, lines int) (string, error)
}

type Collector struct {
	Agents AgentReader
}

func (c Collector) Collect(source, target herdr.Agent, options Options) Bundle {
	bundle := Bundle{Source: source, Target: target, Note: strings.TrimSpace(options.Note)}
	if options.IncludeTranscript {
		lines := options.TranscriptLines
		if lines <= 0 {
			lines = 120
		}
		transcript, err := c.Agents.Read(source.Target(), lines)
		if err != nil {
			bundle.Warnings = append(bundle.Warnings, "Agent transcript unavailable: "+err.Error())
		} else {
			limit := options.TranscriptLimit
			if limit <= 0 {
				limit = defaultTranscriptLimit
			}
			bundle.Transcript = truncate(Redact(transcript), limit)
		}
	}
	if options.IncludeGit {
		cwd := source.Foreground
		if cwd == "" {
			cwd = source.CWD
		}
		gitContext, err := collectGit(cwd, options.DiffLimit)
		if err != nil {
			bundle.Warnings = append(bundle.Warnings, "Git context unavailable: "+err.Error())
		} else {
			bundle.Git = gitContext
		}
	}
	return bundle
}

func collectGit(cwd string, diffLimit int) (GitContext, error) {
	if strings.TrimSpace(cwd) == "" {
		return GitContext{}, errors.New("source agent has no working directory")
	}
	root, err := git(cwd, "rev-parse", "--show-toplevel")
	if err != nil {
		return GitContext{}, errors.New("working directory is not a Git repository")
	}
	root = strings.TrimSpace(root)
	branch, _ := git(root, "branch", "--show-current")
	if strings.TrimSpace(branch) == "" {
		branch, _ = git(root, "rev-parse", "--short", "HEAD")
	}
	status, _ := git(root, "status", "--short", "--untracked-files=normal")
	stat, _ := git(root, "diff", "--stat", "HEAD")
	diff, _ := git(root, "diff", "--no-ext-diff", "--no-color", "HEAD")
	if diffLimit <= 0 {
		diffLimit = defaultDiffLimit
	}
	return GitContext{
		Root:     filepath.Clean(root),
		Branch:   strings.TrimSpace(branch),
		Status:   truncate(Redact(strings.TrimSpace(status)), 8_000),
		DiffStat: truncate(strings.TrimSpace(stat), 8_000),
		Diff:     truncate(Redact(strings.TrimSpace(diff)), diffLimit),
	}, nil
}

func git(cwd string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	argv := append([]string{"-C", cwd}, args...)
	cmd := exec.CommandContext(ctx, "git", argv...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), message)
	}
	return stdout.String(), nil
}

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(password|passwd|secret|token|api[_-]?key|access[_-]?key|private[_-]?key)(\s*[=:]\s*)([^\s"']+)`),
	regexp.MustCompile(`(?i)(authorization\s*:\s*bearer\s+)([^\s]+)`),
	regexp.MustCompile(`\b(gh[pousr]_[A-Za-z0-9_]{20,})\b`),
	regexp.MustCompile(`\b(AKIA[0-9A-Z]{16})\b`),
}

func Redact(input string) string {
	output := input
	output = secretPatterns[0].ReplaceAllString(output, `${1}${2}[REDACTED]`)
	output = secretPatterns[1].ReplaceAllString(output, `${1}[REDACTED]`)
	for _, pattern := range secretPatterns[2:] {
		output = pattern.ReplaceAllString(output, `[REDACTED]`)
	}
	return output
}

func truncate(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	const marker = "\n\n... [truncated by herdr-handoff]"
	if limit <= len(marker) {
		return marker[:limit]
	}
	return value[:limit-len(marker)] + marker
}
