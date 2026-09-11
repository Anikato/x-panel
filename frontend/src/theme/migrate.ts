import { ACCENT_PRESETS } from '../utils/accent-colors.ts'
import { hasTheme } from './catalog.ts'
import { clonePreference, emptyPreference } from './resolve.ts'
import { sanitizeWallpaperUrl } from './wallpaper-store.ts'
import {
  APPEARANCE_SCHEMA_VERSION,
  CARD_VARIANTS,
  DENSITIES,
  DEFAULT_PREFERENCE,
  HEADER_HEIGHTS,
  ICON_CONTAINERS,
  ICON_SETS,
  RADIUS_PRESETS,
  SIDEBAR_VARIANTS,
  SIDEBAR_WIDTHS,
  SUBNAV_VARIANTS,
  coerceChromeTexture,
  SURFACE_PRESETS,
  TERM_WALLPAPERS,
  WALLPAPER_IMAGE_MODES,
  THEME_MODES,
  UI_FONTS,
  type AppearancePreference,
  type CardVariant,
  type Density,
  type HeaderHeight,
  type IconContainer,
  type IconSet,
  type LegacyAppearance,
  type RadiusPreset,
  type SidebarVariant,
  type SidebarWidth,
  type SubnavVariant,
  type ChromeTexture,
  type SurfacePreset,
  type TermWallpaper,
  type WallpaperImageMode,
  type ThemeId,
  type ThemeMode,
  type ThemeOverrides,
  type UiFont,
} from './types.ts'

const ACCENT_KEYS = new Set(['custom', ...ACCENT_PRESETS.map((item) => item.key)])
const TERM_THEMES = new Set(['default', 'dracula', 'onedark', 'solarized', 'monokai', 'paper', 'lumen-dark', 'lumen-light'])
const TERM_FONTS = new Set(['jetbrains', 'firacode', 'cascadia', 'consolas', 'system'])

function includes<T extends string>(list: readonly T[], value: unknown): value is T {
  return typeof value === 'string' && (list as readonly string[]).includes(value)
}

function parseObject(input: unknown): Record<string, unknown> | null {
  if (!input) return null
  if (typeof input === 'string') {
    const trimmed = input.trim()
    if (!trimmed || trimmed === '{}') return null
    try {
      const parsed = JSON.parse(trimmed)
      return parsed && typeof parsed === 'object' ? parsed as Record<string, unknown> : null
    } catch {
      return null
    }
  }
  if (typeof input === 'object') return input as Record<string, unknown>
  return null
}

