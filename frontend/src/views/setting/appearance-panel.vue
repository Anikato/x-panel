<template>
  <div class="appearance-panel">
    <div class="appearance-toolbar">
      <div>
        <div class="appearance-status">{{ statusLabel }}</div>
        <div class="appearance-hint">{{ t('setting.appearanceDraftHint') }}</div>
        <div v-if="hasUnapplied" class="appearance-dirty">{{ t('setting.appearanceUnapplied') }}</div>
      </div>
      <div class="appearance-actions">
        <el-button @click="exportPreset">{{ t('setting.appearanceExport') }}</el-button>
        <el-button @click="pickPresetFile">{{ t('setting.appearanceImport') }}</el-button>
        <el-button @click="cancel">{{ t('commons.cancel') }}</el-button>
        <el-button @click="appearance.restoreTheme()">{{ t('setting.restoreThemeDefaults') }}</el-button>
        <el-button @click="appearance.restoreAll()">{{ t('setting.restoreAllDefaults') }}</el-button>
        <el-button type="primary" :loading="saving" @click="apply">{{ t('setting.applyAppearance') }}</el-button>
        <input ref="presetFileRef" type="file" accept="application/json,.json" class="wallpaper-file" @change="onPresetFile" />
      </div>
    </div>

    <section class="appearance-group">
      <h3 class="appearance-group-title">{{ t('setting.appearanceGroupTheme') }}</h3>
      <div class="theme-grid">
        <button
          v-for="theme in themes"
          :key="theme.id"
          type="button"
          class="theme-card"
          :class="{ active: preference.themeId === theme.id }"
          @click="appearance.patch({ themeId: theme.id })"
        >
          <div class="theme-swatches">
            <span :style="{ background: theme.modes.dark.colors.bgBase }" />
            <span :style="{ background: theme.modes.dark.colors.bgSurface }" />
            <span :style="{ background: theme.modes.light.colors.bgSurface }" />
            <span :style="{ background: accentOf(theme) }" />
          </div>
          <strong>{{ theme.name }}</strong>
          <span>{{ theme.id === 'atelier' ? t('setting.themeAtelierDesc') : t('setting.themeLumenDesc') }}</span>
        </button>
      </div>

      <div class="xp-setting-row">
        <span class="appearance-label">{{ t('setting.themeMode') }}</span>
        <el-radio-group :model-value="preference.mode" @change="(val: ThemeMode) => appearance.patch({ mode: val })">
          <el-radio-button value="dark">{{ t('header.themeDark') }}</el-radio-button>
          <el-radio-button value="light">{{ t('header.themeLight') }}</el-radio-button>
          <el-radio-button value="auto">{{ t('header.themeAuto') }}</el-radio-button>
        </el-radio-group>
      </div>

      <div class="xp-setting-stack">
        <span class="appearance-label">{{ t('header.accentColor') }}</span>
        <div class="accent-picker">
          <div class="accent-grid-large">
            <button
              v-for="preset in ACCENT_PRESETS"
              :key="preset.key"
              type="button"
              class="accent-swatch-large"
              :class="{ active: current.accentKey === preset.key }"
              :style="{ background: preset.primary }"
              :aria-label="preset.name"
              @click="appearance.patch({ overrides: { accentKey: preset.key, accentCustom: '' } })"
            >
              <el-icon v-if="current.accentKey === preset.key" :size="14"><Check /></el-icon>
            </button>
          </div>
          <label class="accent-custom">
            <span>{{ t('setting.accentCustom') }}</span>
            <input
              type="color"
              class="accent-custom-input"
              :value="current.accentCustom || '#7AA2FF'"
              :aria-label="t('setting.accentCustom')"
              @input="onCustomAccent"
            />
          </label>
        </div>
      </div>
    </section>

    <section class="appearance-group">
      <h3 class="appearance-group-title">{{ t('setting.appearanceGroupInterface') }}</h3>
      <div class="xp-setting-row">
        <span class="appearance-label">{{ t('setting.uiFont') }}</span>
        <el-select :model-value="current.uiFont" class="xp-select-md" @change="(val: UiFont) => appearance.setOverride('uiFont', val)">
          <el-option v-for="f in FONT_PRESETS" :key="f.key" :label="f.name" :value="f.key" />
        </el-select>
      </div>

      <div class="xp-setting-row">
        <span class="appearance-label">{{ t('setting.uiDensity') }}</span>
        <div>
          <el-radio-group :model-value="current.density" @change="(val: Density) => appearance.setOverride('density', val)">
            <el-radio-button value="compact">{{ t('setting.densityCompact') }}</el-radio-button>
            <el-radio-button value="default">{{ t('setting.densityDefault') }}</el-radio-button>
            <el-radio-button value="comfortable">{{ t('setting.densityComfortable') }}</el-radio-button>
          </el-radio-group>
          <el-button link type="primary" @click="appearance.restoreGroup('common')">{{ t('setting.restoreGroup') }}</el-button>
        </div>
      </div>

      <div class="xp-setting-row">
        <span class="appearance-label">{{ t('setting.sidebarWidth') }}</span>
        <el-radio-group :model-value="current.sidebarWidth" @change="(val: SidebarWidth) => appearance.setOverride('sidebarWidth', val)">
          <el-radio-button value="narrow">{{ t('setting.sidebarNarrow') }}</el-radio-button>
          <el-radio-button value="default">{{ t('setting.sidebarDefault') }}</el-radio-button>
          <el-radio-button value="wide">{{ t('setting.sidebarWide') }}</el-radio-button>
        </el-radio-group>
      </div>

      <div class="xp-setting-row">
        <span class="appearance-label">{{ t('setting.transparency') }}</span>
        <div>
          <el-switch :model-value="current.transparency" @change="onTransparency" />
          <p class="setting-inline-hint">{{ t('setting.transparencyHint') }}</p>
        </div>
      </div>

      <div class="xp-setting-stack">
        <span class="appearance-label">{{ t('setting.chromeTexture') }}</span>
        <div class="term-theme-grid">
          <button
            v-for="key in CHROME_TEXTURES"
            :key="key"
            type="button"
            class="chrome-wall-swatch"
            :class="{ active: current.chromeTexture === key }"
            :data-chrome-texture="key"
            @click="appearance.setOverride('chromeTexture', key)"
          >
            <span>{{ t(`setting.chromeTextureNames.${key}`) }}</span>
          </button>
        </div>
        <p class="setting-inline-hint">{{ t('setting.chromeTextureHint') }}</p>
      </div>

      <div class="xp-setting-stack">
        <span class="appearance-label">{{ t('setting.chromeImage') }}</span>
        <div class="wallpaper-field">
          <el-input
            :model-value="current.chromeImageUrl"
            :placeholder="t('setting.chromeImageUrlPlaceholder')"
            clearable
            @change="onChromeUrl"
          />
          <div class="wallpaper-actions">
            <el-button @click="pickChromeFile">{{ t('setting.wallpaperUpload') }}</el-button>
            <el-button v-if="hasChromeImage" link type="primary" @click="clearChromeImage">{{ t('setting.wallpaperClear') }}</el-button>
          </div>
          <input ref="chromeFileRef" type="file" accept="image/*" class="wallpaper-file" @change="onChromeFile" />
          <p class="setting-inline-hint">{{ t('setting.chromeImageHint') }}</p>
        </div>
      </div>

      <div class="xp-setting-row">
        <span class="appearance-label">{{ t('setting.reduceMotion') }}</span>
        <el-switch :model-value="preference.reduceMotion" @change="(val: boolean) => appearance.patch({ reduceMotion: val })" />
      </div>

      <div class="xp-setting-row">
        <span class="appearance-label">{{ t('setting.keepPersonalPrefs') }}</span>
        <el-switch :model-value="preference.keepPersonalPrefsAcrossThemes" @change="(val: boolean) => appearance.patch({ keepPersonalPrefsAcrossThemes: val })" />
      </div>
    </section>

    <section class="appearance-group">
    <el-collapse class="appearance-advanced">
      <el-collapse-item :title="t('setting.advancedAppearance')" name="advanced">
        <div class="xp-setting-row">
          <span class="appearance-label">{{ t('setting.cardVariant') }}</span>
          <el-radio-group :model-value="current.card" @change="(val: CardVariant) => appearance.setOverride('card', val)">
            <el-radio-button value="flat">{{ t('setting.cardFlat') }}</el-radio-button>
            <el-radio-button value="outline">{{ t('setting.cardOutline') }}</el-radio-button>
            <el-radio-button value="raised">{{ t('setting.cardRaised') }}</el-radio-button>
          </el-radio-group>
        </div>
        <div class="xp-setting-row">
          <span class="appearance-label">{{ t('setting.sidebarVariant') }}</span>
          <el-radio-group :model-value="current.sidebarVariant" @change="(val: SidebarVariant) => appearance.setOverride('sidebarVariant', val)">
            <el-radio-button value="marker">{{ t('setting.sidebarMarker') }}</el-radio-button>
            <el-radio-button value="block">{{ t('setting.sidebarBlock') }}</el-radio-button>
          </el-radio-group>
        </div>
        <div class="xp-setting-row">
          <span class="appearance-label">{{ t('setting.subnavVariant') }}</span>
          <div>
            <el-radio-group :model-value="current.subnav" @change="(val: SubnavVariant) => appearance.setOverride('subnav', val)">
              <el-radio-button value="line">{{ t('setting.subnavLine') }}</el-radio-button>
              <el-radio-button value="block">{{ t('setting.subnavBlock') }}</el-radio-button>
              <el-radio-button value="pill">{{ t('setting.subnavPill') }}</el-radio-button>
            </el-radio-group>
            <p class="setting-inline-hint">{{ t('setting.subnavHint') }}</p>
          </div>
        </div>
        <div class="xp-setting-row">
          <span class="appearance-label">{{ t('setting.iconSet') }}</span>
          <el-radio-group :model-value="current.iconSet" @change="(val: IconSet) => appearance.setOverride('iconSet', val)">
            <el-radio-button value="outline">{{ t('setting.iconOutline') }}</el-radio-button>
            <el-radio-button value="solid">{{ t('setting.iconSolid') }}</el-radio-button>
          </el-radio-group>
        </div>
        <div class="xp-setting-row">
          <span class="appearance-label">{{ t('setting.borderRadius') }}</span>
          <div>
            <el-radio-group :model-value="current.radius" @change="(val: RadiusPreset) => appearance.setOverride('radius', val)">
              <el-radio-button value="sharp">{{ t('setting.radiusSharp') }}</el-radio-button>
              <el-radio-button value="default">{{ t('setting.radiusDefault') }}</el-radio-button>
              <el-radio-button value="rounded">{{ t('setting.radiusRounded') }}</el-radio-button>
            </el-radio-group>
            <p class="setting-inline-hint">{{ t('setting.radiusHint') }}</p>
          </div>
        </div>
        <div v-if="preference.themeId === 'atelier'" class="xp-setting-row">
          <span class="appearance-label">{{ t('setting.darkSurface') }}</span>
          <div class="accent-grid-large">
            <el-tooltip v-for="bg in SURFACE_PRESET_DEFS" :key="bg.key" :content="t(`setting.bgPresetNames.${bg.key}`)" placement="top">
              <button
                type="button"
                class="bg-swatch"
                :class="{ active: (preference.overridesByTheme.atelier?.surfacePreset || 'graphite') === bg.key }"
                :style="{ background: bg.preview }"
                @click="appearance.setOverride('surfacePreset', bg.key)"
              />
            </el-tooltip>
          </div>
        </div>
        <el-button link type="primary" @click="appearance.restoreGroup('variants')">{{ t('setting.restoreVariants') }}</el-button>
      </el-collapse-item>
    </el-collapse>
    </section>

    <section class="appearance-group">
      <h3 class="appearance-group-title">{{ t('setting.appearanceGroupTerminal') }}</h3>
      <div class="xp-setting-stack">
        <span class="appearance-label">{{ t('setting.termTheme') }}</span>
        <div class="term-theme-grid">
          <button
            v-for="tt in TERMINAL_THEME_PRESETS"
            :key="tt.key"
            type="button"
            class="term-theme-swatch"
            :class="{ active: current.termTheme === tt.key }"
            @click="appearance.setOverride('termTheme', tt.key)"
          >
            <div class="term-preview" :style="{ background: tt.theme.background, color: tt.theme.foreground }">
              <span :style="{ color: tt.theme.green }">$</span>
              <span :style="{ color: tt.theme.cyan }"> ls</span>
            </div>
            <span class="term-theme-name">{{ tt.name }}</span>
          </button>
        </div>
      </div>
      <div class="xp-setting-row">
        <span class="appearance-label">{{ t('setting.termFont') }}</span>
        <el-select :model-value="current.termFont" class="xp-select-md" @change="(val: string) => appearance.setOverride('termFont', val)">
          <el-option v-for="f in TERMINAL_FONT_PRESETS" :key="f.key" :label="f.name" :value="f.key" />
        </el-select>
      </div>
      <div class="xp-setting-row">
        <span class="appearance-label">{{ t('setting.termFontSize') }}</span>
        <el-slider :model-value="current.termFontSize" :min="10" :max="24" :step="1" show-input class="xp-slider-md" @input="(val: number) => appearance.setOverride('termFontSize', val)" />
      </div>
      <div class="xp-setting-row">
        <span class="appearance-label">{{ t('setting.termFollowChrome') }}</span>
        <div>
          <el-switch :model-value="current.termFollowChrome" @change="onTermFollow" />
          <p class="setting-inline-hint">{{ t('setting.termFollowChromeHint') }}</p>
        </div>
      </div>
      <template v-if="!current.termFollowChrome">
        <div class="xp-setting-stack">
          <span class="appearance-label">{{ t('setting.termImage') }}</span>
          <div class="wallpaper-field">
            <el-input
              :model-value="current.termImageUrl"
              :placeholder="t('setting.chromeImageUrlPlaceholder')"
              clearable
              @change="onTermUrl"
            />
            <div class="wallpaper-actions">
              <el-button @click="pickTermFile">{{ t('setting.wallpaperUpload') }}</el-button>
              <el-button v-if="hasTermImage" link type="primary" @click="clearTermImage">{{ t('setting.wallpaperClear') }}</el-button>
            </div>
            <input ref="termFileRef" type="file" accept="image/*" class="wallpaper-file" @change="onTermFile" />
          </div>
        </div>
      </template>
      <div class="xp-setting-row">
        <span class="appearance-label">{{ t('setting.termBgOpacity') }}</span>
        <div>
          <el-slider :model-value="current.termBgOpacity" :min="0.3" :max="1" :step="0.05" show-input class="xp-slider-md" @input="(val: number) => appearance.setOverride('termBgOpacity', val)" />
          <p class="setting-inline-hint">{{ t('setting.termBgOpacityHint') }}</p>
        </div>
      </div>
    </section>

    <section class="appearance-group workspace-prefs">
      <h3 class="appearance-group-title">{{ t('setting.appearanceGroupWorkspace') }}</h3>
      <p class="setting-inline-hint">{{ t('setting.workspacePrefsHint') }}</p>
      <div class="xp-setting-row">
        <span class="appearance-label">{{ t('setting.showServerClock') }}</span>
        <el-switch v-model="globalStore.showServerClock" />
      </div>
      <div class="xp-setting-row">
        <span class="appearance-label">{{ t('setting.dashboardRefresh') }}</span>
        <el-select v-model="globalStore.dashboardRefreshInterval" class="xp-select-sm">
          <el-option :label="'2 ' + t('setting.seconds')" :value="2000" />
          <el-option :label="'5 ' + t('setting.seconds')" :value="5000" />
          <el-option :label="'10 ' + t('setting.seconds')" :value="10000" />
          <el-option :label="'30 ' + t('setting.seconds')" :value="30000" />
          <el-option :label="t('setting.disableAutoRefresh')" :value="0" />
        </el-select>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Check } from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'
