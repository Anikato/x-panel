<template>
  <div class="app-main">
    <router-view v-slot="{ Component }">
      <transition name="page-slide" mode="out-in">
        <component :is="Component" :key="$route.path" />
      </transition>
    </router-view>
  </div>
</template>

<script setup lang="ts"></script>

<style lang="scss" scoped>
.app-main {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
  background: var(--xp-bg-base);
}

.page-slide-enter-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.page-slide-leave-active {
  transition: opacity 0.15s ease;
}
.page-slide-enter-from {
  opacity: 0;
  transform: translateY(8px);
}
.page-slide-leave-to {
  opacity: 0;
}
</style>
