<script setup lang="ts">
import { computed, ref } from 'vue'
import type { MetricKey } from '../../types/report'
import type { ReportPageData, ReportPageTab } from '../../types/reports'
import { genVisitType } from '../../mocks/reports'
import { useModulePage } from '../../composables/useModulePage'
import { tableCsv } from '../../utils/csv'
import ReportShell from '../../components/report/ReportShell.vue'
import ReportMetricStrip from '../../components/report/ReportMetricStrip.vue'
import ReportTabs from '../../components/report/ReportTabs.vue'
import SplitCompare from '../../components/report/SplitCompare.vue'
import TrendChart from '../../components/report/TrendChart.vue'
import PageTable from '../../components/common/PageTable.vue'

const page = useModulePage<ReportPageData>({
  fetchData: (ctx) => genVisitType(ctx),
  buildCsv: (d) => tableCsv(d.table),
  exportFilename: 'visitor-type.csv',
  hasContent: (d) => !!d && (d.cards.length > 0 || d.table.rows.length > 0)
})

const tabs: ReportPageTab[] = [
  { key: 'main', label: '新老访客对比' },
  { key: 'secondary', label: '访客来源对比' }
]
const activeTab = ref<string>('main')
const currentTable = computed(() => {
  const d = page.data.value
  if (!d) return { columns: [], rows: [] }
  return activeTab.value === 'secondary' && d.secondary ? d.secondary : d.table
})
const trend = computed(() => page.data.value?.trend ?? [])
const metrics: MetricKey[] = ['uv', 'pv']
</script>

<template>
  <ReportShell
    :page="page"
    test-id="view-visitor-type"
    title="新老访客"
    desc="对比新访客与老访客的访问行为与转化表现，评估留存与拉新效果。"
    keyword-placeholder="过滤来源"
    :show-source="false"
  >
    <ReportMetricStrip label="访客指标" :cards="page.data.value?.cards ?? []" :loading="page.loading.value" />

    <el-row :gutter="18" class="navigation-report__grid">
      <el-col :xs="24" :md="14">
        <el-card shadow="never">
          <template #header><span>新老访客构成</span></template>
          <SplitCompare :items="page.data.value?.rank ?? []" />
        </el-card>
      </el-col>
      <el-col :xs="24" :md="10">
        <el-card shadow="never">
          <template #header><span>访客趋势</span></template>
          <TrendChart :trend="trend" :metrics="metrics" :loading="page.loading.value" />
        </el-card>
      </el-col>
    </el-row>

    <ReportTabs v-model="activeTab" :tabs="tabs" />

    <el-card shadow="never" class="navigation-report__table-card">
      <template #header><span>{{ activeTab === 'secondary' ? '访客来源对比明细' : '新老访客对比明细' }}</span></template>
      <PageTable
        :rows="currentTable.rows"
        :columns="currentTable.columns"
        :loading="page.loading.value"
        :subtitle="`${page.filters.value.startDate} 至 ${page.filters.value.endDate}`"
        drawer-title="访客明细"
      />
    </el-card>
  </ReportShell>
</template>
