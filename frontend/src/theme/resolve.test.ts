import assert from 'node:assert/strict'
import test from 'node:test'
import { ACCENT_PRESETS, contrastRatio, getPresetByKey } from '../utils/accent-colors.ts'
import {
  DEFAULT_PREFERENCE,
  type AppearancePreference,
  type ThemeOverrides,
} from './types.ts'
import { listThemes, getTheme } from './catalog.ts'
import { resolveAppearance, switchTheme, isCustomized } from './resolve.ts'
import { mixSrgb, relativeLuminance } from './pack-color.ts'
import { hydrateAppearance, migrateLegacyAppearance, sanitizePreference } from './migrate.ts'
import {
  applyPreview,
  beginPreview,
  cancelPreview,
  patchDraft,
  restoreAllDefaults,
  restoreGroup,
  restoreThemeDefaults,
} from './preview.ts'

const env = { systemDark: true, systemReduceMotion: false, transparencySupported: true }

test('catalog ships atelier and lumen with complete dark and light packs', () => {
  const themes = listThemes()
  assert.deepEqual(themes.map((item) => item.id), ['atelier', 'lumen', 'ink', 'harbor', 'quartz'])
  for (const theme of themes) {
    assert.equal(theme.schemaVersion, 1)
    assert.ok(theme.modes.dark.colors.bgBase)
    assert.ok(theme.modes.light.colors.bgBase)
    assert.notEqual(theme.modes.dark.colors.bgBase, theme.modes.light.colors.bgBase)
    assert.ok(theme.variants.sidebar)
    assert.ok(theme.variants.card)
    assert.ok(theme.iconSet)
    assert.ok(theme.terminal.dark)
    assert.ok(theme.editor.light)
  }
})

test('lumen is not just an accent swap — nav, card, icon and material differ', () => {
  const atelier = getTheme('atelier')
  const lumen = getTheme('lumen')
  assert.notEqual(atelier.variants.sidebar, lumen.variants.sidebar)
  assert.notEqual(atelier.variants.subnav, lumen.variants.subnav)
  assert.notEqual(atelier.variants.card, lumen.variants.card)
  assert.notEqual(atelier.iconSet, lumen.iconSet)
  assert.notEqual(atelier.tokens.materials.transparency, lumen.tokens.materials.transparency)
  assert.notEqual(atelier.defaults.uiFont, lumen.defaults.uiFont)
})

test('resolve order: defaults < theme < mode < user override < a11y fallback', () => {
  const pref: AppearancePreference = {
    ...DEFAULT_PREFERENCE,
    themeId: 'atelier',
    mode: 'light',
    reduceMotion: false,
    overridesByTheme: {
      atelier: { density: 'compact', accentKey: 'emerald', transparency: true },
    },
  }
  const resolved = resolveAppearance(pref, env)
  assert.equal(resolved.colorMode, 'light')
  assert.equal(resolved.density, 'compact')
  assert.equal(resolved.accent.key, 'emerald')
  assert.equal(resolved.colors.bgBase, getTheme('atelier').modes.light.colors.bgBase)
  assert.equal(resolved.transparency, true)
  assert.equal(resolved.materials.blurPx, getTheme('atelier').tokens.materials.blurPx)
  assert.equal(resolved.materials.surfaceOpacity, getTheme('atelier').tokens.materials.surfaceOpacity)

  const reduced = resolveAppearance(pref, { ...env, systemReduceMotion: true, transparencySupported: false })
  assert.equal(reduced.reduceMotion, true)
  assert.equal(reduced.transparency, false)
  assert.equal(reduced.materials.blurPx, 0)
})

test('auto mode follows the system color scheme', () => {
  const pref: AppearancePreference = { ...DEFAULT_PREFERENCE, mode: 'auto', overridesByTheme: {} }
  assert.equal(resolveAppearance(pref, { ...env, systemDark: false }).colorMode, 'light')
  assert.equal(resolveAppearance(pref, { ...env, systemDark: true }).colorMode, 'dark')
})

test('overrides stay scoped to the current theme', () => {
  const pref: AppearancePreference = {
    ...DEFAULT_PREFERENCE,
    themeId: 'atelier',
    overridesByTheme: {
      atelier: { density: 'comfortable', accentKey: 'rose' },
      lumen: { density: 'compact' },
    },
  }
  const atelier = resolveAppearance(pref, env)
  assert.equal(atelier.density, 'comfortable')
  assert.equal(atelier.accent.key, 'rose')

  const switched = switchTheme(pref, 'lumen')
  const lumen = resolveAppearance(switched, env)
  assert.equal(lumen.themeId, 'lumen')
  assert.equal(lumen.density, 'compact')
  assert.notEqual(lumen.accent.key, 'rose')
})