import { ACCENT_PRESETS, getPresetByKey } from '@/utils/accent-colors'
import { FONT_PRESETS } from '@/utils/appearance'
import { TERMINAL_FONT_PRESETS, TERMINAL_THEME_PRESETS } from '@/utils/terminal-theme'
import {
  listThemes,
  SURFACE_PRESET_DEFS,
  getTheme,
  CHROME_TEXTURES,
  fileToWallpaperDataUrl,
  hasWallpaperDraft,
  sanitizeWallpaperUrl,
  setDraftWallpaper,
  wallpaperDraftGen,
  wallpaperJobMatches,
} from '@/theme'
import type {
  CardVariant,
  ChromeTexture,
  Density,
  IconSet,
  RadiusPreset,
  SidebarVariant,
  SidebarWidth,
  SubnavVariant,
  ThemeDefinition,
  ThemeMode,
  UiFont,
} from '@/theme/types.ts'
import { useAppearanceStore } from '@/store/modules/appearance'
import { useGlobalStore } from '@/store/modules/global'

const { t } = useI18n()
const appearance = useAppearanceStore()
const globalStore = useGlobalStore()
const saving = ref(false)
const chromeFileRef = ref<HTMLInputElement | null>(null)
const termFileRef = ref<HTMLInputElement | null>(null)
const presetFileRef = ref<HTMLInputElement | null>(null)
const themes = listThemes()
const preference = computed(() => appearance.preference)
const resolved = computed(() => appearance.resolved)

