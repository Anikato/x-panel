import { getPresetByKey } from '../utils/accent-colors.ts'
import {
  BLACK,
  WHITE,
  compositeRgbaOnHex,
  contrastRatio,
  correctAgainst,
  isHex,
  mixSrgb,
  monotonicLuminance,
  normalizeHex,
  parseRgba,
  rgbDistance,
  rgbaFromHex,
} from './pack-color.ts'

export const THEME_PACK_KIND = 'x-panel.theme'
const MAX_BYTES = 256 * 1024
const MAX_DEPTH = 16
const SLUG = /^[a-z0-9][a-z0-9-]{1,63}$/

const SAFE = {
  dark: {
    bg: '#111318', surface: '#191C23', text: '#EEF0F4', textMuted: '#7B8494',
    border: '#303641', accent: '#CB2028', success: '#22C55E', warning: '#F59E0B',
    danger: '#EF4444', info: '#3B82F6',
  },
  light: {
    bg: '#E4E6EC', surface: '#F4F5F8', text: '#141820', textMuted: '#5B6472',
    border: '#C3C9D4', accent: '#5B8CFF', success: '#15803D', warning: '#B45309',
    danger: '#B91C1C', info: '#1D4ED8',
  },
} as const

export const CHART_SAFE = {
  dark: ['#7AA2FF', '#34D399', '#FBBF24', '#FB7185', '#A78BFA', '#22D3EE'],
  light: ['#1D4ED8', '#15803D', '#92400E', '#B91C1C', '#7E22CE', '#0E7490'],
} as const

const ENUMS = {
  mode: ['dark', 'light', 'auto'],
  density: ['compact', 'default', 'comfortable'],
  uiFont: ['system', 'inter', 'noto', 'lxgw'],
  sidebarWidth: ['narrow', 'default', 'wide'],
  headerHeight: ['compact', 'default', 'comfortable'],
  radius: ['sharp', 'default', 'rounded'],
  iconSet: ['outline', 'solid'],
  chromeTexture: ['none', 'ribbon', 'galaxy', 'starfield', 'grain'],
  sidebar: ['marker', 'block', 'rail'],
  heading: ['compact', 'display'],
  subnav: ['line', 'block', 'pill'],
  card: ['flat', 'outline', 'raised'],
  iconContainer: ['none', 'tile'],
  termFont: ['jetbrains', 'firacode', 'cascadia', 'consolas', 'system'],
} as const

const FORBIDDEN_DERIVE = new Set(['cache', 'computed'])

export type ColorMode = 'dark' | 'light'

export interface ThemePackOverlay {
  accentKey?: string
  accentCustom?: string
  accentSecondary?: string
  surfaces?: {
    card?: { topEdge?: boolean }
    inset?: { hoverStrength?: number, [k: string]: unknown }
    [k: string]: unknown
  }
  [k: string]: unknown
}

export interface ModeRuntime {
  bg: string
  surface: string
  elevated: string
  overlay: string
  inset: string
  sidebar: string
  header: string
  input: string
  tableHeader: string
  auth: string
  text: string
  textSecondary: string
  textMuted: string
  border: string
  borderLight: string
  borderHover: string
  accent: string
  accentSecondary: string
  success: string
  warning: string
  danger: string
  info: string
  hover: string
  muted: string
  onAccent: string
  fill1: string
  fill2: string
  categorical: string[]
  sequential: string[]
}

export interface ResolvedPack {
  id: string
  name: string
  version: string
  schemaVersion: number
  schemaMinor: number
  strategyDeclared: string
  strategyUsed: 'v1'
  defaults: {
    mode: string
    density: string
    uiFont: string
    sidebarWidth: string
    headerHeight: string
    radius: string
    transparency: boolean
    iconSet: string
    termFont: string
    termFontSize: number
    termBgOpacity: number
    chromeTexture: string
    variants: { sidebar: string, subnav: string, card: string, iconContainer: string }
  }
  surfaces: { cardTopEdge: boolean, hoverStrength: number }
  modes: { dark: ModeRuntime, light: ModeRuntime }
  warnings: string[]
  stripped: string[]
  isolated: Record<string, unknown>
  overlay: ThemePackOverlay
}

type Warn = (path: string, msg: string) => void

