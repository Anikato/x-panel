import { ensureContrast, generatePaletteFromHex, getPresetByKey } from '../utils/accent-colors.ts'
import { getTheme, hasTheme } from './catalog.ts'
import { getSurfacePreset, lightSurfaceFamily } from './surfaces.ts'
import { BLACK, isHex, mixSrgb, normalizeHex } from './pack-color.ts'
import {
  CROSS_THEME_KEYS,
  DEFAULT_PREFERENCE,
  coerceChromeTexture,
  TERM_WALLPAPERS,
  WALLPAPER_IMAGE_MODES,
  type AppearancePreference,
  type ChromeTexture,
  type TermWallpaper,
  type WallpaperImageMode,
  type ColorMode,
  type ResolveEnv,
  type ResolvedAccent,
  type ResolvedAppearance,
  type ThemeId,
  type ThemeModePack,
  type ThemeOverrides,
  type ThemeVariants,
} from './types.ts'

function hexToRgb(hex: string): string {
  const h = hex.replace('#', '')
  if (h.length < 6) return '122, 162, 255'
  return `${parseInt(h.slice(0, 2), 16)}, ${parseInt(h.slice(2, 4), 16)}, ${parseInt(h.slice(4, 6), 16)}`
}

function luminance(hex: string): number {
  const h = hex.replace('#', '')
  const r = parseInt(h.slice(0, 2), 16)
  const g = parseInt(h.slice(2, 4), 16)
  const b = parseInt(h.slice(4, 6), 16)
  return (0.299 * r + 0.587 * g + 0.114 * b) / 255
}

function resolveColorMode(mode: AppearancePreference['mode'], systemDark: boolean): ColorMode {
  if (mode === 'auto') return systemDark ? 'dark' : 'light'
  return mode === 'light' ? 'light' : 'dark'
}

function currentOverrides(pref: AppearancePreference): ThemeOverrides {
  return pref.overridesByTheme[pref.themeId] || {}
}

export function isCustomized(pref: AppearancePreference): boolean {
  const over = currentOverrides(pref)
  return Object.values(over).some((value) => value !== undefined && value !== '')
}

function resolvePackOrUserAccent(pack: ThemeModePack, over: ThemeOverrides): ResolvedAccent {
  if (over.accentKey || over.accentCustom) {
    return resolveAccent(over.accentKey || pack.accentKey, over.accentCustom || '')
  }
  if (pack.accentHex) {
    return {
      key: 'custom',
      custom: pack.accentHex,
      primary: pack.accentHex,
      hover: pack.accentHover || generatePaletteFromHex(pack.accentHex).hover,
      muted: pack.accentMuted || generatePaletteFromHex(pack.accentHex).muted,
      glow: generatePaletteFromHex(pack.accentHex).glow,
      secondary: pack.accentSecondaryHex || generatePaletteFromHex(pack.accentHex).secondary,
      onAccent: pack.onAccent || (luminance(pack.accentHex) > 0.62 ? '#0B0E14' : '#F8FAFC'),
      rgb: hexToRgb(pack.accentHex),
    }
  }
  return resolveAccent(pack.accentKey, '')
}

function resolveAccent(key: string, custom: string): ResolvedAccent {
  const palette = key === 'custom' && custom
    ? generatePaletteFromHex(custom)
    : getPresetByKey(key) || getPresetByKey('steel')!
  const onAccent = luminance(palette.primary) > 0.62 ? '#0B0E14' : '#F8FAFC'
  return {
    key: palette.key,
    custom: key === 'custom' ? custom : '',
    primary: palette.primary,
    hover: palette.hover,
    muted: palette.muted,
    glow: palette.glow,
    secondary: palette.secondary,
    onAccent,
    rgb: hexToRgb(palette.primary),
  }
}

function withAlpha(hex: string, alpha: number): string {
  if (!hex.startsWith('#') || hex.length < 7) return hex
  return `rgba(${hexToRgb(hex)}, ${alpha})`
}

