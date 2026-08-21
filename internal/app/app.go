package app

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/devops-fj/herdr-handoff/internal/handoff"
	"github.com/devops-fj/herdr-handoff/internal/herdr"
)

type App struct {
	Client  *herdr.Client
	In      io.Reader
	Out     io.Writer
	Err     io.Writer
	Version string
}

func New() *App {
	return &App{
		Client:  herdr.NewClient(),
		In:      os.Stdin,
		Out:     os.Stdout,
		Err:     os.Stderr,
		Version: "dev",
	}
}

func (a *App) Run(args []string) error {
	if len(args) == 0 {
		return a.open()
	}
	switch args[0] {
	case "open":
		return a.open()
	case "agents":
		return a.agents(args[1:])
	case "preview":
		return a.transfer(args[1:], false)
	case "send":
		return a.transfer(args[1:], true)
	case "version", "--version", "-v":
		fmt.Fprintln(a.Out, a.Version)
		return nil
	case "help", "--help", "-h":
		a.usage()
		return nil
	default:
		return fmt.Errorf("unknown command %q; run herdr-handoff help", args[0])
	}
}

func (a *App) usage() {
	fmt.Fprintln(a.Out, `herdr-handoff — preview and send local context between Herdr agents

Usage:
  herdr-handoff open
  herdr-handoff agents [--json]
  herdr-handoff preview --from TARGET --to TARGET [options]
  herdr-handoff send --from TARGET --to TARGET --yes [options]

Options:
  --note TEXT          Add an instruction for the receiving agent
  --no-transcript      Do not include recent source-agent output
  --no-git             Do not include Git status and diff
  --lines N            Transcript lines to request (default 120)

TARGET is a unique Herdr agent name or pane ID. Sending always requires an
interactive SEND confirmation or the explicit --yes flag.`)
}

func (a *App) agents(args []string) error {
	set := flag.NewFlagSet("agents", flag.ContinueOnError)
	set.SetOutput(a.Err)
	jsonOutput := set.Bool("json", false, "print JSON")
	if err := set.Parse(args); err != nil {
		return err
	}
	agents, err := a.Client.Agents()
	if err != nil {
		return err
	}
	if *jsonOutput {
		encoder := json.NewEncoder(a.Out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(agents)
	}
	for _, agent := range agents {
		fmt.Fprintln(a.Out, agent.Label())
	}
	return nil
}

func (a *App) transfer(args []string, send bool) error {
	set := flag.NewFlagSet("handoff", flag.ContinueOnError)
	set.SetOutput(a.Err)
	from := set.String("from", "", "source agent name or pane ID")
	to := set.String("to", "", "target agent name or pane ID")
	note := set.String("note", "", "handoff note")
	noTranscript := set.Bool("no-transcript", false, "omit transcript")
	noGit := set.Bool("no-git", false, "omit Git context")
	lines := set.Int("lines", 120, "transcript lines")
	yes := set.Bool("yes", false, "confirm noninteractive send")
	if err := set.Parse(args); err != nil {
		return err
	}
	if *from == "" || *to == "" {
		return errors.New("--from and --to are required")
	}
	agents, err := a.Client.Agents()
	if err != nil {
		return err
	}
	source, err := findAgent(agents, *from)
	if err != nil {
		return fmt.Errorf("source: %w", err)
	}
	target, err := findAgent(agents, *to)
	if err != nil {
		return fmt.Errorf("target: %w", err)
	}
	if source.PaneID == target.PaneID {
		return errors.New("source and target must be different agents")
	}
	bundle := handoff.Collector{Agents: a.Client}.Collect(source, target, handoff.Options{
		IncludeTranscript: !*noTranscript,
		IncludeGit:        !*noGit,
		TranscriptLines:   *lines,
		Note:              *note,
	})
	prompt := handoff.BuildPrompt(bundle)
	if !send {
		fmt.Fprint(a.Out, prompt)
		return nil
	}
	if !*yes {
		return errors.New("refusing to send without --yes; use preview first")
	}
	if err := a.Client.Prompt(target.Target(), prompt); err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "Handoff sent to %s (%s).\n", displayName(target), target.PaneID)
	return nil
}

