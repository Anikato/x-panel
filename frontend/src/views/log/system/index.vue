<template>
  <div class="card-container">
    <div class="header">
      <div class="left-actions">
        <el-select v-model="filter.level" class="level-select" :placeholder="$t('menu.level')" @change="fetchLog" clearable>
          <el-option :label="$t('menu.levelAll')" value="" />
          <el-option :label="$t('menu.levelInfo')" value="INFO" />
          <el-option :label="$t('menu.levelWarn')" value="WARN" />
          <el-option :label="$t('menu.levelError')" value="ERROR" />
        </el-select>
        <el-input 
          v-model="filter.keyword" 
          class="keyword-input" 
          :placeholder="$t('menu.keywordPlaceholder')" 
          clearable 
          @change="fetchLog"
          @keyup.enter="fetchLog">
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>
      <div class="right-actions">
        <el-switch
          v-model="autoRefresh"
          :active-text="$t('menu.autoRefresh')"
          @change="handleAutoRefreshChange"
        />
        <el-select v-model="filter.lines" class="lines-select" @change="fetchLog">
          <el-option :label="$t('menu.lines100')" :value="100" />
          <el-option :label="$t('menu.lines500')" :value="500" />
          <el-option :label="$t('menu.lines1000')" :value="1000" />
          <el-option :label="$t('menu.lines5000')" :value="5000" />
        </el-select>
        <el-button @click="downloadLog" :disabled="!logContent">
          <el-icon><Download /></el-icon>
          {{ $t('commons.download') }}
        </el-button>
        <el-button type="primary" @click="fetchLog" :loading="loading" :disabled="autoRefresh">
          <el-icon><Refresh /></el-icon>
          {{ $t('commons.refresh') }}
        </el-button>
        <el-popconfirm
          :title="$t('log.cleanConfirm')"
          @confirm="handleClean"
          width="250"
        >
          <template #reference>
            <el-button type="danger">
              <el-icon><Delete /></el-icon>
              {{ $t('log.clean') }}
            </el-button>
          </template>
        </el-popconfirm>
      </div>
    </div>
    
    <div class="log-container" ref="logContainerRef" @scroll="handleScroll">
      <el-skeleton :rows="10" animated v-if="loading && !logContent" />
      <div v-else-if="!logContent" class="empty-log">
        {{ $t('commons.noData') }}
      </div>
      <div v-else class="log-content" v-html="parsedLog"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, nextTick } from 'vue'
import { getSystemLog, cleanSystemLog } from '@/api/modules/log'
import { ElMessage } from 'element-plus'
import { Refresh, Search, Delete, Download } from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const loading = ref(false)
const autoRefresh = ref(false)
const logContent = ref('')
const logContainerRef = ref<HTMLElement | null>(null)
let refreshTimer: any = null
const isAtBottom = ref(true)

const filter = ref({
  level: '',
  keyword: '',
  lines: 500
})

const fetchLog = async () => {
  loading.value = true
  try {
    const res = await getSystemLog(filter.value.lines, filter.value.level, filter.value.keyword)
    logContent.value = res.data || ''
    
    // 如果用户当前在底部，或者开启了自动刷新，则新数据来时保持在底部
    if (isAtBottom.value || autoRefresh.value) {
      scrollToBottom()
    }
  } catch (error: any) {
    ElMessage.error(error.message || '获取日志失败')
    if (autoRefresh.value) {
      autoRefresh.value = false
      clearInterval(refreshTimer)
    }
  } finally {
    loading.value = false
  }
}

const handleClean = async () => {
  try {
    await cleanSystemLog()
    ElMessage.success(t('commons.operationSuccess'))
    fetchLog()
  } catch (error: any) {
    ElMessage.error(error.message || '清理失败')
  }
}

const downloadLog = () => {
  if (!logContent.value) return
  const blob = new Blob([logContent.value], { type: 'text/plain;charset=utf-8' })
  const url = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.setAttribute('download', `xpanel-system-${new Date().getTime()}.log`)
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  window.URL.revokeObjectURL(url)
}

const handleAutoRefreshChange = (val: boolean) => {
  if (val) {
    fetchLog()
    refreshTimer = setInterval(() => {
      fetchLog()
    }, 3000)
  } else {
    clearInterval(refreshTimer)
  }
}

const handleScroll = (e: Event) => {
  const target = e.target as HTMLElement
  // 判定是否在底部的阈值
  isAtBottom.value = target.scrollHeight - target.scrollTop - target.clientHeight < 10
}

const scrollToBottom = () => {
  nextTick(() => {
    if (logContainerRef.value) {
      logContainerRef.value.scrollTop = logContainerRef.value.scrollHeight
    }
  })
}

// 解析日志进行高亮
const parsedLog = computed(() => {
  if (!logContent.value) return ''
  
  // 防止 xss，并处理换行
  const escapeHTML = (str: string) => {
    return str.replace(/[&<>'"]/g, 
      tag => ({
          '&': '&amp;',
          '<': '&lt;',
          '>': '&gt;',
          "'": '&#39;',
          '"': '&quot;'
        }[tag] || tag)
    )
  }

  const lines = logContent.value.split('\n')
  return lines.map(line => {
    if (!line.trim()) return ''
    
    const escaped = escapeHTML(line)
    // 匹配 logrus text 格式: time="2026-04-19 10:45:04" level=info msg="message..."
    const match = escaped.match(/^time="(.*?)"\s+level=([a-zA-Z]+)\s+msg=(.*)$/)
    
    if (match) {
      const time = match[1]
      const level = match[2].toUpperCase()
      const msg = match[3]
      
      let levelClass = 'log-level-info'
      if (level === 'WARNING' || level === 'WARN') levelClass = 'log-level-warn'
      if (['ERROR', 'FATAL', 'PANIC'].includes(level)) levelClass = 'log-level-error'
      if (level === 'DEBUG') levelClass = 'log-level-debug'
      
      return `<div class="log-line">
        <span class="log-level ${levelClass}">${level}</span>
        <span class="log-time">[${time}]</span>
        <span class="log-msg">${msg}</span>
      </div>`
    }
    
    // 如果没有匹配上格式，按原样输出（可能是一些 panic 堆栈或异常）
    return `<div class="log-line"><span class="log-msg">${escaped}</span></div>`
  }).join('')
})

onMounted(() => {
  fetchLog()
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
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
  flex-wrap: wrap;
  gap: 16px;
  
  .left-actions {
    display: flex;
    gap: 12px;
    
    .level-select {
      width: 140px;
    }
    .keyword-input {
      width: 240px;
    }
  }
  
  .right-actions {
    display: flex;
    gap: 12px;
    align-items: center;
    
    .lines-select {
      width: 130px;
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
    font-family: 'JetBrains Mono', Consolas, Monaco, monospace;
    font-size: 13px;
    line-height: 1.6;
    word-break: break-all;
    
    :deep(.log-line) {
      display: flex;
      gap: 8px;
      padding: 2px 0;
      border-bottom: 1px solid rgba(255,255,255,0.03);
      
      &:hover {
        background: rgba(255,255,255,0.05);
      }
      
      .log-level {
        font-weight: 700;
        min-width: 45px;
      }
      .log-time {
        color: #858585;
        white-space: nowrap;
      }
      .log-msg {
        color: #d4d4d4;
        flex: 1;
        white-space: pre-wrap;
      }
      
      .log-level-info { color: #4CAF50; }
      .log-level-warn { color: #FF9800; }
      .log-level-error { color: #F44336; }
      .log-level-debug { color: #2196F3; }
    }
  }
}
</style>
