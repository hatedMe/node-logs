<script setup lang="ts">
// 访客分析 → 地域分布报表：对齐公开页面真实结构（工具栏、高级筛选、地域 tab、地图/列表切换）。
// 约束：筛选区最后一个 ElRadioGroup 必须是地域层级（旧测试用 groups[last] 切换按省/按国家）；
// 新增按钮文案避开“查询/重置/下载/重试/查看明细”的精确匹配；窄屏栅格断点保持不变。
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import type { MetricKey } from '../types/report'
import { GEO_LEVEL_OPTIONS, SOURCE_OPTIONS, VISITOR_OPTIONS } from '../types/report'
import { defaultFilters, useReport } from '../composables/useReport'
import { downloadCsv, exportRows } from '../utils/export'
import ReportFilters from '../components/report/ReportFilters.vue'
import VisitorToolbar from '../components/report/VisitorToolbar.vue'
import MetricCards from '../components/report/MetricCards.vue'
import TrendChart from '../components/report/TrendChart.vue'
import GeoView from '../components/report/GeoView.vue'
import MapPlaceholder from '../components/report/MapPlaceholder.vue'
import ReportTable from '../components/report/ReportTable.vue'

/** 地域分布页默认指标（5 项）：PV、UV、IP、跳出率、平均访问时长。 */
const VISITOR_DEFAULT_METRICS: MetricKey[] = ['pv', 'uv', 'ip', 'bounceRate', 'avgDuration']

const tableDefaultSort: { prop: string; order: 'ascending' | 'descending' } = {
  prop: 'pv',
  order: 'descending'
}

const { filters, result, loading, error, validationErrors, hasData, query } = useReport()

// 视图层覆盖默认指标：共享 DEFAULT_METRICS 保持 PV/UV 不变，旧测试不受影响。
filters.value.metrics = [...VISITOR_DEFAULT_METRICS]

function labelOf(options: Array<{ value: string; label: string }>, value: string): string {
  return options.find((o) => o.value === value)?.label ?? value
}

const tableSubtitle = computed(() => {
  const f = filters.value
  let base =
    `${f.startDate} 至 ${f.endDate} · ${labelOf(SOURCE_OPTIONS, f.source)} · ` +
    `${labelOf(VISITOR_OPTIONS, f.visitorType)} · ${labelOf(GEO_LEVEL_OPTIONS, f.geoLevel)}`
  const areas = f.areas ?? []
  if (areas.length > 0) base += ` · 地域：${areas.join('、')}`
  const engines = f.engines ?? []
  if (f.source === 'search' && engines.length > 0) base += ` · 引擎：${engines.join('、')}`
  return f.compareEnabled ? `${base} · 对比 ${f.compareStartDate} 至 ${f.compareEndDate}` : base
})

function onDownload(): void {
  if (!result.value || result.value.geoRows.length === 0) {
    ElMessage.warning('当前无数据可下载')
    return
  }
  downloadCsv(
    exportRows(result.value.geoRows, filters.value.metrics),
    `pvuv-report-${filters.value.startDate}-${filters.value.endDate}.csv`
  )
  ElMessage.success('已导出当前结果')
}

async function onReset(): Promise<void> {
  filters.value = { ...defaultFilters(), metrics: [...VISITOR_DEFAULT_METRICS] }
  validationErrors.value = []
  revealCode.value = null
  await query()
}

function onGeoDetail(region: { name: string; code: string }): void {
  revealCode.value = region.code
}

// 按省/按国家 tab 切换后立即重查（与真实页面一致）；其余筛选仍走查询按钮。
function onGeoTabChange(): void {
  void query()
}

// 地图/排行视图切换：纯展示切换，不触发查询；两视图都常驻 DOM（v-show），联动逻辑不变。
const geoViewTab = ref<'map' | 'rank'>('map')

// 地域排行与明细表联动：新结果到达即清空旧联动，保证重复点击同地域仍可触发。
const revealCode = ref<string | null>(null)
watch(result, () => {
  revealCode.value = null
})

onMounted(() => {
  void query()
})
</script>

<template>
  <div class="report-page" data-testid="view-visitor">
    <header class="report-header">
      <h1>PV/UV 统计报表</h1>
      <p class="report-desc">本地 mock 数据演示：筛选、对比、钻取与导出交互</p>
    </header>

    <div class="report-section">
      <VisitorToolbar :can-download="hasData" @download="onDownload" />
    </div>

    <el-card class="report-section" shadow="never">
      <ReportFilters
        v-model:filters="filters"
        :loading="loading"
        :can-download="hasData"
        @submit="query"
        @reset="onReset"
        @download="onDownload"
      />
    </el-card>

    <el-alert
      v-if="error"
      class="report-section report-error"
      type="error"
      :title="error"
      show-icon
      :closable="false"
    >
      <el-button size="small" class="retry-btn" @click="query">重试</el-button>
    </el-alert>

    <section class="report-section" aria-label="指标摘要">
      <MetricCards :result="result" :metrics="filters.metrics" :loading="loading" />
    </section>

    <el-row :gutter="12" class="report-section">
      <el-col :xs="24" :md="14">
        <el-card shadow="never" class="chart-card">
          <template #header><span>趋势</span></template>
          <TrendChart
            :trend="result?.trend ?? []"
            :metrics="filters.metrics"
            :compare-enabled="filters.compareEnabled"
            :loading="loading"
          />
        </el-card>
      </el-col>
      <el-col :xs="24" :md="10">
        <el-card shadow="never" class="chart-card">
          <el-tabs v-model="filters.geoLevel" @tab-change="onGeoTabChange">
            <el-tab-pane label="按省" name="province" />
            <el-tab-pane label="按国家" name="country" />
          </el-tabs>
          <el-tabs v-model="geoViewTab">
            <el-tab-pane label="地图分布" name="map" />
            <el-tab-pane label="地域排行" name="rank" />
          </el-tabs>
          <MapPlaceholder
            v-show="geoViewTab === 'map'"
            :rows="result?.geoRows ?? []"
            :loading="loading"
            @view-detail="onGeoDetail"
          />
          <div v-show="geoViewTab === 'rank'">
            <GeoView
              :rows="result?.geoRows ?? []"
              :geo-level="filters.geoLevel"
              :loading="loading"
              @view-detail="onGeoDetail"
            />
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-card class="report-section" shadow="never">
      <template #header><span>地域明细</span></template>
      <ReportTable
        :rows="result?.geoRows ?? []"
        :metrics="filters.metrics"
        :loading="loading"
        :total="result?.total ?? 0"
        :subtitle="tableSubtitle"
        :reveal-code="revealCode"
        :page-size="20"
        :default-sort="tableDefaultSort"
        @drawer-close="revealCode = null"
      />
    </el-card>
  </div>
</template>
