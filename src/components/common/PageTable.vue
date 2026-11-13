<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { TableColumn, TableRow } from '../../types/modules'
import { formatCellValue } from '../../types/modules'

const props = withDefaults(
  defineProps<{
    rows: TableRow[]
    columns: TableColumn[]
    loading?: boolean
    total?: number
    subtitle?: string
    pageSize?: number
    drawerTitle?: string
  }>(),
  { loading: false, total: undefined, subtitle: '', pageSize: 10, drawerTitle: '明细' }
)

const page = ref(1)
const sortProp = ref('')
const sortOrder = ref<'ascending' | 'descending' | null>(null)
const drawerRow = ref<TableRow | null>(null)

// 新查询回到第一页并关闭抽屉，避免残留旧数据。
watch(
  () => [props.rows, props.columns],
  () => {
    page.value = 1
    drawerRow.value = null
  }
)
watch(
  () => props.pageSize,
  () => {
    page.value = 1
  }
)

const total = computed(() => props.total ?? props.rows.length)

const sortableProps = computed(() => new Set(props.columns.filter((c) => c.sortable !== false).map((c) => c.prop)))

const sortedRows = computed(() => {
  if (!sortProp.value || !sortOrder.value || !sortableProps.value.has(sortProp.value)) return props.rows
  const dir = sortOrder.value === 'ascending' ? 1 : -1
  return [...props.rows].sort((a, b) => {
    const av = a[sortProp.value]
    const bv = b[sortProp.value]
    if (typeof av === 'number' && typeof bv === 'number') return (av - bv) * dir
    return String(av).localeCompare(String(bv), 'zh') * dir
  })
})

const pagedRows = computed(() =>
  sortedRows.value.slice((page.value - 1) * props.pageSize, page.value * props.pageSize)
)

function onSortChange(payload: { prop: string; order: 'ascending' | 'descending' | null }): void {
  sortProp.value = payload.prop ?? ''
  sortOrder.value = payload.order ?? null
}

function fmtCell(row: TableRow, col: TableColumn): string {
  return formatCellValue(col.format, row[col.prop])
}

const nameProp = computed(() => props.columns[0]?.prop ?? '')

function drawerTitleText(): string {
  if (!drawerRow.value) return props.drawerTitle
  const name = nameProp.value ? drawerRow.value[nameProp.value] : ''
  return name ? `${name} ${props.drawerTitle}` : props.drawerTitle
}
</script>

<template>
  <div class="page-table">
    <el-skeleton v-if="loading && rows.length === 0" :rows="4" animated />
    <el-empty v-else-if="rows.length === 0" description="暂无明细数据" />
    <div v-else v-loading="loading">
      <el-table :data="pagedRows" stripe style="width: 100%" @sort-change="onSortChange" @row-click="drawerRow = $event">
        <el-table-column
          v-for="col in columns"
          :key="col.prop"
          :prop="col.prop"
          :label="col.label"
          :sortable="col.sortable !== false ? 'custom' : false"
          :align="col.align ?? 'left'"
        >
          <template #default="{ row }">{{ fmtCell(row, col) }}</template>
        </el-table-column>
      </el-table>
      <el-pagination
        class="table-pagination"
        :current-page="page"
        :page-size="pageSize"
        :total="total"
        layout="total, prev, pager, next"
        @current-change="page = $event"
      />
    </div>
    <el-drawer
      :model-value="drawerRow !== null"
      :title="drawerTitleText()"
      direction="rtl"
      @update:model-value="(v: boolean) => { if (!v) drawerRow = null }"
    >
      <div v-if="drawerRow">
        <p v-if="subtitle" class="drawer-subtitle">{{ subtitle }}</p>
        <el-descriptions :column="1" border>
          <el-descriptions-item v-for="col in columns" :key="col.prop" :label="col.label">
            {{ fmtCell(drawerRow, col) }}
          </el-descriptions-item>
        </el-descriptions>
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
</style>
