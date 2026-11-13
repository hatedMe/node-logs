<script setup lang="ts">
import { computed } from 'vue'
import type { OptimizeData } from '../types/modules'
import { formatCellValue } from '../types/modules'
import { genOptimize } from '../mocks/modules'
import { useModulePage } from '../composables/useModulePage'
import { toCsv } from '../utils/csv'
import FilterBar from '../components/common/FilterBar.vue'
import StatCards from '../components/common/StatCards.vue'
import RankList from '../components/common/RankList.vue'
import PageTable from '../components/common/PageTable.vue'

function buildCsv(d: OptimizeData): string {
  const t = d.crawlErrors
  return toCsv(
    t.columns.map((c) => c.label),
    t.rows.map((r) => t.columns.map((c) => formatCellValue(c.format, r[c.prop])))
  )
}

const page = useModulePage<OptimizeData>({
  fetchData: (ctx) => genOptimize(ctx),
  buildCsv,
  exportFilename: 'optimize-crawl-errors.csv',
  hasContent: (d) => !!d && (d.cards.length > 0 || d.crawlErrors.rows.length > 0)
})

const subtitle = computed(
  () => `${page.filters.value.startDate} 至 ${page.filters.value.endDate}`
)
</script>

<template>
  <div class="module-page" data-testid="view-optimize">
    <header class="module-header">
      <h2>优化分析</h2>
      <p>索引量、网站速度诊断、抓取异常与 SEO 优化建议</p>
    </header>

    <el-card class="module-section" shadow="never">
      <FilterBar
        v-model:filters="page.filters.value"
        :loading="page.loading.value"
        :can-export="page.hasContent.value"
        :show-source="false"
        show-keyword
        keyword-placeholder="过滤异常 URL/类型"
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

    <section class="module-section" aria-label="优化指标">
      <StatCards :cards="page.data.value?.cards ?? []" :loading="page.loading.value" />
    </section>

    <el-row :gutter="12" class="module-section">
      <el-col :xs="24" :md="10">
        <el-card shadow="never">
          <RankList title="慢页面 TOP（加载耗时）" :items="page.data.value?.slowPages ?? []" :loading="page.loading.value" />
        </el-card>
      </el-col>
      <el-col :xs="24" :md="14">
        <el-card shadow="never">
          <template #header><span>抓取异常明细</span></template>
          <PageTable
            :rows="page.data.value?.crawlErrors.rows ?? []"
            :columns="page.data.value?.crawlErrors.columns ?? []"
            :loading="page.loading.value"
            :subtitle="subtitle"
            drawer-title="异常明细"
          />
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>
