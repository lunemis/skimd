# Contributing to skimd

Thank you for your interest in contributing!

## Development

```bash
git clone https://github.com/lunemis/skimd.git
cd skimd
make build
make check   # runs test + test-race + vet
```

## Pull Requests

1. Fork the repo and create a branch from `main`
2. Write tests for new functionality
3. Run `make check` before submitting
4. Keep PRs focused — one feature or fix per PR

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Keep packages focused: `browser/` for filesystem, `render/` for markdown, `ui/` for TUI
- Write table-driven tests where possible

## Reporting Issues

Use the GitHub issue templates for bug reports and feature requests.
