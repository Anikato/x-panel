<template>
  <div class="app-main" :class="{ 'is-workbench': isWorkbench }">
    <router-view v-slot="{ Component }">
      <transition name="fade">
        <keep-alive :include="['FileManager', 'Terminal']">
          <component :is="Component" />
        </keep-alive>
      </transition>
    </router-view>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const isWorkbench = computed(() => route.meta?.template === 'workbench')
</script>

<style lang="scss" scoped>
.app-main {
  flex: 1;
  min-height: 0;
  padding: var(--xp-spacing, 24px);
  overflow-y: auto;
  background: transparent;
  font-family: var(--xp-font-family);

  &.is-workbench {
    padding: 12px 16px 0;
  }
}

@media (max-width: 1366px) {
  .app-main {
    padding: 16px;
  }
}
</style>
