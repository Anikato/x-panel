import assert from 'node:assert/strict'
import test from 'node:test'
import { searchShortcutLabel } from './search-shortcut.ts'

test('searchShortcutLabel uses Command on mac and Ctrl elsewhere', () => {
  assert.equal(searchShortcutLabel('MacIntel'), '⌘K')
  assert.equal(searchShortcutLabel('Linux x86_64'), 'Ctrl+K')
  assert.equal(searchShortcutLabel('Win32'), 'Ctrl+K')
})
