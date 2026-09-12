import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { mixSrgb } from './pack-color.ts'
import { CHART_SAFE, parseThemePack, restoreOverlayGroup, type ThemePackOverlay } from './pack-parse.ts'
import { getTheme, hasTheme } from './catalog.ts'
import { installThemePack, resetInstalledPacksForTests } from './pack-store.ts'
import { resolveAppearance } from './resolve.ts'

const dir = dirname(fileURLToPath(import.meta.url))
const atelier = JSON.parse(readFileSync(join(dir, 'examples/atelier.theme.json'), 'utf8'))
const lumen = JSON.parse(readFileSync(join(dir, 'examples/lumen.theme.json'), 'utf8'))

test('Math.round mix matches documented sequential stops', () => {
  assert.equal(mixSrgb('#191C23', '#CB2028', 0.25), '#461D24')
  assert.equal(mixSrgb('#191C23', '#CB2028', 0.5), '#721E26')
  assert.equal(mixSrgb('#191C23', '#CB2028', 0.75), '#9F1F27')
})

test('atelier empty overlay inherits author secondary, topEdge true, hover 1', () => {
  const r = parseThemePack(atelier)
  assert.equal(r.ok, true)
  if (!r.ok) return
  assert.equal(r.pack.surfaces.cardTopEdge, true)
  assert.equal(r.pack.surfaces.hoverStrength, 1)
  assert.equal(r.pack.modes.dark.accentSecondary, '#E85A62')
  assert.equal(r.pack.modes.dark.accent, '#CB2028')
  assert.deepEqual(r.pack.modes.dark.sequential, ['#191C23', '#461D24', '#721E26', '#9F1F27', '#CB2028'])
  assert.equal(r.pack.modes.dark.categorical[0], '#CB2028')
  assert.deepEqual(r.pack.modes.light.categorical, [...CHART_SAFE.light])
  assert.ok(r.pack.warnings.some((w) => w.includes('charts.categorical.light')))
})

test('lumen empty overlay keeps explicit variants and materials via file not id', () => {
  const r = parseThemePack(lumen)
  assert.equal(r.ok, true)
  if (!r.ok) return
  assert.equal(r.pack.defaults.transparency, true)
  assert.equal(r.pack.defaults.radius, 'rounded')
  assert.equal(r.pack.defaults.variants.subnav, 'pill')
  assert.equal(r.pack.modes.dark.accent, '#E08A4A')
})

test('same palette different id yields same colors', () => {
  const a = structuredClone(atelier)
  const b = structuredClone(atelier)
  b.id = 'other-pack'
  const ra = parseThemePack(a)
  const rb = parseThemePack(b)
  assert.equal(ra.ok && rb.ok, true)
  if (!ra.ok || !rb.ok) return
  assert.deepEqual(ra.pack.modes, rb.pack.modes)
  assert.notEqual(ra.pack.id, rb.pack.id)
})

test('theme topEdge false and hover 0.4 not overwritten', () => {
  const file = structuredClone(atelier)
  file.surfaces = { card: { topEdge: false }, inset: { hoverStrength: 0.4 } }
  const r = parseThemePack(file)
  assert.equal(r.ok, true)
  if (!r.ok) return
  assert.equal(r.pack.surfaces.cardTopEdge, false)
  assert.equal(r.pack.surfaces.hoverStrength, 0.4)
})

test('overlay can set topEdge true and hover 0', () => {
  const file = structuredClone(atelier)
  file.surfaces = { card: { topEdge: false }, inset: { hoverStrength: 0.4 } }
  const r = parseThemePack(file, { surfaces: { card: { topEdge: true }, inset: { hoverStrength: 0 } } })
  assert.equal(r.ok, true)
  if (!r.ok) return
  assert.equal(r.pack.surfaces.cardTopEdge, true)
  assert.equal(r.pack.surfaces.hoverStrength, 0)
})

test('missing palette secondary derives with violet mix', () => {
  const file = structuredClone(atelier)
  delete file.palette.dark.accentSecondary
  delete file.palette.light.accentSecondary
  const r = parseThemePack(file)
  assert.equal(r.ok, true)
  if (!r.ok) return
  assert.equal(r.pack.modes.dark.accentSecondary, mixSrgb('#CB2028', '#8B5CF6', 0.6))
})

