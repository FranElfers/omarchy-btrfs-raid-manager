.pragma library

// Formats byte counts into human readable decimal or binary units.
function formatBytes(bytes, decimals) {
  if (!bytes || bytes === 0) return "0 B"
  if (decimals === undefined) decimals = 1

  var k = 1024
  var sizes = ["B", "KiB", "MiB", "GiB", "TiB", "PiB"]
  var i = Math.floor(Math.log(bytes) / Math.log(k))
  if (i < 0) i = 0
  if (i >= sizes.length) i = sizes.length - 1

  var val = bytes / Math.pow(k, i)
  return val.toFixed(decimals) + " " + sizes[i]
}

// Formats a decimal percentage to 1 decimal place.
function formatPercent(pct) {
  if (pct === undefined || isNaN(pct)) return "0%"
  return pct.toFixed(1) + "%"
}
