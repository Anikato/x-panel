<template>
  <svg viewBox="0 0 24 24" :width="size" :height="size" fill="none" xmlns="http://www.w3.org/2000/svg">
    <!-- Folder icons -->
    <template v-if="isDir">
      <template v-if="specialDir === 'git'">
        <path d="M2 6V18C2 19.1 2.9 20 4 20H20C21.1 20 22 19.1 22 18V8C22 6.9 21.1 6 20 6H12L10 4H4C2.9 4 2 4.9 2 6Z" :fill="'#f05033'" fill-opacity="0.2" />
        <path d="M2 6V18C2 19.1 2.9 20 4 20H20C21.1 20 22 19.1 22 18V8C22 6.9 21.1 6 20 6H12L10 4H4C2.9 4 2 4.9 2 6Z" :stroke="'#f05033'" stroke-width="1.5" stroke-linejoin="round" />
      </template>
      <template v-else-if="specialDir === 'node_modules'">
        <path d="M2 6V18C2 19.1 2.9 20 4 20H20C21.1 20 22 19.1 22 18V8C22 6.9 21.1 6 20 6H12L10 4H4C2.9 4 2 4.9 2 6Z" fill="#8cc84b" fill-opacity="0.2" />
        <path d="M2 6V18C2 19.1 2.9 20 4 20H20C21.1 20 22 19.1 22 18V8C22 6.9 21.1 6 20 6H12L10 4H4C2.9 4 2 4.9 2 6Z" stroke="#8cc84b" stroke-width="1.5" stroke-linejoin="round" />
      </template>
      <template v-else-if="specialDir === 'conf'">
        <path d="M2 6V18C2 19.1 2.9 20 4 20H20C21.1 20 22 19.1 22 18V8C22 6.9 21.1 6 20 6H12L10 4H4C2.9 4 2 4.9 2 6Z" fill="#60a5fa" fill-opacity="0.2" />
        <path d="M2 6V18C2 19.1 2.9 20 4 20H20C21.1 20 22 19.1 22 18V8C22 6.9 21.1 6 20 6H12L10 4H4C2.9 4 2 4.9 2 6Z" stroke="#60a5fa" stroke-width="1.5" stroke-linejoin="round" />
      </template>
      <template v-else-if="specialDir === 'log'">
        <path d="M2 6V18C2 19.1 2.9 20 4 20H20C21.1 20 22 19.1 22 18V8C22 6.9 21.1 6 20 6H12L10 4H4C2.9 4 2 4.9 2 6Z" fill="#fbbf24" fill-opacity="0.2" />
        <path d="M2 6V18C2 19.1 2.9 20 4 20H20C21.1 20 22 19.1 22 18V8C22 6.9 21.1 6 20 6H12L10 4H4C2.9 4 2 4.9 2 6Z" stroke="#fbbf24" stroke-width="1.5" stroke-linejoin="round" />
      </template>
      <template v-else>
        <path d="M2 6V18C2 19.1 2.9 20 4 20H20C21.1 20 22 19.1 22 18V8C22 6.9 21.1 6 20 6H12L10 4H4C2.9 4 2 4.9 2 6Z" fill="#22d3ee" fill-opacity="0.15" />
        <path d="M2 6V18C2 19.1 2.9 20 4 20H20C21.1 20 22 19.1 22 18V8C22 6.9 21.1 6 20 6H12L10 4H4C2.9 4 2 4.9 2 6Z" stroke="#22d3ee" stroke-width="1.5" stroke-linejoin="round" />
      </template>
    </template>

    <!-- File icons -->
    <template v-else>
      <!-- Base file shape -->
      <path d="M6 2H14L20 8V20C20 21.1 19.1 22 18 22H6C4.9 22 4 21.1 4 20V4C4 2.9 4.9 2 6 2Z" :fill="bgColor" fill-opacity="0.12" />
      <path d="M6 2H14L20 8V20C20 21.1 19.1 22 18 22H6C4.9 22 4 21.1 4 20V4C4 2.9 4.9 2 6 2Z" :stroke="iconColor" stroke-width="1.5" stroke-linejoin="round" />
      <path d="M14 2V8H20" :stroke="iconColor" stroke-width="1.5" stroke-linejoin="round" />

      <!-- Extension label -->
      <text v-if="extLabel" x="12" y="17" text-anchor="middle" :fill="iconColor" font-size="6" font-weight="700" font-family="system-ui, sans-serif">{{ extLabel }}</text>
    </template>
  </svg>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  name: string
  isDir: boolean
  size?: number
}>()

