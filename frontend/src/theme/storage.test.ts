import assert from 'node:assert/strict'
import test from 'node:test'
import { APPEARANCE_STORAGE_KEY, readLocalPreference, writeLocalPreference } from './storage.ts'
import { DEFAULT_PREFERENCE } from './types.ts'

function mockStorage() {
  const mem = new Map<string, string>()
  let fail = false
  const localStorage = {
    getItem: (key: string) => mem.get(key) ?? null,
    setItem: (key: string, value: string) => {
      if (fail) throw new Error('quota')
      mem.set(key, String(value))
    },
    removeItem: (key: string) => { mem.delete(key) },
    clear: () => mem.clear(),
    key: () => null,
    get length() { return mem.size },
  }
  Object.defineProperty(globalThis, 'localStorage', { value: localStorage, configurable: true })
  return {
    mem,
    failWrites() { fail = true },
  }
}

test('empty local preference is not written as an authoritative default', () => {
  const { mem } = mockStorage()
  const pref = readLocalPreference()
  assert.equal(pref.themeId, 'atelier')
  assert.equal(mem.has(APPEARANCE_STORAGE_KEY), false)
})

test('writeLocalPreference returns false instead of throwing on quota errors', () => {
  const { failWrites } = mockStorage()
  failWrites()
  assert.equal(writeLocalPreference(DEFAULT_PREFERENCE), false)
})
