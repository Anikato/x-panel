import assert from 'node:assert/strict'
import test from 'node:test'
import { encodeTerminalDetach, readTerminalServerFrame } from './terminal-frame.ts'

test('detach frame is a binary message with prefix 2 and does not include the prefix in the json', () => {
  const frame = encodeTerminalDetach('编译')
  assert.equal(frame[0], 2)
  assert.deepEqual(JSON.parse(new TextDecoder().decode(frame.slice(1))), { op: 'detach', name: '编译' })
})

test('binary server frames stay terminal output', () => {
  const raw = new TextEncoder().encode('编译中\n')
  const frame = readTerminalServerFrame(raw.buffer.slice(raw.byteOffset, raw.byteOffset + raw.byteLength))
  assert.equal(frame.kind, 'output')
  if (frame.kind !== 'output' || !('bytes' in frame)) {
    assert.fail('expected bytes')
  }
  assert.equal(new TextDecoder().decode(frame.bytes), '编译中\n')
})

test('json text frames with op are control and other text stays output', () => {
  const control = readTerminalServerFrame('{"op":"detached","id":"abc","message":""}')
  assert.deepEqual(control, { kind: 'control', op: 'detached', id: 'abc', message: '' })

  const output = readTerminalServerFrame('\r\n连接错误')
  assert.deepEqual(output, { kind: 'output', text: '\r\n连接错误' })
})
