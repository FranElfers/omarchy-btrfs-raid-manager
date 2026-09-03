import QtQuick
import QtQuick.Layouts
import qs.Commons
import qs.Ui as Ui
import "../Format.js" as Format

Rectangle {
  id: root

  property var device: ({})
  property string mountpoint: ""

  signal removeRequested(string devNode)

  implicitWidth: rowLayout.implicitWidth + Style.space(24)
  implicitHeight: rowLayout.implicitHeight + Style.space(16)
  color: Style.surfaceFillFor("popups", 0.03)
  radius: Style.radius(6)
  border.color: device.missing || device.smart_status === "failing"
    ? Color.urgent
    : (device.smart_status === "warning" ? Color.warning : Color.border)
  border.width: 1

  RowLayout {
    id: rowLayout
    anchors.fill: parent
    anchors.margins: Style.space(8)
    spacing: Style.space(10)

    // Disk SVG icon
    Image {
      Layout.preferredWidth: Style.space(24)
      Layout.preferredHeight: Style.space(24)
      Layout.alignment: Qt.AlignVCenter
      source: Qt.resolvedUrl("../../assets/icon-disk.svg")
      sourceSize.width: Style.space(24)
      sourceSize.height: Style.space(24)
      opacity: device.missing ? 0.4 : 1.0
    }

    // Disk details
    ColumnLayout {
      Layout.fillWidth: true
      spacing: Style.space(2)

      RowLayout {
        spacing: Style.space(8)

        Text {
          text: device.dev_node || "Unknown"
          font.family: Style.font.family
          font.pixelSize: Style.font.body
          font.bold: true
          renderType: Text.NativeRendering
          color: device.missing ? Color.urgent : Color.foreground
        }

        Text {
          text: device.size_bytes > 0 ? Format.formatBytes(device.size_bytes) : ""
          font.family: Style.font.family
          font.pixelSize: Style.font.caption
          renderType: Text.NativeRendering
          color: Qt.darker(Color.foreground, 1.4)
          visible: text !== ""
        }
      }

      Text {
        text: device.model ? (device.model + (device.serial ? " (" + device.serial + ")" : "")) : "Device #" + (device.dev_id || "")
        font.family: Style.font.family
        font.pixelSize: Style.font.caption
        renderType: Text.NativeRendering
        color: Qt.darker(Color.foreground, 1.3)
        elide: Text.ElideRight
        Layout.fillWidth: true
      }

      // Errors counter if any exist
      Text {
        text: "Errors: Write " + (device.write_errs || 0) + " · Read " + (device.read_errs || 0) + " · Corruption " + (device.corruption_errs || 0)
        font.family: Style.font.family
        font.pixelSize: Style.font.caption * 0.9
        renderType: Text.NativeRendering
        color: (device.write_errs > 0 || device.read_errs > 0 || device.corruption_errs > 0)
          ? Color.urgent
          : Qt.darker(Color.foreground, 1.6)
        visible: (device.write_errs > 0 || device.read_errs > 0 || device.corruption_errs > 0)
      }
    }

    // SMART / Health Badge
    Rectangle {
      Layout.alignment: Qt.AlignVCenter
      implicitWidth: smartText.implicitWidth + Style.space(12)
      implicitHeight: smartText.implicitHeight + Style.space(6)
      radius: Style.radius(4)
      color: {
        if (device.missing) return Qt.rgba(0.9, 0.2, 0.2, 0.15)
        if (device.smart_status === "failing") return Qt.rgba(0.9, 0.2, 0.2, 0.15)
        if (device.smart_status === "warning") return Qt.rgba(0.95, 0.6, 0.1, 0.15)
        if (device.smart_status === "passed") return Qt.rgba(0.2, 0.8, 0.3, 0.15)
        return Qt.rgba(0.5, 0.5, 0.5, 0.15)
      }
      border.color: {
        if (device.missing) return Color.urgent
        if (device.smart_status === "failing") return Color.urgent
        if (device.smart_status === "warning") return Color.warning
        if (device.smart_status === "passed") return Color.accent
        return Color.border
      }
      border.width: 1

      Text {
        id: smartText
        anchors.centerIn: parent
        font.family: Style.font.family
        font.pixelSize: Style.font.caption * 0.9
        font.bold: true
        renderType: Text.NativeRendering
        color: {
          if (device.missing) return Color.urgent
          if (device.smart_status === "failing") return Color.urgent
          if (device.smart_status === "warning") return Color.warning
          if (device.smart_status === "passed") return Color.accent
          return Qt.darker(Color.foreground, 1.4)
        }
        text: {
          if (device.missing) return "MISSING"
          if (device.smart_status === "failing") return "SMART FAILING"
          if (device.smart_status === "warning") return "SECTOR WARNING"
          if (device.smart_status === "passed") {
            var t = "SMART OK"
            if (device.smart_temperature_c > 0) {
              t += " (" + Math.round(device.smart_temperature_c) + "°C)"
            }
            return t
          }
          if (device.smart_status === "disabled" || device.smart_status === "unsupported") return "SMART N/A"
          return "SMART N/A"
        }
      }
    }

    // Action button to remove disk
    ActionButton {
      Layout.alignment: Qt.AlignVCenter
      text: "Remove"
      destructive: true
      tooltipText: "Remove this disk from the Btrfs pool (requires Polkit authentication)"
      visible: device.missing || (device.smart_status === "failing")
      onClicked: root.removeRequested(device.dev_node || String(device.dev_id))
    }
  }
}
