<script setup lang="ts">
import { computed } from 'vue'
import type { GeoLevel, GeoRow } from '../../types/report'
import { formatMetricValue } from '../../types/report'

const props = withDefaults(
  defineProps<{
    rows: GeoRow[]
    geoLevel: GeoLevel
    loading?: boolean
  }>(),
  { loading: false }
)

const emit = defineEmits<{
  (e: 'view-detail', value: { name: string; code: string }): void
}>()

const title = computed(() => (props.geoLevel === 'country' ? '地域排行（按国家）' : '地域排行（按省）'))
const totalPv = computed(() => props.rows.reduce((s, r) => s + r.values.pv, 0))
const topRows = computed(() => [...props.rows].sort((a, b) => b.values.pv - a.values.pv).slice(0, 8))

function share(row: GeoRow): string {
  if (totalPv.value <= 0) return '0'
  return ((row.values.pv / totalPv.value) * 100).toFixed(1)
}
</script>

<template>
  <div class="geo-view">
    <h3>{{ title }}</h3>
    <p class="geo-summary">共 {{ rows.length }} 个地域，PV 合计 {{ formatMetricValue('pv', totalPv) }}</p>
    <el-skeleton v-if="loading && rows.length === 0" :rows="3" animated />
    <el-empty v-else-if="rows.length === 0" description="暂无地域数据" />
    <ol v-else class="geo-rank">
      <li v-for="(row, i) in topRows" :key="row.code" class="geo-item">
        <span class="geo-rank-no">{{ i + 1 }}</span>
        <span class="geo-name">{{ row.name }}</span>
        <span class="geo-bar" aria-hidden="true">
          <span class="geo-bar-fill" :style="{ width: `${share(row)}%` }" />
        </span>
        <span class="geo-pv">{{ formatMetricValue('pv', row.values.pv) }}</span>
        <el-button size="small" @click="emit('view-detail', { name: row.name, code: row.code })">
          查看明细
        </el-button>
      </li>
    </ol>
  </div>
</template>

<style scoped>
.geo-view h3 {
  margin: 0 0 4px;
  font-size: 15px;
}
.geo-summary {
  margin: 0 0 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.geo-rank {
  list-style: none;
  margin: 0;
  padding: 0;
}
.geo-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.geo-rank-no {
  width: 18px;
  color: var(--el-text-color-secondary);
}
.geo-name {
  width: 64px;
}
.geo-bar {
  flex: 1;
  height: 8px;
  background: var(--el-fill-color-light);
  border-radius: 4px;
  overflow: hidden;
}
.geo-bar-fill {
  display: block;
  height: 100%;
  background: var(--el-color-primary);
}
.geo-pv {
  width: 80px;
  text-align: right;
  font-variant-numeric: tabular-nums;
}
</style>