func (a *App) open() error {
	agents, err := a.Client.Agents()
	if err != nil {
		return err
	}
	if len(agents) < 2 {
		return errors.New("herdr-handoff needs at least two live Herdr agents")
	}
	reader := bufio.NewReader(a.In)
	fmt.Fprintln(a.Out, "Herdr Handoff")
	fmt.Fprintln(a.Out, "──────────────")
	fmt.Fprintln(a.Out, "Context stays local. Nothing is sent until you type SEND.")

	defaultSource := contextPane(os.Getenv("HERDR_PLUGIN_CONTEXT_JSON"))
	if defaultSource == "" {
		for _, agent := range agents {
			if agent.Focused {
				defaultSource = agent.PaneID
				break
			}
		}
	}
	source, err := chooseAgent(reader, a.Out, "Source agent", agents, defaultSource, "")
	if err != nil {
		return err
	}
	targetAgents := prioritizeTargets(agents, source)
	defaultTarget := ""
	for _, candidate := range targetAgents {
		if candidate.PaneID != source.PaneID {
			defaultTarget = candidate.PaneID
			break
		}
	}
	target, err := chooseAgent(reader, a.Out, "Target agent", targetAgents, defaultTarget, source.PaneID)
	if err != nil {
		return err
	}
	if target.Status == "working" {
		fmt.Fprintln(a.Out, "Note: the target is currently working; Herdr may queue this prompt behind its active turn.")
	}
	note, err := readLine(reader, a.Out, "Handoff note (optional)")
	if err != nil {
		return err
	}
	includeTranscript, err := confirm(reader, a.Out, "Include recent source-agent output?", true)
	if err != nil {
		return err
	}
	includeGit, err := confirm(reader, a.Out, "Include Git status and diff?", true)
	if err != nil {
		return err
	}

	fmt.Fprintln(a.Out, "\nCollecting local context...")
	bundle := handoff.Collector{Agents: a.Client}.Collect(source, target, handoff.Options{
		IncludeTranscript: includeTranscript,
		IncludeGit:        includeGit,
		TranscriptLines:   120,
		Note:              note,
	})
	prompt := handoff.BuildPrompt(bundle)
	fmt.Fprintln(a.Out, "\nPreview")
	fmt.Fprintln(a.Out, "───────")
	fmt.Fprintln(a.Out, prompt)
	fmt.Fprintf(a.Out, "Prompt size: %d bytes\n", len(prompt))
	fmt.Fprint(a.Out, "Type SEND to deliver this handoff, or press Enter to cancel: ")
	answer, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if strings.TrimSpace(answer) != "SEND" {
		fmt.Fprintln(a.Out, "Cancelled. Nothing was sent.")
		return nil
	}
	if err := a.Client.Prompt(target.Target(), prompt); err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "Handoff sent to %s (%s).\n", displayName(target), target.PaneID)
	return nil
}

func chooseAgent(reader *bufio.Reader, out io.Writer, title string, agents []herdr.Agent, defaultPane, excludePane string) (herdr.Agent, error) {
	var choices []herdr.Agent
	defaultIndex := -1
	fmt.Fprintf(out, "\n%s:\n", title)
	for _, agent := range agents {
		if agent.PaneID == excludePane {
			continue
		}
		choices = append(choices, agent)
		index := len(choices)
		marker := " "
		if agent.PaneID == defaultPane {
			marker = "*"
			defaultIndex = index - 1
		}
		fmt.Fprintf(out, "  %s %2d. %s\n", marker, index, agent.Label())
	}
	for {
		prompt := "Select a number"
		if defaultIndex >= 0 {
			prompt += fmt.Sprintf(" [%d]", defaultIndex+1)
		}
		line, err := readLine(reader, out, prompt)
		if err != nil {
			return herdr.Agent{}, err
		}
		if line == "" && defaultIndex >= 0 {
			return choices[defaultIndex], nil
		}
		selected, err := strconv.Atoi(line)
		if err == nil && selected >= 1 && selected <= len(choices) {
			return choices[selected-1], nil
		}
		fmt.Fprintln(out, "Enter one of the listed numbers.")
	}
}

func readLine(reader *bufio.Reader, out io.Writer, prompt string) (string, error) {
	fmt.Fprintf(out, "%s: ", prompt)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func confirm(reader *bufio.Reader, out io.Writer, prompt string, defaultYes bool) (bool, error) {
	suffix := "[y/N]"
	if defaultYes {
		suffix = "[Y/n]"
	}
	for {
		line, err := readLine(reader, out, prompt+" "+suffix)
		if err != nil {
			return false, err
		}
		switch strings.ToLower(line) {
		case "":
			return defaultYes, nil
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			fmt.Fprintln(out, "Enter y or n.")
		}
	}
}

func findAgent(agents []herdr.Agent, target string) (herdr.Agent, error) {
	for _, agent := range agents {
		if agent.PaneID == target {
			return agent, nil
		}
	}
	var matches []herdr.Agent
	for _, agent := range agents {
		if agent.Name == target {
			matches = append(matches, agent)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return herdr.Agent{}, fmt.Errorf("agent name %q is not unique; use a pane ID", target)
	}
	return herdr.Agent{}, fmt.Errorf("agent %q is not live", target)
}

func prioritizeTargets(agents []herdr.Agent, source herdr.Agent) []herdr.Agent {
	result := append([]herdr.Agent(nil), agents...)
	rank := func(agent herdr.Agent) int {
		if agent.PaneID == source.PaneID {
			return 100
		}
		if source.Workspace != "" && agent.Workspace == source.Workspace {
			return 0
		}
		if source.CWD != "" && agent.CWD == source.CWD {
			return 1
		}
		return 2
	}
	for i := 1; i < len(result); i++ {
		for j := i; j > 0 && rank(result[j]) < rank(result[j-1]); j-- {
			result[j], result[j-1] = result[j-1], result[j]
		}
	}
	return result
}

func contextPane(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	var value any
	if json.Unmarshal([]byte(raw), &value) != nil {
		return ""
	}
	return findString(value, "pane_id")
}

func findString(value any, key string) string {
	switch typed := value.(type) {
	case map[string]any:
		if direct, ok := typed[key].(string); ok {
			return direct
		}
		for _, child := range typed {
			if found := findString(child, key); found != "" {
				return found
			}
		}
	case []any:
		for _, child := range typed {
			if found := findString(child, key); found != "" {
				return found
			}
		}
	}
	return ""
}

func displayName(agent herdr.Agent) string {
	if agent.Name != "" {
		return agent.Name
	}
	if agent.Kind != "" {
		return agent.Kind
	}
	return "agent"
}
