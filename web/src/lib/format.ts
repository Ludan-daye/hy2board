// Human-readable byte / speed formatting shared across pages.

export function fmtBytes(n: number): string {
  let v = Math.abs(n)
  const units = ["B", "KB", "MB", "GB", "TB", "PB"]
  let i = 0
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(v < 10 && i > 0 ? 1 : 0)}${units[i]}`
}

export function fmtSpeed(bytesPerSec: number): string {
  if (bytesPerSec <= 0) return "0"
  return `${fmtBytes(bytesPerSec)}/s`
}
