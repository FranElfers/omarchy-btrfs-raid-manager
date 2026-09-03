# omarchy-btrfs-raid-manager: Architecture and Dataflow

## 1. System Overview

`omarchy-btrfs-raid-manager` is a top-bar applet and administration tool for Omarchy. It monitors Btrfs RAID1 storage pools without periodic polling. The applet displays pool health, disk capacity, and device telemetry while saving CPU cycles.

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

## 2. Event-Driven Design

Traditional applets run commands like `btrfs fi usage` or `df` on a timer. This wastes CPU cycles, drains battery, and delays UI updates.

`omarchy-btrfs-raid-manager` uses an event-driven model instead:

1. **Direct Kernel Sysfs Reads:**
   The daemon reads storage allocation, chunk profiles (`raid1`, `single`), error counters (`write_errs`, `read_errs`), and device sector sizes directly from `/sys/fs/btrfs/<uuid>/`. It spawns no child processes for telemetry.
2. **D-Bus Signal Subscriptions:**
   The `raid-manager stream` daemon listens to two D-Bus sources:
   - `org.freedesktop.UDisks2`: `PropertiesChanged` and `ObjectManager` signals report disk changes, mounts, unmounts, and SMART status.
   - `org.freedesktop.systemd1.Manager`: Unit lifecycle signals (`JobNew`, `JobRemoved`, `UnitNew`, `UnitRemoved`) report maintenance task progress.
3. **Debounced Updates:**
   The daemon debounces rapid signal bursts across a 200 ms sliding window. It sends only consolidated updates to standard output.
4. **Adaptive Ticking:**
   During active maintenance (scrub or balance), the daemon starts a 2-second timer to report progress percentages. The timer stops immediately when the operation ends.

---

## 3. Process Lifecycle

Quickshell reloads plugins dynamically. To prevent orphaned processes, `raid-manager stream` uses two safeguards:

- **Standard Input EOF:**
  Quickshell connects child processes with UNIX pipes. The Go daemon monitors `os.Stdin` in a dedicated goroutine. When Quickshell closes standard input (on reload or logout), the daemon receives `io.EOF`, cancels its context, and exits.
- **Signals and Broken Pipes:**
  The daemon traps `SIGINT`, `SIGTERM`, and `SIGPIPE`. If standard output closes during a write, the daemon stops immediately without hanging.

---

## 4. Security and Authorization

The project avoids long-lived root daemons and applies least privilege:

- **Mount and Unmount:** UDisks2 methods (`org.freedesktop.UDisks2.Filesystem.Mount` and `Unmount`) handle mounts through standard system rules.
- **Maintenance Tasks:** Systemd handles scrub and balance services and timers through native systemd Polkit rules.
- **Destructive Operations:** Custom Polkit policy (`polkit/org.omarchy.btrfs.raidmanager.policy`) covers only destructive disk actions:
  - `org.omarchy.btrfs.raidmanager.device-add`
  - `org.omarchy.btrfs.raidmanager.device-remove`
  - `org.omarchy.btrfs.raidmanager.device-replace`
  The UI calls short-lived CLI commands (`raid-manager admin ...`), which request root permissions through `pkexec` when needed.

---

## 5. UI Integration and NDJSON Protocol

Quickshell runs `raid-manager stream` with `Quickshell.Io.Process` and parses output lines with `SplitParser`. Each line is an independent JSON object:

```json
{
  "pools": [
    {
      "uuid": "b295f839-ce80-4f55-8ce2-5798fe2d816c",
      "label": "data_pool",
      "mountpoint": "/mnt/datos",
      "is_mounted": true,
      "total_bytes": 439028236288,
      "used_bytes": 23007428608,
      "free_bytes": 416020807680,
      "raw_total_bytes": 880198811648,
      "raw_used_bytes": 46014889984,
      "percent_used": 5.24,
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
          "model": "WDC WD3200BPVT",
          "serial": "WD-WX11E33J3770",
          "size_bytes": 320072933376,
          "smart_status": "passed",
          "smart_temperature_c": 24.0,
          "smart_bad_sectors": 0,
          "write_errs": 0,
          "read_errs": 0,
          "corruption_errs": 0
        },
        {
          "dev_node": "/dev/sdb1",
          "dev_id": 2,
          "missing": false,
          "model": "WDC WD3200BPVT",
          "serial": "WD-WXQ1CB1H4655",
          "size_bytes": 320072933376,
          "smart_status": "passed",
          "smart_temperature_c": 24.0,
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
  "timestamp": 1788453000
}
```

### 5.1 Storage Calculation Rules

Btrfs RAID profiles use data redundancy multipliers (for example, 2.0 for RAID1, RAID10, and DUP). The backend calculates storage to match `btrfs filesystem usage`:

- **Estimated Free Space (`free_bytes`):** Unallocated raw disk bytes divided by the profile multiplier, plus free space inside allocated data chunks.
- **Usable Used Space (`used_bytes`):** Logical data and metadata bytes occupied by user files (matching `df -h`).
- **Total Usable Capacity (`total_bytes`):** Sum of used bytes and estimated free bytes.
- **Raw Disk Metrics (`raw_used_bytes`, `raw_total_bytes`):** Physical disk bytes occupied across all disks and total physical drive capacity.