function depthOf(value: unknown, d = 0): number {
  if (d > MAX_DEPTH) return d
  if (!value || typeof value !== 'object') return d
  let max = d
  for (const v of Object.values(value as Record<string, unknown>)) {
    max = Math.max(max, depthOf(v, d + 1))
  }
  return max
}

function stripForbiddenKeys(value: unknown, path: string, stripped: string[]): unknown {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return value
  const out: Record<string, unknown> = {}
  for (const [k, v] of Object.entries(value as Record<string, unknown>)) {
    if (k === '__proto__' || k === 'prototype' || k === 'constructor') {
      stripped.push(path ? `${path}.${k}` : k)
      continue
    }
    out[k] = stripForbiddenKeys(v, path ? `${path}.${k}` : k, stripped)
  }
  return out
}

function isObj(v: unknown): v is Record<string, unknown> {
  return Boolean(v) && typeof v === 'object' && !Array.isArray(v)
}

function readEnum(raw: unknown, allowed: readonly string[], fallback: string, path: string, warnings: string[], isolated: Record<string, unknown>): string {
  if (raw === undefined) return fallback
  if (typeof raw !== 'string' || !allowed.includes(raw)) {
    warnings.push(`${path}: unknown enum`)
    if (typeof raw === 'string') isolated[path] = raw
    return fallback
  }
  return raw
}

function readBool(raw: unknown, fallback: boolean, path: string, warnings: string[]): boolean {
  if (raw === undefined) return fallback
  if (typeof raw !== 'boolean') {
    warnings.push(`${path}: not boolean`)
    return fallback
  }
  return raw
}

function readNumber(raw: unknown, fallback: number, min: number, max: number, path: string, warnings: string[], integer = false): number {
  if (raw === undefined || raw === null) return fallback
  if (typeof raw !== 'number' || !Number.isFinite(raw)) {
    warnings.push(`${path}: not number`)
    return fallback
  }
  let n = integer ? Math.round(raw) : raw
  const clamped = Math.min(max, Math.max(min, n))
  if (clamped !== raw || (integer && n !== raw)) warnings.push(`${path}: clamped`)
  return clamped
}

function readHex(raw: unknown, allowRgba: boolean): { ok: true, hex: string, rgba?: string } | { ok: false } {
  if (typeof raw !== 'string') return { ok: false }
  if (isHex(raw)) return { ok: true, hex: normalizeHex(raw) }
  if (allowRgba && parseRgba(raw)) return { ok: true, hex: '', rgba: raw }
  return { ok: false }
}

function asHex(raw: unknown): string | undefined {
  return isHex(raw) ? normalizeHex(String(raw)) : undefined
}

function requiredHex(mode: ColorMode, key: string, raw: unknown, warnings: string[]): string {
  const parsed = readHex(raw, false)
  if (parsed.ok) return parsed.hex
  warnings.push(`palette.${mode}.${key}: invalid, using safe`)
  return SAFE[mode][key as keyof typeof SAFE.dark] || SAFE[mode].surface
}

function onAccentFor(bg: string): string {
  const black = contrastRatio(BLACK, bg)
  const white = contrastRatio(WHITE, bg)
  if (white > black) return WHITE
  return BLACK
}

function chartOk(colors: string[], surface: string): boolean {
  if (colors.length < 3) return false
  for (let i = 0; i < colors.length; i++) {
    if (contrastRatio(colors[i], surface) < 3) return false
    for (let j = i + 1; j < colors.length; j++) {
      if (rgbDistance(colors[i], colors[j]) < 48) return false
    }
  }
  return true
}

