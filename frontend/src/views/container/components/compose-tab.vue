<template>
  <div>
    <div class="app-toolbar">
      <el-button type="primary" @click="openCreate">{{ t('container.composeCreate') }}</el-button>
      <el-button @click="openAttach">{{ t('container.composeAttach') }}</el-button>
    </div>
    <el-table :data="items" v-loading="loading">
      <el-table-column prop="name" :label="t('container.composeName')" min-width="140" show-overflow-tooltip />
      <el-table-column :label="t('container.composeSource')" width="110">
        <template #default="{ row }">
          <el-tag :type="sourceTagType(row.source)" size="small">{{ sourceLabel(row.source) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="status" :label="t('container.status')" width="140" show-overflow-tooltip />
      <el-table-column prop="path" :label="t('container.composePath')" min-width="240" show-overflow-tooltip />
      <el-table-column :label="t('commons.actions')" width="360" fixed="right">
        <template #default="{ row }">
          <template v-if="row.id > 0">
            <el-button
              link
              type="primary"
              :loading="isOpBusy(row, 'up')"
              :disabled="isBusy(row)"
              @click="operate(row, 'up')"
            >{{ t('container.start') }}</el-button>
            <el-button
              link
              type="primary"
              :loading="isOpBusy(row, 'stop')"
              :disabled="isBusy(row)"
              @click="operate(row, 'stop')"
            >{{ t('container.stop') }}</el-button>
            <el-button
              link
              type="primary"
              :loading="isOpBusy(row, 'restart')"
              :disabled="isBusy(row)"
              @click="operate(row, 'restart')"
            >{{ t('container.restart') }}</el-button>
            <el-button
              link
              type="primary"
              :loading="isOpBusy(row, 'update')"
              :disabled="isBusy(row)"
              @click="operate(row, 'update')"
            >{{ t('container.composeUpdate') }}</el-button>
            <el-dropdown trigger="click" :disabled="isBusy(row)">
              <el-button link type="primary" :disabled="isBusy(row)">
                {{ t('container.more') }}<el-icon class="el-icon--right"><arrow-down /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item @click="openEdit(row)">{{ t('container.composeEdit') }}</el-dropdown-item>
                  <el-dropdown-item @click="operate(row, 'down')">{{ t('container.composeDown') }}</el-dropdown-item>
                  <el-dropdown-item divided @click="handleDelete(row)">
                    <span style="color: var(--el-color-danger)">{{ t('commons.delete') }}</span>
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
          <el-button
            v-else-if="row.source === 'unmanaged'"
            link
            type="primary"
            :loading="isOpBusy(row, 'attach')"
            :disabled="isBusy(row)"
            @click="attachUnmanaged(row)"
          >{{ t('container.composeAttachNow') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-drawer v-model="createDrawer" :title="t('container.composeCreate')" size="640px" destroy-on-close>
      <el-alert type="info" :closable="false" style="margin-bottom:16px">{{ t('container.composeCreateHint') }}</el-alert>
      <el-form label-width="110px">
        <el-form-item :label="t('container.composeName')">
          <el-input v-model="createForm.name" />
        </el-form-item>
        <el-form-item :label="t('container.composeContent')">
          <el-input v-model="createForm.content" type="textarea" :rows="16" class="compose-yaml" />
        </el-form-item>
        <el-form-item :label="t('container.composeFile')">
          <input type="file" accept=".yml,.yaml,text/yaml,text/plain" @change="onCreateFileChange" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDrawer = false">{{ t('commons.cancel') }}</el-button>
        <el-button type="primary" :loading="isNameBusy(createForm.name)" @click="submitCreate">{{ t('commons.confirm') }}</el-button>
      </template>
    </el-drawer>

    <el-drawer v-model="attachDrawer" :title="t('container.composeAttach')" size="520px" destroy-on-close>
      <el-alert type="info" :closable="false" style="margin-bottom:16px">{{ t('container.composeAttachHint') }}</el-alert>
      <el-form label-width="110px">
        <el-form-item :label="t('container.composeName')">
          <el-input v-model="attachForm.name" />
        </el-form-item>
        <el-form-item :label="t('container.composePath')">
          <el-input v-model="attachForm.path" placeholder="/opt/app/docker-compose.yml" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="attachDrawer = false">{{ t('commons.cancel') }}</el-button>
        <el-button type="primary" :loading="isNameBusy(attachForm.name)" @click="submitAttach">{{ t('commons.confirm') }}</el-button>
      </template>
    </el-drawer>

    <el-drawer v-model="editDrawer" :title="t('container.composeEdit')" size="640px" destroy-on-close>
      <el-alert type="info" :closable="false" style="margin-bottom:16px">{{ t('container.composeEditHint') }}</el-alert>
      <el-form label-width="110px">
        <el-form-item :label="t('container.composeContent')">
          <el-input v-model="editContent" type="textarea" :rows="20" class="compose-yaml" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDrawer = false">{{ t('commons.cancel') }}</el-button>
        <el-button type="primary" :loading="isNameBusy(editName)" @click="submitEdit">{{ t('commons.save') }}</el-button>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowDown } from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'
import type { ComposeItem } from '@/api/interface'
import {
  listCompose,
  createCompose,
  operateCompose,
  getComposeContent,
  updateComposeContent,
  deleteCompose,
} from '@/api/modules/container'
import { isValidComposeName, composeCreateMode } from './compose-form.ts'

const { t } = useI18n()

const loading = ref(false)
const items = ref<ComposeItem[]>([])
const busyMap = ref<Record<string, string>>({})

const createDrawer = ref(false)
const createForm = reactive({ name: '', content: '' })

const attachDrawer = ref(false)
const attachForm = reactive({ name: '', path: '' })

const editDrawer = ref(false)
const editId = ref(0)
const editName = ref('')
const editContent = ref('')

const isNameBusy = (name: string) => !!busyMap.value[name.trim()]
const isBusy = (row: ComposeItem) => isNameBusy(row.name)
const isOpBusy = (row: ComposeItem, op: string) => busyMap.value[row.name] === op

const sourceLabel = (source: ComposeItem['source']) => {
  if (source === 'created') return t('container.composeSourceCreated')
  if (source === 'attached') return t('container.composeSourceAttached')
  return t('container.composeSourceUnmanaged')
}

const sourceTagType = (source: ComposeItem['source']) => {
  if (source === 'created') return 'success'
  if (source === 'attached') return 'warning'
  return 'info'
}

const loadItems = async () => {
  loading.value = true
  try {
    const res = await listCompose()
    items.value = res.data || []
  } finally {
    loading.value = false
  }
}

const runBusy = async (name: string, op: string, fn: () => Promise<void>) => {
  const key = name.trim()
  if (busyMap.value[key]) {
    ElMessage.warning(t('container.composeBusy'))
    return
  }
  busyMap.value = { ...busyMap.value, [key]: op }
  try {
    await fn()
    ElMessage.success(t('commons.success'))
  } catch {
    /* interceptor already toasts */
  } finally {
    const next = { ...busyMap.value }
    delete next[key]
    busyMap.value = next
    // Refresh even after failure (e.g. create persist-then-up-fail still shows the row).
    await loadItems()
  }
}

const operate = (row: ComposeItem, operation: string) => {
  return runBusy(row.name, operation, async () => {
    await operateCompose({ id: row.id, operation })
  })
}

const attachUnmanaged = (row: ComposeItem) => {
  return runBusy(row.name, 'attach', async () => {
    await operateCompose({ name: row.name, path: row.path, operation: 'attach' })
  })
}

const openCreate = () => {
  createForm.name = ''
  createForm.content = ''
  createDrawer.value = true
}

const onCreateFileChange = (event: Event) => {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file) return
  const reader = new FileReader()
  reader.onload = () => {
    createForm.content = String(reader.result || '')
  }
  reader.readAsText(file)
}

const submitCreate = async () => {
  const name = createForm.name.trim()
  if (!isValidComposeName(name) || composeCreateMode({ content: createForm.content, path: '' }) !== 'create') {
    return
  }
  await runBusy(name, 'create', async () => {
    await createCompose({ name, content: createForm.content })
    createDrawer.value = false
  })
}

const openAttach = () => {
  attachForm.name = ''
  attachForm.path = ''
  attachDrawer.value = true
}

const submitAttach = async () => {
  const name = attachForm.name.trim()
  const path = attachForm.path.trim()
  if (!isValidComposeName(name) || composeCreateMode({ content: '', path }) !== 'attach') {
    return
  }
  await runBusy(name, 'attach', async () => {
    await createCompose({ name, path })
    attachDrawer.value = false
  })
}

const openEdit = async (row: ComposeItem) => {
  if (isBusy(row)) {
    ElMessage.warning(t('container.composeBusy'))
    return
  }
  try {
    const res = await getComposeContent(row.id)
    editId.value = row.id
    editName.value = row.name
    editContent.value = res.data || ''
    editDrawer.value = true
  } catch {
    /* interceptor already toasts */
  }
}

const submitEdit = async () => {
  await runBusy(editName.value, 'edit', async () => {
    await updateComposeContent({ id: editId.value, content: editContent.value })
    editDrawer.value = false
  })
}

const handleDelete = async (row: ComposeItem) => {
  if (isBusy(row)) {
    ElMessage.warning(t('container.composeBusy'))
    return
  }
  try {
    await ElMessageBox.confirm(
      row.source === 'created' ? t('container.composeDeleteCreatedConfirm') : t('container.composeDeleteAttachedConfirm'),
      t('commons.tip'),
      { type: 'warning' },
    )
  } catch {
    return
  }
  await runBusy(row.name, 'delete', async () => {
    await deleteCompose({ id: row.id })
  })
}

onMounted(() => {
  loadItems()
})
</script>

<style scoped>
.compose-yaml :deep(textarea) {
  font-family: var(--xp-font-mono);
  font-size: 13px;
}
</style>
