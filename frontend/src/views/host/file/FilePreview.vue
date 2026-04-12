<template>
  <el-dialog
    v-model="open"
    :title="fileName"
    :width="dialogWidth"
    destroy-on-close
    append-to-body
    class="file-preview-dialog"
    :close-on-click-modal="true"
  >
    <div class="preview-body" v-loading="loading">
      <!-- 图片 -->
      <template v-if="fileType === 'image'">
        <el-image :src="previewUrl" fit="contain" class="preview-image" :preview-src-list="[previewUrl]" />
      </template>

      <!-- 视频 -->
      <template v-else-if="fileType === 'video'">
        <video :src="previewUrl" controls autoplay class="preview-video">
          {{ t('file.previewNotSupported') }}
        </video>
      </template>

      <!-- 音频 -->
      <template v-else-if="fileType === 'audio'">
        <div class="audio-wrapper">
          <div class="audio-icon">
            <el-icon :size="64"><Headset /></el-icon>
          </div>
          <audio :src="previewUrl" controls autoplay class="preview-audio" />
        </div>
      </template>

      <!-- PDF -->
      <template v-else-if="fileType === 'pdf'">
        <iframe :src="previewUrl" class="preview-pdf" />
      </template>

      <!-- Excel / CSV -->
      <template v-else-if="fileType === 'excel'">
        <div class="excel-wrapper">
          <el-tabs v-if="sheetNames.length > 1" v-model="activeSheet" type="card" class="sheet-tabs" @tab-change="onSheetChange">
            <el-tab-pane v-for="name in sheetNames" :key="name" :label="name" :name="name" />
          </el-tabs>
          <div class="excel-table-wrap">
            <el-table :data="sheetData" size="small" stripe border max-height="520" :show-header="true">
              <el-table-column
                v-for="(col, idx) in sheetColumns"
                :key="idx"
                :prop="col"
                :label="col"
                min-width="120"
                show-overflow-tooltip
              />
            </el-table>
          </div>
        </div>
      </template>

      <!-- 不支持 -->
      <template v-else>
        <el-empty :description="t('file.previewNotSupported')" />
      </template>
    </div>

    <template #footer>
      <el-button @click="open = false">{{ t('commons.close') }}</el-button>
      <el-button type="primary" @click="handleDownload">
        <el-icon><Download /></el-icon>{{ t('commons.download') }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { getDownloadUrl } from '@/api/modules/file'
import { Headset, Download } from '@element-plus/icons-vue'
import { read, utils } from 'xlsx'

const { t } = useI18n()

const open = ref(false)
const loading = ref(false)
const fileName = ref('')
const filePath = ref('')

const EXT_MAP: Record<string, string> = {
  jpg: 'image', jpeg: 'image', png: 'image', gif: 'image',
  webp: 'image', svg: 'image', bmp: 'image', ico: 'image',
  mp4: 'video', webm: 'video', ogg: 'video', mov: 'video', mkv: 'video',
  mp3: 'audio', wav: 'audio', flac: 'audio', aac: 'audio', m4a: 'audio',
  pdf: 'pdf',
  xlsx: 'excel', xls: 'excel', csv: 'excel',
}

const PREVIEWABLE = new Set(Object.keys(EXT_MAP))

const getExt = (name: string) => name.split('.').pop()?.toLowerCase() || ''

const fileType = computed(() => EXT_MAP[getExt(fileName.value)] || 'unknown')

const dialogWidth = computed(() => {
  const t = fileType.value
  if (t === 'image' || t === 'video' || t === 'pdf') return '80%'
  if (t === 'excel') return '90%'
  if (t === 'audio') return '500px'
  return '60%'
})

const previewUrl = computed(() => getDownloadUrl(filePath.value))

// Excel state
const sheetNames = ref<string[]>([])
const activeSheet = ref('')
const sheetData = ref<Record<string, string>[]>([])
const sheetColumns = ref<string[]>([])
let workbookSheets: Record<string, { columns: string[]; data: Record<string, string>[] }> = {}

const isPreviewable = (name: string) => PREVIEWABLE.has(getExt(name))

const acceptParams = async (row: { name: string; path?: string }) => {
  fileName.value = row.name
  filePath.value = row.path || ''
  sheetNames.value = []
  sheetData.value = []
  sheetColumns.value = []
  workbookSheets = {}
  open.value = true

  if (fileType.value === 'excel') {
    await loadExcel()
  }
}

const loadExcel = async () => {
  loading.value = true
  try {
    const url = getDownloadUrl(filePath.value)
    const resp = await fetch(url)
    const buf = await resp.arrayBuffer()
    const wb = read(buf, { type: 'array' })

    for (const name of wb.SheetNames) {
      const sheet = wb.Sheets[name]
      const json = utils.sheet_to_json<Record<string, string>>(sheet, { header: 'A', defval: '' })
      if (json.length === 0) {
        workbookSheets[name] = { columns: [], data: [] }
        continue
      }
      const firstRow = json[0]
      const cols = Object.keys(firstRow)
      const headerRow = json[0]
      const dataRows = json.slice(1)
      const namedCols = cols.map(c => String(headerRow[c] || c))

      const mapped = dataRows.map(row => {
        const obj: Record<string, string> = {}
        cols.forEach((c, i) => { obj[namedCols[i]] = String(row[c] ?? '') })
        return obj
      })
      workbookSheets[name] = { columns: namedCols, data: mapped }
    }

    sheetNames.value = wb.SheetNames
    activeSheet.value = wb.SheetNames[0] || ''
    applySheet(activeSheet.value)
  } catch {
    sheetData.value = []
    sheetColumns.value = []
  } finally {
    loading.value = false
  }
}

const applySheet = (name: string) => {
  const s = workbookSheets[name]
  if (s) {
    sheetColumns.value = s.columns
    sheetData.value = s.data
  }
}

const onSheetChange = (name: string | number) => {
  applySheet(String(name))
}

const handleDownload = () => {
  window.open(previewUrl.value, '_blank')
}

defineExpose({ acceptParams, isPreviewable })
</script>

<style lang="scss" scoped>
.preview-body {
  min-height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.preview-image {
  max-width: 100%;
  max-height: 70vh;
  border-radius: 4px;
}

.preview-video {
  max-width: 100%;
  max-height: 70vh;
  border-radius: 4px;
  background: #000;
}

.audio-wrapper {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 24px;
  padding: 32px 0;
  width: 100%;
}

.audio-icon {
  color: var(--xp-text-muted);
  opacity: 0.5;
}

.preview-audio {
  width: 100%;
}

.preview-pdf {
  width: 100%;
  height: 75vh;
  border: none;
  border-radius: 4px;
}

.excel-wrapper {
  width: 100%;
}

.sheet-tabs {
  margin-bottom: 8px;
}

.excel-table-wrap {
  width: 100%;
  overflow: auto;
}
</style>

<style lang="scss">
.file-preview-dialog {
  .el-dialog__body {
    padding: 12px 20px;
  }
}
</style>
