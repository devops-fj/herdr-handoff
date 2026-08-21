# Herdr Handoff

Preview and send local working context from one [Herdr](https://herdr.dev)
coding agent to another.

Herdr Handoff is deliberately small: pick a source agent, pick a target agent,
review the generated handoff, and type `SEND`. It never uploads context to a
third-party service and never sends a prompt without explicit confirmation.

## What it includes

- Source and target agent identity, status, pane, and working directory
- Recent source-agent terminal output
- Git root, branch, working-tree status, diff stat, and current diff
- An optional handoff note from you
- Collection warnings when transcript or Git context is unavailable

Common token, password, API-key, bearer-token, GitHub-token, and AWS access-key
patterns are redacted. Transcript and diff sizes are capped before the preview
is generated. Redaction is a safety net, not a guarantee: always inspect the
preview before sending it.

## Requirements

- Herdr 0.7.5 or newer
- Git
- Go 1.23 or newer for installation from source

## Install

```bash
herdr plugin install devops-fj/herdr-handoff
```

Install a pinned release:

```bash
herdr plugin install devops-fj/herdr-handoff --ref v0.1.1
```

Open it from Herdr's plugin actions, or invoke the action directly:

```bash
herdr plugin action invoke devops-fj.herdr-handoff.open
```

The handoff opens in a local popup. Select the source and target agents, add an
optional note, review the exact prompt, and type `SEND` to deliver it.

## CLI

The installed binary also supports scripting and inspection:

```bash
# List live agents without reading their output
bin/herdr-handoff agents

# Generate a preview; nothing is sent
bin/herdr-handoff preview --from w1:p1 --to reviewer \
  --note "Review the implementation and run focused tests."

# Explicit noninteractive delivery
bin/herdr-handoff send --from w1:p1 --to reviewer --yes \
  --note "Continue the task from this context."
```

Use `--no-transcript` or `--no-git` to omit either source. Agent names are
accepted only when unique; pane IDs always identify a single live agent.

## Privacy and trust

All collection and prompt construction happens on the local machine. The
plugin invokes only the local `herdr` and `git` executables. It does not make
network requests.

The receiving coding agent may itself use remote model APIs according to that
agent's configuration. Sending a handoff therefore has the same privacy impact
as manually pasting the preview into that agent. Review diffs and transcripts
for credentials, private source, customer data, or unrelated terminal output.

## Local development

```bash
go test ./...
go build -o bin/herdr-handoff ./cmd/herdr-handoff
herdr plugin link "$PWD"
herdr plugin action invoke devops-fj.herdr-handoff.open
```

`plugin link` does not run manifest build commands, so build the binary before
linking. Remove the development link with:

```bash
herdr plugin unlink devops-fj.herdr-handoff
```

## Release process

1. Update `version` in `herdr-plugin.toml` and `CHANGELOG.md`.
2. Merge the release commit into `main`.
3. Tag the same semantic version, for example `v0.1.0`.
4. Push the tag. GitHub Actions runs tests and publishes checksums and
   cross-platform archives through GoReleaser.
5. Keep the repository public and add the `herdr-plugin` GitHub topic. Herdr's
   marketplace refreshes automatically.

## Scope

Version 0.1 focuses on a safe, inspectable one-to-one handoff. It does not
automatically create agents, summarize transcripts with another model, or run
multi-agent pipelines. Those features may be added only if they preserve the
preview-first workflow.

## License

[MIT](LICENSE)
