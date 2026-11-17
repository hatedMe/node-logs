<script setup lang="ts">
import { computed, ref } from 'vue'
import type { MetricKey } from '../../types/report'
import type { ReportPageData, ReportPageTab } from '../../types/reports'
import { genTrafficLatest } from '../../mocks/reports'
import { useModulePage } from '../../composables/useModulePage'
import { tableCsv } from '../../utils/csv'
import ReportShell from '../../components/report/ReportShell.vue'
import ReportMetricStrip from '../../components/report/ReportMetricStrip.vue'
import ReportTabs from '../../components/report/ReportTabs.vue'
import BarList from '../../components/report/BarList.vue'
import TrendChart from '../../components/report/TrendChart.vue'
import PageTable from '../../components/common/PageTable.vue'

const page = useModulePage<ReportPageData>({
  fetchData: (ctx) => genTrafficLatest(ctx),
  buildCsv: (d) => tableCsv(d.table),
  exportFilename: 'traffic-latest.csv',
  hasContent: (d) => !!d && (d.cards.length > 0 || d.table.rows.length > 0)
})

const tabs: ReportPageTab[] = [
  { key: 'main', label: '实时访客' },
  { key: 'secondary', label: '今日按小时' }
]
const activeTab = ref<string>('main')
const currentTable = computed(() => {
  const d = page.data.value
  if (!d) return { columns: [], rows: [] }
  return activeTab.value === 'secondary' && d.secondary ? d.secondary : d.table
})
const trend = computed(() => page.data.value?.trend ?? [])
const metrics: MetricKey[] = ['pv', 'uv']
</script>

<template>
  <ReportShell
    :page="page"
    test-id="view-traffic-latest"
    title="实时访客"
    desc="实时查看当前正在访问网站的访客、来源与停留情况。"
    keyword-placeholder="过滤访客"
  >
    <ReportMetricStrip label="今日流量" :cards="page.data.value?.cards ?? []" :loading="page.loading.value" />

    <ReportTabs v-model="activeTab" :tabs="tabs" />

    <el-card shadow="never" class="navigation-report__table-card">
      <template #header><span>{{ activeTab === 'secondary' ? '今日按小时明细' : '实时访客明细' }}</span></template>
      <PageTable
        :rows="currentTable.rows"
        :columns="currentTable.columns"
        :loading="page.loading.value"
        :subtitle="`${page.filters.value.startDate} 至 ${page.filters.value.endDate}`"
        drawer-title="访客明细"
      />
    </el-card>

    <el-row :gutter="18" class="navigation-report__grid">
      <el-col :xs="24" :md="14">
        <el-card shadow="never">
          <template #header><span>近 7 日趋势</span></template>
          <TrendChart :trend="trend" :metrics="metrics" :loading="page.loading.value" />
        </el-card>
      </el-col>
      <el-col :xs="24" :md="10">
        <el-card shadow="never">
          <template #header><span>当前在线页面分布</span></template>
          <BarList title="当前在线页面分布" :items="page.data.value?.rank ?? []" :loading="page.loading.value" />
        </el-card>
      </el-col>
    </el-row>
  </ReportShell>
</template>