### 5.2 Pool Priority and Selection

The backend and UI sort pools in this order:

1. **Degraded pools:** Urgent alert priority.
2. **Active maintenance:** Pools running scrub or balance.
3. **RAID / Multi-device pools:** Prioritized over single-disk system partitions.
4. **Single-device pools:** Available as secondary pools.

When multiple pools exist, `Flyout.qml` displays selection tabs so users can switch between pools.

---

## 6. Functional Styling System (`ThemeStyle.js`)

All QML components get styles and computed tokens from `qml/ThemeStyle.js` (`import "ThemeStyle.js" as ThemeStyle`).

Components must not calculate colors, border math, or corner radii inline.

### 6.1 Radius Scale and Shape Rules

The function `radiusFor(elementType, baseRadius)` maps component roles to corner radii based on `Style.cornerRadius`:

```javascript
radiusFor(elementType, baseRadius)
```

- **Zero-Rounding Rule:** If `baseRadius <= 0`, `radiusFor` returns `0` for all elements. This keeps sharp corners consistent.
- **Radius Limits:**
  - `"card"`, `"dialog"`, `"row"`: `Math.min(base, 8)`
  - `"button"`, `"input"`, `"textfield"`, `"toggle"`, `"toggleRing"`: `Math.min(base, 6)`
  - `"badge"`, `"tag"`, `"tooltip"`: `Math.min(base, 4)`
  - `"gauge"`, `"track"`, `"progress"`: `Math.min(base, 3)`
  - `"pill"`: `Math.max(base, 12)`
  - default: `base`

### 6.2 API Reference

#### Color Utilities

- `resolveTheme(theme)`: Normalizes `theme` into `{ foreground, background, accent, urgent, muted, warning, cornerRadius }`.
- `colorWithAlpha(color, alpha)`: Parses any Qt color and returns `Qt.rgba(r, g, b, alpha)` clamped between `0.0` and `1.0`.
- `textSecondary(theme)`: Returns secondary text color at 70% opacity.
- `textMuted(theme)`: Returns muted text color at 48% opacity.
- `warningColor(theme)`: Returns the semantic warning color.
- `diskIconColor(theme, isHovered, isUrgent, isWarning)`: Returns disk row icon color (priority: urgent > warning > hover accent > foreground).
- `poolIconColor(theme, status)`: Returns pool status icon color (priority: degraded/urgent > active accent > healthy accent).

#### Interactive States

- `interactiveBackground(theme, isHovered, isActive, isUrgent)`: Returns surface background fill by priority (urgent active > urgent hover > urgent > active > hover > transparent).
- `interactiveBorder(theme, isHovered, isActive, isUrgent)`: Returns surface border color by priority (urgent > active > hover highlight > muted border).

#### Component Style Objects

- `cardStyle(theme, isHovered, isActive, isUrgent)`:
  Returns `{ background, border, borderWidth, radius }` for container cards.
- `buttonStyle(theme, isHovered, isActive, isUrgent, isPressed)`:
  Returns `{ background, border, borderWidth, foreground, radius }` for buttons.
- `toggleStyle(theme, isHovered, isChecked)`:
  Returns `{ trackBackground, trackBorder, knobColor, ringBorder, radius, trackRadius }` for toggle controls.
- `badgeStyle(theme, status)`:
  Returns `{ background, border, borderWidth, text, radius }` for status badges (`"missing"`, `"failing"`, `"warning"`, `"passed"`, `"working"`, `"muted"`).
- `tooltipStyle(theme)`:
  Returns `{ background, border, borderWidth, text, radius, fontSize }` for tooltip overlays.
- `progressGaugeStyle(theme, percent, isUrgent)`:
  Returns `{ trackColor, fillColor, radius }` for progress bars.

#### Typography and Sizes

- `fontSize(token)`: Standard font ramp:
  - `"captionSmall"`: 9px
  - `"caption"`: 10px
  - `"bodySmall"`: 11px
  - `"body"`: 12px
  - `"subtitle"`: 13px
  - `"title"`: 14px
  - `"heading"`: 16px
  - `"display"`: 24px
- `iconSize(token)`: Standard icon dimensions (`"small"`: 14, `"bar"`: 16, `"row"`: 24, `"header"`: 26, `"large"`: 32).
- `paddingFor(role)`: Standard padding sizes (`"control"`, `"card"`, `"badge"`, `"flyout"`).

### 6.3 Control Uniformity

- Controls (`ActionButton`, `ToggleSwitch`) use `ThemeStyle.radiusFor("button")` and `radiusFor("toggleRing")`. This ensures identical corner radii (`Math.min(base, 6)` or `0`).
- Flyout controls use `qml/Components/StyledToolTip.qml` for consistent backgrounds, borders, text colors, and corner rounding.

---

## 7. Related Documentation

- [CLI Reference](cli-reference.md): Subcommands, flags, and JSON response formats.
- [Btrfs Guide](btrfs-guide.md): Btrfs RAID1 operations, maintenance, and recovery.
- [References](references.md): External links and technical API specifications.
