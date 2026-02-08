<template>
  <div class="command-manage">
    <div class="manage-header">
      <el-button link type="primary" @click="emit('back')">
        <el-icon><ArrowLeft /></el-icon>
        {{ $t('terminal.title') }}
      </el-button>
      <h3>{{ $t('command.title') }}</h3>
      <div class="header-actions">
        <el-button size="small" type="primary" @click="openDialog()">
          <el-icon><Plus /></el-icon>
          {{ $t('command.addCommand') }}
        </el-button>
      </div>
    </div>

    <div class="manage-body">
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

      <!-- 命令卡片列表 -->
      <div class="command-grid">
        <div
          v-for="cmd in commands"
          :key="cmd.id"
          class="command-card"
        >
          <div class="card-header">
            <span class="cmd-name">{{ cmd.name }}</span>
            <div class="card-actions">
              <el-button link type="success" size="small" @click="emit('execute', cmd.command)" :title="$t('command.execute')">
                <el-icon><VideoPlay /></el-icon>
              </el-button>
              <el-button link type="info" size="small" @click="handleCopy(cmd.command)" :title="$t('command.copyCmd')">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
              <el-button link type="primary" size="small" @click="openDialog(cmd)" :title="$t('commons.edit')">
                <el-icon><Edit /></el-icon>
              </el-button>
              <el-button link type="danger" size="small" @click="handleDelete(cmd)" :title="$t('commons.delete')">
                <el-icon><Delete /></el-icon>
              </el-button>
            </div>
          </div>
          <div class="card-body">
            <code class="cmd-code">{{ cmd.command }}</code>
          </div>
        </div>

        <div v-if="commands.length === 0" class="empty-state">
          <el-empty :description="$t('command.noCommand')" :image-size="80">
            <el-button type="primary" size="small" @click="openDialog()">
              {{ $t('command.addCommand') }}
            </el-button>
          </el-empty>
        </div>
      </div>

      <el-pagination
        v-if="total > pageSize"
        class="mt-pagination"
        :current-page="page"
        :page-size="pageSize"
        :total="total"
        layout="total, prev, pager, next"
        @current-change="handlePageChange"
      />
    </div>

    <!-- 新增/编辑命令对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="editMode ? $t('command.editCommand') : $t('command.addCommand')"
      width="480px"
      destroy-on-close
    >
      <el-form :model="form" label-width="80px" size="default">
        <el-form-item :label="$t('command.name')">
          <el-input v-model="form.name" placeholder="例：查看磁盘空间" />
        </el-form-item>
        <el-form-item :label="$t('command.command')">
          <el-input v-model="form.command" type="textarea" :rows="4" placeholder="df -h" />
        </el-form-item>
        <el-form-item :label="$t('command.group')">
          <el-select v-model="form.groupID" :placeholder="$t('commons.group')" clearable style="width: 100%">
            <el-option :label="'无分组'" :value="0" />
            <el-option v-for="g in groups" :key="g.id" :label="g.name" :value="g.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ $t('commons.cancel') }}</el-button>
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
  searchCommand,
  createCommand,
  updateCommand,
  deleteCommand,
  getGroupList,
  createGroup,
  deleteGroup,
} from '@/api/modules/host'
import { ElMessage, ElMessageBox } from 'element-plus'

const emit = defineEmits<{
  execute: [cmd: string]
  back: []
}>()

const commands = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(50)
const searchInfo = ref('')
const searchGroupID = ref(0)
const groups = ref<any[]>([])

const dialogVisible = ref(false)
const editMode = ref(false)
const submitting = ref(false)

const groupDialogVisible = ref(false)
const newGroupName = ref('')

const defaultForm = () => ({
  id: 0,
  groupID: 0,
  name: '',
  command: '',
})
const form = ref(defaultForm())

const loadData = async () => {
  try {
    const res = await searchCommand({
      page: page.value,
      pageSize: pageSize.value,
      info: searchInfo.value,
      groupID: searchGroupID.value || undefined,
    })
    commands.value = res.data?.items || []
    total.value = res.data?.total || 0
  } catch {
    commands.value = []
  }
}

const loadGroups = async () => {
  try {
    const res = await getGroupList('command')
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
  if (!form.value.name || !form.value.command) {
    ElMessage.warning('请填写名称和命令')
    return
  }
  submitting.value = true
  try {
    if (editMode.value) {
      await updateCommand(form.value as any)
    } else {
      await createCommand(form.value as any)
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
  await ElMessageBox.confirm('确定要删除该命令吗？', '提示', { type: 'warning' })
  try {
    await deleteCommand(row.id)
    ElMessage.success('删除成功')
    loadData()
  } catch {
    ElMessage.error('删除失败')
  }
}

const handleCopy = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制')
  } catch {
    ElMessage.error('复制失败')
  }
}

const handleCreateGroup = async () => {
  if (!newGroupName.value.trim()) return
  try {
    await createGroup({ name: newGroupName.value.trim(), type: 'command' })
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
.command-manage {
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

.command-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 12px;
}

.command-card {
  background: var(--xp-bg-surface);
  border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius-sm);
  padding: 12px 16px;
  transition: all 0.2s;

  &:hover {
    border-color: var(--xp-accent);
    box-shadow: var(--xp-accent-glow);
  }

  .card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 8px;

    .cmd-name {
      font-size: 14px;
      font-weight: 500;
      color: var(--xp-text-primary);
    }

    .card-actions {
      display: flex;
      gap: 2px;
    }
  }

  .card-body {
    .cmd-code {
      display: block;
      background: var(--xp-bg-base);
      padding: 8px 12px;
      border-radius: 4px;
      font-family: 'JetBrains Mono', 'Fira Code', monospace;
      font-size: 12px;
      color: var(--xp-accent);
      word-break: break-all;
      white-space: pre-wrap;
      max-height: 80px;
      overflow-y: auto;
    }
  }
}

.empty-state {
  grid-column: 1 / -1;
  padding: 40px 0;
}

.mt-pagination {
  margin-top: 16px;
  justify-content: flex-end;
}

.group-manage {
  .group-add-row {
    display: flex;
    gap: 8px;
    margin-bottom: 12px;
  }
}
</style>