test('custom accent keeps explicit secondary; hover still recomputes', () => {
  const r = parseThemePack(atelier, { accentKey: 'custom', accentCustom: '#123456', accentSecondary: '#ABCDEF' })
  assert.equal(r.ok, true)
  if (!r.ok) return
  assert.equal(r.pack.modes.dark.accentSecondary, '#ABCDEF')
  assert.equal(r.pack.modes.dark.hover, mixSrgb('#123456', '#000000', 0.2))
  assert.equal(r.pack.modes.dark.accent, '#123456')
  assert.ok(r.pack.warnings.some((w) => w.includes('charts.categorical.dark')))
})

test('overlay secondary only does not change author accent or stop charts', () => {
  const r = parseThemePack(atelier, { accentSecondary: '#ABCDEF' })
  assert.equal(r.ok, true)
  if (!r.ok) return
  assert.equal(r.pack.modes.dark.accent, '#CB2028')
  assert.equal(r.pack.modes.dark.accentSecondary, '#ABCDEF')
  assert.equal(r.pack.modes.dark.categorical[0], '#CB2028')
})

test('restore advanced.recipe drops surfaces isolation but not futureOption', () => {
  const overlay: ThemePackOverlay = {
    surfaces: { inset: { hoverStrength: 1, future: 0.2 } },
    futureOption: true,
  }
  const r = parseThemePack(atelier, overlay)
  assert.equal(r.ok, true)
  if (!r.ok) return
  assert.equal(r.pack.isolated['surfaces.inset.future'], 0.2)
  assert.equal(r.pack.isolated.futureOption, true)
  const restored = restoreOverlayGroup(r.pack.overlay, r.pack.isolated, 'advanced.recipe')
  assert.equal(restored.isolated['surfaces.inset.future'], undefined)
  assert.equal(restored.isolated.futureOption, true)
  const again = parseThemePack(atelier, restored.overlay)
  assert.equal(again.ok, true)
  if (!again.ok) return
  assert.equal(again.pack.surfaces.hoverStrength, 1)
})

test('schemaMinor 99 still reads known fields', () => {
  const file = structuredClone(atelier)
  file.schemaMinor = 99
  const r = parseThemePack(file)
  assert.equal(r.ok, true)
  if (!r.ok) return
  assert.equal(r.pack.schemaMinor, 99)
  assert.equal(r.pack.modes.dark.accent, '#CB2028')
})

test('unsupported schemaVersion is rejected', () => {
  const file = structuredClone(atelier)
  file.schemaVersion = 2
  const r = parseThemePack(file)
  assert.equal(r.ok, false)
})

test('installing a pack makes catalog resolve it; missing id keeps overlay', () => {
  resetInstalledPacksForTests()
  const file = structuredClone(atelier)
  file.id = 'demo-pack'
  const installed = installThemePack(file)
  assert.equal(installed.ok, true)
  assert.equal(hasTheme('demo-pack'), true)
  const theme = getTheme('demo-pack')
  assert.equal(theme.source, 'pack')
  assert.equal(theme.modes.dark.accentHex, '#CB2028')
  const pref = {
    schemaVersion: 1 as const,
    themeId: 'demo-pack',
    mode: 'dark' as const,
    reduceMotion: false,
    keepPersonalPrefsAcrossThemes: false,
    overridesByTheme: { 'demo-pack': { density: 'compact' as const } },
  }
  resetInstalledPacksForTests()
  assert.equal(hasTheme('demo-pack'), false)
  const resolved = resolveAppearance(pref, { systemDark: true, systemReduceMotion: false, transparencySupported: true })
  assert.equal(resolved.themeMissing, true)
  assert.equal(resolved.themeId, 'demo-pack')
  assert.equal(pref.overridesByTheme['demo-pack']?.density, 'compact')
})

test('unknown derive.strategy stays on export path; cache is stripped', () => {
  const file = structuredClone(atelier)
  file.derive = { strategy: 'future', cache: {} }
  const r = parseThemePack(file)
  assert.equal(r.ok, true)
  if (!r.ok) return
  assert.equal(r.pack.strategyDeclared, 'future')
  assert.equal(r.pack.strategyUsed, 'v1')
  assert.ok(r.pack.stripped.includes('derive.cache'))
})
