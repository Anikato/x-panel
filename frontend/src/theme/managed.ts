import {
  BUILTIN_THEME_IDS, THEME_MODES, DENSITIES, UI_FONTS, RADIUS_PRESETS, SIDEBAR_WIDTHS,
  type AppearancePreference, type ThemeMode, type Density, type UiFont, type RadiusPreset, type SidebarWidth,
} from './types.ts'

export interface ManagedAppearance {
  schemaVersion: 1
  themeId: typeof BUILTIN_THEME_IDS[number]
  mode: ThemeMode
  density?: Density
  font?: Exclude<UiFont, 'custom'>
  radius?: RadiusPreset
  sidebarWidth?: SidebarWidth
  accent?: string
}

export function parseManagedAppearance(raw?: string): ManagedAppearance | null {
  if (!raw) return null
  try {
    const value: ManagedAppearance = JSON.parse(raw)
    if (!value || value.schemaVersion !== 1 ||
        !BUILTIN_THEME_IDS.includes(value.themeId) || !THEME_MODES.includes(value.mode) ||
        (value.density && !DENSITIES.includes(value.density)) ||
        (value.font && !UI_FONTS.includes(value.font)) ||
        (value.radius && !RADIUS_PRESETS.includes(value.radius)) ||
        (value.sidebarWidth && !SIDEBAR_WIDTHS.includes(value.sidebarWidth)) ||
        (value.accent && !/^#[0-9a-f]{6}$/i.test(value.accent))) return null
    return value
  } catch {
    return null
  }
}

export function applyManagedAppearance(
  personal: AppearancePreference,
  managed: ManagedAppearance | null,
): AppearancePreference {
  if (!managed) return personal
  const overrides = { ...personal.overridesByTheme[managed.themeId] }
  delete overrides.density
  delete overrides.uiFont
  delete overrides.radius
  delete overrides.sidebarWidth
  delete overrides.accentKey
  delete overrides.accentCustom
  delete overrides.accentSecondary
  delete overrides.surfacePreset
  delete overrides.surfaces
  return {
    ...personal,
    themeId: managed.themeId,
    mode: managed.mode,
    overridesByTheme: {
      ...personal.overridesByTheme,
      [managed.themeId]: {
        ...overrides,
        ...(managed.density && { density: managed.density }),
        ...(managed.font && { uiFont: managed.font }),
        ...(managed.radius && { radius: managed.radius }),
        ...(managed.sidebarWidth && { sidebarWidth: managed.sidebarWidth }),
        ...(managed.accent && { accentKey: 'custom', accentCustom: managed.accent }),
      },
    },
  }
}
