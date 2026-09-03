# Developer References and API Links

External documentation and reference links used by `omarchy-btrfs-raid-manager`.

## 1. Omarchy Plugin SDK and Quickshell

- **Omarchy Plugin Development Guide:** https://plugins.omarchy.org/develop.html
  Plugin layout, manifest format, and lifecycle validation.
- **Quickshell Documentation:** https://quickshell.outfoxxed.me/
  `Quickshell.Io.Process` for process execution and `Quickshell.Io.SplitParser` for line-by-line NDJSON parsing.
- **Omarchy Shell Source and Components:**
  - `qs.Ui`: `BarWidget`, `Panel`, `KeyboardPanel`, `WidgetButton`, `BorderSurface`.
  - `qs.Commons`: `Color`, `Style`, `Util`.

---

## 2. Btrfs Documentation

- **Official Btrfs Kernel Documentation:** https://btrfs.readthedocs.io/
- **Btrfs Wiki:** https://btrfs.wiki.kernel.org/
- **Btrfs Sysfs ABI Interface:** https://www.kernel.org/doc/Documentation/ABI/testing/sysfs-fs-btrfs
  - Path reference: `/sys/fs/btrfs/<uuid>/`
  - Allocation data: `allocation/data/`, `allocation/metadata/`
  - Device error counters: `devinfo/<id>/error_stats`

---

## 3. UDisks2 D-Bus API

- **UDisks2 API Specification:** https://storaged.org/doc/udisks2-api/latest/
- **Interfaces Used:**
  - `org.freedesktop.UDisks2.Manager`: Device discovery and object tracking.
  - `org.freedesktop.UDisks2.Block`: Block paths, sizes, and drive associations.
  - `org.freedesktop.UDisks2.Filesystem`: Standard Mount and Unmount methods.
  - `org.freedesktop.UDisks2.Drive.Ata`: SMART health, temperature, and bad sector attributes.

---

## 4. Systemd D-Bus API

- **Systemd D-Bus Specification:** https://www.freedesktop.org/wiki/Software/systemd/dbus/
- **Manager Interface (`org.freedesktop.systemd1.Manager`):**
  - Unit lifecycle control: `StartUnit`, `StopUnit`, `EnableUnitFiles`, `DisableUnitFiles`.
  - Signals: `JobNew`, `JobRemoved`, `UnitNew`, `UnitRemoved`.
- **Systemd Unit Manuals:**
  - `systemd.service(5)`: https://www.freedesktop.org/software/systemd/man/systemd.service.html
  - `systemd.timer(5)`: https://www.freedesktop.org/software/systemd/man/systemd.timer.html
  - Specifiers: `%f` (unescaped path), `%i` (instance name).

---

## 5. Polkit Authorization

- **Polkit Documentation:** https://www.freedesktop.org/software/polkit/docs/latest/
- **Policy File Path:** `/usr/share/polkit-1/actions/org.omarchy.btrfs.raidmanager.policy`
  - Action definitions for destructive mutations (`add`, `remove`, `replace`).

---

## 6. Go and D-Bus Libraries

- **Go Documentation:** https://go.dev/doc/
- **godbus/dbus:** https://pkg.go.dev/github.com/godbus/dbus/v5
  System bus connection, signal filtering, and property reflection.

---

## 7. Internal Project Documentation

- [Architecture and Dataflow](architecture.md): System components, D-Bus dataflow, and design system tokens.
- [CLI Reference](cli-reference.md): Command-line syntax, subcommands, and NDJSON streaming format.
- [Btrfs Guide](btrfs-guide.md): Pool concepts, sysfs paths, maintenance, and disk replacement.
