import { loadNavMemory, rememberModulePath } from './memory.ts'
import type { NavGroupId, NavLocation, NavModule } from './registry.ts'
import { parseStoredLocation, resolveRememberedPath, resolveModule, serializeLocation } from './registry.ts'

const LAST_KEY = 'xp-nav-last'
const GROUP_KEY = 'xp-nav-groups'

export function readLastPaths() {
  if (typeof localStorage === 'undefined') return new Map<string, string>()
  return loadNavMemory(localStorage.getItem(LAST_KEY))
}

export function writeLastPaths(store: Map<string, string>) {
  if (typeof localStorage === 'undefined') return
  localStorage.setItem(LAST_KEY, JSON.stringify(Object.fromEntries(store)))
}

export function rememberCurrentRoute(path: string, query: Record<string, string> = {}) {
  const mod = resolveModule(path)
  if (!mod) return
  const loc: NavLocation = { path, query }
  const store = readLastPaths()
  rememberModulePath(mod.id, serializeLocation(loc), store)
  writeLastPaths(store)
}

export function destinationFor(mod: NavModule): NavLocation {
  return resolveRememberedPath(mod, readLastPaths().get(mod.id))
}

export function readGroupCollapsed(): Partial<Record<NavGroupId, boolean>> {
  if (typeof localStorage === 'undefined') return {}
  try {
    const parsed = JSON.parse(localStorage.getItem(GROUP_KEY) || '{}')
    return parsed && typeof parsed === 'object' ? parsed : {}
  } catch {
    return {}
  }
}

export function writeGroupCollapsed(state: Partial<Record<NavGroupId, boolean>>) {
  if (typeof localStorage === 'undefined') return
  localStorage.setItem(GROUP_KEY, JSON.stringify(state))
}
