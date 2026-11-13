<script setup lang="ts">
import { computed } from 'vue'
import type { ConversionData } from '../types/modules'
import { formatCellValue } from '../types/modules'
import { genConversion } from '../mocks/modules'
import { useModulePage } from '../composables/useModulePage'
import { toCsv } from '../utils/csv'
import FilterBar from '../components/common/FilterBar.vue'
import StatCards from '../components/common/StatCards.vue'
import FunnelSteps from '../components/common/FunnelSteps.vue'
import PageTable from '../components/common/PageTable.vue'

function buildCsv(d: ConversionData): string {
  const t = d.orders
  return toCsv(
    t.columns.map((c) => c.label),
    t.rows.map((r) => t.columns.map((c) => formatCellValue(c.format, r[c.prop])))
  )
}

const page = useModulePage<ConversionData>({
  fetchData: (ctx) => genConversion(ctx),
  buildCsv,
  exportFilename: 'conversion-orders.csv',
  hasContent: (d) => !!d && (d.cards.length > 0 || d.orders.rows.length > 0)
})

const subtitle = computed(
  () => `${page.filters.value.startDate} 至 ${page.filters.value.endDate}`
)
</script>

<template>
  <div class="module-page" data-testid="view-conversion">
    <header class="module-header">
      <h2>转化分析</h2>
      <p>转化概况、转化漏斗、订单明细与转化收益</p>
    </header>

    <el-card class="module-section" shadow="never">
      <FilterBar
        v-model:filters="page.filters.value"
        :loading="page.loading.value"
        :can-export="page.hasContent.value"
        show-keyword
        keyword-placeholder="过滤订单号/渠道"
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

    <section class="module-section" aria-label="转化指标">
      <StatCards :cards="page.data.value?.cards ?? []" :loading="page.loading.value" />
    </section>

    <el-card class="module-section" shadow="never">
      <FunnelSteps
        title="转化漏斗"
        :steps="page.data.value?.funnel ?? []"
        :loading="page.loading.value"
      />
    </el-card>

    <el-card class="module-section" shadow="never">
      <template #header><span>订单明细</span></template>
      <PageTable
        :rows="page.data.value?.orders.rows ?? []"
        :columns="page.data.value?.orders.columns ?? []"
        :loading="page.loading.value"
        :subtitle="subtitle"
        drawer-title="订单明细"
      />
    </el-card>
  </div>
</template>
