<template>
  <div class="website-page">
    <div class="page-header">
      <h3>{{ $t('website.title') }}</h3>
      <el-button size="small" type="primary" @click="openCreateDialog">
        <el-icon><Plus /></el-icon>
        {{ $t('website.create') }}
      </el-button>
    </div>

    <div class="filter-bar">
      <el-input v-model="searchInfo" :placeholder="$t('commons.search')" prefix-icon="Search" size="small" clearable class="search-input" @input="loadWebsites" />
      <el-select v-model="filterType" size="small" clearable :placeholder="$t('website.type')" @change="loadWebsites">
        <el-option :label="$t('website.typeStatic')" value="static" />
        <el-option :label="$t('website.typeProxy')" value="reverse_proxy" />
      </el-select>
      <el-select v-model="filterStatus" size="small" clearable :placeholder="$t('website.status')" @change="loadWebsites">
        <el-option :label="$t('website.running')" value="running" />
        <el-option :label="$t('website.stopped')" value="stopped" />
      </el-select>
    </div>

    <el-table :data="websites" style="width: 100%" v-loading="loading">
      <el-table-column prop="primaryDomain" :label="$t('website.domain')" min-width="200">
        <template #default="{ row }">
          <div class="domain-cell">
            <el-link type="primary" @click="goConfig(row.id)">{{ row.primaryDomain }}</el-link>
            <el-tag v-if="row.sslEnable" type="success" size="small" effect="plain" class="ssl-badge">SSL</el-tag>
            <span v-if="row.domains" class="domain-extra">+{{ row.domains.split(',').length }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column :label="$t('website.type')" width="120">
        <template #default="{ row }">
          <el-tag :type="row.type === 'static' ? 'info' : 'warning'" size="small">
            {{ row.type === 'static' ? $t('website.typeStatic') : $t('website.typeProxy') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="$t('website.status')" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'running' ? 'success' : 'danger'" size="small" effect="dark" round>
            {{ row.status === 'running' ? $t('website.running') : $t('website.stopped') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="remark" :label="$t('commons.description')" min-width="120" show-overflow-tooltip />
      <el-table-column :label="$t('commons.actions')" width="280" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" size="small" @click="goConfig(row.id)">
            {{ $t('commons.edit') }}
          </el-button>
          <el-button v-if="row.status === 'stopped'" link type="success" size="small" @click="handleEnable(row)">
            {{ $t('website.enable') }}
          </el-button>
          <el-button v-else link type="warning" size="small" @click="handleDisable(row)">
            {{ $t('website.disable') }}
          </el-button>
          <el-button link type="danger" size="small" @click="handleDelete(row)">
            {{ $t('commons.delete') }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-pagination v-if="total > 0" class="mt-pagination" :current-page="page" :page-size="pageSize" :total="total" layout="total, prev, pager, next" @current-change="(p: number) => { page = p; loadWebsites() }" />

    <!-- 创建网站对话框 -->
    <el-dialog v-model="createDialogVisible" :title="$t('website.create')" width="540px" destroy-on-close>
      <el-form :model="createForm" label-width="100px">
        <el-form-item :label="$t('website.type')">
          <el-radio-group v-model="createForm.type">
            <el-radio value="static">{{ $t('website.typeStatic') }}</el-radio>
            <el-radio value="reverse_proxy">{{ $t('website.typeProxy') }}</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="$t('website.domain')">
          <el-input v-model="createForm.primaryDomain" placeholder="example.com" />
        </el-form-item>
        <el-form-item :label="$t('website.alias')">
          <el-input v-model="createForm.alias" :placeholder="createForm.primaryDomain ? createForm.primaryDomain.replace(/\./g, '_') : 'example_com'" />
          <div class="form-tip">{{ $t('website.aliasHint') }}</div>
        </el-form-item>
        <el-form-item :label="$t('website.otherDomains')">
          <el-input v-model="createForm.domains" placeholder="www.example.com" />
          <div class="form-tip">{{ $t('website.otherDomainsHint') }}</div>
        </el-form-item>
        <el-form-item v-if="createForm.type === 'static'" :label="$t('website.siteDir')">
          <div style="display:flex; gap:8px; width:100%;">
            <el-input v-model="createForm.siteDir" :placeholder="`/var/www/${effectiveAlias || 'example_com'}`" style="flex:1" />
            <el-button :icon="FolderOpened" @click="openDirBrowser('create')" />
          </div>
          <div class="form-tip">{{ $t('website.siteDirHint') }}，留空自动生成 /var/www/{{ effectiveAlias || 'alias名称' }}</div>
        </el-form-item>
        <el-form-item v-if="createForm.type === 'reverse_proxy'" :label="$t('website.proxyPass')">
          <el-input v-model="createForm.proxyPass" placeholder="http://127.0.0.1:8080" />
          <div class="form-tip">{{ $t('website.proxyPassHint') }}</div>
        </el-form-item>
        <el-form-item :label="$t('commons.description')">
          <el-input v-model="createForm.remark" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleCreate" :loading="createLoading">{{ $t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- 目录浏览器 Dialog -->
    <el-dialog v-model="dirBrowserVisible" title="选择目录" width="520px" destroy-on-close>
      <div class="dir-browser">
        <div class="dir-browser-bar">
          <el-input v-model="dirBrowserPath" size="small" style="flex:1" placeholder="输入路径" @keyup.enter="loadDirList" />
          <el-button size="small" :icon="RefreshRight" title="刷新" @click="loadDirList" />
          <el-button size="small" :icon="ArrowUp" title="上级目录" @click="goParentDir" />
        </div>
        <div class="dir-browser-list" v-loading="dirLoading">
          <div
            v-for="item in dirList"
            :key="item.path"
            class="dir-item"
            :class="{ 'dir-item--selected': dirBrowserPath === item.path }"
            @click="selectDir(item.path)"
            @dblclick="enterDir(item.path)"
          >
            <el-icon color="#f59e0b"><Folder /></el-icon>
            <span>{{ item.name }}</span>
          </div>
          <div v-if="dirList.length === 0 && !dirLoading" class="dir-empty">无子目录（双击目录进入，单击选中）</div>
        </div>
        <div class="dir-browser-current">当前选择：<code>{{ dirBrowserPath }}</code></div>
      </div>
      <template #footer>
        <el-button @click="dirBrowserVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmDirSelect">选择此目录</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { FolderOpened, RefreshRight, ArrowUp, Folder } from '@element-plus/icons-vue'
import { searchWebsite, createWebsite, deleteWebsite, enableWebsite, disableWebsite } from '@/api/modules/website'
import { listFiles } from '@/api/modules/file'
import type { Website } from '@/api/interface'

const router = useRouter()
const { t } = useI18n()

const loading = ref(false)
const websites = ref<Website[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const searchInfo = ref('')
const filterType = ref('')
const filterStatus = ref('')

const createDialogVisible = ref(false)
const createLoading = ref(false)
const createForm = ref({
  primaryDomain: '',
  alias: '',
  domains: '',
  type: 'static',
  remark: '',
  siteDir: '',
  proxyPass: '',
})

// alias 有效值：优先用填写的，其次由域名推导
const effectiveAlias = computed(() => {
  if (createForm.value.alias.trim()) return createForm.value.alias.trim()
  if (createForm.value.primaryDomain) return createForm.value.primaryDomain.replace(/[.*]/g, '_')
  return ''
})

// --- 目录浏览器 ---
const dirBrowserVisible = ref(false)
const dirBrowserPath = ref('/var/www')
const dirList = ref<{ name: string; path: string }[]>([])
const dirLoading = ref(false)
let dirBrowserTarget = '' // 'create' | 'config'

const loadDirList = async () => {
  dirLoading.value = true
  try {
    const res = await listFiles({ path: dirBrowserPath.value, showHidden: false })
    const items = res.data?.items || []
    dirList.value = items
      .filter((f: any) => f.isDir)
      .map((f: any) => ({ name: f.name, path: f.path }))
  } catch {
    dirList.value = []
  } finally {
    dirLoading.value = false
  }
}

const openDirBrowser = (target: string) => {
  dirBrowserTarget = target
  // 用当前输入的路径初始化，若为空则默认 /var/www
  const cur = target === 'create' ? createForm.value.siteDir : ''
  dirBrowserPath.value = cur || '/var/www'
  dirBrowserVisible.value = true
  loadDirList()
}

const enterDir = (path: string) => {
  dirBrowserPath.value = path
  loadDirList()
}

const goParentDir = () => {
  const parts = dirBrowserPath.value.split('/').filter(Boolean)
  parts.pop()
  dirBrowserPath.value = '/' + parts.join('/')
  loadDirList()
}

const selectDir = (path: string) => {
  dirBrowserPath.value = path
}

const confirmDirSelect = () => {
  if (dirBrowserTarget === 'create') {
    createForm.value.siteDir = dirBrowserPath.value
  }
  dirBrowserVisible.value = false
}
// --- /目录浏览器 ---

const loadWebsites = async () => {
  loading.value = true
  try {
    const res = await searchWebsite({
      page: page.value,
      pageSize: pageSize.value,
      info: searchInfo.value,
      type: filterType.value,
      status: filterStatus.value,
    })
    websites.value = res.data?.items || []
    total.value = res.data?.total || 0
  } catch { websites.value = [] }
  finally { loading.value = false }
}

const openCreateDialog = () => {
  createForm.value = { primaryDomain: '', alias: '', domains: '', type: 'static', remark: '', siteDir: '', proxyPass: '' }
  createDialogVisible.value = true
}

const handleCreate = async () => {
  if (!createForm.value.primaryDomain) { ElMessage.warning('请输入域名'); return }
  if (createForm.value.type === 'reverse_proxy' && !createForm.value.proxyPass) { ElMessage.warning('请输入代理地址'); return }
  createLoading.value = true
  try {
    await createWebsite(createForm.value)
    ElMessage.success(t('commons.success'))
    createDialogVisible.value = false
    loadWebsites()
  } catch {}
  finally { createLoading.value = false }
}

const handleEnable = async (row: Website) => {
  try {
    await ElMessageBox.confirm(t('website.enableConfirm'), t('commons.tip'), { type: 'info' })
    await enableWebsite(row.id)
    ElMessage.success(t('commons.success'))
    loadWebsites()
  } catch {}
}

const handleDisable = async (row: Website) => {
  try {
    await ElMessageBox.confirm(t('website.disableConfirm'), t('commons.tip'), { type: 'warning' })
    await disableWebsite(row.id)
    ElMessage.success(t('commons.success'))
    loadWebsites()
  } catch {}
}

const handleDelete = async (row: Website) => {
  try {
    await ElMessageBox.confirm(t('website.deleteConfirm'), t('commons.tip'), { type: 'error' })
    await deleteWebsite(row.id)
    ElMessage.success(t('commons.success'))
    loadWebsites()
  } catch {}
}

const goConfig = (id: number) => {
  router.push(`/website/websites/${id}`)
}

onMounted(() => loadWebsites())
</script>

<style lang="scss" scoped>
.website-page {
  height: 100%;
}

.filter-bar {
  display: flex;
  gap: 10px;
  margin-bottom: 12px;

  .search-input {
    width: 240px;
  }
}

.mt-pagination {
  margin-top: 16px;
  justify-content: flex-end;
}

.domain-cell {
  display: flex;
  align-items: center;
  gap: 6px;

  .ssl-badge {
    font-size: 10px;
    padding: 0 4px;
    height: 18px;
    line-height: 18px;
  }

  .domain-extra {
    font-size: 11px;
    padding: 1px 5px;
    border-radius: 3px;
    background: var(--xp-accent-muted);
    color: var(--xp-accent);
  }
}

.form-tip {
  font-size: 12px;
  color: var(--xp-text-muted);
  margin-top: 4px;
}

.dir-browser {
  display: flex;
  flex-direction: column;
  gap: 10px;

  .dir-browser-bar {
    display: flex;
    gap: 6px;
    align-items: center;
  }

  .dir-browser-list {
    height: 260px;
    overflow-y: auto;
    border: 1px solid var(--el-border-color);
    border-radius: 6px;
    padding: 4px;

    .dir-item {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 6px 10px;
      border-radius: 4px;
      cursor: pointer;
      font-size: 13px;
      user-select: none;
      transition: background 0.15s;

      &:hover { background: var(--el-fill-color-light); }
      &--selected { background: var(--el-color-primary-light-9); color: var(--el-color-primary); }
    }

    .dir-empty {
      text-align: center;
      color: var(--xp-text-muted);
      font-size: 13px;
      padding: 40px 0;
    }
  }

  .dir-browser-current {
    font-size: 12px;
    color: var(--xp-text-muted);
    code { font-size: 12px; color: var(--el-color-primary); }
  }
}
</style>