const current = computed(() => {
  const theme = getTheme(preference.value.themeId)
  const over = preference.value.overridesByTheme[preference.value.themeId] || {}
  return {
    accentKey: over.accentKey || resolved.value.accent.key,
    accentCustom: over.accentCustom || resolved.value.accent.custom,
    uiFont: over.uiFont || theme.defaults.uiFont,
    density: over.density || theme.defaults.density,
    sidebarWidth: over.sidebarWidth || theme.defaults.sidebarWidth,
    transparency: over.transparency ?? theme.defaults.transparency,
    chromeTexture: (over.chromeTexture || resolved.value.chromeTexture) as ChromeTexture,
    chromeImageMode: over.chromeImageMode || resolved.value.chromeImageMode,
    chromeImageUrl: over.chromeImageUrl || resolved.value.chromeImageUrl,
    card: over.card || theme.variants.card,
    sidebarVariant: over.sidebarVariant || theme.variants.sidebar,
    subnav: over.subnav || theme.variants.subnav,
    iconSet: over.iconSet || theme.iconSet,
    radius: over.radius || theme.defaults.radius,
    termTheme: over.termTheme || resolved.value.terminalTheme,
    termFont: over.termFont || theme.defaults.termFont,
    termFontSize: over.termFontSize || theme.defaults.termFontSize,
    termBgOpacity: over.termBgOpacity ?? theme.defaults.termBgOpacity,
    termFollowChrome: resolved.value.termFollowChrome,
    termImageMode: over.termImageMode || resolved.value.termImageMode,
    termImageUrl: over.termImageUrl || resolved.value.termImageUrl,
  }
})

