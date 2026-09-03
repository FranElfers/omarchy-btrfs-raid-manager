.pragma library

/**
 * ThemeStyle.js - Centralized Functional Styling Utilities for Omarchy Btrfs RAID Manager
 *
 * Single source of truth for visual attributes, shape consistency,
 * interaction state evaluation, typography, and theme token composition.
 */

// 1. Theme and Palette Resolution
function resolveTheme(theme) {
  var t = theme || {}
  var fg = t.foreground || (typeof Color !== "undefined" ? Color.foreground : "#cacccc")
  var bg = t.background || (typeof Color !== "undefined" ? Color.background : "#101315")
  var acc = t.accent || (typeof Color !== "undefined" ? Color.accent : fg)
  var urg = t.urgent || (typeof Color !== "undefined" ? Color.urgent : "#a55555")
  var mut = t.muted || (typeof Color !== "undefined" ? Color.muted : "#707880")
  var warn = t.warning || (typeof Color !== "undefined" && Color.warning ? Color.warning : "#f5a97f")
  var round = (t.cornerRadius !== undefined && t.cornerRadius !== null)
    ? t.cornerRadius
    : ((typeof Style !== "undefined" && Style.cornerRadius !== undefined) ? Style.cornerRadius : 0)

  return {
    foreground: fg,
    background: bg,
    accent: acc,
    urgent: urg,
    muted: mut,
    warning: warn,
    cornerRadius: round
  }
}

// 2. Color Alpha Manipulation
function colorWithAlpha(color, alpha) {
  if (!color || color === "transparent") return Qt.rgba(0, 0, 0, 0)
  var qc = Qt.color(color)
  var a = Math.max(0, Math.min(1, Number(alpha) || 0))
  return Qt.rgba(qc.r, qc.g, qc.b, a)
}

// 3. Shape Consistency & Corner Radii Scale
function radiusFor(elementType, baseRadius) {
  var base = (baseRadius !== undefined && baseRadius !== null)
    ? (typeof baseRadius === "object" ? resolveTheme(baseRadius).cornerRadius : Number(baseRadius))
    : ((typeof Style !== "undefined" && Style.cornerRadius !== undefined) ? Style.cornerRadius : 0)

  if (!isFinite(base) || base <= 0) return 0

  switch (elementType) {
    case "card":
    case "dialog":
    case "row":
      return Math.min(base, 8)
    case "button":
    case "input":
    case "textfield":
      return Math.min(base, 6)
    case "badge":
    case "tag":
    case "tooltip":
      return Math.min(base, 4)
    case "gauge":
    case "track":
    case "progress":
      return Math.min(base, 3)
    case "pill":
      return Math.max(base, 12)
    default:
      return base
  }
}

// 4. Typography Scale
function fontSize(token) {
  var baseSize = (typeof Style !== "undefined" && Style.font && Style.font.baseSize) ? Style.font.baseSize : 12
  var caption = (typeof Style !== "undefined" && Style.font && Style.font.caption) ? Style.font.caption : Math.round(baseSize * 0.833)

  switch (token) {
    case "captionSmall":
      return Math.max(1, Math.round(caption * 0.85))
    case "caption":
      return caption
    case "bodySmall":
      return (typeof Style !== "undefined" && Style.font && Style.font.bodySmall) ? Style.font.bodySmall : Math.round(baseSize * 0.917)
    case "body":
      return (typeof Style !== "undefined" && Style.font && Style.font.body) ? Style.font.body : baseSize
    case "subtitle":
      return (typeof Style !== "undefined" && Style.font && Style.font.subtitle) ? Style.font.subtitle : Math.round(baseSize * 1.083)
    case "title":
      return (typeof Style !== "undefined" && Style.font && Style.font.title) ? Style.font.title : Math.round(baseSize * 1.167)
    case "heading":
      return (typeof Style !== "undefined" && Style.font && Style.font.heading) ? Style.font.heading : Math.round(baseSize * 1.333)
    case "display":
      return (typeof Style !== "undefined" && Style.font && Style.font.display) ? Style.font.display : Math.round(baseSize * 2.0)
    default:
      return baseSize
  }
}

// 5. Text Hierarchy Colors
function textSecondary(theme) {
  var t = resolveTheme(theme)
  return colorWithAlpha(t.foreground, 0.70)
}

function textMuted(theme) {
  var t = resolveTheme(theme)
  return colorWithAlpha(t.foreground, 0.48)
}

function warningColor(theme) {
  var t = resolveTheme(theme)
  return t.warning
}

// 6. Interactive Fill & Border Evaluation
function interactiveBackground(theme, isHovered, isActive, isUrgent) {
  var t = resolveTheme(theme)
  if (isUrgent) {
    if (isActive) return colorWithAlpha(t.urgent, 0.22)
    if (isHovered) return colorWithAlpha(t.urgent, 0.14)
    return colorWithAlpha(t.urgent, 0.08)
  }
  if (isActive) {
    return colorWithAlpha(t.accent, 0.18)
  }
  if (isHovered) {
    return colorWithAlpha(t.foreground, 0.08)
  }
  return "transparent"
}

