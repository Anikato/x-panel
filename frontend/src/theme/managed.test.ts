import assert from 'node:assert/strict'
import test from 'node:test'

import { applyManagedAppearance, parseManagedAppearance } from './managed.ts'
import { DEFAULT_PREFERENCE } from './types.ts'

test('managed appearance applies the fleet preset while preserving personal settings', () => {
  const personal = {
    ...DEFAULT_PREFERENCE,
    mode: 'light' as const,
    overridesByTheme: {
      harbor: { accentCustom: '#123456', density: 'compact' as const, termFontSize: 17 },
    },
  }
  const managed = parseManagedAppearance('{"schemaVersion":1,"themeId":"harbor","mode":"dark","accent":"#ABCDEF"}')
  assert.ok(managed)

  const effective = applyManagedAppearance(personal, managed)
  assert.equal(effective.themeId, 'harbor')
  assert.equal(effective.mode, 'dark')
  assert.deepEqual(effective.overridesByTheme.harbor, {
    accentKey: 'custom', accentCustom: '#ABCDEF', termFontSize: 17,
  })
  assert.equal(applyManagedAppearance(personal, null).overridesByTheme.harbor?.density, 'compact')
})
