<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { ReportPageData, ReportPageTab } from '../../types/reports'
import { CLICKABLE_PAGES, genPageClick } from '../../mocks/reports'
import { useModulePage } from '../../composables/useModulePage'
import { tableCsv } from '../../utils/csv'
import ReportShell from '../../components/report/ReportShell.vue'
import ReportMetricStrip from '../../components/report/ReportMetricStrip.vue'
import ReportTabs from '../../components/report/ReportTabs.vue'
import BarList from '../../components/report/BarList.vue'
import HeatGrid from '../../components/common/HeatGrid.vue'
import PageTable from '../../components/common/PageTable.vue'

const selectedPage = ref<string>(CLICKABLE_PAGES[0])

const page = useModulePage<ReportPageData>({
  fetchData: (ctx) => genPageClick(ctx, selectedPage.value),
  buildCsv: (d) => tableCsv(d.table),
  exportFilename: 'visit-pageclick.csv',
  hasContent: (d) => !!d && (d.cards.length > 0 || (d.heat?.length ?? 0) > 0)
})

// 切换受访页面后立即重查（与原版点击图选择页面后刷新一致）。
watch(selectedPage, () => {
  void page.onQuery()
})

const tabs: ReportPageTab[] = [
  { key: 'main', label: '点击元素明细' },
  { key: 'secondary', label: '点击时段分布' }
]
const activeTab = ref<string>('main')
const currentTable = computed(() => {
  const d = page.data.value
  if (!d) return { columns: [], rows: [] }
  return activeTab.value === 'secondary' && d.secondary ? d.secondary : d.table
})
const heat = computed(() => page.data.value?.heat ?? [])
const topElements = computed(() => page.data.value?.rank ?? [])
</script>

<template>
  <ReportShell
    :page="page"
    test-id="view-visit-pageclick"
    title="页面点击图"
    desc="查看页面元素的点击分布与热门点击区域，定位用户关注点与交互热点。"
    :show-source="false"
  >
    <div class="page-picker">
      <span>受访页面：</span>
      <el-select v-model="selectedPage" size="small" data-testid="page-picker" aria-label="受访页面">
        <el-option v-for="p in page.data.value?.pages ?? CLICKABLE_PAGES" :key="p" :value="p" :label="p" />
      </el-select>
    </div>

    <ReportMetricStrip label="点击概况" :cards="page.data.value?.cards ?? []" :loading="page.loading.value" />

    <el-row :gutter="18" class="navigation-report__grid">
      <el-col :xs="24" :md="14">
        <el-card shadow="never">
          <template #header><span>页面点击热区</span></template>
          <HeatGrid :cells="heat" :loading="page.loading.value" />
        </el-card>
      </el-col>
      <el-col :xs="24" :md="10">
        <el-card shadow="never">
          <template #header><span>热门点击元素</span></template>
          <BarList :items="topElements" :loading="page.loading.value" />
        </el-card>
      </el-col>
    </el-row>

    <ReportTabs v-model="activeTab" :tabs="tabs" />

    <el-card shadow="never" class="navigation-report__table-card">
      <template #header><span>{{ activeTab === 'secondary' ? '点击时段分布' : '点击元素明细' }}</span></template>
      <PageTable
        :rows="currentTable.rows"
        :columns="currentTable.columns"
        :loading="page.loading.value"
        :subtitle="`${selectedPage}　${page.filters.value.startDate} 至 ${page.filters.value.endDate}`"
        drawer-title="点击明细"
      />
    </el-card>
  </ReportShell>
</template>

<style scoped>
.page-picker {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
  padding: 10px 14px;
  background: #fff;
  border: 1px solid #e2e6f0;
  color: #545b66;
  font-size: 12px;
}
.page-picker .el-select {
  width: 260px;
}
</style>
