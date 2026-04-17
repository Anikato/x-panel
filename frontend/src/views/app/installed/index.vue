<template>
  <div>
    <div class="page-header">
      <h3>{{ $t('app.installed') }}</h3>
      <el-button type="primary" @click="router.push('/app/store')">
        <el-icon><Plus /></el-icon>{{ $t('app.installMore') }}
      </el-button>
    </div>

    <el-card shadow="never">
      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-input
          v-model="searchForm.name"
          :placeholder="$t('app.searchInstalled')"
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

        <el-button type="primary" @click="search" style="margin-left: 12px;">
          {{ $t('commons.search') }}
        </el-button>
        <el-button @click="search" style="margin-left: 12px;">
          <el-icon><Refresh /></el-icon>{{ $t('commons.refresh') }}
        </el-button>
      </div>

      <!-- 应用列表 -->
      <el-table :data="apps" v-loading="loading" stripe>
        <el-table-column prop="appIcon" label="" width="60">
          <template #default="{ row }">
            <img :src="row.appIcon || '/default-app-icon.png'" style="width: 40px; height: 40px; object-fit: contain;" />
          </template>
        </el-table-column>
        <el-table-column prop="name" :label="$t('app.installName')" min-width="150" />
        <el-table-column prop="appName" :label="$t('app.appName')" min-width="120" />
        <el-table-column prop="version" :label="$t('app.version')" width="100" />
        <el-table-column prop="status" :label="$t('commons.status')" width="100">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" size="small">
              {{ statusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="webUi" :label="$t('app.webUI')" width="120">
          <template #default="{ row }">
            <el-link v-if="row.webUi" :href="row.webUi" target="_blank" type="primary">
              {{ $t('app.openWebUI') }}
            </el-link>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="installedAt" :label="$t('app.installedAt')" width="160" />
        <el-table-column :label="$t('commons.actions')" width="280" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.status === 'running'"
              size="small"
              @click="handleOperate(row, 'stop')"
            >
              {{ $t('app.stop') }}
            </el-button>
            <el-button
              v-else-if="row.status === 'stopped'"
              size="small"
              type="success"
              @click="handleOperate(row, 'start')"
            >
              {{ $t('app.start') }}
            </el-button>
            <el-button size="small" @click="handleOperate(row, 'restart')">
              {{ $t('app.restart') }}
            </el-button>
            <el-button size="small" @click="showBackupDialog(row)">
              {{ $t('app.backup') }}
            </el-button>
            <el-dropdown @command="(cmd: string) => handleDropdown(cmd, row)">
              <el-button size="small">
                {{ $t('commons.more') }}<el-icon class="el-icon--right"><arrow-down /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="update" v-if="row.canUpdate">
                    {{ $t('app.update') }} ({{ row.latestVersion }})
                  </el-dropdown-item>
                  <el-dropdown-item command="logs">{{ $t('app.viewLogs') }}</el-dropdown-item>
                  <el-dropdown-item command="uninstall" divided>{{ $t('app.uninstall') }}</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <el-pagination
        v-if="paginationConfig.total > 0"
        v-model:current-page="paginationConfig.page"
        v-model:page-size="paginationConfig.pageSize"
        :total="paginationConfig.total"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next"
        @current-change="search"
        @size-change="search"
        style="margin-top: 16px; justify-content: flex-end;"
      />
    </el-card>

    <!-- 备份对话框 -->
    <el-dialog
      v-model="backupDialogVisible"
      :title="$t('app.createBackup')"
      width="500px"
    >
      <el-form :model="backupForm" :rules="backupRules" ref="backupFormRef" label-width="100px">
        <el-form-item :label="$t('app.backupName')" prop="backupName">
          <el-input v-model="backupForm.backupName" :placeholder="$t('app.backupNameHint')" />
        </el-form-item>
        <el-form-item :label="$t('app.description')">
          <el-input v-model="backupForm.description" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="backupDialogVisible = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleBackup" :loading="backing">
          {{ $t('app.backup') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 日志对话框 -->
    <el-dialog
      v-model="logsDialogVisible"
      :title="$t('app.containerLogs')"
      width="800px"
    >
      <el-input
        v-model="logs"
        type="textarea"
        :rows="20"
        readonly
        style="font-family: monospace; font-size: 12px;"
      />
      <template #footer>
        <el-button @click="logsDialogVisible = false">{{ $t('commons.close') }}</el-button>
        <el-button @click="loadLogs">{{ $t('commons.refresh') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Plus, Search, ArrowDown } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import {
  searchInstalled,
  operateApp,
  uninstallApp,
  updateApp,
  backupApp,
  getAppLogs,
  type App
} from '@/api/modules/app'

const { t } = useI18n()
const router = useRouter()

const loading = ref(false)
const apps = ref<App.AppInstallDTO[]>([])

const searchForm = reactive({
  name: '',
  type: '',
})

const paginationConfig = reactive({
  page: 1,
  pageSize: 20,
  total: 0,
})

// 备份对话框
const backupDialogVisible = ref(false)
const backing = ref(false)
const currentApp = ref<App.AppInstallDTO | null>(null)
const backupFormRef = ref<FormInstance>()
const backupForm = reactive({
  backupName: '',
  description: '',
})

const backupRules: FormRules = {
  backupName: [{ required: true, message: t('app.backupNameRequired'), trigger: 'blur' }],
}

// 日志对话框
const logsDialogVisible = ref(false)
const logs = ref('')

const statusType = (status: string) => {
  const map: Record<string, string> = {
    running: 'success',
    stopped: 'info',
    installing: 'warning',
    error: 'danger',
  }
  return map[status] || 'info'
}

const statusText = (status: string) => {
  const map: Record<string, string> = {
    running: t('app.running'),
    stopped: t('app.stopped'),
    installing: t('app.installing'),
    error: t('app.error'),
  }
  return map[status] || status
}

const search = async () => {
  loading.value = true
  try {
    const res = await searchInstalled({
      page: paginationConfig.page,
      pageSize: paginationConfig.pageSize,
      name: searchForm.name || undefined,
      type: searchForm.type || undefined,
    })
    apps.value = res.data.items
    paginationConfig.total = res.data.total
  } finally {
    loading.value = false
  }
}

const handleOperate = async (app: App.AppInstallDTO, operation: 'start' | 'stop' | 'restart') => {
  try {
    await operateApp({
      installId: app.id,
      operation,
    })
    ElMessage.success(t('commons.operationSuccess'))
    search()
  } catch (err: any) {
    ElMessage.error(err.message || t('commons.operationFailed'))
  }
}

const handleDropdown = async (command: string, app: App.AppInstallDTO) => {
  currentApp.value = app

  switch (command) {
    case 'update':
      await handleUpdate(app)
      break
    case 'logs':
      await showLogs(app)
      break
    case 'uninstall':
      await handleUninstall(app)
      break
  }
}

const handleUpdate = async (app: App.AppInstallDTO) => {
  await ElMessageBox.confirm(
    t('app.updateConfirm', { name: app.name, version: app.latestVersion }),
    t('commons.warning'),
    { type: 'warning' }
  )

  try {
    // TODO: 需要获取最新版本的 appDetailId
    ElMessage.info(t('app.updateNotImplemented'))
  } catch (err: any) {
    ElMessage.error(err.message || t('app.updateFailed'))
  }
}

const showLogs = async (app: App.AppInstallDTO) => {
  currentApp.value = app
  logsDialogVisible.value = true
  await loadLogs()
}

const loadLogs = async () => {
  if (!currentApp.value) return
  
  try {
    const res = await getAppLogs(currentApp.value.id, 500)
    logs.value = res.data || ''
  } catch (err: any) {
    ElMessage.error(err.message || t('app.loadLogsFailed'))
    logs.value = `加载日志失败: ${err.message || '未知错误'}`
  }
}

const handleUninstall = async (app: App.AppInstallDTO) => {
  await ElMessageBox.confirm(
    t('app.uninstallConfirm', { name: app.name }),
    t('commons.warning'),
    {
      type: 'warning',
      confirmButtonText: t('app.uninstall'),
      cancelButtonText: t('commons.cancel'),
    }
  )

  try {
    await uninstallApp({
      installId: app.id,
      deleteData: true,
      forceDelete: false,
    })
    ElMessage.success(t('app.uninstallSuccess'))
    search()
  } catch (err: any) {
    ElMessage.error(err.message || t('app.uninstallFailed'))
  }
}

const showBackupDialog = (app: App.AppInstallDTO) => {
  currentApp.value = app
  backupForm.backupName = `${app.name}_${new Date().toISOString().slice(0, 10)}`
  backupForm.description = ''
  backupDialogVisible.value = true
}

const handleBackup = async () => {
  if (!backupFormRef.value || !currentApp.value) return
  
  await backupFormRef.value.validate()
  
  backing.value = true
  try {
    await backupApp({
      installId: currentApp.value.id,
      backupName: backupForm.backupName,
      description: backupForm.description || undefined,
    })
    ElMessage.success(t('app.backupSuccess'))
    backupDialogVisible.value = false
  } catch (err: any) {
    ElMessage.error(err.message || t('app.backupFailed'))
  } finally {
    backing.value = false
  }
}

onMounted(() => {
  search()
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
  margin-bottom: 16px;
}
</style>
