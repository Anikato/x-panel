<template>
  <div class="sidebar" :class="{ 'is-collapse': globalStore.menuCollapse }">
    <div class="sidebar-logo">
      <div class="logo-icon">
        <XPanelLogo />
      </div>
      <span v-if="!globalStore.menuCollapse" class="logo-text">
        {{ globalStore.panelName || 'X-Panel' }}
      </span>
    </div>

    <el-scrollbar class="sidebar-menu-scroll">
      <nav class="nav-tree" aria-label="主导航">
        <section v-for="group in groups" :key="group.id" class="nav-group">
          <button
            v-if="!globalStore.menuCollapse"
            type="button"
            class="nav-group-title"
            :class="{ 'has-active': groupCollapsed[group.id] && groupHasActive(group.id) }"
            :aria-expanded="!groupCollapsed[group.id]"
            @click="toggleGroup(group.id)"
          >
            <span>{{ t(group.titleKey) }}</span>
            <el-icon class="group-caret" :class="{ collapsed: groupCollapsed[group.id] }"><ArrowDown /></el-icon>
          </button>
          <div v-show="globalStore.menuCollapse || !groupCollapsed[group.id]" class="nav-group-items">
            <router-link
              v-for="mod in modulesIn(group.id)"
              :key="mod.id"
              :to="toLocation(mod)"
              class="nav-item"
              :class="{ active: isActive(mod) }"
              :title="globalStore.menuCollapse ? t(mod.titleKey) : undefined"
            >
              <el-icon><component :is="iconOf(mod)" /></el-icon>
              <span v-if="!globalStore.menuCollapse" class="nav-item-label">{{ t(mod.titleKey) }}</span>
            </router-link>
          </div>
        </section>
      </nav>
    </el-scrollbar>

    <div class="sidebar-footer">
      <router-link
        v-for="mod in footerModules"
        :key="mod.id"
        :to="toLocation(mod)"
        class="nav-item"
        :class="{ active: isActive(mod) }"
        :title="globalStore.menuCollapse ? t(mod.titleKey) : undefined"
      >
        <el-icon><component :is="mod.icon" /></el-icon>
        <span v-if="!globalStore.menuCollapse" class="nav-item-label">{{ t(mod.titleKey) }}</span>
      </router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useGlobalStore } from '@/store/modules/global'
import {
  NAV_GROUPS,
  SIDEBAR_FOOTER_MODULES,
  SIDEBAR_MAIN_MODULES,
  resolveModule,
  type NavGroupId,
  type NavModule,
} from '@/navigation/registry'
import { mapNavIcon } from '@/theme'
import { useAppearanceStore } from '@/store/modules/appearance'
import ShieldIcon from '@/components/icons/ShieldIcon.vue'
import XPanelLogo from '@/components/brand/XPanelLogo.vue'
import { destinationFor, readGroupCollapsed, rememberCurrentRoute, writeGroupCollapsed } from '@/navigation/session'

const route = useRoute()
const globalStore = useGlobalStore()
const appearanceStore = useAppearanceStore()
const { t } = useI18n()

const groups = NAV_GROUPS
const footerModules = SIDEBAR_FOOTER_MODULES
const groupCollapsed = reactive<Partial<Record<NavGroupId, boolean>>>(readGroupCollapsed())

const currentModule = computed(() => resolveModule(route.path))

const queryRecord = computed(() => {
  const query: Record<string, string> = {}
  for (const [key, value] of Object.entries(route.query)) {
    if (typeof value === 'string') query[key] = value
  }
  return query
})

watch(
  () => [route.path, route.query] as const,
  () => rememberCurrentRoute(route.path, queryRecord.value),
  { immediate: true },
)

const modulesIn = (groupId: NavGroupId) => SIDEBAR_MAIN_MODULES.filter((mod) => mod.group === groupId)

const groupHasActive = (groupId: NavGroupId) => currentModule.value?.group === groupId

const isActive = (mod: NavModule) => currentModule.value?.id === mod.id

const toLocation = (mod: NavModule) => {
  const dest = destinationFor(mod)
  return { path: dest.path, query: dest.query }
}

const extraIcons: Record<string, unknown> = { ShieldIcon }
const iconOf = (mod: NavModule) => {
  const name = mapNavIcon(mod.icon, appearanceStore.resolved.iconSet)
  return extraIcons[name] || name
}

