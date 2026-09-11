import type { ThemeDefinition, ThemeId } from './types.ts'
import { atelierTheme } from './themes/atelier.ts'
import { lumenTheme } from './themes/lumen.ts'

const THEMES: Record<ThemeId, ThemeDefinition> = {
  atelier: atelierTheme,
  lumen: lumenTheme,
}

export function listThemes(): ThemeDefinition[] {
  return [atelierTheme, lumenTheme]
}

export function getTheme(id: string): ThemeDefinition {
  return THEMES[id as ThemeId] || atelierTheme
}

export function hasTheme(id: string): id is ThemeId {
  return id === 'atelier' || id === 'lumen'
}
