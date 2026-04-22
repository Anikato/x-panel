import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface UploadItem {
  id: number
  name: string
  progress: number
  error: boolean
  targetPath: string
  bytesDone: number
  bytesTotal: number
  speed: number           // bytes/s
  _lastBytes: number      // 内部：上次采样字节数
  _lastTime: number       // 内部：上次采样时间戳
}

export const useUploadStore = defineStore('upload', () => {
  const queue = ref<UploadItem[]>([])
  let idSeq = 0

  const doneCount = computed(() => queue.value.filter(i => i.progress >= 100 || i.error).length)
  const allDone = computed(() => queue.value.length > 0 && doneCount.value === queue.value.length)
  const hasActive = computed(() => queue.value.length > 0 && !allDone.value)

  async function addFiles(targetPath: string, files: File[]) {
    const startIdx = queue.value.length
    const items: UploadItem[] = files.map(f => ({
      id: ++idSeq,
      name: f.name,
      progress: 0,
      error: false,
      targetPath,
      bytesDone: 0,
      bytesTotal: f.size,
      speed: 0,
      _lastBytes: 0,
      _lastTime: Date.now(),
    }))
    queue.value.push(...items)

    for (let i = 0; i < files.length; i++) {
      const item = queue.value[startIdx + i]
      const fileSize = files[i].size

      try {
        await uploadFileWithProgress(
          targetPath,
          files[i],
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
      } catch {
        item.error = true
      }
    }
  }

  function clear() {
    queue.value = []
  }

  return { queue, doneCount, allDone, hasActive, addFiles, clear }
})

// 内部：封装 XHR，传递 loaded/total（不依赖 axios 的百分比转换）
function uploadFileWithProgress(
  path: string,
  file: File,
  onProgress: (loaded: number, total: number) => void
): Promise<void> {
  return new Promise((resolve, reject) => {
    const token = sessionStorage.getItem('token') || ''
    const formData = new FormData()
    formData.append('path', path)
    formData.append('file', file)

    const xhr = new XMLHttpRequest()
    xhr.open('POST', '/api/v1/files/upload')
    xhr.setRequestHeader('Authorization', token)

    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) onProgress(e.loaded, e.total)
    }

    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve()
      } else {
        reject(new Error(`HTTP ${xhr.status}`))
      }
    }

    xhr.onerror = () => reject(new Error('Network error'))
    xhr.send(formData)
  })
}
