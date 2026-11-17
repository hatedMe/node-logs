<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { ReportPageData, ReportPageTab } from '../../types/reports'
import { FLOWABLE_PAGES, genPageFlow } from '../../mocks/reports'
import { useModulePage } from '../../composables/useModulePage'
import { tableCsv } from '../../utils/csv'
import ReportShell from '../../components/report/ReportShell.vue'
import ReportMetricStrip from '../../components/report/ReportMetricStrip.vue'
import ReportTabs from '../../components/report/ReportTabs.vue'
import FlowTrack from '../../components/report/FlowTrack.vue'
import PageTable from '../../components/common/PageTable.vue'

const selectedPage = ref<string>(FLOWABLE_PAGES[0])

const page = useModulePage<ReportPageData>({
  fetchData: (ctx) => genPageFlow(ctx, selectedPage.value),
  buildCsv: (d) => tableCsv(d.table),
  exportFilename: 'visit-flow.csv',
  hasContent: (d) => !!d && (d.cards.length > 0 || d.table.rows.length > 0)
})

// 切换受访页面后立即重查。
watch(selectedPage, () => {
  void page.onQuery()
})

const tabs: ReportPageTab[] = [
  { key: 'main', label: '上游页面' },
  { key: 'secondary', label: '下游页面' }
]
const activeTab = ref<string>('main')
const currentTable = computed(() => {
  const d = page.data.value
  if (!d) return { columns: [], rows: [] }
  return activeTab.value === 'secondary' && d.secondary ? d.secondary : d.table
})
</script>

<template>
  <ReportShell
    :page="page"
    test-id="view-visit-flow"
    title="页面上下游"
    desc="分析页面的上游来源与下游去向，还原用户在站内的浏览流转路径。"
    keyword-placeholder="过滤页面"
    :show-source="false"
  >
    <div class="page-picker">
      <span>受访页面：</span>
      <el-select v-model="selectedPage" size="small" data-testid="page-picker" aria-label="受访页面">
        <el-option v-for="p in page.data.value?.pages ?? FLOWABLE_PAGES" :key="p" :value="p" :label="p" />
      </el-select>
    </div>

    <ReportMetricStrip label="流转指标" :cards="page.data.value?.cards ?? []" :loading="page.loading.value" />

    <el-card shadow="never">
      <template #header><span>页面流转路径</span></template>
      <FlowTrack title="页面流转路径" :steps="page.data.value?.funnel ?? []" :loading="page.loading.value" value-unit="人" />
    </el-card>

    <ReportTabs v-model="activeTab" :tabs="tabs" />

    <el-card shadow="never" class="navigation-report__table-card">
      <template #header><span>{{ activeTab === 'secondary' ? '下游页面明细' : '上游页面明细' }}</span></template>
      <PageTable
        :rows="currentTable.rows"
        :columns="currentTable.columns"
        :loading="page.loading.value"
        :subtitle="`${page.filters.value.startDate} 至 ${page.filters.value.endDate}`"
        drawer-title="流转明细"
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