const toggleGroup = (groupId: NavGroupId) => {
  groupCollapsed[groupId] = !groupCollapsed[groupId]
  writeGroupCollapsed({ ...groupCollapsed })
}
</script>

<style lang="scss" scoped>
.sidebar {
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  z-index: 1001;
  display: flex;
  flex-direction: column;
  width: var(--xp-sidebar-width);
  overflow: hidden;
  background: var(--xp-bg-sidebar);
  border-right: 1px solid var(--xp-border);
  transition: width 180ms cubic-bezier(0.2, 0, 0, 1);

  &.is-collapse {
    width: var(--xp-sidebar-collapse-width);
  }
}

.sidebar-logo {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  height: var(--xp-header-height);
  padding: 0 16px;
  gap: 10px;
  border-bottom: 1px solid var(--xp-border);

  .logo-icon {
    display: flex;
    flex-shrink: 0;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    padding: 3px;
    background: var(--xp-accent-muted);
    border: 1px solid var(--xp-border-light);
    border-radius: var(--xp-radius-sm);
  }

  .logo-text {
    overflow: hidden;
    color: var(--xp-text-primary);
    font-size: 15px;
    font-weight: 650;
    letter-spacing: -0.02em;
    white-space: nowrap;
  }
}

.sidebar-menu-scroll {
  flex: 1;
  overflow: hidden;
}

.nav-tree {
  padding: 10px 8px 16px;
}

.nav-group {
  margin-bottom: 8px;
}

.nav-group-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  margin: 4px 0 2px;
  padding: 6px 10px;
  color: var(--xp-text-muted);
  font-size: 11px;
  font-weight: 650;
  letter-spacing: 0.06em;
  text-transform: none;
  background: transparent;
  border: 0;
  border-radius: var(--xp-radius-sm);
  cursor: pointer;

  &:hover {
    color: var(--xp-text-secondary);
    background: transparent;
  }

  &.has-active {
    color: var(--xp-accent);
  }

  .group-caret {
    font-size: 12px;
    transition: transform 160ms cubic-bezier(0.2, 0, 0, 1);

    &.collapsed {
      transform: rotate(-90deg);
    }
  }
}

.nav-item {
  display: flex;
  align-items: center;
  min-height: 36px;
  margin: 1px 0;
  padding: 0 10px;
  gap: 10px;
  color: var(--xp-text-secondary);
  font-size: 13.5px;
  line-height: 1.2;
  border-radius: var(--xp-radius-sm);
  text-decoration: none;
  transition:
    color var(--xp-motion-hover, 100ms) var(--xp-motion-ease, cubic-bezier(0.2, 0, 0, 1)),
    background-color var(--xp-motion-hover, 100ms) var(--xp-motion-ease, cubic-bezier(0.2, 0, 0, 1)),
    box-shadow var(--xp-motion-hover, 100ms) var(--xp-motion-ease, cubic-bezier(0.2, 0, 0, 1));

  .el-icon {
    flex-shrink: 0;
    width: 20px;
    height: 20px;
    color: inherit;
    font-size: 18px;

    :deep(svg) {
      display: block;
      width: 18px;
      height: 18px;
    }
  }

  &:hover {
    color: var(--xp-text-primary);
    background: var(--xp-accent-muted);
  }

  &.active {
    color: var(--xp-accent);
    font-weight: 600;
    background: var(--xp-accent-muted);
  }
}

:global(html[data-sidebar-variant='marker']) .nav-item.active {
  background: transparent;
  box-shadow: inset 2px 0 0 var(--xp-accent);
  border-radius: 0 var(--xp-radius-sm) var(--xp-radius-sm) 0;
}

:global(html[data-sidebar-variant='block']) .nav-item {
  border-radius: var(--xp-radius-sm);
}

:global(html[data-sidebar-variant='block']) .nav-item.active {
  color: var(--xp-on-accent);
  background: var(--xp-accent);
}

.nav-item-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sidebar-footer {
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  padding: 8px;
  gap: 2px;
  border-top: 1px solid var(--xp-border);
}

@media (max-width: 900px) {
  .sidebar {
    width: min(var(--xp-sidebar-width), 82vw);
    box-shadow: 0 24px 60px rgba(0, 0, 0, 0.45);
    transform: translateX(0);
    transition: transform 180ms cubic-bezier(0.2, 0, 0, 1);

    &.is-collapse {
      width: min(var(--xp-sidebar-width), 82vw);
      box-shadow: none;
      transform: translateX(-100%);
    }
  }
}
</style>
