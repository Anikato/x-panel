import { defineStore } from 'pinia'
import { getToken } from '@/utils/auth'
import { ref, computed } from 'vue'
import { buildUploadRequestHeaders } from './upload-request'

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
  _lastBytes: number      // 内部：上次采样字节数
  _lastTime: number       // 内部：上次采样时间戳
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
      _lastBytes: 0,
      _lastTime: Date.now(),
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
          (loaded: number, total: number) => {
            item.bytesDone = loaded
            item.bytesTotal = total || fileSize
            item.progress = total ? Math.round(loaded / total * 100) : 0

            // 速度：500ms 采样一次
            const now = Date.now()
            const elapsed = (now - item._lastTime) / 1000
            if (elapsed >= 0.5) {
              item.speed = Math.round((loaded - item._lastBytes) / elapsed)
              item._lastBytes = loaded
              item._lastTime = now
            }
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

// 内部：封装 XHR，传递 loaded/total（不依赖 axios 的百分比转换）
function uploadFileWithProgress(
  path: string,
  relativePath: string,
  file: File,
  overwrite: boolean,
  batch: boolean,
  nodeID: number,
  onProgress: (loaded: number, total: number) => void
): Promise<void> {
  return new Promise((resolve, reject) => {
    const token = getToken()
    const formData = new FormData()
    formData.append('path', path)
    formData.append('relativePath', relativePath)
    formData.append('overwrite', String(overwrite))
    formData.append('batch', String(batch))
    formData.append('file', file)

    const xhr = new XMLHttpRequest()
    xhr.open('POST', '/api/v1/files/upload')
    for (const [name, value] of Object.entries(buildUploadRequestHeaders(token, nodeID))) {
      xhr.setRequestHeader(name, value)
    }

    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) onProgress(e.loaded, e.total)
    }

    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve()
      } else {
        let message = `HTTP ${xhr.status}`
        try {
          message = JSON.parse(xhr.responseText)?.message || message
        } catch { /* ignore invalid error response */ }
        reject(new Error(message))
      }
    }

    xhr.onerror = () => reject(new Error('Network error'))
    xhr.send(formData)
  })
}
