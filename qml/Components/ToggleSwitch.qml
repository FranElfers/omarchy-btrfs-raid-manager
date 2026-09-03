import QtQuick
import qs.Commons
import "../ThemeStyle.js" as ThemeStyle

Item {
  id: root

  property bool checked: false
  property bool busy: false
  property color foreground: Color.foreground
  property color accent: Color.accent
  property string tooltipText: ""

  signal toggled()
  signal hovered(bool isHovered)

  readonly property bool containsMouse: mouseArea.containsMouse
  readonly property bool hot: mouseArea.containsMouse

  property int trackHeight: Math.max(22, Math.round(Style.spacing.controlHeight * 0.55))
  property int trackWidth: Math.round(trackHeight * 1.9)
  property int knobSize: Math.max(6, Math.round(trackHeight * 0.72))
  property int knobInset: Math.max(1, Math.round((trackHeight - knobSize) / 2))
  property int cursorPad: Style.space(5)

  implicitWidth: trackWidth + cursorPad * 2
  implicitHeight: trackHeight + cursorPad * 2

  readonly property var toggleToken: ThemeStyle.toggleStyle(
    { foreground: root.foreground, accent: root.accent, cornerRadius: Style.cornerRadius },
    root.hot,
    root.checked
  )

  // 1. Outer hover ring (standardized radius matching ActionButton)
  Rectangle {
    anchors.fill: parent
    visible: root.hot
    color: "transparent"
    radius: root.toggleToken.radius
    border.color: root.toggleToken.ringBorder
    border.width: 1

    Behavior on border.color { ColorAnimation { duration: 120 } }
  }

  // 2. Inner track
  Rectangle {
    id: track
    width: root.trackWidth
    height: root.trackHeight
    anchors.centerIn: parent
    radius: root.toggleToken.trackRadius > 0 ? height / 2 : 0
    color: root.toggleToken.trackBackground
    border.color: root.toggleToken.trackBorder
    border.width: 1

    Behavior on color { ColorAnimation { duration: 120 } }
    Behavior on border.color { ColorAnimation { duration: 120 } }

    // 3. Sliding knob
    Rectangle {
      id: knob
      width: root.knobSize
      height: root.knobSize
      radius: root.toggleToken.trackRadius > 0 ? height / 2 : 0
      x: root.checked ? track.width - width - root.knobInset : root.knobInset
      anchors.verticalCenter: parent.verticalCenter
      color: root.toggleToken.knobColor

      Behavior on x { NumberAnimation { duration: 120; easing.type: Easing.OutCubic } }
      Behavior on color { ColorAnimation { duration: 120 } }
    }
  }

  MouseArea {
    id: mouseArea
    anchors.fill: parent
    hoverEnabled: true
    cursorShape: Qt.PointingHandCursor
    onClicked: {
      if (!root.busy) {
        root.toggled()
      }
    }
    onContainsMouseChanged: root.hovered(containsMouse)
  }

  StyledToolTip {
    visible: root.tooltipText !== "" && root.hot
    text: root.tooltipText
  }
}
