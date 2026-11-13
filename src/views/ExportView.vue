<script setup lang="ts">
import { computed, ref } from 'vue'
import { ElMessage } from 'element-plus'
import type { ExportData, TableRow } from '../types/modules'
import { formatCellValue } from '../types/modules'
import { genExport } from '../mocks/modules'
import { useModulePage } from '../composables/useModulePage'
import { toCsv } from '../utils/csv'
import FilterBar from '../components/common/FilterBar.vue'
import StatCards from '../components/common/StatCards.vue'
import PageTable from '../components/common/PageTable.vue'

// 本地追加的任务/密钥：演示"创建"交互，数据只存内存。
const extraTasks = ref<TableRow[]>([])
const extraKeys = ref<TableRow[]>([])

function buildCsv(d: ExportData): string {
  const t = d.tasks
  const all = [...t.rows, ...extraTasks.value]
  return toCsv(
    t.columns.map((c) => c.label),
    all.map((r) => t.columns.map((c) => formatCellValue(c.format, r[c.prop] ?? '-')))
  )
}

const page = useModulePage<ExportData>({
  fetchData: (ctx) => genExport(ctx),
  buildCsv,
  exportFilename: 'export-tasks.csv',
  hasContent: (d) => !!d && (d.cards.length > 0 || d.tasks.rows.length + extraTasks.value.length > 0)
})

const allTasks = computed(() => [...(page.data.value?.tasks.rows ?? []), ...extraTasks.value])
const allKeys = computed(() => [...(page.data.value?.apiKeys.rows ?? []), ...extraKeys.value])

const subtitle = computed(
  () => `${page.filters.value.startDate} 至 ${page.filters.value.endDate}`
)

function onCreateTask(): void {
  extraTasks.value = [
    ...extraTasks.value,
    {
      name: `手动导出任务${extraTasks.value.length + 1}`,
      range: subtitle.value,
      status: '排队中',
      created: '2026-11-13 10:30'
    }
  ]
  ElMessage.success('已创建导出任务')
}

function onRefreshStatus(): void {
  extraTasks.value = extraTasks.value.map((t) =>
    t.status === '排队中' ? { ...t, status: '已完成' } : t
  )
  ElMessage.success('任务状态已刷新')
}

function onCreateKey(): void {
  extraKeys.value = [
    ...extraKeys.value,
    {
      name: `新密钥${extraKeys.value.length + 1}`,
      prefix: 'ak_live_new…',
      calls: 0,
      status: '启用'
    }
  ]
  ElMessage.success('已创建 API 密钥（演示）')
}

const API_DOC = `# Data API 拉取地域报告（示例）
curl "https://api.example.com/v1/report/visit/district?siteId=16847648&start=20261106&end=20261112" \\
  -H "Authorization: Bearer <API_KEY>"

# JS API 上报业务转化（示例）
window._hmt.push(['_trackOrder', { orderId: 'ORD-1', amount: 99 }])`
</script>

<template>
  <div class="module-page" data-testid="view-export">
    <header class="module-header">
      <h2>数据导出与开放平台</h2>
      <p>报告下载、Data API 拉取与 JS API 上报：把统计数据接入内部报表</p>
    </header>

    <el-card class="module-section" shadow="never">
      <FilterBar
        v-model:filters="page.filters.value"
        :loading="page.loading.value"
        :can-export="page.hasContent.value"
        :show-source="false"
        :show-device="false"
        @submit="page.onQuery"
        @reset="page.onReset"
        @export="page.onExport"
      />
    </el-card>

    <el-alert
      v-if="page.error.value"
      class="module-section"
      type="error"
      :title="page.error.value"
      show-icon
      :closable="false"
    >
      <el-button size="small" @click="page.onRetry">重试</el-button>
    </el-alert>

    <section class="module-section" aria-label="开放平台指标">
      <StatCards :cards="page.data.value?.cards ?? []" :loading="page.loading.value" />
    </section>

    <el-card class="module-section" shadow="never">
      <template #header>
        <span>导出任务</span>
        <span class="header-actions">
          <el-button size="small" type="primary" @click="onCreateTask">新建任务</el-button>
          <el-button size="small" @click="onRefreshStatus">刷新状态</el-button>
        </span>
      </template>
      <PageTable
        :rows="allTasks"
        :columns="page.data.value?.tasks.columns ?? []"
        :loading="page.loading.value"
        :subtitle="subtitle"
        drawer-title="任务明细"
      />
    </el-card>

    <el-card class="module-section" shadow="never">
      <template #header>
        <span>API 密钥</span>
        <span class="header-actions">
          <el-button size="small" @click="onCreateKey">新建密钥</el-button>
        </span>
      </template>
      <PageTable
        :rows="allKeys"
        :columns="page.data.value?.apiKeys.columns ?? []"
        :loading="page.loading.value"
        drawer-title="密钥明细"
      />
    </el-card>

    <el-card class="module-section" shadow="never">
      <template #header><span>接入文档（示例）</span></template>
      <pre class="api-doc">{{ API_DOC }}</pre>
    </el-card>
  </div>
</template>

<style scoped>
.header-actions {
  float: right;
  display: inline-flex;
  gap: 8px;
}
.api-doc {
  margin: 0;
  padding: 12px;
  background: var(--el-fill-color-light);
  border-radius: 4px;
  font-size: 12px;
  overflow-x: auto;
}
</style>
