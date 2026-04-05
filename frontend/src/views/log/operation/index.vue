<template>
  <div>
    <el-card>
      <template #header>
        <div class="card-header">
          <div class="card-header-title">
            <el-icon><Notebook /></el-icon>
            <span>{{ t('log.operationLog') }}</span>
          </div>
          <el-button type="danger" plain size="small" @click="handleClean">
            <el-icon><Delete /></el-icon>{{ t('log.clean') }}
          </el-button>
        </div>
      </template>

      <el-table :data="tableData" v-loading="loading">
        <el-table-column :label="t('log.operation')" min-width="220">
          <template #default="{ row }">
            <div class="op-cell">
              <el-tag :type="methodType(row.method)" size="small" class="op-method">{{ row.method }}</el-tag>
              <span class="op-desc">{{ describeOperation(row) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="path" :label="t('log.path')" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            <code class="path-text">{{ row.path }}</code>
          </template>
        </el-table-column>
        <el-table-column prop="ip" :label="t('log.ip')" width="140" />
        <el-table-column prop="status" :label="t('log.status')" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 'Success' ? 'success' : 'danger'" size="small" round>
              {{ row.status === 'Success' ? t('log.success') : t('log.failed') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="latency" :label="t('log.latency')" width="100" align="center">
          <template #default="{ row }">
            <span class="latency-text">{{ row.latency || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('log.time')" width="170">
          <template #default="{ row }">
            <span class="time-text">{{ formatTime(row.createdAt) }}</span>
          </template>
        </el-table-column>
      </el-table>

      <div class="table-footer">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @size-change="fetchData"
          @current-change="fetchData"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessageBox, ElMessage } from 'element-plus'
import { getOperationLogs, cleanOperationLogs } from '@/api/modules/log'
import { useI18n } from 'vue-i18n'
import type { OperationLog } from '@/api/interface'

const { t } = useI18n()
const loading = ref(false)
const tableData = ref<OperationLog[]>([])
const pagination = reactive({ page: 1, pageSize: 20, total: 0 })

const methodType = (m: string): string => ({ GET: 'info', POST: 'success', PUT: 'warning', DELETE: 'danger' }[m] || 'info')

const operationMap: Record<string, string> = {
  '/api/v1/auth/login': '用户登录',
  '/api/v1/auth/logout': '用户退出',
  '/api/v1/auth/init': '初始化系统',
  '/api/v1/settings/update': '更新面板设置',
  '/api/v1/settings/password': '修改密码',
  '/api/v1/settings/username': '修改用户名',
  '/api/v1/settings/port': '修改面板端口',
  '/api/v1/websites': '创建网站',
  '/api/v1/websites/search': '查询网站列表',
  '/api/v1/databases/servers/search': '查询数据库列表',
  '/api/v1/logs/operation': '查看操作日志',
  '/api/v1/logs/operation/clean': '清空操作日志',
  '/api/v1/logs/login': '查看登录日志',
  '/api/v1/logs/login/clean': '清空登录日志',
  '/api/v1/cronjobs/search': '查询计划任务',
  '/api/v1/firewall/rules': '管理防火墙规则',
  '/api/v1/containers': '管理容器',
  '/api/v1/files/upload': '上传文件',
  '/api/v1/files/delete': '删除文件',
  '/api/v1/files/mkdir': '创建目录',
  '/api/v1/nginx/reload': '重载 Nginx',
  '/api/v1/traffic/configs': '管理流量监控',
}

const describeOperation = (row: OperationLog): string => {
  if (operationMap[row.path]) return operationMap[row.path]

  const parts = (row.path || '').replace('/api/v1/', '').split('/')
  const group = parts[0] || ''
  const action = parts.slice(1).join('/') || ''

  const groupNames: Record<string, string> = {
    auth: '认证', websites: '网站', databases: '数据库', containers: '容器',
    files: '文件', firewall: '防火墙', cronjobs: '计划任务',
    nginx: 'Nginx', settings: '设置', logs: '日志', traffic: '流量',
    monitor: '监控', disk: '磁盘', host: '主机', ssh: 'SSH', ssl: '证书',
    backup: '备份', nodes: '节点', process: '进程', toolbox: '工具箱',
  }

  const methodNames: Record<string, string> = {
    POST: '操作', PUT: '更新', DELETE: '删除',
  }

  const gn = groupNames[group] || group
  const mn = methodNames[row.method] || row.method
  return action ? `${mn} ${gn} / ${action}` : `${mn} ${gn}`
}

const formatTime = (timeStr: string): string => {
  if (!timeStr) return '-'
  try {
    const d = new Date(timeStr)
    if (isNaN(d.getTime())) return timeStr
    const now = new Date()
    const isToday = d.toDateString() === now.toDateString()
    const yesterday = new Date(now)
    yesterday.setDate(yesterday.getDate() - 1)
    const isYesterday = d.toDateString() === yesterday.toDateString()

    const time = `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}:${String(d.getSeconds()).padStart(2, '0')}`

    if (isToday) return `今天 ${time}`
    if (isYesterday) return `昨天 ${time}`
    const month = d.getMonth() + 1
    const day = d.getDate()
    if (d.getFullYear() === now.getFullYear()) {
      return `${month}月${day}日 ${time}`
    }
    return `${d.getFullYear()}/${month}/${day} ${time}`
  } catch {
    return timeStr
  }
}

const fetchData = async () => {
  loading.value = true
  try {
    const res = await getOperationLogs({ page: pagination.page, pageSize: pagination.pageSize })
    tableData.value = res.data?.items || []
    pagination.total = res.data?.total || 0
  } catch { /* */ } finally { loading.value = false }
}

const handleClean = async () => {
  try {
    await ElMessageBox.confirm(t('log.cleanConfirm'), t('commons.tip'), { type: 'warning' })
    await cleanOperationLogs()
    ElMessage.success(t('commons.success'))
    fetchData()
  } catch { /* cancelled */ }
}

onMounted(() => fetchData())
</script>

<style lang="scss" scoped>
.op-cell {
  display: flex;
  align-items: center;
  gap: 8px;

  .op-method {
    flex-shrink: 0;
    font-family: 'JetBrains Mono', monospace;
    font-size: 11px;
  }

  .op-desc {
    font-size: 13px;
    color: var(--xp-text-primary);
  }
}

.path-text {
  font-size: 12px;
  color: var(--xp-text-muted);
  font-family: 'JetBrains Mono', monospace;
}

.latency-text {
  font-size: 12px;
  font-family: 'JetBrains Mono', monospace;
  color: var(--xp-text-secondary);
}

.time-text {
  font-size: 12px;
  color: var(--xp-text-secondary);
}
</style>