function deriveMode(
  mode: ColorMode,
  src: Record<string, unknown>,
  overlay: ThemePackOverlay,
  warnings: string[],
): ModeRuntime {
  const safe = SAFE[mode]
  const req = (key: keyof typeof safe, raw: unknown) => {
    const p = readHex(raw, false)
    if (p.ok) return p.hex
    if (raw !== undefined) warnings.push(`palette.${mode}.${key}: invalid`)
    if (['bg', 'surface', 'text', 'textMuted', 'border', 'accent', 'success', 'warning', 'danger', 'info'].includes(key)) {
      if (raw === undefined) {
        /* required missing handled by caller reject */
      }
    }
    return requiredHex(mode, key, raw, warnings)
  }

  const bg = req('bg', src.bg)
  const surface = req('surface', src.surface)
  const text = req('text', src.text)
  const textMuted = req('textMuted', src.textMuted)
  const borderIn = src.border
  let border: string
  const borderParsed = readHex(borderIn, true)
  if (borderParsed.ok && borderParsed.rgba) {
    border = compositeRgbaOnHex(borderParsed.rgba, surface)
  } else if (borderParsed.ok) {
    border = borderParsed.hex
  } else {
    warnings.push(`palette.${mode}.border: invalid`)
    border = safe.border
  }

  const elevated = asHex(src.elevated) || (mode === 'dark' ? mixSrgb(surface, text, 0.06) : mixSrgb(surface, WHITE, 0.4))
  const overlayC = asHex(src.overlay) || elevated
  const inset = asHex(src.inset) || mixSrgb(surface, BLACK, mode === 'dark' ? 0.12 : 0.04)
  const sidebar = asHex(src.sidebar) || mixSrgb(bg, BLACK, mode === 'dark' ? 0.12 : 0.04)
  const header = asHex(src.header) || sidebar
  const auth = asHex(src.auth) || mixSrgb(bg, BLACK, mode === 'dark' ? 0.12 : 0.04)
  const input = asHex(src.input) || inset
  const tableHeader = asHex(src.tableHeader) || inset
  const textSecondary = asHex(src.textSecondary) || mixSrgb(text, textMuted, 0.5)
  const borderSolid = borderParsed.ok && borderParsed.rgba ? compositeRgbaOnHex(borderParsed.rgba, surface) : border
  const borderLight = asHex(src.borderLight)
    || (src.borderLight && parseRgba(String(src.borderLight))
      ? compositeRgbaOnHex(String(src.borderLight), surface)
      : mixSrgb(borderSolid, surface, 0.5))
  const borderHover = asHex(src.borderHover) || mixSrgb(borderSolid, text, 0.2)

  let accent = req('accent', src.accent)
  const paletteSecondary = asHex(src.accentSecondary) || null
  const success = req('success', src.success)
  const warning = req('warning', src.warning)
  const danger = req('danger', src.danger)
  const info = req('info', src.info)

  const userAccent = resolveUserAccent(overlay)
  if (userAccent) accent = userAccent
  if (mode === 'light') {
    const fixed = correctAgainst(accent, [surface], 4.5)
    if (!fixed.ok) warnings.push(`palette.${mode}.accent: contrast unresolved`)
    accent = fixed.color
  }

  const textBg = [bg, surface, inset, overlayC]
  for (const [label, value] of [['text', text], ['textSecondary', textSecondary], ['textMuted', textMuted]] as const) {
    const fixed = correctAgainst(value, textBg, 4.5)
    if (!fixed.ok) warnings.push(`palette.${mode}.${label}: contrast unresolved`)
    if (label === 'text') {
      /* assign below */
    }
  }
  const textFix = correctAgainst(text, textBg, 4.5)
  const textSecondaryFix = correctAgainst(textSecondary, textBg, 4.5)
  const textMutedFix = correctAgainst(textMuted, textBg, 4.5)
  if (!textFix.ok) warnings.push(`palette.${mode}.text: contrast unresolved`)
  if (!textSecondaryFix.ok) warnings.push(`palette.${mode}.textSecondary: contrast unresolved`)
  if (!textMutedFix.ok) warnings.push(`palette.${mode}.textMuted: contrast unresolved`)

  const hover = mixSrgb(accent, BLACK, 0.2)
  const muted = rgbaFromHex(accent, 0.15)
  const derivedSecondary = mixSrgb(accent, '#8B5CF6', 0.6)
  let accentSecondary: string
  const userSec = asHex(overlay.accentSecondary) || null
  if (userSec) accentSecondary = userSec
  else if (userAccent) accentSecondary = derivedSecondary
  else if (paletteSecondary) accentSecondary = paletteSecondary
  else accentSecondary = derivedSecondary

  const fill1 = inset
  const fill2 = mixSrgb(inset, textFix.color, 0.08)
  const onAccent = onAccentFor(accent)

  let categorical: string[]
  const authorCat = Array.isArray(src._chartsCat)
    ? src._chartsCat.filter((c): c is string => isHex(c)).map((c) => normalizeHex(c))
    : null
  if (authorCat && authorCat.length >= 3) {
    categorical = userAccent ? [accent, ...authorCat.slice(1)] : authorCat.slice()
  } else {
    categorical = [accent, success, warning, danger, accentSecondary, info]
  }
  if (!chartOk(categorical, surface)) {
    warnings.push(`charts.categorical.${mode}: fallback safe sequence`)
    categorical = [...CHART_SAFE[mode]]
  }

  let sequential: string[]
  const authorSeq = Array.isArray(src._chartsSeq)
    ? src._chartsSeq.filter((c): c is string => isHex(c)).map((c) => normalizeHex(c))
    : null
  if (authorSeq && authorSeq.length >= 3 && !userAccent) {
    sequential = authorSeq
  } else {
    sequential = [0, 0.25, 0.5, 0.75, 1].map((t) => mixSrgb(surface, accent, t))
  }
  const increasing = relativeIncreasing(sequential)
  if (!increasing) {
    warnings.push(`charts.sequential.${mode}: fallback ends`)
    sequential = mode === 'dark' ? ['#303641', '#EEF0F4'] : ['#F4F5F8', '#141820']
  }

  return {
    bg, surface, elevated, overlay: overlayC, inset, sidebar, header, input, tableHeader, auth,
    text: textFix.color, textSecondary: textSecondaryFix.color, textMuted: textMutedFix.color,
    border, borderLight, borderHover, accent, accentSecondary, success, warning, danger, info,
    hover, muted, onAccent, fill1, fill2, categorical, sequential,
  }
}

