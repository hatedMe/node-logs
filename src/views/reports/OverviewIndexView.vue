<script setup lang="ts">
import { computed } from 'vue'
import type { MetricKey } from '../../types/report'
import type { ReportPageData } from '../../types/reports'
import { genOverviewIndex } from '../../mocks/reports'
import { useModulePage } from '../../composables/useModulePage'
import { tableCsv } from '../../utils/csv'
import ReportShell from '../../components/report/ReportShell.vue'
import ReportMetricStrip from '../../components/report/ReportMetricStrip.vue'
import SourceDonut from '../../components/report/SourceDonut.vue'
import TrendChart from '../../components/report/TrendChart.vue'
import PageTable from '../../components/common/PageTable.vue'

const page = useModulePage<ReportPageData>({
  fetchData: (ctx) => genOverviewIndex(ctx),
  buildCsv: (d) => tableCsv(d.table),
  exportFilename: 'overview.csv',
  hasContent: (d) => !!d && (d.cards.length > 0 || d.table.rows.length > 0)
})

const trend = computed(() => page.data.value?.trend ?? [])
const metrics: MetricKey[] = ['pv', 'uv', 'ip']
</script>

<template>
  <ReportShell
    :page="page"
    test-id="view-overview"
    title="网站概况"
    desc="汇总网站整体流量、来源构成与 TOP 榜单，快速把握运营概况。"
    keyword-placeholder="过滤搜索词"
  >
    <ReportMetricStrip label="核心指标" :cards="page.data.value?.cards ?? []" :loading="page.loading.value" />

    <el-row :gutter="18" class="navigation-report__grid">
      <el-col :xs="24" :md="14">
        <el-card shadow="never">
          <template #header><span>流量趋势</span></template>
          <TrendChart :trend="trend" :metrics="metrics" :loading="page.loading.value" />
        </el-card>
      </el-col>
      <el-col :xs="24" :md="10">
        <el-card shadow="never">
          <template #header><span>来源构成</span></template>
          <SourceDonut title="来源构成" :items="page.data.value?.rank ?? []" :loading="page.loading.value" />
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="18" class="navigation-report__grid">
      <el-col :xs="24" :md="12">
        <el-card shadow="never">
          <template #header><span>Top10 搜索词</span></template>
          <PageTable
            :rows="page.data.value?.table.rows ?? []"
            :columns="page.data.value?.table.columns ?? []"
            :loading="page.loading.value"
            :subtitle="`${page.filters.value.startDate} 至 ${page.filters.value.endDate}`"
            drawer-title="搜索词明细"
          />
        </el-card>
      </el-col>
      <el-col :xs="24" :md="12">
        <el-card shadow="never">
          <template #header><span>Top10 受访页面</span></template>
          <PageTable
            :rows="page.data.value?.secondary?.rows ?? []"
            :columns="page.data.value?.secondary?.columns ?? []"
            :loading="page.loading.value"
            :subtitle="`${page.filters.value.startDate} 至 ${page.filters.value.endDate}`"
            drawer-title="受访页面明细"
          />
        </el-card>
      </el-col>
    </el-row>
  </ReportShell>
</template>
