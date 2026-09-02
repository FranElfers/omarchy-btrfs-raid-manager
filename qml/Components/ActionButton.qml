import QtQuick
import QtQuick.Controls
import qs.Commons
import qs.Ui as Ui

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
    enabled: root.enabled
    active: root.active
    accent: root.accent
    foreground: root.destructive ? Color.urgent : root.foreground
    onClicked: root.clicked()

    ToolTip.visible: root.tooltipText !== "" && hovered
    ToolTip.delay: 350
    ToolTip.text: root.tooltipText
  }
}