test('switching themes does not copy old overrides unless keepPersonalPrefs is on', () => {
  const pref: AppearancePreference = {
    ...DEFAULT_PREFERENCE,
    themeId: 'atelier',
    keepPersonalPrefsAcrossThemes: false,
    overridesByTheme: {
      atelier: { density: 'comfortable', uiFont: 'lxgw', accentKey: 'amber', termFontSize: 18 },
    },
  }
  const plain = switchTheme(pref, 'lumen')
  assert.equal(plain.overridesByTheme.lumen?.density, undefined)
  assert.equal(plain.overridesByTheme.lumen?.accentKey, undefined)
  assert.deepEqual(plain.overridesByTheme.atelier, pref.overridesByTheme.atelier)

  const kept = switchTheme({ ...pref, keepPersonalPrefsAcrossThemes: true }, 'lumen')
  assert.equal(kept.overridesByTheme.lumen?.density, 'comfortable')
  assert.equal(kept.overridesByTheme.lumen?.uiFont, 'lxgw')
  assert.equal(kept.overridesByTheme.lumen?.termFontSize, 18)
  assert.equal(kept.overridesByTheme.lumen?.accentKey, undefined)
})

test('unknown values fall back without throwing', () => {
  const sanitized = sanitizePreference({
    schemaVersion: 1,
    themeId: 'neon-dreams',
    mode: 'solar',
    reduceMotion: 'yes',
    overridesByTheme: {
      atelier: { density: 'ultra', uiFont: 'comic', termFontSize: 99, termBgOpacity: 4 },
      'neon-dreams': { density: 'compact' },
    },
  })
  assert.equal(sanitized.themeId, 'neon-dreams')
  assert.equal(sanitized.mode, 'dark')
  assert.equal(sanitized.reduceMotion, true)
  assert.equal(sanitized.overridesByTheme.atelier?.density, undefined)
  assert.equal(sanitized.overridesByTheme.atelier?.termFontSize, undefined)
  assert.equal(sanitized.overridesByTheme['neon-dreams']?.density, 'compact')
  assert.doesNotThrow(() => resolveAppearance(sanitized, env))
  const resolved = resolveAppearance(sanitized, env)
  assert.equal(resolved.themeId, 'neon-dreams')
  assert.equal(resolved.themeMissing, true)
  assert.equal(resolved.termWallpaper, 'none')
})

test('unsupported schema major version is rejected and previous preference is kept', () => {
  const previous = { ...DEFAULT_PREFERENCE, themeId: 'lumen' as const }
  const hydrated = hydrateAppearance({
    localV1: { schemaVersion: 9, themeId: 'atelier' },
    previous,
  })
  assert.equal(hydrated.preference.themeId, 'lumen')
  assert.ok(hydrated.warnings.some((item) => /schema/i.test(item)))
})

test('legacy pinia fields migrate into atelier overrides without dropping values', () => {
  const migrated = migrateLegacyAppearance({
    theme: 'light',
    accentKey: 'neon',
    accentCustom: '',
    bgPreset: 'cosmos',
    uiFont: 'noto',
    uiDensity: 'compact',
    borderRadiusPreset: 'rounded',
    reduceMotion: true,
    cardBorderStyle: 'shadow-only',
    sidebarWidth: 'wide',
    termTheme: 'dracula',
    termFont: 'firacode',
    termFontSize: 16,
    termBgOpacity: 0.8,
  })
  assert.equal(migrated.themeId, 'atelier')
  assert.equal(migrated.mode, 'light')
  assert.equal(migrated.reduceMotion, true)
  const over = migrated.overridesByTheme.atelier as ThemeOverrides
  assert.equal(over.accentKey, 'neon')
  assert.equal(over.uiFont, 'noto')
  assert.equal(over.density, 'compact')
  assert.equal(over.radius, 'rounded')
  assert.equal(over.card, 'raised')
  assert.equal(over.sidebarWidth, 'wide')
  assert.equal(over.surfacePreset, 'cosmos')
  assert.equal(over.termTheme, 'dracula')
  assert.equal(over.termFont, 'firacode')
  assert.equal(over.termFontSize, 16)
  assert.equal(over.termBgOpacity, 0.8)
})