export function resolveAppearance(pref: AppearancePreference, env: ResolveEnv): ResolvedAppearance {
  const theme = getTheme(pref.themeId)
  const colorMode = resolveColorMode(pref.mode, env.systemDark)
  const pack = theme.modes[colorMode]
  const over = currentOverrides(pref)
  const warnings: string[] = []

  const density = over.density || theme.defaults.density
  const uiFontKey = over.uiFont || theme.defaults.uiFont
  const sidebarWidthKey = over.sidebarWidth || theme.defaults.sidebarWidth
  const headerHeightKey = over.headerHeight || theme.defaults.headerHeight
  const radiusKey = over.radius || theme.defaults.radius
  const reduceMotion = Boolean(pref.reduceMotion || env.systemReduceMotion)
  const wantTransparency = over.transparency ?? theme.defaults.transparency ?? theme.tokens.materials.transparency
  const transparency = Boolean(wantTransparency && env.transparencySupported && !reduceMotion)

  const variants: ThemeVariants = {
    sidebar: over.sidebarVariant || theme.variants.sidebar,
    subnav: over.subnav || theme.variants.subnav,
    card: over.card || theme.variants.card,
    iconContainer: over.iconContainer || theme.variants.iconContainer,
  }
  const iconSet = over.iconSet || theme.iconSet || theme.defaults.iconSet
  let accent = resolvePackOrUserAccent(pack, over)

  let colors = { ...pack.colors }
  if (over.surfacePreset) {
    if (colorMode === 'dark') {
      const surface = getSurfacePreset(over.surfacePreset)
      if (surface) colors = { ...colors, ...surface.colors }
      else warnings.push(`unknown surface preset: ${over.surfacePreset}`)
    } else {
      colors = { ...colors, ...lightSurfaceFamily(over.surfacePreset) }
    }
  }

  if (colorMode === 'light') {
    colors = {
      ...colors,
      textSecondary: ensureContrast(colors.textSecondary, colors.bgSurface, 4.5),
      textMuted: ensureContrast(colors.textMuted, colors.bgSurface, 4.5),
    }
    const primary = ensureContrast(accent.primary, colors.bgSurface, 4.5)
    const hover = ensureContrast(accent.hover, colors.bgSurface, 4.5)
    accent = { ...accent, primary, hover, rgb: hexToRgb(primary) }
  }
  const onAccent = ensureContrast(
    luminance(accent.primary) > 0.62 ? '#0B0E14' : '#F8FAFC',
    accent.primary,
    4.5,
  )
  accent = { ...accent, onAccent }
  if (over.accentSecondary && isHex(over.accentSecondary)) {
    accent = { ...accent, secondary: normalizeHex(over.accentSecondary) }
  }

  const materials = {
    ...theme.tokens.materials,
    transparency,
    blurPx: transparency ? theme.tokens.materials.blurPx : 0,
    sidebarOpacity: transparency ? theme.tokens.materials.sidebarOpacity : 1,
    headerOpacity: transparency ? theme.tokens.materials.headerOpacity : 1,
    overlayOpacity: transparency ? theme.tokens.materials.overlayOpacity : 1,
    surfaceOpacity: transparency ? theme.tokens.materials.surfaceOpacity : 1,
  }

  const densityTokens = theme.tokens.densities[density]
  const shapes = theme.tokens.shapes[radiusKey]
  const motion = reduceMotion
    ? { hoverMs: 0, pageMs: 0, menuMs: 0, drawerMs: 0, ease: 'linear' }
    : theme.tokens.motion

  const terminalTheme = over.termTheme || (colorMode === 'light' ? theme.terminal.light : theme.terminal.dark)
  const termFont = over.termFont || theme.defaults.termFont
  const termFontSize = over.termFontSize || theme.defaults.termFontSize
  const termBgOpacity = over.termBgOpacity ?? theme.defaults.termBgOpacity
  const termWallpaper: TermWallpaper = TERM_WALLPAPERS.includes(over.termWallpaper as TermWallpaper)
    ? over.termWallpaper as TermWallpaper
    : 'none'
  const chromeTexture: ChromeTexture = coerceChromeTexture(over.chromeTexture ?? theme.defaults.chromeTexture)
  const chromeImageMode: WallpaperImageMode = WALLPAPER_IMAGE_MODES.includes(over.chromeImageMode as WallpaperImageMode)
    ? over.chromeImageMode as WallpaperImageMode
    : 'none'
  const chromeImageUrl = typeof over.chromeImageUrl === 'string' ? over.chromeImageUrl.trim() : ''
  const hasChromeLayer = chromeTexture !== 'none' || chromeImageMode !== 'none'
  const termFollowChrome = typeof over.termFollowChrome === 'boolean' ? over.termFollowChrome : hasChromeLayer
  const termImageMode: WallpaperImageMode = WALLPAPER_IMAGE_MODES.includes(over.termImageMode as WallpaperImageMode)
    ? over.termImageMode as WallpaperImageMode
    : 'none'
  const termImageUrl = typeof over.termImageUrl === 'string' ? over.termImageUrl.trim() : ''
  const editorTheme = colorMode === 'light' ? theme.editor.light : theme.editor.dark
  const hoverStrength = over.surfaces?.inset?.hoverStrength ?? theme.surfaces?.hoverStrength ?? 1
  const insetHover = hoverStrength <= 0
    ? 'transparent'
    : `color-mix(in srgb, ${colors.textPrimary} ${Math.round(8 * hoverStrength)}%, ${colors.bgInset})`
  const insetSelected = mixSrgb(colors.bgInset, accent.primary, 0.15)
  const primaryTint = colorMode === 'light' ? '#FFFFFF' : BLACK

  const sidebarSolid = colors.bgSidebar
  const headerSolid = colors.bgHeader
  const overlaySolid = colors.bgOverlay

  const cssVars: Record<string, string> = {
    '--xp-bg-base': colors.bgBase,
    '--xp-bg-surface': colors.bgSurface,
    '--xp-bg-card': colors.bgSurface,
    '--xp-bg-elevated': colors.bgElevated,
    '--xp-bg-overlay': transparency ? withAlpha(overlaySolid, materials.overlayOpacity) : overlaySolid,
    '--xp-bg-sidebar': transparency ? withAlpha(sidebarSolid, materials.sidebarOpacity) : sidebarSolid,
    '--xp-bg-sidebar-solid': sidebarSolid,
    '--xp-bg-header': transparency ? withAlpha(headerSolid, materials.headerOpacity) : headerSolid,
    '--xp-bg-header-solid': headerSolid,
    '--xp-bg-input': colors.bgInput,
    '--xp-bg-table-header': colors.bgTableHeader,
    '--xp-bg-inset': colors.bgInset,
    '--xp-bg-main-gradient': colors.bgBase,
    '--xp-bg-auth': colors.bgAuth,
    '--xp-text-primary': colors.textPrimary,
    '--xp-text-secondary': colors.textSecondary,
    '--xp-text-muted': colors.textMuted,
    '--xp-border': colors.border,
    '--xp-border-light': colors.borderLight,
    '--xp-border-hover': colors.borderHover,
    '--xp-success': colors.success,
    '--xp-warning': colors.warning,
    '--xp-danger': colors.danger,
    '--xp-info': colors.info,
    '--xp-accent': accent.primary,
    '--xp-on-accent': accent.onAccent,
    '--xp-accent-rgb': accent.rgb,
    '--xp-accent-hover': accent.hover,
    '--xp-accent-muted': accent.muted,
    '--xp-accent-glow': accent.glow,
    '--xp-accent-secondary': accent.secondary,
    '--xp-logo-primary': accent.primary,
    '--xp-logo-secondary': accent.secondary,
    '--xp-font-family': theme.tokens.fonts.ui[uiFontKey] || theme.tokens.fonts.ui.system,
    '--xp-font-mono': theme.tokens.fonts.mono,
    '--xp-font-size': densityTokens.fontSize,
    '--xp-spacing': densityTokens.spacing,
    '--xp-form-margin': densityTokens.formMargin,
    '--xp-control-height': densityTokens.controlHeight,
    '--xp-table-row-height': densityTokens.rowHeight,
    '--xp-radius': shapes.radiusCard,
    '--xp-radius-sm': shapes.radiusControl,
    '--xp-radius-lg': shapes.radiusOverlay,
    '--xp-sidebar-width': theme.tokens.sidebarWidths[sidebarWidthKey],
    '--xp-sidebar-collapse-width': '64px',
    '--xp-header-height': theme.tokens.headerHeights[headerHeightKey],
    '--xp-blur': `${materials.blurPx}px`,
    '--xp-surface-opacity': String(materials.surfaceOpacity),
    '--xp-sidebar-opacity': String(materials.sidebarOpacity),
    '--xp-header-opacity': String(materials.headerOpacity),
    '--xp-overlay-opacity': String(materials.overlayOpacity),
    '--xp-motion-hover': `${motion.hoverMs}ms`,
    '--xp-motion-page': `${motion.pageMs}ms`,
    '--xp-motion-menu': `${motion.menuMs}ms`,
    '--xp-motion-drawer': `${motion.drawerMs}ms`,
    '--xp-motion-ease': motion.ease,
    '--xp-shadow-card': colorMode === 'dark' ? '0 8px 24px rgba(0, 0, 0, 0.28)' : '0 10px 28px rgba(26, 23, 20, 0.08)',
    '--xp-shadow-overlay': colorMode === 'dark' ? '0 18px 48px rgba(0, 0, 0, 0.45)' : '0 18px 40px rgba(26, 23, 20, 0.12)',
    '--el-bg-color': colors.bgSurface,
    '--el-bg-color-overlay': overlaySolid,
    '--el-bg-color-page': colors.bgBase,
    '--el-text-color-primary': colors.textPrimary,
    '--el-text-color-regular': colors.textSecondary,
    '--el-text-color-secondary': colors.textMuted,
    '--el-border-color': colors.border,
    '--el-border-color-light': colors.borderLight,
    '--el-fill-color-blank': colors.bgSurface,
    '--el-mask-color': colorMode === 'dark' ? 'rgba(0, 0, 0, 0.62)' : 'rgba(26, 23, 20, 0.42)',
    '--el-box-shadow': 'var(--xp-shadow-overlay)',
    '--el-card-bg-color': colors.bgSurface,
    '--el-dialog-bg-color': overlaySolid,
    '--el-drawer-bg-color': colors.bgSurface,
    '--el-color-primary': accent.primary,
    '--el-color-primary-dark-2': mixSrgb(accent.primary, BLACK, 0.2),
    '--el-color-primary-light-3': mixSrgb(accent.primary, primaryTint, colorMode === 'light' ? 0.35 : 0.28),
    '--el-color-primary-light-5': mixSrgb(accent.primary, primaryTint, colorMode === 'light' ? 0.55 : 0.46),
    '--el-color-primary-light-7': mixSrgb(accent.primary, primaryTint, colorMode === 'light' ? 0.72 : 0.62),
    '--el-color-primary-light-8': mixSrgb(accent.primary, primaryTint, colorMode === 'light' ? 0.84 : 0.74),
    '--el-color-primary-light-9': mixSrgb(accent.primary, colorMode === 'light' ? '#FFFFFF' : colors.bgSurface, colorMode === 'light' ? 0.92 : 0.78),
    '--el-menu-active-color': accent.primary,
    '--xp-btn-primary-bg': accent.hover,
    '--xp-btn-primary-hover': accent.primary,
    '--xp-chart-0': theme.charts.categorical[0] || accent.primary,
    '--xp-chart-1': theme.charts.categorical[1] || colors.success,
    '--xp-chart-2': theme.charts.categorical[2] || colors.warning,
    '--xp-chart-3': theme.charts.categorical[3] || colors.danger,
    '--xp-chart-4': theme.charts.categorical[4] || accent.secondary,
    '--xp-chart-5': theme.charts.categorical[5] || colors.info,
    '--el-popup-modal-bg-color': overlaySolid,
    '--xp-card-top-edge-shadow': (over.surfaces?.card?.topEdge ?? theme.surfaces?.cardTopEdge ?? true)
      ? `inset 0 1px 0 color-mix(in srgb, ${accent.primary} 16%, transparent)`
      : 'none',
    '--xp-inset-hover': insetHover,
    '--xp-inset-selected': insetSelected,
    '--el-table-row-hover-bg-color': insetHover,
    '--el-table-current-row-bg-color': insetSelected,
    '--xp-chrome-veil': chromeImageMode !== 'none'
      ? `linear-gradient(${withAlpha(colors.bgBase, colorMode === 'dark' ? 0.62 : 0.72)}, ${withAlpha(colors.bgBase, colorMode === 'dark' ? 0.78 : 0.86)})`
      : 'none',
  }

  const datasets: Record<string, string> = {
    theme: theme.id,
    density,
    'card-variant': variants.card,
    'sidebar-variant': variants.sidebar,
    'subnav-variant': variants.subnav,
    'icon-set': iconSet,
    'icon-container': variants.iconContainer,
    transparency: transparency ? 'on' : 'off',
    'color-mode': colorMode,
    'term-wallpaper': termWallpaper,
    'chrome-texture': chromeTexture,
    'chrome-photo': chromeImageMode !== 'none' ? 'on' : 'off',
    'term-follow': termFollowChrome ? 'on' : 'off',
  }

  const themeMissing = !hasTheme(pref.themeId)
  return {
    themeId: pref.themeId,
    themeName: themeMissing ? pref.themeId : theme.name,
    themeVersion: theme.version,
    themeMissing,
    customized: isCustomized(pref),
    colorMode,
    modePreference: pref.mode,
    reduceMotion,
    transparency,
    density,
    uiFontKey,
    uiFont: theme.tokens.fonts.ui[uiFontKey] || theme.tokens.fonts.ui.system,
    monoFont: theme.tokens.fonts.mono,
    sidebarWidth: theme.tokens.sidebarWidths[sidebarWidthKey],
    headerHeight: theme.tokens.headerHeights[headerHeightKey],
    variants,
    iconSet,
    colors,
    accent,
    shapes,
    densityTokens,
    materials,
    motion,
    cssVars,
    datasets,
    terminalTheme,
    termFont,
    termFontSize,
    termBgOpacity,
    termWallpaper,
    chromeTexture,
    chromeImageMode,
    chromeImageUrl,
    termFollowChrome,
    termImageMode,
    termImageUrl,
    editorTheme,
    charts: theme.charts,
    warnings,
  }
}