function relativeIncreasing(colors: string[]): boolean {
  return monotonicLuminance(colors, true) || monotonicLuminance(colors, false)
}

function resolveUserAccent(overlay: ThemePackOverlay): string | null {
  if (overlay.accentKey === 'custom') {
    return asHex(overlay.accentCustom) || null
  }
  if (typeof overlay.accentKey === 'string' && overlay.accentKey && overlay.accentKey !== 'custom') {
    const preset = getPresetByKey(overlay.accentKey)
    return preset ? normalizeHex(preset.primary) : null
  }
  if (overlay.accentCustom && overlay.accentKey !== 'custom') return null
  return null
}

function collectUnknown(obj: Record<string, unknown>, known: Set<string>, prefix: string, isolated: Record<string, unknown>) {
  for (const key of Object.keys(obj)) {
    if (!known.has(key)) isolated[prefix ? `${prefix}.${key}` : key] = obj[key]
  }
}

export function parseThemePack(raw: unknown, overlay: ThemePackOverlay = {}): { ok: true, pack: ResolvedPack } | { ok: false, error: string } {
  const warnings: string[] = []
  const stripped: string[] = []
  const isolated: Record<string, unknown> = {}

  let data = raw
  if (typeof raw === 'string') {
    if (raw.length > MAX_BYTES) return { ok: false, error: 'too-large' }
    try {
      data = JSON.parse(raw)
    } catch {
      return { ok: false, error: 'invalid-json' }
    }
  }
  if (Array.isArray(data) || !isObj(data)) return { ok: false, error: 'not-object' }
  if (JSON.stringify(data).length > MAX_BYTES) return { ok: false, error: 'too-large' }
  if (depthOf(data) > MAX_DEPTH) return { ok: false, error: 'too-deep' }

  const obj = stripForbiddenKeys(data, '', stripped) as Record<string, unknown>
  if (obj.kind !== THEME_PACK_KIND) return { ok: false, error: 'kind' }
  if (obj.schemaVersion !== 1) return { ok: false, error: 'schemaVersion' }

  const schemaMinor = readNumber(obj.schemaMinor, 0, 0, 1_000_000, 'schemaMinor', warnings, true)
  if (typeof obj.id !== 'string' || !SLUG.test(obj.id)) return { ok: false, error: 'id' }
  if (typeof obj.name !== 'string' || obj.name.length === 0) return { ok: false, error: 'name' }
  const name = [...obj.name].slice(0, 64).join('')
  if (name !== obj.name) warnings.push('name: truncated')
  const version = typeof obj.version === 'string' ? obj.version : (warnings.push('version: invalid'), String(obj.version ?? ''))

  const palette = obj.palette
  if (!isObj(palette) || !isObj(palette.dark) || !isObj(palette.light)) return { ok: false, error: 'palette' }
  for (const mode of ['dark', 'light'] as const) {
    const p = palette[mode] as Record<string, unknown>
    for (const key of ['bg', 'surface', 'text', 'textMuted', 'border', 'accent', 'success', 'warning', 'danger', 'info']) {
      if (p[key] === undefined) return { ok: false, error: `palette.${mode}.${key}` }
    }
  }

  const derive = isObj(obj.derive) ? obj.derive : {}
  let strategyDeclared = typeof derive.strategy === 'string' ? derive.strategy : 'v1'
  if (strategyDeclared !== 'v1') warnings.push('derive.strategy: fallback v1')
  for (const key of Object.keys(derive)) {
    if (key === 'strategy') continue
    if (FORBIDDEN_DERIVE.has(key) || key.startsWith('derived')) {
      stripped.push(`derive.${key}`)
    } else {
      isolated[`derive.${key}`] = derive[key]
    }
  }

  const charts = isObj(obj.charts) ? obj.charts : {}
  const defaultsRaw = isObj(obj.defaults) ? obj.defaults : {}
  const variantsRaw = isObj(defaultsRaw.variants) ? defaultsRaw.variants : {}
  const surfacesRaw = isObj(obj.surfaces) ? obj.surfaces : {}
  const cardSurf = isObj(surfacesRaw.card) ? surfacesRaw.card : {}
  const insetSurf = isObj(surfacesRaw.inset) ? surfacesRaw.inset : {}

  const defaults = {
    mode: readEnum(defaultsRaw.mode, ENUMS.mode, 'dark', 'defaults.mode', warnings, isolated),
    density: readEnum(defaultsRaw.density, ENUMS.density, 'default', 'defaults.density', warnings, isolated),
    uiFont: readEnum(defaultsRaw.uiFont, ENUMS.uiFont, 'system', 'defaults.uiFont', warnings, isolated),
    sidebarWidth: readEnum(defaultsRaw.sidebarWidth, ENUMS.sidebarWidth, 'default', 'defaults.sidebarWidth', warnings, isolated),
    headerHeight: readEnum(defaultsRaw.headerHeight, ENUMS.headerHeight, 'default', 'defaults.headerHeight', warnings, isolated),
    radius: readEnum(defaultsRaw.radius, ENUMS.radius, 'default', 'defaults.radius', warnings, isolated),
    transparency: readBool(defaultsRaw.transparency, false, 'defaults.transparency', warnings),
    iconSet: readEnum(defaultsRaw.iconSet, ENUMS.iconSet, 'outline', 'defaults.iconSet', warnings, isolated),
    termFont: readEnum(defaultsRaw.termFont, ENUMS.termFont, 'jetbrains', 'defaults.termFont', warnings, isolated),
    termFontSize: readNumber(defaultsRaw.termFontSize, 14, 10, 24, 'defaults.termFontSize', warnings, true),
    termBgOpacity: readNumber(defaultsRaw.termBgOpacity, 1, 0, 1, 'defaults.termBgOpacity', warnings),
    chromeTexture: readEnum(defaultsRaw.chromeTexture, ENUMS.chromeTexture, 'none', 'defaults.chromeTexture', warnings, isolated),
    variants: {
      sidebar: readEnum(variantsRaw.sidebar, ENUMS.sidebar, 'marker', 'defaults.variants.sidebar', warnings, isolated),
      subnav: readEnum(variantsRaw.subnav, ENUMS.subnav, 'block', 'defaults.variants.subnav', warnings, isolated),
      card: readEnum(variantsRaw.card, ENUMS.card, 'outline', 'defaults.variants.card', warnings, isolated),
      iconContainer: readEnum(variantsRaw.iconContainer, ENUMS.iconContainer, 'none', 'defaults.variants.iconContainer', warnings, isolated),
      heading: readEnum(variantsRaw.heading, ENUMS.heading, 'compact', 'defaults.variants.heading', warnings, isolated),
    },
  }

  const themeTopEdge = readBool(cardSurf.topEdge, true, 'surfaces.card.topEdge', warnings)
  const themeHover = readNumber(insetSurf.hoverStrength, 1, 0, 1, 'surfaces.inset.hoverStrength', warnings)
  const overlaySurfaces = isObj(overlay.surfaces) ? overlay.surfaces : {}
  const overlayCard = isObj(overlaySurfaces.card) ? overlaySurfaces.card : {}
  const overlayInset = isObj(overlaySurfaces.inset) ? overlaySurfaces.inset : {}
  const cardTopEdge = typeof overlayCard.topEdge === 'boolean' ? overlayCard.topEdge : themeTopEdge
  const hoverStrength = typeof overlayInset.hoverStrength === 'number' && Number.isFinite(overlayInset.hoverStrength)
    ? Math.min(1, Math.max(0, overlayInset.hoverStrength))
    : themeHover

  const darkSrc = { ...(palette.dark as Record<string, unknown>), _chartsCat: charts.categorical, _chartsSeq: charts.sequential }
  const lightSrc = { ...(palette.light as Record<string, unknown>), _chartsCat: charts.categorical, _chartsSeq: charts.sequential }

  collectUnknown(obj, new Set([
    'kind', 'schemaVersion', 'schemaMinor', 'exportedAt', 'id', 'name', 'version',
    'palette', 'derive', 'defaults', 'tokens', 'surfaces', 'terminal', 'editor', 'charts', 'fonts', 'assets',
  ]), '', isolated)

  const overlayKnown = new Set(['accentKey', 'accentCustom', 'accentSecondary', 'surfaces'])
  collectUnknown(overlay, overlayKnown, '', isolated)
  if (isObj(overlaySurfaces)) {
    collectUnknown(overlaySurfaces, new Set(['card', 'inset']), 'surfaces', isolated)
    if (isObj(overlayInset)) collectUnknown(overlayInset, new Set(['hoverStrength']), 'surfaces.inset', isolated)
    if (isObj(overlayCard)) collectUnknown(overlayCard, new Set(['topEdge']), 'surfaces.card', isolated)
  }

  const pack: ResolvedPack = {
    id: obj.id as string,
    name,
    version,
    schemaVersion: 1,
    schemaMinor,
    strategyDeclared,
    strategyUsed: 'v1',
    defaults,
    surfaces: { cardTopEdge, hoverStrength },
    modes: {
      dark: deriveMode('dark', darkSrc, overlay, warnings),
      light: deriveMode('light', lightSrc, overlay, warnings),
    },
    warnings,
    stripped,
    isolated,
    overlay: { ...overlay },
  }
  return { ok: true, pack }
}

