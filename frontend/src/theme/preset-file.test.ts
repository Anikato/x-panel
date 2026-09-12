import assert from 'node:assert/strict'
import test from 'node:test'
import { DEFAULT_PREFERENCE } from './types.ts'
import { buildAppearancePreset, parseAppearancePreset } from './preset-file.ts'

test('appearance preset round-trips sanitized preference', () => {
  const source = {
    ...DEFAULT_PREFERENCE,
    themeId: 'atelier' as const,
    overridesByTheme: {
      atelier: { uiFont: 'lxgw' as const, termFont: 'consolas', chromeTexture: 'ribbon' as const },
    },
  }
  const file = buildAppearancePreset(source, new Date('2026-09-11T00:00:00.000Z'))
  assert.equal(file.kind, 'x-panel.appearance')
  assert.equal(file.schemaVersion, 1)
  const parsed = parseAppearancePreset(JSON.stringify(file))
  assert.equal(parsed.ok, true)
  if (parsed.ok) {
    assert.equal(parsed.preference.themeId, 'atelier')
    assert.equal(parsed.preference.overridesByTheme.atelier?.uiFont, 'lxgw')
    assert.equal(parsed.preference.overridesByTheme.atelier?.termFont, 'consolas')
    assert.equal(parsed.preference.overridesByTheme.atelier?.chromeTexture, 'ribbon')
  }
})

test('appearance preset strips uploaded wallpapers and rejects junk', () => {
  const file = buildAppearancePreset({
    ...DEFAULT_PREFERENCE,
    overridesByTheme: {
      atelier: { chromeImageMode: 'upload', chromeImageUrl: 'data:image/png;base64,aaa' },
    },
  })
  assert.equal(file.preference.overridesByTheme.atelier?.chromeImageMode, 'none')
  assert.equal(file.preference.overridesByTheme.atelier?.chromeImageUrl, '')
  assert.equal(parseAppearancePreset('{"foo":1}').ok, false)
  assert.equal(parseAppearancePreset('not-json').ok, false)
})

test('outer 1 + inner 1 with schemaMinor 99 still reads known fields', () => {
  const parsed = parseAppearancePreset({
    kind: 'x-panel.appearance',
    schemaVersion: 1,
    schemaMinor: 99,
    futureRoot: true,
    preference: {
      schemaVersion: 1,
      schemaMinor: 99,
      themeId: 'atelier',
      mode: 'dark',
      reduceMotion: false,
      keepPersonalPrefsAcrossThemes: false,
      overridesByTheme: {
        atelier: { uiFont: 'lxgw', futureOption: true },
      },
    },
  })
  assert.equal(parsed.ok, true)
  if (!parsed.ok) return
  assert.equal(parsed.preference.themeId, 'atelier')
  assert.equal(parsed.preference.overridesByTheme.atelier?.uiFont, 'lxgw')
  assert.equal((parsed.preference.overridesByTheme.atelier as { futureOption?: boolean } | undefined)?.futureOption, undefined)
  assert.equal(parsed.preference.isolated?.['overridesByTheme.atelier.futureOption'], true)
  const round = buildAppearancePreset(parsed.preference)
  assert.equal((round.preference.overridesByTheme.atelier as { futureOption?: boolean })?.futureOption, true)
})

test('inner preference schemaVersion 2 is rejected', () => {
  const parsed = parseAppearancePreset({
    kind: 'x-panel.appearance',
    schemaVersion: 1,
    preference: {
      ...DEFAULT_PREFERENCE,
      schemaVersion: 2,
    },
  })
  assert.equal(parsed.ok, false)
})

test('export rewrites custom uiFont to system without mutating the live preference', () => {
  const live = {
    ...DEFAULT_PREFERENCE,
    overridesByTheme: { atelier: { uiFont: 'custom' as const } },
  }
  const file = buildAppearancePreset(live)
  assert.equal(file.preference.overridesByTheme.atelier?.uiFont, 'system')
  assert.equal(live.overridesByTheme.atelier?.uiFont, 'custom')
})
