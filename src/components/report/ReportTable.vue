<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { GeoRow, MetricKey } from '../../types/report'
import { METRIC_LABELS, formatMetricValue } from '../../types/report'

const props = withDefaults(
  defineProps<{
    rows: GeoRow[]
    metrics: MetricKey[]
    loading?: boolean
    total: number
    subtitle?: string
    /** 外部联动钻取的地域编码：变化时跳到所在页并打开抽屉。 */
    revealCode?: string | null
    /** 每页条数：默认 10；地域分布页传入 20。 */
    pageSize?: number
    /** 初始排序：默认无；地域分布列表默认 PV 降序。 */
    defaultSort?: { prop: string; order: 'ascending' | 'descending' } | null
  }>(),
  { loading: false, subtitle: '', revealCode: null, pageSize: 10, defaultSort: null }
)

const emit = defineEmits<{
  (e: 'sort-change', value: { prop: string; order: 'ascending' | 'descending' | null }): void
  (e: 'page-change', value: { page: number; pageSize: number }): void
  (e: 'drilldown', value: GeoRow): void
  (e: 'drawer-close'): void
}>()

const page = ref(1)
const sortProp = ref(props.defaultSort?.prop ?? '')
const sortOrder = ref<'ascending' | 'descending' | null>(props.defaultSort?.order ?? null)
const drawerRow = ref<GeoRow | null>(null)
const drawerVisible = computed({
  get: () => drawerRow.value !== null,
  set: (value: boolean) => onDrawerVisible(value)
})

// 新查询回到第一页并关闭抽屉，避免残留旧地域数据。
watch(
  () => [props.rows, props.total],
  () => {
    page.value = 1
    drawerRow.value = null
  }
)

// 地域排行联动：跳到该地域所在页、高亮行并打开抽屉。
watch(
  () => props.revealCode,
  (code) => {
    if (!code) return
    const idx = sortedRows.value.findIndex((r) => r.code === code)
    if (idx < 0) return
    page.value = Math.floor(idx / props.pageSize) + 1
    drawerRow.value = sortedRows.value[idx]
    emit('drilldown', sortedRows.value[idx])
  }
)

// 排序列被取消选择时清除排序。
watch(
  () => props.metrics,
  (m) => {
    if (sortProp.value !== '' && sortProp.value !== 'name' && !m.includes(sortProp.value as MetricKey)) {
      sortProp.value = ''
      sortOrder.value = null
    }
  }
)

const sortedRows = computed(() => {
  if (!sortProp.value || !sortOrder.value) return props.rows
  const dir = sortOrder.value === 'ascending' ? 1 : -1
  return [...props.rows].sort((a, b) => {
    if (sortProp.value === 'name') return a.name.localeCompare(b.name, 'zh') * dir
    const k = sortProp.value as MetricKey
    return (a.values[k] - b.values[k]) * dir
  })
})

const pagedRows = computed(() => sortedRows.value.slice((page.value - 1) * props.pageSize, page.value * props.pageSize))

function onSortChange(payload: { prop: string; order: 'ascending' | 'descending' | null }): void {
  sortProp.value = payload.prop ?? ''
  sortOrder.value = payload.order ?? null
  emit('sort-change', { prop: sortProp.value, order: sortOrder.value })
}

function onPageChange(p: number): void {
  page.value = p
  emit('page-change', { page: p, pageSize: props.pageSize })
}

function onRowClick(row: GeoRow): void {
  drawerRow.value = row
  emit('drilldown', row)
}

function onDrawerVisible(v: boolean): void {
  if (v) return
  drawerRow.value = null
  emit('drawer-close')
}

function rowClassName(data: { row: GeoRow; rowIndex: number }): string {
  void data.rowIndex
  return data.row.code === props.revealCode && props.revealCode !== null ? 'revealed-row' : ''
}

function fmtCell(row: GeoRow, k: MetricKey): string {
  return formatMetricValue(k, row.values[k])
}

// 抽屉内的来源构成：按固定占比由该地域 PV 推导的展示数据。
const SOURCE_SHARES = [
  { label: '搜索引擎', share: 0.41 },
  { label: '直接访问', share: 0.32 },
  { label: '外部链接', share: 0.18 },
  { label: '其他来源', share: 0.09 }
]

const drawerSources = computed(() => {
  if (!drawerRow.value) return []
  const pv = drawerRow.value.values.pv
  return SOURCE_SHARES.map((s) => ({
    label: s.label,
    pv: Math.round(pv * s.share),
    share: `${(s.share * 100).toFixed(0)}%`
  }))
})
</script>

<template>
  <div class="report-table">
    <el-skeleton v-if="loading && rows.length === 0" :rows="4" animated />
    <el-empty v-else-if="rows.length === 0" description="暂无明细数据" />
    <div v-else v-loading="loading">
      <el-table :data="pagedRows" stripe style="width: 100%" :default-sort="props.defaultSort ?? undefined" :row-class-name="rowClassName" @sort-change="onSortChange" @row-click="onRowClick">
        <el-table-column prop="name" label="地域" sortable="custom" />
        <el-table-column
          v-for="k in metrics"
          :key="k"
          :prop="k"
          :label="METRIC_LABELS[k]"
          sortable="custom"
          align="right"
        >
          <template #default="{ row }">{{ fmtCell(row, k) }}</template>
        </el-table-column>
      </el-table>
      <el-pagination
        class="table-pagination"
        :current-page="page"
        :page-size="props.pageSize"
        :total="total"
        layout="total, prev, pager, next"
        @current-change="onPageChange"
      />
    </div>
    <el-drawer
      v-model="drawerVisible"
      :title="drawerRow ? `${drawerRow.name} 明细` : '地域明细'"
      direction="rtl"
    >
      <div v-if="drawerRow">
        <p v-if="subtitle" class="drawer-subtitle">{{ subtitle }}</p>
        <el-descriptions :column="1" border>
          <el-descriptions-item
            v-for="k in metrics"
            :key="k"
            :label="METRIC_LABELS[k]"
          >
            {{ fmtCell(drawerRow, k) }}
          </el-descriptions-item>
        </el-descriptions>
        <h4>来源构成（模拟）</h4>
        <el-table :data="drawerSources" size="small">
          <el-table-column prop="label" label="来源" />
          <el-table-column prop="pv" label="PV" align="right" />
          <el-table-column prop="share" label="占比" align="right" />
        </el-table>
      </div>
    </el-drawer>
  </div>
</template>

<style scoped>
.table-pagination {
  margin-top: 12px;
  justify-content: flex-end;
}
.drawer-subtitle {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin: 0 0 12px;
}
/* 联动高亮行：背景 + 左侧色条，不只依赖颜色（同时打开了抽屉）。 */
:deep(.revealed-row > td.el-table__cell) {
  background-color: var(--el-color-primary-light-9);
}
:deep(.revealed-row > td.el-table__cell:first-child) {
  box-shadow: inset 3px 0 0 var(--el-color-primary);
}
</style>
