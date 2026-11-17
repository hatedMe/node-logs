<script setup lang="ts">
import { computed, ref } from 'vue'
import type { MetricKey } from '../../types/report'
import type { ReportPageData, ReportPageTab } from '../../types/reports'
import { genTrendTime } from '../../mocks/reports'
import { useModulePage } from '../../composables/useModulePage'
import { tableCsv } from '../../utils/csv'
import ReportShell from '../../components/report/ReportShell.vue'
import ReportMetricStrip from '../../components/report/ReportMetricStrip.vue'
import ReportTabs from '../../components/report/ReportTabs.vue'
import TrendChart from '../../components/report/TrendChart.vue'
import PageTable from '../../components/common/PageTable.vue'

const page = useModulePage<ReportPageData>({
  fetchData: (ctx) => genTrendTime(ctx),
  buildCsv: (d) => tableCsv(d.table),
  exportFilename: 'traffic-time.csv',
  hasContent: (d) => !!d && (d.cards.length > 0 || d.table.rows.length > 0)
})

const tabs: ReportPageTab[] = [
  { key: 'main', label: '按天' },
  { key: 'secondary', label: '按小时' }
]
const activeTab = ref<string>('main')
const currentTable = computed(() => {
  const d = page.data.value
  if (!d) return { columns: [], rows: [] }
  return activeTab.value === 'secondary' && d.secondary ? d.secondary : d.table
})
const trend = computed(() => page.data.value?.trend ?? [])
const metrics: MetricKey[] = ['pv', 'uv', 'visits']
</script>

<template>
  <ReportShell
    :page="page"
    test-id="view-traffic-time"
    title="趋势分析"
    desc="查看访问量随时间的变化趋势，按天/按小时下钻分析流量波动。"
    keyword-placeholder="过滤日期"
  >
    <ReportMetricStrip label="核心指标" :cards="page.data.value?.cards ?? []" :loading="page.loading.value" />

    <el-card shadow="never">
      <template #header><span>流量趋势</span></template>
      <TrendChart :trend="trend" :metrics="metrics" :loading="page.loading.value" />
    </el-card>

    <ReportTabs v-model="activeTab" :tabs="tabs" />

    <el-card shadow="never" class="navigation-report__table-card">
      <template #header><span>{{ activeTab === 'secondary' ? '按小时趋势明细' : '按天趋势明细' }}</span></template>
      <PageTable
        :rows="currentTable.rows"
        :columns="currentTable.columns"
        :loading="page.loading.value"
        :subtitle="`${page.filters.value.startDate} 至 ${page.filters.value.endDate}`"
        drawer-title="趋势明细"
      />
    </el-card>
  </ReportShell>
</template>