const hasChromeImage = computed(() => current.value.chromeImageMode !== 'none')
const hasTermImage = computed(() => current.value.termImageMode !== 'none')

const ensureTermSeeThrough = () => {
  if ((current.value.termBgOpacity ?? 1) >= 0.98) {
    appearance.setOverride('termBgOpacity', 0.82)
  }
}

const onTransparency = (val: boolean) => {
  appearance.setOverride('transparency', val)
  if (val && current.value.chromeTexture === 'none' && current.value.chromeImageMode === 'none') {
    appearance.setOverride('chromeTexture', 'starfield')
  }
}

const onTermFollow = (val: boolean) => {
  appearance.setOverride('termFollowChrome', val)
  if (val) ensureTermSeeThrough()
}

const onChromeUrl = (val: string) => {
  const url = sanitizeWallpaperUrl(val)
  if (val && !url) {
    ElMessage.warning(t('setting.wallpaperUrlInvalid'))
    return
  }
  appearance.patch({ overrides: { chromeImageMode: url ? 'url' : 'none', chromeImageUrl: url } })
  if (url) ensureTermSeeThrough()
}

const onTermUrl = (val: string) => {
  const url = sanitizeWallpaperUrl(val)
  if (val && !url) {
    ElMessage.warning(t('setting.wallpaperUrlInvalid'))
    return
  }
  appearance.patch({ overrides: { termImageMode: url ? 'url' : 'none', termImageUrl: url } })
  if (url) ensureTermSeeThrough()
}

