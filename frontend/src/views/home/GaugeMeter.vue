<template>
  <div class="gauge-meter" :class="tone" :style="{ '--gauge-base': color }">
    <div class="gauge-well">
      <svg viewBox="0 0 140 112" class="gauge-svg" aria-hidden="true">
        <path class="gauge-track" :d="arcD" pathLength="100" />
        <path
          class="gauge-fill"
          :d="arcD"
          pathLength="100"
          :stroke-dasharray="`${fill} 100`"
        />
        <circle
          v-for="tick in ticks"
          :key="tick.p"
          class="gauge-tick"
          :cx="tick.x"
          :cy="tick.y"
          r="1.4"
        />
      </svg>
      <div class="gauge-readout">
        <strong>{{ display }}</strong>
      </div>
    </div>
    <div class="gauge-caption">
      <span class="gauge-label">{{ label }}</span>
      <span v-if="sub" class="gauge-sub">{{ sub }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

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
const CY = 64
const R = 48
const START = 135
const SWEEP = 270

const clamp = computed(() => Math.max(0, Math.min(100, props.percent || 0)))
const fill = computed(() => (clamp.value < 0.6 && clamp.value > 0 ? 0.6 : clamp.value))
const display = computed(() => props.value || `${clamp.value.toFixed(1)}%`)

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

const ticks = computed(() => [0, 25, 50, 75, 100].map((p) => {
  const pt = polar(START + (SWEEP * p) / 100, R + 7)
  return { p, x: pt.x.toFixed(2), y: pt.y.toFixed(2) }
}))
</script>

<style scoped lang="scss">
.gauge-meter {
  --gauge-color: var(--gauge-base, var(--xp-accent));
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
  overflow: hidden;
  pointer-events: none;
  contain: paint;
}

.gauge-track {
  fill: none;
  stroke: color-mix(in srgb, var(--xp-text-primary) 10%, transparent);
  stroke-width: var(--gauge-stroke, 9);
  stroke-linecap: round;
}

.gauge-fill {
  fill: none;
  stroke: var(--gauge-color);
  stroke-width: var(--gauge-stroke, 9);
  stroke-linecap: round;
}

.gauge-tick {
  fill: color-mix(in srgb, var(--xp-text-primary) 28%, transparent);
}

.gauge-readout {
  position: absolute;
  left: 50%;
  top: 58%;
  transform: translate(-50%, -50%);
  text-align: center;
  pointer-events: none;

  strong {
    display: block;
    color: var(--xp-text-primary);
    font-size: clamp(20px, 1.8vw, 28px);
    font-weight: 800;
    letter-spacing: -0.04em;
    font-variant-numeric: tabular-nums;
    line-height: 1;
  }
}

.gauge-caption {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  margin-top: -4px;
  text-align: center;
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
