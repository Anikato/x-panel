import assert from 'node:assert/strict'
import test from 'node:test'
import {
  BG_PRESETS,
  DENSITY_MAP,
  SIDEBAR_WIDTH_MAP,
  pickBgVars,
  sanitizeAppearance,
  shouldReduceMotion,
} from './appearance.ts'
import { ACCENT_PRESETS, getPresetByKey } from './accent-colors.ts'

test('sidebar width presets map to the new 200/224/256 baseline', () => {
  assert.equal(SIDEBAR_WIDTH_MAP.narrow, '200px')
  assert.equal(SIDEBAR_WIDTH_MAP.default, '224px')
  assert.equal(SIDEBAR_WIDTH_MAP.wide, '256px')
})

test('graphite is the default dark background and has a light counterpart', () => {
  const graphite = BG_PRESETS.find((item) => item.key === 'graphite')
  assert.ok(graphite)
  assert.equal(pickBgVars('graphite', true)['--xp-bg-base'], '#111318')
  assert.equal(pickBgVars('graphite', false)['--xp-bg-base'], '#E4E6EC')
})

test('light mode never applies a dark background hex from a dark preset', () => {
  const light = pickBgVars('void', false)
  assert.equal(light['--xp-bg-base'], '#E4E6EC')
  assert.equal(light['--xp-bg-surface'], '#F4F5F8')
})

test('unknown appearance enums fall back to safe defaults', () => {
  const sanitized = sanitizeAppearance({
    bgPreset: 'neon-dreams',
    uiFont: 'comic-sans',
    uiDensity: 'ultra',
    borderRadiusPreset: 'pill',
    cardBorderStyle: 'glow',
    sidebarWidth: 'huge',
    accentKey: 'not-a-color',
  })
  assert.equal(sanitized.bgPreset, 'graphite')
  assert.equal(sanitized.uiFont, 'system')
  assert.equal(sanitized.uiDensity, 'default')
  assert.equal(sanitized.borderRadiusPreset, 'default')
  assert.equal(sanitized.cardBorderStyle, 'accent-left')
  assert.equal(sanitized.sidebarWidth, 'default')
  assert.equal(sanitized.accentKey, 'steel')
})

test('broken appearance JSON does not throw', () => {
  assert.doesNotThrow(() => sanitizeAppearance(JSON.parse('{"bgPreset":1}')))
  assert.equal(sanitizeAppearance(null).bgPreset, 'graphite')
})

test('system reduced motion always wins', () => {
  assert.equal(shouldReduceMotion(false, true), true)
  assert.equal(shouldReduceMotion(true, false), true)
  assert.equal(shouldReduceMotion(false, false), false)
})

test('density also controls control and table metrics', () => {
  assert.equal(DENSITY_MAP.compact.controlHeight, '32px')
  assert.equal(DENSITY_MAP.default.controlHeight, '36px')
  assert.equal(DENSITY_MAP.comfortable.controlHeight, '40px')
  assert.equal(DENSITY_MAP.compact.rowHeight, '36px')
  assert.equal(DENSITY_MAP.default.rowHeight, '44px')
  assert.equal(DENSITY_MAP.comfortable.rowHeight, '52px')
})

test('steel is the new default accent and old neon remains available', () => {
  assert.equal(getPresetByKey('steel')?.primary.toLowerCase(), '#7aa2ff')
  assert.ok(ACCENT_PRESETS.some((item) => item.key === 'neon'))
})
