export const APPEARANCE_CHANGED = 'xp-appearance-changed'

export function cssVar(name: string, fallback = ''): string {
  if (typeof document === 'undefined') return fallback
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim() || fallback
}

export function colorAlpha(color: string, alpha: number): string {
  const value = color.trim()
  const hex = value.match(/^#([0-9a-f]{3}|[0-9a-f]{6})$/i)
  if (hex) {
    let h = hex[1]
    if (h.length === 3) h = h.split('').map((c) => c + c).join('')
    const r = Number.parseInt(h.slice(0, 2), 16)
    const g = Number.parseInt(h.slice(2, 4), 16)
    const b = Number.parseInt(h.slice(4, 6), 16)
    return `rgba(${r}, ${g}, ${b}, ${alpha})`
  }
  const rgb = value.match(/^rgba?\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)/i)
  if (rgb) return `rgba(${rgb[1]}, ${rgb[2]}, ${rgb[3]}, ${alpha})`
  return value
}

export function chartTokens() {
  const accent = cssVar('--xp-accent', '#7AA2FF')
  const success = cssVar('--xp-success', '#22c55e')
  const warning = cssVar('--xp-warning', '#f59e0b')
  const danger = cssVar('--xp-danger', '#ef4444')
  const info = cssVar('--xp-info', '#3b82f6')
  const secondary = cssVar('--xp-accent-secondary', '#A4B8E8')
  const series = [
    cssVar('--xp-chart-0', accent),
    cssVar('--xp-chart-1', success),
    cssVar('--xp-chart-2', warning),
    cssVar('--xp-chart-3', danger),
    cssVar('--xp-chart-4', secondary),
    cssVar('--xp-chart-5', info),
  ]
  return {
    accent,
    success,
    warning,
    danger,
    info,
    secondary,
    text: cssVar('--xp-text-secondary', '#A4ADBC'),
    muted: cssVar('--xp-text-muted', '#7B8494'),
    primaryText: cssVar('--xp-text-primary', '#EEF0F4'),
    border: cssVar('--xp-border', '#303641'),
    borderLight: cssVar('--xp-border-light', 'rgba(255,255,255,0.06)'),
    tooltipBg: cssVar('--xp-bg-overlay', '#222630'),
    tooltipText: cssVar('--xp-text-primary', '#EEF0F4'),
    track: cssVar('--xp-progress-trail', 'rgba(148,163,184,0.16)'),
    up: cssVar('--xp-color-up', accent),
    down: cssVar('--xp-color-down', secondary),
    series,
    status: {
      '2xx': success,
      '3xx': info,
      '4xx': warning,
      '5xx': danger,
    } as Record<string, string>,
  }
}

export function emitAppearanceChanged() {
  if (typeof window === 'undefined') return
  window.dispatchEvent(new Event(APPEARANCE_CHANGED))
}

export function onAppearanceChange(cb: () => void): () => void {
  if (typeof window === 'undefined') return () => undefined
  window.addEventListener(APPEARANCE_CHANGED, cb)
  return () => window.removeEventListener(APPEARANCE_CHANGED, cb)
}
