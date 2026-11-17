<script setup lang="ts">
import { computed, ref } from 'vue'
import type { ReportPageData, ReportPageTab } from '../../types/reports'
import { genConversionPath } from '../../mocks/reports'
import { useModulePage } from '../../composables/useModulePage'
import { tableCsv } from '../../utils/csv'
import ReportShell from '../../components/report/ReportShell.vue'
import ReportMetricStrip from '../../components/report/ReportMetricStrip.vue'
import ReportTabs from '../../components/report/ReportTabs.vue'
import FunnelSteps from '../../components/common/FunnelSteps.vue'
import PageTable from '../../components/common/PageTable.vue'

const page = useModulePage<ReportPageData>({
  fetchData: (ctx) => genConversionPath(ctx),
  buildCsv: (d) => tableCsv(d.table),
  exportFilename: 'conversion-path.csv',
  hasContent: (d) => !!d && (d.cards.length > 0 || d.table.rows.length > 0)
})

const tabs: ReportPageTab[] = [
  { key: 'main', label: '转化路径' },
  { key: 'secondary', label: '步骤流失' }
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
    test-id="view-conversion-path"
    title="转化路径"
    desc="查看用户从访问到完成转化的路径与各步骤流失，定位转化瓶颈。"
    keyword-placeholder="过滤路径"
    :show-source="false"
  >
    <ReportMetricStrip label="路径指标" :cards="page.data.value?.cards ?? []" :loading="page.loading.value" />

    <el-card shadow="never">
      <template #header><span>转化路径漏斗</span></template>
      <FunnelSteps title="转化路径漏斗" :steps="page.data.value?.funnel ?? []" :loading="page.loading.value" />
    </el-card>

    <ReportTabs v-model="activeTab" :tabs="tabs" />

    <el-card shadow="never" class="navigation-report__table-card">
      <template #header><span>{{ activeTab === 'secondary' ? '步骤流失明细' : '转化路径明细' }}</span></template>
      <PageTable
        :rows="currentTable.rows"
        :columns="currentTable.columns"
        :loading="page.loading.value"
        :subtitle="`${page.filters.value.startDate} 至 ${page.filters.value.endDate}`"
        drawer-title="路径明细"
      />
    </el-card>
  </ReportShell>
</template>
