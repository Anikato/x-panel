<template>
  <div class="header">
    <div class="header-left">
      <div class="collapse-btn" @click="globalStore.toggleMenuCollapse">
        <el-icon :size="18">
          <Fold v-if="!globalStore.menuCollapse" />
          <Expand v-else />
        </el-icon>
      </div>
      <el-breadcrumb separator="/">
        <el-breadcrumb-item v-for="item in breadcrumbs" :key="item.path">
          {{ item.title }}
        </el-breadcrumb-item>
      </el-breadcrumb>
    </div>
    <div class="header-right">
      <el-select
        v-model="currentNode"
        size="small"
        style="width: 160px; margin-right: 4px"
        @change="onNodeChange"
      >
        <el-option :label="t('node.local')" :value="0" />
        <el-option v-for="n in nodes" :key="n.id" :label="n.name" :value="n.id" />
      </el-select>

      <!-- 主题色选择 -->
      <el-popover placement="bottom" :width="240" trigger="click" :show-arrow="true">
        <template #reference>
          <div class="theme-btn">
            <div class="accent-dot" :style="{ background: currentAccentColor }"></div>
          </div>
        </template>
        <div class="accent-panel">
          <div class="accent-section">
            <div class="accent-panel-title">{{ t('header.accentColor') }}</div>
            <div class="accent-grid">
              <div
                v-for="preset in ACCENT_PRESETS"
                :key="preset.key"
                class="accent-swatch"
                :class="{ active: globalStore.accentKey === preset.key }"
                :style="{ background: preset.primary }"
                :title="preset.name"
                @click="selectAccent(preset.key)"
              >
                <el-icon v-if="globalStore.accentKey === preset.key" :size="12"><Check /></el-icon>
              </div>
            </div>
          </div>
          <div class="accent-custom-row">
            <span class="accent-custom-label">{{ t('header.customColor') }}</span>
            <input
              type="color"
              class="accent-color-input"
              :value="globalStore.accentCustom || '#22d3ee'"
              @input="onCustomColor"
            />
          </div>
        </div>
      </el-popover>

      <!-- 深浅模式切换 -->
      <el-tooltip :content="themeLabel" placement="bottom">
        <div class="theme-btn" @click="globalStore.cycleTheme()">
          <el-icon :size="16">
            <Moon v-if="globalStore.theme === 'dark'" />
            <Sunny v-else-if="globalStore.theme === 'light'" />
            <Monitor v-else />
          </el-icon>
        </div>
      </el-tooltip>

      <el-dropdown @command="handleCommand" trigger="click">
        <div class="user-dropdown">
          <div class="user-avatar">
            <el-icon :size="14"><UserFilled /></el-icon>
          </div>
          <span class="username">{{ userStore.name || 'admin' }}</span>
          <el-icon :size="12" class="arrow"><ArrowDown /></el-icon>
        </div>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="password">
              <el-icon><Lock /></el-icon>{{ t('header.changePassword') }}
            </el-dropdown-item>
            <el-dropdown-item command="logout" divided>
              <el-icon><SwitchButton /></el-icon>{{ t('header.logout') }}
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessageBox, ElMessage } from 'element-plus'
import { useGlobalStore } from '@/store/modules/global'
import { useUserStore } from '@/store/modules/user'
import { logout as logoutApi } from '@/api/modules/auth'
import { listNodes } from '@/api/modules/node'
import { useI18n } from 'vue-i18n'
import type { NodeItem } from '@/api/interface'
import { Moon, Sunny, Check } from '@element-plus/icons-vue'
import { ACCENT_PRESETS, getPresetByKey, applyAccentPalette, generatePaletteFromHex } from '@/utils/accent-colors'

const route = useRoute()
const router = useRouter()
const globalStore = useGlobalStore()
const userStore = useUserStore()
const { t } = useI18n()

const themeLabel = computed(() => {
  const labels = { dark: t('header.themeDark'), light: t('header.themeLight'), auto: t('header.themeAuto') }
  return labels[globalStore.theme] || labels.dark
})

const currentAccentColor = computed(() => {
  if (globalStore.accentKey === 'custom' && globalStore.accentCustom) return globalStore.accentCustom
  return getPresetByKey(globalStore.accentKey)?.primary || '#22d3ee'
})

const selectAccent = (key: string) => {
  globalStore.setAccent(key)
  const preset = getPresetByKey(key)
  if (preset) applyAccentPalette(preset)
}

