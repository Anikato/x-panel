export { APPEARANCE_SCHEMA_VERSION, DEFAULT_PREFERENCE, CROSS_THEME_KEYS, OVERRIDE_GROUPS, TERM_WALLPAPERS, CHROME_TEXTURES, WALLPAPER_IMAGE_MODES, coerceChromeTexture } from './types.ts'
export type {
  AppearancePreference,
  ThemeDefinition,
  ThemeId,
  ThemeMode,
  ThemeOverrides,
  ResolvedAppearance,
  Density,
  UiFont,
  OverrideGroup,
  TermWallpaper,
  ChromeTexture,
  WallpaperImageMode,
} from './types.ts'
export { listThemes, getTheme, hasTheme } from './catalog.ts'
export { resolveAppearance, switchTheme, clonePreference, isCustomized, emptyPreference } from './resolve.ts'
export { hydrateAppearance, migrateLegacyAppearance, sanitizePreference } from './migrate.ts'
export { buildAppearancePreset, parseAppearancePreset, APPEARANCE_PRESET_KIND } from './preset-file.ts'
export {
  beginPreview,
  patchDraft,
  applyPreview,
  cancelPreview,
  restoreGroup,
  restoreThemeDefaults,
  restoreAllDefaults,
  type PreviewSession,
} from './preview.ts'
export { applyResolvedAppearance, readResolveEnv } from './apply.ts'
export { mapNavIcon, resolveSemanticIcon } from './icons.ts'
export { chartTokens, cssVar, colorAlpha, onAppearanceChange, APPEARANCE_CHANGED } from './charts.ts'
export { SURFACE_PRESET_DEFS, getSurfacePreset } from './surfaces.ts'
export { readLocalPreference, writeLocalPreference, readStoredV1, APPEARANCE_STORAGE_KEY } from './storage.ts'
export {
  fileToWallpaperDataUrl,
  hasWallpaperDraft,
  readWallpaperData,
  sanitizeWallpaperUrl,
  setDraftWallpaper,
  wallpaperDraftGen,
  wallpaperJobMatches,
  writeWallpaperData,
} from './wallpaper-store.ts'
export { UI_FONT_STACKS, DENSITY_TOKENS, SIDEBAR_WIDTH_TOKENS, FONT_CDN } from './shared-tokens.ts'
