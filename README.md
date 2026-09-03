# Btrfs RAID Manager — Omarchy bar widget

Inspect, mount, and manage Btrfs RAID1 storage pools from the bar.

![Applet preview](preview.png)

State is streamed over NDJSON by a resident Go companion process via D-Bus (`UDisks2` and `systemd`) and `/sys/fs/btrfs/` — no periodic polling scripts, no spawning subcommands on a timer.

## Install

```sh
omarchy plugin add [https://github.com/franelfers/omarchy-btrfs-raid-manager.git](https://github.com/franelfers/omarchy-btrfs-raid-manager.git) --enable
omarchy bar move io.github.franelfers.btrfs-raid-manager --section right
omarchy restart shell
```

### Local development

```sh
git clone [https://github.com/franelfers/omarchy-btrfs-raid-manager.git](https://github.com/franelfers/omarchy-btrfs-raid-manager.git)
cd omarchy-btrfs-raid-manager
./install.sh --link --enable
```

## Usage

- **Bar glyph** — shows pool health at a glance (normal accent, pulsing during scrub/balance, urgent on degraded pool or SMART errors) with used percentage.
- **Click** — toggles the flyout panel.
- **Flyout**:
- Pool capacity gauge (`Free estimated` vs. `Used`) and mount/unmount toggle.
- Disk topology list with device nodes, models, and ATA S.M.A.R.T. health badges.
- Trigger or cancel scrub and balance tasks.
- Background maintenance schedule toggle (systemd timers).

## What it reads

|                       | Source                     | Shown                                                             |
| --------------------- | -------------------------- | ----------------------------------------------------------------- |
| Topology & Allocation | `/sys/fs/btrfs/<uuid>/`    | Real capacity, device list, raid profile                          |
| Mounts & Devices      | `org.freedesktop.UDisks2`  | Mount state, block topology, drive attachment                     |
| Maintenance progress  | `btrfs --format=json`      | Scrub and balance completion percentage                           |
| Drive health          | `smartctl`                 | Temperature and SMART attributes (falls back to `N/A` if missing) |
| Services & Timers     | `org.freedesktop.systemd1` | Scrub/balance unit state and timer schedules                      |

## Requirements

- `omarchy-shell` with Quickshell runtime
- `btrfs-progs`
- `smartmontools` _(optional, for SMART telemetry)_

Standard mount/unmount actions use native UDisks2 Polkit rules. Destructive mutations (`device add/remove/replace`) require the bundled Polkit policy:

```sh
sudo cp polkit/org.omarchy.btrfs.raidmanager.policy /usr/share/polkit-1/actions/
```

To enable background scheduled maintenance:

```sh
sudo cp systemd/btrpool-* /etc/systemd/system/
sudo systemctl daemon-reload
```

## Verification

```sh
go test -v ./...
omarchy plugin validate .
systemd-analyze verify systemd/btrpool-*
```

## License

GPL-3.0 — see [LICENSE](https://www.google.com/search?q=LICENSE).
