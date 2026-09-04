# Contributing to omarchy-btrfs-raid-manager

Thank you for your interest in contributing to **omarchy-btrfs-raid-manager**! This document provides guidelines and instructions for contributing.

## Code of Conduct

All contributors are expected to follow our [Code of Conduct](CODE_OF_CONDUCT.md). Please read it before participating in discussions or submitting contributions.

## Architectural Principles

1. **Simplicity First**: Rely on event-driven monitoring via D-Bus and sysfs instead of periodic polling loops.
2. **Lifecycle Safety**: The resident daemon must always monitor EOF on `os.Stdin` and handle `SIGPIPE` / exit signals cleanly to prevent orphan processes.
3. **Minimal Polkit Footprint**: Use native UDisks2 and systemd authorization for standard operations; reserve custom Polkit policy strictly for destructive disk operations (`add`, `remove`, `replace`).
4. **Decoupled Systemd Maintenance**: Balance and scrub tasks are executed via parametric systemd units (`btrpool-scrub@.service`, `btrpool-balance@.service`) and timers.
5. **Vectors Only**: All visual disk glyphs and icons are SVGs.

## Development Setup

### Prerequisites

- Go (1.24+)
- Omarchy Desktop Environment with Quickshell
- `btrfs-progs` (v6.0+)
- `udisks2` and `systemd`

### Building the Project

Compile the Go daemon:

```bash
go build -o bin/raid-manager ./cmd/raid-manager
```

Run tests:

```bash
go test -v ./...
```

Validate the plugin manifest:

```bash
omarchy plugin validate .
```

Verify systemd units:

```bash
systemd-analyze verify systemd/btrpool-scrub@.service systemd/btrpool-balance@.service
```

## Submitting Pull Requests

1. Fork the repository and create a feature branch (`git checkout -b feature/my-enhancement`).
2. Follow Go conventions: format code with `gofmt` / `goimports` and verify with `golangci-lint`.
3. Add unit tests for new logic (use mock sysfs fixtures and JSON payloads in `tests/fixtures/`).
4. Ensure all CI checks pass (`build.yml` and `test.yml`).
5. Open a PR using the provided Pull Request template.
