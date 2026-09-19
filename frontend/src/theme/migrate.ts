import { ACCENT_PRESETS } from '../utils/accent-colors.ts'
import { isThemeSlug } from './catalog.ts'
import { clonePreference, emptyPreference } from './resolve.ts'
import { sanitizeWallpaperUrl } from './wallpaper-store.ts'
import {
  APPEARANCE_SCHEMA_VERSION,
  CARD_VARIANTS,
  DENSITIES,
  DEFAULT_PREFERENCE,
  HEADER_HEIGHTS,
  ICON_CONTAINERS,
  HEADING_STYLES,
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

const KNOWN_OVERRIDE_KEYS = new Set([
  'accentKey', 'accentCustom', 'accentSecondary', 'density', 'uiFont', 'sidebarWidth', 'headerHeight',
  'radius', 'card', 'sidebarVariant', 'headingStyle', 'subnav', 'iconSet', 'iconContainer', 'transparency',
  'surfacePreset', 'termTheme', 'termFont', 'termFontSize', 'termBgOpacity', 'termWallpaper',
  'chromeTexture', 'chromeImageMode', 'chromeImageUrl', 'termFollowChrome', 'termImageMode', 'termImageUrl',
  'surfaces',
])

function isolate(isolated: Record<string, unknown>, path: string, value: unknown) {
  isolated[path] = value
}

function takeEnum<T extends string>(
  raw: unknown,
  allowed: readonly T[] | Set<string>,
  path: string,
  isolated: Record<string, unknown>,
): T | undefined {
  if (raw === undefined) return undefined
  const ok = allowed instanceof Set ? typeof raw === 'string' && allowed.has(raw) : includes(allowed, raw)
  if (ok) {
    delete isolated[path]
    return raw as T
  }
  isolate(isolated, path, raw)
  return undefined
}

function sanitizeSurfaces(raw: unknown, prefix: string, isolated: Record<string, unknown>): ThemeOverrides['surfaces'] {
  if (!raw || typeof raw !== 'object') {
    if (raw !== undefined) isolate(isolated, prefix, raw)
    return undefined
  }
  const src = raw as Record<string, unknown>
  for (const key of Object.keys(src)) {
    if (key !== 'card' && key !== 'inset') isolate(isolated, `${prefix}.${key}`, src[key])
  }
  const next: NonNullable<ThemeOverrides['surfaces']> = {}
  if (src.card !== undefined) {
    if (!src.card || typeof src.card !== 'object') isolate(isolated, `${prefix}.card`, src.card)
    else {
      const card = src.card as Record<string, unknown>
      for (const key of Object.keys(card)) {
        if (key !== 'topEdge') isolate(isolated, `${prefix}.card.${key}`, card[key])
      }
      if (typeof card.topEdge === 'boolean') {
        delete isolated[`${prefix}.card.topEdge`]
        next.card = { topEdge: card.topEdge }
      } else if (card.topEdge !== undefined) isolate(isolated, `${prefix}.card.topEdge`, card.topEdge)
    }
  }
  if (src.inset !== undefined) {
    if (!src.inset || typeof src.inset !== 'object') isolate(isolated, `${prefix}.inset`, src.inset)
    else {
      const inset = src.inset as Record<string, unknown>
      for (const key of Object.keys(inset)) {
        if (key !== 'hoverStrength') isolate(isolated, `${prefix}.inset.${key}`, inset[key])
      }
      if (typeof inset.hoverStrength === 'number' && Number.isFinite(inset.hoverStrength)) {
        delete isolated[`${prefix}.inset.hoverStrength`]
        next.inset = { hoverStrength: Math.min(1, Math.max(0, inset.hoverStrength)) }
      } else if (inset.hoverStrength !== undefined) isolate(isolated, `${prefix}.inset.hoverStrength`, inset.hoverStrength)
    }
  }
  return Object.keys(next).length ? next : undefined
}

function sanitizeOverrides(raw: unknown, prefix: string, isolated: Record<string, unknown>): ThemeOverrides {
  const src = raw && typeof raw === 'object' ? raw as Record<string, unknown> : {}
  const next: ThemeOverrides = {}
  for (const key of Object.keys(src)) {
    if (!KNOWN_OVERRIDE_KEYS.has(key)) isolate(isolated, `${prefix}.${key}`, src[key])
  }
  const accentKey = takeEnum(src.accentKey, ACCENT_KEYS, `${prefix}.accentKey`, isolated)
  if (accentKey) next.accentKey = accentKey
  if (typeof src.accentCustom === 'string') {
    next.accentCustom = src.accentCustom
    delete isolated[`${prefix}.accentCustom`]
  } else if (src.accentCustom !== undefined) isolate(isolated, `${prefix}.accentCustom`, src.accentCustom)
  if (typeof src.accentSecondary === 'string' && /^#[0-9a-fA-F]{6}$/.test(src.accentSecondary.trim())) {
    next.accentSecondary = src.accentSecondary.trim().toUpperCase()
    delete isolated[`${prefix}.accentSecondary`]
  } else if (src.accentSecondary !== undefined) isolate(isolated, `${prefix}.accentSecondary`, src.accentSecondary)
  const density = takeEnum(src.density, DENSITIES, `${prefix}.density`, isolated)
  if (density) next.density = density
  const uiFont = takeEnum(src.uiFont, UI_FONTS, `${prefix}.uiFont`, isolated)
  if (uiFont) next.uiFont = uiFont
  const sidebarWidth = takeEnum(src.sidebarWidth, SIDEBAR_WIDTHS, `${prefix}.sidebarWidth`, isolated)
  if (sidebarWidth) next.sidebarWidth = sidebarWidth
  const headerHeight = takeEnum(src.headerHeight, HEADER_HEIGHTS, `${prefix}.headerHeight`, isolated)
  if (headerHeight) next.headerHeight = headerHeight
  const radius = takeEnum(src.radius, RADIUS_PRESETS, `${prefix}.radius`, isolated)
  if (radius) next.radius = radius
  const card = takeEnum(src.card, CARD_VARIANTS, `${prefix}.card`, isolated)
  if (card) next.card = card
  const sidebarVariant = takeEnum(src.sidebarVariant, SIDEBAR_VARIANTS, `${prefix}.sidebarVariant`, isolated)
  if (sidebarVariant) next.sidebarVariant = sidebarVariant
  const headingStyle = takeEnum(src.headingStyle, HEADING_STYLES, `${prefix}.headingStyle`, isolated)
  if (headingStyle) next.headingStyle = headingStyle
  const subnav = takeEnum(src.subnav, SUBNAV_VARIANTS, `${prefix}.subnav`, isolated)
  if (subnav) next.subnav = subnav
  const iconSet = takeEnum(src.iconSet, ICON_SETS, `${prefix}.iconSet`, isolated)
  if (iconSet) next.iconSet = iconSet
  const iconContainer = takeEnum(src.iconContainer, ICON_CONTAINERS, `${prefix}.iconContainer`, isolated)
  if (iconContainer) next.iconContainer = iconContainer
  if (typeof src.transparency === 'boolean') next.transparency = src.transparency
  else if (src.transparency !== undefined) isolate(isolated, `${prefix}.transparency`, src.transparency)
  const surfacePreset = takeEnum(src.surfacePreset, SURFACE_PRESETS, `${prefix}.surfacePreset`, isolated)
  if (surfacePreset) next.surfacePreset = surfacePreset
  const termTheme = takeEnum(src.termTheme, TERM_THEMES, `${prefix}.termTheme`, isolated)
  if (termTheme) next.termTheme = termTheme
  const termFont = takeEnum(src.termFont, TERM_FONTS, `${prefix}.termFont`, isolated)
  if (termFont) next.termFont = termFont
  if (typeof src.termFontSize === 'number' && src.termFontSize >= 10 && src.termFontSize <= 24) next.termFontSize = src.termFontSize
  else if (src.termFontSize !== undefined) isolate(isolated, `${prefix}.termFontSize`, src.termFontSize)
  if (typeof src.termBgOpacity === 'number' && src.termBgOpacity >= 0 && src.termBgOpacity <= 1) next.termBgOpacity = src.termBgOpacity
  else if (src.termBgOpacity !== undefined) isolate(isolated, `${prefix}.termBgOpacity`, src.termBgOpacity)
  const termWallpaper = takeEnum(src.termWallpaper, TERM_WALLPAPERS, `${prefix}.termWallpaper`, isolated)
  if (termWallpaper) next.termWallpaper = termWallpaper
  if (typeof src.chromeTexture === 'string') {
    const texture = coerceChromeTexture(src.chromeTexture)
    if (texture !== 'none' || src.chromeTexture === 'none') {
      next.chromeTexture = texture
      delete isolated[`${prefix}.chromeTexture`]
    } else isolate(isolated, `${prefix}.chromeTexture`, src.chromeTexture)
  } else if (src.chromeTexture !== undefined) isolate(isolated, `${prefix}.chromeTexture`, src.chromeTexture)
  const chromeImageMode = takeEnum(src.chromeImageMode, WALLPAPER_IMAGE_MODES, `${prefix}.chromeImageMode`, isolated)
  if (chromeImageMode) next.chromeImageMode = chromeImageMode
  if (typeof src.chromeImageUrl === 'string') {
    const url = sanitizeWallpaperUrl(src.chromeImageUrl)
    if (url) next.chromeImageUrl = url
    else if (/^https?:\/\//i.test(src.chromeImageUrl.trim()) && !src.chromeImageUrl.trim().toLowerCase().startsWith('https://')) {
      isolate(isolated, `${prefix}.chromeImageUrl`, src.chromeImageUrl)
    }
  }
  if (typeof src.termFollowChrome === 'boolean') next.termFollowChrome = src.termFollowChrome
  else if (src.termFollowChrome !== undefined) isolate(isolated, `${prefix}.termFollowChrome`, src.termFollowChrome)
  const termImageMode = takeEnum(src.termImageMode, WALLPAPER_IMAGE_MODES, `${prefix}.termImageMode`, isolated)
  if (termImageMode) next.termImageMode = termImageMode
  if (typeof src.termImageUrl === 'string') {
    const url = sanitizeWallpaperUrl(src.termImageUrl)
    if (url) next.termImageUrl = url
    else if (/^https?:\/\//i.test(src.termImageUrl.trim()) && !src.termImageUrl.trim().toLowerCase().startsWith('https://')) {
      isolate(isolated, `${prefix}.termImageUrl`, src.termImageUrl)
    }
  }
  const surfaces = sanitizeSurfaces(src.surfaces, `${prefix}.surfaces`, isolated)
  if (surfaces) next.surfaces = surfaces
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
  const themeId: ThemeId = isThemeSlug(String(raw.themeId || '')) ? String(raw.themeId) : 'atelier'
  const mode = includes(THEME_MODES, raw.mode) ? raw.mode as ThemeMode : 'dark'
  const isolated: Record<string, unknown> = raw.isolated && typeof raw.isolated === 'object' && !Array.isArray(raw.isolated)
    ? { ...raw.isolated as Record<string, unknown> }
    : {}
  const overridesByTheme: AppearancePreference['overridesByTheme'] = {}
  const srcOverrides = raw.overridesByTheme && typeof raw.overridesByTheme === 'object'
    ? raw.overridesByTheme as Record<string, unknown>
    : {}
  for (const key of Object.keys(srcOverrides)) {
    if (!isThemeSlug(key)) {
      isolate(isolated, `overridesByTheme.${key}`, srcOverrides[key])
      continue
    }
    overridesByTheme[key] = sanitizeOverrides(srcOverrides[key], `overridesByTheme.${key}`, isolated)
  }
  const schemaMinor = typeof raw.schemaMinor === 'number' && raw.schemaMinor >= 0 ? Math.round(raw.schemaMinor) : 0
  return {
    schemaVersion: APPEARANCE_SCHEMA_VERSION,
    schemaMinor,
    themeId,
    mode,
    reduceMotion: Boolean(raw.reduceMotion),
    keepPersonalPrefsAcrossThemes: Boolean(raw.keepPersonalPrefsAcrossThemes),
    overridesByTheme,
    isolated: Object.keys(isolated).length ? isolated : undefined,
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
