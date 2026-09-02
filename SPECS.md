# SPECS: omarchy-btrfs-raid-manager (Omarchy Top-Bar Plugin)

## 1. Vision & Core Principles

omarchy-btrfs-raid-manager is a lightweight, non-complex Omarchy top-bar applet to inspect, mount, and manage Btrfs RAID1 storage pools.

* **Simplicity First:** Avoid brittle custom text regex parsing and periodic polling loops that repeatedly spawn short-lived processes. Rely on a single resident Go daemon managed via Quickshell's `Process` model, streaming newline-delimited JSON (NDJSON) reactively by listening to D-Bus signals (`PropertiesChanged` on UDisks2 and systemd).
* **Lifecycle & Orphan Prevention:** The resident daemon monitors EOF on standard input (`os.Stdin`) and exits gracefully on `SIGPIPE` upon pipe closure by Quickshell, ensuring zero orphaned processes lingering on the bus.
* **Visual Integration:** Dynamic QML binding via `Omarchy.Theme` to reflect system colors, fonts, and dark/light modes instantly.
* **Safety by Design & Minimal Polkit Footprint:** Do not duplicate existing system authorization layers. Delegate mount/unmount authorization directly to UDisks2's native Polkit rules and unit lifecycles to systemd's native Polkit rules. Reserve the custom Polkit policy (`org.omarchy.btrfs.raidmanager.policy`) strictly for destructive device operations (`btrfs device add/remove/replace`).
* **Discrete Execution for Mutations:** Avoid long-lived privileged daemons or complex stateful UNIX domain sockets. Trigger privileged mutation actions strictly on-demand as discrete CLI subcommands (`raid-manager admin ...`), ensuring predictable lifecycle, isolation, and standard Polkit prompt boundaries.
* **Native Maintenance Delegation:** Do not reinvent custom maintenance loops, retries, or execution locks in userland. Delegate balance and scrub tasks directly to dedicated systemd services and timers (`systemd-run` or packaged parametric systemd units) controlled via standard systemd D-Bus interfaces.
* **Vectors Only:** All UI icons and disk glyphs use resolution-independent `.svg` assets.

---

## 2. Tech Stack & Official Sources

* **Omarchy Plugin SDK & Quickshell Runtime:** https://plugins.omarchy.org/develop.html
  * Tool: `omarchy-cli` (used for `omarchy plugin init`, `omarchy plugin validate`, and live dev mode via `omarchy plugin dev`).
  * Runtime: Quickshell process execution model (`Quickshell.Io.Process` consuming stdout streams via `SplitParser` or line collectors).
* **Go (Golang):** https://go.dev/doc/
  * Used for the resident event-driven monitoring daemon (`godbus/dbus`), kernel `sysfs` reading, orchestrating structured JSON interfaces, and discrete privileged execution helpers.
* **Btrfs Documentation & Utilities:** https://btrfs.readthedocs.io/ and https://btrfs.wiki.kernel.org/
  * Path references: `/sys/fs/btrfs/<uuid>/`.
  * Structured tooling: `btrfs-progs` (leveraging standard `btrfs --format=json` commands for scrub, balance, and filesystem statistics), `smartmontools`.
* **UDisks2 API Documentation:** https://storaged.org/doc/udisks2-api/latest/
  * D-Bus Interface: `org.freedesktop.UDisks2` (Block, Filesystem, and Job interfaces for reactive signal listening, mount status, disk topology, and authenticated actions).
* **Systemd D-Bus API Documentation:** https://www.freedesktop.org/wiki/Software/systemd/dbus/
  * D-Bus Interface: `org.freedesktop.systemd1.Manager` (Unit lifecycle, timer control, active job signal monitoring, and tracking transient maintenance jobs).
* **Polkit Documentation:** https://www.freedesktop.org/software/polkit/docs/latest/
  * Action definitions scoped strictly to destructive operations via `/usr/share/polkit-1/actions/org.omarchy.btrfs.raidmanager.policy`.
* **GitHub Open-Source Contribution Standards:** https://docs.github.com/en/contributing
  * Standard templates, issue workflows, and clear contribution pathways.

---

## 3. Repository Layout

