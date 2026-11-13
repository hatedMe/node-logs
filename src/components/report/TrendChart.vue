<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { MetricKey, TrendPoint } from '../../types/report'
import { METRIC_LABELS, METRIC_ORDER, formatMetricValue } from '../../types/report'

const props = withDefaults(
  defineProps<{
    trend: TrendPoint[]
    metrics: MetricKey[]
    compareEnabled?: boolean
    loading?: boolean
  }>(),
  { compareEnabled: false, loading: false }
)

const COLORS = ['#409eff', '#67c23a', '#e6a23c', '#f56c6c', '#909399', '#a052ff', '#00b8a9', '#ff7ab8']
const W = 640
const H = 260
const PAD = { l: 48, r: 12, t: 12, b: 28 }

function colorOf(k: MetricKey): string {
  return COLORS[METRIC_ORDER.indexOf(k) % COLORS.length]
}

const hidden = ref<MetricKey[]>([])
watch(
  () => props.metrics,
  () => {
    hidden.value = []
  }
)
const visible = computed(() => props.metrics.filter((k) => !hidden.value.includes(k)))

function toggle(k: MetricKey): void {
  hidden.value = hidden.value.includes(k)
    ? hidden.value.filter((x) => x !== k)
    : [...hidden.value, k]
}

// 各指标按自身峰值归一化到 0–100%：PV（万级）与跳出率（0–1）等不再共用单一量级，避免小量级被压平。
const peakOf = computed(() => {
  const peaks = {} as Record<MetricKey, number>
  for (const k of visible.value) {
    let m = 0
    for (const p of props.trend) {
      m = Math.max(m, p.values[k], p.compareValues?.[k] ?? 0)
    }
    peaks[k] = m
  }
  return peaks
})

function x(i: number, n: number): number {
  if (n <= 1) return (W - PAD.l - PAD.r) / 2 + PAD.l
  return PAD.l + (i * (W - PAD.l - PAD.r)) / (n - 1)
}

function ynorm(v: number, k: MetricKey): number {
  const peak = peakOf.value[k]
  const ratio = peak > 0 ? v / peak : 0
  return H - PAD.b - ratio * (H - PAD.t - PAD.b)
}

function points(k: MetricKey, cmp: boolean): string {
  return props.trend
    .map((p, i) => {
      const v = cmp ? (p.compareValues?.[k] ?? 0) : p.values[k]
      return `${x(i, props.trend.length).toFixed(1)},${ynorm(v, k).toFixed(1)}`
    })
    .join(' ')
}

const gridLines = computed(() =>
  [0, 25, 50, 75, 100].map((pct) => ({
    y: H - PAD.b - (pct / 100) * (H - PAD.t - PAD.b),
    label: `${pct}%`
  }))
)

const xLabels = computed(() => {
  const n = props.trend.length
  const step = Math.max(1, Math.ceil(n / 7))
  return props.trend
    .map((p, i) => ({ x: x(i, n), label: p.date.slice(5) }))
    .filter((_, i) => i % step === 0)
})

const hovered = ref<number | null>(null)

// 仅当对比数据真实存在时才画虚线：用户刚开启对比开关、尚未提交时不画零线误导。
const hasCompareData = computed(
  () => props.compareEnabled && props.trend.some((p) => p.compareValues !== null)
)

// 读屏用等效文本：区间、归一化说明与各指标首末实际值。
const chartSummary = computed(() => {
  if (props.trend.length === 0 || visible.value.length === 0) return '趋势图暂无数据'
  const first = props.trend[0].date
  const last = props.trend[props.trend.length - 1].date
  const parts = visible.value.map((k) => {
    const vals = props.trend.map((p) => formatMetricValue(k, p.values[k]))
    return `${METRIC_LABELS[k]}由${vals[0]}变为${vals[vals.length - 1]}`
  })
  return `趋势图（${first}至${last}，各指标按峰值归一化）：${parts.join('；')}`
})
</script>

