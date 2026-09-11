export const CHROME_IMAGE_KEY = 'xp-chrome-image-v1'
export const TERM_IMAGE_KEY = 'xp-term-image-v1'
const MAX_EDGE = 1920
const JPEG_QUALITY = 0.82
const MAX_SOURCE_BYTES = 8 * 1024 * 1024

type WallpaperKind = 'chrome' | 'term'

const draft: {
  gen: number
  chrome: Map<string, string>
  term: Map<string, string>
} = {
  gen: 0,
  chrome: new Map(),
  term: new Map(),
}

function baseKey(kind: WallpaperKind) {
  return kind === 'chrome' ? CHROME_IMAGE_KEY : TERM_IMAGE_KEY
}

function themeKey(kind: WallpaperKind, themeId: string) {
  return `${baseKey(kind)}:${themeId}`
}

export function wallpaperDraftGen() {
  return draft.gen
}

export function hasWallpaperDraft() {
  return draft.chrome.size > 0 || draft.term.size > 0
}

export function beginWallpaperDraft() {
  draft.gen += 1
  draft.chrome.clear()
  draft.term.clear()
  return draft.gen
}

export function cancelWallpaperDraft() {
  draft.gen += 1
  draft.chrome.clear()
  draft.term.clear()
}

export function setDraftWallpaper(kind: WallpaperKind, themeId: string, dataUrl: string) {
  draft[kind].set(themeId, dataUrl)
}

function readStoredItem(key: string): string | null {
  if (typeof localStorage === 'undefined') return null
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

function writeStoredItem(key: string, dataUrl: string) {
  if (typeof localStorage === 'undefined') return
  localStorage.setItem(key, dataUrl)
}

export function sanitizeWallpaperUrl(raw: string): string {
  const url = (raw || '').trim().slice(0, 2048)
  if (!url) return ''
  try {
    const parsed = new URL(url)
    if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') return ''
    return parsed.href
  } catch {
    return ''
  }
}

export function readCommittedWallpaper(kind: WallpaperKind, themeId: string): string {
  const keyed = readStoredItem(themeKey(kind, themeId))
  if (keyed !== null) return keyed
  return readStoredItem(baseKey(kind)) || ''
}

export function readEffectiveWallpaper(kind: WallpaperKind, themeId: string): string {
  if (draft[kind].has(themeId)) return draft[kind].get(themeId) || ''
  return readCommittedWallpaper(kind, themeId)
}

export function wallpaperJobMatches(gen: number, themeId: string, currentThemeId: string) {
  return gen === draft.gen && themeId === currentThemeId
}

export function snapshotWallpaperDraftTargets(): Array<{ key: string, prev: string | null }> {
  const snap: Array<{ key: string, prev: string | null }> = []
  const collect = (kind: WallpaperKind) => {
    for (const themeId of draft[kind].keys()) {
      const key = themeKey(kind, themeId)
      snap.push({ key, prev: readStoredItem(key) })
    }
  }
  collect('chrome')
  collect('term')
  return snap
}

export function restoreWallpaperSnapshot(snap: Array<{ key: string, prev: string | null }>) {
  if (typeof localStorage === 'undefined') return
  for (const { key, prev } of [...snap].reverse()) {
    try {
      if (prev === null) localStorage.removeItem(key)
      else localStorage.setItem(key, prev)
    } catch {
      /* keep going so as many keys as possible return to the last committed set */
    }
  }
}

export function flushWallpaperDraftToStorage() {
  const rollback = snapshotWallpaperDraftTargets()
  try {
    const writeKind = (kind: WallpaperKind) => {
      for (const [themeId, dataUrl] of draft[kind]) {
        writeStoredItem(themeKey(kind, themeId), dataUrl)
      }
    }
    writeKind('chrome')
    writeKind('term')
  } catch {
    restoreWallpaperSnapshot(rollback)
    throw new Error('quota')
  }
}

export function clearWallpaperDraftMaps() {
  draft.chrome.clear()
  draft.term.clear()
}

export function commitWallpaperDraft() {
  flushWallpaperDraftToStorage()
  clearWallpaperDraftMaps()
}

export function readWallpaperData(kind: WallpaperKind, themeId = 'atelier'): string {
  return readEffectiveWallpaper(kind, themeId)
}

export function writeWallpaperData(kind: WallpaperKind, dataUrl: string, themeId = 'atelier') {
  writeStoredItem(themeKey(kind, themeId), dataUrl)
}

export async function fileToWallpaperDataUrl(file: File): Promise<string> {
  if (!file.type.startsWith('image/')) throw new Error('not-image')
  if (file.size > MAX_SOURCE_BYTES) throw new Error('too-large')
  const bitmap = await createImageBitmap(file)
  const scale = Math.min(1, MAX_EDGE / Math.max(bitmap.width, bitmap.height))
  const width = Math.max(1, Math.round(bitmap.width * scale))
  const height = Math.max(1, Math.round(bitmap.height * scale))
  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('canvas')
  ctx.drawImage(bitmap, 0, 0, width, height)
  bitmap.close()
  return canvas.toDataURL('image/jpeg', JPEG_QUALITY)
}

export function cssImage(url: string): string {
  if (!url) return 'none'
  const safe = url.replace(/\\/g, '\\\\').replace(/"/g, '\\"')
  return `url("${safe}")`
}
