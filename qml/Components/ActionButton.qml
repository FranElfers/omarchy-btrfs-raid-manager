import QtQuick
import qs.Commons
import qs.Ui as Ui
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

  implicitWidth: btn.implicitWidth
  implicitHeight: btn.implicitHeight

  Ui.Button {
    id: btn
    anchors.fill: parent
    text: root.text
    tooltipText: "" // Delegate to StyledToolTip for consistent theme-styled overlay and radius
    enabled: root.enabled
    active: root.active
    accent: root.destructive ? Color.urgent : root.accent
    foreground: root.destructive ? Color.urgent : root.foreground
    radius: ThemeStyle.radiusFor("button", Style.cornerRadius)
    onClicked: root.clicked()
  }

  StyledToolTip {
    visible: root.tooltipText !== "" && btn.hot
    text: root.tooltipText
  }
}
