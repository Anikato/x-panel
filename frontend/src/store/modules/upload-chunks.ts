export const UPLOAD_CHUNK_SIZE = 5 * 1024 * 1024
export const UPLOAD_CHUNK_THRESHOLD = 10 * 1024 * 1024

type DigestCrypto = Pick<Crypto, 'subtle'>
type RandomCrypto = Pick<Crypto, 'getRandomValues'>

export function shouldUseChunkedUpload(
  fileSize: number,
  cryptoProvider?: DigestCrypto,
): boolean {
  return fileSize > UPLOAD_CHUNK_THRESHOLD && !!cryptoProvider?.subtle
}

export function getUploadChunkCount(totalSize: number): number {
  if (!Number.isSafeInteger(totalSize) || totalSize <= 0) throw new Error('文件大小无效')
  return Math.ceil(totalSize / UPLOAD_CHUNK_SIZE)
}

export function getUploadChunkBounds(totalSize: number, chunkIndex: number): { start: number; end: number } {
  const count = getUploadChunkCount(totalSize)
  if (!Number.isInteger(chunkIndex) || chunkIndex < 0 || chunkIndex >= count) throw new Error('分片序号无效')
  const start = chunkIndex * UPLOAD_CHUNK_SIZE
  return { start, end: Math.min(start + UPLOAD_CHUNK_SIZE, totalSize) }
}

export async function sha256Hex(blob: Blob, cryptoProvider: DigestCrypto = globalThis.crypto): Promise<string> {
  const digest = await cryptoProvider.subtle.digest('SHA-256', await blob.arrayBuffer())
  return Array.from(new Uint8Array(digest), byte => byte.toString(16).padStart(2, '0')).join('')
}

export function createUploadID(cryptoProvider: RandomCrypto = globalThis.crypto): string {
  const bytes = new Uint8Array(16)
  cryptoProvider.getRandomValues(bytes)
  return Array.from(bytes, byte => byte.toString(16).padStart(2, '0')).join('')
}

export function calculateChunkUploadProgress(
  confirmedBytes: number,
  currentChunkBytes: number,
  totalSize: number,
  finalized = false,
): { loaded: number; percentage: number } {
  const loaded = Math.min(totalSize, Math.max(0, confirmedBytes + currentChunkBytes))
  if (totalSize <= 0) return { loaded: 0, percentage: finalized ? 100 : 0 }
  const percentage = finalized ? 100 : Math.min(99, Math.round(loaded / totalSize * 100))
  return { loaded, percentage }
}
