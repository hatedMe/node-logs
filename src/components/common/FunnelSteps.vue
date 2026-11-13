<script setup lang="ts">
import { computed } from 'vue'
import type { FunnelStep } from '../../types/modules'

const props = withDefaults(
  defineProps<{
    title: string
    steps: FunnelStep[]
    loading?: boolean
  }>(),
  { loading: false }
)

const first = computed(() => props.steps[0]?.value ?? 0)

function widthOf(v: number): number {
  if (first.value <= 0) return 0
  return Math.min(100, Math.max(2, (v / first.value) * 100))
}
</script>

<template>
  <div class="funnel-steps">
    <h3>{{ title }}</h3>
    <el-skeleton v-if="loading && steps.length === 0" :rows="3" animated />
    <el-empty v-else-if="steps.length === 0" description="暂无漏斗数据" />
    <ol v-else class="funnel-ol">
      <li v-for="(s, i) in steps" :key="s.label" class="funnel-item">
        <span class="funnel-label">{{ i + 1 }}. {{ s.label }}</span>
        <span class="funnel-bar" aria-hidden="true">
          <span class="funnel-bar-fill" :style="{ width: `${widthOf(s.value)}%` }" />
        </span>
        <span class="funnel-value">{{ s.display }}</span>
        <span class="funnel-rate">{{ s.rate }}</span>
      </li>
    </ol>
  </div>
</template>

<style scoped>
.funnel-steps h3 {
  margin: 0 0 8px;
  font-size: 15px;
}
.funnel-ol {
  list-style: none;
  margin: 0;
  padding: 0;
}
.funnel-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
}
.funnel-label {
  width: 150px;
}
.funnel-bar {
  flex: 1;
  height: 18px;
  background: var(--el-fill-color-light);
  border-radius: 4px;
  overflow: hidden;
}
.funnel-bar-fill {
  display: block;
  height: 100%;
  background: linear-gradient(90deg, var(--el-color-primary), var(--el-color-success));
}
.funnel-value {
  width: 80px;
  text-align: right;
  font-variant-numeric: tabular-nums;
}
.funnel-rate {
  width: 64px;
  text-align: right;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
</style>
