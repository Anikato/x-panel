<template>
  <div class="app-main">
    <router-view v-slot="{ Component }">
      <transition name="fade" mode="out-in">
        <component :is="Component" />
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
</style>
