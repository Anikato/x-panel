import assert from 'node:assert/strict'
import test from 'node:test'
import { colorAlpha } from './charts.ts'

test('colorAlpha converts hex and rgb into rgba', () => {
  assert.equal(colorAlpha('#7AA2FF', 0.2), 'rgba(122, 162, 255, 0.2)')
  assert.equal(colorAlpha('#abc', 0.5), 'rgba(170, 187, 204, 0.5)')
  assert.equal(colorAlpha('rgb(34, 197, 94)', 0.15), 'rgba(34, 197, 94, 0.15)')
  assert.equal(colorAlpha('rgba(239, 68, 68, 1)', 0.4), 'rgba(239, 68, 68, 0.4)')
})
