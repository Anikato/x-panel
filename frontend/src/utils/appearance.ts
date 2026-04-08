/**
 * Appearance system — background presets, font, density, radius, card style, sidebar width
 * All settings are applied by setting CSS custom properties on document.documentElement.
 */

import type { BgPreset, UiFont, UiDensity, BorderRadiusPreset, CardBorderStyle, SidebarWidthPreset } from '@/store/modules/global'

// ─── Background Presets ───────────────────────────────────────────

export interface BgPresetDef {
  key: BgPreset
  name: string
  preview: string // small gradient for swatch
  vars: Record<string, string>
}

export const BG_PRESETS: BgPresetDef[] = [
  {
    key: 'abyss',
    name: '深渊 Abyss',
    preview: 'linear-gradient(160deg, #080a10, #0e1420)',
    vars: {
      '--xp-bg-base': '#080a10',
      '--xp-bg-surface': '#141c2b',
      '--xp-bg-card': '#141c2b',
      '--xp-bg-elevated': '#1c2638',
      '--xp-bg-overlay': '#1e293b',
      '--xp-bg-sidebar': '#060810',
      '--xp-bg-header': 'rgba(8, 10, 16, 0.85)',
      '--xp-bg-input': '#0f172a',
      '--xp-bg-table-header': '#0f172a',
      '--xp-bg-inset': '#0f172a',
      '--xp-bg-main-gradient': 'linear-gradient(160deg, #080a10 0%, #0c1018 50%, #0e1420 100%)',
    },
  },
  {
    key: 'void',
    name: '纯黑 Void',
    preview: 'linear-gradient(160deg, #000000, #0a0a0a)',
    vars: {
      '--xp-bg-base': '#050508',
      '--xp-bg-surface': '#111115',
      '--xp-bg-card': '#111115',
      '--xp-bg-elevated': '#1a1a1e',
      '--xp-bg-overlay': '#222226',
      '--xp-bg-sidebar': '#020204',
      '--xp-bg-header': 'rgba(5, 5, 8, 0.85)',
      '--xp-bg-input': '#0a0a0e',
      '--xp-bg-table-header': '#0a0a0e',
      '--xp-bg-inset': '#0a0a0e',
      '--xp-bg-main-gradient': '#050508',
    },
  },
  {
    key: 'tinted',
    name: '微染 Tinted',
    preview: 'linear-gradient(160deg, #080a10, #0a1510)',
    vars: {
      '--xp-bg-base': '#080a10',
      '--xp-bg-surface': '#121c28',
      '--xp-bg-card': '#121c28',
      '--xp-bg-elevated': '#1a2634',
      '--xp-bg-overlay': '#1e2e3b',
      '--xp-bg-sidebar': '#060810',
      '--xp-bg-header': 'rgba(8, 10, 16, 0.85)',
      '--xp-bg-input': '#0d1620',
      '--xp-bg-table-header': '#0d1620',
      '--xp-bg-inset': '#0d1620',
      '--xp-bg-main-gradient': 'var(--xp-bg-tinted-gradient)',
    },
  },
  {
    key: 'cosmos',
    name: '星空 Cosmos',
    preview: 'linear-gradient(160deg, #0a0818, #14102a)',
    vars: {
      '--xp-bg-base': '#0a0818',
      '--xp-bg-surface': '#161230',
      '--xp-bg-card': '#161230',
      '--xp-bg-elevated': '#1e1840',
      '--xp-bg-overlay': '#252048',
      '--xp-bg-sidebar': '#06050f',
      '--xp-bg-header': 'rgba(10, 8, 24, 0.85)',
      '--xp-bg-input': '#100e24',
      '--xp-bg-table-header': '#100e24',
      '--xp-bg-inset': '#100e24',
      '--xp-bg-main-gradient': 'linear-gradient(160deg, #0a0818 0%, #0e0c22 50%, #14102a 100%)',
    },
  },
  {
    key: 'warm',
    name: '暖夜 Warm',
    preview: 'linear-gradient(160deg, #100c08, #1a1410)',
    vars: {
      '--xp-bg-base': '#100c08',
      '--xp-bg-surface': '#1c1610',
      '--xp-bg-card': '#1c1610',
      '--xp-bg-elevated': '#262018',
      '--xp-bg-overlay': '#302820',
      '--xp-bg-sidebar': '#0a0806',
      '--xp-bg-header': 'rgba(16, 12, 8, 0.85)',
      '--xp-bg-input': '#14100c',
      '--xp-bg-table-header': '#14100c',
      '--xp-bg-inset': '#14100c',
      '--xp-bg-main-gradient': 'linear-gradient(160deg, #100c08 0%, #14100c 50%, #1a1410 100%)',
    },
  },
]

// ─── Font Presets ─────────────────────────────────────────────────

export interface FontPresetDef {
  key: UiFont
  name: string
  family: string
  cdnUrl?: string
}

