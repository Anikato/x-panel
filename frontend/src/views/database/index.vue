<template>
  <div>
    <el-tabs v-model="activeTab" @tab-change="onTabChange">
      <el-tab-pane label="MySQL / MariaDB" name="mysql">
        <DatabaseTab db-type="mysql" ref="mysqlTabRef" />
      </el-tab-pane>
      <el-tab-pane label="PostgreSQL" name="postgresql">
        <DatabaseTab db-type="postgresql" ref="pgTabRef" />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import DatabaseTab from './DatabaseTab.vue'

const activeTab = ref('mysql')
const mysqlTabRef = ref()
const pgTabRef = ref()

const onTabChange = (tab: string) => {
  if (tab === 'mysql') mysqlTabRef.value?.refresh()
  else pgTabRef.value?.refresh()
}
</script>
