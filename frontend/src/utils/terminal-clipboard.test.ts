import assert from 'node:assert/strict'
import test from 'node:test'
import { decodeTerminalOsc52, terminalXtversionReply } from './terminal-clipboard.ts'

test('xtversion reply names xterm.js', () => {
  assert.equal(terminalXtversionReply(), '\x1bP>|xterm.js(5.5.0)\x1b\\')
})

test('osc 52 clipboard payload decodes and queries are ignored', () => {
  const hello = new TextEncoder().encode('你好')
  const b64 = btoa(String.fromCharCode(...hello))
  assert.equal(decodeTerminalOsc52('c;' + b64), '你好')
  assert.equal(decodeTerminalOsc52('c;'), '')
  assert.equal(decodeTerminalOsc52('c;?'), null)
  assert.equal(decodeTerminalOsc52('s;aGVsbG8='), null)
  assert.equal(decodeTerminalOsc52('c;@@@'), null)
})
