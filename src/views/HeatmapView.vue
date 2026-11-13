<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { HeatmapData } from '../types/modules'
import { formatCellValue } from '../types/modules'
import { genHeatmap } from '../mocks/modules'
import { useModulePage } from '../composables/useModulePage'
import { toCsv } from '../utils/csv'
import FilterBar from '../components/common/FilterBar.vue'
import HeatGrid from '../components/common/HeatGrid.vue'
import PageTable from '../components/common/PageTable.vue'

// 受访页面选择是本页局部状态：切换后直接重查，热图与元素排行同步更新。
const heatPage = ref('/index.html')

function buildCsv(d: HeatmapData): string {
  const t = d.elements
  return toCsv(
    t.columns.map((c) => c.label),
    t.rows.map((r) => t.columns.map((c) => formatCellValue(c.format, r[c.prop])))
  )
}

const page = useModulePage<HeatmapData>({
  fetchData: (ctx) => genHeatmap(ctx, heatPage.value),
  buildCsv,
  exportFilename: 'heatmap-elements.csv',
  hasContent: (d) => !!d && (d.cells.length > 0 || d.elements.rows.length > 0)
})

watch(heatPage, () => {
  void page.onQuery()
})

const subtitle = computed(
  () => `${heatPage.value} · ${page.filters.value.startDate} 至 ${page.filters.value.endDate}`
)
</script>

<template>
  <div class="module-page" data-testid="view-heatmap">
    <header class="module-header">
      <h2>热力图与点击分析</h2>
      <p>页面点击热图：观察用户与页面元素的交互，指导布局优化</p>
    </header>

    <el-card class="module-section" shadow="never">
      <FilterBar
        v-model:filters="page.filters.value"
        :loading="page.loading.value"
        :can-export="page.hasContent.value"
        show-keyword
        keyword-placeholder="过滤点击区域"
        @submit="page.onQuery"
        @reset="page.onReset"
        @export="page.onExport"
      />
      <el-form label-width="84px">
        <el-form-item label="受访页面">
          <el-select v-model="heatPage" data-testid="heatmap-page">
            <el-option
              v-for="p in (page.data.value?.pages ?? ['/index.html'])"
              :key="p"
              :value="p"
              :label="p"
            />
          </el-select>
        </el-form-item>
      </el-form>
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

    <el-card class="module-section" shadow="never">
      <template #header><span>点击热图（点击色块查看明细）</span></template>
      <HeatGrid :cells="page.data.value?.cells ?? []" :loading="page.loading.value" />
    </el-card>

    <el-card class="module-section" shadow="never">
      <template #header><span>点击元素排行</span></template>
      <PageTable
        :rows="page.data.value?.elements.rows ?? []"
        :columns="page.data.value?.elements.columns ?? []"
        :loading="page.loading.value"
        :subtitle="subtitle"
        drawer-title="元素明细"
      />
    </el-card>
  </div>
</template>
