import QtQuick
import QtQuick.Controls
import Quickshell
import Quickshell.Io
import qs.Commons
import qs.Ui as Ui
import "Format.js" as Format

Ui.BarWidget {
  id: root
  moduleName: "org.omarchy.btrfs-raid-manager"

  // Parsed state from resident daemon
  property var pools: []
  readonly property var primaryPool: pools.length > 0 ? pools[0] : null

  readonly property string poolStatus: primaryPool ? (primaryPool.status || "healthy") : "healthy"
  readonly property bool isDegraded: primaryPool ? (primaryPool.is_degraded === true) : false
  readonly property bool isWorking: primaryPool ? (primaryPool.status === "working") : false

  // Status icon dynamically reflecting pool condition
  readonly property string statusIcon: {
    if (isDegraded) return Qt.resolvedUrl("../assets/icon-pool-degraded.svg")
    if (isWorking) return Qt.resolvedUrl("../assets/icon-pool-working.svg")
    return Qt.resolvedUrl("../assets/icon-pool-healthy.svg")
  }

  // Status badge text
  readonly property string badgeText: {
    if (!primaryPool) return ""
    if (primaryPool.scrub && primaryPool.scrub.active) {
      return "Scrub: " + Format.formatPercent(primaryPool.scrub.progress_percent)
    }
    if (primaryPool.balance && primaryPool.balance.active) {
      return "Bal: " + Format.formatPercent(primaryPool.balance.progress_percent)
    }
    if (primaryPool.percent_used !== undefined) {
      return Format.formatPercent(primaryPool.percent_used)
    }
    return ""
  }

  readonly property string tooltipSummary: {
    if (!primaryPool) return "Btrfs RAID Manager: No pools discovered"
    var summary = (primaryPool.label || "RAID Pool") + " (" + (primaryPool.raid_profile || "RAID1") + ")\n"
    summary += "Status: " + poolStatus.toUpperCase() + "\n"
    summary += "Used: " + Format.formatBytes(primaryPool.used_bytes) + " / " + Format.formatBytes(primaryPool.total_bytes) + " (" + Format.formatPercent(primaryPool.percent_used) + ")\n"
    summary += "Disks: " + (primaryPool.devices ? primaryPool.devices.length : 0)
    return summary
  }

  function injectFlyout() {
    var target = flyoutLoader.item
    if (!target) return
    if ("bar" in target) target.bar = root.bar
    if ("settings" in target) target.settings = root.settings
    if ("anchorItem" in target) target.anchorItem = btn
    if ("hostWidget" in target) target.hostWidget = root
    if ("pool" in target) target.pool = root.primaryPool || ({})
  }

  onPrimaryPoolChanged: {
    if (flyoutLoader.item && "pool" in flyoutLoader.item) {
      flyoutLoader.item.pool = root.primaryPool || ({})
    }
  }

  onBarChanged: injectFlyout()

  function toggleFlyout() {
    if (flyoutLoader.item && flyoutLoader.item.toggle) {
      flyoutLoader.item.toggle()
    }
  }

  // Contract for shell popup panel routing
  readonly property bool opened: flyoutLoader.item ? flyoutLoader.item.opened === true : false
  function open() { if (flyoutLoader.item && flyoutLoader.item.open) flyoutLoader.item.open() }
  function close() { if (flyoutLoader.item && flyoutLoader.item.close) flyoutLoader.item.close() }

  implicitWidth: btn.implicitWidth
  implicitHeight: btn.implicitHeight

  // 1. Resident NDJSON Stream Process
  Process {
    id: streamProcess
    command: {
      var pluginDir = Qt.resolvedUrl("..").toString().replace(/^file:\/\//, "")
      var localBin = pluginDir + "/bin/raid-manager"
      return [
        "/bin/sh",
        "-c",
        "if [ -x \"" + localBin + "\" ]; then exec \"" + localBin + "\" stream; elif command -v raid-manager >/dev/null 2>&1; then exec raid-manager stream; elif command -v go >/dev/null 2>&1; then (cd \"" + pluginDir + "\" && go build -o bin/raid-manager ./cmd/raid-manager) && exec \"" + localBin + "\" stream; else echo 'raid-manager binary not found' >&2; exit 1; fi"
      ]
    }
    running: true

    stdout: SplitParser {
      onRead: function(line) {
        if (!line || line.trim() === "") return
        try {
          var state = JSON.parse(line)
          if (state && state.pools) {
            root.pools = state.pools
          }
        } catch (e) {
          console.warn("Failed to parse raid-manager NDJSON:", e, line)
        }
      }
    }
  }

  // 2. Discrete Mutation Process
  Process {
    id: adminProcess
    property var callback: null
    running: false

    function execute(args, cb) {
      var pluginDir = Qt.resolvedUrl("..").toString().replace(/^file:\/\//, "")
      var localBin = pluginDir + "/bin/raid-manager"
      var argStr = args.map(function(a) { return "'" + String(a).replace(/'/g, "'\\''") + "'" }).join(" ")
      command = [
        "/bin/sh",
        "-c",
        "if [ -x \"" + localBin + "\" ]; then exec \"" + localBin + "\" admin " + argStr + "; elif command -v raid-manager >/dev/null 2>&1; then exec raid-manager admin " + argStr + "; else echo 'raid-manager binary not found' >&2; exit 1; fi"
      ]
      callback = cb
      running = true
    }

    onExited: function(exitCode) {
      if (callback) {
        callback(exitCode)
        callback = null
      }
    }
  }

  // 3. Dropdown Flyout Panel Loader
  Loader {
    id: flyoutLoader
    active: true
    source: Qt.resolvedUrl("Flyout.qml")
    visible: false
    onLoaded: {
      root.injectFlyout()
      flyoutLoader.item.runAdmin.connect(function(args) {
        adminProcess.execute(args, null)
      })
    }
  }

  // 4. Bar Button Visual
  Ui.WidgetButton {
    id: btn
    anchors.fill: parent
    bar: root.bar
    text: root.vertical ? "" : root.badgeText
    labelVisible: !root.vertical && root.badgeText !== ""
    hasVisualContent: true
    tooltipText: root.tooltipSummary
    active: root.isWorking || root.isDegraded

    onPressed: function(mouse) {
      root.toggleFlyout()
    }

    // Working animated pulse
    SequentialAnimation on opacity {
      running: root.isWorking
      loops: Animation.Infinite
      alwaysRunToEnd: false
      NumberAnimation { to: 0.6; duration: 800; easing.type: Easing.InOutQuad }
      NumberAnimation { to: 1.0; duration: 800; easing.type: Easing.InOutQuad }
    }

    Row {
      anchors.centerIn: parent
      spacing: Style.space(6)
      visible: !root.vertical

      Image {
        width: Style.bar.iconSlot
        height: Style.bar.iconSlot
        source: root.statusIcon
        sourceSize.width: Style.bar.iconSlot
        sourceSize.height: Style.bar.iconSlot
        anchors.verticalCenter: parent.verticalCenter
      }

      Text {
        text: root.badgeText
        font.family: btn.fontFamily
        font.pixelSize: btn.fontSize
        color: root.isDegraded ? Color.urgent : (btn.active ? btn.activeColor : btn.foreground)
        anchors.verticalCenter: parent.verticalCenter
        visible: text !== ""
      }
    }
  }
}
