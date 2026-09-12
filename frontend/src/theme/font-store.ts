import { CUSTOM_UI_FONT_FAMILY } from './shared-tokens.ts'

export { CUSTOM_UI_FONT_FAMILY }

export const CUSTOM_FONT_MAX_BYTES = 8 * 1024 * 1024
export const CUSTOM_FONT_META_KEY = 'xp-custom-ui-font-meta-v1'

const DB_NAME = 'xp-custom-fonts-v1'
const STORE = 'faces'
const RECORD_KEY = 'ui'

export type FontFormat = 'woff2' | 'woff' | 'truetype' | 'opentype'

export interface CustomFontMeta {
  family: string
  format: FontFormat
  name: string
  size: number
}

let loadedFace: FontFace | null = null

const draft: { gen: number, rec: { buf: ArrayBuffer, meta: CustomFontMeta } | null | undefined } = {
  gen: 0,
  rec: undefined,
}

export function beginFontDraft() {
  draft.gen += 1
  draft.rec = undefined
}

export function cancelFontDraft() {
  draft.gen += 1
  draft.rec = undefined
}

export function setDraftFont(rec: { buf: ArrayBuffer, meta: CustomFontMeta } | null) {
  draft.rec = rec
}

export function hasFontDraft() {
  return draft.rec !== undefined
}

export function peekFontDraft() {
  return draft.rec
}

export function fontDraftGen() {
  return draft.gen
}

export function sniffFontFormat(bytes: Uint8Array): FontFormat | null {
  if (bytes.length < 4) return null
  const tag = String.fromCharCode(bytes[0], bytes[1], bytes[2], bytes[3])
  if (tag === 'wOF2') return 'woff2'
  if (tag === 'wOFF') return 'woff'
  if (tag === 'OTTO') return 'opentype'
  if (bytes[0] === 0x00 && bytes[1] === 0x01 && bytes[2] === 0x00 && bytes[3] === 0x00) return 'truetype'
  return null
}

export function formatFromName(name: string): FontFormat | null {
  const lower = name.toLowerCase()
  if (lower.endsWith('.woff2')) return 'woff2'
  if (lower.endsWith('.woff')) return 'woff'
  if (lower.endsWith('.ttf')) return 'truetype'
  if (lower.endsWith('.otf')) return 'opentype'
  return null
}

export function validateFontFile(name: string, bytes: Uint8Array): { ok: true, format: FontFormat } | { ok: false, error: 'too-large' | 'bad-format' } {
  if (bytes.byteLength > CUSTOM_FONT_MAX_BYTES) return { ok: false, error: 'too-large' }
  const sniffed = sniffFontFormat(bytes)
  const named = formatFromName(name)
  if (!sniffed || !named || sniffed !== named) return { ok: false, error: 'bad-format' }
  return { ok: true, format: sniffed }
}

export function readCustomFontMeta(): CustomFontMeta | null {
  if (typeof localStorage === 'undefined') return null
  try {
    const raw = localStorage.getItem(CUSTOM_FONT_META_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw) as Partial<CustomFontMeta>
    if (!parsed || typeof parsed.name !== 'string' || typeof parsed.format !== 'string') return null
    if (parsed.format !== 'woff2' && parsed.format !== 'woff' && parsed.format !== 'truetype' && parsed.format !== 'opentype') return null
    return {
      family: CUSTOM_UI_FONT_FAMILY,
      format: parsed.format,
      name: parsed.name.slice(0, 128),
      size: typeof parsed.size === 'number' ? parsed.size : 0,
    }
  } catch {
    return null
  }
}

function writeMeta(meta: CustomFontMeta | null) {
  if (typeof localStorage === 'undefined') return
  try {
    if (!meta) localStorage.removeItem(CUSTOM_FONT_META_KEY)
    else localStorage.setItem(CUSTOM_FONT_META_KEY, JSON.stringify(meta))
  } catch {
    /* quota */
  }
}

function openDb(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, 1)
    req.onupgradeneeded = () => {
      if (!req.result.objectStoreNames.contains(STORE)) req.result.createObjectStore(STORE)
    }
    req.onsuccess = () => resolve(req.result)
    req.onerror = () => reject(req.error)
  })
}

export async function fileToCustomFont(file: File): Promise<{ buf: ArrayBuffer, meta: CustomFontMeta }> {
  const buf = await file.arrayBuffer()
  const checked = validateFontFile(file.name, new Uint8Array(buf))
  if (!checked.ok) throw new Error(checked.error)
  return {
    buf,
    meta: {
      family: CUSTOM_UI_FONT_FAMILY,
      format: checked.format,
      name: file.name.replace(/\.[^.]+$/, '').slice(0, 64) || 'Custom',
      size: buf.byteLength,
    },
  }
}

export async function saveCustomFont(buf: ArrayBuffer, meta: CustomFontMeta): Promise<void> {
  const db = await openDb()
  await new Promise<void>((resolve, reject) => {
    const tx = db.transaction(STORE, 'readwrite')
    tx.oncomplete = () => resolve()
    tx.onerror = () => reject(tx.error)
    tx.objectStore(STORE).put({ buf, meta }, RECORD_KEY)
  })
  db.close()
  writeMeta(meta)
}

export async function loadCustomFont(): Promise<{ buf: ArrayBuffer, meta: CustomFontMeta } | null> {
  if (typeof indexedDB === 'undefined') return null
  try {
    const db = await openDb()
    const rec = await new Promise<{ buf: ArrayBuffer, meta: CustomFontMeta } | undefined>((resolve, reject) => {
      const tx = db.transaction(STORE, 'readonly')
      const req = tx.objectStore(STORE).get(RECORD_KEY)
      req.onsuccess = () => resolve(req.result)
      req.onerror = () => reject(req.error)
    })
    db.close()
    if (!rec?.buf) return null
    return rec
  } catch {
    return null
  }
}

export async function clearCustomFont(): Promise<void> {
  if (typeof indexedDB !== 'undefined') {
    try {
      const db = await openDb()
      await new Promise<void>((resolve, reject) => {
        const tx = db.transaction(STORE, 'readwrite')
        tx.oncomplete = () => resolve()
        tx.onerror = () => reject(tx.error)
        tx.objectStore(STORE).delete(RECORD_KEY)
      })
      db.close()
    } catch {
      /* keep going */
    }
  }
  writeMeta(null)
  unloadCustomFontFace()
}

export function unloadCustomFontFace() {
  if (typeof document === 'undefined' || !loadedFace) return
  document.fonts.delete(loadedFace)
  loadedFace = null
}

export async function flushFontDraft(): Promise<void> {
  if (draft.rec === undefined) return
  if (draft.rec === null) await clearCustomFont()
  else await saveCustomFont(draft.rec.buf, draft.rec.meta)
  draft.rec = undefined
}

export async function applyCustomFontFace(): Promise<boolean> {
  if (typeof document === 'undefined' || typeof FontFace === 'undefined') return false
  if (draft.rec === null) {
    unloadCustomFontFace()
    return false
  }
  const rec = draft.rec || await loadCustomFont()
  if (!rec) return false
  unloadCustomFontFace()
  const face = new FontFace(CUSTOM_UI_FONT_FAMILY, rec.buf, {
    style: 'normal',
    display: 'swap',
  })
  await face.load()
  document.fonts.add(face)
  loadedFace = face
  return true
}