function interactiveBorder(theme, isHovered, isActive, isUrgent) {
  var t = resolveTheme(theme)
  if (isUrgent) return t.urgent
  if (isActive) return t.accent
  if (isHovered) return colorWithAlpha(t.accent, 0.6)
  return t.muted || colorWithAlpha(t.foreground, 0.2)
}

// 7. Component Style Objects
function cardStyle(theme, isHovered, isActive, isUrgent) {
  var t = resolveTheme(theme)
  var bg = interactiveBackground(theme, isHovered, isActive, isUrgent)
  if (bg === "transparent") {
    bg = colorWithAlpha(t.foreground, 0.05)
  }
  var bc = interactiveBorder(theme, isHovered, isActive, isUrgent)
  return {
    background: bg,
    border: bc,
    borderWidth: 1,
    radius: radiusFor("card", t.cornerRadius)
  }
}

function buttonStyle(theme, isHovered, isActive, isUrgent, isPressed) {
  var t = resolveTheme(theme)
  var bg = "transparent"
  if (isPressed) {
    bg = colorWithAlpha(isUrgent ? t.urgent : t.accent, 0.25)
  } else if (isActive) {
    bg = colorWithAlpha(isUrgent ? t.urgent : t.accent, 0.18)
  } else if (isHovered) {
    bg = colorWithAlpha(isUrgent ? t.urgent : t.foreground, 0.10)
  }

  var fg = isUrgent ? t.urgent : (isActive ? t.accent : t.foreground)
  var bc = isUrgent ? colorWithAlpha(t.urgent, isHovered ? 1.0 : 0.6)
                    : (isActive ? t.accent : (isHovered ? colorWithAlpha(t.accent, 0.5) : "transparent"))

  return {
    background: bg,
    border: bc,
    borderWidth: 1,
    foreground: fg,
    radius: radiusFor("button", t.cornerRadius)
  }
}

function badgeStyle(theme, status) {
  var t = resolveTheme(theme)
  var s = String(status || "").toLowerCase()
  var col = t.muted
  var bgAlpha = 0.12

  if (s === "missing" || s === "failing" || s === "urgent" || s === "degraded" || s === "error") {
    col = t.urgent
    bgAlpha = 0.16
  } else if (s === "warning" || s === "warn") {
    col = t.warning
    bgAlpha = 0.16
  } else if (s === "passed" || s === "ok" || s === "healthy") {
    col = t.accent
    bgAlpha = 0.14
  } else if (s === "working" || s === "active") {
    col = t.accent
    bgAlpha = 0.20
  }

  return {
    background: colorWithAlpha(col, bgAlpha),
    border: col,
    borderWidth: 1,
    text: col,
    radius: radiusFor("badge", t.cornerRadius)
  }
}

function tooltipStyle(theme) {
  var t = resolveTheme(theme)
  var bg = (typeof Color !== "undefined" && Color.tooltip && Color.tooltip.background)
    ? Color.tooltip.background
    : colorWithAlpha(t.background, 0.96)
  var text = (typeof Color !== "undefined" && Color.tooltip && Color.tooltip.text)
    ? Color.tooltip.text
    : t.foreground
  var border = (typeof Color !== "undefined" && Color.tooltip && Color.tooltip.border)
    ? Color.tooltip.border
    : colorWithAlpha(t.foreground, 0.3)

  return {
    background: bg,
    text: text,
    border: border,
    borderWidth: 1,
    radius: radiusFor("tooltip", t.cornerRadius),
    fontSize: fontSize("bodySmall")
  }
}

function progressGaugeStyle(theme, percent, isUrgent) {
  var t = resolveTheme(theme)
  var pct = Number(percent) || 0
  var fill = t.accent
  if (isUrgent || pct > 90) {
    fill = t.urgent
  } else if (pct > 75) {
    fill = t.warning
  }

  return {
    trackColor: colorWithAlpha(t.foreground, 0.1),
    fillColor: fill,
    radius: radiusFor("gauge", t.cornerRadius)
  }
}

// 8. Dimensions and Sizing
function iconSize(token) {
  if (typeof Style !== "undefined" && Style.space) {
    if (token === "small") return Style.space(14)
    if (token === "bar") return 16
    if (token === "row") return Style.space(24)
    if (token === "header") return Style.space(26)
    if (token === "large") return Style.space(32)
  }
  var map = {
    small: 14,
    bar: 16,
    row: 24,
    header: 26,
    large: 32
  }
  return map[token] || 16
}

function paddingFor(role) {
  if (typeof Style !== "undefined") {
    if (role === "control") {
      return {
        x: (Style.spacing && Style.spacing.controlPaddingX) || Style.space(10),
        y: (Style.spacing && Style.spacing.controlPaddingY) || Style.space(6)
      }
    }
    if (role === "card") {
      return {
        x: Style.space(12),
        y: Style.space(8)
      }
    }
    if (role === "badge") {
      return {
        x: Style.space(6),
        y: Style.space(3)
      }
    }
    if (role === "flyout") {
      return Style.space(12)
    }
  }
  var fallback = {
    control: { x: 10, y: 6 },
    card: { x: 12, y: 8 },
    badge: { x: 6, y: 3 },
    flyout: 12
  }
  return fallback[role] || { x: 8, y: 6 }
}
