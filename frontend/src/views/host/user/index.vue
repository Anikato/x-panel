<template>
  <div>
    <el-card shadow="never">
      <template #header>
        <div style="display: flex; align-items: center; justify-content: space-between;">
          <span>{{ $t('userManage.title') }}</span>
          <div>
            <el-checkbox v-model="showSystem" @change="loadUsers" style="margin-right: 12px;">
              {{ $t('userManage.showSystem') }}
            </el-checkbox>
            <el-button type="primary" @click="openCreate">
              <el-icon><Plus /></el-icon>
              {{ $t('userManage.createUser') }}
            </el-button>
          </div>
        </div>
      </template>

      <el-table :data="filteredUsers" v-loading="loading" stripe>
        <el-table-column prop="username" :label="$t('userManage.username')" min-width="120">
          <template #default="{ row }">
            <el-tag v-if="row.uid === 0" type="danger" size="small" style="margin-right: 4px;">root</el-tag>
            <span>{{ row.username }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="uid" label="UID" width="80" />
        <el-table-column prop="gid" label="GID" width="80" />
        <el-table-column prop="comment" :label="$t('userManage.comment')" min-width="120" show-overflow-tooltip />
        <el-table-column prop="home" :label="$t('userManage.home')" min-width="160" show-overflow-tooltip />
        <el-table-column prop="shell" :label="$t('userManage.shell')" min-width="140" show-overflow-tooltip />
        <el-table-column prop="groups" :label="$t('userManage.groups')" min-width="160" show-overflow-tooltip />
        <el-table-column :label="$t('userManage.sudo')" width="80">
          <template #default="{ row }">
            <el-tag v-if="row.isSudo" type="warning" size="small">sudo</el-tag>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('commons.actions')" width="160" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)" :disabled="row.uid === 0">
              {{ $t('commons.edit') }}
            </el-button>
            <el-button link type="danger" @click="handleDelete(row)" :disabled="row.uid === 0 || row.isSystem">
              {{ $t('commons.delete') }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 创建/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? $t('userManage.editUser') : $t('userManage.createUser')"
      width="520px"
      destroy-on-close
    >
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item :label="$t('userManage.username')" prop="username">
          <el-input v-model="form.username" :disabled="isEdit" />
        </el-form-item>
        <el-form-item :label="$t('userManage.password')" prop="password">
          <el-input v-model="form.password" type="password" show-password :placeholder="$t('userManage.passwordHint')" />
        </el-form-item>
        <el-form-item :label="$t('userManage.comment')">
          <el-input v-model="form.comment" />
        </el-form-item>
        <el-form-item :label="$t('userManage.home')">
          <el-input v-model="form.home" placeholder="/home/username" />
        </el-form-item>
        <el-form-item :label="$t('userManage.shell')">
          <el-select v-model="form.shell" filterable allow-create style="width: 100%;">
            <el-option v-for="s in shells" :key="s" :label="s" :value="s" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="!isEdit" :label="$t('userManage.createHome')">
          <el-switch v-model="form.createHome" />
        </el-form-item>
        <el-form-item :label="$t('userManage.sudo')">
          <el-switch v-model="form.sudo" />
          <el-text type="info" size="small" style="margin-left: 8px;">{{ $t('userManage.sudoHint') }}</el-text>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">{{ $t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import {
  listLinuxUsers,
  createLinuxUser,
  updateLinuxUser,
  deleteLinuxUser,
  listShells,
} from '@/api/modules/host'

const { t } = useI18n()

const loading = ref(false)
const users = ref<any[]>([])
const shells = ref<string[]>([])
const showSystem = ref(false)
const search = ref('')

const filteredUsers = computed(() => {
  if (!search.value) return users.value
  const kw = search.value.toLowerCase()
  return users.value.filter(
    (u: any) =>
      u.username.toLowerCase().includes(kw) ||
      (u.comment && u.comment.toLowerCase().includes(kw))
  )
})

const dialogVisible = ref(false)
const isEdit = ref(false)
const submitting = ref(false)
const formRef = ref<FormInstance>()

const defaultForm = () => ({
  username: '',
  password: '',
  comment: '',
  home: '',
  shell: '/bin/bash',
  createHome: true,
  sudo: false,
})

const form = reactive(defaultForm())

const usernamePattern = /^[a-z_][a-z0-9_-]*$/
const rules = reactive<FormRules>({
  username: [
    { required: true, message: () => t('userManage.usernameRequired'), trigger: 'blur' },
    { pattern: usernamePattern, message: () => t('userManage.usernameRule'), trigger: 'blur' },
  ],
})

const loadUsers = async () => {
  loading.value = true
  try {
    const res = await listLinuxUsers(showSystem.value)
    users.value = res.data || []
  } finally {
    loading.value = false
  }
}

const loadShells = async () => {
  try {
    const res = await listShells()
    shells.value = res.data || []
  } catch {}
}

const openCreate = () => {
  isEdit.value = false
  Object.assign(form, defaultForm())
  dialogVisible.value = true
}

const openEdit = (row: any) => {
  isEdit.value = true
  Object.assign(form, {
    username: row.username,
    password: '',
    comment: row.comment,
    home: row.home,
    shell: row.shell,
    createHome: false,
    sudo: row.isSudo || false,
  })
  dialogVisible.value = true
}

const handleSubmit = async () => {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    if (isEdit.value) {
      await updateLinuxUser({
        username: form.username,
        password: form.password || undefined,
        comment: form.comment,
        home: form.home,
        shell: form.shell,
        sudo: form.sudo,
      })
    } else {
      await createLinuxUser({
        username: form.username,
        password: form.password || undefined,
        comment: form.comment,
        home: form.home,
        shell: form.shell,
        createHome: form.createHome,
        sudo: form.sudo,
      })
    }
    ElMessage.success(t('commons.success'))
    dialogVisible.value = false
    loadUsers()
  } finally {
    submitting.value = false
  }
}

const handleDelete = (row: any) => {
  ElMessageBox.confirm(
    t('userManage.deleteConfirm', { name: row.username }),
    t('commons.tip'),
    {
      confirmButtonText: t('commons.confirm'),
      cancelButtonText: t('commons.cancel'),
      type: 'warning',
      showInput: false,
    }
  )
    .then(async () => {
      const removeHome = await ElMessageBox.confirm(
        t('userManage.removeHome'),
        t('commons.tip'),
        {
          confirmButtonText: t('commons.confirm'),
          cancelButtonText: t('commons.cancel'),
          type: 'info',
          distinguishCancelAndClose: true,
        }
      )
        .then(() => true)
        .catch(() => false)

      await deleteLinuxUser({ username: row.username, removeHome })
      ElMessage.success(t('commons.deleteSuccess'))
      loadUsers()
    })
    .catch(() => {})
}

onMounted(() => {
  loadUsers()
  loadShells()
})
</script>
