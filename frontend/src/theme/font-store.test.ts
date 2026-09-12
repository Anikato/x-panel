import assert from 'node:assert/strict'
import test from 'node:test'
import { formatFromName, sniffFontFormat, validateFontFile } from './font-store.ts'

test('sniff woff2/woff/otf/ttf magic', () => {
  assert.equal(sniffFontFormat(Uint8Array.from([0x77, 0x4f, 0x46, 0x32])), 'woff2')
  assert.equal(sniffFontFormat(Uint8Array.from([0x77, 0x4f, 0x46, 0x46])), 'woff')
  assert.equal(sniffFontFormat(Uint8Array.from([0x4f, 0x54, 0x54, 0x4f])), 'opentype')
  assert.equal(sniffFontFormat(Uint8Array.from([0x00, 0x01, 0x00, 0x00])), 'truetype')
  assert.equal(sniffFontFormat(Uint8Array.from([0x3c, 0x73, 0x63, 0x72])), null)
})

test('reject mismatched extension, scripts, and oversized files', () => {
  const ttf = Uint8Array.from([0x00, 0x01, 0x00, 0x00, 0, 0, 0, 0])
  assert.equal(validateFontFile('face.ttf', ttf).ok, true)
  assert.equal(validateFontFile('face.woff2', ttf).ok, false)
  assert.equal(validateFontFile('payload.js', ttf).ok, false)
  const allowed = new Uint8Array(2 * 1024 * 1024 + 1)
  allowed.set([0x77, 0x4f, 0x46, 0x32])
  assert.equal(validateFontFile('mid.woff2', allowed).ok, true)
  const huge = new Uint8Array(8 * 1024 * 1024 + 1)
  huge.set([0x77, 0x4f, 0x46, 0x32])
  const over = validateFontFile('big.woff2', huge)
  assert.equal(over.ok, false)
  if (!over.ok) assert.equal(over.error, 'too-large')
})

test('formatFromName only allows font suffixes', () => {
  assert.equal(formatFromName('LXGW.woff2'), 'woff2')
  assert.equal(formatFromName('face.OTF'), 'opentype')
  assert.equal(formatFromName('face.ttf.exe'), null)
})
