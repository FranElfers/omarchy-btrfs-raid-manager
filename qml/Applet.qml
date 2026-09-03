import QtQuick
import QtQuick.Effects
import Quickshell
import Quickshell.Io
import qs.Commons
import qs.Ui as Ui
import "Format.js" as Format
import "ThemeStyle.js" as ThemeStyle

Item {
  id: root

  // Injected host properties from Bar
  property QtObject bar: null
  property string moduleName: "org.omarchy.btrfs-raid-manager"
  property var settings: ({})

  readonly property bool vertical: bar ? bar.vertical === true : false
  readonly property real padding: Style.space(6)
  readonly property int iconSize: bar ? Math.max(14, Math.round(bar.barSize * 0.6)) : 16

  implicitHeight: bar ? bar.barSize : 26
  implicitWidth: bar && bar.vertical ? (bar.barSize || 28) : Math.ceil(contentRow.implicitWidth + (padding * 2))

  // Parsed state from resident daemon
  property var pools: []
  readonly property var primaryPool: {
    if (!pools || pools.length === 0) return null
    for (var i = 0; i < pools.length; i++) {
      if (pools[i] && (pools[i].is_degraded || pools[i].status === "degraded")) {
        return pools[i]
      }
    }
    for (var j = 0; j < pools.length; j++) {
      if (pools[j] && (pools[j].status === "working" || (pools[j].scrub && pools[j].scrub.active) || (pools[j].balance && pools[j].balance.active))) {
        return pools[j]
      }
    }
    return pools[0]
  }

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
      return "Scrub " + Format.formatPercent(primaryPool.scrub.progress_percent)
    }
    if (primaryPool.balance && primaryPool.balance.active) {
      return "Bal " + Format.formatPercent(primaryPool.balance.progress_percent)
    }
    if (primaryPool.percent_used !== undefined) {
      return Format.formatPercent(primaryPool.percent_used)
    }
    return ""
  }

  // Commit identification for live verification
  property string commitId: "ad46176"
  readonly property bool tooltipHovered: visible && opacity > 0 && mouseArea.containsMouse && !root.opened

  Process {
    id: gitRevProcess
    command: {
      var pluginDir = Qt.resolvedUrl("..").toString().replace(/^file:\/\//, "")
      return ["git", "-C", pluginDir, "rev-parse", "--short=7", "HEAD"]
    }
    running: true
    stdout: SplitParser {
      onRead: function(line) {
        var trimmed = String(line || "").trim()
        if (trimmed.length >= 7) {
          root.commitId = trimmed.substring(0, 7)
        }
      }
    }
  }

  readonly property string tooltipSummary: {
    var prefix = "Btrfs RAID Manager (" + root.commitId + ")"
    if (!primaryPool) return prefix + ": No pools discovered"
    var summary = prefix + "\n"
    summary += (primaryPool.label || "RAID Pool") + " (" + (primaryPool.raid_profile || "RAID1") + ")\n"
    summary += "Status: " + poolStatus.toUpperCase() + "\n"
    summary += "Used: " + Format.formatBytes(primaryPool.used_bytes) + " / " + Format.formatBytes(primaryPool.total_bytes) + " (" + Format.formatPercent(primaryPool.percent_used) + ")\n"
    summary += "Disks: " + (primaryPool.devices ? primaryPool.devices.length : 0)
    return summary
  }

  readonly property bool hasSmartIssues: {
    if (!primaryPool || !primaryPool.devices) return false
    for (var i = 0; i < primaryPool.devices.length; i++) {
      var d = primaryPool.devices[i]
      if (d.missing === true) return true
      if (d.smart_status === "failing" || d.smart_status === "warning") return true
      if ((d.write_errs || 0) > 0 || (d.read_errs || 0) > 0 || (d.corruption_errs || 0) > 0) return true
    }
    return false
  }

  readonly property bool isAttentionNeeded: isDegraded || hasSmartIssues

  // Color tokens bound dynamically to host bar properties and theme
  readonly property color iconColor: {
    if (isAttentionNeeded) return bar ? bar.urgent : Color.urgent
    if (isWorking) return bar ? (bar.accent || Color.accent) : Color.accent
    return bar ? bar.foreground : Color.foreground
  }

  readonly property color textColor: iconColor

  // Popout coordination with Bar host
  readonly property bool opened: (bar && bar.activePopout === root) || (flyoutLoader.item ? flyoutLoader.item.opened === true : false)
  readonly property bool popoutSwitchClosing: flyoutLoader.item ? flyoutLoader.item.popoutSwitchClosing === true : false

  function open() {
    if (bar) {
      bar.hideTooltip(root)
      bar.requestPopout(root)
    }
    if (flyoutLoader.item && flyoutLoader.item.open) {
      flyoutLoader.item.open()
    }
  }

  function close() {
    if (bar && bar.activePopout === root) {
      bar.releasePopout(root)
    }
    if (flyoutLoader.item && flyoutLoader.item.close) {
      flyoutLoader.item.close()
    }
  }

  function closeForPopoutSwitch() {
    if (flyoutLoader.item && flyoutLoader.item.closeForPopoutSwitch) {
      flyoutLoader.item.closeForPopoutSwitch()
    } else if (flyoutLoader.item && flyoutLoader.item.close) {
      flyoutLoader.item.close()
    }
  }

  function toggle() {
    if (opened) close()
    else open()
  }

  Connections {
    target: root.bar || null
    function onActivePopoutChanged() {
      if (root.bar && root.bar.activePopout !== root && root.opened) {
        if (flyoutLoader.item && flyoutLoader.item.close) {
          flyoutLoader.item.close()
        }
      }
    }
  }

  function injectFlyout() {
    var target = flyoutLoader.item
    if (!target) return
    if ("bar" in target) target.bar = root.bar
    if ("settings" in target) target.settings = root.settings
    if ("anchorItem" in target) target.anchorItem = root
    if ("hostWidget" in target) target.hostWidget = root
    if ("pool" in target) target.pool = root.primaryPool || ({})
  }

  onPrimaryPoolChanged: {
    if (flyoutLoader.item && "pool" in flyoutLoader.item) {
      flyoutLoader.item.pool = root.primaryPool || ({})
    }
  }

  onBarChanged: injectFlyout()
  onSettingsChanged: injectFlyout()

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

  // 4. Background pill for hover and popout active states
  Rectangle {
    id: bg
    anchors.fill: parent
    anchors.margins: Style.space(2)
    radius: ThemeStyle.radiusFor("pill", Style.cornerRadius)
    color: ThemeStyle.interactiveBackground(
      root.bar || Color,
      mouseArea.containsMouse,
      root.opened,
      root.isAttentionNeeded
    )
    Behavior on color { ColorAnimation { duration: 150 } }
  }

  // 5. Clean horizontal flow: single icon followed by single text label
  Row {
    id: contentRow
    anchors.centerIn: parent
    spacing: Style.space(5)

    Image {
      id: iconImg
      width: root.iconSize
      height: root.iconSize
      source: root.statusIcon
      sourceSize: Qt.size(root.iconSize, root.iconSize)
      fillMode: Image.PreserveAspectFit
      visible: false
      layer.enabled: true
    }

    MultiEffect {
      width: root.iconSize
      height: root.iconSize
      anchors.verticalCenter: parent.verticalCenter
      source: iconImg
      colorization: 1.0
      colorizationColor: root.iconColor

      // Working animated pulse during scrub or balance
      SequentialAnimation on opacity {
        running: root.isWorking
        loops: Animation.Infinite
        alwaysRunToEnd: false
        NumberAnimation { to: 0.35; duration: 800; easing.type: Easing.InOutQuad }
        NumberAnimation { to: 1.0; duration: 800; easing.type: Easing.InOutQuad }
      }
    }

    Text {
      id: labelText
      textFormat: Text.PlainText
      anchors.verticalCenter: parent.verticalCenter
      visible: !root.vertical && root.badgeText !== ""
      text: root.badgeText
      color: root.textColor
      font.family: root.bar ? root.bar.fontFamily : Style.font.family
      font.pixelSize: ThemeStyle.fontSize("bodySmall")
      renderType: Text.NativeRendering
      verticalAlignment: Text.AlignVCenter
    }
  }

  // 6. Interaction Area
  MouseArea {
    id: mouseArea
    anchors.fill: parent
    hoverEnabled: true
    acceptedButtons: Qt.LeftButton | Qt.RightButton

    onEntered: {
      if (root.bar && root.tooltipSummary !== "" && !root.opened) {
        root.bar.showTooltip(root, root.tooltipSummary)
      }
    }

    onExited: {
      if (root.bar) {
        root.bar.hideTooltip(root)
      }
    }

    onClicked: function(mouse) {
      if (mouse.button === Qt.LeftButton) {
        root.toggle()
      }
    }
  }

  onOpenedChanged: {
    if (root.opened && root.bar) {
      root.bar.hideTooltip(root)
    }
  }

  Component.onDestruction: {
    if (root.bar) {
      root.bar.hideTooltip(root)
    }
  }
}
