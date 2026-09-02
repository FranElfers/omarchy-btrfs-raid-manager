# omarchy-btrfs-raid-manager

[![Build Status](https://github.com/franelfers/omarchy-btrfs-raid-manager/actions/workflows/build.yml/badge.svg)](https://github.com/franelfers/omarchy-btrfs-raid-manager/actions/workflows/build.yml)
[![Test Suite](https://github.com/franelfers/omarchy-btrfs-raid-manager/actions/workflows/test.yml/badge.svg)](https://github.com/franelfers/omarchy-btrfs-raid-manager/actions/workflows/test.yml)
[![License: GPL-3.0](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)

A lightweight, non-complex Omarchy top-bar applet to inspect, mount, and manage **Btrfs RAID1** storage pools with zero periodic polling overhead.

---

## Features

* **Zero-Polling Reactivity**: Single resident Go daemon managed via Quickshell's `Process` model, streaming newline-delimited JSON (NDJSON) reactively by listening to kernel `sysfs` and D-Bus signals (`org.freedesktop.UDisks2` and `systemd`).
* **Lifecycle & Orphan Prevention**: Monitors standard input (`os.Stdin`) for `io.EOF` and traps `SIGPIPE` upon pipe closure by Quickshell, ensuring zero orphaned processes lingering on the bus.
* **Top-Bar Status Applet**:
  * Dynamic SVG health icons:
    * 🟢 **Normal**: Clean themed accent.
    * 🔵 **Active Operation**: Animated pulse during Scrub or Balance.
    * 🔴 **Degraded / Error**: Urgent warning for missing disks or SMART failures.
  * Real-time badges for used capacity percentage or current task progress (`Scrub: 42%`).
* **Interactive Flyout**:
  * **Header**: Mountpoint, UUID, real capacity gauge (`Free estimated` vs. `Used`), and fast Mount/Unmount button.
  * **Disk Topology**: Per-disk cards showing device nodes, drive model, and ATA S.M.A.R.T. health badges (temperature, bad sector detection).
  * **Maintenance & Scheduling**: One-click Scrub and Balance triggers with informative tooltips, real-time progress indicators, and an automated background maintenance toggle switch (systemd timers).
* **Minimal Polkit Footprint**: Standard mount/unmount is delegated to UDisks2 native Polkit permissions, while custom Polkit policies (`org.omarchy.btrfs.raidmanager.policy`) are strictly reserved for destructive device operations (`add`, `remove`, `replace`).
* **100% Vector Assets**: Clean, resolution-independent SVG icons.

---

## Architecture

```text
┌─────────────────────────────────────────────────────────────┐
│                    Omarchy Bar / Quickshell                 │
│              (Applet.qml  ◄──►  Flyout.qml)                 │
└──────────────────────────────▲──────────────────────────────┘
                               │ NDJSON via Stdout
┌──────────────────────────────▼──────────────────────────────┐
│                    raid-manager stream                      │
│            (Resident Go Event-Driven Monitor)               │
└──────────────┬───────────────────────────────┬──────────────┘
               │                               │
        Sysfs Telemetry                  D-Bus Signals
  (/sys/fs/btrfs/<uuid>/)         (UDisks2 & systemd1 Manager)
```

For in-depth architectural details, see [docs/architecture.md](docs/architecture.md).

---

## Requirements

* **Operating System**: Linux with systemd
* **Desktop**: Omarchy Desktop Environment with Quickshell
* **Filesystem Utilities**: `btrfs-progs` (v6.0+)
* **System Daemons**: `udisks2`, `systemd`, `polkit`
* **Go Compiler**: Go 1.22+ (for building from source)

---

## Installation & Setup

### 1. Build the Binary
```bash
git clone https://github.com/franelfers/omarchy-btrfs-raid-manager.git
cd omarchy-btrfs-raid-manager
go build -o bin/raid-manager ./cmd/raid-manager
```

### 2. Install Polkit Policy & Systemd Units
```bash
# Install Polkit policy for disk mutation operations
sudo cp polkit/org.omarchy.btrfs.raidmanager.policy /usr/share/polkit-1/actions/

# Install parametric maintenance units
sudo cp systemd/btrpool-* /etc/systemd/system/
sudo systemctl daemon-reload
```

### 3. Install Plugin into Omarchy
Link or clone into your Omarchy plugins directory:
```bash
mkdir -p ~/.config/omarchy/plugins
ln -s "$(pwd)" ~/.config/omarchy/plugins/org.omarchy.btrfs-raid-manager

# Enable the plugin in Omarchy
omarchy plugin enable org.omarchy.btrfs-raid-manager
```

---

## Testing & Verification

Run the test suite:
```bash
go test -v ./...
```

Validate the plugin manifest:
```bash
omarchy plugin validate .
```

Verify systemd unit syntax:
```bash
systemd-analyze verify systemd/btrpool-scrub@.service systemd/btrpool-balance@.service
```

Validate Polkit XML policy:
```bash
xmllint --noout polkit/org.omarchy.btrfs.raidmanager.policy
```

---

## License

This project is licensed under the **GNU General Public License v3.0** (GPL-3.0). See [LICENSE](LICENSE) for details.
