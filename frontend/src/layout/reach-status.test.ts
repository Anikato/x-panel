import assert from 'node:assert/strict'
import test from 'node:test'
import { reachStatus } from './reach-status.ts'

test('reachStatus maps fail streak to green yellow red', () => {
  assert.equal(reachStatus(0), 'online')
  assert.equal(reachStatus(1), 'warning')
  assert.equal(reachStatus(2), 'offline')
  assert.equal(reachStatus(5), 'offline')
})
