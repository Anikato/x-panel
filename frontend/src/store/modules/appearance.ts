import { defineStore } from 'pinia'
import { getSettingInfo, updateSetting } from '@/api/modules/setting'
import {
  applyPreview,
  applyResolvedAppearance,
  beginPreview,
  cancelPreview,
  clonePreference,
  hydrateAppearance,
  patchDraft,
  readLocalPreference,
  readResolveEnv,
  readStoredV1,
  resolveAppearance,
  restoreAllDefaults,
  restoreGroup,
  restoreThemeDefaults,
  sanitizePreference,
  writeLocalPreference,
  type AppearancePreference,
  type OverrideGroup,
  type PreviewSession,
  type ResolvedAppearance,
  type ThemeId,
  type ThemeMode,
  type ThemeOverrides,
} from '@/theme'
import { readLegacyFromPinia } from '@/theme/storage.ts'
import {
  beginWallpaperDraft,
  cancelWallpaperDraft,
  clearWallpaperDraftMaps,
  flushWallpaperDraftToStorage,
  restoreWallpaperSnapshot,
  snapshotWallpaperDraftTargets,
} from '@/theme/wallpaper-store.ts'
import { useGlobalStore } from './global'

function currentEnv() {
  return readResolveEnv()
}

function mirrorRuntime(resolved: ResolvedAppearance) {
  const globalStore = useGlobalStore()
  globalStore.theme = resolved.modePreference
  globalStore.accentKey = resolved.accent.key
  globalStore.accentCustom = resolved.accent.custom
  globalStore.uiFont = resolved.uiFontKey
  globalStore.uiDensity = resolved.density
  globalStore.reduceMotion = resolved.reduceMotion
  globalStore.termTheme = resolved.terminalTheme
  globalStore.termFont = resolved.termFont
  globalStore.termFontSize = resolved.termFontSize
  globalStore.termBgOpacity = resolved.termBgOpacity
}

export const useAppearanceStore = defineStore('appearance', {
  state: () => ({
    preference: readLocalPreference() as AppearancePreference,
    previewing: false,
    persistError: '',
    lastSource: readStoredV1() ? 'local-v1' : 'default',
    session: null as PreviewSession | null,
  }),
  getters: {
    resolved(): ResolvedAppearance {
      return resolveAppearance(this.preference, currentEnv())
    },
  },
  actions: {
    applyCurrent() {
      const resolved = resolveAppearance(this.preference, currentEnv())
      applyResolvedAppearance(resolved)
      mirrorRuntime(resolved)
    },

    startPreview() {
      this.session = beginPreview(this.preference)
      this.previewing = true
      beginWallpaperDraft()
    },

    ensurePreview() {
      if (!this.previewing || !this.session) this.startPreview()
    },

    patch(patch: {
      mode?: ThemeMode
      reduceMotion?: boolean
      keepPersonalPrefsAcrossThemes?: boolean
      themeId?: ThemeId
      overrides?: ThemeOverrides
    }) {
      this.ensurePreview()
      if (!this.session) return
      patchDraft(this.session, patch)
      this.preference = clonePreference(this.session.draft)
      this.applyCurrent()
    },

    setOverride<K extends keyof ThemeOverrides>(key: K, value: ThemeOverrides[K]) {
      this.patch({ overrides: { [key]: value } as ThemeOverrides })
    },

    setLiveOverride<K extends keyof ThemeOverrides>(key: K, value: ThemeOverrides[K]) {
      if (this.previewing) {
        this.setOverride(key, value)
        return
      }
      const themeId = this.preference.themeId
      this.preference = sanitizePreference({
        ...this.preference,
        overridesByTheme: {
          ...this.preference.overridesByTheme,
          [themeId]: {
            ...(this.preference.overridesByTheme[themeId] || {}),
            [key]: value,
          },
        },
      })
      this.applyCurrent()
      void this.persist()
    },

    async commitPreview() {
      if (!this.session) return this.persist()
      const next = applyPreview(this.session)
      this.persistError = ''
      const imageSnap = snapshotWallpaperDraftTargets()
      try {
        flushWallpaperDraftToStorage()
      } catch {
        this.persistError = 'local'
        return false
      }
      if (!writeLocalPreference(next)) {
        restoreWallpaperSnapshot(imageSnap)
        this.persistError = 'local'
        return false
      }
      clearWallpaperDraftMaps()
      this.preference = next
      this.session = null
      this.previewing = false
      this.applyCurrent()
      try {
        await updateSetting({ key: 'AppearanceConfig', value: JSON.stringify(sanitizePreference(next)) })
      } catch {
        this.persistError = 'remote'
      }
      return true
    },

    cancelPreview() {
      if (this.session) this.preference = cancelPreview(this.session)
      this.session = null
      this.previewing = false
      cancelWallpaperDraft()
      this.applyCurrent()
    },

    restoreGroup(group: OverrideGroup) {
      this.ensurePreview()
      if (!this.session) return
      restoreGroup(this.session, group)
      this.preference = clonePreference(this.session.draft)
      this.applyCurrent()
    },

    restoreTheme() {
      this.ensurePreview()
      if (!this.session) return
      restoreThemeDefaults(this.session)
      this.preference = clonePreference(this.session.draft)
      this.applyCurrent()
    },

    restoreAll() {
      this.ensurePreview()
      if (!this.session) return
      restoreAllDefaults(this.session)
      this.preference = clonePreference(this.session.draft)
      this.applyCurrent()
    },

    applyImmediate(patch: Partial<AppearancePreference>) {
      this.preference = sanitizePreference({
        ...this.preference,
        ...patch,
        overridesByTheme: {
          ...this.preference.overridesByTheme,
          ...(patch.overridesByTheme || {}),
        },
      })
      if (this.session) this.session.draft = clonePreference(this.preference)
      this.applyCurrent()
      if (!this.previewing) void this.persist()
    },

    async persist() {
      this.persistError = ''
      const payload = sanitizePreference(this.preference)
      if (!writeLocalPreference(payload)) {
        this.persistError = 'local'
        return false
      }
      try {
        await updateSetting({ key: 'AppearanceConfig', value: JSON.stringify(payload) })
      } catch {
        this.persistError = 'remote'
      }
      return true
    },

    hydrateFromBackend(raw: unknown) {
      if (this.previewing) return
      const result = hydrateAppearance({
        localV1: readStoredV1(),
        localLegacy: readLegacyFromPinia(),
        backendRaw: raw,
        previous: this.preference,
      })
      this.preference = result.preference
      this.lastSource = result.source
      if (!result.warnings.some((item) => /schema/i.test(item))) {
        writeLocalPreference(this.preference)
      }
      this.applyCurrent()
    },

    async loadFromBackend() {
      try {
        const res = await getSettingInfo()
        const raw = res.data?.appearanceConfig || res.data?.AppearanceConfig
        this.hydrateFromBackend(raw)
      } catch {
        this.applyCurrent()
      }
    },
  },
})
