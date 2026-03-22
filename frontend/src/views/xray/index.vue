<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import * as echarts from 'echarts/core'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import {
  Connection, Plus, Delete, Edit, User, Refresh, Key, CopyDocument, Download,
} from '@element-plus/icons-vue'
import type {
  XrayNode, XrayNodeCreate, XrayUser, XrayUserCreate, XrayUserUpdate,
  XrayStatus, XrayRealityKeys, XrayTrafficDaily,
} from '@/api/modules/xray'
import {
  getXrayStatus, startXrayInstall, getXrayInstallLog,
  listXrayNodes, createXrayNode, updateXrayNode, deleteXrayNode, toggleXrayNode,
  searchXrayUsers, createXrayUser, updateXrayUser, deleteXrayUser,
  generateRealityKeys, getXrayShareLink, getXrayTrafficHistory,
} from '@/api/modules/xray'

echarts.use([LineChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer])

const { t } = useI18n()

// ==================== 状态 ====================
const xrayStatus = ref<XrayStatus>({
  installed: false, running: false, version: '', configPath: '', binPath: '',
})
const nodes = ref<XrayNode[]>([])
const activeNodeId = ref<number | null>(null)
const users = ref<XrayUser[]>([])
const usersTotal = ref(0)
const userPage = ref(1)
const userPageSize = ref(15)
const loading = ref(false)
const usersLoading = ref(false)

// 安装相关
const installing = ref(false)
const installLog = ref('')
let installPollTimer: ReturnType<typeof setInterval> | null = null

// 节点对话框
const nodeDialogVisible = ref(false)
const nodeDialogMode = ref<'create' | 'edit'>('create')
const editingNode = ref<XrayNode | null>(null)
const nodeForm = ref<XrayNodeCreate>({
  name: '', protocol: 'vless', port: 443, transport: 'tcp', security: 'reality',
  domain: '', realityPrivateKey: '', realityPublicKey: '',
  realityShortIds: '[""]', realityServerNames: '["www.apple.com"]',
  path: '/', serviceName: '', remark: '',
})
const generatingKeys = ref(false)

// 用户对话框
const userDialogVisible = ref(false)
const userDialogMode = ref<'create' | 'edit'>('create')
const userForm = ref<XrayUserCreate & { id?: number; enabled?: boolean }>({
  nodeId: 0, name: '', uuid: '', level: 0, expireAt: null, remark: '',
})

// 分享链接
const shareLinkDialog = ref(false)
const shareLink = ref('')

// 流量历史图表
const chartDialog = ref(false)
const chartUser = ref<XrayUser | null>(null)
const chartRef = ref<HTMLDivElement | null>(null)
let chartInstance: echarts.ECharts | null = null
const chartLoading = ref(false)

// ==================== 初始化 ====================
const activeNode = computed(() => nodes.value.find((n) => n.id === activeNodeId.value) || null)

onMounted(async () => {
  await loadStatus()
})

onUnmounted(() => {
  if (installPollTimer) clearInterval(installPollTimer)
  chartInstance?.dispose()
})

async function loadStatus() {
  const res: any = await getXrayStatus()
  xrayStatus.value = res.data
  if (xrayStatus.value.installed) {
    await loadNodes()
  }
}

async function loadNodes() {
  loading.value = true
  try {
    const res: any = await listXrayNodes()
    nodes.value = res.data || []
    if (nodes.value.length > 0 && !activeNodeId.value) {
      activeNodeId.value = nodes.value[0].id
      await loadUsers()
    }
  } finally {
    loading.value = false
  }
}

async function loadUsers() {
  if (!activeNodeId.value) return
  usersLoading.value = true
  try {
    const res: any = await searchXrayUsers({
      nodeId: activeNodeId.value,
      page: userPage.value,
      pageSize: userPageSize.value,
    })
    users.value = res.data?.items || []
    usersTotal.value = res.data?.total || 0
  } finally {
    usersLoading.value = false
  }
}

