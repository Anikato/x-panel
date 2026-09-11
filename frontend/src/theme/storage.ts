import { hydrateAppearance, sanitizePreference } from './migrate.ts'
import { type AppearancePreference, type LegacyAppearance } from './types.ts'

export const APPEARANCE_STORAGE_KEY = 'xp-appearance-v1'
export const PINIA_GLOBAL_KEY = 'global'

export function readStoredV1(): unknown {
  if (typeof localStorage === 'undefined') return null
  try {
    const raw = localStorage.getItem(APPEARANCE_STORAGE_KEY)
    return raw ? JSON.parse(raw) : null
  } catch {
    return null
  }
}

export function readLocalPreference(): AppearancePreference {
  const stored = readStoredV1()
  const result = hydrateAppearance({
    localV1: stored,
    localLegacy: readLegacyFromPinia(),
  })
  if (result.source === 'local-legacy') writeLocalPreference(result.preference)
  return result.preference
}

export function writeLocalPreference(pref: AppearancePreference): boolean {
  if (typeof localStorage === 'undefined') return false
  try {
    localStorage.setItem(APPEARANCE_STORAGE_KEY, JSON.stringify(sanitizePreference(pref)))
    return true
  } catch {
    return false
  }
}

export function readLegacyFromPinia(): LegacyAppearance | null {
  if (typeof localStorage === 'undefined') return null
  try {
    const raw = localStorage.getItem(PINIA_GLOBAL_KEY)
    return raw ? JSON.parse(raw) as LegacyAppearance : null
  } catch {
    return null
  }
}
