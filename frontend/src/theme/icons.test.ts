import assert from 'node:assert/strict'
import test from 'node:test'
import { mapNavIcon } from './icons.ts'
import { RADIUS_TOKENS } from './shared-tokens.ts'

test('security nav uses the filled shield in both icon sets', () => {
  assert.equal(mapNavIcon('Shield', 'outline'), 'ShieldIcon')
  assert.equal(mapNavIcon('Shield', 'solid'), 'ShieldIcon')
})

test('radius presets stay in the 2–18px chrome range and never become capsules', () => {
  for (const preset of Object.values(RADIUS_TOKENS)) {
    for (const value of Object.values(preset)) {
      const px = Number.parseInt(value, 10)
      assert.ok(px >= 2 && px <= 18, value)
      assert.notEqual(value, '999px')
    }
  }
})
