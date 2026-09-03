# raid-manager CLI Reference

`raid-manager` is the companion CLI binary for `omarchy-btrfs-raid-manager`. It provides two modes:

1. `stream`: A resident daemon that emits pool telemetry over NDJSON.
2. `admin`: A discrete CLI tool for pool mutations.

---

## 1. Syntax Overview

```bash
raid-manager stream
raid-manager admin <subcommand> [arguments...]
raid-manager version
raid-manager help
```

---

## 2. Stream Mode (`raid-manager stream`)

Runs the resident event monitor. Quickshell executes this process via `Quickshell.Io.Process`.

### Behavior

- **Input Monitoring:** Monitors `os.Stdin` for `io.EOF`. The daemon exits immediately when the parent process closes standard input.
- **Signals:** Traps `SIGINT`, `SIGTERM`, and `SIGPIPE`. Stops immediately when standard output closes.
- **Output:** Emits one JSON object per line (NDJSON) to standard output.
- **Debounce:** Debounces D-Bus signal bursts across a 200 ms window.
- **Progress Ticking:** Emits progress updates every 2 seconds during active scrub or balance operations.

### Output Example

See [Architecture: NDJSON Protocol](architecture.md#5-ui-integration-and-ndjson-protocol) for the full payload schema.

---

## 3. Admin Mode (`raid-manager admin`)

Executes single administrative operations and exits.

### Response Format

Commands return a structured JSON response and a standard process exit code.

- **Success (exit code 0, stdout):**
  ```json
  {"success": true, "message": "mount operation succeeded"}
  ```
- **Error (exit code 1, stderr):**
  ```json
  {"success": false, "error": "device /dev/sda1 not found"}
  ```

### Subcommands

#### Mount and Unmount

Uses UDisks2 over D-Bus (`org.freedesktop.UDisks2.Filesystem`). Requires standard user privileges.

```bash
# Mount a block device
raid-manager admin mount /dev/sda1

# Unmount a block device
raid-manager admin unmount /dev/sda1
```

#### Maintenance Operations

Controls parametric systemd services (`btrpool-scrub@.service` and `btrpool-balance@.service`).

```bash
# Start or cancel a scrub
raid-manager admin scrub start /mnt/datos
raid-manager admin scrub cancel /mnt/datos

# Start or cancel a balance
raid-manager admin balance start /mnt/datos
raid-manager admin balance cancel /mnt/datos
```

#### Maintenance Timers

Enables or disables scheduled systemd timers (`btrpool-scrub@.timer` and `btrpool-balance@.timer`).

```bash
# Enable or disable monthly scrub schedule
raid-manager admin timer enable /mnt/datos scrub
raid-manager admin timer disable /mnt/datos scrub

# Enable or disable weekly balance schedule
raid-manager admin timer enable /mnt/datos balance
raid-manager admin timer disable /mnt/datos balance
```

#### Device Mutations (Privileged)

Mutations change array devices. These operations execute through `pkexec` and require Polkit authentication (`org.omarchy.btrfs.raidmanager.policy`).

```bash
# Add a new block device to a mounted pool
raid-manager admin add /dev/sdc1 /mnt/datos

# Remove a device from a mounted pool
raid-manager admin remove /dev/sdb1 /mnt/datos

# Replace a missing or failing disk with a new device
raid-manager admin replace 2 /dev/sdd1 /mnt/datos
```

---

## 4. Utility Commands

```bash
# Print binary version
raid-manager version

# Print CLI help and available commands
raid-manager help
```

---

## 5. Related Documentation

- [Architecture and Dataflow](architecture.md): System layout and D-Bus interfaces.
- [Btrfs Guide](btrfs-guide.md): Pool concepts, balance filters, and disk recovery.
- [References](references.md): Technical specifications and upstream links.
