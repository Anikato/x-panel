import { APPEARANCE_SCHEMA_VERSION, type AppearancePreference, type ThemeId, type ThemeOverrides } from './types.ts'
import { clonePreference } from './resolve.ts'
import { sanitizePreference } from './migrate.ts'

export const APPEARANCE_PRESET_KIND = 'x-panel.appearance'

export interface AppearancePresetFile {
  kind: typeof APPEARANCE_PRESET_KIND
  schemaVersion: typeof APPEARANCE_SCHEMA_VERSION
  exportedAt: string
  preference: AppearancePreference
}

function stripLocalAssets(pref: AppearancePreference): AppearancePreference {
  const next = clonePreference(pref)
  for (const key of Object.keys(next.overridesByTheme) as ThemeId[]) {
    const over = next.overridesByTheme[key]
    if (!over) continue
    const cleaned: ThemeOverrides = { ...over }
    if (cleaned.chromeImageMode === 'upload') {
      cleaned.chromeImageMode = 'none'
      cleaned.chromeImageUrl = ''
    }
    if (cleaned.termImageMode === 'upload') {
      cleaned.termImageMode = 'none'
      cleaned.termImageUrl = ''
    }
    next.overridesByTheme[key] = cleaned
  }
  return next
}

export function buildAppearancePreset(pref: AppearancePreference, now = new Date()): AppearancePresetFile {
  return {
    kind: APPEARANCE_PRESET_KIND,
    schemaVersion: APPEARANCE_SCHEMA_VERSION,
    exportedAt: now.toISOString(),
    preference: stripLocalAssets(sanitizePreference(pref)),
  }
}

export function parseAppearancePreset(raw: unknown): { ok: true, preference: AppearancePreference } | { ok: false, error: 'invalid' } {
  let data = raw
  if (typeof raw === 'string') {
    try {
      data = JSON.parse(raw)
    } catch {
      return { ok: false, error: 'invalid' }
    }
  }
  if (!data || typeof data !== 'object') return { ok: false, error: 'invalid' }
  const file = data as Partial<AppearancePresetFile> & { preference?: unknown }
  if (file.kind !== APPEARANCE_PRESET_KIND) return { ok: false, error: 'invalid' }
  if (file.schemaVersion !== APPEARANCE_SCHEMA_VERSION) return { ok: false, error: 'invalid' }
  if (!file.preference || typeof file.preference !== 'object') return { ok: false, error: 'invalid' }
  return { ok: true, preference: stripLocalAssets(sanitizePreference(file.preference)) }
}
