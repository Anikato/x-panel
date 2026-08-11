import assert from 'node:assert/strict'
import test from 'node:test'

import { buildUploadRequestHeaders } from './upload-request.ts'

test('buildUploadRequestHeaders preserves the selected node context', () => {
  assert.deepEqual(buildUploadRequestHeaders('token', 42), {
    Authorization: 'token',
    'X-Node-ID': '42',
  })
})

test('buildUploadRequestHeaders omits the node header for the local panel', () => {
  assert.deepEqual(buildUploadRequestHeaders('token', 0), {
    Authorization: 'token',
  })
})
