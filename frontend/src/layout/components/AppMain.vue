<template>
  <div class="app-main">
    <router-view v-slot="{ Component }">
      <transition name="fade" mode="out-in">
        <keep-alive :include="['FileManager', 'Terminal']">
          <component :is="Component" />
        </keep-alive>
      </transition>
    </router-view>
  </div>
</template>

<script setup lang="ts"></script>

<style lang="scss" scoped>
.app-main {
  flex: 1;
  padding: var(--xp-spacing, 20px);
  overflow-y: auto;
  background: var(--xp-bg-main-gradient, var(--xp-bg-base));
  font-family: var(--xp-font-family);
}
</style>
