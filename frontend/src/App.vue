<template>
  <router-view />
</template>

<script setup lang="ts">
import { watch, onMounted } from 'vue'
import { useGlobalStore } from '@/store/modules/global'
import type { ThemeMode } from '@/store/modules/global'
import { applyAccentPalette, getPresetByKey, generatePaletteFromHex } from '@/utils/accent-colors'
import { applyAppearance } from '@/utils/appearance'

const globalStore = useGlobalStore()

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

const appearanceKeys = [
  () => globalStore.bgPreset,
  () => globalStore.uiFont,
  () => globalStore.uiDensity,
  () => globalStore.borderRadiusPreset,
  () => globalStore.reduceMotion,
  () => globalStore.cardBorderStyle,
  () => globalStore.sidebarWidth,
] as const

for (const getter of appearanceKeys) {
  watch(getter, () => applyAllAppearance())
}

onMounted(() => {
  applyTheme(globalStore.theme)
  applyAccent()
  applyAllAppearance()

  const mq = window.matchMedia('(prefers-color-scheme: dark)')
  mq.addEventListener('change', () => {
    if (globalStore.theme === 'auto') applyTheme('auto')
  })
})
</script>