test('local v1 preference wins over backend and empty backend never wipes local', () => {
  const local = {
    schemaVersion: 1,
    themeId: 'lumen',
    mode: 'light',
    reduceMotion: true,
    keepPersonalPrefsAcrossThemes: false,
    overridesByTheme: { lumen: { density: 'comfortable' } },
  }
  const fromLocal = hydrateAppearance({
    localV1: local,
    backendRaw: { theme: 'dark', accentKey: 'neon', uiDensity: 'compact' },
  })
  assert.equal(fromLocal.source, 'local-v1')
  assert.equal(fromLocal.preference.themeId, 'lumen')
  assert.equal(fromLocal.preference.overridesByTheme.lumen?.density, 'comfortable')

  const emptyBackend = hydrateAppearance({
    localLegacy: { theme: 'light', accentKey: 'emerald', uiDensity: 'compact' },
    backendRaw: '{}',
  })
  assert.equal(emptyBackend.source, 'local-legacy')
  assert.equal(emptyBackend.preference.mode, 'light')
  assert.equal(emptyBackend.preference.overridesByTheme.atelier?.accentKey, 'emerald')

  const backendSeed = hydrateAppearance({
    backendRaw: { theme: 'light', accentKey: 'violet', bgPreset: 'warm' },
  })
  assert.equal(backendSeed.source, 'backend')
  assert.equal(backendSeed.preference.mode, 'light')
  assert.equal(backendSeed.preference.overridesByTheme.atelier?.accentKey, 'violet')
})

test('broken JSON and null hydrate to the default theme instead of throwing', () => {
  assert.doesNotThrow(() => hydrateAppearance({ localV1: null, backendRaw: '{not json' }))
  const result = hydrateAppearance({ localV1: null, backendRaw: null })
  assert.equal(result.preference.themeId, 'atelier')
  assert.equal(result.preference.mode, 'dark')
})

test('preview patches apply to draft only; cancel restores committed', () => {
  const committed = { ...DEFAULT_PREFERENCE }
  const session = beginPreview(committed)
  patchDraft(session, { mode: 'light' })
  patchDraft(session, { overrides: { density: 'comfortable' } })
  assert.equal(session.draft.mode, 'light')
  assert.equal(session.committed.mode, 'dark')
  assert.equal(cancelPreview(session).mode, 'dark')
  assert.equal(applyPreview(beginPreview({ ...committed, mode: 'light' })).mode, 'light')
})

test('restore group and restore all clear the expected override keys', () => {
  const session = beginPreview({
    ...DEFAULT_PREFERENCE,
    themeId: 'lumen',
    mode: 'light',
    overridesByTheme: {
      lumen: { density: 'compact', accentKey: 'rose', card: 'flat', termFontSize: 20 },
    },
  })
  restoreGroup(session, 'accent')
  assert.equal(session.draft.overridesByTheme.lumen?.accentKey, undefined)
  assert.equal(session.draft.overridesByTheme.lumen?.density, 'compact')
  restoreThemeDefaults(session)
  assert.deepEqual(session.draft.overridesByTheme.lumen, {})
  restoreAllDefaults(session)
  assert.equal(session.draft.themeId, 'atelier')
  assert.equal(session.draft.mode, 'dark')
  assert.deepEqual(session.draft.overridesByTheme, {})
})

test('atelier defaults to vermilion ribbon lxgw consolas and half-opaque terminal', () => {
  const atelier = resolveAppearance(DEFAULT_PREFERENCE, env)
  assert.equal(atelier.accent.primary.toLowerCase(), getPresetByKey('vermilion')?.primary.toLowerCase())
  assert.equal(atelier.accent.primary.toLowerCase(), '#cb2028')
  assert.equal(atelier.chromeTexture, 'ribbon')
  assert.equal(atelier.uiFontKey, 'lxgw')
  assert.equal(atelier.termFont, 'consolas')
  assert.equal(atelier.termBgOpacity, 0.5)
  assert.equal(atelier.colors.bgSurface.toLowerCase(), '#191c23')
})

test('neon still resolves as a custom accent', () => {
  const atelier = resolveAppearance(DEFAULT_PREFERENCE, env)
  assert.ok(atelier.accent.primary)
  const neon = resolveAppearance({
    ...DEFAULT_PREFERENCE,
    overridesByTheme: { atelier: { accentKey: 'neon' } },
  }, env)
  assert.equal(neon.accent.key, 'neon')
  assert.ok(ACCENT_PRESETS.some((item) => item.key === 'neon'))
})

