import QtQuick
import QtQuick.Controls
import qs.Commons
import "../ThemeStyle.js" as ThemeStyle

ToolTip {
  id: root

  property var theme: Color

  delay: 400
  padding: 0

  readonly property var styleToken: ThemeStyle.tooltipStyle(root.theme)

  background: Rectangle {
    color: root.styleToken.background
    radius: root.styleToken.radius
    border.color: root.styleToken.border
    border.width: root.styleToken.borderWidth
  }

  contentItem: Text {
    textFormat: Text.PlainText
    text: root.text
    color: root.styleToken.text
    font.family: Style.font.family
    font.pixelSize: root.styleToken.fontSize
    leftPadding: Style.spacing.controlPaddingX
    rightPadding: Style.spacing.controlPaddingX
    topPadding: Style.spacing.controlPaddingY
    bottomPadding: Style.spacing.controlPaddingY
  }
}
