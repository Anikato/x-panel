<template>
  <div class="card-container">
    <div class="header">
      <h3>{{ $t('menu.systemLog') }}</h3>
      <div class="actions">
        <el-select v-model="lines" class="lines-select" @change="fetchLog">
          <el-option label="最近 100 行" :value="100" />
          <el-option label="最近 500 行" :value="500" />
          <el-option label="最近 1000 行" :value="1000" />
        </el-select>
        <el-button type="primary" @click="fetchLog" :loading="loading">
          <el-icon><Refresh /></el-icon>
          {{ $t('commons.refresh') }}
        </el-button>
      </div>
    </div>
    
    <div class="log-container" ref="logContainer">
      <el-skeleton :rows="10" animated v-if="loading && !logContent" />
      <div v-else-if="!logContent" class="empty-log">
        {{ $t('commons.noData') }}
      </div>
      <pre v-else class="log-content">{{ logContent }}</pre>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { getSystemLog } from '@/api/modules/log'
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'

const loading = ref(false)
const lines = ref(500)
const logContent = ref('')
const logContainer = ref<HTMLElement | null>(null)

const fetchLog = async () => {
  loading.value = true
  try {
    const res = await getSystemLog(lines.value)
    logContent.value = res.data || ''
    scrollToBottom()
  } catch (error: any) {
    ElMessage.error(error.message || '获取日志失败')
  } finally {
    loading.value = false
  }
}

const scrollToBottom = () => {
  nextTick(() => {
    if (logContainer.value) {
      logContainer.value.scrollTop = logContainer.value.scrollHeight
    }
  })
}

onMounted(() => {
  fetchLog()
})
</script>

<style lang="scss" scoped>
.card-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--xp-bg-card);
  border-radius: var(--xp-radius-lg);
  border: 1px solid var(--xp-border-light);
  padding: 20px;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  
  h3 {
    margin: 0;
    font-size: 16px;
    font-weight: 600;
    color: var(--xp-text-primary);
  }
  
  .actions {
    display: flex;
    gap: 12px;
    
    .lines-select {
      width: 140px;
    }
  }
}

.log-container {
  flex: 1;
  background: #1e1e1e;
  border-radius: var(--xp-radius-md);
  padding: 16px;
  overflow: auto;
  border: 1px solid var(--xp-border-light);
  
  .empty-log {
    display: flex;
    justify-content: center;
    align-items: center;
    height: 100%;
    color: #666;
  }
  
  .log-content {
    margin: 0;
    color: #d4d4d4;
    font-family: 'JetBrains Mono', Consolas, Monaco, monospace;
    font-size: 13px;
    line-height: 1.5;
    white-space: pre-wrap;
    word-break: break-all;
  }
}
</style>
