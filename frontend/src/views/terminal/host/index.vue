<template>
  <div class="host-manage">
    <div class="manage-header">
      <el-button link type="primary" @click="emit('back')">
        <el-icon><ArrowLeft /></el-icon>
        {{ $t('terminal.title') }}
      </el-button>
      <h3>{{ $t('host.title') }}</h3>
      <div class="header-actions">
        <el-button size="small" type="primary" @click="openDialog()">
          <el-icon><Plus /></el-icon>
          {{ $t('host.addHost') }}
        </el-button>
      </div>
    </div>

    <div class="manage-body">
      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-input
          v-model="searchInfo"
          :placeholder="$t('commons.search')"
          prefix-icon="Search"
          size="small"
          clearable
          @input="handleSearch"
          class="search-input"
        />
        <el-select v-model="searchGroupID" :placeholder="$t('commons.group')" size="small" clearable @change="loadData">
          <el-option :label="'全部'" :value="0" />
          <el-option v-for="g in groups" :key="g.id" :label="g.name" :value="g.id" />
        </el-select>
        <el-button size="small" type="info" plain @click="groupDialogVisible = true">
          <el-icon><Folder /></el-icon>
          {{ $t('group.title') }}
        </el-button>
      </div>

      <!-- 主机列表 -->
      <el-table :data="hosts" style="width: 100%">
        <el-table-column prop="name" :label="$t('host.name')" min-width="120" />
        <el-table-column prop="addr" :label="$t('host.addr')" min-width="130">
          <template #default="{ row }">
            {{ row.addr }}:{{ row.port }}
          </template>
        </el-table-column>
        <el-table-column prop="user" :label="$t('host.user')" width="100" />
        <el-table-column prop="authMode" :label="$t('host.authMode')" width="100">
          <template #default="{ row }">
            <el-tag :type="row.authMode === 'password' ? 'info' : 'warning'" size="small">
              {{ row.authMode === 'password' ? $t('host.authPassword') : $t('host.authKey') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="groupName" :label="$t('host.group')" width="100">
          <template #default="{ row }">
            {{ row.groupName || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="description" :label="$t('host.description')" min-width="120" show-overflow-tooltip />
        <el-table-column :label="$t('commons.actions')" width="200" fixed="right">
          <template #default="{ row }">
            <el-button link type="success" size="small" @click="emit('connect', row.id, row.name + ' (' + row.addr + ')')">
              {{ $t('host.connect') }}
            </el-button>
            <el-button link type="primary" size="small" @click="openDialog(row)">
              {{ $t('commons.edit') }}
            </el-button>
            <el-button link type="info" size="small" @click="handleTest(row.id)">
              {{ $t('host.testConn') }}
            </el-button>
            <el-button link type="danger" size="small" @click="handleDelete(row)">
              {{ $t('commons.delete') }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-if="total > 0"
        class="mt-pagination"
        :current-page="page"
        :page-size="pageSize"
        :total="total"
        layout="total, prev, pager, next"
        @current-change="handlePageChange"
      />
    </div>

    <!-- 新增/编辑主机对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="editMode ? $t('host.editHost') : $t('host.addHost')"
      width="520px"
      destroy-on-close
    >
      <el-form :model="form" label-width="90px" size="default">
        <el-form-item :label="$t('host.name')">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item :label="$t('host.addr')">
          <div class="addr-row">
            <el-input v-model="form.addr" placeholder="192.168.1.1" class="addr-input" />
            <el-input-number v-model="form.port" :min="1" :max="65535" controls-position="right" class="port-input" />
          </div>
        </el-form-item>
        <el-form-item :label="$t('host.user')">
          <el-input v-model="form.user" placeholder="root" />
        </el-form-item>
        <el-form-item :label="$t('host.group')">
          <el-select v-model="form.groupID" :placeholder="$t('commons.group')" clearable style="width: 100%">
            <el-option :label="'无分组'" :value="0" />
            <el-option v-for="g in groups" :key="g.id" :label="g.name" :value="g.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('host.authMode')">
          <el-radio-group v-model="form.authMode">
            <el-radio value="password">{{ $t('host.authPassword') }}</el-radio>
            <el-radio value="key">{{ $t('host.authKey') }}</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="form.authMode === 'password'" :label="$t('host.password')">
          <el-input v-model="form.password" type="password" show-password />
        </el-form-item>
        <el-form-item v-if="form.authMode === 'key'" :label="$t('host.privateKey')">
          <el-input v-model="form.privateKey" type="textarea" :rows="4" placeholder="-----BEGIN RSA PRIVATE KEY-----" />
        </el-form-item>
        <el-form-item v-if="form.authMode === 'key'" :label="$t('host.passPhrase')">
          <el-input v-model="form.passPhrase" type="password" show-password placeholder="可选" />
        </el-form-item>
        <el-form-item :label="$t('host.description')">
          <el-input v-model="form.description" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="info" @click="handleTestForm" :loading="testing">
          {{ $t('host.testConn') }}
        </el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">
          {{ $t('commons.confirm') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 分组管理对话框 -->
    <el-dialog v-model="groupDialogVisible" :title="$t('group.title')" width="440px" destroy-on-close>
      <div class="group-manage">
        <div class="group-add-row">
          <el-input v-model="newGroupName" :placeholder="$t('group.name')" size="small" />
          <el-button type="primary" size="small" @click="handleCreateGroup" :disabled="!newGroupName.trim()">
            {{ $t('group.addGroup') }}
          </el-button>
        </div>
        <el-table :data="groups" size="small">
          <el-table-column prop="name" :label="$t('group.name')" />
          <el-table-column :label="$t('commons.actions')" width="120">
            <template #default="{ row }">
              <el-button link type="danger" size="small" @click="handleDeleteGroup(row.id)">
                {{ $t('commons.delete') }}
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  searchHost,
  createHost,
  updateHost,
  deleteHost,
  testHost,
  testHostConn,
  getGroupList,
  createGroup,
  deleteGroup,
} from '@/api/modules/host'
import { ElMessage, ElMessageBox } from 'element-plus'

const emit = defineEmits<{
  connect: [hostId: number, label: string]
  back: []
}>()

const hosts = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const searchInfo = ref('')
const searchGroupID = ref(0)
const groups = ref<any[]>([])

const dialogVisible = ref(false)
const editMode = ref(false)
const submitting = ref(false)
const testing = ref(false)

const groupDialogVisible = ref(false)
const newGroupName = ref('')

const defaultForm = () => ({
  id: 0,
  groupID: 0,
  name: '',
  addr: '',
  port: 22,
  user: 'root',
  authMode: 'password' as string,
  password: '',
  privateKey: '',
  passPhrase: '',
  description: '',
})
const form = ref(defaultForm())

const loadData = async () => {
  try {
    const res = await searchHost({
      page: page.value,
      pageSize: pageSize.value,
      info: searchInfo.value,
      groupID: searchGroupID.value || undefined,
    })
    hosts.value = res.data?.items || []
    total.value = res.data?.total || 0
  } catch {
    hosts.value = []
  }
}

const loadGroups = async () => {
  try {
    const res = await getGroupList('host')
    groups.value = res.data || []
  } catch {
    groups.value = []
  }
}

const handleSearch = () => {
  page.value = 1
  loadData()
}

const handlePageChange = (p: number) => {
  page.value = p
  loadData()
}

const openDialog = (row?: any) => {
  if (row) {
    editMode.value = true
    form.value = { ...defaultForm(), ...row }
  } else {
    editMode.value = false
    form.value = defaultForm()
  }
  dialogVisible.value = true
}

const handleSubmit = async () => {
  if (!form.value.name || !form.value.addr) {
    ElMessage.warning('请填写名称和地址')
    return
  }
  submitting.value = true
  try {
    if (editMode.value) {
      await updateHost(form.value as any)
    } else {
      await createHost(form.value as any)
    }
    ElMessage.success('操作成功')
    dialogVisible.value = false
    loadData()
  } catch {
    ElMessage.error('操作失败')
  } finally {
    submitting.value = false
  }
}

const handleDelete = async (row: any) => {
  await ElMessageBox.confirm('确定要删除该主机吗？', '提示', { type: 'warning' })
  try {
    await deleteHost(row.id)
    ElMessage.success('删除成功')
    loadData()
  } catch {
    ElMessage.error('删除失败')
  }
}

const handleTest = async (id: number) => {
  const res = await testHost(id)
  if (res.data) {
    ElMessage.success('连接成功')
  } else {
    ElMessage.error('连接失败')
  }
}

const handleTestForm = async () => {
  if (!form.value.addr) {
    ElMessage.warning('请填写地址')
    return
  }
  testing.value = true
  try {
    const res = await testHostConn(form.value as any)
    if (res.data) {
      ElMessage.success('连接成功')
    } else {
      ElMessage.error('连接失败')
    }
  } catch {
    ElMessage.error('连接失败')
  } finally {
    testing.value = false
  }
}

const handleCreateGroup = async () => {
  if (!newGroupName.value.trim()) return
  try {
    await createGroup({ name: newGroupName.value.trim(), type: 'host' })
    newGroupName.value = ''
    loadGroups()
    ElMessage.success('创建成功')
  } catch {
    ElMessage.error('创建失败')
  }
}

const handleDeleteGroup = async (id: number) => {
  await ElMessageBox.confirm('确定要删除该分组吗？分组内的数据不会被删除。', '提示', { type: 'warning' })
  try {
    await deleteGroup(id)
    loadGroups()
    ElMessage.success('删除成功')
  } catch {
    ElMessage.error('删除失败')
  }
}

onMounted(() => {
  loadData()
  loadGroups()
})
</script>

<style lang="scss" scoped>
.host-manage {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.manage-header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--xp-border-light);
  margin-bottom: 16px;

  h3 {
    flex: 1;
    margin: 0;
    font-size: 16px;
    color: var(--xp-text-primary);
  }
}

.manage-body {
  flex: 1;
  overflow-y: auto;
}

.search-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;

  .search-input {
    width: 240px;
  }
}

.mt-pagination {
  margin-top: 16px;
  justify-content: flex-end;
}

.addr-row {
  display: flex;
  gap: 8px;
  width: 100%;

  .addr-input {
    flex: 1;
  }

  .port-input {
    width: 120px;
  }
}

.group-manage {
  .group-add-row {
    display: flex;
    gap: 8px;
    margin-bottom: 12px;
  }
}
</style>
