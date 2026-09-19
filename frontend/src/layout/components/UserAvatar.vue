<template>
  <svg class="user-avatar-mark" viewBox="0 0 32 32" aria-hidden="true">
    <g v-if="index === 0">
      <circle cx="16" cy="16" r="12" fill="none" stroke="currentColor" stroke-width="2.4" />
      <circle cx="16" cy="16" r="6" fill="currentColor" />
    </g>
    <g v-else-if="index === 1">
      <path d="M16 4a12 12 0 1 1 0 24" fill="currentColor" opacity=".35" />
      <path d="M16 4a12 12 0 0 1 0 24Z" fill="currentColor" />
    </g>
    <g v-else-if="index === 2">
      <rect x="6" y="6" width="20" height="20" rx="3" transform="rotate(45 16 16)" fill="currentColor" />
    </g>
    <g v-else-if="index === 3">
      <circle cx="11" cy="11" r="4" fill="currentColor" />
      <circle cx="21" cy="11" r="4" fill="currentColor" opacity=".55" />
      <circle cx="11" cy="21" r="4" fill="currentColor" opacity=".55" />
      <circle cx="21" cy="21" r="4" fill="currentColor" />
    </g>
    <g v-else-if="index === 4">
      <path d="M6 22 16 6l10 16Z" fill="currentColor" />
      <path d="M10 22h12l-6-10Z" fill="var(--xp-on-accent, #fff)" opacity=".28" />
    </g>
    <g v-else-if="index === 5">
      <path d="M16 4 28 12v8L16 28 4 20v-8Z" fill="currentColor" />
    </g>
    <g v-else-if="index === 6">
      <circle cx="16" cy="16" r="12" fill="currentColor" opacity=".28" />
      <path d="M6 20c3-6 7-9 10-9s7 3 10 9" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" />
      <circle cx="12.5" cy="13" r="1.6" fill="currentColor" />
      <circle cx="19.5" cy="13" r="1.6" fill="currentColor" />
    </g>
    <g v-else>
      <rect x="6" y="6" width="8" height="8" rx="1.5" fill="currentColor" />
      <rect x="18" y="6" width="8" height="8" rx="1.5" fill="currentColor" opacity=".45" />
      <rect x="6" y="18" width="8" height="8" rx="1.5" fill="currentColor" opacity=".45" />
      <rect x="18" y="18" width="8" height="8" rx="1.5" fill="currentColor" />
    </g>
  </svg>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { avatarIndex } from '../user-avatar.ts'

const props = defineProps<{ name?: string }>()
const index = computed(() => avatarIndex(props.name || ''))
</script>

<style scoped>
.user-avatar-mark {
  display: block;
  width: 18px;
  height: 18px;
}
</style>
