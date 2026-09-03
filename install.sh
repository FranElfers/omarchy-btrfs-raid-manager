#!/usr/bin/env bash
# Install (or reinstall) the Btrfs RAID Manager plugin into the Omarchy shell.
#
#   ./install.sh              copy the plugin into ~/.config/omarchy/plugins and reload
#   ./install.sh --link       symlink it instead, for development
#   ./install.sh --enable     install and immediately enable in the bar
#   ./install.sh --reload     rebuild, clear QML cache, and restart shell
#   ./install.sh --no-restart skip restarting the shell after install/update
#   ./install.sh --system     install systemd units and polkit policy (needs sudo)
#   ./install.sh --remove     uninstall, and delete the cache and state it wrote
#   ./install.sh --keep-data  with --remove, leave cache and state in place
#
# Adding it to the bar manually:
#   omarchy plugin enable org.omarchy.btrfs-raid-manager
#   omarchy bar move org.omarchy.btrfs-raid-manager --section right

set -euo pipefail

id="org.omarchy.btrfs-raid-manager"
source_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
target_dir="${XDG_CONFIG_HOME:-$HOME/.config}/omarchy/plugins/$id"
cache_dir="${XDG_CACHE_HOME:-$HOME/.cache}/omarchy/btrfs-raid-manager"
state_dir="${XDG_STATE_HOME:-$HOME/.local/state}/omarchy/btrfs-raid-manager"
mode="copy"
keep_data="no"
enable_now="no"
install_system="no"
restart_shell="yes"

for arg in "$@"; do
  case "$arg" in
  --link) mode="link" ;;
  --remove) mode="remove" ;;
  --enable) enable_now="yes" ;;
  --system) install_system="yes" ;;
  --keep-data) keep_data="yes" ;;
  --reload | --restart) restart_shell="yes" ;;
  --no-restart) restart_shell="no" ;;
  -h | --help)
    sed -n '2,15p' "${BASH_SOURCE[0]}" | sed 's/^# \?//'
    exit 0
    ;;
  *)
    echo "unknown option: $arg" >&2
    exit 1
    ;;
  esac
done

build_binary() {
  if command -v go >/dev/null 2>&1; then
    echo "Building raid-manager companion binary..."
    mkdir -p "$source_dir/bin"
    (cd "$source_dir" && go build -trimpath -ldflags="-s -w" -o "$source_dir/bin/raid-manager" ./cmd/raid-manager)
  elif [[ -x "$source_dir/bin/raid-manager" ]]; then
    echo "Using existing bin/raid-manager binary"
  else
    echo "Warning: go compiler not found and bin/raid-manager does not exist. Applet will attempt auto-build on launch." >&2
  fi
}

rescan_and_reload() {
  # 1. Purge Quickshell QML bytecode cache so updated QML files are freshly compiled
  local qs_cache="${XDG_CACHE_HOME:-$HOME/.cache}/quickshell/qmlcache"
  if [[ -d "$qs_cache" ]]; then
    rm -rf "$qs_cache"
  fi

  # 2. Rescan plugin manifests
  if command -v omarchy-shell >/dev/null 2>&1 && omarchy-shell shell ping >/dev/null 2>&1; then
    omarchy-shell shell rescanPlugins >/dev/null 2>&1 || true
    echo "shell plugins rescanned"

    # 3. Restart the shell to reload in-memory QML components
    if [[ $restart_shell == yes ]]; then
      echo "Restarting Omarchy shell to apply QML updates..."
      if command -v omarchy >/dev/null 2>&1; then
        omarchy restart shell >/dev/null 2>&1 || true
      elif command -v omarchy-restart-shell >/dev/null 2>&1; then
        omarchy-restart-shell >/dev/null 2>&1 || true
      fi
      echo "Omarchy shell reloaded. Plugin changes are now live!"
    fi
  else
    echo "Omarchy shell not running — it will load the plugin on next start."
  fi
}

# Optional system-wide installation of Polkit and systemd units
if [[ $install_system == yes ]]; then
  if [[ $EUID -ne 0 ]]; then
    echo "Error: --system requires root privileges (run with sudo ./install.sh --system)" >&2
    exit 1
  fi
  echo "Installing Polkit policy..."
  install -Dm644 "$source_dir/polkit/org.omarchy.btrfs.raidmanager.policy" /usr/share/polkit-1/actions/org.omarchy.btrfs.raidmanager.policy

  echo "Installing systemd units and timers..."
  install -Dm644 "$source_dir/systemd/btrpool-scrub@.service" /etc/systemd/system/btrpool-scrub@.service
  install -Dm644 "$source_dir/systemd/btrpool-scrub@.timer" /etc/systemd/system/btrpool-scrub@.timer
  install -Dm644 "$source_dir/systemd/btrpool-balance@.service" /etc/systemd/system/btrpool-balance@.service
  install -Dm644 "$source_dir/systemd/btrpool-balance@.timer" /etc/systemd/system/btrpool-balance@.timer
  systemctl daemon-reload
  echo "System components installed successfully."
  exit 0
fi

if [[ $mode == remove ]]; then
  if command -v omarchy-plugin-disable >/dev/null 2>&1; then
    omarchy-plugin-disable "$id" >/dev/null 2>&1 || true
  fi
  rm -rf "$target_dir"
  echo "removed $target_dir"
  if [[ $keep_data == yes ]]; then
    echo "kept $cache_dir and $state_dir"
  else
    for dir in "$cache_dir" "$state_dir"; do
      if [[ -d $dir ]]; then
        rm -rf "$dir"
        echo "removed $dir"
      fi
    done
  fi
  rescan_and_reload
  exit 0
fi

build_binary

mkdir -p "$(dirname "$target_dir")"
rm -rf "$target_dir"

if [[ $mode == link ]]; then
  ln -s "$source_dir" "$target_dir"
  echo "linked $target_dir -> $source_dir"
else
  mkdir -p "$target_dir"
  cp "$source_dir/manifest.json" "$target_dir/"
  cp -r "$source_dir/qml" "$target_dir/"
  cp -r "$source_dir/assets" "$target_dir/"
  if [[ -d "$source_dir/bin" ]]; then
    cp -r "$source_dir/bin" "$target_dir/"
  fi
  echo "installed $target_dir"
fi

if command -v omarchy-shell >/dev/null 2>&1 && omarchy-shell shell ping >/dev/null 2>&1; then
  omarchy-shell shell rescanPlugins >/dev/null 2>&1 || true
  echo "shell plugins rescanned"
fi

if [[ $enable_now == yes ]]; then
  if command -v omarchy-plugin-enable >/dev/null 2>&1; then
    omarchy-plugin-enable "$id" --section right || true
    echo "enabled $id and placed in bar"
  elif command -v omarchy >/dev/null 2>&1; then
    omarchy plugin enable "$id" || true
    omarchy bar move "$id" --section right || true
    echo "enabled $id and placed in bar"
  fi
fi

rescan_and_reload

if [[ $enable_now != yes ]]; then
  cat <<EOF

Next:
  omarchy plugin enable $id        # add it to the bar
  omarchy bar move $id --section right

Optional (system-wide maintenance timers & Polkit):
  sudo ./install.sh --system
EOF
fi
