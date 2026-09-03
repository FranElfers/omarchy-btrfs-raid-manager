# Btrfs RAID1 Guide and Operational Cheatsheet

## 1. Btrfs RAID1 Concepts

Traditional hardware or mdadm RAID1 mirrors entire disks. Btrfs implements RAID1 at the chunk allocation level instead:

- **Two-Copy Rule:** Btrfs RAID1 stores exactly two copies of each data and metadata block on two different physical disks.
- **Mixed Disk Sizes:** Drives do not require equal sizes. You can use all mirrored space as long as no single drive exceeds 50% of the total raw capacity.
- **Checksum Verification:** Every data and metadata block has a checksum (CRC32C, XXHASH64, or BLAKE2b). If a checksum fails, Btrfs reads the good copy from the mirror and repairs the damaged block.
- **Degraded Mounting:** If a drive fails, mount the pool with `-o degraded` to repair the array or replace the disk.

---

## 2. Kernel Sysfs Hierarchy

The Linux kernel exposes Btrfs pool status in `/sys/fs/btrfs/<uuid>/`:

| Path | Description |
|------|-------------|
| `label` | Filesystem volume label |
| `allocation/data/total_bytes` | Total allocated chunk space for data |
| `allocation/data/bytes_used` | Actual space occupied by user files |
| `allocation/data/<profile>/` | Active RAID profile (such as `raid1` or `single`) |
| `allocation/metadata/` | Metadata chunk allocations and usage |
| `devices/<name>` | Symlinks to underlying block devices |
| `devinfo/<id>/missing` | Flag (0 or 1) indicating if the device is missing |
| `devinfo/<id>/error_stats` | Error counters (write, read, flush, corruption, generation) |

---

## 3. Maintenance Operations

### Scrubbing

Scrubbing reads all data and metadata across all disks. It checks block checksums and repairs bad blocks from good mirrors.

- **Start foreground scrub:**
  ```bash
  btrfs scrub start -B /mnt/datos
  ```
- **Check scrub progress:**
  ```bash
  btrfs scrub status --raw /mnt/datos
  ```
- **Cancel running scrub:**
  ```bash
  btrfs scrub cancel /mnt/datos
  ```

### Filtered Balancing

Writing and deleting files can leave 1 GiB chunks mostly empty. This can cause allocation errors even with free space remaining. Balancing compacts data and frees unused chunks.

- **Run filtered balance (recommended):**
  ```bash
  # Compact data and metadata chunks that are under 50% utilized
  btrfs balance start -dusage=50 -musage=50 /mnt/datos
  ```
- **Check balance status:**
  ```bash
  btrfs balance status /mnt/datos
  ```
- **Cancel running balance:**
  ```bash
  btrfs balance cancel /mnt/datos
  ```

---

## 4. Systemd Maintenance Services and Timers

`omarchy-btrfs-raid-manager` provides template units for systemd:

```text
systemd/btrpool-scrub@.service     # Runs: btrfs scrub start -B %f
systemd/btrpool-scrub@.timer       # Monthly schedule
systemd/btrpool-balance@.service   # Runs: btrfs balance start -dusage=50 -musage=50 %f
systemd/btrpool-balance@.timer     # Weekly schedule
```

The instance name `%i` uses the escaped mount path (for example, `/mnt/datos` becomes `mnt-datos` via `systemd-escape -p`).

Manage timers with `systemctl`:

```bash
# Enable monthly scrub for /mnt/datos
systemctl enable --now btrpool-scrub@mnt-datos.timer

# Enable weekly balance for /mnt/datos
systemctl enable --now btrpool-balance@mnt-datos.timer
```

---

## 5. Disaster Recovery: Replace a Failed Disk

When a drive fails in a RAID1 pool:

1. **Check Pool Status:**
   Find the missing device node or device ID:
   ```bash
   btrfs filesystem show /mnt/datos
   ```
2. **Mount in Degraded Mode (if needed):**
   ```bash
   mount -o degraded /dev/sda1 /mnt/datos
   ```
3. **Replace the Failed Disk:**
   Replace the old device ID (for example, `2`) with a partitioned new device (`/dev/sdd1`):
   ```bash
   btrfs replace start 2 /dev/sdd1 /mnt/datos
   ```
   Or add the new device and remove the missing device:
   ```bash
   btrfs device add -f /dev/sdd1 /mnt/datos
   btrfs device remove missing /mnt/datos
   ```
4. **Balance the Array:**
   ```bash
   btrfs balance start -dusage=50 -musage=50 /mnt/datos
   ```

---

## 6. Related Documentation

- [Architecture and Dataflow](architecture.md): Internal design, D-Bus interfaces, and UI theme rules.
- [CLI Reference](cli-reference.md): Command-line options, subcommands, and JSON error reporting.
- [References](references.md): External documentation and API specifications.