function selectNode(id: number) {
  activeNodeId.value = id
  userPage.value = 1
  loadUsers()
}

// ==================== 安装引导 ====================
async function handleInstall() {
  await ElMessageBox.confirm(t('xray.confirmInstall'), t('xray.installTitle'), { type: 'info' })
  installing.value = true
  installLog.value = ''
  await startXrayInstall()

  installPollTimer = setInterval(async () => {
    const res: any = await getXrayInstallLog()
    installLog.value = res.data?.log || ''
    const log = installLog.value
    const done = log.includes('[DONE]') || log.includes('[ERROR]')
    if (done) {
      if (installPollTimer) clearInterval(installPollTimer)
      installing.value = false
      if (log.includes('[DONE]')) {
        ElMessage.success(t('xray.installSuccess'))
        await loadStatus()
      } else {
        ElMessage.error(t('xray.installFailed'))
      }
    }
  }, 2000)
}

// ==================== 节点操作 ====================
function openCreateNode() {
  nodeDialogMode.value = 'create'
  editingNode.value = null
  nodeForm.value = {
    name: '', protocol: 'vless', port: 443, transport: 'tcp', security: 'reality',
    domain: '', realityPrivateKey: '', realityPublicKey: '',
    realityShortIds: '[""]', realityServerNames: '["www.apple.com"]',
    path: '/', serviceName: '', remark: '',
  }
  nodeDialogVisible.value = true
}

function openEditNode(node: XrayNode) {
  nodeDialogMode.value = 'edit'
  editingNode.value = node
  nodeForm.value = {
    name: node.name, protocol: node.protocol, port: node.port,
    transport: node.transport, security: node.security, domain: node.domain,
    realityPrivateKey: '', realityPublicKey: node.realityPublicKey,
    realityShortIds: node.realityShortIds || '[""]',
    realityServerNames: node.realityServerNames || '["www.apple.com"]',
    path: node.path, serviceName: node.serviceName, remark: node.remark,
  }
  nodeDialogVisible.value = true
}

async function handleGenerateKeys() {
  generatingKeys.value = true
  try {
    const res: any = await generateRealityKeys()
    nodeForm.value.realityPrivateKey = res.data.privateKey
    nodeForm.value.realityPublicKey = res.data.publicKey
    nodeForm.value.realityShortIds = `["${randomHex(8)}"]`
  } finally {
    generatingKeys.value = false
  }
}

async function submitNodeForm() {
  if (nodeDialogMode.value === 'create') {
    await createXrayNode(nodeForm.value)
    ElMessage.success(t('xray.nodeCreated'))
  } else if (editingNode.value) {
    await updateXrayNode({
      id: editingNode.value.id,
      name: nodeForm.value.name,
      transport: nodeForm.value.transport!,
      security: nodeForm.value.security!,
      domain: nodeForm.value.domain,
      realityPrivateKey: nodeForm.value.realityPrivateKey,
      realityPublicKey: nodeForm.value.realityPublicKey,
      realityShortIds: nodeForm.value.realityShortIds,
      realityServerNames: nodeForm.value.realityServerNames,
      path: nodeForm.value.path,
      serviceName: nodeForm.value.serviceName,
      remark: nodeForm.value.remark,
      enabled: editingNode.value.enabled,
    })
    ElMessage.success(t('common.updateSuccess'))
  }
  nodeDialogVisible.value = false
  await loadNodes()
}

async function handleDeleteNode(node: XrayNode) {
  await ElMessageBox.confirm(
    t('xray.confirmDeleteNode', { name: node.name }), t('common.warning'), { type: 'warning' },
  )
  await deleteXrayNode(node.id)
  ElMessage.success(t('common.deleteSuccess'))
  if (activeNodeId.value === node.id) { activeNodeId.value = null; users.value = [] }
  await loadNodes()
}

async function handleToggleNode(node: XrayNode) {
  await toggleXrayNode(node.id)
  node.enabled = !node.enabled
}

