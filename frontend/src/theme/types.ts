export const APPEARANCE_SCHEMA_VERSION = 1 as const

export type ThemeId = 'atelier' | 'lumen'
export type ColorMode = 'dark' | 'light'
export type ThemeMode = 'dark' | 'light' | 'auto'
export type Density = 'compact' | 'default' | 'comfortable'
export type UiFont = 'system' | 'inter' | 'noto' | 'lxgw'
export type SidebarWidth = 'narrow' | 'default' | 'wide'
export type HeaderHeight = 'compact' | 'default' | 'comfortable'
export type RadiusPreset = 'sharp' | 'default' | 'rounded'
export type CardVariant = 'flat' | 'outline' | 'raised'
export type SidebarVariant = 'marker' | 'block'
export type SubnavVariant = 'line' | 'block' | 'pill'
export type IconSet = 'outline' | 'solid'
export type IconContainer = 'none' | 'tile'
export type SurfacePreset = 'graphite' | 'abyss' | 'void' | 'tinted' | 'cosmos' | 'warm'
export type TermWallpaper = 'none'
export type ChromeTexture = 'none' | 'ribbon' | 'galaxy' | 'starfield'
export type WallpaperImageMode = 'none' | 'url' | 'upload'

export const THEME_IDS: ThemeId[] = ['atelier', 'lumen']
export const THEME_MODES: ThemeMode[] = ['dark', 'light', 'auto']
export const DENSITIES: Density[] = ['compact', 'default', 'comfortable']
export const UI_FONTS: UiFont[] = ['system', 'inter', 'noto', 'lxgw']
export const SIDEBAR_WIDTHS: SidebarWidth[] = ['narrow', 'default', 'wide']
export const HEADER_HEIGHTS: HeaderHeight[] = ['compact', 'default', 'comfortable']
export const RADIUS_PRESETS: RadiusPreset[] = ['sharp', 'default', 'rounded']
export const CARD_VARIANTS: CardVariant[] = ['flat', 'outline', 'raised']
export const SIDEBAR_VARIANTS: SidebarVariant[] = ['marker', 'block']
export const SUBNAV_VARIANTS: SubnavVariant[] = ['line', 'block', 'pill']
export const ICON_SETS: IconSet[] = ['outline', 'solid']
export const ICON_CONTAINERS: IconContainer[] = ['none', 'tile']
export const SURFACE_PRESETS: SurfacePreset[] = ['graphite', 'abyss', 'void', 'tinted', 'cosmos', 'warm']
export const TERM_WALLPAPERS: TermWallpaper[] = ['none']
export const CHROME_TEXTURES: ChromeTexture[] = ['none', 'ribbon', 'galaxy', 'starfield']

export function coerceChromeTexture(raw: unknown): ChromeTexture {
  if (raw === 'ribbon' || raw === 'galaxy' || raw === 'starfield' || raw === 'none') return raw
  if (raw === 'diagonal') return 'ribbon'
  if (raw === 'dots') return 'starfield'
  if (raw === 'grain' || raw === 'grid') return 'galaxy'
  return 'none'
}
export const WALLPAPER_IMAGE_MODES: WallpaperImageMode[] = ['none', 'url', 'upload']

export const CROSS_THEME_KEYS = ['density', 'uiFont', 'termFont', 'termFontSize'] as const
export type CrossThemeKey = typeof CROSS_THEME_KEYS[number]

export const OVERRIDE_GROUPS = {
  common: ['density', 'uiFont', 'sidebarWidth', 'headerHeight'],
  accent: ['accentKey', 'accentCustom'],
  variants: ['card', 'sidebarVariant', 'subnav', 'iconSet', 'iconContainer', 'radius'],
  material: ['transparency', 'surfacePreset', 'chromeTexture', 'chromeImageMode', 'chromeImageUrl'],
  terminal: ['termTheme', 'termFont', 'termFontSize', 'termBgOpacity', 'termWallpaper', 'termFollowChrome', 'termImageMode', 'termImageUrl'],
} as const

export type OverrideGroup = keyof typeof OVERRIDE_GROUPS

export interface SemanticColors {
  bgBase: string
  bgSurface: string
  bgElevated: string
  bgOverlay: string
  bgSidebar: string
  bgHeader: string
  bgInput: string
  bgTableHeader: string
  bgInset: string
  bgAuth: string
  textPrimary: string
  textSecondary: string
  textMuted: string
  border: string
  borderLight: string
  borderHover: string
  success: string
  warning: string
  danger: string
  info: string
}

export interface MaterialTokens {
  transparency: boolean
  surfaceOpacity: number
  blurPx: number
  sidebarOpacity: number
  headerOpacity: number
  overlayOpacity: number
}

export interface ShapeTokens {
  radiusControl: string
  radiusCard: string
  radiusOverlay: string
}

export interface MotionTokens {
  hoverMs: number
  pageMs: number
  menuMs: number
  drawerMs: number
  ease: string
}

