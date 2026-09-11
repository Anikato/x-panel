import { moduleContainsPath, NAV_MODULES, parseStoredLocation } from './registry.ts'

export function rememberModulePath(moduleId: string, path: string, store: Map<string, string>) {
  const mod = NAV_MODULES.find((item) => item.id === moduleId)
  if (!mod) return
  if (!moduleContainsPath(mod, path)) return
  const loc = parseStoredLocation(path)
  if (/^\/website\/websites\/[^/]+$/.test(loc.path)) return
  store.set(moduleId, path)
}

export function loadNavMemory(raw: string | null): Map<string, string> {
  const store = new Map<string, string>()
  if (!raw) return store
  try {
    const parsed = JSON.parse(raw)
    if (!parsed || typeof parsed !== 'object') return store
    for (const [key, value] of Object.entries(parsed)) {
      if (typeof value === 'string') store.set(key, value)
    }
  } catch {
    return store
  }
  return store
}

export function dumpNavMemory(store: Map<string, string>) {
  return JSON.stringify(Object.fromEntries(store))
}
