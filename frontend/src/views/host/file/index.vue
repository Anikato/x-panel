<template>
  <div
    class="file-manager"
    @contextmenu.prevent
    @dragover.prevent="handleDragover"
    @drop.prevent="handleDrop"
    @dragleave.prevent="isDragging = false"
  >
    <!-- 多标签 -->
    <el-tabs
      v-model="activeTabId"
      type="card"
      class="file-tabs"
      editable
      @tab-add="addTab"
      @tab-remove="removeTab"
      @tab-change="switchTab"
    >
      <el-tab-pane
        v-for="tab in tabs"
        :key="tab.id"
        :label="tab.name || t('file.root')"
        :name="tab.id"
        :closable="tabs.length > 1"
      />
    </el-tabs>

    <!-- 导航栏 -->
    <div class="file-nav">
      <el-tooltip :content="t('file.back')" placement="top">
        <el-button @click="goBack" :disabled="!canGoBack" :icon="Back" circle size="small" />
      </el-tooltip>
      <el-tooltip :content="t('file.forward')" placement="top">
        <el-button @click="goForward" :disabled="!canGoForward" :icon="Right" circle size="small" />
      </el-tooltip>
      <el-tooltip :content="t('file.goUp')" placement="top">
        <el-button @click="goUp" :disabled="currentTab?.path === '/'" :icon="Top" circle size="small" />
      </el-tooltip>
      <el-tooltip :content="t('file.refresh')" placement="top">
        <el-button @click="refreshFiles" :icon="Refresh" circle size="small" />
      </el-tooltip>
      <el-input
        v-model="pathInput"
        class="path-input"
        @keyup.enter="navigateTo(pathInput)"
        :placeholder="t('file.enterPath')"
        size="default"
      >
        <template #prefix>
          <el-icon><FolderOpened /></el-icon>
        </template>
      </el-input>
    </div>

    <!-- 工具栏（参考 1Panel 布局） -->
    <div class="file-toolbar">
      <div class="toolbar-left">
        <!-- 创建下拉 -->
        <el-dropdown @command="handleCreateCommand" trigger="click">
          <el-button type="primary" size="small">
            {{ t('commons.create') }}<el-icon class="el-icon--right"><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="file"><el-icon><Document /></el-icon>{{ t('file.newFile') }}</el-dropdown-item>
              <el-dropdown-item command="dir"><el-icon><Folder /></el-icon>{{ t('file.newDir') }}</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <!-- 上传 -->
        <el-upload
          :show-file-list="false"
          :before-upload="() => false"
          :auto-upload="false"
          @change="handleUploadChange"
          multiple
        >
          <el-button size="small"><el-icon><Upload /></el-icon>{{ t('file.upload') }}</el-button>
        </el-upload>
        <!-- 远程下载 -->
        <el-button size="small" @click="showRemoteDownload">
          <el-icon><Download /></el-icon>{{ t('file.remoteDownload') }}
        </el-button>

        <!-- 批量操作按钮组 -->
        <el-button-group class="batch-btn-group">
          <el-button size="small" :disabled="selectedRows.length === 0" @click="setClipboard('copy', selectedRows)">
            {{ t('file.copyTo') }}
          </el-button>
          <el-button size="small" :disabled="selectedRows.length === 0" @click="setClipboard('cut', selectedRows)">
            {{ t('file.moveTo') }}
          </el-button>
          <el-button size="small" :disabled="selectedRows.length === 0" @click="batchCompress">
            {{ t('file.compress') }}
          </el-button>
          <el-button size="small" :disabled="selectedRows.length === 0" @click="openBatchPermission">
            {{ t('file.changePermission') }}
          </el-button>
          <el-button size="small" type="danger" plain :disabled="selectedRows.length === 0" @click="handleBatchDelete">
            {{ t('file.delete') }}
          </el-button>
        </el-button-group>

        <!-- 终端 -->
        <el-button size="small" @click="openTerminal">
          <el-icon><Monitor /></el-icon>{{ t('file.openTerminal') }}
        </el-button>

        <!-- 剪贴板粘贴 -->
        <el-button-group v-if="clipboard.paths.length > 0" class="paste-btn-group">
          <el-button type="success" size="small" @click="doPaste">
            {{ t('file.pasteHere') }}({{ clipboard.paths.length }})
          </el-button>
          <el-button size="small" @click="clearClipboard" :icon="Close" />
        </el-button-group>
      </div>
      <div class="toolbar-right">
        <!-- 显示隐藏文件 -->
        <el-tooltip :content="showHidden ? t('file.hiddenFiles') : t('file.hiddenFiles')" placement="top">
          <el-button
            circle
            size="small"
            :type="showHidden ? 'primary' : ''"
            :icon="showHidden ? View : Hide"
            @click="showHidden = !showHidden; refreshFiles()"
          />
        </el-tooltip>
        <!-- 搜索（带子目录勾选） -->
        <el-input
          v-model="searchKeyword"
          :placeholder="t('file.searchPlaceholder')"
          clearable
          size="small"
          class="search-input"
          @clear="handleSearchClear"
          @keydown.enter="refreshFiles"
        >
          <template #prepend>
            <el-checkbox v-model="containSub" size="small">{{ t('file.containSub') }}</el-checkbox>
          </template>
          <template #append>
            <el-button :icon="Search" @click="refreshFiles" />
          </template>
        </el-input>
      </div>
    </div>

    <!-- 面包屑路径 -->
    <div class="file-breadcrumb">
      <span
        v-for="(seg, idx) in pathSegments"
        :key="idx"
        class="breadcrumb-item"
        @click="navigateTo(seg.path)"
      >
        <span class="breadcrumb-separator" v-if="idx > 0">/</span>
        <span class="breadcrumb-text">{{ seg.name }}</span>
      </span>
    </div>

    <!-- 拖拽上传覆盖层 -->
    <div v-if="isDragging" class="drop-overlay">
      <div class="drop-text">
        <el-icon :size="48"><Upload /></el-icon>
        <p>{{ t('file.dropHere') }}</p>
      </div>
    </div>

    <!-- 文件表格 -->
    <el-table
      ref="tableRef"
      :data="fileList"
      v-loading="loading"
      @row-dblclick="handleRowDblClick"
      @row-contextmenu="handleContextMenu"
      @selection-change="handleSelectionChange"
      row-key="path"
      highlight-current-row
      stripe
      :height="tableHeight"
    >
      <el-table-column type="selection" width="36" />
      <el-table-column :label="t('file.name')" min-width="300" sortable :sort-method="sortByName">
        <template #default="{ row }">
          <div class="file-name-cell">
            <el-icon class="file-icon" :style="{ color: getFileIconColor(row) }">
              <component :is="getFileIcon(row)" />
            </el-icon>
            <span class="file-name" :class="{ 'is-link': row.isSymlink }">{{ row.name }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column :label="t('file.size')" width="110" sortable prop="size">
        <template #default="{ row }">
          {{ row.isDir ? '-' : formatSize(row.size) }}
        </template>
      </el-table-column>
      <el-table-column :label="t('file.mode')" width="120">
        <template #default="{ row }">
          <el-link :underline="false" type="primary" @click.stop="openPermission(row)">
            {{ row.mode }}
          </el-link>
        </template>
      </el-table-column>
      <el-table-column :label="t('file.user')" width="100">
        <template #default="{ row }">
          <el-link :underline="false" type="primary" @click.stop="openChown(row)">
            {{ row.user }}
          </el-link>
        </template>
      </el-table-column>
      <el-table-column :label="t('file.modTime')" width="180" prop="modTime">
        <template #default="{ row }">
          {{ formatTime(row.modTime) }}
        </template>
      </el-table-column>
      <el-table-column :label="t('commons.operate')" width="280" fixed="right">
        <template #default="{ row }">
          <el-button v-if="canEdit(row)" link type="primary" size="small" @click.stop="openEditor(row)">
            {{ t('file.edit') }}
          </el-button>
          <el-button v-if="!row.isDir" link type="primary" size="small" @click.stop="handleDownload(row)">
            {{ t('commons.download') }}
          </el-button>
          <el-button link type="info" size="small" @click.stop="openDetail(row)">
            {{ t('file.detail') }}
          </el-button>
          <el-dropdown trigger="click" @command="(cmd: string) => handleMoreAction(cmd, row)">
            <el-button link type="primary" size="small" @click.stop>
              {{ t('file.more') }}<el-icon class="el-icon--right"><ArrowDown /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="rename">{{ t('file.rename') }}</el-dropdown-item>
                <el-dropdown-item command="copyPath">{{ t('file.copyPath') }}</el-dropdown-item>
                <el-dropdown-item command="copy">{{ t('file.copyTo') }}</el-dropdown-item>
                <el-dropdown-item command="cut">{{ t('file.moveTo') }}</el-dropdown-item>
                <el-dropdown-item command="permission">{{ t('file.changePermission') }}</el-dropdown-item>
                <el-dropdown-item command="chown">{{ t('file.changeOwner') }}</el-dropdown-item>
                <el-dropdown-item v-if="isCompressFile(row)" command="decompress">{{ t('file.decompress') }}</el-dropdown-item>
                <el-dropdown-item command="compress">{{ t('file.compress') }}</el-dropdown-item>
                <el-dropdown-item command="delete" divided>
                  <span style="color: var(--el-color-danger)">{{ t('file.delete') }}</span>
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </template>
      </el-table-column>
    </el-table>

    <div class="file-footer">
      <span>{{ fileList.length }} {{ t('file.items', { count: fileList.length }) }}</span>
    </div>

    <!-- 右键菜单 -->
    <Teleport to="body">
      <div
        v-if="contextMenu.visible"
        class="context-menu"
        :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
        @click="contextMenu.visible = false"
      >
        <div class="context-item" @click="handleRowDblClick(contextMenu.row)">
          <el-icon><FolderOpened /></el-icon>{{ t('file.open') }}
        </div>
        <div v-if="canEdit(contextMenu.row)" class="context-item" @click="openEditor(contextMenu.row)">
          <el-icon><EditPen /></el-icon>{{ t('file.edit') }}
        </div>
        <div v-if="!contextMenu.row?.isDir" class="context-item" @click="handleDownload(contextMenu.row)">
          <el-icon><Download /></el-icon>{{ t('commons.download') }}
        </div>
        <div class="context-item" @click="openDetail(contextMenu.row)">
          <el-icon><InfoFilled /></el-icon>{{ t('file.detail') }}
        </div>
        <div class="context-divider" />
        <div class="context-item" @click="copyPathToClipboard(contextMenu.row)">
          <el-icon><CopyDocument /></el-icon>{{ t('file.copyPath') }}
        </div>
        <div class="context-item" @click="setClipboard('copy', [contextMenu.row])">
          <el-icon><DocumentCopy /></el-icon>{{ t('file.copyTo') }}
        </div>
        <div class="context-item" @click="setClipboard('cut', [contextMenu.row])">
          <el-icon><Rank /></el-icon>{{ t('file.moveTo') }}
        </div>
        <div class="context-item" @click="handleRename(contextMenu.row)">
          <el-icon><EditPen /></el-icon>{{ t('file.rename') }}
        </div>
        <div class="context-divider" />
        <div class="context-item" @click="openPermission(contextMenu.row)">
          <el-icon><Lock /></el-icon>{{ t('file.changePermission') }}
        </div>
        <div class="context-item" @click="openChown(contextMenu.row)">
          <el-icon><User /></el-icon>{{ t('file.changeOwner') }}
        </div>
        <div class="context-item" @click="handleCompressSingle(contextMenu.row)">
          <el-icon><Box /></el-icon>{{ t('file.compress') }}
        </div>
        <div v-if="isCompressFile(contextMenu.row)" class="context-item" @click="handleDecompress(contextMenu.row)">
          <el-icon><Files /></el-icon>{{ t('file.decompress') }}
        </div>
        <div class="context-divider" />
        <div class="context-item danger" @click="handleDelete(contextMenu.row)">
          <el-icon><Delete /></el-icon>{{ t('file.delete') }}
        </div>
      </div>
    </Teleport>

    <!-- 新建弹窗 -->
    <el-dialog v-model="createVisible" :title="createIsDir ? t('file.newDir') : t('file.newFile')" width="420px">
      <el-input
        v-model="createName"
        :placeholder="createIsDir ? t('file.newDirName') : t('file.newFileName')"
        @keyup.enter="handleCreate"
        autofocus
      />
      <template #footer>
        <el-button @click="createVisible = false">{{ t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleCreate">{{ t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- 远程下载弹窗 -->
    <el-dialog v-model="remoteDownloadVisible" :title="t('file.remoteDownload')" width="500px" destroy-on-close>
      <el-form label-width="100px">
        <el-form-item :label="t('file.remoteUrl')">
          <el-input v-model="remoteUrl" :placeholder="t('file.remoteUrlPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('file.targetPath')">
          <el-input :model-value="currentTab?.path || '/'" disabled />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="remoteDownloadVisible = false">{{ t('commons.cancel') }}</el-button>
        <el-button type="primary" :loading="remoteDownloading" @click="doRemoteDownload">{{ t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- 子组件 -->
    <CodeEditor ref="codeEditorRef" @saved="refreshFiles" />
    <TerminalDialog ref="terminalRef" />
    <CompressDialog ref="compressRef" @done="refreshFiles" />
    <PermissionDialog ref="permissionRef" @done="refreshFiles" />
    <ChownDialog ref="chownRef" @done="refreshFiles" />
    <DetailDrawer ref="detailRef" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  listFiles, createFile, deleteFile, batchDeleteFile, renameFile,
  moveFile, getDownloadUrl, uploadFile,
} from '@/api/modules/file'
import { useI18n } from 'vue-i18n'
import {
  Back, Right, Top, Refresh, FolderOpened, FolderAdd, DocumentAdd, Upload, Monitor, Search,
  Delete, Download, EditPen, CopyDocument, DocumentCopy, InfoFilled, User,
  Lock, Box, Files, ArrowDown, Document, Folder, Picture, VideoPlay,
  Headset, SetUp, Tickets, Memo, Rank, Close, View, Hide,
} from '@element-plus/icons-vue'
import CodeEditor from './code-editor.vue'
import TerminalDialog from './terminal-dialog.vue'
import CompressDialog from './compress-dialog.vue'
import PermissionDialog from './permission-dialog.vue'
import ChownDialog from './chown-dialog.vue'
import DetailDrawer from './detail-drawer.vue'

const { t } = useI18n()
const loading = ref(false)
const showHidden = ref(false)
const fileList = ref<any[]>([])
const selectedRows = ref<any[]>([])
const tableHeight = ref(500)
const pathInput = ref('/')
const searchKeyword = ref('')
const containSub = ref(false)
let searchTimer: ReturnType<typeof setTimeout> | null = null

// 远程下载
const remoteDownloadVisible = ref(false)
const remoteUrl = ref('')
const remoteDownloading = ref(false)

// ===================== 多标签系统 =====================

interface TabState {
  id: string
  name: string
  path: string
  historyBack: string[]
  historyForward: string[]
}

const tabs = ref<TabState[]>([
  { id: 'tab-1', name: '/', path: '/', historyBack: [], historyForward: [] },
])
const activeTabId = ref('tab-1')
let tabCounter = 1

const currentTab = computed(() => tabs.value.find(t => t.id === activeTabId.value))

function addTab() {
  tabCounter++
  const id = `tab-${tabCounter}`
  const path = currentTab.value?.path || '/'
  tabs.value.push({ id, name: getTabName(path), path, historyBack: [], historyForward: [] })
  activeTabId.value = id
  refreshFiles()
}

function removeTab(targetId: string | number) {
  const id = String(targetId)
  if (tabs.value.length <= 1) return
  const idx = tabs.value.findIndex(t => t.id === id)
  tabs.value.splice(idx, 1)
  if (activeTabId.value === id) {
    activeTabId.value = tabs.value[Math.min(idx, tabs.value.length - 1)].id
    syncFromTab()
    refreshFiles()
  }
}

function switchTab(id: string | number) {
  activeTabId.value = String(id)
  syncFromTab()
  refreshFiles()
}

function syncFromTab() {
  const tab = currentTab.value
  if (tab) {
    pathInput.value = tab.path
    searchKeyword.value = ''
  }
}

function getTabName(path: string): string {
  if (path === '/') return '/'
  return path.split('/').filter(Boolean).pop() || '/'
}

// ===================== 导航历史栈 =====================

const canGoBack = computed(() => (currentTab.value?.historyBack.length ?? 0) > 0)
const canGoForward = computed(() => (currentTab.value?.historyForward.length ?? 0) > 0)

function navigateTo(path: string, addHistory = true) {
  const tab = currentTab.value
  if (!tab) return
  const newPath = path || '/'
  if (addHistory && tab.path !== newPath) {
    tab.historyBack.push(tab.path)
    tab.historyForward = [] // 新导航清空前进历史
  }
  tab.path = newPath
  tab.name = getTabName(newPath)
  pathInput.value = newPath
  searchKeyword.value = ''
  refreshFiles()
}

function goBack() {
  const tab = currentTab.value
  if (!tab || tab.historyBack.length === 0) return
  tab.historyForward.push(tab.path)
  const prev = tab.historyBack.pop()!
  tab.path = prev
  tab.name = getTabName(prev)
  pathInput.value = prev
  searchKeyword.value = ''
  refreshFiles()
}

function goForward() {
  const tab = currentTab.value
  if (!tab || tab.historyForward.length === 0) return
  tab.historyBack.push(tab.path)
  const next = tab.historyForward.pop()!
  tab.path = next
  tab.name = getTabName(next)
  pathInput.value = next
  searchKeyword.value = ''
  refreshFiles()
}

function goUp() {
  const tab = currentTab.value
  if (!tab || tab.path === '/') return
  const parts = tab.path.split('/').filter(Boolean)
  parts.pop()
  navigateTo('/' + parts.join('/'))
}

// ===================== 面包屑 =====================

const pathSegments = computed(() => {
  const tab = currentTab.value
  if (!tab) return [{ name: '/', path: '/' }]
  const parts = tab.path.split('/').filter(Boolean)
  const segs = [{ name: '/', path: '/' }]
  let path = ''
  for (const part of parts) {
    path += '/' + part
    segs.push({ name: part, path })
  }
  return segs
})

// ===================== 搜索防抖 =====================

function handleSearchInput() {
  // 子目录搜索不自动触发，需要手动点击搜索
  if (containSub.value) return
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => refreshFiles(), 300)
}

function handleSearchClear() {
  containSub.value = false
  refreshFiles()
}

// ===================== 核心操作 =====================

const refreshFiles = async () => {
  const tab = currentTab.value
  if (!tab) return
  loading.value = true
  try {
    const res: any = await listFiles({
      path: tab.path,
      showHidden: showHidden.value,
      search: searchKeyword.value || undefined,
      containSub: containSub.value || undefined,
    })
    fileList.value = res.data?.items || []
    pathInput.value = tab.path
  } catch {
    fileList.value = []
  } finally {
    loading.value = false
  }
}

const handleRowDblClick = (row: any) => {
  if (!row) return
  if (row.isDir) {
    navigateTo(row.path)
  } else if (canEdit(row)) {
    openEditor(row)
  }
}

const handleSelectionChange = (rows: any[]) => {
  selectedRows.value = rows
}

const sortByName = (a: any, b: any) => {
  if (a.isDir !== b.isDir) return a.isDir ? -1 : 1
  return a.name.localeCompare(b.name)
}

// ===================== 文件图标 =====================

const imageExts = new Set(['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp', 'ico', 'bmp'])
const videoExts = new Set(['mp4', 'avi', 'mkv', 'mov', 'wmv', 'flv', 'webm'])
const audioExts = new Set(['mp3', 'wav', 'flac', 'aac', 'ogg', 'wma'])
const archiveExts = new Set(['zip', 'tar', 'gz', 'bz2', 'xz', '7z', 'rar', 'tgz'])
const codeExts = new Set(['js', 'ts', 'py', 'go', 'rs', 'java', 'c', 'cpp', 'h', 'vue', 'jsx', 'tsx', 'rb', 'php', 'sh', 'bash'])
const configExts = new Set(['json', 'yaml', 'yml', 'toml', 'xml', 'ini', 'conf', 'cfg', 'env'])

function getExt(name: string): string {
  return (name.split('.').pop() || '').toLowerCase()
}

function getFileIcon(row: any) {
  if (row.isDir) return Folder
  const ext = getExt(row.name)
  if (imageExts.has(ext)) return Picture
  if (videoExts.has(ext)) return VideoPlay
  if (audioExts.has(ext)) return Headset
  if (archiveExts.has(ext)) return Box
  if (codeExts.has(ext)) return Memo
  if (configExts.has(ext)) return SetUp
  if (ext === 'md') return Tickets
  return Document
}

function getFileIconColor(row: any) {
  if (row.isDir) return 'var(--xp-accent)'
  const ext = getExt(row.name)
  if (imageExts.has(ext)) return '#f472b6'
  if (videoExts.has(ext)) return '#fb923c'
  if (audioExts.has(ext)) return '#a78bfa'
  if (archiveExts.has(ext)) return '#fbbf24'
  if (codeExts.has(ext)) return '#4ade80'
  if (configExts.has(ext)) return '#60a5fa'
  return 'var(--xp-text-muted)'
}

// ===================== Refs =====================

const codeEditorRef = ref<InstanceType<typeof CodeEditor>>()
const terminalRef = ref<InstanceType<typeof TerminalDialog>>()
const compressRef = ref<InstanceType<typeof CompressDialog>>()
const permissionRef = ref<InstanceType<typeof PermissionDialog>>()
const chownRef = ref<InstanceType<typeof ChownDialog>>()
const detailRef = ref<InstanceType<typeof DetailDrawer>>()

// ===================== 编辑 =====================

function canEdit(row: any): boolean {
  if (!row || row.isDir) return false
  if (row.size > 10 * 1024 * 1024) return false
  return true
}

const openEditor = (row: any) => {
  codeEditorRef.value?.open(row.path)
}

// ===================== 终端 =====================

const openTerminal = () => {
  terminalRef.value?.open(currentTab.value?.path || '/')
}

// ===================== 下载 =====================

const handleDownload = (row: any) => {
  if (!row || row.isDir) return
  const url = getDownloadUrl(row.path)
  window.open(url, '_blank')
}

// ===================== 详情 =====================

const openDetail = (row: any) => {
  if (!row) return
  detailRef.value?.open(row)
}

// ===================== 重命名 =====================

const handleRename = async (row: any) => {
  if (!row) return
  try {
    const result: any = await ElMessageBox.prompt(t('file.renameTo'), t('file.rename'), {
      inputValue: row.name,
    })
    const value = result.value
    if (!value || value === row.name) return
    const dir = currentTab.value?.path || '/'
    await renameFile({
      oldName: row.path,
      newName: dir + (dir.endsWith('/') ? '' : '/') + value,
    })
    ElMessage.success(t('commons.success'))
    refreshFiles()
  } catch { /* cancelled */ }
}

// ===================== 删除 =====================

const handleDelete = async (row: any) => {
  if (!row) return
  try {
    await ElMessageBox.confirm(t('file.deleteConfirm'), t('commons.tip'), { type: 'warning' })
    await deleteFile({ path: row.path })
    ElMessage.success(t('commons.success'))
    refreshFiles()
  } catch { /* cancelled */ }
}

const handleBatchDelete = async () => {
  try {
    await ElMessageBox.confirm(t('file.deleteConfirm'), t('commons.tip'), { type: 'warning' })
    await batchDeleteFile({ paths: selectedRows.value.map((r: any) => r.path) })
    ElMessage.success(t('commons.success'))
    refreshFiles()
  } catch { /* cancelled */ }
}

// ===================== 新建 =====================

const createVisible = ref(false)
const createIsDir = ref(false)
const createName = ref('')

const showCreate = (type: 'file' | 'dir') => {
  createIsDir.value = type === 'dir'
  createName.value = ''
  createVisible.value = true
}

function handleCreateCommand(cmd: string) {
  showCreate(cmd as 'file' | 'dir')
}

const handleCreate = async () => {
  if (!createName.value) return
  const dir = currentTab.value?.path || '/'
  const fullPath = dir + (dir.endsWith('/') ? '' : '/') + createName.value
  try {
    await createFile({ path: fullPath, isDir: createIsDir.value })
    ElMessage.success(t('commons.success'))
    createVisible.value = false
    refreshFiles()
  } catch { /* */ }
}

// ===================== 上传 =====================

const handleUploadChange = async (file: any) => {
  if (!file?.raw) return
  loading.value = true
  try {
    await uploadFile(currentTab.value?.path || '/', file.raw)
    ElMessage.success(t('commons.success'))
    refreshFiles()
  } catch { /* */ } finally {
    loading.value = false
  }
}

// ===================== 远程下载 =====================

function showRemoteDownload() {
  remoteUrl.value = ''
  remoteDownloadVisible.value = true
}

async function doRemoteDownload() {
  if (!remoteUrl.value) return
  remoteDownloading.value = true
  try {
    const dir = currentTab.value?.path || '/'
    // 使用 wget/curl 下载到当前目录
    const resp: any = await import('@/api/http').then(m =>
      m.default.post('/files/wget', { url: remoteUrl.value, path: dir })
    )
    ElMessage.success(t('commons.success'))
    remoteDownloadVisible.value = false
    refreshFiles()
  } catch { /* */ } finally {
    remoteDownloading.value = false
  }
}

// ===================== 批量权限 =====================

function openBatchPermission() {
  if (selectedRows.value.length === 0) return
  // 打开第一个文件的权限弹窗（后续可扩展为批量）
  const row = selectedRows.value[0]
  permissionRef.value?.open(row.path, row.mode)
}

// ===================== 拖拽上传 =====================

const isDragging = ref(false)

function handleDragover(e: DragEvent) {
  if (e.dataTransfer?.types.includes('Files')) {
    isDragging.value = true
  }
}

async function handleDrop(e: DragEvent) {
  isDragging.value = false
  const files = e.dataTransfer?.files
  if (!files || files.length === 0) return
  loading.value = true
  try {
    for (let i = 0; i < files.length; i++) {
      await uploadFile(currentTab.value?.path || '/', files[i])
    }
    ElMessage.success(t('commons.success'))
    refreshFiles()
  } catch { /* */ } finally {
    loading.value = false
  }
}

// ===================== 剪贴板（复制/移动） =====================

const clipboard = ref<{ type: 'copy' | 'cut'; paths: string[] }>({ type: 'copy', paths: [] })

function setClipboard(type: 'copy' | 'cut', rows: any[]) {
  clipboard.value = { type, paths: rows.map(r => r.path) }
  const msg = type === 'copy'
    ? t('file.clipboardCopy', { count: rows.length })
    : t('file.clipboardCut', { count: rows.length })
  ElMessage.success(msg)
}

function clearClipboard() {
  clipboard.value = { type: 'copy', paths: [] }
}

async function doPaste() {
  if (clipboard.value.paths.length === 0) return
  loading.value = true
  try {
    await moveFile({
      srcPaths: clipboard.value.paths,
      dstPath: currentTab.value?.path || '/',
      isCopy: clipboard.value.type === 'copy',
    })
    ElMessage.success(t('commons.success'))
    clearClipboard()
    refreshFiles()
  } catch { /* */ } finally {
    loading.value = false
  }
}

// ===================== 压缩/解压 =====================

function isCompressFile(row: any): boolean {
  if (!row || row.isDir) return false
  const ext = getExt(row.name)
  return archiveExts.has(ext)
}

function handleCompressSingle(row: any) {
  if (!row) return
  compressRef.value?.openCompress([row.path], currentTab.value?.path || '/')
}

function batchCompress() {
  const paths = selectedRows.value.map(r => r.path)
  compressRef.value?.openCompress(paths, currentTab.value?.path || '/')
}

function handleDecompress(row: any) {
  if (!row) return
  compressRef.value?.openDecompress(row.path, currentTab.value?.path || '/')
}

// ===================== 权限 =====================

function openPermission(row: any) {
  if (!row) return
  permissionRef.value?.open(row.path, row.mode)
}

// ===================== 所有者 =====================

function openChown(row: any) {
  if (!row) return
  chownRef.value?.open(row.path, row.user, row.group)
}

// ===================== 右键菜单 =====================

const contextMenu = ref({ visible: false, x: 0, y: 0, row: null as any })

function handleContextMenu(row: any, _col: any, e: MouseEvent) {
  e.preventDefault()
  contextMenu.value = { visible: true, x: e.clientX, y: e.clientY, row }
}

function closeContextMenu() {
  contextMenu.value.visible = false
}

// ===================== 更多操作 =====================

function handleMoreAction(cmd: string, row: any) {
  switch (cmd) {
    case 'rename': handleRename(row); break
    case 'copyPath': copyPathToClipboard(row); break
    case 'copy': setClipboard('copy', [row]); break
    case 'cut': setClipboard('cut', [row]); break
    case 'permission': openPermission(row); break
    case 'chown': openChown(row); break
    case 'compress': handleCompressSingle(row); break
    case 'decompress': handleDecompress(row); break
    case 'delete': handleDelete(row); break
  }
}

function copyPathToClipboard(row: any) {
  if (!row) return
  navigator.clipboard.writeText(row.path).then(() => {
    ElMessage.success(t('commons.success'))
  })
}

// ===================== 工具函数 =====================

const formatSize = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0) + ' ' + units[i]
}

const formatTime = (iso: string): string => {
  if (!iso) return '-'
  const d = new Date(iso)
  return d.toLocaleString('zh-CN', { hour12: false })
}

// ===================== 表格高度自适应 =====================

function updateTableHeight() {
  // tabs ~42 + toolbar ~50 + breadcrumb ~40 + footer ~36 + padding ~70
  tableHeight.value = window.innerHeight - 330
}

// ===================== 生命周期 =====================

onMounted(() => {
  refreshFiles()
  updateTableHeight()
  window.addEventListener('resize', updateTableHeight)
  document.addEventListener('click', closeContextMenu)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', updateTableHeight)
  document.removeEventListener('click', closeContextMenu)
  if (searchTimer) clearTimeout(searchTimer)
})
</script>

<style lang="scss" scoped>
.file-manager {
  display: flex;
  flex-direction: column;
  height: 100%;
  position: relative;
}

.file-tabs {
  margin-bottom: 8px;

  :deep(.el-tabs__header) {
    margin-bottom: 0;
  }
}

.file-nav {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 10px;

  .path-input {
    flex: 1;
  }
}

.file-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 10px;
  flex-wrap: wrap;

  .toolbar-left {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-wrap: wrap;

    .batch-btn-group {
      .el-button {
        font-size: 12px;
      }
    }

    .paste-btn-group {
      margin-left: 4px;

      .el-button--success {
        font-weight: 500;
      }
    }
  }

  .toolbar-right {
    display: flex;
    align-items: center;
    gap: 6px;

    .search-input {
      width: 320px;

      :deep(.el-input-group__prepend) {
        padding: 0 8px;
        background: var(--xp-bg-surface);
      }

      :deep(.el-input-group__append) {
        padding: 0 8px;
      }

      :deep(.el-checkbox) {
        height: auto;
        margin-right: 0;
      }
    }
  }
}