// ==================== 用户操作 ====================
function openCreateUser() {
  if (!activeNodeId.value) return
  userDialogMode.value = 'create'
  userForm.value = { nodeId: activeNodeId.value, name: '', uuid: '', level: 0, expireAt: null, remark: '' }
  userDialogVisible.value = true
}

function openEditUser(user: XrayUser) {
  userDialogMode.value = 'edit'
  userForm.value = {
    id: user.id, nodeId: user.nodeId, name: user.name, uuid: user.uuid,
    level: user.level, expireAt: user.expireAt, enabled: user.enabled, remark: user.remark,
  }
  userDialogVisible.value = true
}

async function submitUserForm() {
  if (userDialogMode.value === 'create') {
    await createXrayUser(userForm.value)
    ElMessage.success(t('xray.userCreated'))
  } else {
    await updateXrayUser({
      id: userForm.value.id!,
      name: userForm.value.name,
      level: userForm.value.level || 0,
      expireAt: userForm.value.expireAt,
      enabled: userForm.value.enabled ?? true,
      remark: userForm.value.remark,
    } as XrayUserUpdate)
    ElMessage.success(t('common.updateSuccess'))
  }
  userDialogVisible.value = false
  await loadUsers()
}

async function handleDeleteUser(user: XrayUser) {
  await ElMessageBox.confirm(
    t('xray.confirmDeleteUser', { name: user.name }), t('common.warning'), { type: 'warning' },
  )
  await deleteXrayUser(user.id)
  ElMessage.success(t('common.deleteSuccess'))
  await loadUsers()
}

async function handleShareLink(user: XrayUser) {
  const res: any = await getXrayShareLink(user.id)
  shareLink.value = res.data?.link || ''
  shareLinkDialog.value = true
}

async function copyShareLink() {
  await navigator.clipboard.writeText(shareLink.value)
  ElMessage.success(t('common.copied'))
}

// ==================== 流量历史图表 ====================
async function openTrafficChart(user: XrayUser) {
  chartUser.value = user
  chartDialog.value = true
  chartLoading.value = true
  await nextTick()

  const res: any = await getXrayTrafficHistory(user.id)
  const data: XrayTrafficDaily[] = res.data || []
  chartLoading.value = false

  await nextTick()
  if (!chartRef.value) return
  chartInstance?.dispose()
  chartInstance = echarts.init(chartRef.value)

  const dates = data.map((d) => d.date)
  const uploads = data.map((d) => +(d.upload / 1024 / 1024).toFixed(2))
  const downloads = data.map((d) => +(d.download / 1024 / 1024).toFixed(2))

  chartInstance.setOption({
    tooltip: { trigger: 'axis', formatter: (params: any[]) =>
      params.map((p: any) => `${p.marker}${p.seriesName}: ${p.value} MB`).join('<br/>'),
    },
    legend: { data: [t('xray.upload'), t('xray.download')] },
    grid: { left: 40, right: 20, top: 40, bottom: 30 },
    xAxis: { type: 'category', data: dates, axisLabel: { rotate: 30, fontSize: 11 } },
    yAxis: { type: 'value', name: 'MB', axisLabel: { formatter: '{value}' } },
    series: [
      {
        name: t('xray.upload'), type: 'line', smooth: true,
        data: uploads, itemStyle: { color: '#67c23a' },
        areaStyle: { opacity: 0.15 },
      },
      {
        name: t('xray.download'), type: 'line', smooth: true,
        data: downloads, itemStyle: { color: '#409eff' },
        areaStyle: { opacity: 0.15 },
      },
    ],
  })
}

// ==================== 工具函数 ====================
function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}

function randomHex(len: number): string {
  return Array.from({ length: len }, () => Math.floor(Math.random() * 16).toString(16)).join('')
}

function protocolColor(p: string) {
  return { vless: 'primary', vmess: 'success', trojan: 'warning' }[p] || 'info'
}

