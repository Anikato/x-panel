import assert from 'node:assert/strict'
import test from 'node:test'
import { buildTerminalWsUrl } from './terminal-ws.ts'

test('builds a local terminal websocket URL with cwd', () => {
  const url = buildTerminalWsUrl({
    protocol: 'https:',
    host: 'panel.example',
    ticket: 'ticket-value',
    cwd: '/etc/apparmor.d',
  })

  assert.equal(url, 'wss://panel.example/api/v1/terminal?ticket=ticket-value&cwd=%2Fetc%2Fapparmor.d')
})

test('does not attach cwd to remote host terminal URLs', () => {
  const url = buildTerminalWsUrl({
    protocol: 'http:',
    host: 'panel.example',
    ticket: 'ticket',
    hostId: 12,
    cwd: '/etc/app',
  })

  assert.equal(url, 'ws://panel.example/api/v1/terminal?ticket=ticket&id=12')
})