function sanitizeOverrides(raw: unknown): ThemeOverrides {
  const src = raw && typeof raw === 'object' ? raw as Record<string, unknown> : {}
  const next: ThemeOverrides = {}
  if (typeof src.accentKey === 'string' && ACCENT_KEYS.has(src.accentKey)) next.accentKey = src.accentKey
  if (typeof src.accentCustom === 'string') next.accentCustom = src.accentCustom
  if (includes(DENSITIES, src.density)) next.density = src.density as Density
  if (includes(UI_FONTS, src.uiFont)) next.uiFont = src.uiFont as UiFont
  if (includes(SIDEBAR_WIDTHS, src.sidebarWidth)) next.sidebarWidth = src.sidebarWidth as SidebarWidth
  if (includes(HEADER_HEIGHTS, src.headerHeight)) next.headerHeight = src.headerHeight as HeaderHeight
  if (includes(RADIUS_PRESETS, src.radius)) next.radius = src.radius as RadiusPreset
  if (includes(CARD_VARIANTS, src.card)) next.card = src.card as CardVariant
  if (includes(SIDEBAR_VARIANTS, src.sidebarVariant)) next.sidebarVariant = src.sidebarVariant as SidebarVariant
  if (includes(SUBNAV_VARIANTS, src.subnav)) next.subnav = src.subnav as SubnavVariant
  if (includes(ICON_SETS, src.iconSet)) next.iconSet = src.iconSet as IconSet
  if (includes(ICON_CONTAINERS, src.iconContainer)) next.iconContainer = src.iconContainer as IconContainer
  if (typeof src.transparency === 'boolean') next.transparency = src.transparency
  if (includes(SURFACE_PRESETS, src.surfacePreset)) next.surfacePreset = src.surfacePreset as SurfacePreset
  if (typeof src.termTheme === 'string' && TERM_THEMES.has(src.termTheme)) next.termTheme = src.termTheme
  if (typeof src.termFont === 'string' && TERM_FONTS.has(src.termFont)) next.termFont = src.termFont
  if (typeof src.termFontSize === 'number' && src.termFontSize >= 10 && src.termFontSize <= 24) next.termFontSize = src.termFontSize
  if (typeof src.termBgOpacity === 'number' && src.termBgOpacity >= 0.3 && src.termBgOpacity <= 1) next.termBgOpacity = src.termBgOpacity
  if (includes(TERM_WALLPAPERS, src.termWallpaper)) next.termWallpaper = src.termWallpaper as TermWallpaper
  if (typeof src.chromeTexture === 'string') {
    const texture = coerceChromeTexture(src.chromeTexture)
    if (texture !== 'none' || src.chromeTexture === 'none') next.chromeTexture = texture
  }
  if (includes(WALLPAPER_IMAGE_MODES, src.chromeImageMode)) next.chromeImageMode = src.chromeImageMode as WallpaperImageMode
  if (typeof src.chromeImageUrl === 'string') {
    const url = sanitizeWallpaperUrl(src.chromeImageUrl)
    if (url) next.chromeImageUrl = url
  }
  if (typeof src.termFollowChrome === 'boolean') next.termFollowChrome = src.termFollowChrome
  if (includes(WALLPAPER_IMAGE_MODES, src.termImageMode)) next.termImageMode = src.termImageMode as WallpaperImageMode
  if (typeof src.termImageUrl === 'string') {
    const url = sanitizeWallpaperUrl(src.termImageUrl)
    if (url) next.termImageUrl = url
  }
  return next
}

function mapLegacyCard(style: unknown): CardVariant | undefined {
  if (style === 'shadow-only') return 'raised'
  if (style === 'full') return 'outline'
  if (style === 'accent-left') return 'outline'
  return undefined
}

export function migrateLegacyAppearance(input: unknown): AppearancePreference {
  const raw = parseObject(input) || (input && typeof input === 'object' ? input as Record<string, unknown> : {})
  const overrides: ThemeOverrides = {}
  if (typeof raw.accentKey === 'string' && ACCENT_KEYS.has(raw.accentKey)) overrides.accentKey = raw.accentKey
  if (typeof raw.accentCustom === 'string') overrides.accentCustom = raw.accentCustom
  if (includes(UI_FONTS, raw.uiFont)) overrides.uiFont = raw.uiFont as UiFont
  if (includes(DENSITIES, raw.uiDensity)) overrides.density = raw.uiDensity as Density
  if (includes(RADIUS_PRESETS, raw.borderRadiusPreset)) overrides.radius = raw.borderRadiusPreset as RadiusPreset
  const card = mapLegacyCard(raw.cardBorderStyle)
  if (card) overrides.card = card
  if (includes(SIDEBAR_WIDTHS, raw.sidebarWidth)) overrides.sidebarWidth = raw.sidebarWidth as SidebarWidth
  if (includes(SURFACE_PRESETS, raw.bgPreset) && raw.bgPreset !== 'graphite') overrides.surfacePreset = raw.bgPreset as SurfacePreset
  if (typeof raw.termTheme === 'string' && TERM_THEMES.has(raw.termTheme)) overrides.termTheme = raw.termTheme
  if (typeof raw.termFont === 'string' && TERM_FONTS.has(raw.termFont)) overrides.termFont = raw.termFont
  if (typeof raw.termFontSize === 'number' && raw.termFontSize >= 10 && raw.termFontSize <= 24) overrides.termFontSize = raw.termFontSize
  if (typeof raw.termBgOpacity === 'number' && raw.termBgOpacity >= 0.3 && raw.termBgOpacity <= 1) overrides.termBgOpacity = raw.termBgOpacity

  const mode = includes(THEME_MODES, raw.theme) ? raw.theme as ThemeMode : DEFAULT_PREFERENCE.mode
  return {
    schemaVersion: APPEARANCE_SCHEMA_VERSION,
    themeId: 'atelier',
    mode,
    reduceMotion: Boolean(raw.reduceMotion),
    keepPersonalPrefsAcrossThemes: false,
    overridesByTheme: Object.keys(overrides).length ? { atelier: overrides } : {},
  }
}

