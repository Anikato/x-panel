import { defineStore } from 'pinia'
import { getToken } from '@/utils/auth'
import { ref, computed } from 'vue'
import { buildUploadRequestHeaders } from './upload-request'
import {
  calculateChunkUploadProgress,
  createUploadID,
  getUploadChunkBounds,
  getUploadChunkCount,
  sha256Hex,
  shouldUseChunkedUpload,
} from './upload-chunks'

export interface UploadItem {
  id: number
  name: string
  progress: number
  error: boolean
  errorMessage: string
  targetPath: string
  bytesDone: number
  bytesTotal: number
  speed: number           // bytes/s
}

export interface UploadRequest {
  file: File
  relativePath: string
}

export interface UploadResult {
  success: number
  failed: number
}

export const useUploadStore = defineStore('upload', () => {
  const queue = ref<UploadItem[]>([])
  let idSeq = 0

  const doneCount = computed(() => queue.value.filter(i => i.progress >= 100 || i.error).length)
  const allDone = computed(() => queue.value.length > 0 && doneCount.value === queue.value.length)
  const hasActive = computed(() => queue.value.length > 0 && !allDone.value)

  async function addFiles(
    targetPath: string,
    requests: UploadRequest[],
    overwrite: boolean,
    nodeID = 0,
    batch = requests.length > 1 || requests.some(({ relativePath }) => relativePath.includes('/')),
  ): Promise<UploadResult> {
    const items: UploadItem[] = requests.map(({ file, relativePath }) => ({
      id: ++idSeq,
      name: relativePath,
      progress: 0,
      error: false,
      errorMessage: '',
      targetPath,
      bytesDone: 0,
      bytesTotal: file.size,
      speed: 0,
    }))
    queue.value.push(...items)
    let success = 0
    let failed = 0

    for (let i = 0; i < requests.length; i++) {
      const item = items[i]
      const { file, relativePath } = requests[i]
      const fileSize = file.size

      try {
        await uploadFileWithProgress(
          targetPath,
          relativePath,
          file,
          overwrite,
          batch,
          nodeID,
          ({ loaded, total, percentage, speed }) => {
            item.bytesDone = loaded
            item.bytesTotal = total || fileSize
            item.progress = percentage
            if (speed !== undefined) item.speed = speed
          }
        )
        item.progress = 100
        item.speed = 0
        success++
      } catch (error) {
        item.error = true
        item.errorMessage = error instanceof Error ? error.message : '上传失败'
        failed++
      }
    }
    return { success, failed }
  }

  function clear() {
    queue.value = []
  }

  return { queue, doneCount, allDone, hasActive, addFiles, clear }
})

interface UploadProgress {
  loaded: number
  total: number
  percentage: number
  speed?: number
}

type UploadProgressHandler = (progress: UploadProgress) => void

function uploadFileWithProgress(
  path: string,
  relativePath: string,
  file: File,
  overwrite: boolean,
  batch: boolean,
  nodeID: number,
  onProgress: UploadProgressHandler,
): Promise<void> {
  const cryptoProvider = globalThis.crypto
  if (shouldUseChunkedUpload(file.size, cryptoProvider)) {
    return uploadFileInChunks(path, relativePath, file, overwrite, batch, nodeID, cryptoProvider, onProgress)
  }
  return uploadSingleFile(path, relativePath, file, overwrite, batch, nodeID, onProgress)
}

function uploadSingleFile(
  path: string,
  relativePath: string,
  file: File,
  overwrite: boolean,
  batch: boolean,
  nodeID: number,
  onProgress: UploadProgressHandler,
): Promise<void> {
  return new Promise((resolve, reject) => {
    const formData = new FormData()
    formData.append('path', path)
    formData.append('relativePath', relativePath)
    formData.append('overwrite', String(overwrite))
    formData.append('batch', String(batch))
    formData.append('file', file)

    const xhr = new XMLHttpRequest()
    xhr.open('POST', '/api/v1/files/upload')
    for (const [name, value] of Object.entries(buildUploadRequestHeaders(getToken(), nodeID))) {
      xhr.setRequestHeader(name, value)
    }
    const rate = createRateSampler()
    xhr.upload.onprogress = (event) => {
      if (!event.lengthComputable || event.total <= 0) return
      const loaded = Math.min(file.size, Math.round(event.loaded / event.total * file.size))
      const percentage = Math.min(99, file.size ? Math.round(loaded / file.size * 100) : 0)
      onProgress({ loaded, total: file.size, percentage, speed: rate.sample(loaded) })
    }
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        onProgress({ loaded: file.size, total: file.size, percentage: 100, speed: rate.average(file.size) })
        resolve()
      } else {
        reject(uploadResponseError(xhr))
      }
    }
    xhr.onerror = () => reject(new Error('网络错误'))
    xhr.onabort = () => reject(new Error('上传已取消'))
    xhr.send(formData)
  })
}

