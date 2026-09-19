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
    name: '朱红',
    key: 'vermilion',
    primary: '#CB2028',
    hover: '#A81A22',
    muted: 'rgba(203, 32, 40, 0.16)',
    glow: '0 0 18px rgba(203, 32, 40, 0.18)',
    secondary: '#E85A62',
    elPrimaryLevels: ['#E85A62', '#CB2028', '#A81A22', '#8B161C', '#6E1116'],
  },
  {
    name: '钢蓝',
    key: 'steel',
    primary: '#7AA2FF',
    hover: '#5B86E8',
    muted: 'rgba(122, 162, 255, 0.16)',
    glow: '0 0 18px rgba(122, 162, 255, 0.18)',
    secondary: '#A4B8E8',
    elPrimaryLevels: ['#9DB8FF', '#7AA2FF', '#5B86E8', '#355FCC', '#2A4CA3'],
  },
  {
    name: '青蓝',
    key: 'cyan',
    primary: '#22d3ee',
    hover: '#06b6d4',
    muted: 'rgba(34, 211, 238, 0.15)',
    glow: '0 0 20px rgba(34, 211, 238, 0.2)',
    secondary: '#818cf8',
    elPrimaryLevels: ['#38bdf8', '#0ea5e9', '#0284c7', '#0369a1', '#075985'],
  },
  {
    name: '靛蓝',
    key: 'indigo',
    primary: '#818cf8',
    hover: '#6366f1',
    muted: 'rgba(129, 140, 248, 0.15)',
    glow: '0 0 20px rgba(129, 140, 248, 0.2)',
    secondary: '#a78bfa',
    elPrimaryLevels: ['#a5b4fc', '#818cf8', '#6366f1', '#4f46e5', '#4338ca'],
  },
  {
    name: '翡翠',
    key: 'emerald',
    primary: '#34d399',
    hover: '#10b981',
    muted: 'rgba(52, 211, 153, 0.15)',
    glow: '0 0 20px rgba(52, 211, 153, 0.2)',
    secondary: '#60a5fa',
    elPrimaryLevels: ['#6ee7b7', '#34d399', '#10b981', '#059669', '#047857'],
  },
  {
    name: '琥珀',
    key: 'amber',
    primary: '#fbbf24',
    hover: '#f59e0b',
    muted: 'rgba(251, 191, 36, 0.15)',
    glow: '0 0 20px rgba(251, 191, 36, 0.2)',
    secondary: '#fb923c',
    elPrimaryLevels: ['#fde68a', '#fbbf24', '#f59e0b', '#d97706', '#b45309'],
  },
  {
    name: '玫红',
    key: 'rose',
    primary: '#fb7185',
    hover: '#f43f5e',
    muted: 'rgba(251, 113, 133, 0.15)',
    glow: '0 0 20px rgba(251, 113, 133, 0.2)',
    secondary: '#c084fc',
    elPrimaryLevels: ['#fda4af', '#fb7185', '#f43f5e', '#e11d48', '#be123c'],
  },
  {
    name: '天蓝',
    key: 'blue',
    primary: '#60a5fa',
    hover: '#3b82f6',
    muted: 'rgba(96, 165, 250, 0.15)',
    glow: '0 0 20px rgba(96, 165, 250, 0.2)',
    secondary: '#a78bfa',
    elPrimaryLevels: ['#93c5fd', '#60a5fa', '#3b82f6', '#2563eb', '#1d4ed8'],
  },
  {
    name: '紫罗兰',
    key: 'violet',
    primary: '#a78bfa',
    hover: '#8b5cf6',
    muted: 'rgba(167, 139, 250, 0.15)',
    glow: '0 0 20px rgba(167, 139, 250, 0.2)',
    secondary: '#f472b6',
    elPrimaryLevels: ['#c4b5fd', '#a78bfa', '#8b5cf6', '#7c3aed', '#6d28d9'],
  },
  {
    name: '橙色',
    key: 'orange',
    primary: '#fb923c',
    hover: '#f97316',
    muted: 'rgba(251, 146, 60, 0.15)',
    glow: '0 0 20px rgba(251, 146, 60, 0.2)',
    secondary: '#fbbf24',
    elPrimaryLevels: ['#fdba74', '#fb923c', '#f97316', '#ea580c', '#c2410c'],
  },
  {
    name: '荧光绿',
    key: 'neon',
    primary: '#41FB44',
    hover: '#34C936',
    muted: 'rgba(65, 251, 68, 0.15)',
    glow: '0 0 20px rgba(65, 251, 68, 0.2)',
    secondary: '#22d3ee',
    elPrimaryLevels: ['#7AFC7C', '#5EFD62', '#34C936', '#28A02B', '#1D7A20'],
  },
  {
    name: '松绿',
    key: 'pine',
    primary: '#6FAE86',
    hover: '#5A9872',
    muted: 'rgba(111, 174, 134, 0.16)',
    glow: '0 0 18px rgba(111, 174, 134, 0.16)',
    secondary: '#C4B7A2',
    elPrimaryLevels: ['#8FBE9C', '#6FAE86', '#5A9872', '#3D7A5A', '#2F6B4A'],
  },
  {
    name: '松绿深',
    key: 'pine-deep',
    primary: '#2F6B4A',
    hover: '#24563B',
    muted: 'rgba(47, 107, 74, 0.16)',
    glow: '0 0 16px rgba(47, 107, 74, 0.14)',
    secondary: '#8A7A62',
    elPrimaryLevels: ['#4A8C68', '#2F6B4A', '#24563B', '#1C4430', '#163626'],
  },
  {
    name: '黄铜',
    key: 'brass',
    primary: '#C9A35B',
    hover: '#B08D45',
    muted: 'rgba(201, 163, 91, 0.16)',
    glow: '0 0 18px rgba(201, 163, 91, 0.16)',
    secondary: '#7EB6B4',
    elPrimaryLevels: ['#D4B56E', '#C9A35B', '#B08D45', '#8A6A1A', '#6F5514'],
  },
  {
    name: '潮水',
    key: 'tide',
    primary: '#1F6F6E',
    hover: '#185958',
    muted: 'rgba(31, 111, 110, 0.16)',
    glow: '0 0 16px rgba(31, 111, 110, 0.14)',
    secondary: '#8A6A1A',
    elPrimaryLevels: ['#3D9A96', '#1F6F6E', '#185958', '#134746', '#0F3837'],
  },
  {
    name: '青灰',
    key: 'slate',
    primary: '#8FA6BC',
    hover: '#7A91A8',
    muted: 'rgba(143, 166, 188, 0.16)',
    glow: '0 0 16px rgba(143, 166, 188, 0.14)',
    secondary: '#B7A99A',
    elPrimaryLevels: ['#A4B7C9', '#8FA6BC', '#7A91A8', '#4A5D74', '#3D4E63'],
  },
  {
    name: '青灰深',
    key: 'slate-deep',
    primary: '#3D4E63',
    hover: '#314050',
    muted: 'rgba(61, 78, 99, 0.16)',
    glow: '0 0 14px rgba(61, 78, 99, 0.12)',
    secondary: '#8A7F72',
    elPrimaryLevels: ['#5A6E84', '#3D4E63', '#314050', '#263240', '#1C2530'],
  },
]