export const FONT_PRESETS: FontPresetDef[] = [
  { key: 'system', name: '系统默认', family: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Noto Sans SC', sans-serif" },
  { key: 'inter', name: 'Inter', family: "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif" },
  { key: 'noto', name: 'Noto Sans SC', family: "'Noto Sans SC', -apple-system, BlinkMacSystemFont, sans-serif" },
  {
    key: 'lxgw',
    name: 'LXGW WenKai',
    family: "'LXGW WenKai', -apple-system, BlinkMacSystemFont, sans-serif",
    cdnUrl: 'https://cdn.jsdelivr.net/npm/lxgw-wenkai-webfont@1.7.0/style.css',
  },
]

// ─── Density Presets ──────────────────────────────────────────────

export const DENSITY_MAP: Record<UiDensity, { fontSize: string; spacing: string; formMargin: string }> = {
  compact: { fontSize: '13px', spacing: '14px', formMargin: '16px' },
  default: { fontSize: '14px', spacing: '20px', formMargin: '20px' },
  comfortable: { fontSize: '15px', spacing: '24px', formMargin: '24px' },
}

// ─── Border Radius Presets ────────────────────────────────────────

export const RADIUS_MAP: Record<BorderRadiusPreset, { base: string; sm: string; lg: string }> = {
  sharp: { base: '4px', sm: '2px', lg: '6px' },
  default: { base: '10px', sm: '6px', lg: '16px' },
  rounded: { base: '16px', sm: '10px', lg: '24px' },
}

// ─── Card Border Style ────────────────────────────────────────────

export const CARD_BORDER_STYLES: { key: CardBorderStyle; name: string }[] = [
  { key: 'accent-left', name: '左侧强调线' },
  { key: 'full', name: '完整边框' },
  { key: 'shadow-only', name: '仅阴影' },
]

// ─── Sidebar Width ────────────────────────────────────────────────

export const SIDEBAR_WIDTH_MAP: Record<SidebarWidthPreset, string> = {
  narrow: '180px',
  default: '220px',
  wide: '260px',
}

// ─── CDN Font Loader ──────────────────────────────────────────────

const loadedFonts = new Set<string>()

function loadCdnFont(url: string) {
  if (loadedFonts.has(url)) return
  loadedFonts.add(url)
  const link = document.createElement('link')
  link.rel = 'stylesheet'
  link.href = url
  document.head.appendChild(link)
}

// ─── Main Apply Function ──────────────────────────────────────────

export interface AppearanceState {
  bgPreset: BgPreset
  uiFont: UiFont
  uiDensity: UiDensity
  borderRadiusPreset: BorderRadiusPreset
  reduceMotion: boolean
  cardBorderStyle: CardBorderStyle
  sidebarWidth: SidebarWidthPreset
  accentKey?: string
  accentCustom?: string
}

export function applyAppearance(state: AppearanceState): void {
  const root = document.documentElement

  // Background preset
  const bg = BG_PRESETS.find(p => p.key === state.bgPreset) || BG_PRESETS[0]
  for (const [prop, val] of Object.entries(bg.vars)) {
    if (prop === '--xp-bg-main-gradient' && state.bgPreset === 'tinted') {
      // For tinted, we dynamically generate gradient with accent color
      const accentRgb = getComputedStyle(root).getPropertyValue('--xp-accent-rgb').trim() || '65, 251, 68'
      root.style.setProperty('--xp-bg-main-gradient',
        `linear-gradient(160deg, #080a10 0%, rgba(${accentRgb}, 0.015) 40%, #0e1420 100%)`)
    } else {
      root.style.setProperty(prop, val)
    }
  }

  // Font
  const font = FONT_PRESETS.find(f => f.key === state.uiFont) || FONT_PRESETS[0]
  if (font.cdnUrl) loadCdnFont(font.cdnUrl)
  root.style.setProperty('--xp-font-family', font.family)

  // Density
  const density = DENSITY_MAP[state.uiDensity] || DENSITY_MAP.default
  root.style.setProperty('--xp-font-size', density.fontSize)
  root.style.setProperty('--xp-spacing', density.spacing)
  root.style.setProperty('--xp-form-margin', density.formMargin)

  // Border Radius
  const radius = RADIUS_MAP[state.borderRadiusPreset] || RADIUS_MAP.default
  root.style.setProperty('--xp-radius', radius.base)
  root.style.setProperty('--xp-radius-sm', radius.sm)
  root.style.setProperty('--xp-radius-lg', radius.lg)

  // Reduce Motion
  root.classList.toggle('reduce-motion', state.reduceMotion)

  // Card Border Style
  root.dataset.cardBorder = state.cardBorderStyle

  // Sidebar Width
  const sw = SIDEBAR_WIDTH_MAP[state.sidebarWidth] || SIDEBAR_WIDTH_MAP.default
  root.style.setProperty('--xp-sidebar-width', sw)
}

export function getBgPresetByKey(key: string): BgPresetDef | undefined {
  return BG_PRESETS.find(p => p.key === key)
}
