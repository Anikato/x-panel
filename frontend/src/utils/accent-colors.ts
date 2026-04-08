/**
 * Accent color system — predefined palettes + dynamic CSS variable injection
 */

export interface AccentPalette {
  name: string
  key: string
  primary: string
  hover: string
  muted: string
  glow: string
  secondary: string
  elPrimaryLevels: string[]
}

export const ACCENT_PRESETS: AccentPalette[] = [
  {
    name: '青蓝 Cyan',
    key: 'cyan',
    primary: '#22d3ee',
    hover: '#06b6d4',
    muted: 'rgba(34, 211, 238, 0.15)',
    glow: '0 0 20px rgba(34, 211, 238, 0.2)',
    secondary: '#818cf8',
    elPrimaryLevels: ['#38bdf8', '#0ea5e9', '#0284c7', '#0369a1', '#075985'],
  },
  {
    name: '靛蓝 Indigo',
    key: 'indigo',
    primary: '#818cf8',
    hover: '#6366f1',
    muted: 'rgba(129, 140, 248, 0.15)',
    glow: '0 0 20px rgba(129, 140, 248, 0.2)',
    secondary: '#a78bfa',
    elPrimaryLevels: ['#a5b4fc', '#818cf8', '#6366f1', '#4f46e5', '#4338ca'],
  },
  {
    name: '翡翠 Emerald',
    key: 'emerald',
    primary: '#34d399',
    hover: '#10b981',
    muted: 'rgba(52, 211, 153, 0.15)',
    glow: '0 0 20px rgba(52, 211, 153, 0.2)',
    secondary: '#60a5fa',
    elPrimaryLevels: ['#6ee7b7', '#34d399', '#10b981', '#059669', '#047857'],
  },
  {
    name: '琥珀 Amber',
    key: 'amber',
    primary: '#fbbf24',
    hover: '#f59e0b',
    muted: 'rgba(251, 191, 36, 0.15)',
    glow: '0 0 20px rgba(251, 191, 36, 0.2)',
    secondary: '#fb923c',
    elPrimaryLevels: ['#fde68a', '#fbbf24', '#f59e0b', '#d97706', '#b45309'],
  },
  {
    name: '玫红 Rose',
    key: 'rose',
    primary: '#fb7185',
    hover: '#f43f5e',
    muted: 'rgba(251, 113, 133, 0.15)',
    glow: '0 0 20px rgba(251, 113, 133, 0.2)',
    secondary: '#c084fc',
    elPrimaryLevels: ['#fda4af', '#fb7185', '#f43f5e', '#e11d48', '#be123c'],
  },
  {
    name: '天蓝 Blue',
    key: 'blue',
    primary: '#60a5fa',
    hover: '#3b82f6',
    muted: 'rgba(96, 165, 250, 0.15)',
    glow: '0 0 20px rgba(96, 165, 250, 0.2)',
    secondary: '#a78bfa',
    elPrimaryLevels: ['#93c5fd', '#60a5fa', '#3b82f6', '#2563eb', '#1d4ed8'],
  },
  {
    name: '紫罗兰 Violet',
    key: 'violet',
    primary: '#a78bfa',
    hover: '#8b5cf6',
    muted: 'rgba(167, 139, 250, 0.15)',
    glow: '0 0 20px rgba(167, 139, 250, 0.2)',
    secondary: '#f472b6',
    elPrimaryLevels: ['#c4b5fd', '#a78bfa', '#8b5cf6', '#7c3aed', '#6d28d9'],
  },
  {
    name: '橙色 Orange',
    key: 'orange',
    primary: '#fb923c',
    hover: '#f97316',
    muted: 'rgba(251, 146, 60, 0.15)',
    glow: '0 0 20px rgba(251, 146, 60, 0.2)',
    secondary: '#fbbf24',
    elPrimaryLevels: ['#fdba74', '#fb923c', '#f97316', '#ea580c', '#c2410c'],
  },
  {
    name: '荧光绿 Neon',
    key: 'neon',
    primary: '#41FB44',
    hover: '#34C936',
    muted: 'rgba(65, 251, 68, 0.15)',
    glow: '0 0 20px rgba(65, 251, 68, 0.2)',
    secondary: '#22d3ee',
    elPrimaryLevels: ['#7AFC7C', '#5EFD62', '#34C936', '#28A02B', '#1D7A20'],
  },
]

