import assert from 'node:assert/strict'
import test from 'node:test'
import {
  beginWallpaperDraft,
  cancelWallpaperDraft,
  commitWallpaperDraft,
  cssImage,
  flushWallpaperDraftToStorage,
  readEffectiveWallpaper,
  sanitizeWallpaperUrl,
  setDraftWallpaper,
  wallpaperDraftGen,
  wallpaperJobMatches,
} from './wallpaper-store.ts'

test('sanitizeWallpaperUrl only keeps http(s) addresses', () => {
  assert.equal(sanitizeWallpaperUrl(''), '')
  assert.equal(sanitizeWallpaperUrl('javascript:alert(1)'), '')
  assert.equal(sanitizeWallpaperUrl('data:image/png;base64,abcd'), '')
  assert.equal(sanitizeWallpaperUrl('ftp://example.com/a.png'), '')
  assert.equal(sanitizeWallpaperUrl('https://cdn.example.com/wall.jpg'), 'https://cdn.example.com/wall.jpg')
  assert.equal(sanitizeWallpaperUrl('http://10.10.10.2/bg.png'), 'http://10.10.10.2/bg.png')
})

test('cssImage quotes and escapes urls', () => {
  assert.equal(cssImage(''), 'none')
  assert.equal(cssImage('https://a.example/b.jpg'), 'url("https://a.example/b.jpg")')
  assert.equal(cssImage('https://a.example/x"y.jpg'), 'url("https://a.example/x\\"y.jpg")')
})

test('wallpaper drafts do not commit until apply, and cancel restores the previous image', () => {
  const mem = new Map<string, string>()
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    value: {
      getItem: (key: string) => mem.get(key) ?? null,
      setItem: (key: string, value: string) => { mem.set(key, String(value)) },
      removeItem: (key: string) => { mem.delete(key) },
      clear: () => mem.clear(),
      key: () => null,
      get length() { return mem.size },
    },
  })
  beginWallpaperDraft()
  setDraftWallpaper('chrome', 'atelier', 'data:image/jpeg;base64,AAA')
  commitWallpaperDraft()
  assert.equal(readEffectiveWallpaper('chrome', 'atelier'), 'data:image/jpeg;base64,AAA')

  beginWallpaperDraft()
  setDraftWallpaper('chrome', 'atelier', 'data:image/jpeg;base64,BBB')
  assert.equal(readEffectiveWallpaper('chrome', 'atelier'), 'data:image/jpeg;base64,BBB')
  cancelWallpaperDraft()
  assert.equal(readEffectiveWallpaper('chrome', 'atelier'), 'data:image/jpeg;base64,AAA')

  beginWallpaperDraft()
  setDraftWallpaper('chrome', 'lumen', 'data:image/jpeg;base64,CCC')
  commitWallpaperDraft()
  assert.equal(readEffectiveWallpaper('chrome', 'atelier'), 'data:image/jpeg;base64,AAA')
  assert.equal(readEffectiveWallpaper('chrome', 'lumen'), 'data:image/jpeg;base64,CCC')
})

test('failed wallpaper flush rolls back storage and keeps the draft', () => {
  const mem = new Map<string, string>()
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    value: {
      getItem: (key: string) => mem.get(key) ?? null,
      setItem: (key: string, value: string) => {
        if (key.includes('lumen')) throw new Error('quota')
        mem.set(key, String(value))
      },
      removeItem: (key: string) => { mem.delete(key) },
      clear: () => mem.clear(),
      key: () => null,
      get length() { return mem.size },
    },
  })
  beginWallpaperDraft()
  setDraftWallpaper('chrome', 'atelier', 'data:image/jpeg;base64,OK')
  commitWallpaperDraft()
  beginWallpaperDraft()
  setDraftWallpaper('chrome', 'atelier', 'data:image/jpeg;base64,NEW')
  setDraftWallpaper('chrome', 'lumen', 'data:image/jpeg;base64,OTHER')
  assert.throws(() => flushWallpaperDraftToStorage())
  assert.equal(readEffectiveWallpaper('chrome', 'atelier'), 'data:image/jpeg;base64,NEW')
  assert.match(mem.get('xp-chrome-image-v1:atelier') || '', /OK/)
})

test('wallpaper jobs are rejected after cancel or theme change', () => {
  beginWallpaperDraft()
  const gen = wallpaperDraftGen()
  assert.equal(wallpaperJobMatches(gen, 'atelier', 'atelier'), true)
  assert.equal(wallpaperJobMatches(gen, 'atelier', 'lumen'), false)
  cancelWallpaperDraft()
  assert.equal(wallpaperJobMatches(gen, 'atelier', 'atelier'), false)
})
