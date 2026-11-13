<script setup lang="ts">
import { computed, ref } from 'vue'
import { ElMessage } from 'element-plus'
import type { CustomMetric, CustomReportData, TableData } from '../types/modules'
import { CUSTOM_METRIC_LABELS, CUSTOM_METRICS, formatCellValue } from '../types/modules'
import { CUSTOM_DIMENSIONS, genCustomReport } from '../mocks/modules'
import { useModulePage } from '../composables/useModulePage'
import { toCsv } from '../utils/csv'
import FilterBar from '../components/common/FilterBar.vue'
import PageTable from '../components/common/PageTable.vue'

const dims = ref<string[]>(['地域', '来源'])
const metrics = ref<CustomMetric[]>(['pv', 'uv'])

interface SavedReport {
  name: string
  dims: string
  metrics: string
  created: string
}

const savedReports = ref<SavedReport[]>([
  { name: '地域×来源日报', dims: '地域, 来源', metrics: 'PV, UV', created: '2026-11-12 09:00' }
])

function tableToCsv(t: TableData): string {
  return toCsv(
    t.columns.map((c) => c.label),
    t.rows.map((r) => t.columns.map((c) => formatCellValue(c.format, r[c.prop])))
  )
}

const page = useModulePage<CustomReportData>({
  fetchData: (ctx) => genCustomReport(ctx, dims.value, metrics.value),
  buildCsv: (d) => tableToCsv(d.table),
  exportFilename: 'custom-report.csv',
  hasContent: (d) => !!d && d.table.rows.length > 0
})

const subtitle = computed(
  () => `维度：${dims.value.join('×') || '-'} · ${page.filters.value.startDate} 至 ${page.filters.value.endDate}`
)

function onSave(): void {
  if (dims.value.length === 0 || metrics.value.length === 0) {
    ElMessage.warning('请至少选择 1 个维度和 1 个指标')
    return
  }
  const name = `自定义报表${savedReports.value.length + 1}`
  savedReports.value = [
    ...savedReports.value,
    {
      name,
      dims: dims.value.join(', '),
      metrics: metrics.value.map((m) => CUSTOM_METRIC_LABELS[m]).join(', '),
      created: `${page.filters.value.endDate} 09:00`
    }
  ]
  ElMessage.success(`已保存${name}`)
}

function onDeleteSaved(name: string): void {
  savedReports.value = savedReports.value.filter((r) => r.name !== name)
}

const savedColumns = [
  { prop: 'name', label: '报表名称', sortable: true },
  { prop: 'dims', label: '维度', sortable: false },
  { prop: 'metrics', label: '指标', sortable: false },
  { prop: 'created', label: '创建时间', sortable: true }
]
</script>

<template>
  <div class="module-page" data-testid="view-custom">
    <header class="module-header">
      <h2>自定义报告</h2>
      <p>自由组合维度和指标，生成日常监控报表并保存复用</p>
    </header>

    <el-card class="module-section" shadow="never">
      <FilterBar
        v-model:filters="page.filters.value"
        :loading="page.loading.value"
        :can-export="page.hasContent.value"
        @submit="page.onQuery"
        @reset="page.onReset"
        @export="page.onExport"
      />
      <el-form label-width="84px">
        <el-form-item label="维度组合">
          <el-checkbox-group v-model="dims">
            <el-checkbox v-for="d in CUSTOM_DIMENSIONS" :key="d" :value="d">{{ d }}</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item label="指标组合">
          <el-checkbox-group v-model="metrics">
            <el-checkbox v-for="m in CUSTOM_METRICS" :key="m" :value="m">
              {{ CUSTOM_METRIC_LABELS[m] }}
            </el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="page.loading.value" @click="page.onQuery">生成报表</el-button>
          <el-button @click="onSave">保存当前组合</el-button>
        </el-form-item>
      </el-form>
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

    <el-card class="module-section" shadow="never">
      <template #header><span>报表结果</span></template>
      <PageTable
        :rows="page.data.value?.table.rows ?? []"
        :columns="page.data.value?.table.columns ?? []"
        :loading="page.loading.value"
        :subtitle="subtitle"
        drawer-title="维度明细"
      />
    </el-card>

    <el-card class="module-section" shadow="never">
      <template #header><span>已保存报表</span></template>
      <el-table :data="savedReports" stripe style="width: 100%">
        <el-table-column
          v-for="col in savedColumns"
          :key="col.prop"
          :prop="col.prop"
          :label="col.label"
        />
        <el-table-column label="操作" width="90">
          <template #default="{ row }">
            <el-button size="small" type="danger" @click="onDeleteSaved(row.name)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>
