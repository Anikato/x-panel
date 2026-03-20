<template>
  <router-view />
</template>

<script setup lang="ts">
import { watch, onMounted } from 'vue'
import { useGlobalStore } from '@/store/modules/global'
import type { ThemeMode } from '@/store/modules/global'
import { applyAccentPalette, getPresetByKey, generatePaletteFromHex } from '@/utils/accent-colors'

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

watch(() => globalStore.theme, (mode) => applyTheme(mode))
watch(() => globalStore.accentKey, () => applyAccent())
watch(() => globalStore.accentCustom, () => applyAccent())

onMounted(() => {
  applyTheme(globalStore.theme)
  applyAccent()

  const mq = window.matchMedia('(prefers-color-scheme: dark)')
  mq.addEventListener('change', () => {
    if (globalStore.theme === 'auto') applyTheme('auto')
  })
})
</script>
