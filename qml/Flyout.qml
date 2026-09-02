import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import qs.Commons
import qs.Ui as Ui
import "Components"
import "Format.js" as Format

Ui.Panel {
  id: root
  moduleName: "org.omarchy.btrfs-raid-manager"
  ipcTarget: "org.omarchy.btrfs-raid-manager"
  manageIpc: false

  property var pool: ({})
  property var anchorItem: null
  property var hostWidget: null
  readonly property var barIdentity: hostWidget || root

  property bool addDiskOpen: false
  property string newDevicePath: ""

  signal runAdmin(var args)

  function open() {
    root.controller.show()
  }

  function close() {
    addDiskOpen = false
    newDevicePath = ""
    root.controller.hide()
  }

  function toggle() {
    if (root.opened) root.close()
    else root.open()
  }

  Ui.KeyboardPanel {
    id: panel
    anchorItem: root.anchorItem
    owner: root.barIdentity
    bar: root.bar
    open: root.opened
    centerOnBar: true
    contentWidth: Style.space(440)
    contentHeight: contentColumn.implicitHeight + Style.space(24)

    ColumnLayout {
      id: contentColumn
      anchors.fill: parent
      anchors.margins: Style.space(12)
      spacing: Style.space(12)

      // Section 1: Header
      PoolHeader {
        Layout.fillWidth: true
        pool: root.pool
        onMountToggled: {
          if (root.pool.is_mounted) {
            root.runAdmin(["unmount", root.pool.uuid || root.pool.mountpoint])
          } else {
            root.runAdmin(["mount", root.pool.uuid || (root.pool.devices && root.pool.devices.length > 0 ? root.pool.devices[0].dev_node : "")])
          }
        }
      }

      Ui.PanelSeparator { Layout.fillWidth: true }

      // Section 2: Disk Topology
      RowLayout {
        Layout.fillWidth: true

        Ui.PanelSectionHeader {
          text: "Pool Disks"
          Layout.fillWidth: true
        }

        ActionButton {
          text: root.addDiskOpen ? "Cancel" : "+ Add Disk"
          tooltipText: root.addDiskOpen ? "Close add disk dialog" : "Add a new block device to this RAID pool"
          onClicked: {
            root.addDiskOpen = !root.addDiskOpen
            root.newDevicePath = ""
          }
        }
      }

      // Add Disk Dialog inline
      Rectangle {
        Layout.fillWidth: true
        visible: root.addDiskOpen
        implicitHeight: addColumn.implicitHeight + Style.space(16)
        radius: Style.radius(6)
        color: Style.surfaceFillFor("popups", 0.05)
        border.color: Color.accent
        border.width: 1

        ColumnLayout {
          id: addColumn
          anchors.fill: parent
          anchors.margins: Style.space(8)
          spacing: Style.space(8)

          Text {
            text: "Add Device to Pool (requires Polkit)"
            font.family: Style.font.family
            font.pixelSize: Style.font.caption
            font.bold: true
            color: Color.foreground
          }

          RowLayout {
            Layout.fillWidth: true
            spacing: Style.space(8)

            Ui.TextField {
              id: devInput
              Layout.fillWidth: true
              placeholderText: "/dev/sdd1"
              text: root.newDevicePath
              onTextChanged: root.newDevicePath = text
            }

            ActionButton {
              text: "Add"
              enabled: root.newDevicePath.trim() !== ""
              tooltipText: "Execute btrfs device add on " + root.newDevicePath
              onClicked: {
                if (root.newDevicePath.trim() !== "" && root.pool.mountpoint) {
                  root.runAdmin(["add", root.newDevicePath.trim(), root.pool.mountpoint])
                  root.addDiskOpen = false
                  root.newDevicePath = ""
                }
              }
            }
          }
        }
      }

      // Disk list
      ColumnLayout {
        Layout.fillWidth: true
        spacing: Style.space(6)

        Repeater {
          model: root.pool.devices || []

          DiskRow {
            Layout.fillWidth: true
            device: modelData
            mountpoint: root.pool.mountpoint || ""
            onRemoveRequested: function(devNode) {
              if (root.pool.mountpoint) {
                root.runAdmin(["remove", devNode, root.pool.mountpoint])
              }
            }
          }
        }
      }

      Ui.PanelSeparator { Layout.fillWidth: true }

      // Section 3: Maintenance Bar
      Ui.PanelSectionHeader {
        text: "Maintenance & Health"
        Layout.fillWidth: true
      }

      RowLayout {
        Layout.fillWidth: true
        spacing: Style.space(10)

        ActionButton {
          Layout.fillWidth: true
          text: (root.pool.scrub && root.pool.scrub.active) ? "Cancel Scrub" : "Scrub Pool"
          active: root.pool.scrub && root.pool.scrub.active
          destructive: root.pool.scrub && root.pool.scrub.active
          tooltipText: "Scan and verify data checksums across disks to detect and repair silent data corruption"
          onClicked: {
            if (root.pool.mountpoint) {
              var act = (root.pool.scrub && root.pool.scrub.active) ? "cancel" : "start"
              root.runAdmin(["scrub", act, root.pool.mountpoint])
            }
          }
        }

        ActionButton {
          Layout.fillWidth: true
          text: (root.pool.balance && root.pool.balance.active) ? "Cancel Balance" : "Balance Pool"
          active: root.pool.balance && root.pool.balance.active
          destructive: root.pool.balance && root.pool.balance.active
          tooltipText: "Reallocate under-utilized chunks across drives to reclaim unused storage capacity"
          onClicked: {
            if (root.pool.mountpoint) {
              var act = (root.pool.balance && root.pool.balance.active) ? "cancel" : "start"
              root.runAdmin(["balance", act, root.pool.mountpoint])
            }
          }
        }
      }

      // Active operation status display
      ColumnLayout {
        Layout.fillWidth: true
        spacing: Style.space(4)
        visible: (root.pool.scrub && root.pool.scrub.active) || (root.pool.balance && root.pool.balance.active)

        RowLayout {
          Layout.fillWidth: true

          Text {
            text: {
              if (root.pool.scrub && root.pool.scrub.active) {
                return "Scrubbing: " + Format.formatPercent(root.pool.scrub.progress_percent)
              }
              if (root.pool.balance && root.pool.balance.active) {
                return "Balancing: " + Format.formatPercent(root.pool.balance.progress_percent)
              }
              return ""
            }
            font.family: Style.font.family
            font.pixelSize: Style.font.caption
            font.bold: true
            color: Color.accent
          }

          Item { Layout.fillWidth: true }

          Text {
            text: (root.pool.scrub && root.pool.scrub.active) ? ("Errors: " + (root.pool.scrub.errors || 0)) : ""
            font.family: Style.font.family
            font.pixelSize: Style.font.caption
            color: (root.pool.scrub && root.pool.scrub.errors > 0) ? Color.urgent : Color.foreground
          }
        }

        Rectangle {
          Layout.fillWidth: true
          height: Style.space(6)
          radius: Style.radius(3)
          color: Style.surfaceFillFor("popups", 0.1)

          Rectangle {
            height: parent.height
            width: {
              var p = 0
              if (root.pool.scrub && root.pool.scrub.active) p = root.pool.scrub.progress_percent || 0
              else if (root.pool.balance && root.pool.balance.active) p = root.pool.balance.progress_percent || 0
              return Math.min(parent.width, Math.max(0, parent.width * (p / 100.0)))
            }
            radius: Style.radius(3)
            color: Color.accent
          }
        }
      }

      // Automated schedule switch
      RowLayout {
        Layout.fillWidth: true
        spacing: Style.space(8)

        ColumnLayout {
          Layout.fillWidth: true
          spacing: Style.space(1)

          Text {
            text: "Automated Maintenance"
            font.family: Style.font.family
            font.pixelSize: Style.font.caption
            font.bold: true
            color: Color.foreground
          }

          Text {
            text: "Monthly scrub and weekly balance systemd timers"
            font.family: Style.font.family
            font.pixelSize: Style.font.caption * 0.85
            color: Qt.darker(Color.foreground, 1.4)
          }
        }

        Ui.ToggleSwitch {
          checked: (root.pool.scrub && root.pool.scrub.timer_enabled) || (root.pool.balance && root.pool.balance.timer_enabled)
          onToggled: {
            if (!root.pool.mountpoint) return
            var act = checked ? "enable" : "disable"
            root.runAdmin(["timer", act, root.pool.mountpoint, "scrub"])
            root.runAdmin(["timer", act, root.pool.mountpoint, "balance"])
          }
        }
      }
    }
  }
}
