<template>
  <el-dialog v-model="visible" :title="t('file.changeOwner')" width="480px" destroy-on-close>
    <el-form label-width="90px" v-loading="loadingUsers">
      <el-form-item :label="t('file.path')">
        <el-input :model-value="form.path" disabled />
      </el-form-item>
      <el-form-item :label="t('file.ownerUser')">
        <el-select v-model="form.user" filterable style="width: 100%">
          <el-option
            v-for="u in users"
            :key="u.username"
            :label="`${u.username} (${u.group})`"
            :value="u.username"
          />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('file.ownerGroup')">
        <el-select v-model="form.group" filterable style="width: 100%">
          <el-option v-for="g in groups" :key="g" :label="g" :value="g" />
        </el-select>
      </el-form-item>
      <el-form-item>
        <el-checkbox v-model="form.sub">{{ t('file.recursive') }}</el-checkbox>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">{{ t('commons.cancel') }}</el-button>
      <el-button type="primary" :loading="loading" @click="doChange">{{ t('commons.confirm') }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { changeFileOwner, getUsersAndGroups } from '@/api/modules/file'

const { t } = useI18n()
const emit = defineEmits(['done'])
const visible = ref(false)
const loading = ref(false)
const loadingUsers = ref(false)
const form = ref({ path: '', user: '', group: '', sub: false })
const users = ref<{ username: string; group: string }[]>([])
const groups = ref<string[]>([])

const loadUsersAndGroups = async () => {
  loadingUsers.value = true
  try {
    const res: any = await getUsersAndGroups()
    users.value = res.data?.users || []
    groups.value = res.data?.groups || []
  } catch { /* */ } finally {
    loadingUsers.value = false
  }
}

const open = (path: string, currentUser: string, currentGroup: string) => {
  form.value = { path, user: currentUser, group: currentGroup, sub: false }
  visible.value = true
  loadUsersAndGroups()
}

const doChange = async () => {
  if (!form.value.user || !form.value.group) return
  loading.value = true
  try {
    await changeFileOwner({
      path: form.value.path,
      user: form.value.user,
      group: form.value.group,
      sub: form.value.sub,
    })
    ElMessage.success(t('commons.success'))
    visible.value = false
    emit('done')
  } catch { /* */ } finally {
    loading.value = false
  }
}

defineExpose({ open })
</script>
