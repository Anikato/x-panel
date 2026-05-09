<template>
  <div class="notification-page">
    <div class="page-header">
      <div>
        <h2>{{ t('notification.title') }}</h2>
        <p>{{ t('notification.subtitle') }}</p>
      </div>
      <div class="header-actions">
        <el-button :icon="Refresh" @click="loadNotifications">{{ t('commons.refresh') }}</el-button>
        <el-button :icon="CircleCheck" type="primary" @click="handleMarkAllRead">
          {{ t('notification.markAllRead') }}
        </el-button>
        <el-button :icon="Delete" type="danger" plain @click="handleClearRead">
          {{ t('notification.clearRead') }}
        </el-button>
      </div>
    </div>

    <div class="filter-row">
      <el-segmented v-model="query.status" :options="statusOptions" @change="reloadFirstPage" />
      <el-select v-model="query.type" clearable style="width: 140px" :placeholder="t('notification.type')">
        <el-option :label="t('commons.all')" value="" />
        <el-option :label="t('notification.typeSuccess')" value="success" />
        <el-option :label="t('notification.typeError')" value="error" />
        <el-option :label="t('notification.typeWarning')" value="warning" />
        <el-option :label="t('notification.typeInfo')" value="info" />
      </el-select>
      <el-input
        v-model="query.info"
        class="filter-keyword"
        clearable
        :placeholder="t('notification.keywordPlaceholder')"
        @keyup.enter="reloadFirstPage"
      />
      <el-button type="primary" @click="reloadFirstPage">{{ t('commons.search') }}</el-button>
    </div>

    <el-table v-loading="loading" :data="items" class="notification-table" row-key="id">
      <el-table-column width="64">
        <template #default="{ row }">
          <span class="type-dot" :class="row.type"></span>
        </template>
      </el-table-column>
      <el-table-column :label="t('notification.content')" min-width="360">
        <template #default="{ row }">
          <div class="notification-main">
            <div class="notification-title">
              <span v-if="!row.readAt" class="unread-dot"></span>
              <span>{{ row.title }}</span>
            </div>
            <div v-if="row.content" class="notification-content">{{ row.content }}</div>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="source" :label="t('notification.source')" width="120">
        <template #default="{ row }">
          <el-tag size="small" effect="plain">{{ sourceLabel(row.source) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="t('notification.status')" width="100">
        <template #default="{ row }">
          <el-tag :type="row.readAt ? 'info' : 'success'" size="small">
            {{ row.readAt ? t('notification.read') : t('notification.unread') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="createdAt" :label="t('notification.createdAt')" width="180">
        <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
      </el-table-column>
      <el-table-column :label="t('commons.actions')" width="220" fixed="right">
        <template #default="{ row }">
          <el-button v-if="!row.readAt" link type="primary" @click="handleMarkRead(row)">
            {{ t('notification.markRead') }}
          </el-button>
          <el-button v-if="row.targetUrl" link type="primary" @click="openTarget(row)">
            {{ t('notification.viewTarget') }}
          </el-button>
          <el-button link type="danger" @click="handleDelete(row)">{{ t('commons.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pager-row">
      <el-pagination
        v-model:current-page="query.page"
        v-model:page-size="query.pageSize"
        :total="total"
        layout="total, prev, pager, next"
        @current-change="loadNotifications"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import { CircleCheck, Delete, Refresh } from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'
import type { NotificationItem, NotificationSearchReq } from '@/api/interface'
import {
  clearReadNotifications,
  deleteNotification,
  markAllNotificationsRead,
  markNotificationsRead,
  searchNotifications,
} from '@/api/modules/notification'

const { t } = useI18n()
const router = useRouter()

const loading = ref(false)
const items = ref<NotificationItem[]>([])
const total = ref(0)
const query = reactive<NotificationSearchReq>({
  page: 1,
  pageSize: 12,
  status: 'all',
  type: '',
  source: '',
  info: '',
})

const statusOptions = computed(() => [
  { label: t('commons.all'), value: 'all' },
  { label: t('notification.unread'), value: 'unread' },
  { label: t('notification.read'), value: 'read' },
])

const loadNotifications = async () => {
  loading.value = true
  try {
    const res: any = await searchNotifications(query)
    items.value = res.data?.items || []
    total.value = res.data?.total || 0
  } finally {
    loading.value = false
  }
}

const reloadFirstPage = () => {
  query.page = 1
  loadNotifications()
}

const handleMarkRead = async (row: NotificationItem) => {
  await markNotificationsRead({ ids: [row.id] })
  await loadNotifications()
}

const handleMarkAllRead = async () => {
  await markAllNotificationsRead()
  await loadNotifications()
}

const handleClearRead = async () => {
  await ElMessageBox.confirm(t('notification.clearReadConfirm'), t('commons.tip'), { type: 'warning' })
  await clearReadNotifications()
  await reloadFirstPage()
}

const handleDelete = async (row: NotificationItem) => {
  await deleteNotification({ id: row.id })
  await loadNotifications()
}

const openTarget = async (row: NotificationItem) => {
  if (!row.readAt) {
    await markNotificationsRead({ ids: [row.id] })
  }
  router.push(row.targetUrl)
}

const sourceLabel = (source: string) => {
  const labels: Record<string, string> = {
    file: t('notification.sourceFile'),
    database: t('notification.sourceDatabase'),
    cronjob: t('notification.sourceCronjob'),
    system: t('notification.sourceSystem'),
    security: t('notification.sourceSecurity'),
  }
  return labels[source] || source || '-'
}

const formatTime = (value: string) => {
  if (!value) return '-'
  return new Date(value).toLocaleString('zh-CN', { hour12: false })
}

onMounted(loadNotifications)
</script>

<style lang="scss" scoped>
.notification-page {
  padding: 20px;
  height: 100%;
  overflow: auto;
}

.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;

  h2 {
    margin: 0;
    font-size: 22px;
    color: var(--xp-text-primary);
  }

  p {
    margin: 6px 0 0;
    color: var(--xp-text-secondary);
    font-size: 13px;
  }
}

.header-actions,
.filter-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.filter-row {
  margin-bottom: 14px;
}

.filter-keyword {
  width: 260px;
}

.notification-table {
  width: 100%;
}

.type-dot {
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--el-color-info);

  &.success { background: var(--el-color-success); }
  &.warning { background: var(--el-color-warning); }
  &.error { background: var(--el-color-danger); }
}

.notification-main {
  min-width: 0;
}

.notification-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  color: var(--xp-text-primary);
}

.unread-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--xp-accent);
  flex: 0 0 auto;
}

.notification-content {
  margin-top: 4px;
  color: var(--xp-text-secondary);
  font-size: 12px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.pager-row {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

@media (max-width: 900px) {
  .page-header {
    flex-direction: column;
  }

  .filter-keyword {
    width: 100%;
  }
}
</style>