export function sanitizePreference(input: unknown): AppearancePreference {
  const raw = parseObject(input) || (input && typeof input === 'object' ? input as Record<string, unknown> : {})
  const themeId: ThemeId = hasTheme(String(raw.themeId || '')) ? raw.themeId as ThemeId : 'atelier'
  const mode = includes(THEME_MODES, raw.mode) ? raw.mode as ThemeMode : 'dark'
  const overridesByTheme: AppearancePreference['overridesByTheme'] = {}
  const srcOverrides = raw.overridesByTheme && typeof raw.overridesByTheme === 'object'
    ? raw.overridesByTheme as Record<string, unknown>
    : {}
  for (const key of Object.keys(srcOverrides)) {
    if (!hasTheme(key)) continue
    overridesByTheme[key] = sanitizeOverrides(srcOverrides[key])
  }
  return {
    schemaVersion: APPEARANCE_SCHEMA_VERSION,
    themeId,
    mode,
    reduceMotion: Boolean(raw.reduceMotion),
    keepPersonalPrefsAcrossThemes: Boolean(raw.keepPersonalPrefsAcrossThemes),
    overridesByTheme,
  }
}

function looksLikeV1(raw: Record<string, unknown> | null): boolean {
  return Boolean(raw && typeof raw.schemaVersion === 'number')
}

function looksLikeLegacy(raw: Record<string, unknown> | null): boolean {
  if (!raw) return false
  return Boolean(
    raw.theme || raw.accentKey || raw.bgPreset || raw.uiFont || raw.uiDensity
    || raw.cardBorderStyle || raw.termTheme || raw.borderRadiusPreset || raw.sidebarWidth,
  )
}

export interface HydrateResult {
  preference: AppearancePreference
  source: 'local-v1' | 'local-legacy' | 'backend' | 'default'
  warnings: string[]
}

export function hydrateAppearance(input: {
  localV1?: unknown
  localLegacy?: unknown
  backendRaw?: unknown
  previous?: AppearancePreference
}): HydrateResult {
  const warnings: string[] = []
  const localV1 = parseObject(input.localV1)
  if (looksLikeV1(localV1)) {
    const version = Number(localV1!.schemaVersion)
    if (version > APPEARANCE_SCHEMA_VERSION) {
      warnings.push(`unsupported appearance schema ${version}`)
      return { preference: clonePreference(input.previous || DEFAULT_PREFERENCE), source: input.previous ? 'local-v1' : 'default', warnings }
    }
    return { preference: sanitizePreference(localV1), source: 'local-v1', warnings }
  }

  const localLegacy = parseObject(input.localLegacy) || (input.localLegacy && typeof input.localLegacy === 'object' ? input.localLegacy as Record<string, unknown> : null)
  if (looksLikeLegacy(localLegacy)) {
    return { preference: migrateLegacyAppearance(localLegacy as LegacyAppearance), source: 'local-legacy', warnings }
  }

  const backend = parseObject(input.backendRaw)
  if (looksLikeV1(backend)) {
    const version = Number(backend!.schemaVersion)
    if (version > APPEARANCE_SCHEMA_VERSION) {
      warnings.push(`unsupported appearance schema ${version}`)
      return { preference: clonePreference(input.previous || DEFAULT_PREFERENCE), source: 'default', warnings }
    }
    return { preference: sanitizePreference(backend), source: 'backend', warnings }
  }
  if (looksLikeLegacy(backend)) {
    return { preference: migrateLegacyAppearance(backend as LegacyAppearance), source: 'backend', warnings }
  }

  return { preference: emptyPreference(), source: 'default', warnings }
}
