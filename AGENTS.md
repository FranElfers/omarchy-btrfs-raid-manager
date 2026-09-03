## Core Architectural Constraints

- **Host Integration Boundary:** Quickshell does not provide generic D-Bus introspection in QML. All system communication occurs strictly via `Quickshell.Io.Process` consuming standard output from the Go companion binary.
- **Zero Custom Polling:** Never implement sleep/polling loops in QML or Go. The Go background daemon (`raid-manager stream`) must rely solely on D-Bus signals (`PropertiesChanged` from UDisks2 and systemd).
- **No `inotify` on `/proc/mounts`:** Mount and unmount transitions are tracked exclusively via `org.freedesktop.UDisks2.Filesystem` D-Bus signals.
- **Lifecycle & Orphan Prevention:** The resident Go daemon must block on `io.Copy(io.Discard, os.Stdin)` or monitor `os.Stdin` for `io.EOF`, exiting immediately if Quickshell terminates or closes the execution pipe.
- **Minimal Polkit Surface:** Do not write custom Polkit rules for `mount`, `unmount`, or timer operations. Delegate them to native UDisks2 and systemd interfaces. The file `polkit/org.omarchy.btrfs.raidmanager.policy` is reserved solely for destructive mutations (`add`, `remove`, `replace`).

---

## Technical Stack & Standards

- **Go Modules (`cmd/`, `internal/`):**
    - Target Go version matching project configuration.
    - Use `godbus/dbus` for D-Bus IPC.
    - Parse structured JSON from system tools using `btrfs --format=json` (no plain text regex parsing).
    - Keep dependencies minimal; avoid heavy frameworks.
- **QML & UI (`qml/`):**
    - Adhere strictly to Omarchy SDK standards.
    - Consume `raid-manager stream` via `Quickshell.Io.Process` piped into `SplitParser` for NDJSON handling.
    - Visual styling must dynamically bind to `Omarchy.Theme` tokens.
    - All icon references must use vector `.svg` assets in `assets/`.
- **Systemd Maintenance (`systemd/`):**
    - Balance and scrub routines must run as parametric templates: `btrpool-scrub@.service` and `btrpool-balance@.service`.

---

## Workflow & Contribution Commands

Before committing changes, agents must ensure all validations pass:

```bash
# Linting & verification
golangci-lint run ./...
qmllint qml/**/*.qml
systemd-analyze verify systemd/*

# Testing
go test -v -race ./...
```

---

## Code Generation Rules

1. **Discrete vs. Stream Logic:** Ensure background monitoring stays inside `cmd/raid-manager` under the `stream` verb. Privileged actions belong under `admin <subcommand>`.

2. **Error Emission:** Output machine-readable JSON errors to `stderr` from CLI invocations to allow clear reporting in the QML Flyout.

3. **No Phantom Files:** Do not create temporary shell wrappers, ad-hoc state sockets, or standalone cron scripts. Rely solely on the defined repository structure.

---

## UI, Theming & Layout Directives (QML)

- **Zero Native Qt Tooltips:** Never use native Qt Quick `ToolTip.text`, `ToolTip.visible`, or unstyled `ToolTip { ... }` blocks.
    - In the top bar: Tooltips are strictly delegated to the Omarchy host shell via `bar.showTooltip(this, text)` on `containsMouse: true` and `bar.hideTooltip(this)` on exit.
    - Inside the Flyout: All action buttons and toggle controls must use custom, theme-styled tooltip overlays matching `Omarchy.Theme` tokens.
- **Top-Bar Icon Theming & Sizing:** Never render icons in raw or hardcoded colors.
    - Tint all SVG glyphs dynamically using `bar.foreground` or semantic `Omarchy.Theme` tokens (accent, active pulse, urgent/error).
    - Use `sourceSize: Qt.size(16, 16)` to guarantee crisp vector rasterization on high-DPI displays.
- **Flyout Sizing & Bounding:** Never hardcode fixed window `height` in `Flyout.qml`. Root height must strictly track `contentLayout.implicitHeight` plus outer padding (or wrap inside a `Flickable`/`ScrollView`) so bottom sections like "Automated Maintenance" are never clipped.
- **Layout Isolation:** Keep the collapsed bar widget simple. Never mix manual anchors (`anchors.centerIn`, `anchors.left`) with `RowLayout`/`ColumnLayout`, and never render multiple competing text nodes in the bar.

---

## Functional Styling & Design System Consistency

- **Single Source of Truth for Styling:** Never define ad-hoc `radius`, inline border math, or hardcoded hex/rgba values inside individual QML components. All visual attributes must be derived from the centralized functional helper (`ThemeStyle.js` / `ThemeHelper`).
- **Functional Token Composition:** Use pure functions that accept current interaction state flags (`hovered`, `active`, `urgent`) and return standardized theme values (e.g., background tints, border highlights, and standard corner radii).
- **Strict Shape Consistency:** All interactive surfaces (buttons, toggles, badges, flyout cards, and tooltips) must conform strictly to the design system's radius scale. Disconnected square corners alongside rounded elements are strictly prohibited.
- **Documentation Contract:** Any new UI helper or factory function added must be documented with parameter contracts in `docs/architecture.md` so subsequent agent iterations reuse them by default.