function hexToRgb(hex: string): [number, number, number] {
  const h = hex.replace('#', '')
  return [parseInt(h.slice(0, 2), 16), parseInt(h.slice(2, 4), 16), parseInt(h.slice(4, 6), 16)]
}

function rgbToHex(r: number, g: number, b: number): string {
  return '#' + [r, g, b].map(v => v.toString(16).padStart(2, '0')).join('')
}

function mixColor(c1: string, c2: string, weight: number): string {
  const [r1, g1, b1] = hexToRgb(c1)
  const [r2, g2, b2] = hexToRgb(c2)
  const w = weight / 100
  return rgbToHex(
    Math.round(r1 * w + r2 * (1 - w)),
    Math.round(g1 * w + g2 * (1 - w)),
    Math.round(b1 * w + b2 * (1 - w)),
  )
}

export function generatePaletteFromHex(hex: string): AccentPalette {
  const [r, g, b] = hexToRgb(hex)
  return {
    name: '自定义',
    key: 'custom',
    primary: hex,
    hover: mixColor(hex, '#000000', 80),
    muted: `rgba(${r}, ${g}, ${b}, 0.15)`,
    glow: `0 0 20px rgba(${r}, ${g}, ${b}, 0.2)`,
    secondary: mixColor(hex, '#8b5cf6', 40),
    elPrimaryLevels: [
      mixColor(hex, '#ffffff', 70),
      mixColor(hex, '#ffffff', 50),
      mixColor(hex, '#000000', 80),
      mixColor(hex, '#000000', 65),
      mixColor(hex, '#000000', 50),
    ],
  }
}

export function applyAccentPalette(palette: AccentPalette): void {
  const root = document.documentElement
  const [r, g, b] = hexToRgb(palette.primary)

  root.style.setProperty('--xp-accent', palette.primary)
  root.style.setProperty('--xp-accent-rgb', `${r}, ${g}, ${b}`)
  root.style.setProperty('--xp-accent-hover', palette.hover)
  root.style.setProperty('--xp-accent-muted', palette.muted)
  root.style.setProperty('--xp-accent-glow', palette.glow)
  root.style.setProperty('--xp-accent-secondary', palette.secondary)
  root.style.setProperty('--xp-color-up', palette.primary)

  root.style.setProperty('--xp-context-hover', `rgba(${r}, ${g}, ${b}, 0.08)`)

  root.style.setProperty('--xp-btn-primary-bg', palette.hover)
  root.style.setProperty('--xp-btn-primary-hover', palette.primary)
  root.style.setProperty('--xp-btn-primary-active', mixColor(palette.hover, '#000000', 80))
  root.style.setProperty('--xp-btn-primary-gradient', `linear-gradient(135deg, ${palette.hover}, ${palette.primary})`)
  root.style.setProperty('--xp-btn-primary-gradient-hover', `linear-gradient(135deg, ${palette.primary}, ${mixColor(palette.primary, '#ffffff', 80)})`)

  // Element Plus primary color levels
  root.style.setProperty('--el-color-primary', palette.primary)
  root.style.setProperty('--el-color-primary-light-3', palette.elPrimaryLevels[0])
  root.style.setProperty('--el-color-primary-light-5', palette.elPrimaryLevels[1])
  root.style.setProperty('--el-color-primary-light-7', palette.elPrimaryLevels[2])
  root.style.setProperty('--el-color-primary-light-8', palette.elPrimaryLevels[3])
  root.style.setProperty('--el-color-primary-light-9', palette.elPrimaryLevels[4])
  root.style.setProperty('--el-color-primary-dark-2', palette.hover)

  root.style.setProperty('--el-menu-active-color', palette.primary)
  root.style.setProperty('--el-pagination-hover-color', palette.primary)
}

export function getPresetByKey(key: string): AccentPalette | undefined {
  return ACCENT_PRESETS.find(p => p.key === key)
}
