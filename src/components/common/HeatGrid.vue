<script setup lang="ts">
import { ref } from 'vue'
import type { HeatCell } from '../../types/modules'

defineProps<{
  cells: HeatCell[]
  loading?: boolean
}>()

const selected = ref<HeatCell | null>(null)

/** 点击强度映射为蓝→红透明度，保证浅色/深色主题都可读（数值同时用文字展示）。 */
function cellStyle(intensity: number): Record<string, string> {
  const t = Math.min(1, Math.max(0, intensity))
  const r = Math.round(64 + t * 191)
  const g = Math.round(158 - t * 108)
  const b = Math.round(255 - t * 180)
  return { background: `rgb(${r}, ${g}, ${b})` }
}
</script>

<template>
  <div class="heat-grid">
    <el-skeleton v-if="loading && cells.length === 0" :rows="3" animated />
    <el-empty v-else-if="cells.length === 0" description="暂无点击数据" />
    <div v-else class="heat-cells" role="list">
      <button
        v-for="c in cells"
        :key="c.label"
        type="button"
        role="listitem"
        class="heat-cell"
        :style="cellStyle(c.intensity)"
        :title="`${c.label}：${c.display}`"
        @click="selected = c"
      >
        <span class="heat-label">{{ c.label }}</span>
        <span class="heat-value">{{ c.display }}</span>
      </button>
    </div>
    <el-drawer
      :model-value="selected !== null"
      :title="selected ? `${selected.label} 点击明细` : '点击明细'"
      direction="rtl"
      @update:model-value="(v: boolean) => { if (!v) selected = null }"
    >
      <div v-if="selected">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="区域">{{ selected.label }}</el-descriptions-item>
          <el-descriptions-item label="点击次数">{{ selected.display }}</el-descriptions-item>
          <el-descriptions-item label="点击强度">{{ (selected.intensity * 100).toFixed(1) }}%</el-descriptions-item>
        </el-descriptions>
      </div>
    </el-drawer>
  </div>
</template>

<style scoped>
.heat-cells {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
}
.heat-cell {
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  padding: 12px 8px;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: center;
  color: #fff;
  text-shadow: 0 1px 2px rgb(0 0 0 / 40%);
}
.heat-label {
  font-size: 12px;
}
.heat-value {
  font-size: 16px;
  font-weight: 600;
}
@media (max-width: 768px) {
  .heat-cells {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
