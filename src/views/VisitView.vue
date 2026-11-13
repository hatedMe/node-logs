<script setup lang="ts">
import { computed } from 'vue'
import type { VisitData } from '../types/modules'
import { formatCellValue } from '../types/modules'
import { genVisit } from '../mocks/modules'
import { useModulePage } from '../composables/useModulePage'
import { toCsv } from '../utils/csv'
import FilterBar from '../components/common/FilterBar.vue'
import StatCards from '../components/common/StatCards.vue'
import RankList from '../components/common/RankList.vue'
import PageTable from '../components/common/PageTable.vue'

function buildCsv(d: VisitData): string {
  const t = d.pages
  return toCsv(
    t.columns.map((c) => c.label),
    t.rows.map((r) => t.columns.map((c) => formatCellValue(c.format, r[c.prop])))
  )
}

const page = useModulePage<VisitData>({
  fetchData: (ctx) => genVisit(ctx),
  buildCsv,
  exportFilename: 'visit-pages.csv',
  hasContent: (d) => !!d && (d.cards.length > 0 || d.pages.rows.length > 0)
})

const subtitle = computed(
  () => `${page.filters.value.startDate} 至 ${page.filters.value.endDate}`
)
</script>

<template>
  <div class="module-page" data-testid="view-visit">
    <header class="module-header">
      <h2>访问分析</h2>
      <p>受访页面、入口页面、退出页面与页面停留/退出情况</p>
    </header>

    <el-card class="module-section" shadow="never">
      <FilterBar
        v-model:filters="page.filters.value"
        :loading="page.loading.value"
        :can-export="page.hasContent.value"
        show-keyword
        keyword-placeholder="过滤页面 URL"
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

    <section class="module-section" aria-label="访问指标">
      <StatCards :cards="page.data.value?.cards ?? []" :loading="page.loading.value" />
    </section>

    <el-row :gutter="12" class="module-section">
      <el-col :xs="24" :md="12">
        <el-card shadow="never">
          <RankList title="入口页面 TOP" :items="page.data.value?.entries ?? []" :loading="page.loading.value" :limit="6" />
        </el-card>
      </el-col>
      <el-col :xs="24" :md="12">
        <el-card shadow="never">
          <RankList title="退出页面 TOP" :items="page.data.value?.exits ?? []" :loading="page.loading.value" :limit="6" />
        </el-card>
      </el-col>
    </el-row>

    <el-card class="module-section" shadow="never">
      <template #header><span>受访页面明细</span></template>
      <PageTable
        :rows="page.data.value?.pages.rows ?? []"
        :columns="page.data.value?.pages.columns ?? []"
        :loading="page.loading.value"
        :subtitle="subtitle"
        drawer-title="页面明细"
      />
    </el-card>
  </div>
</template>