const RESTORE: Record<string, { keys: string[], prefixes: string[] }> = {
  'advanced.recipe': {
    keys: ['surfaces.card.topEdge', 'surfaces.inset.hoverStrength'],
    prefixes: ['surfaces'],
  },
  'advanced.secondary': { keys: ['accentSecondary'], prefixes: [] },
  'harmony.accent': { keys: ['accentKey', 'accentCustom'], prefixes: [] },
}

export function restoreOverlayGroup(
  overlay: ThemePackOverlay,
  isolated: Record<string, unknown>,
  group: string,
): { overlay: ThemePackOverlay, isolated: Record<string, unknown> } {
  const spec = RESTORE[group]
  const nextOverlay: ThemePackOverlay = { ...overlay }
  let nextIsolated = { ...isolated }
  if (!spec) return { overlay: nextOverlay, isolated: nextIsolated }
  if (group === 'advanced.recipe') {
    const surfaces = isObj(nextOverlay.surfaces) ? { ...nextOverlay.surfaces } : {}
    if (isObj(surfaces.card)) {
      const card = { ...surfaces.card }
      delete card.topEdge
      surfaces.card = card
    }
    if (isObj(surfaces.inset)) {
      const inset = { ...surfaces.inset }
      delete inset.hoverStrength
      surfaces.inset = inset
    }
    nextOverlay.surfaces = surfaces
  }
  if (group === 'advanced.secondary') delete nextOverlay.accentSecondary
  if (group === 'harmony.accent') {
    delete nextOverlay.accentKey
    delete nextOverlay.accentCustom
  }
  for (const prefix of spec.prefixes) nextIsolated = dropIsolatedByPrefix(nextIsolated, prefix)
  return { overlay: nextOverlay, isolated: nextIsolated }
}

function prefixMatch(path: string, prefix: string): boolean {
  const segs = path.split('.')
  const pre = prefix.split('.')
  return pre.every((s, i) => segs[i] === s)
}

export function dropIsolatedByPrefix(isolated: Record<string, unknown>, prefix: string): Record<string, unknown> {
  const next = { ...isolated }
  for (const path of Object.keys(next)) {
    if (prefixMatch(path, prefix)) delete next[path]
  }
  return next
}