const pickChromeFile = () => chromeFileRef.value?.click()
const pickTermFile = () => termFileRef.value?.click()

const persistWallpaper = async (kind: 'chrome' | 'term', file?: File | null) => {
  if (!file) return
  appearance.ensurePreview()
  const gen = wallpaperDraftGen()
  const themeId = preference.value.themeId
  try {
    const data = await fileToWallpaperDataUrl(file)
    if (!wallpaperJobMatches(gen, themeId, preference.value.themeId)) return
    setDraftWallpaper(kind, themeId, data)
    if (kind === 'chrome') {
      appearance.patch({ overrides: { chromeImageMode: 'upload', chromeImageUrl: '' } })
    } else {
      appearance.patch({ overrides: { termImageMode: 'upload', termImageUrl: '' } })
    }
    ensureTermSeeThrough()
  } catch (error) {
    const reason = error instanceof Error ? error.message : ''
    if (reason === 'too-large') ElMessage.error(t('setting.wallpaperTooLarge'))
    else if (reason === 'not-image') ElMessage.error(t('setting.wallpaperNotImage'))
    else ElMessage.error(t('setting.wallpaperUploadFailed'))
  }
}

const onChromeFile = (event: Event) => {
  const input = event.target as HTMLInputElement
  void persistWallpaper('chrome', input.files?.[0])
  input.value = ''
}

