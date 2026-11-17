<script setup lang="ts">
import { computed, ref } from 'vue'
import type { ReportPageData, ReportPageTab } from '../../types/reports'
import { genTransOverview } from '../../mocks/reports'
import { useModulePage } from '../../composables/useModulePage'
import { tableCsv } from '../../utils/csv'
import ReportShell from '../../components/report/ReportShell.vue'
import ReportMetricStrip from '../../components/report/ReportMetricStrip.vue'
import ReportTabs from '../../components/report/ReportTabs.vue'
import SourceDonut from '../../components/report/SourceDonut.vue'
import FunnelSteps from '../../components/common/FunnelSteps.vue'
import PageTable from '../../components/common/PageTable.vue'

const page = useModulePage<ReportPageData>({
  fetchData: (ctx) => genTransOverview(ctx),
  buildCsv: (d) => tableCsv(d.table),
  exportFilename: 'conversion-overview.csv',
  hasContent: (d) => !!d && (d.cards.length > 0 || d.table.rows.length > 0)
})

const tabs: ReportPageTab[] = [
  { key: 'main', label: '转化目标' },
  { key: 'secondary', label: '转化来源' }
]
const activeTab = ref<string>('main')
const currentTable = computed(() => {
  const d = page.data.value
  if (!d) return { columns: [], rows: [] }
  return activeTab.value === 'secondary' && d.secondary ? d.secondary : d.table
})
const funnel = computed(() => page.data.value?.funnel ?? [])
const goals = computed(() => page.data.value?.rank ?? [])
</script>

<template>
  <ReportShell
    :page="page"
    test-id="view-conversion-overview"
    title="转化概况"
    desc="查看网站目标的转化次数、转化率与收益，评估各转化目标的达成效果。"
    :show-source="false"
    keyword-placeholder="过滤转化目标"
  >
    <ReportMetricStrip label="转化指标" :cards="page.data.value?.cards ?? []" :loading="page.loading.value" />

    <el-row :gutter="18" class="navigation-report__grid">
      <el-col :xs="24" :md="14">
        <el-card shadow="never">
          <template #header><span>转化漏斗</span></template>
          <FunnelSteps title="访问 → 支付转化路径" :steps="funnel" :loading="page.loading.value" />
        </el-card>
      </el-col>
      <el-col :xs="24" :md="10">
        <el-card shadow="never">
          <template #header><span>转化目标占比</span></template>
          <SourceDonut title="转化目标" :items="goals" :loading="page.loading.value" />
        </el-card>
      </el-col>
    </el-row>

    <ReportTabs v-model="activeTab" :tabs="tabs" />

    <el-card shadow="never" class="navigation-report__table-card">
      <template #header><span>{{ activeTab === 'secondary' ? '转化来源明细' : '转化目标明细' }}</span></template>
      <PageTable
        :rows="currentTable.rows"
        :columns="currentTable.columns"
        :loading="page.loading.value"
        :subtitle="`${page.filters.value.startDate} 至 ${page.filters.value.endDate}`"
        drawer-title="转化明细"
      />
    </el-card>
  </ReportShell>
</template>
