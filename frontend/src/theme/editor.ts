import type * as Monaco from 'monaco-editor'
import { mixSrgb, normalizeHex, relativeLuminance } from './pack-color.ts'
import { getTermThemeByKey } from '../utils/terminal-theme.ts'

export type MonacoBuiltinTheme = 'vs' | 'vs-dark' | 'hc-black'

export function monacoThemeOf(theme: string | undefined): MonacoBuiltinTheme {
  if (theme === 'vs' || theme === 'vs-dark' || theme === 'hc-black') return theme
  return 'vs-dark'
}

function termBackgroundHex(termTheme: string): string {
  const bg = getTermThemeByKey(termTheme).background || '#1E1E2E'
  if (bg.startsWith('#')) {
    try { return normalizeHex(bg.slice(0, 7)) } catch { return '#1E1E2E' }
  }
  const m = bg.match(/rgba?\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)/i)
  if (!m) return '#1E1E2E'
  return normalizeHex('#' + [m[1], m[2], m[3]].map((n) => Number(n).toString(16).padStart(2, '0')).join(''))
}

const LIGHT_PLATE = '#F4F5F8'
const DARK_PLATE = '#191C23'

export function editorWorkspaceBackground(
  termTheme: string,
  opacity: number,
  surfaceHex: string,
  syntax: MonacoBuiltinTheme = 'vs-dark',
): string {
  let surface = syntax === 'vs' ? LIGHT_PLATE : DARK_PLATE
  try {
    const given = normalizeHex(surfaceHex)
    if (syntax === 'vs' && relativeLuminance(given) >= 0.45) surface = given
    if (syntax !== 'vs' && relativeLuminance(given) <= 0.25) surface = given
  } catch { /* keep plate */ }
  const term = termBackgroundHex(termTheme)
  const termFits = syntax === 'vs' ? relativeLuminance(term) >= 0.5 : relativeLuminance(term) <= 0.2
  if (!termFits) return surface
  const t = Math.min(1, Math.max(0, opacity))
  const mixed = mixSrgb(surface, term, t)
  if (syntax === 'vs' && relativeLuminance(mixed) < 0.62) return surface
  if (syntax !== 'vs' && relativeLuminance(mixed) > 0.14) return surface
  return mixed
}

export function applyMonacoWorkspaceTheme(
  monaco: typeof Monaco,
  editorTheme: string | undefined,
  termTheme: string,
  opacity: number,
  surfaceHex: string,
) {
  const syntax = monacoThemeOf(editorTheme)
  const bg = editorWorkspaceBackground(termTheme, opacity, surfaceHex, syntax)
  monaco.editor.defineTheme('xp-follow-term', {
    base: syntax === 'vs' ? 'vs' : 'vs-dark',
    inherit: true,
    rules: [],
    colors: {
      'editor.background': bg,
      'editorGutter.background': bg,
      'minimap.background': bg,
    },
  })
  monaco.editor.setTheme('xp-follow-term')
}