.file-breadcrumb {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  background: var(--xp-bg-surface);
  border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius-sm);
  margin-bottom: 10px;
  font-size: 13px;
  overflow-x: auto;
  white-space: nowrap;

  .breadcrumb-item {
    cursor: pointer;
    color: var(--xp-text-secondary);
    transition: color 0.2s;

    &:hover .breadcrumb-text {
      color: var(--xp-accent);
    }

    &:last-child .breadcrumb-text {
      color: var(--xp-text-primary);
      font-weight: 500;
    }
  }

  .breadcrumb-separator {
    margin: 0 4px;
    color: var(--xp-text-muted);
  }
}

// 拖拽上传覆盖层
.drop-overlay {
  position: absolute;
  inset: 0;
  z-index: 100;
  background: rgba(34, 211, 238, 0.08);
  border: 2px dashed var(--xp-accent);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: none;

  .drop-text {
    text-align: center;
    color: var(--xp-accent);
    font-size: 16px;
    font-weight: 500;

    .el-icon {
      margin-bottom: 8px;
    }
  }
}

.file-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: default;

  .file-icon {
    font-size: 18px;
    flex-shrink: 0;
  }

  .file-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;

    &.is-link {
      font-style: italic;
      color: var(--xp-accent-secondary);
    }
  }
}

.file-footer {
  display: flex;
  align-items: center;
  padding: 10px 12px;
  font-size: 12px;
  color: var(--xp-text-muted);
  border-top: 1px solid var(--xp-border-light);
  margin-top: 8px;
}

// 右键菜单
.context-menu {
  position: fixed;
  z-index: 9999;
  background: var(--xp-bg-surface);
  border: 1px solid var(--xp-border);
  border-radius: 8px;
  padding: 4px 0;
  min-width: 180px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);

  .context-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 16px;
    font-size: 13px;
    color: var(--xp-text-secondary);
    cursor: pointer;
    transition: all 0.15s;

    &:hover {
      background: rgba(34, 211, 238, 0.08);
      color: var(--xp-accent);
    }

    &.danger:hover {
      background: rgba(239, 68, 68, 0.08);
      color: var(--el-color-danger);
    }

    .el-icon {
      font-size: 15px;
    }
  }

  .context-divider {
    height: 1px;
    margin: 4px 8px;
    background: var(--xp-border-light);
  }
}
</style>
