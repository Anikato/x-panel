import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { uploadFile } from '@/api/modules/file'

export interface UploadItem {
  id: number
  name: string
  progress: number
  error: boolean
  targetPath: string
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
    }))
    queue.value.push(...items)

    for (let i = 0; i < files.length; i++) {
      const item = queue.value[startIdx + i]
      try {
        await uploadFile(targetPath, files[i], (pct: number) => {
          item.progress = pct
        })
        item.progress = 100
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
