<template>
  <div>
    <div class="page-header">
      <h3>{{ $t('gost.chainTitle') }}</h3>
      <el-button type="primary" @click="openDialog()">
        <el-icon><Plus /></el-icon>{{ $t('gost.createChain') }}
      </el-button>
    </div>

    <el-card shadow="never">
      <el-table :data="tableData" v-loading="loading" stripe>
        <el-table-column prop="name" :label="$t('gost.name')" min-width="150" />
        <el-table-column prop="hopCount" :label="$t('gost.hopCount')" width="120" align="center" />
        <el-table-column prop="refCount" :label="$t('gost.refCount')" width="120" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.refCount > 0" size="small" type="warning">{{ row.refCount }}</el-tag>
            <span v-else class="text-muted">0</span>
          </template>
        </el-table-column>
        <el-table-column prop="remark" :label="$t('gost.remark')" min-width="160" />
        <el-table-column :label="$t('commons.actions')" width="140" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDialog(row)">{{ $t('commons.edit') }}</el-button>
            <el-button link type="danger" @click="handleDelete(row)">{{ $t('commons.delete') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="table-footer" v-if="pagination.total > pagination.pageSize">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          layout="total, prev, pager, next"
          @current-change="search"
        />
      </div>
    </el-card>

    <!-- 创建 / 编辑 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? $t('gost.editChain') : $t('gost.createChain')" width="720px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="80px">
        <el-form-item :label="$t('gost.name')" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item :label="$t('gost.remark')">
          <el-input v-model="form.remark" />
        </el-form-item>

        <el-divider content-position="left">{{ $t('gost.hopNode') }}</el-divider>

        <div v-for="(hop, hIdx) in form.hops" :key="hIdx" class="hop-block">
          <div class="hop-header">
            <span class="hop-label">Hop {{ hIdx + 1 }}</span>
            <el-button link type="danger" size="small" @click="removeHop(hIdx)" :disabled="form.hops.length <= 1">
              {{ $t('gost.removeHop') }}
            </el-button>
          </div>
          <div v-for="(node, nIdx) in hop.nodes" :key="nIdx" class="node-row">
            <el-row :gutter="12">
              <el-col :span="8">
                <el-form-item :label="$t('gost.nodeAddr')" label-width="70px">
                  <el-input v-model="node.addr" :placeholder="$t('gost.nodeAddrHint')" size="small" />
                </el-form-item>
              </el-col>
              <el-col :span="5">
                <el-form-item :label="$t('gost.connectorType')" label-width="70px">
                  <el-select v-model="node.connector.type" size="small" style="width: 100%">
                    <el-option label="Relay" value="relay" />
                    <el-option label="SOCKS5" value="socks5" />
                    <el-option label="HTTP" value="http" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :span="5">
                <el-form-item :label="$t('gost.dialerType')" label-width="70px">
                  <el-select v-model="node.dialer.type" size="small" style="width: 100%">
                    <el-option label="TCP" value="tcp" />
                    <el-option label="TLS" value="tls" />
                    <el-option label="WS" value="ws" />
                    <el-option label="WSS" value="wss" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :span="3">
                <el-form-item :label="$t('gost.nodeAuthUser')" label-width="50px">
                  <el-input v-model="node.authUser" size="small" />
                </el-form-item>
              </el-col>
              <el-col :span="3">
                <el-form-item :label="$t('gost.nodeAuthPass')" label-width="50px">
                  <el-input v-model="node.authPass" type="password" show-password size="small" />
                </el-form-item>
              </el-col>
            </el-row>
          </div>
        </div>
        <el-button type="primary" plain @click="addHop" style="margin-top: 8px;">
          <el-icon><Plus /></el-icon>{{ $t('gost.addHop') }}
        </el-button>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitLoading">{{ $t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { searchGostChain, createGostChain, updateGostChain, deleteGostChain } from '@/api/modules/gost'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const loading = ref(false)
const tableData = ref<any[]>([])
const pagination = reactive({ page: 1, pageSize: 20, total: 0 })
const dialogVisible = ref(false)
const isEdit = ref(false)
const submitLoading = ref(false)
const formRef = ref<FormInstance>()

interface NodeForm {
  name: string
  addr: string
  connector: { type: string }
  dialer: { type: string; auth?: { username: string; password: string } }
  authUser: string
  authPass: string
}

interface HopForm {
  name: string
  nodes: NodeForm[]
}

const createEmptyNode = (): NodeForm => ({
  name: 'node-0',
  addr: '',
  connector: { type: 'relay' },
  dialer: { type: 'wss' },
  authUser: '',
  authPass: '',
})

const createEmptyHop = (idx: number): HopForm => ({
  name: `hop-${idx}`,
  nodes: [createEmptyNode()],
})

const form = reactive({
  id: 0,
  name: '',
  remark: '',
  hops: [createEmptyHop(0)] as HopForm[],
})

const rules = reactive<FormRules>({
  name: [{ required: true, message: t('gost.nameRequired'), trigger: 'blur' }],
})

const search = async () => {
  loading.value = true
  try {
    const res = await searchGostChain({ page: pagination.page, pageSize: pagination.pageSize })
    if (res.data) {
      tableData.value = res.data.items || []
      pagination.total = res.data.total || 0
    }
  } finally {
    loading.value = false
  }
}

const openDialog = (row?: any) => {
  isEdit.value = !!row
  if (row) {
    let hops: HopForm[] = [createEmptyHop(0)]
    try {
      const parsed = JSON.parse(row.hops)
      if (Array.isArray(parsed) && parsed.length > 0) {
        hops = parsed.map((h: any, hIdx: number) => ({
          name: h.name || `hop-${hIdx}`,
          nodes: (h.nodes || []).map((n: any) => ({
            name: n.name || 'node-0',
            addr: n.addr || '',
            connector: { type: n.connector?.type || 'relay' },
            dialer: { type: n.dialer?.type || 'wss' },
            authUser: n.dialer?.auth?.username || '',
            authPass: n.dialer?.auth?.password || '',
          })),
        }))
      }
    } catch { /* use default */ }
    Object.assign(form, { id: row.id, name: row.name, remark: row.remark || '', hops })
  } else {
    Object.assign(form, { id: 0, name: '', remark: '', hops: [createEmptyHop(0)] })
  }
  dialogVisible.value = true
}

const addHop = () => {
  form.hops.push(createEmptyHop(form.hops.length))
}

const removeHop = (idx: number) => {
  form.hops.splice(idx, 1)
}

const buildHopsJSON = (): string => {
  const hops = form.hops.map((h, hIdx) => ({
    name: `hop-${hIdx}`,
    nodes: h.nodes.map((n, nIdx) => {
      const node: any = {
        name: `node-${nIdx}`,
        addr: n.addr,
        connector: { type: n.connector.type },
        dialer: { type: n.dialer.type },
      }
      if (n.authUser) {
        node.dialer.auth = { username: n.authUser, password: n.authPass }
      }
      return node
    }),
  }))
  return JSON.stringify(hops)
}

const handleSubmit = async () => {
  await formRef.value?.validate()
  if (form.hops.length === 0 || form.hops.every(h => h.nodes.every(n => !n.addr))) {
    ElMessage.warning(t('gost.hopsRequired'))
    return
  }
  submitLoading.value = true
  try {
    const payload = { id: form.id, name: form.name, hops: buildHopsJSON(), remark: form.remark }
    if (isEdit.value) {
      await updateGostChain(payload)
    } else {
      await createGostChain(payload)
    }
    ElMessage.success(t('commons.operationSuccess'))
    dialogVisible.value = false
    await search()
  } finally {
    submitLoading.value = false
  }
}

const handleDelete = async (row: any) => {
  if (row.refCount > 0) {
    ElMessage.warning(t('gost.chainInUse'))
    return
  }
  await ElMessageBox.confirm(t('gost.deleteChainConfirm'), t('commons.warning'), { type: 'warning' })
  await deleteGostChain(row.id)
  ElMessage.success(t('commons.deleteSuccess'))
  await search()
}

onMounted(() => search())
</script>

<style lang="scss" scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  h3 { margin: 0; }
}
.text-muted { color: var(--xp-text-muted); }
.table-footer {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
.hop-block {
  background: var(--el-fill-color-lighter);
  border-radius: 8px;
  padding: 12px 16px;
  margin-bottom: 12px;
  .hop-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
    .hop-label {
      font-weight: 600;
      font-size: 13px;
      color: var(--xp-text-secondary);
    }
  }
  .node-row {
    margin-bottom: 4px;
  }
}
</style>
