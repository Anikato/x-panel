<template>
  <div>
    <div class="page-header">
      <h3>{{ $t('app.store') }}</h3>
      <div>
        <el-button @click="showImportDialog">
          <el-icon><Upload /></el-icon>{{ $t('app.importFromBackup') }}
        </el-button>
        <el-button @click="syncStore" :loading="syncing" style="margin-left: 12px;">
          <el-icon><Refresh /></el-icon>{{ $t('app.syncStore') }}
        </el-button>
      </div>
    </div>

    <el-card shadow="never">
      <!-- 搜索和筛选 -->
      <div class="search-bar">
        <el-input
          v-model="searchForm.name"
          :placeholder="$t('app.searchApp')"
          clearable
          style="width: 300px;"
          @clear="search"
          @keyup.enter="search"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>

        <el-select
          v-model="searchForm.type"
          :placeholder="$t('app.appType')"
          clearable
          style="width: 150px; margin-left: 12px;"
          @change="search"
        >
          <el-option label="全部类型" value="" />
          <el-option label="网站" value="website" />
          <el-option label="数据库" value="database" />
          <el-option label="开发工具" value="tool" />
          <el-option label="其他" value="other" />
        </el-select>

        <div style="margin-left: 12px;">
          <el-tag
            v-for="tag in tags"
            :key="tag.key"
            :type="selectedTags.includes(tag.key) ? 'primary' : 'info'"
            style="margin-right: 8px; cursor: pointer;"
            @click="toggleTag(tag.key)"
          >
            {{ tag.name }}
          </el-tag>
        </div>

        <el-button type="primary" @click="search" style="margin-left: auto;">
          {{ $t('commons.search') }}
        </el-button>
      </div>

      <!-- 应用列表 -->
      <div class="app-grid" v-loading="loading">
        <el-empty v-if="!loading && apps.length === 0" :description="$t('commons.noData')" />
        
        <div v-for="app in apps" :key="app.id" class="app-card">
          <div class="app-icon">
            <img :src="app.icon || '/default-app-icon.png'" :alt="app.name" />
          </div>
          <div class="app-info">
            <div class="app-name">{{ app.name }}</div>
            <div class="app-desc">{{ app.shortDescZh || app.shortDescEn }}</div>
            <div class="app-meta">
              <el-tag size="small" type="info">{{ app.type }}</el-tag>
              <span class="install-count">{{ app.installedCount }} 次安装</span>
            </div>
          </div>
          <div class="app-actions">
            <el-button type="primary" size="small" @click="showInstallDialog(app)">
              {{ $t('app.install') }}
            </el-button>
            <el-button size="small" @click="showAppDetail(app)">
              {{ $t('commons.view') }}
            </el-button>
          </div>
        </div>
      </div>

      <!-- 分页 -->
      <el-pagination
        v-if="paginationConfig.total > 0"
        v-model:current-page="paginationConfig.page"
        v-model:page-size="paginationConfig.pageSize"
        :total="paginationConfig.total"
        :page-sizes="[12, 24, 48]"
        layout="total, sizes, prev, pager, next"
        @current-change="search"
        @size-change="search"
        style="margin-top: 16px; justify-content: center;"
      />
    </el-card>

    <!-- 安装对话框 -->
    <el-dialog
      v-model="installDialogVisible"
      :title="$t('app.installApp')"
      width="600px"
      @close="resetInstallForm"
    >
      <el-form :model="installForm" :rules="installRules" ref="installFormRef" label-width="120px">
        <el-form-item :label="$t('app.appName')" prop="name">
          <el-input v-model="installForm.name" :placeholder="$t('app.installNameHint')" />
        </el-form-item>

        <el-form-item :label="$t('app.version')" prop="version">
          <el-select v-model="installForm.version" style="width: 100%;" @change="loadAppDetail">
            <el-option
              v-for="ver in currentApp?.versions || []"
              :key="ver"
              :label="ver"
              :value="ver"
            />
          </el-select>
        </el-form-item>

        <!-- 动态参数 -->
        <div v-if="appDetail">
          <el-form-item
            v-for="(param, key) in appDetail.params"
            :key="key"
            :label="param.labelZh || param.labelEn || key"
            :prop="`params.${key}`"
          >
            <el-input
              v-if="param.type === 'text' || !param.type"
              v-model="installForm.params[key]"
              :placeholder="param.default"
            />
            <el-input-number
              v-else-if="param.type === 'number'"
              v-model="installForm.params[key]"
              :min="param.min"
              :max="param.max"
              style="width: 100%;"
            />
            <el-switch
              v-else-if="param.type === 'boolean'"
              v-model="installForm.params[key]"
            />
            <el-select
              v-else-if="param.type === 'select'"
              v-model="installForm.params[key]"
              style="width: 100%;"
            >
              <el-option
                v-for="opt in param.options"
                :key="opt.value"
                :label="opt.label"
                :value="opt.value"
              />
            </el-select>
            <template #extra v-if="param.description">
              <div style="color: #909399; font-size: 12px;">{{ param.description }}</div>
            </template>
          </el-form-item>
        </div>
      </el-form>

      <template #footer>
        <el-button @click="installDialogVisible = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleInstall" :loading="installing">
          {{ $t('app.install') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 应用详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      :title="currentApp?.name"
      width="800px"
    >
      <div v-if="currentApp" class="app-detail">
        <div class="detail-header">
          <img :src="currentApp.icon" class="detail-icon" />
          <div class="detail-info">
            <h2>{{ currentApp.name }}</h2>
            <p>{{ currentApp.shortDescZh || currentApp.shortDescEn }}</p>
            <div class="detail-meta">
              <el-tag v-for="tag in currentApp.tags" :key="tag" size="small" style="margin-right: 8px;">
                {{ tag }}
              </el-tag>
            </div>
          </div>
        </div>
        <el-divider />
        <div class="detail-content">
          <h3>{{ $t('app.description') }}</h3>
          <div v-html="currentApp.description"></div>
          
          <h3 style="margin-top: 24px;">{{ $t('app.versions') }}</h3>
          <el-tag v-for="ver in currentApp.versions" :key="ver" style="margin-right: 8px;">
            {{ ver }}
          </el-tag>

          <div v-if="currentApp.website || currentApp.github || currentApp.document" style="margin-top: 24px;">
            <h3>{{ $t('app.links') }}</h3>
            <div>
              <el-link v-if="currentApp.website" :href="currentApp.website" target="_blank" type="primary">
                {{ $t('app.officialWebsite') }}
              </el-link>
              <el-link v-if="currentApp.github" :href="currentApp.github" target="_blank" type="primary" style="margin-left: 16px;">
                GitHub
              </el-link>
              <el-link v-if="currentApp.document" :href="currentApp.document" target="_blank" type="primary" style="margin-left: 16px;">
                {{ $t('app.documentation') }}
              </el-link>
            </div>
          </div>
        </div>
      </div>
    </el-dialog>

    <!-- 导入对话框 -->
    <el-dialog
      v-model="importDialogVisible"
      :title="$t('app.importFromBackup')"
      width="600px"
      @close="resetImportForm"
    >
      <el-form :model="importForm" :rules="importRules" ref="importFormRef" label-width="120px">
        <el-form-item :label="$t('app.installName')" prop="name">
          <el-input v-model="importForm.name" :placeholder="$t('app.installNameHint')" />
        </el-form-item>

        <el-form-item :label="$t('app.backupFilePath')" prop="backupPath">
          <el-input v-model="importForm.backupPath" :placeholder="$t('app.backupPathHint')" />
          <template #extra>
            <div style="color: #909399; font-size: 12px; margin-top: 4px;">
              {{ $t('app.backupPathDesc') }}
            </div>
          </template>
        </el-form-item>

        <el-form-item :label="$t('app.appKey')" prop="appKey">
          <el-input v-model="importForm.appKey" :placeholder="$t('app.appKeyHint')" />
          <template #extra>
            <div style="color: #909399; font-size: 12px; margin-top: 4px;">
              {{ $t('app.appKeyDesc') }}
            </div>
          </template>
        </el-form-item>

        <el-form-item :label="$t('app.version')" prop="version">
          <el-input v-model="importForm.version" :placeholder="$t('app.versionHint')" />
        </el-form-item>

        <el-alert
          :title="$t('app.importWarning')"
          type="warning"
          :closable="false"
          show-icon
          style="margin-bottom: 16px;"
        >
          <template #default>
            <div v-html="$t('app.importWarningDesc')"></div>
          </template>
        </el-alert>
      </el-form>

      <template #footer>
        <el-button @click="importDialogVisible = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleImport" :loading="importing">
          {{ $t('app.import') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 导入进度对话框 -->
    <el-dialog
      v-model="showProgress"
      :title="$t('app.importProgress')"
      width="500px"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      @close="closeProgressDialog"
    >
      <div v-if="importProgress" class="import-progress">
        <div class="progress-info">
          <div class="progress-step">{{ importProgress.currentStep }}</div>
          <div class="progress-bar">
            <el-progress 
              :percentage="importProgress.progress" 
              :status="importProgress.status === 'failed' ? 'exception' : undefined"
              :stroke-width="8"
            />
          </div>
          <div class="progress-details">
            <div><strong>{{ $t('app.installName') }}:</strong> {{ importProgress.name }}</div>
            <div><strong>{{ $t('app.status') }}:</strong> 
              <el-tag 
                :type="importProgress.status === 'success' ? 'success' : 
                       importProgress.status === 'failed' ? 'danger' : 
                       importProgress.status === 'running' ? 'warning' : 'info'"
              >
                {{ $t(`app.${importProgress.status}`) }}
              </el-tag>
            </div>
            <div v-if="importProgress.message"><strong>{{ $t('app.message') }}:</strong> {{ importProgress.message }}</div>
          </div>
        </div>
      </div>

      <template #footer>
        <el-button 
          v-if="importProgress?.status === 'success' || importProgress?.status === 'failed'" 
          @click="closeProgressDialog"
        >
          {{ $t('commons.close') }}
        </el-button>
        <el-button 
          v-else 
          @click="closeProgressDialog"
          type="info"
        >
          {{ $t('app.runInBackground') }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Refresh, Search, Upload } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import {
  syncAppStore,
  searchApps,
  getAppTags,
  getAppDetail,
  installApp,
  importApp,
  getImportProgress,
  type App
} from '@/api/modules/app'

const { t } = useI18n()
const router = useRouter()

const loading = ref(false)
const syncing = ref(false)
const apps = ref<App.AppDTO[]>([])
const tags = ref<App.TagDTO[]>([])
const selectedTags = ref<string[]>([])

const searchForm = reactive({
  name: '',
  type: '',
})

const paginationConfig = reactive({
  page: 1,
  pageSize: 12,
  total: 0,
})

// 安装对话框
const installDialogVisible = ref(false)
const installing = ref(false)
const currentApp = ref<App.AppDTO | null>(null)
const appDetail = ref<App.AppDetailDTO | null>(null)
const installFormRef = ref<FormInstance>()
const installForm = reactive({
  name: '',
  version: '',
  params: {} as Record<string, any>,
})

const installRules: FormRules = {
  name: [{ required: true, message: t('app.installNameRequired'), trigger: 'blur' }],
  version: [{ required: true, message: t('app.versionRequired'), trigger: 'change' }],
}

// 详情对话框
const detailDialogVisible = ref(false)

// 导入对话框
const importDialogVisible = ref(false)
const importing = ref(false)
const importFormRef = ref<FormInstance>()
const importForm = reactive({
  name: '',
  backupPath: '',
  appKey: '',
  version: '',
})

// 导入进度
const importProgress = ref<App.AppImportTask | null>(null)
const showProgress = ref(false)
let progressTimer: NodeJS.Timeout | null = null

const importRules: FormRules = {
  name: [{ required: true, message: t('app.installNameRequired'), trigger: 'blur' }],
  backupPath: [{ required: true, message: t('app.backupPathRequired'), trigger: 'blur' }],
}

const search = async () => {
  loading.value = true
  try {
    const res = await searchApps({
      page: paginationConfig.page,
      pageSize: paginationConfig.pageSize,
      name: searchForm.name || undefined,
      type: searchForm.type || undefined,
      tags: selectedTags.value.length > 0 ? selectedTags.value : undefined,
    })
    apps.value = res.data.items
    paginationConfig.total = res.data.total
  } finally {
    loading.value = false
  }
}

const loadTags = async () => {
  try {
    const res = await getAppTags()
    tags.value = res.data
  } catch (err) {
    console.error('Failed to load tags:', err)
  }
}

const toggleTag = (tagKey: string) => {
  const index = selectedTags.value.indexOf(tagKey)
  if (index > -1) {
    selectedTags.value.splice(index, 1)
  } else {
    selectedTags.value.push(tagKey)
  }
  search()
}

const syncStore = async () => {
  syncing.value = true
  try {
    await syncAppStore({ force: false })
    ElMessage.success(t('app.syncSuccess'))
    search()
    loadTags()
  } catch (err: any) {
    ElMessage.error(err.message || t('app.syncFailed'))
  } finally {
    syncing.value = false
  }
}

const showInstallDialog = (app: App.AppDTO) => {
  currentApp.value = app
  installForm.name = app.key
  installForm.version = app.versions[0] || ''
  installForm.params = {}
  installDialogVisible.value = true
  if (installForm.version) {
    loadAppDetail()
  }
}

const loadAppDetail = async () => {
  if (!currentApp.value || !installForm.version) return
  try {
    const res = await getAppDetail(currentApp.value.id, installForm.version)
    appDetail.value = res.data
    // 初始化参数默认值
    if (appDetail.value?.params) {
      Object.keys(appDetail.value.params).forEach(key => {
        const param = appDetail.value!.params[key]
        if (param.default !== undefined && !installForm.params[key]) {
          installForm.params[key] = param.default
        }
      })
    }
  } catch (err: any) {
    ElMessage.error(err.message || t('app.loadDetailFailed'))
  }
}

const handleInstall = async () => {
  if (!installFormRef.value || !currentApp.value || !appDetail.value) return
  
  await installFormRef.value.validate()
  
  installing.value = true
  try {
    await installApp({
      name: installForm.name,
      appId: currentApp.value.id,
      appDetailId: appDetail.value.id,
      params: installForm.params,
    })
    ElMessage.success(t('app.installStarted'))
    installDialogVisible.value = false
    router.push('/app/installed')
  } catch (err: any) {
    ElMessage.error(err.message || t('app.installFailed'))
  } finally {
    installing.value = false
  }
}

const resetInstallForm = () => {
  installFormRef.value?.resetFields()
  installForm.params = {}
  appDetail.value = null
}

const showAppDetail = (app: App.AppDTO) => {
  currentApp.value = app
  detailDialogVisible.value = true
}

const showImportDialog = () => {
  importForm.name = ''
  importForm.backupPath = ''
  importForm.appKey = ''
  importForm.version = ''
  importDialogVisible.value = true
}

const handleImport = async () => {
  if (!importFormRef.value) return
  
  await importFormRef.value.validate()
  
  importing.value = true
  try {
    await importApp({
      name: importForm.name,
      backupPath: importForm.backupPath,
      appKey: importForm.appKey || undefined,
      version: importForm.version || undefined,
    })
    
    ElMessage.success(t('app.importTaskStarted'))
    importDialogVisible.value = false
    
    // 显示进度对话框
    showProgress.value = true
    startProgressPolling(importForm.name)
    
  } catch (err: any) {
    ElMessage.error(err.message || t('app.importFailed'))
  } finally {
    importing.value = false
  }
}

// 开始轮询进度
const startProgressPolling = (taskName: string) => {
  const pollProgress = async () => {
    try {
      const res = await getImportProgress(taskName)
      importProgress.value = res.data
      
      if (res.data.status === 'success') {
        ElMessage.success(t('app.importSuccess'))
        showProgress.value = false
        stopProgressPolling()
        router.push('/app/installed')
      } else if (res.data.status === 'failed') {
        ElMessage.error(res.data.message || t('app.importFailed'))
        showProgress.value = false
        stopProgressPolling()
      } else if (res.data.status === 'running' || res.data.status === 'pending') {
        // 继续轮询
        progressTimer = setTimeout(pollProgress, 2000)
      }
    } catch (err: any) {
      console.error('Failed to get import progress:', err)
      stopProgressPolling()
    }
  }
  
  pollProgress()
}

// 停止轮询进度
const stopProgressPolling = () => {
  if (progressTimer) {
    clearTimeout(progressTimer)
    progressTimer = null
  }
}

// 关闭进度对话框
const closeProgressDialog = () => {
  showProgress.value = false
  stopProgressPolling()
}

const resetImportForm = () => {
  importFormRef.value?.resetFields()
}

onMounted(() => {
  search()
  loadTags()
})
</script>

<style lang="scss" scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  h3 { margin: 0; }
}

.search-bar {
  display: flex;
  align-items: center;
  margin-bottom: 24px;
}

.app-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
  min-height: 200px;
}

.app-card {
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  padding: 16px;
  transition: all 0.3s;
  display: flex;
  flex-direction: column;

  &:hover {
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
    transform: translateY(-2px);
  }

  .app-icon {
    width: 64px;
    height: 64px;
    margin-bottom: 12px;

    img {
      width: 100%;
      height: 100%;
      object-fit: contain;
    }
  }

  .app-info {
    flex: 1;

    .app-name {
      font-size: 16px;
      font-weight: 600;
      margin-bottom: 8px;
    }

    .app-desc {
      font-size: 13px;
      color: #909399;
      margin-bottom: 12px;
      display: -webkit-box;
      -webkit-line-clamp: 2;
      -webkit-box-orient: vertical;
      overflow: hidden;
    }

    .app-meta {
      display: flex;
      align-items: center;
      gap: 12px;
      font-size: 12px;
      color: #909399;
    }
  }

  .app-actions {
    display: flex;
    gap: 8px;
    margin-top: 12px;
  }
}

.app-detail {
  .detail-header {
    display: flex;
    gap: 16px;

    .detail-icon {
      width: 80px;
      height: 80px;
      object-fit: contain;
    }

    .detail-info {
      flex: 1;

      h2 {
        margin: 0 0 8px 0;
      }

      p {
        color: #606266;
        margin: 0 0 12px 0;
      }
    }
  }

  .detail-content {
    h3 {
      margin: 0 0 12px 0;
      font-size: 16px;
    }
  }
}
</style>

.import-progress {
  .progress-info {
    text-align: center;
    
    .progress-step {
      font-size: 16px;
      font-weight: 500;
      margin-bottom: 16px;
      color: var(--el-text-color-primary);
    }
    
    .progress-bar {
      margin-bottom: 24px;
    }
    
    .progress-details {
      text-align: left;
      background: var(--el-fill-color-lighter);
      padding: 16px;
      border-radius: 8px;
      
      > div {
        margin-bottom: 8px;
        
        &:last-child {
          margin-bottom: 0;
        }
      }
    }
  }
}