const onTermFile = (event: Event) => {
  const input = event.target as HTMLInputElement
  void persistWallpaper('term', input.files?.[0])
  input.value = ''
}

const clearChromeImage = () => {
  appearance.ensurePreview()
  setDraftWallpaper('chrome', preference.value.themeId, '')
  appearance.patch({ overrides: { chromeImageMode: 'none', chromeImageUrl: '' } })
}

const clearTermImage = () => {
  appearance.ensurePreview()
  setDraftWallpaper('term', preference.value.themeId, '')
  appearance.patch({ overrides: { termImageMode: 'none', termImageUrl: '' } })
}

const statusLabel = computed(() => {
  const name = resolved.value.themeName
  return resolved.value.customized ? `${name} · ${t('setting.customized')}` : name
})

const hasUnapplied = computed(() => {
  const session = appearance.session
  if (!appearance.previewing || !session) return false
  return JSON.stringify(session.draft) !== JSON.stringify(session.committed) || hasWallpaperDraft()
})

const accentOf = (theme: ThemeDefinition) => getPresetByKey(theme.modes.dark.accentKey)?.primary || '#7AA2FF'

const onCustomAccent = (event: Event) => {
  const hex = (event.target as HTMLInputElement).value
  appearance.patch({ overrides: { accentKey: 'custom', accentCustom: hex } })
}

