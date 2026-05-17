<template>
  <div class="notification-page">
    <div class="page-header">
      <div>
        <h2>{{ t('notification.title') }}</h2>
        <p>{{ t('notification.subtitle') }}</p>
      </div>
      <div class="header-actions">
        <el-button :icon="Refresh" @click="loadNotifications">{{ t('commons.refresh') }}</el-button>
        <el-button :icon="Setting" @click="openPreference">{{ t('notification.preference') }}</el-button>
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
      <el-select v-model="query.source" clearable style="width: 140px" :placeholder="t('notification.source')">
        <el-option :label="t('commons.all')" value="" />
        <el-option v-for="item in sourceOptions" :key="item.value" :label="item.label" :value="item.value" />
      </el-select>
      <el-select v-model="query.event" clearable style="width: 190px" :placeholder="t('notification.event')">
        <el-option :label="t('commons.all')" value="" />
        <el-option v-for="item in eventOptions" :key="item.value" :label="item.label" :value="item.value" />
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
      <el-table-column prop="event" :label="t('notification.event')" width="160">
        <template #default="{ row }">
          <span class="event-label">{{ eventLabel(row.event) }}</span>
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

    <el-drawer v-model="preferenceVisible" :title="t('notification.preference')" size="520px">
      <div class="preference-section">
        <div class="preference-title">{{ t('notification.defaultRule') }}</div>
        <div class="preference-row">
          <span>{{ t('notification.writeCenter') }}</span>
          <el-switch v-model="preference.defaults.center" />
        </div>
        <div class="preference-row">
          <span>{{ t('notification.showBadge') }}</span>
          <el-switch v-model="preference.defaults.badge" />
        </div>
        <div class="preference-row">
          <span>{{ t('notification.showPopup') }}</span>
          <el-switch v-model="preference.defaults.popup" />
        </div>
      </div>

      <div class="preference-section">
        <div class="preference-title">{{ t('notification.eventRule') }}</div>
        <div v-for="item in eventOptions" :key="item.value" class="event-rule">
          <div class="event-rule-head">
            <strong>{{ item.label }}</strong>
            <code>{{ item.value }}</code>
          </div>
          <div class="event-rule-controls">
            <el-checkbox v-model="preference.events[item.value].center">{{ t('notification.writeCenter') }}</el-checkbox>
            <el-checkbox v-model="preference.events[item.value].badge">{{ t('notification.showBadge') }}</el-checkbox>
            <el-checkbox v-model="preference.events[item.value].popup">{{ t('notification.showPopup') }}</el-checkbox>
          </div>
        </div>
      </div>

      <template #footer>
        <el-button @click="preferenceVisible = false">{{ t('commons.cancel') }}</el-button>
        <el-button type="primary" :loading="preferenceSaving" @click="savePreference">{{ t('commons.save') }}</el-button>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CircleCheck, Delete, Refresh, Setting } from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'
import type { NotificationItem, NotificationPreference, NotificationSearchReq } from '@/api/interface'
import {
  clearReadNotifications,
  deleteNotification,
  getNotificationPreference,
  markAllNotificationsRead,
  markNotificationsRead,
  searchNotifications,
  updateNotificationPreference,
} from '@/api/modules/notification'

const { t } = useI18n()
const router = useRouter()

const loading = ref(false)
const preferenceVisible = ref(false)
const preferenceSaving = ref(false)
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

const preference = reactive<NotificationPreference>({
  defaults: { center: true, badge: true, popup: false },
  events: {},
})

const statusOptions = computed(() => [
  { label: t('commons.all'), value: 'all' },
  { label: t('notification.unread'), value: 'unread' },
  { label: t('notification.read'), value: 'read' },
])

const sourceOptions = computed(() => [
  { label: t('notification.sourceFile'), value: 'file' },
  { label: t('notification.sourceDatabase'), value: 'database' },
  { label: t('notification.sourceCronjob'), value: 'cronjob' },
  { label: t('notification.sourceSystem'), value: 'system' },
  { label: t('notification.sourceSecurity'), value: 'security' },
])

const eventOptions = computed(() => [
  { label: t('notification.eventFileUpload'), value: 'file.upload.completed' },
  { label: t('notification.eventFileTaskFailed'), value: 'file.task.failed' },
  { label: t('notification.eventDatabaseTaskFailed'), value: 'database.task.failed' },
  { label: t('notification.eventCronjobFailed'), value: 'cronjob.failed' },
  { label: t('notification.eventOperationFailed'), value: 'operation.failed' },
  { label: t('notification.eventSystemLogError'), value: 'system.log.error' },
])

const ensurePreferenceEvents = () => {
  eventOptions.value.forEach((item) => {
    if (!preference.events[item.value]) {
      preference.events[item.value] = { ...preference.defaults }
    }
  })
}

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

const openPreference = async () => {
  const res: any = await getNotificationPreference()
  Object.assign(preference.defaults, res.data?.defaults || { center: true, badge: true, popup: false })
  preference.events = { ...(res.data?.events || {}) }
  ensurePreferenceEvents()
  preferenceVisible.value = true
}

const savePreference = async () => {
  preferenceSaving.value = true
  try {
    ensurePreferenceEvents()
    await updateNotificationPreference(JSON.parse(JSON.stringify(preference)))
    ElMessage.success(t('commons.saveSuccess'))
    preferenceVisible.value = false
    await loadNotifications()
  } finally {
    preferenceSaving.value = false
  }
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

const eventLabel = (event: string) => {
  const found = eventOptions.value.find((item) => item.value === event)
  return found?.label || event || '-'
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

.event-label {
  color: var(--xp-text-secondary);
  font-size: 12px;
}

.pager-row {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.preference-section {
  margin-bottom: 22px;
}

.preference-title {
  margin-bottom: 10px;
  font-weight: 700;
  color: var(--xp-text-primary);
}

.preference-row,
.event-rule-controls {
  display: flex;
  align-items: center;
  gap: 14px;
}

.preference-row {
  justify-content: space-between;
  padding: 9px 0;
  border-bottom: 1px solid var(--xp-border-light);
}

.event-rule {
  padding: 12px 0;
  border-bottom: 1px solid var(--xp-border-light);
}

.event-rule-head {
  display: flex;
  flex-direction: column;
  gap: 3px;
  margin-bottom: 8px;

  code {
    color: var(--xp-text-muted);
    font-size: 12px;
  }
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
