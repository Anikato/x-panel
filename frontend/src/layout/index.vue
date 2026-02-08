<template>
  <div class="layout-container">
    <Sidebar />
    <div class="layout-main" :class="{ 'is-collapse': globalStore.menuCollapse }">
      <Header />
      <AppMain />
    </div>
  </div>
</template>

<script setup lang="ts">
import Sidebar from './components/Sidebar.vue'
import Header from './components/Header.vue'
import AppMain from './components/AppMain.vue'
import { useGlobalStore } from '@/store/modules/global'

const globalStore = useGlobalStore()
</script>

<style lang="scss" scoped>
.layout-container {
  display: flex;
  height: 100vh;
  width: 100%;
  background: var(--xp-bg-base);
}

.layout-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  margin-left: var(--xp-sidebar-width);
  transition: margin-left 0.3s cubic-bezier(0.4, 0, 0.2, 1);

  &.is-collapse {
    margin-left: var(--xp-sidebar-collapse-width);
  }
}
</style>