test('light accent and muted text meet 4.5:1 on the content surface', () => {
  const atelier = resolveAppearance({ ...DEFAULT_PREFERENCE, mode: 'light' }, env)
  assert.ok(contrastRatio(atelier.accent.primary, atelier.colors.bgSurface) >= 4.5)
  assert.ok(contrastRatio(atelier.colors.textMuted, atelier.colors.bgSurface) >= 4.5)
  assert.ok(contrastRatio(atelier.accent.onAccent, atelier.accent.primary) >= 4.5)
  const lumen = resolveAppearance({ ...DEFAULT_PREFERENCE, themeId: 'lumen', mode: 'light' }, env)
  assert.ok(contrastRatio(lumen.accent.primary, lumen.colors.bgSurface) >= 4.5)
  assert.ok(contrastRatio(lumen.colors.textMuted, lumen.colors.bgSurface) >= 4.5)
  assert.ok(contrastRatio(lumen.accent.onAccent, lumen.accent.primary) >= 4.5)
  for (const id of ['ink', 'harbor', 'quartz'] as const) {
    const resolved = resolveAppearance({ ...DEFAULT_PREFERENCE, themeId: id, mode: 'light' }, env)
    assert.ok(contrastRatio(resolved.accent.primary, resolved.colors.bgSurface) >= 4.5, `${id} accent`)
    assert.ok(contrastRatio(resolved.colors.textMuted, resolved.colors.bgSurface) >= 4.5, `${id} muted`)
    assert.ok(contrastRatio(resolved.accent.onAccent, resolved.accent.primary) >= 4.5, `${id} onAccent`)
  }
})

test('ink harbor quartz are not accent-only clones of atelier or lumen', () => {
  const atelier = getTheme('atelier')
  const lumen = getTheme('lumen')
  const ink = getTheme('ink')
  const harbor = getTheme('harbor')
  const quartz = getTheme('quartz')
  assert.notEqual(ink.modes.dark.colors.bgBase, atelier.modes.dark.colors.bgBase)
  assert.notEqual(harbor.modes.dark.colors.bgBase, lumen.modes.dark.colors.bgBase)
  assert.notEqual(quartz.variants.card, atelier.variants.card)
  assert.notEqual(ink.variants.subnav, atelier.variants.subnav)
  assert.notEqual(harbor.variants.subnav, lumen.variants.subnav)
  assert.notEqual(quartz.defaults.uiFont, lumen.defaults.uiFont)
  assert.notEqual(ink.defaults.chromeTexture, atelier.defaults.chromeTexture)
  assert.equal(harbor.variants.sidebar, 'rail')
  assert.equal(ink.variants.heading, 'display')
  assert.equal(ink.defaults.chromeTexture, 'grain')
  assert.equal(quartz.variants.iconContainer, 'none')
  assert.notEqual(ink.modes.dark.colors.bgBase, harbor.modes.dark.colors.bgBase)
  assert.notEqual(harbor.modes.dark.colors.bgBase, quartz.modes.dark.colors.bgBase)
})

test('light mode never copies a dark surface hex from a dark preset', () => {
  const resolved = resolveAppearance({
    ...DEFAULT_PREFERENCE,
    mode: 'light',
    overridesByTheme: { atelier: { surfacePreset: 'void' } },
  }, env)
  assert.equal(resolved.colorMode, 'light')
  assert.equal(resolved.colors.bgBase, mixSrgb('#E4E6EC', '#000000', 0.06))
  assert.notEqual(resolved.colors.bgBase.toLowerCase(), getTheme('atelier').modes.dark.colors.bgBase.toLowerCase())
})

test('density tokens match the 32/36/40 and 36/44/52 baseline', () => {
  const compact = resolveAppearance({
    ...DEFAULT_PREFERENCE,
    overridesByTheme: { atelier: { density: 'compact' } },
  }, env)
  const comfy = resolveAppearance({
    ...DEFAULT_PREFERENCE,
    overridesByTheme: { atelier: { density: 'comfortable' } },
  }, env)
  assert.equal(compact.densityTokens.controlHeight, '32px')
  assert.equal(compact.densityTokens.rowHeight, '36px')
  assert.equal(comfy.densityTokens.controlHeight, '40px')
  assert.equal(comfy.densityTokens.rowHeight, '52px')
})

test('customized flag is true only when the current theme has overrides', () => {
  assert.equal(isCustomized(DEFAULT_PREFERENCE), false)
  assert.equal(isCustomized({
    ...DEFAULT_PREFERENCE,
    overridesByTheme: { lumen: { density: 'compact' } },
  }), false)
  assert.equal(isCustomized({
    ...DEFAULT_PREFERENCE,
    overridesByTheme: { atelier: { density: 'compact' } },
  }), true)
})

