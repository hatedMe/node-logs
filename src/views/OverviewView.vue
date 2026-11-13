<script setup lang="ts">
import { computed } from 'vue'
import type { MetricKey } from '../types/report'
import type { OverviewData } from '../types/modules'
import { formatCellValue } from '../types/modules'
import { genOverview } from '../mocks/modules'
import { useModulePage } from '../composables/useModulePage'
import { toCsv } from '../utils/csv'
import FilterBar from '../components/common/FilterBar.vue'
import StatCards from '../components/common/StatCards.vue'
import TrendChart from '../components/report/TrendChart.vue'
import RankList from '../components/common/RankList.vue'
import PageTable from '../components/common/PageTable.vue'

const trendMetrics: MetricKey[] = ['pv', 'uv']

function buildCsv(d: OverviewData): string {
  const t = d.keywords
  return toCsv(
    t.columns.map((c) => c.label),
    t.rows.map((r) => t.columns.map((c) => formatCellValue(c.format, r[c.prop])))
  )
}

const page = useModulePage<OverviewData>({
  fetchData: (ctx) => genOverview(ctx),
  buildCsv,
  exportFilename: 'overview-top-words.csv',
  hasContent: (d) => !!d && (d.cards.length > 0 || d.keywords.rows.length > 0)
})

const subtitle = computed(
  () => `${page.filters.value.startDate} 至 ${page.filters.value.endDate}`
)
</script>

<template>
  <div class="module-page" data-testid="view-overview">
    <header class="module-header">
      <h2>网站概况</h2>
      <p>汇总网站运营表现：核心指标、TOP 来源、TOP 入口页面与 TOP 搜索词</p>
    </header>

    <el-card class="module-section" shadow="never">
      <FilterBar
        v-model:filters="page.filters.value"
        :loading="page.loading.value"
        :can-export="page.hasContent.value"
        show-keyword
        keyword-placeholder="过滤来源/搜索词"
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

    <section class="module-section" aria-label="核心指标">
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
      <el-col :xs="24" :md="12">
        <el-card shadow="never">
          <RankList title="TOP 来源" :items="page.data.value?.topSources ?? []" :loading="page.loading.value" />
        </el-card>
      </el-col>
      <el-col :xs="24" :md="12">
        <el-card shadow="never">
          <RankList title="TOP 入口页面" :items="page.data.value?.topPages ?? []" :loading="page.loading.value" />
        </el-card>
      </el-col>
    </el-row>

    <el-card class="module-section" shadow="never">
      <template #header><span>TOP 搜索词</span></template>
      <PageTable
        :rows="page.data.value?.keywords.rows ?? []"
        :columns="page.data.value?.keywords.columns ?? []"
        :loading="page.loading.value"
        :subtitle="subtitle"
        drawer-title="搜索词明细"
      />
    </el-card>
  </div>
</template>
