<template>
  <div class="module-chrome" :class="{ 'is-side': isSide }">
    <div v-if="chromeHeading.show" class="module-heading">
      <h2>{{ t(chromeHeading.titleKey) }}</h2>
    </div>
    <div v-if="partitions.length" class="module-partitions">
      <div class="module-tabs-track">
        <router-link
          v-for="partition in partitions"
          :key="partition.id"
          class="partition-link"
          :class="{ active: activePartition?.id === partition.id }"
          :to="partitionLink(partition)"
        >
          {{ t(partition.titleKey) }}
        </router-link>
      </div>
    </div>

    <nav v-if="isTabs" class="module-tabs" aria-label="模块导航">
      <div class="module-tabs-track">
        <router-link
          v-for="item in tabItems"
          :key="item.id"
          class="module-tab"
          :class="{ active: isItemActive(item) }"
          :to="{ path: item.path, query: mergedQuery(item.query) }"
        >
          {{ t(item.titleKey) }}
        </router-link>
      </div>
    </nav>

    <div class="module-chrome-body">
      <aside v-if="isSide" class="module-side" aria-label="模块页面">
        <template v-for="item in sideItems" :key="item.id">
          <div v-if="item.children?.length" class="side-group">
            <div class="side-group-title">{{ t(item.titleKey) }}</div>
            <router-link
              v-for="child in item.children"
              :key="child.id"
              class="side-link"
              :class="{ active: isItemActive(child) }"
              :to="{ path: child.path, query: child.query }"
            >
              {{ t(child.titleKey) }}
            </router-link>
          </div>
          <router-link
            v-else
            class="side-link"
            :class="{ active: isItemActive(item) }"
            :to="{ path: item.path, query: item.query }"
          >
            {{ t(item.titleKey) }}
          </router-link>
        </template>
      </aside>
      <div class="module-chrome-main">
        <slot />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  findLocalNavItem,
  hideLocalNav,
  locationMatches,
  resolveChromeHeading,
  resolveModule,
  resolveNetworkPartition,
  type LocalNavItem,
  type NavPartition,
} from '@/navigation/registry'

const route = useRoute()
const { t } = useI18n()

const currentModule = computed(() => resolveModule(route.path))
const hidden = computed(() => hideLocalNav(currentModule.value, route.path))
const isTabs = computed(() => !hidden.value && currentModule.value?.localNav === 'tabs')
const isSide = computed(() => !hidden.value && currentModule.value?.localNav === 'side')

const queryRecord = computed(() => {
  const query: Record<string, string> = {}
  for (const [key, value] of Object.entries(route.query)) {
    if (typeof value === 'string') query[key] = value
  }
  return query
})

const partitions = computed(() => (!hidden.value && currentModule.value?.partitions) || [])
const activePartition = computed(() => resolveNetworkPartition(route.path))
const tabItems = computed(() => currentModule.value?.items || [])
const sideItems = computed(() => {
  if (partitions.value.length) return activePartition.value?.items || []
  return currentModule.value?.items || []
})

const isItemActive = (item: LocalNavItem): boolean => {
  if (item.children?.length) return item.children.some((child) => isItemActive(child))
  return locationMatches(item, route.path, queryRecord.value)
    || (!item.query && findLocalNavItem(route.path, queryRecord.value)?.id === item.id)
}

const mergedQuery = (query?: Record<string, string>) => {
  if (!query) return undefined
  return { ...queryRecord.value, ...query }
}

const partitionLink = (partition: NavPartition) => {
  const current = partition.items.find((item) => item.path === route.path)
  return { path: current?.path || partition.defaultPath }
}

const chromeHeading = computed(() => resolveChromeHeading(route.path, queryRecord.value))
</script>

<style lang="scss" scoped>
.module-chrome {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
}

.module-heading {
  flex-shrink: 0;
  padding: 18px 24px 10px;

  h2 {
    margin: 0;
    color: var(--xp-text-primary);
    font-weight: 650;
    line-height: 1.2;
  }
}

.module-partitions,
.module-tabs {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  padding: 8px 24px 0;
}

.module-tabs-track {
  display: inline-flex;
  align-items: center;
}

.partition-link,
.module-tab {
  display: inline-flex;
  align-items: center;
  min-height: 32px;
  padding: 0 12px;
  color: var(--xp-text-secondary);
  font-size: 13px;
  border-radius: 0;
  text-decoration: none;

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

.module-chrome-body {
  display: flex;
  flex: 1;
  min-height: 0;
}

.module-side {
  flex-shrink: 0;
  width: 180px;
  padding: 16px 10px;
  overflow: auto;
  border-right: 1px solid var(--xp-border);
}

.side-group {
  margin-bottom: 10px;
}

.side-group-title {
  padding: 6px 10px;
  color: var(--xp-text-muted);
  font-size: 11px;
  font-weight: 650;
}

.side-link {
  display: flex;
  align-items: center;
  min-height: 32px;
  padding: 0 10px;
  color: var(--xp-text-secondary);
  font-size: 13px;
  border-radius: var(--xp-radius-sm);
  text-decoration: none;

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

.module-chrome-main {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
}

@media (max-width: 1100px) {
  .module-side {
    width: 156px;
  }
}
</style>
