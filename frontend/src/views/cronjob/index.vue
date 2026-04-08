<template>
  <div>
    <div class="app-toolbar">
      <el-button type="primary" @click="openCreate">{{ t('commons.create') }}</el-button>
      <div style="flex:1" />
      <el-select v-model="searchType" :placeholder="t('cronjob.type')" clearable style="width:140px;margin-right:10px" @change="search">
        <el-option v-for="tp in typeOptions" :key="tp" :label="t('cronjob.type_' + tp)" :value="tp" />
      </el-select>
      <el-input v-model="searchInfo" :placeholder="t('commons.search')" style="width:200px" clearable @clear="search" @keyup.enter="search" />
    </div>
    <el-table :data="data" v-loading="loading" style="width:100%">
      <el-table-column prop="name" :label="t('commons.name')" min-width="140" />
      <el-table-column prop="type" :label="t('cronjob.type')" width="120">
        <template #default="{ row }">
          <el-tag size="small" :type="typeTagMap[row.type] || 'info'">{{ t('cronjob.type_' + row.type) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="t('cronjob.schedule')" width="200">
        <template #default="{ row }">
          <span class="cron-desc">{{ describeCron(row.spec) }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="t('cronjob.status')" width="100">
        <template #default="{ row }">
          <el-switch :model-value="row.status === 'Enable'" @change="(v: boolean) => toggleStatus(row, v)" />
        </template>
      </el-table-column>
      <el-table-column :label="t('commons.actions')" width="240" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openRecords(row)">{{ t('cronjob.records') }}</el-button>
          <el-button link type="primary" @click="handleOnce(row)">{{ t('cronjob.runOnce') }}</el-button>
          <el-button link type="primary" @click="openEdit(row)">{{ t('commons.edit') }}</el-button>
          <el-button link type="danger" @click="handleDelete(row)">{{ t('commons.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>
    <div class="app-pagination">
      <el-pagination
        v-model:current-page="pager.page" v-model:page-size="pager.pageSize"
        :total="pager.total" :page-sizes="[20,50,100]" layout="total, sizes, prev, pager, next"
        @size-change="search" @current-change="search"
      />
    </div>

    <!-- Create / Edit drawer -->
    <el-drawer v-model="drawerVisible" :title="editMode ? t('commons.edit') : t('commons.create')" size="560px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="110px">
        <el-form-item :label="t('commons.name')" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item :label="t('cronjob.type')" prop="type">
          <el-select v-model="form.type" style="width:100%">
            <el-option v-for="tp in typeOptions" :key="tp" :label="t('cronjob.type_' + tp)" :value="tp" />
          </el-select>
        </el-form-item>

        <!-- Visual cron scheduler -->
        <el-form-item :label="t('cronjob.schedule')" required>
          <div class="cron-builder">
            <div class="cron-row">
              <el-select v-model="cronMode" style="width:160px" @change="buildCron">
                <el-option :label="t('cronjob.perMinute')" value="perMinute" />
                <el-option :label="t('cronjob.perHour')" value="perHour" />
                <el-option :label="t('cronjob.perDay')" value="perDay" />
                <el-option :label="t('cronjob.perWeek')" value="perWeek" />
                <el-option :label="t('cronjob.perMonth')" value="perMonth" />
                <el-option :label="t('cronjob.perNMinute')" value="perNMinute" />
                <el-option :label="t('cronjob.perNHour')" value="perNHour" />
                <el-option :label="t('cronjob.custom')" value="custom" />
              </el-select>
            </div>

            <div v-if="cronMode === 'perNMinute'" class="cron-row">
              <span>{{ t('cronjob.every') }}</span>
              <el-input-number v-model="cronEveryN" :min="1" :max="59" size="small" style="width:100px" @change="buildCron" />
              <span>{{ t('cronjob.minutes') }}</span>
            </div>

            <div v-if="cronMode === 'perNHour'" class="cron-row">
              <span>{{ t('cronjob.every') }}</span>
              <el-input-number v-model="cronEveryN" :min="1" :max="23" size="small" style="width:100px" @change="buildCron" />
              <span>{{ t('cronjob.hours') }}</span>
              <span>{{ t('cronjob.atMinute') }}</span>
              <el-input-number v-model="cronMinute" :min="0" :max="59" size="small" style="width:80px" @change="buildCron" />
            </div>

            <div v-if="cronMode === 'perHour'" class="cron-row">
              <span>{{ t('cronjob.atMinute') }}</span>
              <el-input-number v-model="cronMinute" :min="0" :max="59" size="small" style="width:80px" @change="buildCron" />
            </div>

            <div v-if="cronMode === 'perDay'" class="cron-row">
              <el-time-picker v-model="cronTime" format="HH:mm" :placeholder="t('cronjob.selectTime')" style="width:140px" @change="buildCron" />
            </div>

            <div v-if="cronMode === 'perWeek'" class="cron-row">
              <el-select v-model="cronWeekday" style="width:120px" @change="buildCron">
                <el-option v-for="d in 7" :key="d-1" :label="t('cronjob.weekday' + (d-1))" :value="d-1" />
              </el-select>
              <el-time-picker v-model="cronTime" format="HH:mm" :placeholder="t('cronjob.selectTime')" style="width:140px" @change="buildCron" />
            </div>

            <div v-if="cronMode === 'perMonth'" class="cron-row">
              <span>{{ t('cronjob.dayOfMonth') }}</span>
              <el-input-number v-model="cronDayOfMonth" :min="1" :max="31" size="small" style="width:80px" @change="buildCron" />
              <el-time-picker v-model="cronTime" format="HH:mm" :placeholder="t('cronjob.selectTime')" style="width:140px" @change="buildCron" />
            </div>

            <div v-if="cronMode === 'custom'" class="cron-row">
              <el-input v-model="form.spec" placeholder="*/5 * * * *" style="width:100%" />
            </div>

            <div class="cron-preview">
              <el-tag type="info" effect="plain" size="small">{{ form.spec || '...' }}</el-tag>
              <span class="cron-preview-desc">{{ describeCron(form.spec) }}</span>
            </div>
          </div>
        </el-form-item>

        <el-form-item v-if="form.type === 'shell'" :label="t('cronjob.script')" prop="script">
          <el-input v-model="form.script" type="textarea" :rows="6" placeholder="#!/bin/bash" />
        </el-form-item>
        <el-form-item v-if="form.type === 'curl'" label="URL" prop="url">
          <el-input v-model="form.url" placeholder="https://example.com/api/ping" />
        </el-form-item>
        <el-form-item v-if="form.type === 'website'" :label="t('cronjob.website')">
          <el-input v-model="form.website" :placeholder="t('cronjob.websitePlaceholder')" />
        </el-form-item>
        <el-form-item v-if="form.type === 'database'" :label="t('cronjob.dbType')">
          <el-select v-model="form.dbType" style="width:100%">
            <el-option label="MySQL" value="mysql" />
            <el-option label="PostgreSQL" value="postgresql" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.type === 'database'" :label="t('cronjob.dbName')">
          <el-input v-model="form.dbName" placeholder="my_database" />
        </el-form-item>
        <el-form-item v-if="form.type === 'directory'" :label="t('cronjob.sourceDir')">
          <el-input v-model="form.sourceDir" placeholder="/data/myapp" />
        </el-form-item>
        <el-form-item v-if="['website','database','directory'].includes(form.type)" :label="t('cronjob.retainCopies')">
          <el-input-number v-model="form.retainCopies" :min="1" :max="999" />
        </el-form-item>
        <el-form-item v-if="['website','directory'].includes(form.type)" :label="t('cronjob.compressFormat')">
          <el-select v-model="form.compressFormat" style="width:100%">
            <el-option label="Gzip (.tar.gz)" value="gzip" />
            <el-option label="Zstd (.tar.zst)" value="zstd" />
            <el-option label="XZ (.tar.xz)" value="xz" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="['website','database','directory'].includes(form.type)" :label="t('cronjob.encryptPassword')">
          <el-input v-model="form.encryptPassword" type="password" show-password :placeholder="t('cronjob.encryptPasswordHint')" />
        </el-form-item>
        <el-form-item v-if="['website','directory'].includes(form.type)" :label="t('cronjob.exclusionRules')">
          <el-input v-model="form.exclusionRules" type="textarea" :rows="3" :placeholder="t('cronjob.exclusionRulesHint')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="drawerVisible = false">{{ t('commons.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="submit">{{ t('commons.confirm') }}</el-button>
      </template>
    </el-drawer>

    <!-- Records drawer -->
    <el-drawer v-model="recordDrawer" :title="recordTitle" size="700px" destroy-on-close>
      <el-table :data="records" v-loading="recordsLoading">
        <el-table-column prop="startTime" :label="t('cronjob.startTime')" width="180">
          <template #default="{ row }">{{ formatTime(row.startTime) }}</template>
        </el-table-column>
        <el-table-column prop="duration" :label="t('cronjob.duration')" width="100">
          <template #default="{ row }">{{ row.duration.toFixed(1) }}s</template>
        </el-table-column>
        <el-table-column prop="status" :label="t('cronjob.status')" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'Success' ? 'success' : 'danger'" size="small">{{ row.status === 'Success' ? t('cronjob.success') : t('cronjob.failed') }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="message" :label="t('cronjob.message')" min-width="240" show-overflow-tooltip />
      </el-table>
      <div class="app-pagination">
        <el-pagination
          v-model:current-page="recordPager.page" v-model:page-size="recordPager.pageSize"
          :total="recordPager.total" layout="total, prev, pager, next"
          @current-change="loadRecords"
        />
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { useI18n } from 'vue-i18n'
import type { Cronjob, CronjobRecord } from '@/api/interface'
import {
  searchCronjob, createCronjob, updateCronjob, deleteCronjob,
  updateCronjobStatus, handleOnceCronjob, searchCronjobRecords
} from '@/api/modules/cronjob'

const { t } = useI18n()
const typeOptions = ['shell', 'curl', 'website', 'database', 'directory']
const typeTagMap: Record<string, string> = {
  shell: '', curl: 'warning', website: 'success', database: 'danger', directory: 'info',
}

const loading = ref(false)
const data = ref<Cronjob[]>([])
const searchType = ref('')
const searchInfo = ref('')
const pager = reactive({ page: 1, pageSize: 20, total: 0 })

const drawerVisible = ref(false)
const editMode = ref(false)
const submitting = ref(false)
const formRef = ref<FormInstance>()

const defaultForm = () => ({
  id: 0, name: '', type: 'shell', spec: '0 2 * * *', script: '', url: '',
  website: '', dbType: 'mysql', dbName: '', sourceDir: '',
  targetAccountID: 0, retainCopies: 7, exclusionRules: '',
  compressFormat: 'gzip', encryptPassword: '',
})
const form = reactive(defaultForm())
const rules: FormRules = {
  name: [{ required: true, message: () => t('cronjob.nameRequired'), trigger: 'blur' }],
  type: [{ required: true }],
}

// Cron builder state
const cronMode = ref('perDay')
const cronMinute = ref(0)
const cronEveryN = ref(5)
const cronWeekday = ref(1)
const cronDayOfMonth = ref(1)
const cronTime = ref<Date | null>(null)

const initCronTime = (h: number, m: number) => {
  const d = new Date()
  d.setHours(h, m, 0, 0)
  return d
}

const buildCron = () => {
  const h = cronTime.value ? cronTime.value.getHours() : 2
  const m = cronTime.value ? cronTime.value.getMinutes() : 0
  switch (cronMode.value) {
    case 'perMinute':
      form.spec = '* * * * *'; break
    case 'perNMinute':
      form.spec = `*/${cronEveryN.value} * * * *`; break
    case 'perHour':
      form.spec = `${cronMinute.value} * * * *`; break
    case 'perNHour':
      form.spec = `${cronMinute.value} */${cronEveryN.value} * * *`; break
    case 'perDay':
      form.spec = `${m} ${h} * * *`; break
    case 'perWeek':
      form.spec = `${m} ${h} * * ${cronWeekday.value}`; break
    case 'perMonth':
      form.spec = `${m} ${h} ${cronDayOfMonth.value} * *`; break
    case 'custom':
      break
  }
}

const parseCronToBuilder = (spec: string) => {
  if (!spec) { cronMode.value = 'perDay'; cronTime.value = initCronTime(2, 0); buildCron(); return }
  const parts = spec.trim().split(/\s+/)
  if (parts.length !== 5) { cronMode.value = 'custom'; return }
  const [min, hour, dom, , dow] = parts

  if (min === '*' && hour === '*' && dom === '*' && dow === '*') {
    cronMode.value = 'perMinute'; return
  }
  if (min.startsWith('*/') && hour === '*' && dom === '*' && dow === '*') {
    cronMode.value = 'perNMinute'; cronEveryN.value = parseInt(min.slice(2)) || 5; return
  }
  if (/^\d+$/.test(min) && hour === '*' && dom === '*' && dow === '*') {
    cronMode.value = 'perHour'; cronMinute.value = parseInt(min); return
  }
  if (/^\d+$/.test(min) && hour.startsWith('*/') && dom === '*' && dow === '*') {
    cronMode.value = 'perNHour'; cronMinute.value = parseInt(min); cronEveryN.value = parseInt(hour.slice(2)) || 1; return
  }
  if (/^\d+$/.test(min) && /^\d+$/.test(hour) && dom === '*' && dow === '*') {
    cronMode.value = 'perDay'; cronTime.value = initCronTime(parseInt(hour), parseInt(min)); return
  }
  if (/^\d+$/.test(min) && /^\d+$/.test(hour) && dom === '*' && /^\d+$/.test(dow)) {
    cronMode.value = 'perWeek'; cronTime.value = initCronTime(parseInt(hour), parseInt(min)); cronWeekday.value = parseInt(dow); return
  }
  if (/^\d+$/.test(min) && /^\d+$/.test(hour) && /^\d+$/.test(dom) && dow === '*') {
    cronMode.value = 'perMonth'; cronTime.value = initCronTime(parseInt(hour), parseInt(min)); cronDayOfMonth.value = parseInt(dom); return
  }
  cronMode.value = 'custom'
}

const describeCron = (spec: string): string => {
  if (!spec) return ''
  const parts = spec.trim().split(/\s+/)
  if (parts.length !== 5) return spec
  const [min, hour, dom, , dow] = parts
  if (min === '*' && hour === '*') return t('cronjob.descEveryMinute')
  if (min.startsWith('*/') && hour === '*') return t('cronjob.descEveryNMin', { n: min.slice(2) })
  if (/^\d+$/.test(min) && hour === '*') return t('cronjob.descHourlyAt', { m: min })
  if (/^\d+$/.test(min) && hour.startsWith('*/')) return t('cronjob.descEveryNHour', { n: hour.slice(2), m: min })
  if (/^\d+$/.test(min) && /^\d+$/.test(hour) && dom === '*' && dow === '*')
    return t('cronjob.descDailyAt', { h: hour.padStart(2, '0'), m: min.padStart(2, '0') })
  if (/^\d+$/.test(min) && /^\d+$/.test(hour) && dom === '*' && /^\d+$/.test(dow))
    return t('cronjob.descWeeklyAt', { day: t('cronjob.weekday' + dow), h: hour.padStart(2, '0'), m: min.padStart(2, '0') })
  if (/^\d+$/.test(min) && /^\d+$/.test(hour) && /^\d+$/.test(dom))
    return t('cronjob.descMonthlyAt', { d: dom, h: hour.padStart(2, '0'), m: min.padStart(2, '0') })
  return spec
}

// Records
const recordDrawer = ref(false)
const recordTitle = ref('')
const recordsLoading = ref(false)
const records = ref<CronjobRecord[]>([])
const recordPager = reactive({ page: 1, pageSize: 20, total: 0 })
let currentRecordCronjobID = 0

const formatTime = (t: string) => t ? new Date(t).toLocaleString() : ''

const search = async () => {
  loading.value = true
  try {
    const res = await searchCronjob({
      page: pager.page, pageSize: pager.pageSize,
      type: searchType.value, info: searchInfo.value,
    })
    data.value = res.data.items || []
    pager.total = res.data.total
  } finally { loading.value = false }
}

const openCreate = () => {
  Object.assign(form, defaultForm())
  editMode.value = false
  cronTime.value = initCronTime(2, 0)
  parseCronToBuilder(form.spec)
  drawerVisible.value = true
}

const openEdit = (row: Cronjob) => {
  Object.assign(form, { ...row })
  editMode.value = true
  parseCronToBuilder(row.spec)
  drawerVisible.value = true
}

const submit = async () => {
  if (!formRef.value) return
  await formRef.value.validate()
  if (!form.spec || !form.spec.trim()) {
    ElMessage.warning(t('cronjob.specRequired'))
    return
  }
  submitting.value = true
  try {
    if (editMode.value) {
      await updateCronjob(form)
    } else {
      await createCronjob(form)
    }
    ElMessage.success(t('commons.success'))
    drawerVisible.value = false
    await search()
  } finally { submitting.value = false }
}

const handleDelete = async (row: Cronjob) => {
  await ElMessageBox.confirm(t('cronjob.deleteConfirm'), t('commons.tip'), { type: 'warning' })
  await deleteCronjob({ id: row.id })
  ElMessage.success(t('commons.success'))
  await search()
}

const toggleStatus = async (row: Cronjob, val: boolean) => {
  await updateCronjobStatus({ id: row.id, status: val ? 'Enable' : 'Disable' })
  ElMessage.success(t('commons.success'))
  await search()
}

const handleOnce = async (row: Cronjob) => {
  await handleOnceCronjob({ id: row.id })
  ElMessage.success(t('cronjob.triggered'))
}

const openRecords = (row: Cronjob) => {
  currentRecordCronjobID = row.id
  recordTitle.value = row.name + ' - ' + t('cronjob.records')
  recordPager.page = 1
  recordDrawer.value = true
  loadRecords()
}

const loadRecords = async () => {
  recordsLoading.value = true
  try {
    const res = await searchCronjobRecords({
      page: recordPager.page, pageSize: recordPager.pageSize,
      cronjobID: currentRecordCronjobID,
    })
    records.value = res.data.items || []
    recordPager.total = res.data.total
  } finally { recordsLoading.value = false }
}

onMounted(() => search())
</script>

<style lang="scss" scoped>
.cron-builder {
  width: 100%;
}
.cron-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  flex-wrap: wrap;
  font-size: 13px;
  color: var(--xp-text-secondary);
}
.cron-preview {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  background: var(--xp-bg-inset);
  border-radius: var(--xp-radius-sm);
  border: 1px solid var(--xp-border-light);
}
.cron-preview-desc {
  color: var(--xp-text-muted);
  font-size: 12px;
}
.cron-desc {
  color: var(--xp-text-secondary);
  font-size: 13px;
  font-family: var(--xp-font-mono);
}
</style>
