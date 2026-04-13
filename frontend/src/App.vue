<template>
  <div class="route-loading-bar" :class="{ active: routeLoading }" />
  <router-view />
</template>

<script setup lang="ts">
import { ref, watch, watchEffect, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useGlobalStore } from '@/store/modules/global'
import type { ThemeMode } from '@/store/modules/global'
import { applyAccentPalette, getPresetByKey, generatePaletteFromHex } from '@/utils/accent-colors'
import { applyAppearance } from '@/utils/appearance'
import { getSettingInfo } from '@/api/modules/setting'

const globalStore = useGlobalStore()
const route = useRoute()
const { t } = useI18n()

watchEffect(() => {
  const panelName = globalStore.panelName || 'X-Panel'
  const titleKey = route.meta?.title as string | undefined
  const pageTitle = titleKey ? t(titleKey) : ''
  document.title = pageTitle ? `${pageTitle} - ${panelName}` : panelName
})

const applyTheme = (mode: ThemeMode) => {
  let isDark: boolean
  if (mode === 'auto') {
    isDark = window.matchMedia('(prefers-color-scheme: dark)').matches
  } else {
    isDark = mode === 'dark'
  }
  document.documentElement.classList.toggle('dark', isDark)
}

const applyAccent = () => {
  const key = globalStore.accentKey
  if (key === 'custom' && globalStore.accentCustom) {
    applyAccentPalette(generatePaletteFromHex(globalStore.accentCustom))
  } else {
    const preset = getPresetByKey(key)
    if (preset) applyAccentPalette(preset)
  }
}

const applyAllAppearance = () => {
  applyAppearance({
    bgPreset: globalStore.bgPreset,
    uiFont: globalStore.uiFont,
    uiDensity: globalStore.uiDensity,
    borderRadiusPreset: globalStore.borderRadiusPreset,
    reduceMotion: globalStore.reduceMotion,
    cardBorderStyle: globalStore.cardBorderStyle,
    sidebarWidth: globalStore.sidebarWidth,
    accentKey: globalStore.accentKey,
    accentCustom: globalStore.accentCustom,
  })
}

watch(() => globalStore.theme, (mode) => applyTheme(mode))
watch(() => globalStore.accentKey, () => { applyAccent(); applyAllAppearance() })
watch(() => globalStore.accentCustom, () => { applyAccent(); applyAllAppearance() })

const allAppearanceKeys = [
  () => globalStore.bgPreset,
  () => globalStore.uiFont,
  () => globalStore.uiDensity,
  () => globalStore.borderRadiusPreset,
  () => globalStore.reduceMotion,
  () => globalStore.cardBorderStyle,
  () => globalStore.sidebarWidth,
  () => globalStore.termTheme,
  () => globalStore.termFont,
  () => globalStore.termFontSize,
  () => globalStore.termBgOpacity,
  () => globalStore.theme,
  () => globalStore.accentKey,
  () => globalStore.accentCustom,
  () => globalStore.showServerClock,
  () => globalStore.dashboardRefreshInterval,
] as const

let syncTimer: ReturnType<typeof setTimeout> | null = null
const debouncedSync = () => {
  if (syncTimer) clearTimeout(syncTimer)
  syncTimer = setTimeout(() => { globalStore.syncAppearanceToBackend() }, 1500)
}

for (const getter of allAppearanceKeys) {
  watch(getter, () => {
    applyAllAppearance()
    debouncedSync()
  })
}

const router = useRouter()
const routeLoading = ref(false)
let loadingTimer: ReturnType<typeof setTimeout> | null = null

router.beforeEach((_to, _from, next) => {
  routeLoading.value = true
  if (loadingTimer) clearTimeout(loadingTimer)
  loadingTimer = setTimeout(() => { routeLoading.value = false }, 8000)
  next()
})

router.afterEach(() => {
  if (loadingTimer) { clearTimeout(loadingTimer); loadingTimer = null }
  setTimeout(() => { routeLoading.value = false }, 150)
})

onMounted(async () => {
  applyTheme(globalStore.theme)
  applyAccent()
  applyAllAppearance()

  // 从后端获取面板名称
  try {
    const res = await getSettingInfo()
    if (res.data?.panelName) {
      globalStore.setPanelName(res.data.panelName)
    }
  } catch { /* backend not ready */ }

  const mq = window.matchMedia('(prefers-color-scheme: dark)')
  mq.addEventListener('change', () => {
    if (globalStore.theme === 'auto') applyTheme('auto')
  })
})

onUnmounted(() => {
  if (loadingTimer) clearTimeout(loadingTimer)
})
</script>

<style scoped>
.route-loading-bar {
  position: fixed;
  top: 0;
  left: 0;
  height: 2px;
  width: 100%;
  background: var(--xp-accent, #41FB44);
  z-index: 99999;
  pointer-events: none;
  transform-origin: left;
  transform: scaleX(0);
  opacity: 0;
  transition: transform 0.3s ease, opacity 0.2s ease 0.15s;
}
.route-loading-bar.active {
  opacity: 1;
  transform: scaleX(0.7);
  transition: transform 2s cubic-bezier(0.1, 0.5, 0.3, 1), opacity 0.15s;
}
</style>
