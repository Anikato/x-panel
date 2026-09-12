import { parseThemePack, THEME_PACK_KIND } from './pack-parse.ts'
import { packToThemeDefinition } from './pack-adapter.ts'
import { BUILTIN_THEME_IDS } from './types.ts'
import type { ThemeDefinition } from './types.ts'

const STORAGE_KEY = 'xp-installed-theme-packs-v1'
let memoryStore: Record<string, unknown> = {}

function canUseStorage(): boolean {
  return typeof localStorage !== 'undefined'
}

function readAll(): Record<string, unknown> {
  if (!canUseStorage()) return { ...memoryStore }
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw)
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? parsed : {}
  } catch {
    return {}
  }
}

function writeAll(map: Record<string, unknown>) {
  memoryStore = { ...map }
  if (!canUseStorage()) return
  localStorage.setItem(STORAGE_KEY, JSON.stringify(map))
}

export function resetInstalledPacksForTests() {
  memoryStore = {}
  if (canUseStorage()) localStorage.removeItem(STORAGE_KEY)
}

export function listInstalledPackRaw(): Record<string, unknown> {
  return readAll()
}

export function getInstalledPackRaw(id: string): Record<string, unknown> | null {
  const row = readAll()[id]
  return row && typeof row === 'object' && !Array.isArray(row) ? row as Record<string, unknown> : null
}

export function hasInstalledPack(id: string): boolean {
  return Boolean(getInstalledPackRaw(id))
}

export function installThemePack(raw: unknown, opts?: { overwriteBuiltin?: boolean }): { ok: true, id: string } | { ok: false, error: string } {
  const parsed = parseThemePack(raw)
  if (!parsed.ok) return { ok: false, error: parsed.error }
  const id = parsed.pack.id
  if ((BUILTIN_THEME_IDS as readonly string[]).includes(id) && !opts?.overwriteBuiltin) {
    return { ok: false, error: 'builtin' }
  }
  const map = readAll()
  const payload = typeof raw === 'string' ? JSON.parse(raw) : raw
  map[id] = payload
  writeAll(map)
  return { ok: true, id }
}

export function uninstallThemePack(id: string) {
  if ((BUILTIN_THEME_IDS as readonly string[]).includes(id)) return
  const map = readAll()
  delete map[id]
  writeAll(map)
}

export function installedThemeDefinitions(): ThemeDefinition[] {
  const out: ThemeDefinition[] = []
  for (const [id, raw] of Object.entries(readAll())) {
    if (!raw || typeof raw !== 'object') continue
    const parsed = parseThemePack(raw)
    if (!parsed.ok) continue
    out.push(packToThemeDefinition(parsed.pack, raw as Record<string, unknown>))
    void id
  }
  return out
}

export function definitionFromInstalled(id: string): ThemeDefinition | null {
  const raw = getInstalledPackRaw(id)
  if (!raw) return null
  const parsed = parseThemePack(raw)
  if (!parsed.ok) return null
  return packToThemeDefinition(parsed.pack, raw)
}