const size = computed(() => props.size || 20)

const ext = computed(() => {
  if (props.isDir) return ''
  return (props.name.split('.').pop() || '').toLowerCase()
})

const specialDir = computed(() => {
  if (!props.isDir) return ''
  const n = props.name.toLowerCase()
  if (n === '.git' || n === 'git') return 'git'
  if (n === 'node_modules' || n === 'vendor') return 'node_modules'
  if (n === 'conf' || n === 'config' || n === 'etc' || n === '.config') return 'conf'
  if (n === 'log' || n === 'logs') return 'log'
  return ''
})

const colorMap: Record<string, string> = {
  // Go
  go: '#00ADD8',
  // JavaScript/TypeScript
  js: '#f7df1e', mjs: '#f7df1e', cjs: '#f7df1e',
  ts: '#3178c6', tsx: '#3178c6', mts: '#3178c6',
  jsx: '#61dafb',
  // Vue/React/Svelte
  vue: '#42b883', svelte: '#ff3e00',
  // Python
  py: '#3776ab', pyw: '#3776ab', pyi: '#3776ab',
  // Rust
  rs: '#dea584', toml: '#9c4221',
  // Ruby
  rb: '#cc342d', erb: '#cc342d',
  // PHP
  php: '#777bb4',
  // Java/Kotlin
  java: '#e76f00', kt: '#A97BFF', kts: '#A97BFF',
  // C/C++
  c: '#555555', h: '#555555', cpp: '#f34b7d', cc: '#f34b7d', cxx: '#f34b7d', hpp: '#f34b7d',
  // Shell
  sh: '#4eaa25', bash: '#4eaa25', zsh: '#4eaa25', fish: '#4eaa25',
  // Config
  json: '#f5a623', yaml: '#cb171e', yml: '#cb171e',
  xml: '#e44d26', ini: '#6d8086', conf: '#6d8086', cfg: '#6d8086',
  env: '#ecd53f',
  // Web
  html: '#e44d26', htm: '#e44d26', css: '#563d7c', scss: '#c6538c', less: '#1d365d', sass: '#c6538c',
  // Data
  sql: '#e38c00', db: '#e38c00', sqlite: '#e38c00',
  csv: '#237346', tsv: '#237346',
  // Markdown/Docs
  md: '#083fa1', mdx: '#083fa1', rst: '#083fa1', txt: '#6d8086',
  // Image
  png: '#a4c639', jpg: '#a4c639', jpeg: '#a4c639', gif: '#a4c639', svg: '#ffb300', webp: '#a4c639', ico: '#a4c639', bmp: '#a4c639',
  // Video
  mp4: '#ff6d00', avi: '#ff6d00', mkv: '#ff6d00', mov: '#ff6d00', wmv: '#ff6d00', flv: '#ff6d00', webm: '#ff6d00',
  // Audio
  mp3: '#7b1fa2', wav: '#7b1fa2', flac: '#7b1fa2', aac: '#7b1fa2', ogg: '#7b1fa2',
  // Archive
  zip: '#dea584', tar: '#dea584', gz: '#dea584', bz2: '#dea584', xz: '#dea584', '7z': '#dea584', rar: '#dea584', tgz: '#dea584',
  // Security
  key: '#f44336', pem: '#f44336', crt: '#f44336', cer: '#f44336', csr: '#f44336', pub: '#f44336',
  // Docker
  dockerfile: '#2496ed',
  // Log
  log: '#6d8086',
  // Binary/System
  so: '#909090', o: '#909090', a: '#909090', bin: '#909090', exe: '#909090',
  // Nginx
  nginx: '#009639',
}

const iconColor = computed(() => colorMap[ext.value] || '#8b949e')
const bgColor = computed(() => iconColor.value)

const extLabel = computed(() => {
  const e = ext.value
  if (!e) return ''
  if (e.length > 5) return e.slice(0, 4)
  return e.toUpperCase()
})
</script>
