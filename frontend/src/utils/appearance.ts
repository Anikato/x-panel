/**
 * Compatibility facade around the typed theme engine.
 * New code should import from `@/theme`.
 */

import {
  DENSITY_TOKENS,
  SIDEBAR_WIDTH_TOKENS,
  SURFACE_PRESET_DEFS,
  UI_FONT_STACKS,
  hydrateAppearance,
  resolveAppearance,
} from '../theme/index.ts'
import type { Density, RadiusPreset, SurfacePreset, UiFont } from '../theme/types.ts'
import { RADIUS_TOKENS } from '../theme/shared-tokens.ts'

export type BgPreset = SurfacePreset
export type { UiFont, Density }
export type BorderRadiusPreset = RadiusPreset
export type CardBorderStyle = 'accent-left' | 'full' | 'shadow-only'
export type SidebarWidthPreset = 'narrow' | 'default' | 'wide'

export interface BgPresetDef {
  key: BgPreset
  name: string
  preview: string
  vars: Record<string, string>
}

export const LIGHT_BG_VARS: Record<string, string> = {
  '--xp-bg-base': '#E4E6EC',
  '--xp-bg-surface': '#F4F5F8',
  '--xp-bg-card': '#F4F5F8',
  '--xp-bg-elevated': '#FAFBFC',
  '--xp-bg-overlay': '#FFFFFF',
  '--xp-bg-sidebar': '#D8DCE4',
  '--xp-bg-header': '#ECEEF3',
  '--xp-bg-input': '#F7F8FB',
  '--xp-bg-table-header': '#E8EAEE',
  '--xp-bg-inset': '#E6E8EE',
  '--xp-bg-main-gradient': '#E4E6EC',
}

export const BG_PRESETS: BgPresetDef[] = SURFACE_PRESET_DEFS.map((item) => ({
  key: item.key,
  name: item.name,
  preview: item.preview,
  vars: {
    '--xp-bg-base': item.colors.bgBase,
    '--xp-bg-surface': item.colors.bgSurface,
    '--xp-bg-card': item.colors.bgSurface,
    '--xp-bg-elevated': item.colors.bgElevated,
    '--xp-bg-overlay': item.colors.bgOverlay,
    '--xp-bg-sidebar': item.colors.bgSidebar,
    '--xp-bg-header': item.colors.bgHeader,
    '--xp-bg-input': item.colors.bgInput,
    '--xp-bg-table-header': item.colors.bgTableHeader,
    '--xp-bg-inset': item.colors.bgInset,
    '--xp-bg-main-gradient': item.colors.bgBase,
  },
}))

export interface FontPresetDef {
  key: UiFont
  name: string
  family: string
  cdnUrl?: string
}

export const FONT_PRESETS: FontPresetDef[] = [
  { key: 'system', name: '系统默认', family: UI_FONT_STACKS.system },
  { key: 'inter', name: 'Inter', family: UI_FONT_STACKS.inter },
  { key: 'noto', name: 'Noto Sans SC', family: UI_FONT_STACKS.noto },
  { key: 'lxgw', name: 'LXGW WenKai', family: UI_FONT_STACKS.lxgw, cdnUrl: 'https://cdn.jsdelivr.net/npm/lxgw-wenkai-webfont@1.7.0/style.css' },
]

export const DENSITY_MAP = DENSITY_TOKENS
export const RADIUS_MAP: Record<RadiusPreset, { base: string; sm: string; lg: string }> = {
  sharp: { base: RADIUS_TOKENS.sharp.radiusCard, sm: RADIUS_TOKENS.sharp.radiusControl, lg: RADIUS_TOKENS.sharp.radiusOverlay },
  default: { base: RADIUS_TOKENS.default.radiusCard, sm: RADIUS_TOKENS.default.radiusControl, lg: RADIUS_TOKENS.default.radiusOverlay },
  rounded: { base: RADIUS_TOKENS.rounded.radiusCard, sm: RADIUS_TOKENS.rounded.radiusControl, lg: RADIUS_TOKENS.rounded.radiusOverlay },
}

export const CARD_BORDER_STYLES: { key: CardBorderStyle; name: string }[] = [
  { key: 'accent-left', name: '左侧强调线' },
  { key: 'full', name: '完整边框' },
  { key: 'shadow-only', name: '仅阴影' },
]

export const SIDEBAR_WIDTH_MAP = SIDEBAR_WIDTH_TOKENS

export function shouldReduceMotion(userPref: boolean, systemPref: boolean) {
  return Boolean(userPref || systemPref)
}

export function pickBgVars(preset: string, isDark: boolean): Record<string, string> {
  if (!isDark) return { ...LIGHT_BG_VARS }
  const bg = BG_PRESETS.find((item) => item.key === preset) || BG_PRESETS[0]
  return { ...bg.vars }
}

export interface AppearanceState {
  bgPreset: BgPreset
  uiFont: UiFont
  uiDensity: Density
  borderRadiusPreset: RadiusPreset
  reduceMotion: boolean
  cardBorderStyle: CardBorderStyle
  sidebarWidth: SidebarWidthPreset
  accentKey?: string
  accentCustom?: string
}

export interface SanitizedAppearance extends AppearanceState {
  accentKey: string
}

export function sanitizeAppearance(input: unknown): SanitizedAppearance {
  const migrated = hydrateAppearance({ localLegacy: input }).preference
  const over = migrated.overridesByTheme.atelier || {}
  const card: CardBorderStyle = over.card === 'raised' ? 'shadow-only' : over.card === 'outline' ? 'full' : 'accent-left'
  return {
    bgPreset: over.surfacePreset || 'graphite',
    uiFont: over.uiFont || 'system',
    uiDensity: over.density || 'default',
    borderRadiusPreset: over.radius || 'default',
    reduceMotion: migrated.reduceMotion,
    cardBorderStyle: card,
    sidebarWidth: over.sidebarWidth || 'default',
    accentKey: over.accentKey || 'steel',
    accentCustom: over.accentCustom || '',
  }
}

export function applyAppearance(state: AppearanceState, isDark = true): void {
  const resolved = resolveAppearance({
    schemaVersion: 1,
    themeId: 'atelier',
    mode: isDark ? 'dark' : 'light',
    reduceMotion: state.reduceMotion,
    keepPersonalPrefsAcrossThemes: false,
    overridesByTheme: {
      atelier: {
        surfacePreset: state.bgPreset,
        uiFont: state.uiFont,
        density: state.uiDensity,
        radius: state.borderRadiusPreset,
        sidebarWidth: state.sidebarWidth,
        accentKey: state.accentKey,
        accentCustom: state.accentCustom,
        card: state.cardBorderStyle === 'shadow-only' ? 'raised' : 'outline',
      },
    },
  }, {
    systemDark: isDark,
    systemReduceMotion: typeof window !== 'undefined' && window.matchMedia
      ? window.matchMedia('(prefers-reduced-motion: reduce)').matches
      : false,
    transparencySupported: true,
  })
  if (typeof document === 'undefined') return
  const root = document.documentElement
  for (const [prop, value] of Object.entries(resolved.cssVars)) {
    root.style.setProperty(prop, value)
  }
  root.classList.toggle('reduce-motion', resolved.reduceMotion)
  root.dataset.cardBorder = state.cardBorderStyle
  root.dataset.cardVariant = resolved.variants.card
}

export function getBgPresetByKey(key: string): BgPresetDef | undefined {
  return BG_PRESETS.find((item) => item.key === key)
}
