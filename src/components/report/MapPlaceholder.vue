<script setup lang="ts">
import { computed } from 'vue'
import type { GeoRow } from '../../types/report'
import { formatMetricValue } from '../../types/report'

// 地域分布地图占位：气泡大小表示 PV 占比，点击地域按钮与明细表联动钻取。
// 真实地图组件接入后替换此占位，view-detail 协议保持不变。

const props = withDefaults(
  defineProps<{
    rows: GeoRow[]
    loading?: boolean
  }>(),
  { loading: false }
)

const emit = defineEmits<{
  (e: 'view-detail', value: { name: string; code: string }): void
}>()

const W = 640
const H = 220

const totalPv = computed(() => props.rows.reduce((s, r) => s + r.values.pv, 0))
const topRows = computed(() => [...props.rows].sort((a, b) => b.values.pv - a.values.pv).slice(0, 8))
const maxPv = computed(() => topRows.value[0]?.values.pv ?? 0)

function radius(pv: number): number {
  if (maxPv.value <= 0) return 0
  return 14 + (pv / maxPv.value) * 34
}

function cx(i: number): number {
  return 70 + i * 72
}

const summary = computed(() => {
  if (topRows.value.length === 0) return '地图暂无数据'
  return `地域分布（PV 前 ${topRows.value.length}）：${topRows.value.map((r) => r.name).join('、')}`
})
</script>

<template>
  <div class="map-placeholder">
    <h3>地域分布地图（占位）</h3>
    <p class="map-summary">共 {{ rows.length }} 个地域，PV 合计 {{ formatMetricValue('pv', totalPv) }}</p>
    <el-skeleton v-if="loading && rows.length === 0" :rows="3" animated />
    <el-empty v-else-if="rows.length === 0" description="暂无地域数据" />
    <template v-else>
      <svg :viewBox="`0 0 ${W} ${H}`" role="img" :aria-label="summary" class="map-svg">
        <g v-for="(row, i) in topRows" :key="row.code">
          <circle :cx="cx(i)" :cy="90" :r="radius(row.values.pv)" class="map-bubble" />
          <text :x="cx(i)" :y="88" text-anchor="middle" class="map-pv">
            {{ formatMetricValue('pv', row.values.pv) }}
          </text>
          <text :x="cx(i)" :y="168" text-anchor="middle" class="map-name">{{ row.name }}</text>
        </g>
      </svg>
      <p class="map-caption">地图为占位示意：气泡大小表示 PV 占比，点击下方地域查看明细</p>
      <div class="map-regions">
        <el-button
          v-for="row in topRows"
          :key="row.code"
          size="small"
          @click="emit('view-detail', { name: row.name, code: row.code })"
        >
          {{ row.name }}
        </el-button>
      </div>
    </template>
  </div>
</template>

<style scoped>
.map-placeholder h3 {
  margin: 0 0 4px;
  font-size: 15px;
}
.map-summary {
  margin: 0 0 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.map-svg {
  width: 100%;
  height: auto;
  display: block;
  border: 1px dashed var(--el-border-color);
  border-radius: 6px;
  background: var(--el-fill-color-lighter);
}
.map-bubble {
  fill: var(--el-color-primary-light-5);
  stroke: var(--el-color-primary);
  stroke-width: 1.5;
  opacity: 0.85;
}
.map-pv {
  font-size: 13px;
  font-weight: 600;
  fill: var(--el-text-color-primary);
}
.map-name {
  font-size: 13px;
  fill: var(--el-text-color-regular);
}
.map-caption {
  margin: 8px 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.map-regions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
</style>