function securityColor(s: string) {
  return { reality: 'primary', tls: 'success', none: 'info' }[s] || 'info'
}

function isExpired(expireAt: string | null): boolean {
  return !!expireAt && new Date(expireAt) < new Date()
}
</script>

<template>
  <div class="xray-container">

    <!-- ==================== 未安装引导 ==================== -->
    <div v-if="!xrayStatus.installed" class="install-banner">
      <div class="install-card">
        <div class="install-icon">
          <el-icon :size="48" color="var(--xp-accent)"><Connection /></el-icon>
        </div>
        <h2>{{ t('xray.notInstalled') }}</h2>
        <p>{{ t('xray.notInstalledDesc') }}</p>

        <div v-if="installing" class="install-progress">
          <el-progress :percentage="100" status="striped" striped-flow :duration="3" />
          <div class="install-log">
            <pre>{{ installLog || t('xray.installing') }}</pre>
          </div>
        </div>

        <el-button v-else type="primary" size="large" :icon="Download" @click="handleInstall">
          {{ t('xray.installBtn') }}
        </el-button>
      </div>
    </div>

    <!-- ==================== 主界面 ==================== -->
    <template v-else>
      <!-- 顶部状态栏 -->
      <div class="xray-header">
        <div class="header-left">
          <el-tag :type="xrayStatus.running ? 'success' : 'danger'" size="large" class="status-tag">
            <span class="status-dot" :class="{ running: xrayStatus.running }" />
            {{ xrayStatus.running ? t('xray.running') : t('xray.stopped') }}
          </el-tag>
          <span class="version-text" v-if="xrayStatus.version">{{ xrayStatus.version }}</span>
        </div>
        <div class="header-right">
          <el-button :icon="Refresh" @click="loadStatus">{{ t('common.refresh') }}</el-button>
          <el-button type="primary" :icon="Plus" @click="openCreateNode">{{ t('xray.addNode') }}</el-button>
        </div>
      </div>

      <div class="xray-content">
        <!-- 左侧节点列表 -->
        <div class="node-sidebar">
          <div class="sidebar-title">{{ t('xray.nodes') }}</div>
          <el-empty v-if="nodes.length === 0 && !loading" :description="t('xray.noNodes')" :image-size="60" />
          <div
            v-for="node in nodes"
            :key="node.id"
            class="node-item"
            :class="{ active: activeNodeId === node.id, disabled: !node.enabled }"
            @click="selectNode(node.id)"
          >
            <div class="node-item-header">
              <span class="node-name">{{ node.name }}</span>
              <div class="node-actions" @click.stop>
                <el-switch
                  :model-value="node.enabled"
                  size="small"
                  @change="handleToggleNode(node)"
                />
                <el-button link :icon="Edit" @click="openEditNode(node)" />
                <el-button link :icon="Delete" type="danger" @click="handleDeleteNode(node)" />
              </div>
            </div>
            <div class="node-meta">
              <el-tag :type="protocolColor(node.protocol)" size="small">{{ node.protocol.toUpperCase() }}</el-tag>
              <el-tag :type="securityColor(node.security)" size="small">{{ node.security }}</el-tag>
              <span class="port-badge">:{{ node.port }}</span>
            </div>
            <div class="node-stats">
              <el-icon><User /></el-icon>
              <span>{{ node.userCount }} {{ t('xray.users') }}</span>
            </div>
          </div>
        </div>

        <!-- 右侧用户列表 -->
        <div class="user-panel">
          <div v-if="!activeNodeId" class="empty-panel">
            <el-empty :description="t('xray.selectNode')" />
          </div>

          <template v-else>
            <div class="user-panel-header">
              <span class="panel-title">
                {{ activeNode?.name }} — {{ t('xray.userManagement') }}
              </span>
              <el-button type="primary" :icon="Plus" size="small" @click="openCreateUser">
                {{ t('xray.addUser') }}
              </el-button>
            </div>

            <el-table :data="users" v-loading="usersLoading" stripe style="width: 100%">
              <el-table-column :label="t('xray.userName')" prop="name" min-width="110" />
              <el-table-column :label="t('xray.uuid')" prop="uuid" min-width="180">
                <template #default="{ row }">
                  <span class="uuid-text">{{ row.uuid }}</span>
                </template>
              </el-table-column>
              <el-table-column :label="t('xray.expireAt')" min-width="120">
                <template #default="{ row }">
                  <span v-if="!row.expireAt" class="never-expire">{{ t('xray.never') }}</span>
                  <el-tag v-else :type="isExpired(row.expireAt) ? 'danger' : 'success'" size="small">
                    {{ new Date(row.expireAt).toLocaleDateString() }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column :label="t('xray.traffic')" min-width="150">
                <template #default="{ row }">
                  <div class="traffic-cell" @click="openTrafficChart(row)" style="cursor:pointer">
                    <span class="upload">↑ {{ formatBytes(row.uploadTotal) }}</span>
                    <span class="download">↓ {{ formatBytes(row.downloadTotal) }}</span>
                  </div>
                </template>
              </el-table-column>
              <el-table-column :label="t('common.status')" width="75">
                <template #default="{ row }">
                  <el-tag :type="row.enabled ? 'success' : 'info'" size="small">
                    {{ row.enabled ? t('common.enabled') : t('common.disabled') }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column :label="t('common.actions')" width="120" fixed="right">
                <template #default="{ row }">
                  <el-button link :icon="CopyDocument" @click="handleShareLink(row)" :title="t('xray.shareLink')" />
                  <el-button link :icon="Edit" @click="openEditUser(row)" />
                  <el-button link :icon="Delete" type="danger" @click="handleDeleteUser(row)" />
                </template>
              </el-table-column>
            </el-table>

            <div class="pagination-wrap">
              <el-pagination
                v-model:current-page="userPage"
                v-model:page-size="userPageSize"
                :total="usersTotal"
                :page-sizes="[15, 30, 50]"
                layout="total, sizes, prev, pager, next"
                @change="loadUsers"
              />
            </div>
          </template>
        </div>
      </div>
    </template>

    <!-- ==================== 节点对话框 ==================== -->
    <el-dialog
      v-model="nodeDialogVisible"
      :title="nodeDialogMode === 'create' ? t('xray.addNode') : t('xray.editNode')"
      width="600px"
      destroy-on-close
    >
      <el-form :model="nodeForm" label-width="130px">
        <el-form-item :label="t('xray.nodeName')" required>
          <el-input v-model="nodeForm.name" :placeholder="t('xray.nodeNamePlaceholder')" />
        </el-form-item>
        <template v-if="nodeDialogMode === 'create'">
          <el-form-item :label="t('xray.protocol')" required>
            <el-select v-model="nodeForm.protocol" style="width:100%">
              <el-option label="VLESS" value="vless" />
              <el-option label="VMess" value="vmess" />
              <el-option label="Trojan" value="trojan" />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('xray.port')" required>
            <el-input-number v-model="nodeForm.port" :min="1" :max="65535" style="width:100%" />
          </el-form-item>
        </template>
        <el-form-item :label="t('xray.transport')">
          <el-select v-model="nodeForm.transport" style="width:100%">
            <el-option label="TCP" value="tcp" />
            <el-option label="WebSocket" value="ws" />
            <el-option label="gRPC" value="grpc" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="nodeForm.transport === 'ws'" :label="t('xray.wsPath')">
          <el-input v-model="nodeForm.path" placeholder="/ws" />
        </el-form-item>
        <el-form-item v-if="nodeForm.transport === 'grpc'" :label="t('xray.grpcService')">
          <el-input v-model="nodeForm.serviceName" placeholder="grpc" />
        </el-form-item>
        <el-form-item :label="t('xray.security')">
          <el-select v-model="nodeForm.security" style="width:100%">
            <el-option label="None" value="none" />
            <el-option label="TLS" value="tls" />
            <el-option label="Reality" value="reality" />
          </el-select>
        </el-form-item>
        <template v-if="nodeForm.security === 'reality'">
          <el-form-item :label="t('xray.realityKeys')">
            <el-button :icon="Key" :loading="generatingKeys" @click="handleGenerateKeys">
              {{ t('xray.generateKeys') }}
            </el-button>
          </el-form-item>
          <el-form-item v-if="nodeForm.realityPublicKey" :label="t('xray.publicKey')">
            <el-input :model-value="nodeForm.realityPublicKey" readonly>
              <template #append>
                <el-button :icon="CopyDocument" @click="navigator.clipboard.writeText(nodeForm.realityPublicKey!)" />
              </template>
            </el-input>
          </el-form-item>
          <el-form-item :label="t('xray.serverNames')">
            <el-input v-model="nodeForm.realityServerNames" placeholder='["www.apple.com"]' />
          </el-form-item>
          <el-form-item :label="t('xray.shortIds')">
            <el-input v-model="nodeForm.realityShortIds" placeholder='["abc123"]' />
          </el-form-item>
        </template>
        <template v-if="nodeForm.security === 'tls'">
          <el-form-item :label="t('xray.domain')">
            <el-input v-model="nodeForm.domain" placeholder="example.com" />
          </el-form-item>
          <el-form-item :label="t('xray.tlsCert')">
            <el-input v-model="nodeForm.tlsCert" :placeholder="t('xray.certPathPlaceholder')" />
          </el-form-item>
          <el-form-item :label="t('xray.tlsKey')">
            <el-input v-model="nodeForm.tlsKey" :placeholder="t('xray.keyPathPlaceholder')" />
          </el-form-item>
        </template>
        <el-form-item :label="t('common.remark')">
          <el-input v-model="nodeForm.remark" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="nodeDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="submitNodeForm">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- ==================== 用户对话框 ==================== -->
    <el-dialog
      v-model="userDialogVisible"
      :title="userDialogMode === 'create' ? t('xray.addUser') : t('xray.editUser')"
      width="460px"
      destroy-on-close
    >
      <el-form :model="userForm" label-width="100px">
        <el-form-item :label="t('xray.userName')" required>
          <el-input v-model="userForm.name" :placeholder="t('xray.userNamePlaceholder')" />
        </el-form-item>
        <el-form-item v-if="userDialogMode === 'create'" :label="t('xray.uuid')">
          <el-input v-model="userForm.uuid" :placeholder="t('xray.uuidPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('xray.expireAt')">
          <el-date-picker
            v-model="userForm.expireAt"
            type="datetime"
            :placeholder="t('xray.expirePlaceholder')"
            style="width:100%"
            value-format="YYYY-MM-DDTHH:mm:ssZ"
          />
        </el-form-item>
        <el-form-item v-if="userDialogMode === 'edit'" :label="t('common.status')">
          <el-switch v-model="userForm.enabled" />
        </el-form-item>
        <el-form-item :label="t('common.remark')">
          <el-input v-model="userForm.remark" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="userDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="submitUserForm">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- ==================== 分享链接 ==================== -->
    <el-dialog v-model="shareLinkDialog" :title="t('xray.shareLink')" width="520px">
      <el-input :model-value="shareLink" type="textarea" :rows="4" readonly />
      <template #footer>
        <el-button @click="shareLinkDialog = false">{{ t('common.close') }}</el-button>
        <el-button type="primary" :icon="CopyDocument" @click="copyShareLink">{{ t('common.copy') }}</el-button>
      </template>
    </el-dialog>

    <!-- ==================== 流量历史图表 ==================== -->
    <el-dialog
      v-model="chartDialog"
      :title="`${chartUser?.name} — ${t('xray.trafficHistory')}`"
      width="680px"
      destroy-on-close
      @closed="chartInstance?.dispose(); chartInstance = null"
    >
      <div v-loading="chartLoading" style="height: 300px">
        <div ref="chartRef" style="width:100%; height:300px" />
        <el-empty v-if="!chartLoading && !chartRef" :description="t('xray.noTrafficData')" />
      </div>
    </el-dialog>

  </div>
</template>

<style scoped lang="scss">
.xray-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* ========== 安装引导 ========== */
.install-banner {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.install-card {
  text-align: center;
  padding: 48px 40px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  max-width: 480px;
  width: 100%;

  .install-icon { margin-bottom: 16px; }
  h2 { margin: 0 0 8px; font-size: 20px; }
  p { color: var(--el-text-color-secondary); margin: 0 0 24px; font-size: 14px; line-height: 1.6; }
}

.install-progress {
  margin-top: 16px;
  .install-log {
    margin-top: 12px;
    background: var(--el-fill-color-darker, #1a1a1a);
    border-radius: 6px;
    padding: 12px;
    text-align: left;
    max-height: 180px;
    overflow-y: auto;
    pre {
      margin: 0;
      font-size: 12px;
      font-family: monospace;
      color: #a0e080;
      white-space: pre-wrap;
      word-break: break-all;
    }
  }
}

/* ========== 主界面 ========== */
.xray-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: var(--el-bg-color);
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);

  .header-left { display: flex; align-items: center; gap: 12px; }

  .status-dot {
    display: inline-block;
    width: 8px; height: 8px;
    border-radius: 50%;
    background: var(--el-color-danger);
    margin-right: 6px;
    &.running {
      background: var(--el-color-success);
      animation: pulse 2s infinite;
    }
  }

  .version-text { font-size: 12px; color: var(--el-text-color-secondary); }
}

.xray-content {
  flex: 1;
  display: flex;
  gap: 16px;
  min-height: 0;
  overflow: hidden;
}

.node-sidebar {
  width: 240px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 12px;
  overflow-y: auto;

  .sidebar-title {
    font-size: 13px;
    font-weight: 600;
    color: var(--el-text-color-secondary);
    padding: 0 4px 8px;
    border-bottom: 1px solid var(--el-border-color-lighter);
    margin-bottom: 4px;
  }
}

.node-item {
  padding: 10px 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
  border: 1px solid transparent;

  &:hover { background: var(--el-fill-color-light); }
  &.active {
    background: var(--xp-accent-muted, rgba(34, 211, 238, 0.1));
    border-color: var(--xp-accent, #22d3ee);
  }
  &.disabled { opacity: 0.5; }

  .node-item-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 6px;

    .node-name { font-size: 13px; font-weight: 500; }
    .node-actions { display: none; align-items: center; gap: 2px; }
  }

  &:hover .node-actions { display: flex; }

  .node-meta {
    display: flex;
    align-items: center;
    gap: 4px;
    margin-bottom: 4px;
    .port-badge { font-size: 11px; color: var(--el-text-color-secondary); font-family: monospace; }
  }

  .node-stats {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 11px;
    color: var(--el-text-color-placeholder);
  }
}

.user-panel {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  overflow: hidden;

  .empty-panel { flex: 1; display: flex; align-items: center; justify-content: center; }

  .user-panel-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 16px;
    border-bottom: 1px solid var(--el-border-color-lighter);
    .panel-title { font-size: 14px; font-weight: 600; }
  }
}

.traffic-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 12px;
  &:hover { text-decoration: underline; }
  .upload { color: var(--el-color-success); }
  .download { color: var(--xp-accent, #22d3ee); }
}

.uuid-text { font-size: 12px; font-family: monospace; color: var(--el-text-color-secondary); }
.never-expire { font-size: 12px; color: var(--el-text-color-placeholder); }

.pagination-wrap {
  padding: 12px 16px;
  display: flex;
  justify-content: flex-end;
  border-top: 1px solid var(--el-border-color-lighter);
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}
</style>
