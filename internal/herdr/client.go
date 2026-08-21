package herdr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
)

type Agent struct {
	Kind       string `json:"agent"`
	Status     string `json:"agent_status"`
	CWD        string `json:"cwd"`
	Foreground string `json:"foreground_cwd"`
	Focused    bool   `json:"focused"`
	Name       string `json:"name"`
	PaneID     string `json:"pane_id"`
	TabID      string `json:"tab_id"`
	Workspace  string `json:"workspace_id"`
}

func (a Agent) Target() string { return a.PaneID }

func (a Agent) Label() string {
	name := a.Name
	if name == "" {
		name = a.Kind
	}
	location := a.TabID
	if location == "" {
		location = a.Workspace
	}
	label := fmt.Sprintf("%s  [%s]  %s  %s", name, a.Status, location, a.PaneID)
	if project := a.ProjectLabel(); project != "" {
		label += "  " + project
	}
	return label
}

// ProjectLabel keeps agent lists useful without filling a popup with full paths.
func (a Agent) ProjectLabel() string {
	path := a.Foreground
	if path == "" {
		path = a.CWD
	}
	path = strings.Trim(strings.ReplaceAll(path, `\`, "/"), "/")
	if path == "" {
		return ""
	}
	parts := strings.Split(path, "/")
	if len(parts) > 2 {
		parts = parts[len(parts)-2:]
	}
	return strings.Join(parts, "/")
}

type Commander interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type ExecCommander struct{}

func (ExecCommander) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = strings.TrimSpace(stdout.String())
		}
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("%s %s: %s", name, strings.Join(args, " "), message)
	}
	return stdout.Bytes(), nil
}

type Client struct {
	Command Commander
	Timeout time.Duration
}

func NewClient() *Client {
	return &Client{Command: ExecCommander{}, Timeout: 15 * time.Second}
}

func (c *Client) run(args ...string) ([]byte, error) {
	if c.Command == nil {
		return nil, errors.New("herdr command runner is nil")
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.Command.Run(ctx, "herdr", args...)
}

func (c *Client) Agents() ([]Agent, error) {
	raw, err := c.run("agent", "list")
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Result struct {
			Agents []Agent `json:"agents"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decode herdr agent list: %w", err)
	}
	agents := envelope.Result.Agents
	sort.SliceStable(agents, func(i, j int) bool {
		if agents[i].Focused != agents[j].Focused {
			return agents[i].Focused
		}
		if statusRank(agents[i].Status) != statusRank(agents[j].Status) {
			return statusRank(agents[i].Status) < statusRank(agents[j].Status)
		}
		return agents[i].PaneID < agents[j].PaneID
	})
	return agents, nil
}

func statusRank(status string) int {
	switch status {
	case "blocked":
		return 0
	case "working":
		return 1
	case "done":
		return 2
	case "idle":
		return 3
	default:
		return 4
	}
}

func (c *Client) Read(target string, lines int) (string, error) {
	if lines <= 0 {
		lines = 120
	}
	raw, err := c.run("agent", "read", target, "--source", "recent-unwrapped", "--lines", fmt.Sprint(lines), "--format", "text")
	if err != nil {
		return "", err
	}
	return extractText(raw), nil
}

func (c *Client) Prompt(target, prompt string) error {
	_, err := c.run("agent", "prompt", target, prompt)
	return err
}

func extractText(raw []byte) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || !json.Valid(raw) {
		return trimmed
	}
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return trimmed
	}
	if text := findText(value); text != "" {
		return text
	}
	return trimmed
}

func findText(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		for _, key := range []string{"text", "output", "content", "snapshot"} {
			if candidate, ok := typed[key].(string); ok && candidate != "" {
				return candidate
			}
		}
		for _, key := range []string{"result", "data"} {
			if nested, ok := typed[key]; ok {
				if text := findText(nested); text != "" {
					return text
				}
			}
		}
	}
	return ""
}