<template>
  <div class="trend-chart">
    <el-skeleton v-if="loading && trend.length === 0" :rows="4" animated />
    <el-empty v-else-if="trend.length === 0 || visible.length === 0" description="暂无趋势数据" />
    <template v-else>
      <div class="legend">
        <button
          v-for="k in metrics"
          :key="k"
          type="button"
          class="legend-item"
          :class="{ off: hidden.includes(k) }"
          :aria-pressed="!hidden.includes(k)"
          @click="toggle(k)"
        >
          <span class="swatch" :style="{ background: colorOf(k) }" />
          {{ METRIC_LABELS[k] }}
        </button>
        <span v-if="hasCompareData" class="legend-compare">虚线：对比区间</span>
        <span class="legend-note">各指标按自身峰值归一化（%），悬停查看实际值</span>
      </div>
      <div class="chart-wrap">
        <svg :viewBox="`0 0 ${W} ${H}`" role="img" :aria-label="chartSummary">
          <line
            v-for="(g, i) in gridLines"
            :key="i"
            :x1="PAD.l"
            :x2="W - PAD.r"
            :y1="g.y"
            :y2="g.y"
            class="grid"
          />
          <text v-for="(g, i) in gridLines" :key="`t${i}`" :x="PAD.l - 4" :y="g.y + 4" class="axis">
            {{ g.label }}
          </text>
          <text
            v-for="(d, i) in xLabels"
            :key="`x${i}`"
            :x="d.x"
            :y="H - 8"
            text-anchor="middle"
            class="axis"
          >
            {{ d.label }}
          </text>
          <template v-for="k in visible" :key="k">
            <polyline
              :points="points(k, false)"
              fill="none"
              :stroke="colorOf(k)"
              stroke-width="2"
              class="trend-line"
              :data-metric="k"
            />
            <polyline
              v-if="hasCompareData"
              :points="points(k, true)"
              fill="none"
              :stroke="colorOf(k)"
              stroke-width="2"
              stroke-dasharray="5 4"
              class="trend-line compare-line"
              :data-metric="k"
            />
          </template>
          <template v-if="trend.length === 1">
            <circle
              v-for="k in visible"
              :key="`single-${k}`"
              :cx="x(0, 1)"
              :cy="ynorm(trend[0].values[k], k)"
              r="4"
              :fill="colorOf(k)"
              class="single-dot"
            />
          </template>
          <circle
            v-for="(p, i) in trend"
            :key="p.date"
            :cx="x(i, trend.length)"
            :cy="ynorm(p.values[visible[0]], visible[0])"
            r="9"
            fill="transparent"
            class="hover-dot"
            tabindex="0"
            @mouseover="hovered = i"
            @mouseleave="hovered = null"
            @focus="hovered = i"
            @blur="hovered = null"
          />
          <circle
            v-if="hovered !== null"
            :cx="x(hovered, trend.length)"
            :cy="ynorm(trend[hovered].values[visible[0]], visible[0])"
            r="4"
            :fill="colorOf(visible[0])"
            class="hover-mark"
          />
        </svg>
        <div v-if="hovered !== null" class="tooltip" role="status">
          <div class="tooltip-date">{{ trend[hovered].date }}</div>
          <div v-for="k in visible" :key="k">
            {{ METRIC_LABELS[k] }}：{{ formatMetricValue(k, trend[hovered].values[k]) }}
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.legend {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  margin-bottom: 8px;
}
.legend-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 1px solid var(--el-border-color);
  background: var(--el-bg-color);
  border-radius: 12px;
  padding: 2px 10px;
  font-size: 12px;
  cursor: pointer;
}
.legend-item.off {
  opacity: 0.45;
}
.swatch {
  width: 10px;
  height: 10px;
  border-radius: 2px;
  display: inline-block;
}
.legend-compare,
.legend-note {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.chart-wrap {
  position: relative;
}
.chart-wrap svg {
  width: 100%;
  height: auto;
  display: block;
}
.grid {
  stroke: var(--el-border-color-lighter);
  stroke-width: 1;
}
.axis {
  font-size: 10px;
  fill: var(--el-text-color-secondary);
  text-anchor: end;
}
text.axis[text-anchor='middle'] {
  text-anchor: middle;
}
.hover-dot {
  cursor: pointer;
  outline: none;
}
.tooltip {
  position: absolute;
  top: 8px;
  left: 56px;
  background: var(--el-bg-color-overlay);
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  padding: 8px 10px;
  font-size: 12px;
  box-shadow: var(--el-box-shadow-light);
  pointer-events: none;
}
.tooltip-date {
  font-weight: 600;
  margin-bottom: 4px;
}
</style>
