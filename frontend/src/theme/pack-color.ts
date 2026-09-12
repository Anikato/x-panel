/** C.2 color math: encoded-sRGB mix with JS Math.round. */

export const BLACK = '#000000'
export const WHITE = '#FFFFFF'

export function mixSrgb(a: string, b: string, t: number): string {
  const A = hexToRgb(a)
  const B = hexToRgb(b)
  return rgbToHex(
    Math.round(A[0] * (1 - t) + B[0] * t),
    Math.round(A[1] * (1 - t) + B[1] * t),
    Math.round(A[2] * (1 - t) + B[2] * t),
  )
}

export function hexToRgb(hex: string): [number, number, number] {
  const h = normalizeHex(hex).slice(1)
  return [parseInt(h.slice(0, 2), 16), parseInt(h.slice(2, 4), 16), parseInt(h.slice(4, 6), 16)]
}

export function rgbToHex(r: number, g: number, b: number): string {
  const clamp = (n: number) => Math.max(0, Math.min(255, n))
  return '#' + [clamp(r), clamp(g), clamp(b)].map((v) => v.toString(16).padStart(2, '0')).join('').toUpperCase()
}

export function normalizeHex(hex: string): string {
  const h = hex.trim()
  if (/^#[0-9a-fA-F]{6}$/.test(h)) return h.toUpperCase()
  throw new Error('hex')
}

export function isHex(raw: unknown): boolean {
  return typeof raw === 'string' && /^#[0-9a-fA-F]{6}$/.test(raw.trim())
}

export function parseRgba(raw: string): { r: number, g: number, b: number, a: number } | null {
  const m = raw.trim().match(/^rgba\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*,\s*([0-9.]+)\s*\)$/i)
  if (!m) return null
  const r = Number(m[1])
  const g = Number(m[2])
  const b = Number(m[3])
  const a = Number(m[4])
  if (![r, g, b].every((n) => Number.isInteger(n) && n >= 0 && n <= 255)) return null
  if (!Number.isFinite(a) || a < 0 || a > 1) return null
  return { r, g, b, a }
}

export function compositeRgbaOnHex(rgba: string, surface: string): string {
  const p = parseRgba(rgba)
  if (!p) return normalizeHex(surface)
  const [sr, sg, sb] = hexToRgb(surface)
  return rgbToHex(
    Math.round(p.r * p.a + sr * (1 - p.a)),
    Math.round(p.g * p.a + sg * (1 - p.a)),
    Math.round(p.b * p.a + sb * (1 - p.a)),
  )
}

export function relativeLuminance(hex: string): number {
  const toLin = (channel: number) => {
    const s = channel / 255
    return s <= 0.04045 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
  }
  const [r, g, b] = hexToRgb(hex)
  return 0.2126 * toLin(r) + 0.7152 * toLin(g) + 0.0722 * toLin(b)
}

export function contrastRatio(a: string, b: string): number {
  const l1 = relativeLuminance(a)
  const l2 = relativeLuminance(b)
  const [hi, lo] = l1 > l2 ? [l1, l2] : [l2, l1]
  return (hi + 0.05) / (lo + 0.05)
}

export function rgbDistance(a: string, b: string): number {
  const A = hexToRgb(a)
  const B = hexToRgb(b)
  return Math.hypot(A[0] - B[0], A[1] - B[1], A[2] - B[2])
}

export function correctAgainst(fg: string, backgrounds: string[], min: number): { color: string, ok: boolean } {
  const works = (c: string) => backgrounds.every((bg) => contrastRatio(c, bg) >= min)
  const start = isHex(fg) ? normalizeHex(fg) : fg
  if (works(start)) return { color: start, ok: true }
  for (let i = 0; i <= 100; i++) {
    const t = i / 100
    const towardBlack = mixSrgb(start, BLACK, t)
    const towardWhite = mixSrgb(start, WHITE, t)
    const blackOk = works(towardBlack)
    const whiteOk = works(towardWhite)
    if (blackOk) return { color: towardBlack, ok: true }
    if (whiteOk) return { color: towardWhite, ok: true }
  }
  return { color: start, ok: false }
}

export function rgbaFromHex(hex: string, alpha: number): string {
  const [r, g, b] = hexToRgb(hex)
  return `rgba(${r}, ${g}, ${b}, ${alpha})`
}

export function monotonicLuminance(colors: string[], increasing: boolean): boolean {
  for (let i = 1; i < colors.length; i++) {
    const d = relativeLuminance(colors[i]) - relativeLuminance(colors[i - 1])
    if (increasing ? d <= 0 : d >= 0) return false
  }
  return true
}
