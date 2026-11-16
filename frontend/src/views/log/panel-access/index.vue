<template>
  <div class="xp-page-shell">
    <form class="app-toolbar" @submit.prevent="search">
      <el-input
        v-model="ip"
        :placeholder="t('panelAccess.ipPlaceholder')"
        :aria-label="t('panelAccess.ipPlaceholder')"
        clearable
        class="ip-filter"
        @clear="search"
      />
      <el-select v-model="kind" :aria-label="t('panelAccess.event')" class="event-filter" @change="search">
        <el-option value="" :label="t('panelAccess.allEvents')" />
        <el-option v-for="event in events" :key="event" :value="event" :label="t('panelAccess.' + event)" />
      </el-select>
      <el-button native-type="submit">{{ t('commons.search') }}</el-button>
      <span class="toolbar-spacer" />
      <el-button size="small" :loading="loading" @click="load">{{ t('commons.refresh') }}</el-button>
    </form>
    <p class="retention">{{ t('panelAccess.hint') }}</p>
    <el-table :data="rows" v-loading="loading" :empty-text="t('panelAccess.empty')">
      <el-table-column prop="ip" :label="t('log.ip')" min-width="170" show-overflow-tooltip />
      <el-table-column :label="t('panelAccess.event')" min-width="150">
        <template #default="{ row }">
          <el-tag :type="eventTypes[row.kind as PanelAccessKind]" size="small">{{ t('panelAccess.' + row.kind) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="hits" :label="t('panelAccess.hits')" width="90" align="right" />
      <el-table-column :label="t('panelAccess.firstSeen')" min-width="165">
        <template #default="{ row }">{{ formatTime(row.firstSeen) }}</template>
      </el-table-column>
      <el-table-column :label="t('panelAccess.lastSeen')" min-width="165">
        <template #default="{ row }">{{ formatTime(row.lastSeen) }}</template>
      </el-table-column>
    </el-table>
    <div class="table-footer">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next"
        @change="load"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getPanelAccessLogs, type PanelAccessKind, type PanelAccessLog } from '@/api/modules/log'

const { t } = useI18n()
const events: PanelAccessKind[] = ['entrance_blocked', 'entrance_matched', 'page_opened', 'login_failed', 'captcha_blocked', 'login_succeeded']
const eventTypes: Record<PanelAccessKind, 'info' | 'warning' | 'danger' | 'success'> = {
  entrance_blocked: 'info',
  entrance_matched: 'warning',
  page_opened: 'info',
  login_failed: 'danger',
  captcha_blocked: 'warning',
  login_succeeded: 'success',
}
const rows = ref<PanelAccessLog[]>([])
const loading = ref(false)
const ip = ref('')
const kind = ref<PanelAccessKind | ''>('')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
let request = 0

const formatter = new Intl.DateTimeFormat(undefined, { dateStyle: 'short', timeStyle: 'medium' })
const formatTime = (value: string) => formatter.format(new Date(value))

async function load() {
  const current = ++request
  loading.value = true
  try {
    const { data } = await getPanelAccessLogs({
      page: page.value, pageSize: pageSize.value, ip: ip.value.trim(), kind: kind.value || undefined,
    })
    if (current !== request) return
    rows.value = data?.items || []
    total.value = data?.total || 0
  } catch { /* The HTTP client displays the query error. */ }
  finally {
    if (current === request) loading.value = false
  }
}

function search() {
  if (page.value === 1) void load()
  else page.value = 1
}

onMounted(load)
</script>

<style scoped>
.ip-filter {
  width: 15rem;
}

.event-filter {
  width: 11rem;
}

.retention {
  margin: 0 0 0.75rem;
  color: var(--xp-text-secondary);
  font-size: 0.8125rem;
}
</style>