async function uploadFileInChunks(
  path: string,
  relativePath: string,
  file: File,
  overwrite: boolean,
  batch: boolean,
  nodeID: number,
  cryptoProvider: Crypto,
  onProgress: UploadProgressHandler,
): Promise<void> {
  const uploadID = createUploadID(cryptoProvider)
  const chunkCount = getUploadChunkCount(file.size)
  const headers = buildUploadRequestHeaders(getToken(), nodeID)
  let confirmedBytes = 0
  let lastSpeed = 0

  try {
    for (let chunkIndex = 0; chunkIndex < chunkCount; chunkIndex++) {
      const { start, end } = getUploadChunkBounds(file.size, chunkIndex)
      const chunk = file.slice(start, end)
      const checksum = await sha256Hex(chunk, cryptoProvider)
      const rate = createRateSampler()
      const formData = new FormData()
      formData.append('path', path)
      formData.append('relativePath', relativePath)
      formData.append('uploadID', uploadID)
      formData.append('chunkIndex', String(chunkIndex))
      formData.append('chunkCount', String(chunkCount))
      formData.append('totalSize', String(file.size))
      formData.append('checksum', checksum)
      formData.append('file', chunk, file.name)

      await sendXHR('/api/v1/files/upload/chunk', formData, headers, (loaded, total) => {
        const chunkLoaded = total > 0 ? Math.min(chunk.size, Math.round(loaded / total * chunk.size)) : 0
        const progress = calculateChunkUploadProgress(confirmedBytes, chunkLoaded, file.size)
        const speed = rate.sample(chunkLoaded)
        if (speed !== undefined) lastSpeed = speed
        onProgress({ ...progress, total: file.size, speed })
      })

      confirmedBytes += chunk.size
      lastSpeed = rate.average(chunk.size)
      const progress = calculateChunkUploadProgress(confirmedBytes, 0, file.size)
      onProgress({ ...progress, total: file.size, speed: lastSpeed })
    }

    await sendJSON('/api/v1/files/upload/chunk/complete', {
      targetPath: path,
      relativePath,
      uploadID,
      totalSize: file.size,
      overwrite,
      batch,
    }, headers)
    const progress = calculateChunkUploadProgress(file.size, 0, file.size, true)
    onProgress({ ...progress, total: file.size, speed: lastSpeed })
  } catch (error) {
    await sendJSON('/api/v1/files/upload/chunk/abort', {
      targetPath: path,
      relativePath,
      uploadID,
      totalSize: file.size,
    }, headers).catch(() => undefined)
    throw error
  }
}

function sendXHR(
  url: string,
  body: XMLHttpRequestBodyInit,
  headers: Record<string, string>,
  onProgress?: (loaded: number, total: number) => void,
): Promise<void> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open('POST', url)
    for (const [name, value] of Object.entries(headers)) xhr.setRequestHeader(name, value)
    if (onProgress) {
      xhr.upload.onprogress = event => {
        if (event.lengthComputable) onProgress(event.loaded, event.total)
      }
    }
    xhr.onload = () => xhr.status >= 200 && xhr.status < 300 ? resolve() : reject(uploadResponseError(xhr))
    xhr.onerror = () => reject(new Error('网络错误'))
    xhr.onabort = () => reject(new Error('上传已取消'))
    xhr.send(body)
  })
}

function sendJSON(url: string, body: object, headers: Record<string, string>): Promise<void> {
  return sendXHR(url, JSON.stringify(body), { ...headers, 'Content-Type': 'application/json' })
}

function uploadResponseError(xhr: XMLHttpRequest): Error {
  let message = `HTTP ${xhr.status}`
  try {
    message = JSON.parse(xhr.responseText)?.message || message
  } catch { /* ignore invalid error response */ }
  return new Error(message)
}

function createRateSampler() {
  const startedAt = performance.now()
  let lastTime = startedAt
  let lastBytes = 0
  return {
    sample(bytes: number): number | undefined {
      const now = performance.now()
      const elapsed = now - lastTime
      if (elapsed < 500) return undefined
      const speed = Math.max(0, Math.round((bytes - lastBytes) * 1000 / elapsed))
      lastTime = now
      lastBytes = bytes
      return speed
    },
    average(bytes: number): number {
      const elapsed = Math.max(1, performance.now() - startedAt)
      return Math.max(0, Math.round(bytes * 1000 / elapsed))
    },
  }
}
