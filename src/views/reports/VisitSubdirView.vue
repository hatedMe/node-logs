<script setup lang="ts">
import { computed } from 'vue'
import type { MetricKey } from '../../types/report'
import type { ReportPageData } from '../../types/reports'
import { genSubDir } from '../../mocks/reports'
import { useModulePage } from '../../composables/useModulePage'
import { tableCsv } from '../../utils/csv'
import ReportShell from '../../components/report/ReportShell.vue'
import ReportMetricStrip from '../../components/report/ReportMetricStrip.vue'
import SourceDonut from '../../components/report/SourceDonut.vue'
import TrendChart from '../../components/report/TrendChart.vue'
import PageTable from '../../components/common/PageTable.vue'

const page = useModulePage<ReportPageData>({
  fetchData: (ctx) => genSubDir(ctx),
  buildCsv: (d) => tableCsv(d.table),
  exportFilename: 'visit-subdir.csv',
  hasContent: (d) => !!d && (d.cards.length > 0 || d.table.rows.length > 0)
})

const trend = computed(() => page.data.value?.trend ?? [])
const rank = computed(() => page.data.value?.rank ?? [])
const metrics: MetricKey[] = ['pv', 'uv']
</script>

<template>
  <ReportShell
    :page="page"
    test-id="view-visit-subdir"
    title="子目录"
    desc="按子目录查看访问数据，比较各内容板块的流量与转化。"
    keyword-placeholder="过滤子目录"
  >
    <ReportMetricStrip label="子目录指标" :cards="page.data.value?.cards ?? []" :loading="page.loading.value" />

    <el-row :gutter="18" class="navigation-report__grid">
      <el-col :xs="24" :md="14">
        <el-card shadow="never">
          <template #header><span>子目录趋势</span></template>
          <TrendChart :trend="trend" :metrics="metrics" :loading="page.loading.value" />
        </el-card>
      </el-col>
      <el-col :xs="24" :md="10">
        <el-card shadow="never">
          <template #header><span>子目录占比</span></template>
          <SourceDonut title="子目录占比" :items="rank" :loading="page.loading.value" />
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never" class="navigation-report__table-card">
      <template #header><span>子目录明细</span></template>
      <PageTable
        :rows="page.data.value?.table.rows ?? []"
        :columns="page.data.value?.table.columns ?? []"
        :loading="page.loading.value"
        :subtitle="`${page.filters.value.startDate} 至 ${page.filters.value.endDate}`"
        drawer-title="子目录明细"
      />
    </el-card>
  </ReportShell>
</template>
