import assert from 'node:assert/strict'
import test from 'node:test'
import { relativeLuminance } from './pack-color.ts'
import { editorWorkspaceBackground, monacoThemeOf } from './editor.ts'

test('monacoThemeOf maps light editor to vs', () => {
  assert.equal(monacoThemeOf('vs'), 'vs')
  assert.equal(monacoThemeOf('vs-dark'), 'vs-dark')
  assert.equal(monacoThemeOf(undefined), 'vs-dark')
})

test('light editor workspace is opaque 6-digit hex and stays light even if terminal is dark', () => {
  const bg = editorWorkspaceBackground('default', 0.5, '#F4F5F8', 'vs')
  assert.match(bg, /^#[0-9A-F]{6}$/)
  assert.ok(relativeLuminance(bg) > 0.55)
})

test('dark editor workspace following default terminal stays dark and opaque', () => {
  const bg = editorWorkspaceBackground('default', 0.5, '#191C23', 'vs-dark')
  assert.match(bg, /^#[0-9A-F]{6}$/)
  assert.ok(relativeLuminance(bg) < 0.18)
})

test('dark editor does not keep a paper terminal plate', () => {
  const bg = editorWorkspaceBackground('paper', 1, '#191C23', 'vs-dark')
  assert.match(bg, /^#[0-9A-F]{6}$/)
  assert.ok(relativeLuminance(bg) < 0.14)
  const half = editorWorkspaceBackground('paper', 0.5, '#191C23', 'vs-dark')
  assert.ok(relativeLuminance(half) < 0.14)
})
