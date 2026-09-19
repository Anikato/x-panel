import assert from 'node:assert/strict'
import test from 'node:test'
import { AVATAR_COUNT, avatarIndex } from './user-avatar.ts'

test('avatarIndex is stable for the same username', () => {
  assert.equal(avatarIndex('admin'), avatarIndex('admin'))
})

test('avatarIndex stays inside the avatar set', () => {
  for (const name of ['admin', 'root', 'kevin', 'ops', '']) {
    const idx = avatarIndex(name)
    assert.ok(idx >= 0 && idx < AVATAR_COUNT, `${name} -> ${idx}`)
  }
})

test('different usernames can land on different avatars', () => {
  const indexes = new Set(['admin', 'alice', 'bob', 'ops', 'root', 'xpanel'].map(avatarIndex))
  assert.ok(indexes.size >= 2)
})
