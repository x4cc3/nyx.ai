# Security Policy

## Supported versions

NYX is under active development.
Security fixes are applied to:

- the `master` branch
- the latest release tag, when one exists

## Reporting a vulnerability

Please **do not open a public issue** for exploitable vulnerabilities.

Use one of these paths instead:

1. GitHub private vulnerability reporting for `x4cc3/nyx.ai`, if it is enabled.
2. If private reporting is not available, contact the maintainer directly through GitHub before sharing full details.

Include as much of the following as you can:

- affected commit, branch, or release
- deployment mode (`local`, `docker`, Compose, frontend, API only, etc.)
- impact and attacker prerequisites
- reproduction steps or a minimal proof
- relevant logs, stack traces, or screenshots
- suggested fix or mitigation, if known

We aim to acknowledge reports within 5 business days.

## Disclosure expectations

- Give maintainers reasonable time to investigate and fix the issue.
- Avoid public disclosure until a fix or mitigation is available.
- Keep proofs minimal and safe. Do not include unnecessary destructive payloads.

## What belongs in public issues

The following are usually better handled as normal GitHub issues instead of vulnerability reports:

- documentation gaps
- general hardening ideas
- missing security headers or defaults without a demonstrated impact path
- questions about expected behavior or configuration
