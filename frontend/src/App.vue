<template>
  <el-config-provider :locale="zhCn">
    <div class="route-loading-bar" :class="{ active: routeLoading }" />
    <router-view />
  </el-config-provider>
</template>

<script setup lang="ts">
import { ref, watch, watchEffect, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import { useGlobalStore } from '@/store/modules/global'
import { useAppearanceStore } from '@/store/modules/appearance'

const globalStore = useGlobalStore()
const appearanceStore = useAppearanceStore()
const route = useRoute()
const { t } = useI18n()

watchEffect(() => {
  const panelName = globalStore.panelName || 'X-Panel'
  const titleKey = route.meta?.title as string | undefined
  const pageTitle = titleKey ? t(titleKey) : ''
  document.title = pageTitle ? `${pageTitle} - ${panelName}` : panelName
})

const router = useRouter()
const routeLoading = ref(false)
let loadingTimer: ReturnType<typeof setTimeout> | null = null
let routeLoadingResetTimer: ReturnType<typeof setTimeout> | null = null
let colorSchemeQuery: MediaQueryList | null = null
let motionQuery: MediaQueryList | null = null

const handleEnvChange = () => {
  appearanceStore.applyCurrent()
}

router.beforeEach((_to, _from, next) => {
  routeLoading.value = true
  if (loadingTimer) clearTimeout(loadingTimer)
  if (routeLoadingResetTimer) clearTimeout(routeLoadingResetTimer)
  loadingTimer = setTimeout(() => { routeLoading.value = false }, 8000)
  next()
})

router.afterEach(() => {
  if (loadingTimer) { clearTimeout(loadingTimer); loadingTimer = null }
  if (routeLoadingResetTimer) clearTimeout(routeLoadingResetTimer)
  routeLoadingResetTimer = setTimeout(() => { routeLoading.value = false }, 150)
})

watch(() => appearanceStore.preference, () => {
  appearanceStore.applyCurrent()
}, { deep: true })

onMounted(async () => {
  appearanceStore.applyCurrent()
  globalStore.loadPanelNameFromBackend()

  colorSchemeQuery = window.matchMedia('(prefers-color-scheme: dark)')
  motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
  colorSchemeQuery.addEventListener('change', handleEnvChange)
  motionQuery.addEventListener('change', handleEnvChange)
})

onUnmounted(() => {
  if (loadingTimer) clearTimeout(loadingTimer)
  if (routeLoadingResetTimer) clearTimeout(routeLoadingResetTimer)
  colorSchemeQuery?.removeEventListener('change', handleEnvChange)
  motionQuery?.removeEventListener('change', handleEnvChange)
})
</script>

<style scoped>
.route-loading-bar {
  position: fixed;
  top: 0;
  left: 0;
  height: 2px;
  width: 100%;
  background: var(--xp-accent, #7AA2FF);
  z-index: 99999;
  pointer-events: none;
  transform-origin: left;
  transform: scaleX(0);
  opacity: 0;
  transition: transform var(--xp-motion-menu, 140ms) var(--xp-motion-ease, cubic-bezier(0.2, 0, 0, 1)),
    opacity var(--xp-motion-hover, 100ms) var(--xp-motion-ease, cubic-bezier(0.2, 0, 0, 1));
}
.route-loading-bar.active {
  opacity: 1;
  transform: scaleX(0.7);
  transition: transform 2s cubic-bezier(0.1, 0.5, 0.3, 1), opacity 0.15s;
}
</style>
