package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/franelfers/omarchy-btrfs-raid-manager/internal/admin"
	"github.com/franelfers/omarchy-btrfs-raid-manager/internal/btrfs"
	"github.com/franelfers/omarchy-btrfs-raid-manager/internal/parser"
	"github.com/franelfers/omarchy-btrfs-raid-manager/internal/sysfs"
	"github.com/franelfers/omarchy-btrfs-raid-manager/internal/systemd"
	"github.com/franelfers/omarchy-btrfs-raid-manager/internal/udisks"
	"github.com/godbus/dbus/v5"
)

var version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "stream":
		runStream()
	case "admin":
		if err := admin.Execute(os.Args[2:]); err != nil {
			admin.PrintError(err)
			os.Exit(1)
		}
	case "version", "--version", "-v":
		fmt.Printf("raid-manager %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`omarchy-btrfs-raid-manager: Storage Pool Daemon and Admin Tool

Usage:
  raid-manager stream                       Run resident NDJSON telemetry monitor
  raid-manager admin <subcommand> [args...] Run discrete privileged administrative mutations
  raid-manager version                      Print version information
  raid-manager help                         Print this help message

Admin subcommands:
  mount <device>                            Mount block device via UDisks2
  unmount <device>                          Unmount block device via UDisks2
  scrub <start|cancel> <mountpoint>         Trigger/stop parametric scrub service
  balance <start|cancel> <mountpoint>       Trigger/stop parametric balance service
  timer <enable|disable> <mp> <scrub|bal>   Enable/disable periodic maintenance timers
  add <device> <mountpoint>                 Add disk to pool (Polkit guarded)
  remove <device> <mountpoint>              Remove disk from pool (Polkit guarded)
  replace <old_dev> <new_dev> <mountpoint>  Replace disk in pool (Polkit guarded)`)
}

func runStream() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Trap OS termination signals including SIGPIPE
	sigChan := make(chan os.Signal, 2)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGPIPE)
	go func() {
		sig := <-sigChan
		_ = sig
		cancel()
	}()

	// 2. Monitor os.Stdin for EOF to prevent orphaned processes when Quickshell closes pipe
	go func() {
		buf := make([]byte, 512)
		for {
			_, err := os.Stdin.Read(buf)
			if err != nil {
				if err == io.EOF || isClosedPipe(err) {
					cancel()
					return
				}
			}
		}
	}()

	// 3. Connect to system D-Bus for reactive signal listening
	busConn, err := dbus.SystemBus()
	var uClient *udisks.Client
	var sysClient *systemd.Client
	var dbusSignals chan *dbus.Signal

	if err == nil && busConn != nil {
		defer busConn.Close()
		uClient = udisks.NewClient(busConn)
		sysClient = systemd.NewClient(busConn)

		_ = uClient.SubscribeEvents(busConn)
		_ = sysClient.SubscribeEvents(busConn)

		dbusSignals = make(chan *dbus.Signal, 64)
		busConn.Signal(dbusSignals)
	} else {
		fmt.Fprintf(os.Stderr, "Warning: failed to connect to system dbus: %v\n", err)
	}

	sysReader := sysfs.NewReader("")
	btrClient := btrfs.NewClient("")

	// 4. Helper function to capture and emit state
	emitState := func() bool {
		state, err := parser.BuildState(ctx, sysReader, uClient, sysClient, btrClient)
		if err != nil {
			return false
		}
		data, err := parser.MarshalNDJSON(state)
		if err != nil {
			return false
		}
		if _, err := os.Stdout.Write(data); err != nil {
			// Write failure on stdout (broken pipe) means Quickshell closed standard input/output
			cancel()
			return false
		}
		_ = os.Stdout.Sync()

		// Check if any pool has active scrub or balance running
		var hasActiveWork bool
		for _, p := range state.Pools {
			if p.Scrub.Active || p.Balance.Active {
				hasActiveWork = true
				break
			}
		}
		return hasActiveWork
	}

	// 5. Initial emission on startup
	hasActiveWork := emitState()

	// Debounce timer for reactive D-Bus bursts
	debounceTimer := time.NewTimer(0)
	if !debounceTimer.Stop() {
		select {
		case <-debounceTimer.C:
		default:
		}
	}
	var debouncePending bool

	// Dynamic ticker active only when background maintenance is running
	var activeTicker *time.Ticker
	var activeTickerC <-chan time.Time

	updateTicker := func(active bool) {
		if active {
			if activeTicker == nil {
				activeTicker = time.NewTicker(2 * time.Second)
				activeTickerC = activeTicker.C
			}
		} else {
			if activeTicker != nil {
				activeTicker.Stop()
				activeTicker = nil
				activeTickerC = nil
			}
		}
	}

	updateTicker(hasActiveWork)
	defer func() {
		if activeTicker != nil {
			activeTicker.Stop()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case <-dbusSignals:
			if !debouncePending {
				debouncePending = true
				debounceTimer.Reset(200 * time.Millisecond)
			}

		case <-debounceTimer.C:
			debouncePending = false
			hasActiveWork = emitState()
			updateTicker(hasActiveWork)

		case <-activeTickerC:
			hasActiveWork = emitState()
			updateTicker(hasActiveWork)
		}
	}
}

func isClosedPipe(err error) bool {
	if err == nil {
		return false
	}
	return os.IsNotExist(err) || err == syscall.EPIPE || err == syscall.EBADF
}