export function switchTheme(pref: AppearancePreference, nextId: ThemeId): AppearancePreference {
  const next: AppearancePreference = {
    ...pref,
    themeId: nextId,
    overridesByTheme: { ...pref.overridesByTheme },
  }
  if (!pref.keepPersonalPrefsAcrossThemes) return next

  const from = pref.overridesByTheme[pref.themeId] || {}
  const existing = { ...(next.overridesByTheme[nextId] || {}) }
  for (const key of CROSS_THEME_KEYS) {
    const value = from[key]
    if (value !== undefined) (existing as ThemeOverrides)[key] = value as never
  }
  next.overridesByTheme[nextId] = existing
  return next
}

export function clonePreference(pref: AppearancePreference): AppearancePreference {
  return {
    schemaVersion: pref.schemaVersion,
    schemaMinor: pref.schemaMinor,
    themeId: pref.themeId,
    mode: pref.mode,
    reduceMotion: pref.reduceMotion,
    keepPersonalPrefsAcrossThemes: pref.keepPersonalPrefsAcrossThemes,
    overridesByTheme: JSON.parse(JSON.stringify(pref.overridesByTheme || {})),
    isolated: pref.isolated ? { ...pref.isolated } : undefined,
  }
}

export function emptyPreference(): AppearancePreference {
  return clonePreference(DEFAULT_PREFERENCE)
}
