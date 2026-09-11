<template>
  <div class="theme-gallery">
    <div class="page-header">
      <h3>{{ t('setting.themeGallery') }}</h3>
      <div class="header-actions">
        <el-radio-group :model-value="appearance.preference.themeId" @change="(val: ThemeId) => appearance.applyImmediate({ themeId: val })">
          <el-radio-button v-for="theme in themes" :key="theme.id" :value="theme.id">{{ theme.name }}</el-radio-button>
        </el-radio-group>
        <el-radio-group :model-value="appearance.preference.mode" @change="(val: ThemeMode) => appearance.applyImmediate({ mode: val })">
          <el-radio-button value="dark">深色</el-radio-button>
          <el-radio-button value="light">浅色</el-radio-button>
        </el-radio-group>
        <el-radio-group :model-value="density" @change="(val: Density) => appearance.applyImmediate({ overridesByTheme: { [appearance.preference.themeId]: { ...currentOver, density: val } } })">
          <el-radio-button value="compact">紧凑</el-radio-button>
          <el-radio-button value="default">标准</el-radio-button>
          <el-radio-button value="comfortable">舒适</el-radio-button>
        </el-radio-group>
      </div>
    </div>

    <el-alert title="开发用规范展示页，不进入主导航。切换主题不应刷新或丢失本页状态。" type="info" show-icon :closable="false" class="mb-16" />

    <el-card class="gallery-card">
      <template #header>按钮 / 输入 / 标签</template>
      <div class="gallery-row">
        <el-button type="primary">主要操作</el-button>
        <el-button>次要操作</el-button>
        <el-button type="danger">危险</el-button>
        <el-button disabled>禁用</el-button>
        <el-button type="primary" loading>加载</el-button>
        <el-input placeholder="搜索域名或路径" class="xp-select-md" />
        <el-select placeholder="选择" class="xp-select-sm">
          <el-option label="选项 A" value="a" />
          <el-option label="选项 B" value="b" />
        </el-select>
        <el-switch />
        <el-tag type="success">运行中</el-tag>
        <el-tag type="danger">已停止</el-tag>
        <el-tag type="warning">即将过期</el-tag>
        <el-tag type="info">静态</el-tag>
      </div>
    </el-card>

    <el-card class="gallery-card">
      <template #header>表格 / 空状态 / 分页</template>
      <el-table :data="rows" style="width: 100%">
        <el-table-column label="站点" min-width="220">
          <template #default="{ row }">{{ row.domain }}</template>
        </el-table-column>
        <el-table-column label="路径" min-width="280">
          <template #default="{ row }"><code class="mono-text">{{ row.path }}</code></template>
        </el-table-column>
        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="row.ok ? 'success' : 'danger'" size="small">{{ row.ok ? '运行中' : '已停止' }}</el-tag>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination class="mt-pagination" layout="total, prev, pager, next" :total="36" :page-size="10" />
      <el-empty description="没有符合筛选条件的数据" />
    </el-card>

    <el-card class="gallery-card">
      <template #header>浮层 / 对话框 / 提示</template>
      <div class="gallery-row">
        <el-button @click="dialog = true">打开弹窗</el-button>
        <el-dropdown>
          <el-button>下拉菜单</el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item>编辑</el-dropdown-item>
              <el-dropdown-item divided>删除</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-tooltip content="这是工具提示">
          <el-button>悬停提示</el-button>
        </el-tooltip>
        <el-button @click="notify">通知</el-button>
        <el-button @click="alertBox">确认框</el-button>
      </div>
    </el-card>

    <el-dialog v-model="dialog" title="挂载到 body 的弹窗" append-to-body width="420px">
      <p>该对话框用于检查主题变量是否作用到 body 挂载层。</p>
      <el-input v-model="formText" placeholder="表单状态应在切换主题后保留" />
      <template #footer>
        <el-button @click="dialog = false">取消</el-button>
        <el-button type="primary" @click="dialog = false">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { ElMessageBox, ElNotification } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { listThemes } from '@/theme'
import { useAppearanceStore } from '@/store/modules/appearance'
import type { Density, ThemeId, ThemeMode } from '@/theme/types.ts'

const { t } = useI18n()
const appearance = useAppearanceStore()
const themes = listThemes()
const dialog = ref(false)
const formText = ref('example.com')
const rows = [
  { domain: 'very-long-subdomain.example-site.com', path: '/var/www/very/long/path/to/current/release', ok: true },
  { domain: '中文站点.example', path: '/data/site/中文目录', ok: false },
]

const currentOver = computed(() => appearance.preference.overridesByTheme[appearance.preference.themeId] || {})
const density = computed(() => currentOver.value.density || 'default')

const notify = () => ElNotification({ title: '通知', message: '主题切换后通知样式应跟随语义色', type: 'success' })
const alertBox = () => ElMessageBox.confirm('确认执行该操作？', '提示', { type: 'warning' }).catch(() => undefined)
</script>

<style scoped lang="scss">
.theme-gallery {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.gallery-card :deep(.el-card__body) {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.gallery-row {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
}

.mb-16 { margin-bottom: 16px; }
.mt-pagination { margin-top: 12px; }
</style>
