<template>
  <el-dialog :model-value="modelValue" @update:model-value="emit('update:modelValue', $event)" :title="`ACL - ${lb?.name || ''}`" width="780px" @closed="emit('closed')">
    <div style="margin-bottom: 12px;">
      <el-button type="primary" size="small" @click="openForm()">
        <el-icon><Plus /></el-icon>{{ $t('haproxy.addACL') }}
      </el-button>
    </div>
    <el-table :data="list" stripe size="small" v-loading="loading">
      <el-table-column prop="priority" :label="$t('haproxy.aclPriority')" width="90" />
      <el-table-column :label="$t('haproxy.aclMatchType')" width="110">
        <template #default="{ row }">
          <el-tag size="small">{{ row.matchType }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="$t('haproxy.aclMatchValue')" min-width="160">
        <template #default="{ row }">
          <span v-if="row.matchHeader">[{{ row.matchHeader }}] </span>
          <code>{{ row.matchValue }}</code>
        </template>
      </el-table-column>
      <el-table-column :label="$t('haproxy.aclTarget')" min-width="130">
        <template #default="{ row }"><el-tag size="small" type="success">{{ row.targetBackend }}</el-tag></template>
      </el-table-column>
      <el-table-column :label="$t('commons.status')" width="80">
        <template #default="{ row }">
          <el-tag size="small" :type="row.enabled ? 'success' : 'info'">
            {{ row.enabled ? $t('commons.enabled') : $t('commons.disabled') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="$t('commons.actions')" width="130" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openForm(row)">{{ $t('commons.edit') }}</el-button>
          <el-button link type="danger" @click="handleDelete(row)">{{ $t('commons.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="formVisible" :title="editing ? $t('haproxy.editACL') : $t('haproxy.addACL')" width="520px" append-to-body destroy-on-close>
      <el-form :model="form" label-width="110px">
        <el-form-item :label="$t('haproxy.aclPriority')">
          <el-input-number v-model="form.priority" :min="1" :max="9999" style="width: 100%;" />
        </el-form-item>
        <el-form-item :label="$t('haproxy.aclMatchType')">
          <el-select v-model="form.matchType" style="width: 100%;">
            <el-option value="host" :label="$t('haproxy.aclMatchHost')" />
            <el-option value="host_end" :label="$t('haproxy.aclMatchHostEnd')" />
            <el-option value="path_beg" :label="$t('haproxy.aclMatchPathBeg')" />
            <el-option value="path_end" :label="$t('haproxy.aclMatchPathEnd')" />
            <el-option value="path_reg" :label="$t('haproxy.aclMatchPathReg')" />
            <el-option value="hdr" :label="$t('haproxy.aclMatchHdr')" />
            <el-option value="src" :label="$t('haproxy.aclMatchSrc')" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.matchType === 'hdr'" :label="$t('haproxy.aclMatchHeader')">
          <el-input v-model="form.matchHeader" placeholder="host / user-agent ..." />
        </el-form-item>
        <el-form-item :label="$t('haproxy.aclMatchValue')">
          <el-input v-model="form.matchValue" />
        </el-form-item>
        <el-form-item :label="$t('haproxy.aclTarget')">
          <el-select v-model="form.targetBackendID" style="width: 100%;">
            <el-option v-for="b in httpBackends" :key="b.id" :label="b.name" :value="b.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('commons.status')">
          <el-switch v-model="form.enabled" />
        </el-form-item>
        <el-form-item :label="$t('haproxy.remark')">
          <el-input v-model="form.remark" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="formVisible = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">{{ $t('commons.save') }}</el-button>
      </template>
    </el-dialog>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { listHAProxyACL, createHAProxyACL, updateHAProxyACL, deleteHAProxyACL } from '@/api/modules/haproxy'

const props = defineProps<{ modelValue: boolean; lb: any; backends: any[] }>()
const emit = defineEmits(['update:modelValue', 'closed'])
const { t } = useI18n()

const loading = ref(false)
const list = ref<any[]>([])
const formVisible = ref(false)
const submitting = ref(false)
const editing = ref(false)
const defaultForm = () => ({
  id: 0, lbID: 0, priority: 100, matchType: 'host',
  matchHeader: '', matchValue: '', targetBackendID: 0,
  enabled: true, remark: '',
})
const form = ref<any>(defaultForm())

const httpBackends = computed(() => props.backends.filter((b) => b.mode === 'http'))

const load = async () => {
  if (!props.lb) return
  loading.value = true
  try {
    const res = await listHAProxyACL(props.lb.id)
    list.value = res.data || []
  } finally {
    loading.value = false
  }
}

watch(() => props.modelValue, (v) => { if (v) load() })

const openForm = (row?: any) => {
  editing.value = !!row
  form.value = row ? { ...row } : { ...defaultForm(), lbID: props.lb.id }
  formVisible.value = true
}

const handleSubmit = async () => {
  submitting.value = true
  try {
    if (editing.value) {
      await updateHAProxyACL(form.value)
    } else {
      await createHAProxyACL(form.value)
    }
    ElMessage.success(t('commons.operationSuccess'))
    formVisible.value = false
    load()
  } finally {
    submitting.value = false
  }
}

const handleDelete = async (row: any) => {
  await ElMessageBox.confirm(t('haproxy.deleteACLConfirm'), t('commons.warning'), { type: 'warning' })
  await deleteHAProxyACL(row.id)
  ElMessage.success(t('commons.deleteSuccess'))
  load()
}
</script>
