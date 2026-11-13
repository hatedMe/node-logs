<script setup lang="ts">
import { computed } from 'vue'
import type { SourceData } from '../types/modules'
import { formatCellValue } from '../types/modules'
import { genSource } from '../mocks/modules'
import { useModulePage } from '../composables/useModulePage'
import { toCsv } from '../utils/csv'
import FilterBar from '../components/common/FilterBar.vue'
import StatCards from '../components/common/StatCards.vue'
import RankList from '../components/common/RankList.vue'
import PageTable from '../components/common/PageTable.vue'

function buildCsv(d: SourceData): string {
  const t = d.searchWords
  return toCsv(
    t.columns.map((c) => c.label),
    t.rows.map((r) => t.columns.map((c) => formatCellValue(c.format, r[c.prop])))
  )
}

const page = useModulePage<SourceData>({
  fetchData: (ctx) => genSource(ctx),
  buildCsv,
  exportFilename: 'source-search-words.csv',
  hasContent: (d) => !!d && (d.cards.length > 0 || d.searchWords.rows.length > 0)
})

const subtitle = computed(
  () => `${page.filters.value.startDate} 至 ${page.filters.value.endDate}`
)
</script>

<template>
  <div class="module-page" data-testid="view-source">
    <header class="module-header">
      <h2>来源分析</h2>
      <p>直接访问、搜索引擎、搜索词、外部链接与自定义广告来源的效果对比</p>
    </header>

    <el-card class="module-section" shadow="never">
      <FilterBar
        v-model:filters="page.filters.value"
        :loading="page.loading.value"
        :can-export="page.hasContent.value"
        show-keyword
        keyword-placeholder="过滤搜索词/搜索引擎"
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

    <section class="module-section" aria-label="来源指标">
      <StatCards :cards="page.data.value?.cards ?? []" :loading="page.loading.value" />
    </section>

    <el-row :gutter="12" class="module-section">
      <el-col :xs="24" :md="10">
        <el-card shadow="never">
          <RankList title="来源类型排行" :items="page.data.value?.sourceTypes ?? []" :loading="page.loading.value" />
        </el-card>
      </el-col>
      <el-col :xs="24" :md="14">
        <el-card shadow="never">
          <template #header><span>搜索词明细</span></template>
          <PageTable
            :rows="page.data.value?.searchWords.rows ?? []"
            :columns="page.data.value?.searchWords.columns ?? []"
            :loading="page.loading.value"
            :subtitle="subtitle"
            drawer-title="搜索词明细"
          />
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>
