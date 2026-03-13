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
        <template #default="{ row }">{{ t('cronjob.type_' + row.type) }}</template>
      </el-table-column>
      <el-table-column prop="spec" label="Cron" width="160" />
      <el-table-column :label="t('cronjob.status')" width="100">
        <template #default="{ row }">
          <el-switch :model-value="row.status === 'Enable'" @change="(v: boolean) => toggleStatus(row, v)" />
        </template>
      </el-table-column>
      <el-table-column :label="t('commons.actions')" width="220" fixed="right">
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

    <!-- Create / Edit dialog -->
    <el-drawer v-model="drawerVisible" :title="editMode ? t('commons.edit') : t('commons.create')" size="520px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item :label="t('commons.name')" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item :label="t('cronjob.type')" prop="type">
          <el-select v-model="form.type" style="width:100%">
            <el-option v-for="tp in typeOptions" :key="tp" :label="t('cronjob.type_' + tp)" :value="tp" />
          </el-select>
        </el-form-item>
        <el-form-item label="Cron" prop="spec">
          <el-input v-model="form.spec" placeholder="*/5 * * * *" />
        </el-form-item>
        <el-form-item v-if="form.type === 'shell'" :label="t('cronjob.script')" prop="script">
          <el-input v-model="form.script" type="textarea" :rows="6" />
        </el-form-item>
        <el-form-item v-if="form.type === 'curl'" label="URL" prop="url">
          <el-input v-model="form.url" />
        </el-form-item>
        <el-form-item v-if="['website','database','directory'].includes(form.type)" :label="t('cronjob.retainCopies')">
          <el-input-number v-model="form.retainCopies" :min="1" :max="999" />
        </el-form-item>
        <el-form-item v-if="form.type === 'website'" :label="t('cronjob.website')">
          <el-input v-model="form.website" />
        </el-form-item>
        <el-form-item v-if="form.type === 'database'" :label="t('cronjob.dbType')">
          <el-select v-model="form.dbType" style="width:100%">
            <el-option label="MySQL" value="mysql" />
            <el-option label="PostgreSQL" value="postgresql" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.type === 'database'" :label="t('cronjob.dbName')">
          <el-input v-model="form.dbName" />
        </el-form-item>
        <el-form-item v-if="form.type === 'directory'" :label="t('cronjob.sourceDir')">
          <el-input v-model="form.sourceDir" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="drawerVisible = false">{{ t('commons.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="submit">{{ t('commons.confirm') }}</el-button>
      </template>
    </el-drawer>

    <!-- Records dialog -->
    <el-drawer v-model="recordDrawer" :title="t('cronjob.records')" size="640px" destroy-on-close>
      <el-table :data="records" v-loading="recordsLoading">
        <el-table-column prop="startTime" :label="t('cronjob.startTime')" width="180">
          <template #default="{ row }">{{ formatTime(row.startTime) }}</template>
        </el-table-column>
        <el-table-column prop="duration" :label="t('cronjob.duration')" width="100">
          <template #default="{ row }">{{ row.duration.toFixed(1) }}s</template>
        </el-table-column>
        <el-table-column prop="status" :label="t('cronjob.status')" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'Success' ? 'success' : 'danger'" size="small">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="message" :label="t('cronjob.message')" min-width="200" show-overflow-tooltip />
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
import {
  searchCronjob, createCronjob, updateCronjob, deleteCronjob,
  updateCronjobStatus, handleOnceCronjob, searchCronjobRecords
} from '@/api/modules/cronjob'

const { t } = useI18n()
const typeOptions = ['shell', 'curl', 'website', 'database', 'directory']

const loading = ref(false)
const data = ref<any[]>([])
const searchType = ref('')
const searchInfo = ref('')
const pager = reactive({ page: 1, pageSize: 20, total: 0 })

const drawerVisible = ref(false)
const editMode = ref(false)
const submitting = ref(false)
const formRef = ref<FormInstance>()

const defaultForm = () => ({
  id: 0, name: '', type: 'shell', spec: '', script: '', url: '',
  website: '', dbType: 'mysql', dbName: '', sourceDir: '',
  targetAccountID: 0, retainCopies: 7, exclusionRules: '',
})
const form = reactive(defaultForm())
const rules: FormRules = {
  name: [{ required: true, message: () => t('cronjob.nameRequired'), trigger: 'blur' }],
  type: [{ required: true }],
  spec: [{ required: true, message: () => t('cronjob.specRequired'), trigger: 'blur' }],
}

const recordDrawer = ref(false)
const recordsLoading = ref(false)
const records = ref<any[]>([])
const recordPager = reactive({ page: 1, pageSize: 20, total: 0 })
let currentRecordCronjobID = 0

const formatTime = (t: string) => t ? new Date(t).toLocaleString() : ''

const search = async () => {
  loading.value = true
  try {
    const res: any = await searchCronjob({
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
  drawerVisible.value = true
}

const openEdit = (row: any) => {
  Object.assign(form, { ...row })
  editMode.value = true
  drawerVisible.value = true
}

const submit = async () => {
  if (!formRef.value) return
  await formRef.value.validate()
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

const handleDelete = async (row: any) => {
  await ElMessageBox.confirm(t('cronjob.deleteConfirm'), t('commons.tip'), { type: 'warning' })
  await deleteCronjob({ id: row.id })
  ElMessage.success(t('commons.success'))
  await search()
}

const toggleStatus = async (row: any, val: boolean) => {
  await updateCronjobStatus({ id: row.id, status: val ? 'Enable' : 'Disable' })
  ElMessage.success(t('commons.success'))
  await search()
}

const handleOnce = async (row: any) => {
  await handleOnceCronjob({ id: row.id })
  ElMessage.success(t('cronjob.triggered'))
}

const openRecords = (row: any) => {
  currentRecordCronjobID = row.id
  recordPager.page = 1
  recordDrawer.value = true
  loadRecords()
}

const loadRecords = async () => {
  recordsLoading.value = true
  try {
    const res: any = await searchCronjobRecords({
      page: recordPager.page, pageSize: recordPager.pageSize,
      cronjobID: currentRecordCronjobID,
    })
    records.value = res.data.items || []
    recordPager.total = res.data.total
  } finally { recordsLoading.value = false }
}

onMounted(() => search())
</script>

<style scoped>
.app-toolbar {
  display: flex;
  align-items: center;
  margin-bottom: 16px;
}
.app-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
