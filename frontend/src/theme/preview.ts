import { clonePreference, emptyPreference, switchTheme } from './resolve.ts'
import {
  OVERRIDE_GROUPS,
  type AppearancePreference,
  type OverrideGroup,
  type ThemeId,
  type ThemeMode,
  type ThemeOverrides,
} from './types.ts'

export interface PreviewSession {
  committed: AppearancePreference
  draft: AppearancePreference
}

export function beginPreview(committed: AppearancePreference): PreviewSession {
  return {
    committed: clonePreference(committed),
    draft: clonePreference(committed),
  }
}

export function patchDraft(session: PreviewSession, patch: {
  mode?: ThemeMode
  reduceMotion?: boolean
  keepPersonalPrefsAcrossThemes?: boolean
  themeId?: ThemeId
  overrides?: ThemeOverrides
}): void {
  if (patch.mode) session.draft.mode = patch.mode
  if (patch.reduceMotion !== undefined) session.draft.reduceMotion = patch.reduceMotion
  if (patch.keepPersonalPrefsAcrossThemes !== undefined) {
    session.draft.keepPersonalPrefsAcrossThemes = patch.keepPersonalPrefsAcrossThemes
  }
  if (patch.themeId && patch.themeId !== session.draft.themeId) {
    session.draft = switchTheme(session.draft, patch.themeId)
  }
  if (patch.overrides) {
    const themeId = session.draft.themeId
    session.draft.overridesByTheme[themeId] = {
      ...(session.draft.overridesByTheme[themeId] || {}),
      ...patch.overrides,
    }
  }
}

export function applyPreview(session: PreviewSession): AppearancePreference {
  return clonePreference(session.draft)
}

export function cancelPreview(session: PreviewSession): AppearancePreference {
  return clonePreference(session.committed)
}

export function restoreGroup(session: PreviewSession, group: OverrideGroup): void {
  const themeId = session.draft.themeId
  const current = { ...(session.draft.overridesByTheme[themeId] || {}) }
  const isolated = { ...(session.draft.isolated || {}) }
  for (const key of OVERRIDE_GROUPS[group]) {
    delete (current as Record<string, unknown>)[key]
    const prefix = `overridesByTheme.${themeId}.${key}`
    for (const path of Object.keys(isolated)) {
      if (path === prefix || path.startsWith(`${prefix}.`)) delete isolated[path]
    }
  }
  session.draft.overridesByTheme[themeId] = current
  session.draft.isolated = Object.keys(isolated).length ? isolated : undefined
}

export function restoreThemeDefaults(session: PreviewSession): void {
  const themeId = session.draft.themeId
  session.draft.overridesByTheme[themeId] = {}
  const isolated = { ...(session.draft.isolated || {}) }
  const prefix = `overridesByTheme.${themeId}.`
  for (const path of Object.keys(isolated)) {
    if (path.startsWith(prefix)) delete isolated[path]
  }
  session.draft.isolated = Object.keys(isolated).length ? isolated : undefined
}

export function restoreAllDefaults(session: PreviewSession): void {
  session.draft = emptyPreference()
}