function hexToRgb(hex: string): [number, number, number] {
  const h = hex.replace('#', '')
  return [parseInt(h.slice(0, 2), 16), parseInt(h.slice(2, 4), 16), parseInt(h.slice(4, 6), 16)]
}

function rgbToHex(r: number, g: number, b: number): string {
  return '#' + [r, g, b].map(v => v.toString(16).padStart(2, '0')).join('')
}

export function hexLuminance(hex: string): number {
  const h = hex.replace('#', '')
  if (h.length < 6) return 0
  const toLin = (channel: number) => {
    const s = channel / 255
    return s <= 0.04045 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
  }
  const r = toLin(Number.parseInt(h.slice(0, 2), 16))
  const g = toLin(Number.parseInt(h.slice(2, 4), 16))
  const b = toLin(Number.parseInt(h.slice(4, 6), 16))
  return 0.2126 * r + 0.7152 * g + 0.0722 * b
}

export function contrastRatio(a: string, b: string): number {
  const l1 = hexLuminance(a)
  const l2 = hexLuminance(b)
  const [hi, lo] = l1 > l2 ? [l1, l2] : [l2, l1]
  return (hi + 0.05) / (lo + 0.05)
}

export function ensureContrast(fg: string, bg: string, min = 4.5): string {
  if (!fg.startsWith('#') || fg.length < 7 || !bg.startsWith('#') || bg.length < 7) return fg
  if (contrastRatio(fg, bg) >= min) return fg
  const towards = hexLuminance(bg) > 0.45 ? '#0B0E14' : '#F8FAFC'
  for (let i = 1; i <= 24; i++) {
    const mixed = mixColor(fg, towards, 100 - (i / 24) * 100)
    if (contrastRatio(mixed, bg) >= min) return mixed
  }
  return towards
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

  const luminance = (0.299 * r + 0.587 * g + 0.114 * b) / 255
  root.style.setProperty('--xp-accent', palette.primary)
  root.style.setProperty('--xp-on-accent', luminance > 0.62 ? '#0B0E14' : '#F8FAFC')
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
