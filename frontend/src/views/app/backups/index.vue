<template>
  <div>
    <div class="page-header">
      <h3>{{ $t('app.backups') }}</h3>
    </div>

    <el-card shadow="never">
      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-input
          v-model="searchForm.name"
          :placeholder="$t('app.searchBackup')"
          clearable
          style="width: 300px;"
          @clear="search"
          @keyup.enter="search"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>

        <el-button type="primary" @click="search" style="margin-left: 12px;">
          {{ $t('commons.search') }}
        </el-button>
        <el-button @click="search" style="margin-left: 12px;">
          <el-icon><Refresh /></el-icon>{{ $t('commons.refresh') }}
        </el-button>
      </div>

      <!-- 备份列表 -->
      <el-table :data="backups" v-loading="loading" stripe>
        <el-table-column prop="appName" :label="$t('app.appName')" min-width="120" />
        <el-table-column prop="backupName" :label="$t('app.backupName')" min-width="180" />
        <el-table-column prop="backupType" :label="$t('app.backupType')" width="100">
          <template #default="{ row }">
            <el-tag size="small">{{ row.backupType }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="sizeStr" :label="$t('app.size')" width="100" />
        <el-table-column prop="status" :label="$t('commons.status')" width="100">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" size="small">
              {{ statusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" :label="$t('app.createdAt')" width="160" />
        <el-table-column :label="$t('commons.actions')" width="200" fixed="right">
          <template #default="{ row }">
            <el-button
              size="small"
              type="primary"
              @click="handleRestore(row)"
              :disabled="row.status !== 'success'"
            >
              {{ $t('app.restore') }}
            </el-button>
            <el-button
              size="small"
              @click="showDetail(row)"
            >
              {{ $t('commons.view') }}
            </el-button>
            <el-popconfirm
              :title="$t('app.deleteBackupConfirm')"
              @confirm="handleDelete(row)"
            >
              <template #reference>
                <el-button size="small" type="danger">
                  {{ $t('commons.delete') }}
                </el-button>
              </template>
            </el-popconfirm>
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

    <!-- 详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      :title="$t('app.backupDetail')"
      width="600px"
    >
      <el-descriptions v-if="currentBackup" :column="1" border>
        <el-descriptions-item :label="$t('app.appName')">
          {{ currentBackup.appName }}
        </el-descriptions-item>
        <el-descriptions-item :label="$t('app.backupName')">
          {{ currentBackup.backupName }}
        </el-descriptions-item>
        <el-descriptions-item :label="$t('app.backupType')">
          {{ currentBackup.backupType }}
        </el-descriptions-item>
        <el-descriptions-item :label="$t('app.backupPath')">
          {{ currentBackup.backupPath }}
        </el-descriptions-item>
        <el-descriptions-item :label="$t('app.size')">
          {{ currentBackup.sizeStr }}
        </el-descriptions-item>
        <el-descriptions-item :label="$t('app.checksum')">
          <code style="font-size: 12px;">{{ currentBackup.checksum }}</code>
        </el-descriptions-item>
        <el-descriptions-item :label="$t('commons.status')">
          <el-tag :type="statusType(currentBackup.status)" size="small">
            {{ statusText(currentBackup.status) }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item :label="$t('app.message')" v-if="currentBackup.message">
          {{ currentBackup.message }}
        </el-descriptions-item>
        <el-descriptions-item :label="$t('app.createdAt')">
          {{ currentBackup.createdAt }}
        </el-descriptions-item>
      </el-descriptions>
      <template #footer>
        <el-button @click="detailDialogVisible = false">{{ $t('commons.close') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Search, Refresh } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import {
  searchBackups,
  restoreApp,
  deleteBackups,
  type App
} from '@/api/modules/app'

const { t } = useI18n()

const loading = ref(false)
const backups = ref<App.AppBackupDTO[]>([])

const searchForm = reactive({
  name: '',
})

const paginationConfig = reactive({
  page: 1,
  pageSize: 20,
  total: 0,
})

// 详情对话框
const detailDialogVisible = ref(false)
const currentBackup = ref<App.AppBackupDTO | null>(null)

const statusType = (status: string) => {
  const map: Record<string, string> = {
    success: 'success',
    failed: 'danger',
    running: 'warning',
  }
  return map[status] || 'info'
}

const statusText = (status: string) => {
  const map: Record<string, string> = {
    success: t('app.backupSuccess'),
    failed: t('app.backupFailed'),
    running: t('app.backupRunning'),
  }
  return map[status] || status
}

const search = async () => {
  loading.value = true
  try {
    const res = await searchBackups({
      page: paginationConfig.page,
      pageSize: paginationConfig.pageSize,
      name: searchForm.name || undefined,
    })
    backups.value = res.data.items
    paginationConfig.total = res.data.total
  } finally {
    loading.value = false
  }
}

const handleRestore = async (backup: App.AppBackupDTO) => {
  await ElMessageBox.confirm(
    t('app.restoreConfirm', { name: backup.appName, backup: backup.backupName }),
    t('commons.warning'),
    { type: 'warning' }
  )

  try {
    await restoreApp({
      installId: backup.appInstallId,
      backupId: backup.id,
    })
    ElMessage.success(t('app.restoreSuccess'))
  } catch (err: any) {
    ElMessage.error(err.message || t('app.restoreFailed'))
  }
}

const showDetail = (backup: App.AppBackupDTO) => {
  currentBackup.value = backup
  detailDialogVisible.value = true
}

const handleDelete = async (backup: App.AppBackupDTO) => {
  try {
    await deleteBackups([backup.id])
    ElMessage.success(t('commons.deleteSuccess'))
    search()
  } catch (err: any) {
    ElMessage.error(err.message || t('commons.operationFailed'))
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
