<script setup lang="ts">
import { computed, ref } from 'vue'
import type { MetricKey } from '../../types/report'
import type { ReportPageData, ReportPageTab } from '../../types/reports'
import { genVisitTopPage } from '../../mocks/reports'
import { useModulePage } from '../../composables/useModulePage'
import { tableCsv } from '../../utils/csv'
import ReportShell from '../../components/report/ReportShell.vue'
import ReportMetricStrip from '../../components/report/ReportMetricStrip.vue'
import ReportTabs from '../../components/report/ReportTabs.vue'
import BarList from '../../components/report/BarList.vue'
import TrendChart from '../../components/report/TrendChart.vue'
import PageTable from '../../components/common/PageTable.vue'

const page = useModulePage<ReportPageData>({
  fetchData: (ctx) => genVisitTopPage(ctx),
  buildCsv: (d) => tableCsv(d.table),
  exportFilename: 'visit-toppage.csv',
  hasContent: (d) => !!d && (d.cards.length > 0 || d.table.rows.length > 0)
})

const tabs: ReportPageTab[] = [
  { key: 'main', label: '受访页面' },
  { key: 'secondary', label: '页面标题' }
]
const activeTab = ref<string>('main')
const currentTable = computed(() => {
  const d = page.data.value
  if (!d) return { columns: [], rows: [] }
  return activeTab.value === 'secondary' && d.secondary ? d.secondary : d.table
})
const trend = computed(() => page.data.value?.trend ?? [])
const rank = computed(() => page.data.value?.rank ?? [])
const metrics: MetricKey[] = ['pv', 'uv']
</script>

<template>
  <ReportShell
    :page="page"
    test-id="view-visit-toppage"
    title="受访页面"
    desc="查看受访页面的浏览量、停留时长与退出情况，评估页面内容表现。"
    keyword-placeholder="过滤页面 URL"
  >
    <ReportMetricStrip label="受访指标" :cards="page.data.value?.cards ?? []" :loading="page.loading.value" />

    <el-row :gutter="18" class="navigation-report__grid">
      <el-col :xs="24" :md="14">
        <el-card shadow="never">
          <template #header><span>受访趋势</span></template>
          <TrendChart :trend="trend" :metrics="metrics" :loading="page.loading.value" />
        </el-card>
      </el-col>
      <el-col :xs="24" :md="10">
        <el-card shadow="never">
          <template #header><span>受访页面 TOP</span></template>
          <BarList title="受访页面 TOP" :items="rank" :loading="page.loading.value" />
        </el-card>
      </el-col>
    </el-row>

    <ReportTabs v-model="activeTab" :tabs="tabs" />

    <el-card shadow="never" class="navigation-report__table-card">
      <template #header><span>{{ activeTab === 'secondary' ? '页面标题明细' : '受访页面明细' }}</span></template>
      <PageTable
        :rows="currentTable.rows"
        :columns="currentTable.columns"
        :loading="page.loading.value"
        :subtitle="`${page.filters.value.startDate} 至 ${page.filters.value.endDate}`"
        drawer-title="受访页面明细"
      />
    </el-card>
  </ReportShell>
</template>
