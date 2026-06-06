import assert from 'node:assert/strict'
import test from 'node:test'
import { buildInitialCwdCommand, normalizeTerminalCwd } from './terminal-cwd.ts'

test('normalizes invalid terminal cwd values to root', () => {
  assert.equal(normalizeTerminalCwd(undefined), '/')
  assert.equal(normalizeTerminalCwd(null), '/')
  assert.equal(normalizeTerminalCwd(''), '/')
  assert.equal(normalizeTerminalCwd('undefined'), '/')
  assert.equal(normalizeTerminalCwd(' null '), '/')
})

test('keeps valid absolute terminal cwd values', () => {
  assert.equal(normalizeTerminalCwd('/data'), '/data')
  assert.equal(normalizeTerminalCwd(' /data/remote '), '/data/remote')
})

test('builds a safe cd command and never emits cd undefined', () => {
  assert.equal(buildInitialCwdCommand('undefined'), 'cd "/" && clear\n')
  assert.equal(buildInitialCwdCommand('/data'), 'cd "/data" && clear\n')
})
