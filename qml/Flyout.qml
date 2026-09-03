import QtQuick
import QtQuick.Layouts
import qs.Commons
import qs.Ui as Ui
import "Components"
import "Format.js" as Format
import "ThemeStyle.js" as ThemeStyle

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
    root.controller.show();
  }

  function close() {
    addDiskOpen = false;
    newDevicePath = "";
    root.controller.hide();
  }

  readonly property bool popoutSwitchClosing: panel ? panel.popoutSwitchClosing === true : false

  function closeForPopoutSwitch() {
    addDiskOpen = false;
    newDevicePath = "";
    if (panel && panel.closeForPopoutSwitch) {
      panel.closeForPopoutSwitch();
    } else {
      root.controller.hide();
    }
  }

  function toggle() {
    if (root.opened)
      root.close();
    else
      root.open();
  }

  Ui.KeyboardPanel {
    id: panel
    anchorItem: root.anchorItem
    owner: root.barIdentity
    bar: root.bar
    open: root.opened
    centerOnBar: false
    contentWidth: panel.fittedContentWidth(Style.space(460))
    contentHeight: panel.fittedContentHeight(contentColumn.implicitHeight)

    Flickable {
      id: scrollFlickable
      anchors.fill: parent
      contentWidth: width
      contentHeight: contentColumn.implicitHeight
      clip: true
      boundsBehavior: Flickable.StopAtBounds
      interactive: contentHeight > height

      ColumnLayout {
        id: contentColumn
        width: scrollFlickable.width
        spacing: Style.space(12)

        // Section 1: Header
        PoolHeader {
          Layout.fillWidth: true
          pool: root.pool
          onMountToggled: {
            if (root.pool.is_mounted) {
              root.runAdmin(["unmount", root.pool.uuid || root.pool.mountpoint]);
            } else {
              root.runAdmin(["mount", root.pool.uuid || (root.pool.devices && root.pool.devices.length > 0 ? root.pool.devices[0].dev_node : "")]);
            }
          }
        }

        Ui.PanelSeparator {
          Layout.fillWidth: true
        }

        // Section 2: Disk Topology
        RowLayout {
          Layout.fillWidth: true

          Ui.PanelSectionHeader {
            text: "Pool Disks"
            Layout.fillWidth: true
          }

          ActionButton {
            text: root.addDiskOpen ? "Cancel" : "+ Add Disk"
            tooltipText: root.addDiskOpen ? "Close partition selection dialog" : "Select an available block device or partition to expand this Btrfs RAID pool"
            onClicked: {
              root.addDiskOpen = !root.addDiskOpen;
              root.newDevicePath = "";
            }
          }
        }

        // Add Disk Dialog inline
        Rectangle {
          Layout.fillWidth: true
          visible: root.addDiskOpen
          implicitHeight: addColumn.implicitHeight + Style.space(16)
          readonly property var addCardToken: ThemeStyle.cardStyle(Color, false, true, false)
          radius: ThemeStyle.radiusFor("card", Style.cornerRadius)
          color: addCardToken.background
          border.color: addCardToken.border
          border.width: addCardToken.borderWidth

          ColumnLayout {
            id: addColumn
            anchors.fill: parent
            anchors.margins: Style.space(8)
            spacing: Style.space(8)

            Text {
              text: "Add Device to Pool (requires Polkit)"
              font.family: Style.font.family
              font.pixelSize: ThemeStyle.fontSize("caption")
              font.bold: true
              renderType: Text.NativeRendering
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
                    root.runAdmin(["add", root.newDevicePath.trim(), root.pool.mountpoint]);
                    root.addDiskOpen = false;
                    root.newDevicePath = "";
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
              onRemoveRequested: function (devNode) {
                if (root.pool.mountpoint) {
                  root.runAdmin(["remove", devNode, root.pool.mountpoint]);
                }
              }
            }
          }
        }

        Ui.PanelSeparator {
          Layout.fillWidth: true
        }

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
            active: Boolean(root.pool.scrub && root.pool.scrub.active)
            destructive: Boolean(root.pool.scrub && root.pool.scrub.active)
            tooltipText: (root.pool.scrub && root.pool.scrub.active) ? "Cancel the active background checksum verification and repair scrub" : "Verify data and metadata checksums across all RAID disks to detect and repair silent data corruption"
            onClicked: {
              if (root.pool.mountpoint) {
                var act = (root.pool.scrub && root.pool.scrub.active) ? "cancel" : "start";
                root.runAdmin(["scrub", act, root.pool.mountpoint]);
              }
            }
          }

          ActionButton {
            Layout.fillWidth: true
            text: (root.pool.balance && root.pool.balance.active) ? "Cancel Balance" : "Balance Pool"
            active: Boolean(root.pool.balance && root.pool.balance.active)
            destructive: Boolean(root.pool.balance && root.pool.balance.active)
            tooltipText: (root.pool.balance && root.pool.balance.active) ? "Cancel the active background chunk reallocation and balance" : "Compact block groups and reallocate data chunks across disks to reclaim unused storage capacity"
            onClicked: {
              if (root.pool.mountpoint) {
                var act = (root.pool.balance && root.pool.balance.active) ? "cancel" : "start";
                root.runAdmin(["balance", act, root.pool.mountpoint]);
              }
            }
          }
        }

        // Active operation status display
        ColumnLayout {
          Layout.fillWidth: true
          spacing: Style.space(4)
          visible: Boolean((root.pool.scrub && root.pool.scrub.active) || (root.pool.balance && root.pool.balance.active))

          RowLayout {
            Layout.fillWidth: true

            Text {
              text: {
                if (root.pool.scrub && root.pool.scrub.active) {
                  return "Scrubbing: " + Format.formatPercent(root.pool.scrub.progress_percent);
                }
                if (root.pool.balance && root.pool.balance.active) {
                  return "Balancing: " + Format.formatPercent(root.pool.balance.progress_percent);
                }
                return "";
              }
              font.family: Style.font.family
              font.pixelSize: ThemeStyle.fontSize("caption")
              font.bold: true
              renderType: Text.NativeRendering
              color: Color.accent
            }

            Item {
              Layout.fillWidth: true
            }

            Text {
              text: (root.pool.scrub && root.pool.scrub.active) ? ("Errors: " + (root.pool.scrub.errors || 0)) : ""
              font.family: Style.font.family
              font.pixelSize: ThemeStyle.fontSize("caption")
              renderType: Text.NativeRendering
              color: (root.pool.scrub && root.pool.scrub.errors > 0) ? Color.urgent : Color.foreground
            }
          }

          Rectangle {
            Layout.fillWidth: true
            height: Style.space(6)
            radius: ThemeStyle.radiusFor("gauge", Style.cornerRadius)
            color: ThemeStyle.colorWithAlpha(Color.foreground, 0.1)

            Rectangle {
              height: parent.height
              width: {
                var p = 0;
                if (root.pool.scrub && root.pool.scrub.active)
                  p = root.pool.scrub.progress_percent || 0;
                else if (root.pool.balance && root.pool.balance.active)
                  p = root.pool.balance.progress_percent || 0;
                return Math.min(parent.width, Math.max(0, parent.width * (p / 100.0)));
              }
              radius: ThemeStyle.radiusFor("gauge", Style.cornerRadius)
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
              font.pixelSize: ThemeStyle.fontSize("caption")
              font.bold: true
              renderType: Text.NativeRendering
              color: Color.foreground
            }

            Text {
              text: "Monthly scrub and weekly balance systemd timers"
              font.family: Style.font.family
              font.pixelSize: ThemeStyle.fontSize("captionSmall")
              renderType: Text.NativeRendering
              color: ThemeStyle.textSecondary(Color)
            }
          }

          ToggleSwitch {
            id: maintenanceSwitch
            checked: Boolean((root.pool.scrub && root.pool.scrub.timer_enabled) || (root.pool.balance && root.pool.balance.timer_enabled))
            tooltipText: "Enable or disable parametric systemd timers (btrpool-scrub@.timer and btrpool-balance@.timer) for scheduled monthly scrubs and weekly balance routines"
            onToggled: {
              if (!root.pool.mountpoint)
                return;
              var act = checked ? "enable" : "disable";
              root.runAdmin(["timer", act, root.pool.mountpoint, "scrub"]);
              root.runAdmin(["timer", act, root.pool.mountpoint, "balance"]);
            }
          }
        }
      }
    }
  }
}
