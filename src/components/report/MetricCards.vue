<script setup lang="ts">
import { computed } from 'vue'
import type { MetricKey, ReportResult } from '../../types/report'
import { METRIC_LABELS, formatChangeRate, formatMetricValue } from '../../types/report'

const props = withDefaults(
  defineProps<{
    result: ReportResult | null
    metrics: MetricKey[]
    loading?: boolean
  }>(),
  { loading: false }
)

// 只展示当前选中指标；取消勾选后卡片即时收起，数值待提交后刷新。
const shown = computed(() => (props.result?.summary ?? []).filter((s) => props.metrics.includes(s.key)))
</script>

<template>
  <div class="metric-cards">
    <el-skeleton v-if="loading && !result" :rows="2" animated />
    <el-empty v-else-if="!result || shown.length === 0" description="暂无指标数据" />
    <el-row v-else :gutter="12">
      <el-col v-for="item in shown" :key="item.key" :xs="12" :sm="8" :md="6">
        <el-card class="metric-card" shadow="hover">
          <div class="metric-label">{{ METRIC_LABELS[item.key] }}</div>
          <div class="metric-value">{{ formatMetricValue(item.key, item.value) }}</div>
          <div
            class="metric-change"
            :class="{ up: (item.changeRate ?? 0) > 0, down: (item.changeRate ?? 0) < 0 }"
          >
            环比 {{ formatChangeRate(item.changeRate) }}
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<style scoped>
.metric-label {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.metric-value {
  font-size: 24px;
  font-weight: 600;
  margin: 4px 0;
}
.metric-change {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.metric-change.up {
  color: var(--el-color-danger);
}
.metric-change.down {
  color: var(--el-color-success);
}
</style>
