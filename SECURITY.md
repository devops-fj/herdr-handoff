# Security policy

## Reporting a vulnerability

Please do not open a public issue for a vulnerability that could expose local
source code, credentials, terminal output, or agent prompts. Use GitHub's
private vulnerability reporting for this repository when available.

## Data handling

Herdr Handoff runs locally and does not make network requests. It reads only
the explicitly selected source agent and that agent's Git working directory.
The generated prompt is displayed before delivery.

Credential redaction is intentionally conservative but cannot recognize every
secret format. Users must treat the preview as the final security boundary.

