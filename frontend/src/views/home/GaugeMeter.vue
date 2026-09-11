<template>
  <div class="gauge-meter" :class="tone" :style="meterStyle">
    <div class="gauge-well">
      <svg viewBox="0 0 140 104" class="gauge-svg" aria-hidden="true">
        <path class="gauge-track" :d="arcD" pathLength="100" />
        <path class="gauge-fill" :d="arcD" pathLength="100" />
        <line
          v-for="tick in ticks"
          :key="tick.p"
          :class="tick.major ? 'gauge-tick-maj' : 'gauge-tick'"
          :x1="tick.x1"
          :y1="tick.y1"
          :x2="tick.x2"
          :y2="tick.y2"
        />
        <g class="gauge-arm">
          <polygon class="blade-glow" points="70,12 74.2,56 65.8,56" />
          <polygon class="blade" points="70,14 72.2,56 67.8,56" />
          <polygon class="blade-core" points="70,20 70.8,56 69.2,56" />
          <circle class="hub" cx="70" cy="56" r="3.1" />
          <circle class="hub-core" cx="70" cy="56" r="1.3" />
        </g>
      </svg>
    </div>
    <div class="gauge-caption">
      <span class="gauge-chip">{{ display }}</span>
      <span class="gauge-label">{{ label }}</span>
      <span v-if="sub" class="gauge-sub">{{ sub }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'

const props = withDefaults(defineProps<{
  label: string
  percent?: number
  sub?: string
  tone?: string
  value?: string
  color?: string
}>(), {
  percent: 0,
  tone: 'c-ok',
  color: 'var(--xp-accent)',
})

const CX = 70
const CY = 56
const R = 48
const START = 150
const SWEEP = 240

const clamp = computed(() => Math.max(0, Math.min(100, props.percent || 0)))
const display = computed(() => props.value || `${clamp.value.toFixed(1)}%`)
const shown = ref(0)
const ready = ref(false)

onMounted(() => {
  requestAnimationFrame(() => {
    shown.value = clamp.value
    ready.value = true
  })
})

watch(clamp, (value) => {
  if (ready.value) shown.value = value
})

const meterStyle = computed(() => ({
  '--gauge-base': props.color,
  '--p': String(shown.value < 0.6 && shown.value > 0 ? 0.6 : shown.value),
}))

const polar = (deg: number, radius = R) => {
  const a = (deg * Math.PI) / 180
  return {
    x: CX + radius * Math.cos(a),
    y: CY + radius * Math.sin(a),
  }
}

const arcD = computed(() => {
  const a0 = polar(START)
  const a1 = polar(START + SWEEP)
  return `M ${a0.x.toFixed(2)} ${a0.y.toFixed(2)} A ${R} ${R} 0 1 1 ${a1.x.toFixed(2)} ${a1.y.toFixed(2)}`
})

const ticks = computed(() => {
  const items = []
  for (let p = 0; p <= 100; p += 5) {
    const major = p % 25 === 0
    const deg = START + (SWEEP * p) / 100
    const inner = polar(deg, major ? 41 : 43.2)
    const outer = polar(deg, 46.8)
    items.push({
      p,
      major,
      x1: inner.x.toFixed(2),
      y1: inner.y.toFixed(2),
      x2: outer.x.toFixed(2),
      y2: outer.y.toFixed(2),
    })
  }
  return items
})
</script>

<style scoped lang="scss">
.gauge-meter {
  --gauge-color: var(--gauge-base, var(--xp-accent));
  --gauge-sweep: 450ms;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.gauge-well {
  position: relative;
  width: 100%;
  max-width: 220px;
  margin: 0 auto;
}

.gauge-svg {
  display: block;
  width: 100%;
  height: auto;
  overflow: visible;
  pointer-events: none;
}

.gauge-track {
  fill: none;
  stroke: color-mix(in srgb, var(--xp-text-primary) 11%, transparent);
  stroke-width: 2.2;
  stroke-linecap: round;
}

.gauge-fill {
  fill: none;
  stroke: var(--gauge-color);
  stroke-width: 2.2;
  stroke-linecap: round;
  stroke-dasharray: 100;
  stroke-dashoffset: calc(100 - var(--p, 0));
  transition: stroke-dashoffset var(--gauge-sweep) cubic-bezier(.2, .8, .2, 1);
}

.gauge-tick {
  stroke: color-mix(in srgb, var(--xp-text-primary) 28%, transparent);
  stroke-width: 1;
}

.gauge-tick-maj {
  stroke: color-mix(in srgb, var(--xp-text-primary) 50%, transparent);
  stroke-width: 1.4;
}

.gauge-arm {
  transform-box: view-box;
  transform-origin: 70px 56px;
  transform: rotate(calc(-120deg + var(--p, 0) * 2.4deg));
  transition: transform var(--gauge-sweep) cubic-bezier(.2, .8, .2, 1);
}

.blade-glow {
  fill: var(--gauge-color);
  opacity: 0.28;
}

.blade {
  fill: var(--gauge-color);
}

.blade-core,
.hub {
  fill: var(--xp-text-primary);
}

.hub-core {
  fill: var(--xp-bg-surface);
}

.gauge-caption {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  margin-top: -6px;
  text-align: center;
}

.gauge-chip {
  width: fit-content;
  padding: 2px 8px;
  color: var(--xp-text-primary);
  font-size: 15px;
  font-weight: 750;
  letter-spacing: -0.04em;
  font-variant-numeric: tabular-nums;
  line-height: 1.25;
  background: color-mix(in srgb, var(--gauge-color) 12%, var(--xp-bg-inset));
  border: 1px solid color-mix(in srgb, var(--gauge-color) 30%, transparent);
  border-radius: 999px;
}

.gauge-label {
  color: var(--xp-text-primary);
  font-size: 13px;
  font-weight: 650;
}

.gauge-sub {
  color: var(--xp-text-muted);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.c-warn {
  --gauge-color: var(--xp-warning);
}

.c-danger {
  --gauge-color: var(--xp-danger);
}
</style>
