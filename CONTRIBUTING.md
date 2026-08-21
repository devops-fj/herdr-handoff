# Contributing

Contributions are welcome. Keep changes focused on inspectable, local
agent-to-agent handoffs.

Before opening a pull request, run:

```bash
gofmt -w cmd internal
go test ./...
go vet ./...
```

Do not add automatic prompt delivery, external telemetry, or network uploads.
Any new context source must be optional, bounded in size, redacted, and visible
in the preview.

