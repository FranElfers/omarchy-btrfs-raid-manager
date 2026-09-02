# omarchy-btrfs-raid-manager: Architecture & Dataflow

## 1. System Architecture Overview

`omarchy-btrfs-raid-manager` is engineered as a lightweight, zero-polling, event-driven top-bar applet and administrative utility for Omarchy. It provides instant visibility into Btrfs RAID1 pool health, real storage capacity, and disk telemetry without spawning short-lived background polling commands or consuming unnecessary CPU cycles.

```mermaid
graph TD
    subgraph Kernel Space
        Sysfs["/sys/fs/btrfs/<uuid>/ (Allocation, DevInfo, Errors)"]
        BlockDevs["Block Layer (/dev/sdX, NVMe)"]
    end

    subgraph System Daemons & D-Bus
        UDisks["UDisks2 Daemon (org.freedesktop.UDisks2)"]
        Systemd["systemd1 Manager (btrpool-scrub / balance units)"]
        Polkit["Polkit Authority (org.omarchy.btrfs.raidmanager.policy)"]
    end

    subgraph Go Backend (raid-manager)
        StreamDaemon["raid-manager stream (Resident Process)"]
        AdminCLI["raid-manager admin (Discrete Mutation Commands)"]
    end

    subgraph Omarchy Quickshell UI
        Applet["Applet.qml (BarWidget)"]
        Flyout["Flyout.qml (Dropdown Panel)"]
    end

    Sysfs -->|Kernel telemetry| StreamDaemon
    UDisks -->|PropertiesChanged & Mount status| StreamDaemon
    Systemd -->|Job & Unit signals| StreamDaemon
    StreamDaemon -->|NDJSON via Stdout| Applet
    Applet -->|State binding| Flyout
    Flyout -->|User action| AdminCLI
    AdminCLI -->|Mount / Unmount| UDisks
    AdminCLI -->|Scrub / Balance / Timers| Systemd
    AdminCLI -->|Device Add / Remove / Replace| Polkit
```

---

## 2. Event-Driven Reactivity & Zero-Polling

Traditional desktop monitoring applets often rely on periodic shell scripts running commands like `btrfs fi usage` or `df` every few seconds. This creates process fork churn, increases wakeups on battery power, and introduces visual latency between operations.

`omarchy-btrfs-raid-manager` eliminates periodic polling through a hybrid event-driven model:

1. **Kernel Sysfs Direct Reading**:
   Telemetry regarding disk usage, allocation chunk profiles (`raid1`, `single`), and device error counters (`write_errs`, `read_errs`, `corruption_errs`) are read directly from `/sys/fs/btrfs/<uuid>/` with zero process execution overhead.
2. **D-Bus Signal Subscriptions**:
   The resident `raid-manager stream` daemon subscribes to:
   - `org.freedesktop.UDisks2`: `org.freedesktop.DBus.PropertiesChanged` and `org.freedesktop.DBus.ObjectManager` signals to detect drive insertions, unmounts, mountpoint updates, and SMART status changes.
   - `org.freedesktop.systemd1.Manager`: Unit lifecycle signals (`JobNew`, `JobRemoved`, `UnitNew`, `UnitRemoved`) to detect when maintenance tasks start, finish, or change state.
3. **Debounced Flush**:
   Rapid signal bursts on the bus (e.g. mounting with multiple partition signals) are automatically debounced across a 200ms sliding window, ensuring only cohesive, consolidated state updates are emitted to stdout.
4. **Adaptive Operation Ticking**:
   Only while an active background operation (such as scrub or balance) is running does the daemon enable a lightweight 2-second ticker to stream real-time progress percentages (`Scrub: 42%`). Once the operation finishes, the ticker immediately stops.

---

## 3. Lifecycle & Orphan Process Prevention

When developing plugins within Quickshell's dynamic reloading lifecycle, process management must be watertight to prevent stray daemons lingering on the system D-Bus.

`raid-manager stream` ensures clean process termination through dual safeguards:
* **Standard Input EOF Monitoring**:
  Quickshell connects child process standard I/O via UNIX pipes. The resident daemon continuously monitors `os.Stdin` in a dedicated goroutine. As soon as Quickshell closes the pipe (on plugin reload, bar restart, or logout), an `io.EOF` is received, immediately triggering context cancellation and graceful bus disconnection.
* **Signal Trapping & `SIGPIPE` Handling**:
  The daemon traps `SIGINT`, `SIGTERM`, and `SIGPIPE`. If Quickshell closes its stdout consumer while the daemon writes an NDJSON update, the kernel sends `SIGPIPE` or write returns `EPIPE`, terminating the daemon without hanging.

---

## 4. Security Architecture & Minimal Polkit Footprint

The project adheres to the principle of least privilege by refusing to introduce long-lived privileged daemons:

* **Mount and Unmount**: Delegated entirely to standard UDisks2 D-Bus methods (`org.freedesktop.UDisks2.Filesystem.Mount` / `Unmount`). These rely on system-standard UDisks2 authorization rules without requiring custom root helpers.
* **Maintenance Unit Lifecycles**: Scrub and balance services/timers are started, stopped, enabled, and disabled through `org.freedesktop.systemd1.Manager`, relying on systemd's built-in Polkit controls.
* **Destructive Disk Operations**: Custom Polkit policy (`polkit/org.omarchy.btrfs.raidmanager.policy`) is strictly reserved for destructive storage mutations:
  - `org.omarchy.btrfs.raidmanager.device-add`
  - `org.omarchy.btrfs.raidmanager.device-remove`
  - `org.omarchy.btrfs.raidmanager.device-replace`
  These actions are triggered on-demand as discrete short-lived CLI calls (`raid-manager admin ...`) which request elevation via `pkexec` when needed.

---

## 5. UI Integration & NDJSON Protocol

Quickshell executes `raid-manager stream` through `Quickshell.Io.Process` and parses lines using `SplitParser`. Each line is an independent JSON object:

```json
{
  "pools": [
    {
      "uuid": "b295f839-ce80-4f55-8ce2-5798fe2d816c",
      "label": "data_pool",
      "mountpoint": "/mnt/datos",
      "is_mounted": true,
      "total_bytes": 49392123904,
      "used_bytes": 22971301888,
      "free_bytes": 26420822016,
      "percent_used": 46.5,
      "raid_profile": "RAID1",
      "meta_profile": "RAID1",
      "status": "healthy",
      "is_degraded": false,
      "active_operation": "none",
      "operation_progress": 0,
      "devices": [
        {
          "dev_node": "/dev/sda1",
          "dev_id": 1,
          "missing": false,
          "model": "Crucial CT1000MX500SSD1",
          "serial": "2145E5E994B6",
          "size_bytes": 1000204886016,
          "smart_status": "passed",
          "smart_temperature_c": 28.0,
          "smart_bad_sectors": 0,
          "write_errs": 0,
          "read_errs": 0,
          "corruption_errs": 0
        }
      ],
      "scrub": {
        "active": false,
        "progress_percent": 0,
        "status_text": "idle",
        "scrubbed_bytes": 0,
        "total_bytes": 0,
        "errors": 0,
        "timer_enabled": true,
        "timer_active": false
      },
      "balance": {
        "active": false,
        "progress_percent": 0,
        "status_text": "idle",
        "timer_enabled": true,
        "timer_active": false
      }
    }
  ],
  "timestamp": 1788391700
}
```

The top-bar widget binds dynamically to `Omarchy.Theme` tokens and updates instantly when disk topology or maintenance state shifts.