onMounted(() => appearance.startPreview())
onBeforeUnmount(() => {
  if (appearance.previewing) appearance.cancelPreview()
})

const apply = async () => {
  saving.value = true
  const ok = await appearance.commitPreview()
  saving.value = false
  if (!ok) {
    ElMessage.error(t('setting.appearancePersistFailed'))
    return
  }
  if (appearance.persistError === 'remote') ElMessage.warning(t('setting.appearanceRemoteBackupFailed'))
  else ElMessage.success(t('setting.appearanceApplied'))
  appearance.startPreview()
}

const cancel = () => {
  appearance.cancelPreview()
  appearance.startPreview()
  ElMessage.info(t('setting.appearanceCancelled'))
}

const exportPreset = () => {
  const blob = new Blob([appearance.exportPresetJSON()], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = 'x-panel-appearance.json'
  link.click()
  URL.revokeObjectURL(url)
  ElMessage.success(t('setting.appearanceExportOk'))
}

const pickPresetFile = () => presetFileRef.value?.click()

const onPresetFile = async (event: Event) => {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  try {
    const text = await file.text()
    const ok = await appearance.importPresetJSON(text)
    if (!ok) {
      ElMessage.error(t('setting.appearanceImportBad'))
      return
    }
    appearance.startPreview()
    ElMessage.success(t('setting.appearanceImportOk'))
  } catch {
    ElMessage.error(t('setting.appearanceImportBad'))
  }
}
</script>

<style scoped lang="scss">
.appearance-panel {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.appearance-toolbar {
  position: sticky;
  top: 0;
  z-index: 6;
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin: -8px 0 12px;
  padding: 10px 0 12px;
  background: var(--xp-bg-surface);
  border-bottom: 1px solid var(--xp-border-light);
}

.appearance-status {
  color: var(--xp-text-primary);
  font-weight: 650;
}

.appearance-hint {
  margin-top: 4px;
  color: var(--xp-text-muted);
  font-size: 12px;
}

.appearance-dirty {
  margin-top: 4px;
  color: var(--xp-warning);
  font-size: 12px;
}

.appearance-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.theme-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 260px));
  gap: 12px;
  margin-bottom: 20px;
}

.theme-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px;
  color: var(--xp-text-secondary);
  text-align: left;
  background: var(--xp-bg-inset);
  border: 1px solid var(--xp-border);
  border-radius: var(--xp-radius);
  cursor: pointer;

  strong {
    color: var(--xp-text-primary);
  }

  &.active {
    border-color: var(--xp-accent);
    box-shadow: inset 0 0 0 1px var(--xp-accent);
  }
}

.theme-swatches {
  display: flex;
  gap: 6px;

  span {
    width: 22px;
    height: 22px;
    border: 1px solid var(--xp-border-light);
    border-radius: 999px;
  }
}

.appearance-advanced {
  margin: 8px 0 4px;
}

.accent-picker {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px 16px;
}

.accent-custom {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--xp-text-secondary);
  font-size: 13px;
}

.accent-custom-input {
  width: 36px;
  height: 28px;
  padding: 0;
  background: transparent;
  border: 1px solid var(--xp-border);
  border-radius: var(--xp-radius-sm);
  cursor: pointer;
}

.setting-inline-hint {
  margin: 8px 0 0;
  color: var(--xp-text-muted);
  font-size: 12px;
  line-height: 1.5;
}

.wallpaper-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
  width: min(100%, 520px);
}

.wallpaper-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.wallpaper-file {
  display: none;
}
</style>