test('chrome texture and wallpaper enums sanitize, and terminals follow panel layers by default', () => {
  const sanitized = sanitizePreference({
    schemaVersion: 1,
    themeId: 'atelier',
    mode: 'dark',
    overridesByTheme: {
      atelier: {
        chromeTexture: 'stars',
        chromeImageMode: 'ftp',
        chromeImageUrl: 'javascript:alert(1)',
        termFollowChrome: 'yes',
        termImageMode: 'url',
        termImageUrl: 'https://example.com/bg.jpg',
      },
    },
  })
  assert.equal(sanitized.overridesByTheme.atelier?.chromeTexture, undefined)
  assert.equal(sanitizePreference({
    schemaVersion: 1,
    themeId: 'atelier',
    overridesByTheme: { atelier: { chromeTexture: 'diagonal' } },
  }).overridesByTheme.atelier?.chromeTexture, 'ribbon')
  assert.equal(sanitized.overridesByTheme.atelier?.chromeImageMode, undefined)
  assert.equal(sanitized.overridesByTheme.atelier?.termFollowChrome, undefined)
  assert.equal(sanitized.overridesByTheme.atelier?.termImageMode, 'url')
  assert.equal(sanitized.overridesByTheme.atelier?.termImageUrl, 'https://example.com/bg.jpg')

  const withTexture = resolveAppearance({
    ...DEFAULT_PREFERENCE,
    overridesByTheme: { atelier: { chromeTexture: 'grid' as never } },
  }, env)
  assert.equal(withTexture.chromeTexture, 'galaxy')
  assert.equal(withTexture.termFollowChrome, true)
  assert.equal(withTexture.datasets['chrome-texture'], 'galaxy')
  assert.equal(withTexture.datasets['term-follow'], 'on')

  const explicit = resolveAppearance({
    ...DEFAULT_PREFERENCE,
    overridesByTheme: { atelier: { chromeTexture: 'starfield', termFollowChrome: false } },
  }, env)
  assert.equal(explicit.termFollowChrome, false)
  assert.equal(explicit.datasets['term-follow'], 'off')
})

test('light surface family tints C.1 bg/surface and does not copy dark hexes', () => {
  const resolved = resolveAppearance({
    ...DEFAULT_PREFERENCE,
    mode: 'light',
    overridesByTheme: { atelier: { surfacePreset: 'abyss' } },
  }, { ...env, systemDark: false })
  assert.equal(resolved.colors.bgBase, mixSrgb('#E4E6EC', '#1D4ED8', 0.06))
  assert.equal(resolved.colors.bgSurface, mixSrgb('#F4F5F8', '#1D4ED8', 0.03))
  assert.equal(resolved.colors.bgBase.startsWith('#0') || resolved.colors.bgBase.startsWith('#1'), false)
})

test('explicit overlay secondary survives a custom accent; hover still recomputes', () => {
  const resolved = resolveAppearance({
    ...DEFAULT_PREFERENCE,
    overridesByTheme: {
      atelier: { accentKey: 'custom', accentCustom: '#123456', accentSecondary: '#ABCDEF' },
    },
  }, env)
  assert.equal(resolved.accent.primary.toUpperCase(), '#123456')
  assert.equal(resolved.accent.secondary.toUpperCase(), '#ABCDEF')
  assert.notEqual(resolved.accent.hover.toUpperCase(), '#123456')
})

test('light primary-light-9 is a pale fill, not a mix toward black', () => {
  const light = resolveAppearance({ ...DEFAULT_PREFERENCE, mode: 'light' }, env)
  const pale = light.cssVars['--el-color-primary-light-9']
  assert.ok(pale)
  assert.ok(relativeLuminance(pale) > 0.72)
  assert.equal(light.cssVars['--el-table-current-row-bg-color'], light.cssVars['--xp-inset-selected'])
  assert.equal(light.cssVars['--el-table-row-hover-bg-color'], light.cssVars['--xp-inset-hover'])
})

test('overlay can turn off card top edge and inset hover', () => {
  const resolved = resolveAppearance({
    ...DEFAULT_PREFERENCE,
    overridesByTheme: {
      atelier: { surfaces: { card: { topEdge: false }, inset: { hoverStrength: 0 } } },
    },
  }, env)
  assert.equal(resolved.cssVars['--xp-card-top-edge-shadow'], 'none')
  assert.match(resolved.cssVars['--xp-inset-hover'] || '', /0%|0\)|transparent/)
})
