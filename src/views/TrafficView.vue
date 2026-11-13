<script setup lang="ts">
import { computed } from 'vue'
import type { MetricKey } from '../types/report'
import type { TrafficData } from '../types/modules'
import { formatCellValue } from '../types/modules'
import { genTraffic } from '../mocks/modules'
import { useModulePage } from '../composables/useModulePage'
import { toCsv } from '../utils/csv'
import FilterBar from '../components/common/FilterBar.vue'
import StatCards from '../components/common/StatCards.vue'
import TrendChart from '../components/report/TrendChart.vue'
import RankList from '../components/common/RankList.vue'
import PageTable from '../components/common/PageTable.vue'

const trendMetrics: MetricKey[] = ['pv', 'uv']

function buildCsv(d: TrafficData): string {
  const t = d.realtime
  return toCsv(
    t.columns.map((c) => c.label),
    t.rows.map((r) => t.columns.map((c) => formatCellValue(c.format, r[c.prop])))
  )
}

const page = useModulePage<TrafficData>({
  fetchData: (ctx) => genTraffic(ctx),
  buildCsv,
  exportFilename: 'traffic-realtime.csv',
  hasContent: (d) => !!d && (d.cards.length > 0 || d.realtime.rows.length > 0)
})

const subtitle = computed(
  () => `${page.filters.value.startDate} 至 ${page.filters.value.endDate}`
)
</script>

<template>
  <div class="module-page" data-testid="view-traffic">
    <header class="module-header">
      <h2>流量分析</h2>
      <p>按时间变化的访问情况：实时访客、当前访问页面与受访排行</p>
    </header>

    <el-card class="module-section" shadow="never">
      <FilterBar
        v-model:filters="page.filters.value"
        :loading="page.loading.value"
        :can-export="page.hasContent.value"
        show-keyword
        keyword-placeholder="过滤页面/来源/地域"
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

    <section class="module-section" aria-label="流量指标">
      <StatCards :cards="page.data.value?.cards ?? []" :loading="page.loading.value" />
    </section>

    <el-card class="module-section" shadow="never">
      <template #header><span>PV/UV 趋势</span></template>
      <TrendChart
        :trend="page.data.value?.trend ?? []"
        :metrics="trendMetrics"
        :loading="page.loading.value"
      />
    </el-card>

    <el-row :gutter="12" class="module-section">
      <el-col :xs="24" :md="10">
        <el-card shadow="never">
          <RankList title="受访页面 TOP" :items="page.data.value?.topPages ?? []" :loading="page.loading.value" />
        </el-card>
      </el-col>
      <el-col :xs="24" :md="14">
        <el-card shadow="never">
          <template #header><span>实时访客</span></template>
          <PageTable
            :rows="page.data.value?.realtime.rows ?? []"
            :columns="page.data.value?.realtime.columns ?? []"
            :loading="page.loading.value"
            :subtitle="subtitle"
            drawer-title="访客明细"
          />
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>
