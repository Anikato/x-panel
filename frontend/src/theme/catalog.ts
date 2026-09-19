import type { ThemeDefinition, ThemeId } from './types.ts'
import { BUILTIN_THEME_IDS } from './types.ts'
import { atelierTheme } from './themes/atelier.ts'
import { lumenTheme } from './themes/lumen.ts'
import { inkTheme } from './themes/ink.ts'
import { harborTheme } from './themes/harbor.ts'
import { quartzTheme } from './themes/quartz.ts'
import { definitionFromInstalled, hasInstalledPack, installedThemeDefinitions } from './pack-store.ts'

const BUILTINS: Record<string, ThemeDefinition> = {
  atelier: atelierTheme,
  lumen: lumenTheme,
  ink: inkTheme,
  harbor: harborTheme,
  quartz: quartzTheme,
}

export function isBuiltinTheme(id: string): boolean {
  return (BUILTIN_THEME_IDS as readonly string[]).includes(id)
}

export function listThemes(): ThemeDefinition[] {
  return [atelierTheme, lumenTheme, inkTheme, harborTheme, quartzTheme, ...installedThemeDefinitions()]
}

export function getTheme(id: string): ThemeDefinition {
  if (BUILTINS[id]) return BUILTINS[id]
  return definitionFromInstalled(id) || atelierTheme
}

export function hasTheme(id: string): boolean {
  return isBuiltinTheme(id) || hasInstalledPack(id)
}

export function isThemeSlug(id: string): id is ThemeId {
  return /^[a-z0-9][a-z0-9-]{1,63}$/.test(id)
}
