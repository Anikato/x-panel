import assert from 'node:assert/strict'
import test from 'node:test'
import {
  UPLOAD_CHUNK_SIZE,
  UPLOAD_CHUNK_THRESHOLD,
  calculateChunkUploadProgress,
  getUploadChunkBounds,
  getUploadChunkCount,
  sha256Hex,
  shouldUseChunkedUpload,
} from './upload-chunks.ts'

const cryptoSupport = { subtle: globalThis.crypto.subtle }

test('chunked upload starts above 10 MB only when Web Crypto is available', () => {
  assert.equal(shouldUseChunkedUpload(UPLOAD_CHUNK_THRESHOLD, cryptoSupport), false)
  assert.equal(shouldUseChunkedUpload(UPLOAD_CHUNK_THRESHOLD + 1, cryptoSupport), true)
  assert.equal(shouldUseChunkedUpload(UPLOAD_CHUNK_THRESHOLD + 1, undefined), false)
})

test('chunk bounds use sequential 5 MB slices', () => {
  const total = UPLOAD_CHUNK_THRESHOLD + 1
  assert.equal(getUploadChunkCount(total), 3)
  assert.deepEqual(getUploadChunkBounds(total, 0), { start: 0, end: UPLOAD_CHUNK_SIZE })
  assert.deepEqual(getUploadChunkBounds(total, 2), { start: UPLOAD_CHUNK_THRESHOLD, end: total })
})

test('sha256Hex returns a lowercase browser-compatible digest', async () => {
  const digest = await sha256Hex(new Blob(['abc']), globalThis.crypto)
  assert.equal(digest, 'ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad')
})

test('aggregate progress includes the current chunk but stays below completion', () => {
  const total = UPLOAD_CHUNK_SIZE * 2
  assert.deepEqual(calculateChunkUploadProgress(UPLOAD_CHUNK_SIZE, UPLOAD_CHUNK_SIZE / 2, total), {
    loaded: UPLOAD_CHUNK_SIZE + UPLOAD_CHUNK_SIZE / 2,
    percentage: 75,
  })
  assert.deepEqual(calculateChunkUploadProgress(total, 0, total), {
    loaded: total,
    percentage: 99,
  })
  assert.deepEqual(calculateChunkUploadProgress(total, 0, total, true), {
    loaded: total,
    percentage: 100,
  })
})
