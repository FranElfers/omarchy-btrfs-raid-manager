import QtQuick
import qs.Commons
import "../ThemeStyle.js" as ThemeStyle

Item {
  id: root

  property string text: ""
  property string tooltipText: ""
  property bool enabled: true
  property bool active: false
  property bool destructive: false
  property color foreground: Color.foreground
  property color accent: destructive ? Color.urgent : Color.accent

  signal clicked()

  implicitWidth: Math.max(Style.space(48), btnText.implicitWidth + ThemeStyle.paddingFor("control").x * 2)
  implicitHeight: Math.max(Style.space(26), btnText.implicitHeight + ThemeStyle.paddingFor("control").y * 2)

  readonly property bool hot: enabled && mouseArea.containsMouse
  readonly property bool pressed: enabled && mouseArea.pressed

  readonly property var btnToken: ThemeStyle.buttonStyle(
    { foreground: root.foreground, accent: root.accent, urgent: Color.urgent, cornerRadius: Style.cornerRadius },
    root.hot,
    root.active,
    root.destructive,
    root.pressed
  )

  Rectangle {
    id: bg
    anchors.fill: parent
    radius: root.btnToken.radius
    color: root.btnToken.background
    border.color: root.btnToken.border
    border.width: root.btnToken.borderWidth
    opacity: root.enabled ? 1.0 : 0.45

    Behavior on color { ColorAnimation { duration: 120 } }
    Behavior on border.color { ColorAnimation { duration: 120 } }

    Text {
      id: btnText
      anchors.centerIn: parent
      textFormat: Text.PlainText
      text: root.text
      color: root.btnToken.foreground
      font.family: Style.font.family
      font.pixelSize: ThemeStyle.fontSize("bodySmall")
      font.bold: root.active
      renderType: Text.NativeRendering
    }
  }

  MouseArea {
    id: mouseArea
    anchors.fill: parent
    enabled: root.enabled
    hoverEnabled: true
    cursorShape: root.enabled ? Qt.PointingHandCursor : Qt.ArrowCursor
    onClicked: root.clicked()
  }

  StyledToolTip {
    visible: root.tooltipText !== "" && root.hot
    text: root.tooltipText
  }
}
