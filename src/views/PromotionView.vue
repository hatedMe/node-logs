<script setup lang="ts">
import { computed } from 'vue'
import type { PromotionData } from '../types/modules'
import { formatCellValue } from '../types/modules'
import { genPromotion } from '../mocks/modules'
import { useModulePage } from '../composables/useModulePage'
import { toCsv } from '../utils/csv'
import FilterBar from '../components/common/FilterBar.vue'
import StatCards from '../components/common/StatCards.vue'
import FunnelSteps from '../components/common/FunnelSteps.vue'
import PageTable from '../components/common/PageTable.vue'

function buildCsv(d: PromotionData): string {
  const t = d.plans
  return toCsv(
    t.columns.map((c) => c.label),
    t.rows.map((r) => t.columns.map((c) => formatCellValue(c.format, r[c.prop])))
  )
}

const page = useModulePage<PromotionData>({
  fetchData: (ctx) => genPromotion(ctx),
  buildCsv,
  exportFilename: 'promotion-plans.csv',
  hasContent: (d) => !!d && (d.cards.length > 0 || d.plans.rows.length > 0)
})

const subtitle = computed(
  () => `${page.filters.value.startDate} 至 ${page.filters.value.endDate}`
)
</script>

<template>
  <div class="module-page" data-testid="view-promotion">
    <header class="module-header">
      <h2>百度推广</h2>
      <p>搜索推广与网盟推广从点击、访问到转化的效果分析</p>
    </header>

    <el-card class="module-section" shadow="never">
      <FilterBar
        v-model:filters="page.filters.value"
        :loading="page.loading.value"
        :can-export="page.hasContent.value"
        show-keyword
        keyword-placeholder="过滤推广计划"
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

    <section class="module-section" aria-label="推广指标">
      <StatCards :cards="page.data.value?.cards ?? []" :loading="page.loading.value" />
    </section>

    <el-card class="module-section" shadow="never">
      <FunnelSteps
        title="推广转化漏斗"
        :steps="page.data.value?.funnel ?? []"
        :loading="page.loading.value"
      />
    </el-card>

    <el-card class="module-section" shadow="never">
      <template #header><span>推广计划效果</span></template>
      <PageTable
        :rows="page.data.value?.plans.rows ?? []"
        :columns="page.data.value?.plans.columns ?? []"
        :loading="page.loading.value"
        :subtitle="subtitle"
        drawer-title="计划明细"
      />
    </el-card>
  </div>
</template>