export interface DensityTokens {
  fontSize: string
  spacing: string
  formMargin: string
  controlHeight: string
  rowHeight: string
}

export interface ThemeVariants {
  sidebar: SidebarVariant
  subnav: SubnavVariant
  card: CardVariant
  iconContainer: IconContainer
}

export interface ThemeModePack {
  colors: SemanticColors
  accentKey: string
  terminalTheme: string
  editorTheme: string
}

export interface ThemeDefinition {
  schemaVersion: typeof APPEARANCE_SCHEMA_VERSION
  id: ThemeId
  name: string
  version: string
  modes: {
    dark: ThemeModePack
    light: ThemeModePack
  }
  defaults: {
    mode: ThemeMode
    density: Density
    uiFont: UiFont
    sidebarWidth: SidebarWidth
    headerHeight: HeaderHeight
    radius: RadiusPreset
    reduceMotion: boolean
    transparency: boolean
    iconSet: IconSet
    termFont: string
    termFontSize: number
    termBgOpacity: number
  }
  tokens: {
    fonts: { ui: Record<UiFont, string>; mono: string }
    materials: MaterialTokens
    shapes: Record<RadiusPreset, ShapeTokens>
    motion: MotionTokens
    densities: Record<Density, DensityTokens>
    sidebarWidths: Record<SidebarWidth, string>
    headerHeights: Record<HeaderHeight, string>
  }
  variants: ThemeVariants
  iconSet: IconSet
  terminal: { dark: string; light: string }
  editor: { dark: string; light: string }
  charts: { categorical: string[]; sequential: string[] }
  assets?: Record<string, never>
}

export interface ThemeOverrides {
  accentKey?: string
  accentCustom?: string
  density?: Density
  uiFont?: UiFont
  sidebarWidth?: SidebarWidth
  headerHeight?: HeaderHeight
  radius?: RadiusPreset
  card?: CardVariant
  sidebarVariant?: SidebarVariant
  subnav?: SubnavVariant
  iconSet?: IconSet
  iconContainer?: IconContainer
  transparency?: boolean
  surfacePreset?: SurfacePreset
  termTheme?: string
  termFont?: string
  termFontSize?: number
  termBgOpacity?: number
  termWallpaper?: TermWallpaper
  chromeTexture?: ChromeTexture
  chromeImageMode?: WallpaperImageMode
  chromeImageUrl?: string
  termFollowChrome?: boolean
  termImageMode?: WallpaperImageMode
  termImageUrl?: string
}

export interface AppearancePreference {
  schemaVersion: number
  themeId: ThemeId
  mode: ThemeMode
  reduceMotion: boolean
  keepPersonalPrefsAcrossThemes: boolean
  overridesByTheme: Partial<Record<ThemeId, ThemeOverrides>>
}

export interface ResolveEnv {
  systemDark: boolean
  systemReduceMotion: boolean
  transparencySupported: boolean
}

export interface ResolvedAccent {
  key: string
  custom: string
  primary: string
  hover: string
  muted: string
  glow: string
  secondary: string
  onAccent: string
  rgb: string
}

export interface ResolvedAppearance {
  themeId: ThemeId
  themeName: string
  themeVersion: string
  customized: boolean
  colorMode: ColorMode
  modePreference: ThemeMode
  reduceMotion: boolean
  transparency: boolean
  density: Density
  uiFontKey: UiFont
  uiFont: string
  monoFont: string
  sidebarWidth: string
  headerHeight: string
  variants: ThemeVariants
  iconSet: IconSet
  colors: SemanticColors
  accent: ResolvedAccent
  shapes: ShapeTokens
  densityTokens: DensityTokens
  materials: MaterialTokens
  motion: MotionTokens
  cssVars: Record<string, string>
  datasets: Record<string, string>
  terminalTheme: string
  termFont: string
  termFontSize: number
  termBgOpacity: number
  termWallpaper: TermWallpaper
  chromeTexture: ChromeTexture
  chromeImageMode: WallpaperImageMode
  chromeImageUrl: string
  termFollowChrome: boolean
  termImageMode: WallpaperImageMode
  termImageUrl: string
  editorTheme: string
  charts: ThemeDefinition['charts']
  warnings: string[]
}

export interface LegacyAppearance {
  theme?: string
  accentKey?: string
  accentCustom?: string
  bgPreset?: string
  uiFont?: string
  uiDensity?: string
  borderRadiusPreset?: string
  reduceMotion?: boolean
  cardBorderStyle?: string
  sidebarWidth?: string
  termTheme?: string
  termFont?: string
  termFontSize?: number
  termBgOpacity?: number
}

export const DEFAULT_PREFERENCE: AppearancePreference = {
  schemaVersion: APPEARANCE_SCHEMA_VERSION,
  themeId: 'atelier',
  mode: 'dark',
  reduceMotion: false,
  keepPersonalPrefsAcrossThemes: false,
  overridesByTheme: {},
}
