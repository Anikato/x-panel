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

function isObj(v: unknown): v is Record<string, unknown> {
  return Boolean(v) && typeof v === 'object' && !Array.isArray(v)
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
    if (cleaned.uiFont === 'custom') cleaned.uiFont = 'system'
    next.overridesByTheme[key] = cleaned
  }
  if (next.isolated) {
    const isolated = { ...next.isolated }
    for (const path of Object.keys(isolated)) {
      if (path.endsWith('chromeImageUrl') || path.endsWith('termImageUrl')) {
        const value = isolated[path]
        if (typeof value === 'string' && !/^https:\/\//i.test(value)) delete isolated[path]
      }
    }
    next.isolated = Object.keys(isolated).length ? isolated : undefined
  }
  return next
}

function applyIsolated(target: Record<string, unknown>, isolated: Record<string, unknown>) {
  for (const [path, value] of Object.entries(isolated)) {
    const segs = path.split('.').filter(Boolean)
    if (!segs.length) continue
    let cur: Record<string, unknown> = target
    for (let i = 0; i < segs.length - 1; i++) {
      const seg = segs[i]
      if (!isObj(cur[seg])) cur[seg] = {}
      cur = cur[seg] as Record<string, unknown>
    }
    cur[segs[segs.length - 1]] = value
  }
}

export function buildAppearancePreset(pref: AppearancePreference, now = new Date()): AppearancePresetFile {
  const sanitized = stripLocalAssets(sanitizePreference(pref))
  const isolated = sanitized.isolated
  const preference = { ...sanitized }
  delete preference.isolated
  if (isolated && Object.keys(isolated).length) {
    applyIsolated(preference as unknown as Record<string, unknown>, isolated)
  }
  return {
    kind: APPEARANCE_PRESET_KIND,
    schemaVersion: APPEARANCE_SCHEMA_VERSION,
    exportedAt: now.toISOString(),
    preference,
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
  if (!data || typeof data !== 'object' || Array.isArray(data)) return { ok: false, error: 'invalid' }
  const file = data as Partial<AppearancePresetFile> & { preference?: { schemaVersion?: unknown } }
  if (file.kind !== APPEARANCE_PRESET_KIND) return { ok: false, error: 'invalid' }
  if (file.schemaVersion !== APPEARANCE_SCHEMA_VERSION) return { ok: false, error: 'invalid' }
  if (!file.preference || typeof file.preference !== 'object') return { ok: false, error: 'invalid' }
  if (file.preference.schemaVersion !== APPEARANCE_SCHEMA_VERSION) return { ok: false, error: 'invalid' }
  return { ok: true, preference: stripLocalAssets(sanitizePreference(file.preference)) }
}
