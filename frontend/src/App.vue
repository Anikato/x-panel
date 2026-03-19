<template>
  <router-view />
</template>

<script setup lang="ts">
import { watch, onMounted } from 'vue'
import { useGlobalStore } from '@/store/modules/global'
import type { ThemeMode } from '@/store/modules/global'

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

watch(() => globalStore.theme, (mode) => applyTheme(mode), { immediate: true })

onMounted(() => {
  applyTheme(globalStore.theme)

  const mq = window.matchMedia('(prefers-color-scheme: dark)')
  mq.addEventListener('change', () => {
    if (globalStore.theme === 'auto') applyTheme('auto')
  })
})
</script>