```text
omarchy-btrfs-raid-manager/
├── .github/
│   ├── ISSUE_TEMPLATE/
│   │   ├── bug_report.md
│   │   └── feature_request.md
│   ├── workflows/
│   │   ├── build.yml             # Compiles Go for amd64 & arm64, runs golangci-lint, qmllint, omarchy validate
│   │   └── test.yml              # Runs `go test` with mocked sysfs fixtures, btrfs JSON payloads, and D-Bus interfaces
│   └── PULL_REQUEST_TEMPLATE.md
├── assets/
│   ├── icon-pool-healthy.svg
│   ├── icon-pool-degraded.svg
│   ├── icon-pool-working.svg
│   ├── icon-disk.svg
│   ├── icon-mount.svg
│   └── icon-unmount.svg
├── cmd/
│   └── raid-manager/
│       └── main.go               # Single Go binary handling resident 'stream' and discrete 'admin' subcommands
├── internal/
│   ├── sysfs/                    # Direct /sys/fs/btrfs/<uuid> reader
│   ├── btrfs/                    # Structured btrfs --format=json executor and parser
│   ├── udisks/                   # UDisks2 D-Bus client, mount tracker, and PropertiesChanged subscriber
│   ├── systemd/                  # systemd1 Manager D-Bus client for scrub/balance units & timers
│   ├── parser/                   # Native structured data aggregation and NDJSON serialization
│   └── admin/                    # Privileged mutation logic guarded by Polkit actions
├── docs/
│   ├── architecture.md           # UI-to-D-Bus/sysfs/systemd dataflow and NDJSON streaming explained
│   ├── btrfs-guide.md            # RAID1 concepts, JSON commands & systemd units cheatsheet
│   └── references.md             # Links and notes for Omarchy SDK, Btrfs sysfs, UDisks2, Polkit, Go
├── polkit/
│   └── org.omarchy.btrfs.raidmanager.policy # Scoped Polkit rules (device add/remove/replace only)
├── qml/
│   ├── Components/
│   │   ├── ActionButton.qml      # Button wrapper with native ToolTip support
│   │   ├── DiskRow.qml           # SVG disk glyph + device info + SMART badge
│   │   └── PoolHeader.qml        # Mount status, mount/unmount trigger, usage bar
│   ├── Applet.qml                # Top-bar icon and status indicator (consumes NDJSON state)
│   └── Flyout.qml                # Interactive dropdown panel
├── systemd/
│   ├── btrpool-scrub@.service     # Parametric maintenance service executing btrfs scrub
│   ├── btrpool-scrub@.timer       # Periodic scrub execution trigger
│   ├── btrpool-balance@.service   # Parametric maintenance service executing filtered btrfs balance
│   └── btrpool-balance@.timer     # Periodic balance execution trigger
├── CONTRIBUTING.md
├── CODE_OF_CONDUCT.md
├── LICENSE                       # GPL-3.0
├── README.md
├── go.mod
├── go.sum
└── manifest.json

```

---

## 4. UI Architecture (Top-Bar & Flyout)

### A. Top-Bar Applet (`Applet.qml`)

* **Process Management:** Launches `raid-manager stream` as a single persistent background `Quickshell.Io.Process` on startup. Listens to stdout line-by-line via `SplitParser` (NDJSON), updating global reactive QML properties instantly without periodic polling overhead.
* **Visuals:** SVG icon dynamically adapting to pool health:
* *Normal:* Themed accent color.
* *Active Operation (Scrub/Balance):* Animated pulse or secondary accent matching active systemd service states.
* *Degraded / Missing Disk / SMART Error:* Warning or destructive color.


* **Badge:** Displays either real free space percentage or current task progress (`Scrub: 42%`) parsed continuously from systemd and btrfs runtime status.

### B. Interactive Flyout (`Flyout.qml`)

1. **Header:**
* Mountpoint and UUID.
* Real capacity gauge (`Free estimated` vs. `Used`).
* Quick **Mount / Unmount** button triggering discrete commands delegated to UDisks2 via native Polkit authorization.


2. **Topology List:**
* Repeated `DiskRow.qml` instances using SVG disk icons.
* Shows device node (`/dev/sda1`), model name, and SMART health indicator (queried via UDisks2 ATA drive properties).
* "Add Disk" action with partition selector (dispatches discrete privileged `raid-manager admin add` invocation).


3. **Maintenance Bar:**
* Buttons for **Scrub** and **Balance**, each equipped with descriptive mouse-hover tooltips explaining the operation. Buttons trigger or cancel the dedicated parametric systemd maintenance services via discrete systemd D-Bus calls.
* Active status indicators showing runtime execution progress (querying `btrfs scrub status --format=json`).
* Toggle switch for background automated maintenance (enables/disables systemd timers directly using systemd D-Bus interface without custom policies).



---

## 5. Binary Specifications (`cmd/` & `internal/`)

### `raid-manager stream` (Resident Event-Driven Monitor)

