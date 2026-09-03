## Hard Constraints (Never Violate)

- [Git] NEVER commit automatically. Commit ONLY on explicit user prompt "commit" using global OS identity and Conventional Commits 1.0.0.
- [IPC] Quickshell QML cannot read D-Bus directly. Spawn companion Go binary via `Quickshell.Io.Process` and read stdout.
- [Polkit] FORBIDDEN: custom Polkit rules for mount, unmount, or timer tasks. Use UDisks2/systemd standard interfaces. Restrict `polkit/org.omarchy.btrfs.raidmanager.policy` strictly to destructive actions (`add`, `remove`, `replace`).
- [Structure] FORBIDDEN: temporary shell scripts, cron jobs, or custom sockets. Adhere strictly to existing repo layout.

## Architecture & Concurrency

- [Events] MUST use D-Bus signals exclusively (`PropertiesChanged` via UDisks2/systemd). FORBIDDEN: sleep loops, polling, or `inotify` on `/proc/mounts`.
- [Mounts] Track mounts and unmounts solely via `org.freedesktop.UDisks2.Filesystem` signals.
- [Lifecycle] Companion Go daemon must monitor `os.Stdin` (`io.EOF` or `io.Copy(io.Discard, os.Stdin)`) and terminate immediately when Quickshell closes stdin.

## Code Standards

- [Go] Match repo Go version. Use `godbus/dbus`. Parse JSON output (`btrfs --format=json`); never parse text via regex. Avoid heavy external frameworks.
- [CLI] Daemon logic belongs in `cmd/raid-manager stream`; privileged commands belong in `admin <subcommand>`. Output errors as JSON to `stderr` for QML consumption.
- [Systemd] Implement maintenance jobs strictly as template units: `btrpool-scrub@.service` and `btrpool-balance@.service`.

## QML & Styling

- [SDK] Follow Omarchy SDK. Pipe `raid-manager stream` from `Quickshell.Io.Process` into `SplitParser` for NDJSON.
- [Theme] FORBIDDEN: hardcoded colors, radii, or border math. Source all visual values through `ThemeStyle.js` or `ThemeHelper` state functions (`hovered`, `active`, `urgent`).
- [Icons] Use vector `.svg` from `assets/`. Set `sourceSize: Qt.size(16, 16)`. Tint via `bar.foreground` or semantic `Omarchy.Theme` tokens.
- [Tooltips] FORBIDDEN: native Qt Quick `ToolTip`. In top bar use `bar.showTooltip(this, text)` / `bar.hideTooltip(this)`. In flyout use custom themed components.
- [Layout] Prevent clipping: calculate flyout height from `contentLayout.implicitHeight` + padding or use `ScrollView`. Keep collapsed bar clean: never mix manual anchors with `RowLayout`/`ColumnLayout`.

## Validation Commands

Run before every commit:

```bash
qmllint qml/**/*.qml
systemd-analyze verify systemd/*
go test -v -race ./...
```

## Documentation & Comments

- Applies only to docs and comments, never to source code.
- Write in concise, plain English (active voice, zero filler words, clear imperative instructions).
- Document new UI helper functions and parameters in `docs/architecture.md`.
