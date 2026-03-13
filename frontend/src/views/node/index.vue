<template>
  <div>
    <div class="app-toolbar">
      <el-button type="primary" @click="openCreate">{{ t('node.addNode') }}</el-button>
    </div>
    <el-table :data="nodes" v-loading="loading">
      <el-table-column prop="name" :label="t('commons.name')" min-width="140" />
      <el-table-column prop="address" :label="t('node.address')" min-width="180" />
      <el-table-column :label="t('node.status')" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'online' ? 'success' : 'danger'" size="small">{{ row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="os" label="OS" width="120" show-overflow-tooltip />
      <el-table-column prop="hostname" :label="t('node.hostname')" width="140" show-overflow-tooltip />
      <el-table-column :label="t('commons.actions')" width="220" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="testConn(row)">{{ t('node.testConn') }}</el-button>
          <el-button link type="primary" @click="openEdit(row)">{{ t('commons.edit') }}</el-button>
          <el-button link type="danger" @click="handleDelete(row)">{{ t('commons.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-drawer v-model="drawerVisible" :title="editMode ? t('commons.edit') : t('node.addNode')" size="480px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item :label="t('commons.name')" prop="name"><el-input v-model="form.name" /></el-form-item>
        <el-form-item :label="t('node.address')" prop="address"><el-input v-model="form.address" placeholder="192.168.1.100:8080" /></el-form-item>
        <el-form-item :label="t('node.token')" prop="token"><el-input v-model="form.token" type="password" show-password /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="drawerVisible = false">{{ t('commons.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="submit">{{ t('commons.confirm') }}</el-button>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { listNodes, createNode, updateNode, deleteNode, testNodeConnection } from '@/api/modules/node'

const { t } = useI18n()
const loading = ref(false)
const nodes = ref<any[]>([])

const drawerVisible = ref(false)
const editMode = ref(false)
const submitting = ref(false)
const formRef = ref<FormInstance>()
const defaultForm = () => ({ id: 0, name: '', address: '', token: '', groupID: 0 })
const form = reactive(defaultForm())
const rules: FormRules = {
  name: [{ required: true, trigger: 'blur' }],
  address: [{ required: true, trigger: 'blur' }],
  token: [{ required: true, trigger: 'blur' }],
}

const load = async () => {
  loading.value = true
  try {
    const res: any = await listNodes()
    nodes.value = res.data || []
  } finally { loading.value = false }
}

const openCreate = () => {
  Object.assign(form, defaultForm())
  editMode.value = false
  drawerVisible.value = true
}

const openEdit = (row: any) => {
  Object.assign(form, { ...row, token: '' })
  editMode.value = true
  drawerVisible.value = true
}

const submit = async () => {
  if (!formRef.value) return
  await formRef.value.validate()
  submitting.value = true
  try {
    if (editMode.value) await updateNode(form)
    else await createNode(form)
    ElMessage.success(t('commons.success'))
    drawerVisible.value = false
    await load()
  } finally { submitting.value = false }
}

const handleDelete = async (row: any) => {
  await ElMessageBox.confirm(t('node.deleteConfirm'), t('commons.tip'), { type: 'warning' })
  await deleteNode({ id: row.id })
  ElMessage.success(t('commons.success'))
  await load()
}

const testConn = async (row: any) => {
  try {
    await testNodeConnection({ id: row.id })
    ElMessage.success(t('node.testSuccess'))
  } catch {
    ElMessage.error(t('node.testFail'))
  }
}

onMounted(() => load())
</script>

<style scoped>
.app-toolbar { display: flex; align-items: center; margin-bottom: 16px; }
</style>
