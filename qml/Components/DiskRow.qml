import QtQuick
import QtQuick.Layouts
import QtQuick.Effects
import qs.Commons
import qs.Ui as Ui
import "../Format.js" as Format
import "../ThemeStyle.js" as ThemeStyle

Rectangle {
  id: root

  property var device: ({})
  property string mountpoint: ""

  signal removeRequested(string devNode)

  readonly property bool isUrgent: Boolean(device.missing || device.smart_status === "failing")
  readonly property bool isWarning: Boolean(device.smart_status === "warning")
  readonly property color diskIconColor: ThemeStyle.diskIconColor(Color, hoverHandler.hovered, isUrgent, isWarning)
  readonly property var cardToken: ThemeStyle.cardStyle(Color, hoverHandler.hovered, false, isUrgent)

  implicitWidth: rowLayout.implicitWidth + Style.space(24)
  implicitHeight: rowLayout.implicitHeight + Style.space(16)
  color: cardToken.background
  radius: ThemeStyle.radiusFor("card", Style.cornerRadius)
  border.color: isUrgent
    ? Color.urgent
    : (isWarning ? ThemeStyle.warningColor(Color) : cardToken.border)
  border.width: cardToken.borderWidth

  HoverHandler {
    id: hoverHandler
  }

  RowLayout {
    id: rowLayout
    anchors.fill: parent
    anchors.margins: Style.space(8)
    spacing: Style.space(10)

    // Disk SVG icon
    Item {
      Layout.preferredWidth: ThemeStyle.iconSize("row")
      Layout.preferredHeight: ThemeStyle.iconSize("row")
      Layout.alignment: Qt.AlignVCenter
      opacity: device.missing ? 0.4 : 1.0

      Image {
        id: diskIconImg
        anchors.fill: parent
        source: Qt.resolvedUrl("../../assets/icon-disk.svg")
        sourceSize.width: ThemeStyle.iconSize("row")
        sourceSize.height: ThemeStyle.iconSize("row")
        fillMode: Image.PreserveAspectFit
        visible: false
        layer.enabled: true
      }

      MultiEffect {
        anchors.fill: parent
        source: diskIconImg
        colorization: 1.0
        colorizationColor: root.diskIconColor
      }
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
          font.pixelSize: ThemeStyle.fontSize("body")
          font.bold: true
          renderType: Text.NativeRendering
          color: device.missing ? Color.urgent : Color.foreground
        }

        Text {
          text: device.size_bytes > 0 ? Format.formatBytes(device.size_bytes) : ""
          font.family: Style.font.family
          font.pixelSize: ThemeStyle.fontSize("caption")
          renderType: Text.NativeRendering
          color: ThemeStyle.textSecondary(Color)
          visible: text !== ""
        }
      }

      Text {
        text: device.model ? (device.model + (device.serial ? " (" + device.serial + ")" : "")) : "Device #" + (device.dev_id || "")
        font.family: Style.font.family
        font.pixelSize: ThemeStyle.fontSize("caption")
        renderType: Text.NativeRendering
        color: ThemeStyle.textSecondary(Color)
        elide: Text.ElideRight
        Layout.fillWidth: true
      }

      // Errors counter if any exist
      Text {
        text: "Errors: Write " + (device.write_errs || 0) + " · Read " + (device.read_errs || 0) + " · Corruption " + (device.corruption_errs || 0)
        font.family: Style.font.family
        font.pixelSize: ThemeStyle.fontSize("captionSmall")
        renderType: Text.NativeRendering
        color: (device.write_errs > 0 || device.read_errs > 0 || device.corruption_errs > 0)
          ? Color.urgent
          : ThemeStyle.textMuted(Color)
        visible: (device.write_errs > 0 || device.read_errs > 0 || device.corruption_errs > 0)
      }
    }

    // SMART / Health Badge
    Rectangle {
      Layout.alignment: Qt.AlignVCenter
      implicitWidth: smartText.implicitWidth + Style.space(12)
      implicitHeight: smartText.implicitHeight + Style.space(6)
      radius: ThemeStyle.radiusFor("badge", Style.cornerRadius)

      readonly property string smartState: {
        if (device.missing) return "missing"
        if (device.smart_status === "failing") return "failing"
        if (device.smart_status === "warning") return "warning"
        if (device.smart_status === "passed") return "passed"
        return "muted"
      }
      readonly property var badgeToken: ThemeStyle.badgeStyle(Color, smartState)

      color: badgeToken.background
      border.color: badgeToken.border
      border.width: badgeToken.borderWidth

      Text {
        id: smartText
        anchors.centerIn: parent
        font.family: Style.font.family
        font.pixelSize: ThemeStyle.fontSize("captionSmall")
        font.bold: true
        renderType: Text.NativeRendering
        color: parent.badgeToken.text
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
