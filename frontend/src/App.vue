<template>
  <router-view />
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useGlobalStore } from '@/store/modules/global'

const globalStore = useGlobalStore()

onMounted(() => {
  globalStore.applyAccentColor()
})
</script>
