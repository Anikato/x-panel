<template>
  <div v-if="analysis" class="site-log-insights">
    <el-row :gutter="16" class="insight-row">
      <el-col :xs="24" :md="12">
        <el-card shadow="never">
          <template #header>热门 URL</template>
          <el-table :data="analysis.topUrls || []" size="small" stripe max-height="280">
            <el-table-column label="URL" show-overflow-tooltip>
              <template #default="{ row }">
                <button type="button" class="linkish" @click="openDrill('url', row.name)">{{ row.name }}</button>
              </template>
            </el-table-column>
            <el-table-column label="次数" prop="count" width="80" align="right" />
          </el-table>
        </el-card>
      </el-col>
      <el-col :xs="24" :md="12">
        <el-card shadow="never">
          <template #header>热门 IP</template>
          <el-table :data="analysis.topIps || []" size="small" stripe max-height="280">
            <el-table-column label="IP">
              <template #default="{ row }">
                <button type="button" class="linkish" @click="openDrill('ip', row.name)">{{ row.name }}</button>
              </template>
            </el-table-column>
            <el-table-column label="位置" min-width="100">
              <template #default="{ row }">{{ row.country || '-' }}<template v-if="row.city"> / {{ row.city }}</template></template>
            </el-table-column>
            <el-table-column label="次数" prop="count" width="80" align="right" />
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="insight-row">
      <el-col :xs="24" :md="12">
        <el-card shadow="never">
          <template #header>具体状态码</template>
          <el-table :data="statusRows" size="small" stripe max-height="240">
            <el-table-column label="状态码" prop="name" />
            <el-table-column label="次数" prop="count" width="90" align="right" />
          </el-table>
        </el-card>
      </el-col>
      <el-col :xs="24" :md="12">
        <el-card shadow="never">
          <template #header>错误日志</template>
          <el-table :data="analysis.errorSummary || []" size="small" stripe max-height="240">
            <el-table-column label="原因" prop="name" show-overflow-tooltip />
            <el-table-column label="次数" prop="count" width="90" align="right" />
          </el-table>
          <el-empty v-if="!(analysis.errorSummary || []).length" description="这段时间没有归类到的错误" :image-size="48" />
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="insight-row">
      <el-col :xs="24" :md="12">
        <el-card shadow="never">
          <template #header>慢请求</template>
          <div class="slow-summary">平均 {{ formatMs(analysis.avgRequestMs) }}，超过 1 秒 {{ analysis.slowRequests || 0 }} 次</div>
          <el-table :data="analysis.topSlowUrls || []" size="small" stripe max-height="240">
            <el-table-column label="URL" prop="name" show-overflow-tooltip />
            <el-table-column label="最慢" width="90" align="right">
              <template #default="{ row }">{{ formatMs(row.maxMs) }}</template>
            </el-table-column>
            <el-table-column label="次数" prop="count" width="70" align="right" />
          </el-table>
        </el-card>
      </el-col>
      <el-col :xs="24" :md="12">
        <el-card shadow="never">
          <template #header>User-Agent</template>
          <el-table :data="analysis.topUserAgents || []" size="small" stripe max-height="260">
            <el-table-column label="User-Agent" prop="name" show-overflow-tooltip />
            <el-table-column label="次数" prop="count" width="80" align="right" />
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-row v-if="(analysis.threatRequests || 0) > 0" :gutter="16" class="insight-row">
      <el-col :xs="24" :md="10">
        <el-card shadow="never">
          <template #header>攻击类型</template>
          <el-table :data="analysis.topThreats || []" size="small" stripe>
            <el-table-column label="类型">
              <template #default="{ row }">
                <button type="button" class="linkish" @click="openDrill('threat', row.name)">{{ row.name }}</button>
              </template>
            </el-table-column>
            <el-table-column label="次数" prop="count" width="80" align="right" />
          </el-table>
        </el-card>
      </el-col>
      <el-col :xs="24" :md="14">
        <el-card shadow="never">
          <template #header>爬虫</template>
          <el-table :data="analysis.topCrawlers || []" size="small" stripe>
            <el-table-column label="名称" prop="name" />
            <el-table-column label="次数" prop="count" width="80" align="right" />
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-dialog v-model="visible" :title="title" width="720px" destroy-on-close>
      <el-table v-if="urls.length" :data="urls" size="small" stripe max-height="220">
        <el-table-column label="URL" prop="name" show-overflow-tooltip />
        <el-table-column label="次数" prop="count" width="80" align="right" />
      </el-table>
      <el-table :data="ips" size="small" stripe max-height="320">
        <el-table-column label="IP" prop="name" />
        <el-table-column label="位置">
          <template #default="{ row }">{{ row.country || '-' }}<template v-if="row.city"> / {{ row.city }}</template></template>
        </el-table-column>
        <el-table-column label="次数" prop="count" width="80" align="right" />
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { drilldownNginxLog } from '@/api/modules/website'

const props = defineProps<{
  analysis: any
  siteId: number
  days: number
}>()

const visible = ref(false)
const title = ref('')
const ips = ref<any[]>([])
const urls = ref<any[]>([])

const statusRows = computed(() => {
  const codes = props.analysis?.statusCodes || {}
  return Object.entries(codes)
    .map(([name, count]) => ({ name, count: Number(count) }))
    .sort((a, b) => b.count - a.count)
})

const formatMs = (value?: number) => {
  if (!value) return '0 ms'
  if (value >= 1000) return `${(value / 1000).toFixed(2)} s`
  return `${Math.round(value)} ms`
}

const openDrill = async (filterType: string, filterValue: string) => {
  title.value = filterValue
  ips.value = []
  urls.value = []
  visible.value = true
  try {
    const res = await drilldownNginxLog({
      site: '',
      siteId: props.siteId,
      days: props.days,
      timeRange: '',
      filterType,
      filterValue,
    })
    ips.value = res.data?.ips || []
    urls.value = res.data?.urls || []
  } catch { /* banner on the page already reports analysis errors */ }
}
</script>

<style scoped>
.insight-row { margin-top: 16px; }
.slow-summary { margin-bottom: 8px; color: var(--xp-text-secondary); font-size: 12px; }
.linkish {
  border: 0;
  background: transparent;
  color: var(--el-color-primary);
  cursor: pointer;
  padding: 0;
}
</style>
