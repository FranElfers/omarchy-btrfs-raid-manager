## Summary of Changes

A concise description of the purpose and design of this PR.

## Related Issues

Fixes #(issue)

## Key Technical Decisions & Invariants Respected

- [ ] **Simplicity First**: No periodic polling loops introduced; reactive via D-Bus / sysfs.
- [ ] **Lifecycle & Orphan Prevention**: Daemon processes terminate cleanly on EOF / SIGPIPE.
- [ ] **Minimal Polkit Footprint**: Privileged mutations remain discrete CLI commands; standard Polkit rules utilized for UDisks2 and systemd.
- [ ] **Vectors Only**: Any new icons/glyphs are SVG.

## Testing & Verification

- [ ] `go test ./...` passes (with mock sysfs and D-Bus fixtures).
- [ ] `omarchy plugin validate .` passes.
- [ ] Systemd units verified via `systemd-analyze verify`.
- [ ] Polkit XML validated via `xmllint`.
- [ ] Tested on target hardware / live btrfs pool.
