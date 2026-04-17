<template>
  <div class="server-list">
    <el-table :data="servers" size="small" border>
      <el-table-column prop="name" :label="$t('haproxy.serverName')" min-width="120" />
      <el-table-column :label="$t('haproxy.serverAddress')" min-width="160">
        <template #default="{ row }"><code>{{ row.address }}:{{ row.port }}</code></template>
      </el-table-column>
      <el-table-column :label="$t('haproxy.serverWeight')" width="150">
        <template #default="{ row }">
          <el-input-number v-model="row.weight" :min="0" :max="256" size="small" style="width: 110px;" @change="(v: any) => handleWeight(row, v)" />
        </template>
      </el-table-column>
      <el-table-column prop="maxConn" :label="$t('haproxy.serverMaxConn')" width="90" />
      <el-table-column :label="$t('haproxy.serverBackup')" width="70">
        <template #default="{ row }"><el-tag size="small" v-if="row.backup" type="warning">B</el-tag></template>
      </el-table-column>
      <el-table-column :label="$t('haproxy.serverSSL')" width="70">
        <template #default="{ row }"><el-tag size="small" v-if="row.ssl" type="success">SSL</el-tag></template>
      </el-table-column>
      <el-table-column :label="$t('commons.status')" width="110">
        <template #default="{ row }">
          <el-switch
            :model-value="!row.disabled"
            @change="(v: any) => handleToggle(row, v)"
            :active-text="$t('commons.online')"
            :inactive-text="$t('commons.offline')"
          />
        </template>
      </el-table-column>
      <el-table-column :label="$t('commons.actions')" width="140">
        <template #default="{ row }">
          <el-button link type="primary" size="small" @click="emit('edit', row)">{{ $t('commons.edit') }}</el-button>
          <el-button link type="danger" size="small" @click="handleDelete(row)">{{ $t('commons.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-empty v-if="!servers.length" :description="$t('haproxy.emptyServers')" :image-size="60" />
  </div>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { deleteHAProxyServer, toggleHAProxyServerLive, setHAProxyServerWeightLive } from '@/api/modules/haproxy'

const props = defineProps<{ backend: any; servers: any[] }>()
const emit = defineEmits(['refresh', 'edit'])
const { t } = useI18n()

const handleDelete = async (row: any) => {
  await ElMessageBox.confirm(t('haproxy.deleteServerConfirm'), t('commons.warning'), { type: 'warning' })
  await deleteHAProxyServer(row.id)
  ElMessage.success(t('commons.deleteSuccess'))
  emit('refresh')
}

const handleToggle = async (row: any, online: boolean) => {
  try {
    await toggleHAProxyServerLive({ id: row.id, disable: !online })
    ElMessage.success(t('commons.operationSuccess'))
    emit('refresh')
  } catch (e) {
    emit('refresh')
  }
}

const handleWeight = async (row: any, weight: number) => {
  try {
    await setHAProxyServerWeightLive({ id: row.id, weight })
    ElMessage.success(t('haproxy.weightUpdated'))
  } catch (e) {
    emit('refresh')
  }
}
</script>

<style lang="scss" scoped>
.server-list {
  padding: 8px 16px;
}
</style>
