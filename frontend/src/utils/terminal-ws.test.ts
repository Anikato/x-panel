import assert from 'node:assert/strict'
import test from 'node:test'
import { buildTerminalWsUrl } from './terminal-ws.ts'

test('background session url carries the session id and not a new cwd', () => {
  const url = new URL(buildTerminalWsUrl({
    protocol: 'https:',
    host: 'panel.example:8080',
    ticket: 'once',
    cwd: '/tmp/work',
    sessionId: 'abc',
  }))
  assert.equal(url.protocol, 'wss:')
  assert.equal(url.pathname, '/api/v1/terminal')
  assert.equal(url.searchParams.get('ticket'), 'once')
  assert.equal(url.searchParams.get('session'), 'abc')
  assert.equal(url.searchParams.get('cwd'), null)
})
