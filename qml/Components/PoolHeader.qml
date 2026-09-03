import QtQuick
import QtQuick.Layouts
import qs.Commons
import qs.Ui as Ui
import "../Format.js" as Format

ColumnLayout {
  id: root

  property var pool: ({})

  signal mountToggled()

  spacing: Style.space(12)

  // Top title row: Label, Health Badge, Mount/Unmount button
  RowLayout {
    Layout.fillWidth: true
    spacing: Style.space(10)

    // Health Icon
    Image {
      Layout.preferredWidth: Style.space(26)
      Layout.preferredHeight: Style.space(26)
      source: {
        if (!pool.status) return Qt.resolvedUrl("../../assets/icon-pool-healthy.svg")
        if (pool.status === "degraded") return Qt.resolvedUrl("../../assets/icon-pool-degraded.svg")
        if (pool.status === "working") return Qt.resolvedUrl("../../assets/icon-pool-working.svg")
        return Qt.resolvedUrl("../../assets/icon-pool-healthy.svg")
      }
      sourceSize.width: Style.space(26)
      sourceSize.height: Style.space(26)
    }

    // Pool name and profile
    ColumnLayout {
      Layout.fillWidth: true
      spacing: Style.space(2)

      RowLayout {
        spacing: Style.space(8)

        Text {
          text: pool.label || "Storage Pool"
          font.family: Style.font.family
          font.pixelSize: Style.font.title
          font.bold: true
          renderType: Text.NativeRendering
          color: Color.foreground
        }

        // RAID Profile tag
        Rectangle {
          implicitWidth: profileText.implicitWidth + Style.space(8)
          implicitHeight: profileText.implicitHeight + Style.space(4)
          radius: Style.cornerRadius > 0 ? Math.min(Style.cornerRadius, 4) : 4
          color: Qt.rgba(Color.foreground.r, Color.foreground.g, Color.foreground.b, 0.08)
          border.color: Color.muted
          border.width: 1

          Text {
            id: profileText
            anchors.centerIn: parent
            text: pool.raid_profile || "RAID1"
            font.family: Style.font.family
            font.pixelSize: Style.font.caption * 0.85
            font.bold: true
            renderType: Text.NativeRendering
            color: Color.accent
          }
        }
      }

      Text {
        text: (pool.is_mounted ? pool.mountpoint : "Not mounted") + " · " + (pool.uuid ? pool.uuid.substring(0, 8) + "…" : "")
        font.family: Style.font.family
        font.pixelSize: Style.font.caption
        renderType: Text.NativeRendering
        color: Qt.darker(Color.foreground, 1.4)
        elide: Text.ElideMiddle
        Layout.fillWidth: true
      }
    }

    // Mount / Unmount button
    ActionButton {
      text: pool.is_mounted ? "Unmount" : "Mount"
      tooltipText: pool.is_mounted
        ? ("Safely unmount " + (pool.label || "RAID pool") + " from " + (pool.mountpoint || "filesystem") + " via UDisks2")
        : ("Mount " + (pool.label || "RAID pool") + " filesystem" + (pool.mountpoint ? " to " + pool.mountpoint : "") + " via UDisks2")
      onClicked: root.mountToggled()
    }
  }

  // Capacity usage gauge
  ColumnLayout {
    Layout.fillWidth: true
    spacing: Style.space(4)

    RowLayout {
      Layout.fillWidth: true

      Text {
        text: "Capacity"
        font.family: Style.font.family
        font.pixelSize: Style.font.caption
        renderType: Text.NativeRendering
        color: Qt.darker(Color.foreground, 1.3)
      }

      Item { Layout.fillWidth: true }

      Text {
        text: Format.formatBytes(pool.used_bytes) + " used of " + Format.formatBytes(pool.total_bytes) + " (" + Format.formatPercent(pool.percent_used) + ")"
        font.family: Style.font.family
        font.pixelSize: Style.font.caption
        renderType: Text.NativeRendering
        color: Color.foreground
      }
    }

    // Progress gauge bar
    Rectangle {
      Layout.fillWidth: true
      height: Style.space(8)
      radius: Style.cornerRadius > 0 ? Math.min(Style.cornerRadius, 4) : 4
      color: Qt.rgba(Color.foreground.r, Color.foreground.g, Color.foreground.b, 0.1)

      Rectangle {
        height: parent.height
        width: Math.min(parent.width, Math.max(0, parent.width * ((pool.percent_used || 0) / 100.0)))
        radius: Style.cornerRadius > 0 ? Math.min(Style.cornerRadius, 4) : 4
        color: {
          if (pool.percent_used > 90) return Color.urgent
          if (pool.percent_used > 75) return Color.warning
          return Color.accent
        }
      }
    }

    RowLayout {
      Layout.fillWidth: true

      Text {
        text: "Free (estimated): " + Format.formatBytes(pool.free_bytes)
        font.family: Style.font.family
        font.pixelSize: Style.font.caption * 0.9
        renderType: Text.NativeRendering
        color: Qt.darker(Color.foreground, 1.4)
      }
    }
  }
}
