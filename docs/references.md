# Developer References & Documentation Links

This document aggregates authoritative external documentation and reference links utilized in the design and implementation of `omarchy-btrfs-raid-manager`.

## 1. Omarchy Plugin SDK & Quickshell

* **Omarchy Plugin Development Guide**: https://plugins.omarchy.org/develop.html
  - Plugin directory specifications, manifest format, and lifecycle validation.
* **Quickshell Documentation**: https://quickshell.outfoxxed.me/
  - `Quickshell.Io.Process`: Asynchronous process spawning, stdin/stdout stream redirection.
  - `Quickshell.Io.SplitParser`: Line-by-line NDJSON parsing.
* **Omarchy Shell Source & Component Library**:
  - `qs.Ui`: `BarWidget`, `Panel`, `KeyboardPanel`, `WidgetButton`, `BorderSurface`.
  - `qs.Commons`: `Color`, `Style`, `Util`.

---

## 2. Btrfs Documentation

* **Official Btrfs Kernel Documentation**: https://btrfs.readthedocs.io/
* **Btrfs Wiki & Administration Cheatsheet**: https://btrfs.wiki.kernel.org/
* **Btrfs Sysfs Telemetry Interface**: https://www.kernel.org/doc/Documentation/ABI/testing/sysfs-fs-btrfs
  - Path reference: `/sys/fs/btrfs/<uuid>/`
  - Allocation statistics: `allocation/data/`, `allocation/metadata/`
  - Kernel error counters: `devinfo/<id>/error_stats`

---

## 3. UDisks2 D-Bus API

* **UDisks2 API Specification**: https://storaged.org/doc/udisks2-api/latest/
* **Interfaces Used**:
  - `org.freedesktop.UDisks2.Manager`: Object discovery and tracking.
  - `org.freedesktop.UDisks2.Block`: Block device paths, sizes, and drive associations.
  - `org.freedesktop.UDisks2.Filesystem`: Standard Mount and Unmount methods.
  - `org.freedesktop.UDisks2.Drive.Ata`: S.M.A.R.T. health, temperature, and sector status.

---

## 4. Systemd D-Bus API

* **Systemd D-Bus Interface**: https://www.freedesktop.org/wiki/Software/systemd/dbus/
* **Manager Interface (`org.freedesktop.systemd1.Manager`)**:
  - Unit lifecycle control: `StartUnit`, `StopUnit`, `EnableUnitFiles`, `DisableUnitFiles`.
  - Signal monitoring: `JobNew`, `JobRemoved`, `UnitNew`, `UnitRemoved`.
* **Systemd Unit Documentation**:
  - `systemd.service(5)`: https://www.freedesktop.org/software/systemd/man/systemd.service.html
  - `systemd.timer(5)`: https://www.freedesktop.org/software/systemd/man/systemd.timer.html
  - `systemd.unit(5)` specifiers: `%f` (unescaped path), `%i` (instance name).

---

## 5. Polkit Authorization

* **Polkit Documentation**: https://www.freedesktop.org/software/polkit/docs/latest/
* **Policy File Reference**: `/usr/share/polkit-1/actions/`
  - Syntax specifications for `.policy` XML files.
  - `<defaults>` mapping for active and inactive console sessions.

---

## 6. Go & D-Bus Libraries

* **Go Language Documentation**: https://go.dev/doc/
* **godbus/dbus**: https://pkg.go.dev/github.com/godbus/dbus/v5
  - Connecting to `dbus.SystemBus()`.
  - Signal filtering and object property reflection.
