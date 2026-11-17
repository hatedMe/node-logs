<script setup lang="ts">
import { computed } from 'vue'
import type { MetricKey } from '../../types/report'
import type { ReportPageData } from '../../types/reports'
import { genVisitLandingPage } from '../../mocks/reports'
import { useModulePage } from '../../composables/useModulePage'
import { tableCsv } from '../../utils/csv'
import ReportShell from '../../components/report/ReportShell.vue'
import ReportMetricStrip from '../../components/report/ReportMetricStrip.vue'
import BarList from '../../components/report/BarList.vue'
import TrendChart from '../../components/report/TrendChart.vue'
import PageTable from '../../components/common/PageTable.vue'

const page = useModulePage<ReportPageData>({
  fetchData: (ctx) => genVisitLandingPage(ctx),
  buildCsv: (d) => tableCsv(d.table),
  exportFilename: 'visit-landingpage.csv',
  hasContent: (d) => !!d && (d.cards.length > 0 || d.table.rows.length > 0)
})

const trend = computed(() => page.data.value?.trend ?? [])
const rank = computed(() => page.data.value?.rank ?? [])
const metrics: MetricKey[] = ['pv', 'uv']
</script>

<template>
  <ReportShell
    :page="page"
    test-id="view-visit-landingpage"
    title="入口页面"
    desc="查看访客进入网站的第一个页面及其跳出情况，优化落地页。"
    keyword-placeholder="过滤入口页面"
  >
    <ReportMetricStrip label="入口指标" :cards="page.data.value?.cards ?? []" :loading="page.loading.value" />

    <el-row :gutter="18" class="navigation-report__grid">
      <el-col :xs="24" :md="14">
        <el-card shadow="never">
          <template #header><span>入口趋势</span></template>
          <TrendChart :trend="trend" :metrics="metrics" :loading="page.loading.value" />
        </el-card>
      </el-col>
      <el-col :xs="24" :md="10">
        <el-card shadow="never">
          <template #header><span>入口页面 TOP</span></template>
          <BarList title="入口页面 TOP" :items="rank" :loading="page.loading.value" />
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never" class="navigation-report__table-card">
      <template #header><span>入口页面明细</span></template>
      <PageTable
        :rows="page.data.value?.table.rows ?? []"
        :columns="page.data.value?.table.columns ?? []"
        :loading="page.loading.value"
        :subtitle="`${page.filters.value.startDate} 至 ${page.filters.value.endDate}`"
        drawer-title="入口页面明细"
      />
    </el-card>
  </ReportShell>
</template>
