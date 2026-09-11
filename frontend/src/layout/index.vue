<template>
  <div class="layout-container">
    <Sidebar />
    <div class="layout-main" :class="{ 'is-collapse': globalStore.menuCollapse }">
      <Header />
      <ModuleChrome>
        <AppMain />
      </ModuleChrome>
    </div>
    <FloatTerminal />
  </div>
</template>

<script setup lang="ts">
import Sidebar from './components/Sidebar.vue'
import Header from './components/Header.vue'
import AppMain from './components/AppMain.vue'
import ModuleChrome from './components/ModuleChrome.vue'
import FloatTerminal from './components/FloatTerminal.vue'
import { useGlobalStore } from '@/store/modules/global'
import { useFileTaskStore } from '@/store/modules/fileTask'
import { onMounted, onUnmounted } from 'vue'

const globalStore = useGlobalStore()
const fileTaskStore = useFileTaskStore()

const applyMobileMenuDefault = () => {
  if (window.innerWidth <= 900) {
    globalStore.menuCollapse = true
  }
}

onMounted(() => {
  fileTaskStore.init()
  applyMobileMenuDefault()
  window.addEventListener('resize', applyMobileMenuDefault)
})

onUnmounted(() => {
  window.removeEventListener('resize', applyMobileMenuDefault)
  fileTaskStore.stopPolling()
})
</script>

<style lang="scss" scoped>
.layout-container {
  display: flex;
  width: 100%;
  height: 100vh;
  overflow: hidden;
  background-color: var(--xp-bg-base);
}

.layout-main {
  position: relative;
  z-index: 1;
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
  margin-left: var(--xp-sidebar-width);
  overflow: hidden;
  transition: margin-left 180ms cubic-bezier(0.2, 0, 0, 1);

  &.is-collapse {
    margin-left: var(--xp-sidebar-collapse-width);
  }
}

@media (max-width: 900px) {
  .layout-main,
  .layout-main.is-collapse {
    margin-left: 0;
  }
}
</style>
