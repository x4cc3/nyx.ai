# Contributing to NYX

Thanks for contributing.

## Before you start

- Search existing issues and pull requests before starting new work.
- Open an issue first for large features, architectural changes, or behavior changes that may affect operators.
- Keep contributions focused. Small, reviewable pull requests are preferred.

## Development setup

For a local development environment, follow the quickstart in [README.md](README.md).

Common commands:

```bash
make fmt
go test ./...
```

Frontend commands:

```bash
cd web
npm install
npm run lint
npm test
```

## Contribution guidelines

### Code changes

- Prefer the smallest correct change.
- Do not mix unrelated refactors into feature or bug-fix pull requests.
- Update docs when behavior, commands, env vars, or API surfaces change.
- Add or update tests when changing behavior.
- Keep public APIs and config changes explicit in the pull request description.

### Security-related contributions

NYX is a security-oriented project. Please keep submissions safe and reviewable.

- Only test against systems you own or are explicitly authorized to assess.
- Do not include credentials, private targets, customer data, or sensitive artifacts in issues or pull requests.
- Keep proofs of concept minimal and non-destructive.
- Use [SECURITY.md](SECURITY.md) for vulnerability disclosure rather than public issues.

## Pull request checklist

Before opening a pull request, please:

- run formatting and relevant tests
- summarize what changed and why
- note any follow-up work or known limitations
- include screenshots for UI changes when helpful
- target the `master` branch unless maintainers ask otherwise

## Review expectations

Maintainers may ask for:

- narrower scope
- tests or docs updates
- safer defaults or stronger guardrails
- changes to naming or API clarity

That is normal and meant to keep the project maintainable.
