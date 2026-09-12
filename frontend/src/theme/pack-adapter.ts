import { atelierTheme } from './themes/atelier.ts'
import { DEFAULT_MOTION, DENSITY_TOKENS, HEADER_HEIGHT_TOKENS, RADIUS_TOKENS, SIDEBAR_WIDTH_TOKENS } from './shared-tokens.ts'
import type { ModeRuntime, ResolvedPack } from './pack-parse.ts'
import type {
  ChromeTexture,
  Density,
  HeaderHeight,
  IconSet,
  RadiusPreset,
  SemanticColors,
  SidebarWidth,
  ThemeDefinition,
  ThemeModePack,
  ThemeVariants,
  UiFont,
} from './types.ts'

function isObj(v: unknown): v is Record<string, unknown> {
  return Boolean(v) && typeof v === 'object' && !Array.isArray(v)
}

function num(raw: unknown, fallback: number): number {
  return typeof raw === 'number' && Number.isFinite(raw) ? raw : fallback
}

function toColors(mode: ModeRuntime): SemanticColors {
  return {
    bgBase: mode.bg,
    bgSurface: mode.surface,
    bgElevated: mode.elevated,
    bgOverlay: mode.overlay,
    bgSidebar: mode.sidebar,
    bgHeader: mode.header,
    bgInput: mode.input,
    bgTableHeader: mode.tableHeader,
    bgInset: mode.inset,
    bgAuth: mode.auth,
    textPrimary: mode.text,
    textSecondary: mode.textSecondary,
    textMuted: mode.textMuted,
    border: mode.border,
    borderLight: mode.borderLight,
    borderHover: mode.borderHover,
    success: mode.success,
    warning: mode.warning,
    danger: mode.danger,
    info: mode.info,
  }
}

function toModePack(mode: ModeRuntime, terminal: string, editor: string): ThemeModePack {
  return {
    colors: toColors(mode),
    accentKey: 'custom',
    accentHex: mode.accent,
    accentHover: mode.hover,
    accentMuted: mode.muted,
    accentSecondaryHex: mode.accentSecondary,
    onAccent: mode.onAccent,
    terminalTheme: terminal,
    editorTheme: editor,
  }
}

export function packToThemeDefinition(pack: ResolvedPack, raw: Record<string, unknown>): ThemeDefinition {
  const tokensRaw = isObj(raw.tokens) ? raw.tokens : {}
  const mat = isObj(tokensRaw.materials) ? tokensRaw.materials : {}
  const motionRaw = isObj(tokensRaw.motion) ? tokensRaw.motion : {}
  const terminal = isObj(raw.terminal) ? raw.terminal : {}
  const editor = isObj(raw.editor) ? raw.editor : {}
  const d = pack.defaults
  return {
    schemaVersion: 1,
    id: pack.id,
    name: pack.name,
    version: pack.version,
    source: 'pack',
    modes: {
      dark: toModePack(pack.modes.dark, String(terminal.dark || 'default'), String(editor.dark || 'vs-dark')),
      light: toModePack(pack.modes.light, String(terminal.light || 'paper'), String(editor.light || 'vs')),
    },
    defaults: {
      mode: d.mode === 'light' || d.mode === 'auto' ? d.mode : 'dark',
      density: d.density as Density,
      uiFont: d.uiFont as UiFont,
      sidebarWidth: d.sidebarWidth as SidebarWidth,
      headerHeight: d.headerHeight as HeaderHeight,
      radius: d.radius as RadiusPreset,
      reduceMotion: false,
      transparency: d.transparency,
      iconSet: d.iconSet as IconSet,
      termFont: d.termFont,
      termFontSize: d.termFontSize,
      termBgOpacity: d.termBgOpacity,
      chromeTexture: d.chromeTexture as ChromeTexture,
    },
    tokens: {
      fonts: atelierTheme.tokens.fonts,
      materials: {
        transparency: d.transparency,
        surfaceOpacity: num(mat.surfaceOpacity, 0.88),
        blurPx: num(mat.blurPx, 12),
        sidebarOpacity: num(mat.sidebarOpacity, 0.78),
        headerOpacity: num(mat.headerOpacity, 0.82),
        overlayOpacity: num(mat.overlayOpacity, 0.9),
      },
      shapes: RADIUS_TOKENS,
      motion: {
        hoverMs: num(motionRaw.hoverMs, DEFAULT_MOTION.hoverMs),
        pageMs: num(motionRaw.pageMs, DEFAULT_MOTION.pageMs),
        menuMs: num(motionRaw.menuMs, DEFAULT_MOTION.menuMs),
        drawerMs: num(motionRaw.drawerMs, DEFAULT_MOTION.drawerMs),
        ease: typeof motionRaw.ease === 'string' ? motionRaw.ease : DEFAULT_MOTION.ease,
      },
      densities: DENSITY_TOKENS,
      sidebarWidths: SIDEBAR_WIDTH_TOKENS,
      headerHeights: HEADER_HEIGHT_TOKENS,
    },
    variants: d.variants as ThemeVariants,
    iconSet: d.iconSet as IconSet,
    terminal: {
      dark: String(terminal.dark || 'default'),
      light: String(terminal.light || 'paper'),
    },
    editor: {
      dark: String(editor.dark || 'vs-dark'),
      light: String(editor.light || 'vs'),
    },
    charts: {
      categorical: pack.modes.dark.categorical,
      sequential: pack.modes.dark.sequential,
    },
    surfaces: pack.surfaces,
  }
}
