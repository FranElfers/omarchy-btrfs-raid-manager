# Btrfs RAID1 Guide & Operational Cheatsheet

## 1. Btrfs RAID1 Concepts

Unlike traditional hardware or mdadm RAID1 where every disk in an array is an identical block clone, Btrfs implements RAID1 at the **chunk allocation level**:

* **Two-Copy Invariant**: Under standard Btrfs `RAID1`, the filesystem ensures that for every block of data or metadata, exactly **two copies** reside on two **different physical disks**.
* **Arbitrary Disk Sizes**: Because chunks are paired dynamically, drives do not need to be of equal capacity. As long as no single drive holds more than 50% of the total raw capacity, 100% of the available mirrored space can be utilized.
* **Checksum Verification**: Every data and metadata block is protected with cryptographic checksums (typically CRC32C, XXHASH64, or BLAKE2b). When a read error or checksum mismatch occurs, Btrfs reads the undamaged copy from the mirror disk and automatically rewrites the corrupted block.
* **Degraded Mounting**: If a disk fails or drops offline, the pool can still be mounted with the `degraded` mount option (`-o degraded`), allowing user access to repair the array or add a replacement disk.

---

## 2. Kernel Sysfs Hierarchy

The kernel exposes real-time Btrfs statistics directly under `/sys/fs/btrfs/<uuid>/`:

| Path | Description |
|------|-------------|
| `label` | Filesystem volume label |
| `allocation/data/total_bytes` | Total allocated chunk capacity for data |
| `allocation/data/bytes_used` | Actual space occupied by user files |
| `allocation/data/<profile>/` | Subdirectory indicating active RAID profile (e.g. `raid1`, `single`) |
| `allocation/metadata/` | Metadata chunk allocations and usage |
| `devices/<name>` | Symlinks to underlying kernel block devices |
| `devinfo/<id>/missing` | Flag (0 or 1) indicating if the device is missing from the pool |
| `devinfo/<id>/error_stats` | Real-time write, read, flush, corruption, and generation error counts |

---

## 3. Maintenance Operations

### Scrubbing

Scrubbing sequentially reads all data and metadata across all disks, computes checksums, and compares them against stored checksum trees. Any bad sectors or corruptions are repaired using good mirrors.

- **Start foreground scrub**:
  ```bash
  btrfs scrub start -B /mnt/datos
  ```
- **Check scrub progress**:
  ```bash
  btrfs scrub status --raw /mnt/datos
  ```
- **Cancel running scrub**:
  ```bash
  btrfs scrub cancel /mnt/datos
  ```

### Filtered Balancing

Over time, writing and deleting files can leave allocated 1GB Btrfs chunks mostly empty, causing allocation exhaustion even when free space remains. Balancing consolidates data and returns empty chunks to unallocated storage.

- **Filtered balance (recommended)**:
  ```bash
  # Only compact data and metadata chunks that are under 50% utilized
  btrfs balance start -dusage=50 -musage=50 /mnt/datos
  ```
- **Check balance status**:
  ```bash
  btrfs balance status /mnt/datos
  ```
- **Cancel balance**:
  ```bash
  btrfs balance cancel /mnt/datos
  ```

---

## 4. Systemd Maintenance Services & Timers

`omarchy-btrfs-raid-manager` packages parametric units that delegate maintenance to systemd:

```text
systemd/btrpool-scrub@.service     # btrfs scrub start -B %f
systemd/btrpool-scrub@.timer       # Monthly trigger
systemd/btrpool-balance@.service   # btrfs balance start -dusage=50 -musage=50 %f
systemd/btrpool-balance@.timer     # Weekly trigger
```

Where the instance name `%i` corresponds to the `systemd-escape -p` representation of the mount point (e.g. `/mnt/datos` becomes `mnt-datos`).

Managing timers via CLI:
```bash
# Enable monthly scrub for /mnt/datos
systemctl enable --now btrpool-scrub@mnt-datos.timer

# Enable weekly balance for /mnt/datos
systemctl enable --now btrpool-balance@mnt-datos.timer
```

---

## 5. Disaster Recovery: Replacing a Failed Disk

If a drive fails in a RAID1 pool:

1. **Check Pool Status**:
   Inspect the degraded disk and note the missing device node or device ID:
   ```bash
   btrfs filesystem show /mnt/datos
   ```
2. **Mount in Degraded Mode (if necessary)**:
   ```bash
   mount -o degraded /dev/sda1 /mnt/datos
   ```
3. **Replace with New Disk**:
   ```bash
   # Replace old device (e.g. devid 2) with freshly partitioned new device (/dev/sdd1)
   btrfs replace start 2 /dev/sdd1 /mnt/datos
   ```
   Or remove the old device and add a new one:
   ```bash
   btrfs device add -f /dev/sdd1 /mnt/datos
   btrfs device remove missing /mnt/datos
   ```
4. **Balance the Array**:
   ```bash
   btrfs balance start -dusage=50 -musage=50 /mnt/datos
   ```
