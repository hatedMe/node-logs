// 横向条形排行（事件/子目录/系统环境/入口 TOP 等占比可视化）。

<script setup lang="ts">
import { computed } from 'vue'
import type { RankItem } from '../../types/modules'

const props = withDefaults(
  defineProps<{
    items: RankItem[]
    title?: string
    valueLabel?: string
    loading?: boolean
  }>(),
  { title: '', valueLabel: '数值', loading: false }
)

const maxShare = computed(() => Math.max(1, ...props.items.map((i) => i.share)))

function widthOf(item: RankItem): number {
  return Math.min(100, (item.share / maxShare.value) * 100)
}
</script>

<template>
  <div class="bar-list">
    <h3 v-if="title" class="bar-title">{{ title }}</h3>
    <el-skeleton v-if="loading && items.length === 0" :rows="4" animated />
    <el-empty v-else-if="items.length === 0" description="暂无排行数据" />
    <div v-else class="bar-rows">
      <div v-for="item in items" :key="item.name" class="bar-row">
        <span class="bar-name" :title="item.name">{{ item.name }}</span>
        <i class="bar-track"><i class="bar-fill" :style="{ width: `${widthOf(item)}%` }" /></i>
        <b class="bar-value">{{ item.display }}</b>
      </div>
    </div>
  </div>
</template>

<style scoped>
.bar-title {
  margin: 0 0 8px;
  font-size: 15px;
}
.bar-rows {
  padding: 4px 0;
}
.bar-row {
  display: grid;
  grid-template-columns: 140px 1fr 72px;
  align-items: center;
  gap: 12px;
  margin: 14px 0;
  color: #545b66;
  font-size: 12px;
}
.bar-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.bar-track {
  height: 10px;
  border-radius: 2px;
  background: #ebedf5;
}
.bar-fill {
  display: block;
  height: 100%;
  border-radius: 2px;
  background: #5a9bff;
}
.bar-value {
  color: #282c33;
  font-weight: 500;
  text-align: right;
  font-variant-numeric: tabular-nums;
}
</style>
