# Changelog

All notable changes to this project will be documented in this file. The
format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and the project follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.2.0] - 2026-08-21

### Added

- Workspace-scoped agent pickers that keep large Herdr sessions manageable.
- Cross-workspace search by agent name, kind, status, pane, workspace, or path.
- Compact project paths in interactive and CLI agent lists.

### Changed

- Prefer explicitly named, ready target agents within the same location scope.
- Keep all-agent selection available through the interactive `a` command.

## [0.1.1] - 2026-08-21

### Fixed

- Resolve the popup executable relative to the plugin root so Herdr 0.7.5 can
  launch the handoff pane after a local link or marketplace installation.

## [0.1.0] - 2026-08-21

### Added

- Interactive source and target agent selection using unique Herdr pane IDs.
- Preview-first handoff generation with an explicit `SEND` confirmation.
- Optional recent transcript and Git working-tree context collection.
- Local redaction for common credential patterns and bounded context sizes.
- Scriptable agent listing, preview, and explicitly confirmed send commands.
- Herdr plugin action and popup entrypoints.
- Cross-platform CI and GitHub Release automation.

[Unreleased]: https://github.com/devops-fj/herdr-handoff/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/devops-fj/herdr-handoff/releases/tag/v0.2.0
[0.1.1]: https://github.com/devops-fj/herdr-handoff/releases/tag/v0.1.1
[0.1.0]: https://github.com/devops-fj/herdr-handoff/releases/tag/v0.1.0