* Operates as a continuous background daemon communicating via newline-delimited JSON (NDJSON) to stdout:
* **Lifecycle & Exit Handlers:** Continuously monitors `os.Stdin` for `io.EOF` and traps standard exit/pipe signals to terminate cleanly when the Quickshell parent process terminates.
* **Event-Driven Reactivity:** Subscribes to `org.freedesktop.DBus.PropertiesChanged` signals from:
* `org.freedesktop.UDisks2`: For block device attachments, SMART changes, and filesystem mount/unmount operations via `org.freedesktop.UDisks2.Filesystem`.
* `org.freedesktop.systemd1`: For scrub/balance service state and timer triggers.


* **Granular Telemetry:** Reads allocation data directly from `/sys/fs/btrfs/<uuid>/allocation/` and calls `btrfs --format=json` only when state change events require updating operational scrub/balance progress.
* Flushes a unified JSON state object on startup and subsequent detected state change events, providing zero-polling reactivity to the QML frontend.

### `raid-manager admin` (Discrete Mutation Subcommands)

* Operates strictly as short-lived, discrete executions invoked only on explicit user actions, terminating immediately after completion:
* **Native Authorization Delegation (No Custom Policy):**
* `mount <device>` / `unmount <device>`: Delegated to `org.freedesktop.UDisks2.Filesystem.Mount` / `Unmount` using native UDisks2 Polkit permissions.
* `scrub <start|cancel> <mountpoint>`: Starts or stops `btrpool-scrub@<escaped_path>.service` via `org.freedesktop.systemd1.Manager` using native systemd Polkit permissions.
* `balance <start|cancel> <mountpoint>`: Starts or stops `btrpool-balance@<escaped_path>.service` via `org.freedesktop.systemd1.Manager` using native systemd Polkit permissions.
* `timer <enable|disable>`: Enables or disables maintenance `.timer` units via `org.freedesktop.systemd1.Manager`.


* **Guarded Custom Operations (`org.omarchy.btrfs.raidmanager.policy`):**
* `add <device> <mountpoint>`: Dispatches `btrfs device add` under explicit Polkit authorization.
* `remove <device> <mountpoint>`: Dispatches `btrfs device remove` under explicit Polkit authorization.
* `replace <old_dev> <new_dev> <mountpoint>`: Dispatches `btrfs device replace` under explicit Polkit authorization.


* Returns standardized exit codes and JSON-formatted error messages to stderr upon completion.

---

## 6. GitHub Actions & CI Verification

### `build.yml`

* Runs `golangci-lint` across the Go codebase.
* Runs `qmllint` across all `.qml` files.
* Validates plugin manifest structure using `omarchy-cli` or schema validation.
* Validates Polkit XML syntax in `polkit/org.omarchy.btrfs.raidmanager.policy`.
* Validates systemd unit files syntax using `systemd-analyze verify`.
* Includes cross-compilation matrix targeting `GOOS=linux` with `GOARCH=amd64` and `GOARCH=arm64`.

### `test.yml`

* Uses `go test ./...` to validate parser, event listener, and data aggregation logic.
* Ingests mock sysfs trees (stored in `tests/fixtures/sysfs/`), pre-recorded `btrfs --format=json` outputs, and mocked UDisks2 / systemd D-Bus signals.
* Asserts the resulting parsed structs and NDJSON stream emissions match expected properties (accurate `free_bytes`, `raid_profile`, `is_degraded`, active operation progress, and disk count).

---

## 7. AI Agent Execution Directives

1. **Bootstrap & Manifest:** Clone or initialize the official Omarchy plugin structure into the current workspace, ensuring `manifest.json` reflects `omarchy-btrfs-raid-manager`.
2. **Community Templates:** Create clean, concise `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, and issue/PR templates matching GitHub standards.
3. **Core Binary & Polkit Policy:** Write safe, strictly-typed Go code for `raid-manager` supporting the persistent `stream` daemon (with `os.Stdin` EOF exit loops) and discrete `admin` subcommands. Install `polkit/org.omarchy.btrfs.raidmanager.policy` defining granular action permissions solely for destructive device actions (`add`/`remove`/`replace`).
4. **Systemd Maintenance Units:** Provide properly formatted parametric systemd service and timer unit templates in `systemd/` for decoupled scrub and balance scheduling.
5. **SVG Visuals:** Implement clean SVG assets in `assets/`.
6. **QML Layout & NDJSON Integration:** Implement `Applet.qml` and `Flyout.qml` using `Quickshell.Io.Process` to consume NDJSON from `raid-manager stream`, binding reactive state to `Omarchy.Theme` components and tooltips.
7. **Documentation Creation:** Populate `docs/architecture.md`, `docs/btrfs-guide.md`, and `docs/references.md` with explicit links to official documentation (including UDisks2, systemd D-Bus, Quickshell Process, and Btrfs documentation).
8. **CI Setup:** Provide the GitHub Actions workflow files for Go cross-compilation (amd64/arm64), linting (including Polkit policy validation and systemd verification), and testing.