const onCustomColor = (e: Event) => {
  const hex = (e.target as HTMLInputElement).value
  globalStore.setAccent('custom', hex)
  applyAccentPalette(generatePaletteFromHex(hex))
}

const nodes = ref<NodeItem[]>([])
const currentNode = ref(globalStore.currentNodeID || 0)

const loadNodes = async () => {
  try {
    const res = await listNodes()
    nodes.value = res.data || []
  } catch { /* ignore */ }
}

const onNodeChange = (val: number) => {
  const node = nodes.value.find((n: NodeItem) => n.id === val)
  globalStore.setCurrentNode(val, node ? node.name : '')
  window.location.reload()
}

onMounted(() => loadNodes())

const breadcrumbs = computed(() => {
  return route.matched
    .filter((item) => item.meta?.title)
    .map((item) => ({
      path: item.path,
      title: t(item.meta.title as string),
    }))
})

const handleCommand = async (command: string) => {
  if (command === 'logout') {
    try {
      await ElMessageBox.confirm(t('header.logoutConfirm'), t('commons.tip'), {
        type: 'warning',
        confirmButtonText: t('commons.confirm'),
        cancelButtonText: t('commons.cancel'),
      })
      await logoutApi()
      userStore.logout()
      globalStore.setLogin(false)
      router.push('/login')
    } catch {
      // cancelled
    }
  } else if (command === 'password') {
    router.push('/setting')
  }
}
</script>

<style lang="scss" scoped>
.header {
  height: var(--xp-header-height);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  background: var(--xp-bg-header);
  backdrop-filter: blur(16px) saturate(1.8);
  border-bottom: 1px solid var(--xp-border-light);
  flex-shrink: 0;
  position: relative;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;

  .collapse-btn {
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--xp-radius-sm);
    color: var(--xp-text-secondary);
    cursor: pointer;
    transition: all 0.2s;

    &:hover {
      background: var(--xp-accent-muted);
      color: var(--xp-accent);
    }
  }
}

.header-right {
  display: flex;
  align-items: center;
  gap: 8px;

  .theme-btn {
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--xp-radius-sm);
    color: var(--xp-text-secondary);
    cursor: pointer;
    transition: all 0.2s;

    &:hover {
      background: var(--xp-accent-muted);
      color: var(--xp-accent);
    }

    .accent-dot {
      width: 18px;
      height: 18px;
      border-radius: 50%;
      border: 2px solid rgba(255, 255, 255, 0.2);
      transition: all 0.2s;
    }
  }

  .user-dropdown {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    padding: 4px 10px;
    border-radius: var(--xp-radius-sm);
    transition: all 0.2s;

    &:hover {
      background: var(--xp-accent-muted);
    }

    .user-avatar {
      width: 28px;
      height: 28px;
      display: flex;
      align-items: center;
      justify-content: center;
      background: linear-gradient(135deg, var(--xp-accent), var(--xp-accent-secondary));
      border-radius: 50%;
      color: #0b0e14;
    }

    .username {
      font-size: 13px;
      color: var(--xp-text-secondary);
      max-width: 100px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .arrow {
      color: var(--xp-text-muted);
    }
  }
}
</style>

<style lang="scss">
.accent-panel {
  .accent-section {
    margin-bottom: 12px;
  }

  .accent-panel-title {
    font-size: 12px;
    font-weight: 600;
    color: var(--xp-text-muted);
    letter-spacing: 0.5px;
    margin-bottom: 10px;
  }

  .accent-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 8px;
    justify-items: center;
  }

  .accent-swatch {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #fff;
    transition: all 0.2s;
    border: 2px solid transparent;
    flex-shrink: 0;

    &:hover {
      transform: scale(1.15);
    }

    &.active {
      border-color: var(--xp-text-primary);
      box-shadow: 0 0 0 2px var(--xp-bg-surface), 0 0 0 3px var(--xp-accent);
    }
  }

  .accent-custom-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding-top: 10px;
    border-top: 1px solid var(--xp-border-light);
  }

  .accent-custom-label {
    font-size: 12px;
    color: var(--xp-text-secondary);
  }

  .accent-color-input {
    width: 32px;
    height: 28px;
    border: 1px solid var(--xp-border);
    border-radius: 6px;
    padding: 2px;
    background: transparent;
    cursor: pointer;

    &::-webkit-color-swatch-wrapper { padding: 2px; }
    &::-webkit-color-swatch { border-radius: 4px; border: none; }
  }
}
</style>
