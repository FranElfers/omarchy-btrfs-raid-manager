import QtQuick
import QtQuick.Layouts
import qs.Commons
import qs.Ui as Ui
import "../Format.js" as Format
import "../ThemeStyle.js" as ThemeStyle

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
      Layout.preferredWidth: ThemeStyle.iconSize("header")
      Layout.preferredHeight: ThemeStyle.iconSize("header")
      source: {
        if (!pool.status) return Qt.resolvedUrl("../../assets/icon-pool-healthy.svg")
        if (pool.status === "degraded") return Qt.resolvedUrl("../../assets/icon-pool-degraded.svg")
        if (pool.status === "working") return Qt.resolvedUrl("../../assets/icon-pool-working.svg")
        return Qt.resolvedUrl("../../assets/icon-pool-healthy.svg")
      }
      sourceSize.width: ThemeStyle.iconSize("header")
      sourceSize.height: ThemeStyle.iconSize("header")
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
          font.pixelSize: ThemeStyle.fontSize("title")
          font.bold: true
          renderType: Text.NativeRendering
          color: Color.foreground
        }

        // RAID Profile tag
        Rectangle {
          readonly property var tagToken: ThemeStyle.badgeStyle(Color, "passed")
          implicitWidth: profileText.implicitWidth + Style.space(8)
          implicitHeight: profileText.implicitHeight + Style.space(4)
          radius: ThemeStyle.radiusFor("badge", Style.cornerRadius)
          color: tagToken.background
          border.color: tagToken.border
          border.width: tagToken.borderWidth

          Text {
            id: profileText
            anchors.centerIn: parent
            text: pool.raid_profile || "RAID1"
            font.family: Style.font.family
            font.pixelSize: ThemeStyle.fontSize("captionSmall")
            font.bold: true
            renderType: Text.NativeRendering
            color: Color.accent
          }
        }
      }

      Text {
        text: (pool.is_mounted ? pool.mountpoint : "Not mounted") + " · " + (pool.uuid ? pool.uuid.substring(0, 8) + "…" : "")
        font.family: Style.font.family
        font.pixelSize: ThemeStyle.fontSize("caption")
        renderType: Text.NativeRendering
        color: ThemeStyle.textSecondary(Color)
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

    readonly property var gaugeToken: ThemeStyle.progressGaugeStyle(Color, pool.percent_used, false)

    RowLayout {
      Layout.fillWidth: true

      Text {
        text: "Capacity"
        font.family: Style.font.family
        font.pixelSize: ThemeStyle.fontSize("caption")
        renderType: Text.NativeRendering
        color: ThemeStyle.textSecondary(Color)
      }

      Item { Layout.fillWidth: true }

      Text {
        text: Format.formatBytes(pool.used_bytes) + " used of " + Format.formatBytes(pool.total_bytes) + " (" + Format.formatPercent(pool.percent_used) + ")"
        font.family: Style.font.family
        font.pixelSize: ThemeStyle.fontSize("caption")
        renderType: Text.NativeRendering
        color: Color.foreground
      }
    }

    // Progress gauge bar
    Rectangle {
      Layout.fillWidth: true
      height: Style.space(8)
      radius: ThemeStyle.radiusFor("gauge", Style.cornerRadius)
      color: root.gaugeToken.trackColor

      Rectangle {
        height: parent.height
        width: Math.min(parent.width, Math.max(0, parent.width * ((pool.percent_used || 0) / 100.0)))
        radius: ThemeStyle.radiusFor("gauge", Style.cornerRadius)
        color: root.gaugeToken.fillColor
      }
    }

    RowLayout {
      Layout.fillWidth: true

      Text {
        text: "Free (estimated): " + Format.formatBytes(pool.free_bytes)
        font.family: Style.font.family
        font.pixelSize: ThemeStyle.fontSize("captionSmall")
        renderType: Text.NativeRendering
        color: ThemeStyle.textSecondary(Color)
      }
    }
  }
}
