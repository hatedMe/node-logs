<script setup lang="ts">
import { computed, ref } from 'vue'
import type { GeoRow } from '../../types/report'
import type { ReportPageData, ReportPageTab } from '../../types/reports'
import { formatCellValue } from '../../types/modules'
import { genVisitDistrict } from '../../mocks/reports'
import { useModulePage } from '../../composables/useModulePage'
import { tableCsv } from '../../utils/csv'
import ReportShell from '../../components/report/ReportShell.vue'
import ReportMetricStrip from '../../components/report/ReportMetricStrip.vue'
import ReportTabs from '../../components/report/ReportTabs.vue'
import BarList from '../../components/report/BarList.vue'
import MapPlaceholder from '../../components/report/MapPlaceholder.vue'
import PageTable from '../../components/common/PageTable.vue'

const page = useModulePage<ReportPageData>({
  fetchData: (ctx) => genVisitDistrict(ctx),
  buildCsv: (d) => tableCsv(d.table),
  exportFilename: 'visitor-district.csv',
  hasContent: (d) => !!d && (d.cards.length > 0 || d.table.rows.length > 0)
})

const tabs: ReportPageTab[] = [
  { key: 'main', label: '按省' },
  { key: 'secondary', label: '按国家/地区' }
]
const activeTab = ref<string>('main')
const currentTable = computed(() => {
  const d = page.data.value
  if (!d) return { columns: [], rows: [] }
  return activeTab.value === 'secondary' && d.secondary ? d.secondary : d.table
})
const geoRows = computed(() => page.data.value?.geo ?? [])
const topAreas = computed(() => page.data.value?.rank ?? [])

// 地图气泡点击 → 钻取该地域明细抽屉（与明细表行点击抽屉同协议）。
const detailRegion = ref<GeoRow | null>(null)
function onViewDetail(target: { name: string; code: string }): void {
  detailRegion.value = geoRows.value.find((r) => r.name === target.name) ?? null
}
</script>

<template>
  <ReportShell
    :page="page"
    test-id="view-visitor-district"
    title="地域分布"
    desc="查看访客的地域来源分布（按省/按国家），评估不同地区的流量质量与转化。"
    :show-source="false"
    keyword-placeholder="过滤地域"
  >
    <ReportMetricStrip label="访客概况" :cards="page.data.value?.cards ?? []" :loading="page.loading.value" />

    <el-row :gutter="18" class="navigation-report__grid">
      <el-col :xs="24" :md="14">
        <el-card shadow="never">
          <template #header><span>访客地域分布</span></template>
          <MapPlaceholder :rows="geoRows" :loading="page.loading.value" @view-detail="onViewDetail" />
        </el-card>
      </el-col>
      <el-col :xs="24" :md="10">
        <el-card shadow="never">
          <template #header><span>地域 TOP</span></template>
          <BarList :items="topAreas" :loading="page.loading.value" />
        </el-card>
      </el-col>
    </el-row>

    <ReportTabs v-model="activeTab" :tabs="tabs" />

    <el-card shadow="never" class="navigation-report__table-card">
      <template #header><span>{{ activeTab === 'secondary' ? '国家/地区明细' : '地域明细' }}</span></template>
      <PageTable
        :rows="currentTable.rows"
        :columns="currentTable.columns"
        :loading="page.loading.value"
        :subtitle="`${page.filters.value.startDate} 至 ${page.filters.value.endDate}`"
        drawer-title="地域明细"
      />
    </el-card>

    <el-drawer
      :model-value="detailRegion !== null"
      :title="detailRegion ? `${detailRegion.name} 访客明细` : '地域明细'"
      direction="rtl"
      @update:model-value="(v: boolean) => { if (!v) detailRegion = null }"
    >
      <el-descriptions v-if="detailRegion" :column="1" border>
        <el-descriptions-item label="地域">{{ detailRegion.name }}</el-descriptions-item>
        <el-descriptions-item label="访客数(UV)">{{ detailRegion.values.uv }}</el-descriptions-item>
        <el-descriptions-item label="浏览量(PV)">{{ detailRegion.values.pv }}</el-descriptions-item>
        <el-descriptions-item label="跳出率">{{ formatCellValue('percent', detailRegion.values.bounceRate) }}</el-descriptions-item>
        <el-descriptions-item label="平均停留时长">{{ formatCellValue('duration', detailRegion.values.avgDuration) }}</el-descriptions-item>
        <el-descriptions-item label="转化率">{{ formatCellValue('percent', detailRegion.values.conversionRate) }}</el-descriptions-item>
      </el-descriptions>
    </el-drawer>
  </ReportShell>
</template>
