<script setup lang="ts">
import { computed, ref } from 'vue'
import type { RankItem } from '../../types/modules'

const props = withDefaults(
  defineProps<{
    title: string
    items: RankItem[]
    loading?: boolean
    limit?: number
  }>(),
  { loading: false, limit: 8 }
)

const emit = defineEmits<{
  (e: 'select', value: RankItem): void
}>()

const topItems = computed(() => props.items.slice(0, props.limit))
const drawerItem = ref<RankItem | null>(null)

function onDetail(item: RankItem): void {
  drawerItem.value = item
  emit('select', item)
}
</script>

<template>
  <div class="rank-list">
    <h3>{{ title }}</h3>
    <el-skeleton v-if="loading && items.length === 0" :rows="3" animated />
    <el-empty v-else-if="items.length === 0" description="暂无排行数据" />
    <ol v-else class="rank-ol">
      <li v-for="(item, i) in topItems" :key="item.name" class="rank-item">
        <span class="rank-no">{{ i + 1 }}</span>
        <span class="rank-name" :title="item.name">{{ item.name }}</span>
        <span class="rank-bar" aria-hidden="true">
          <span class="rank-bar-fill" :style="{ width: `${Math.min(100, Math.max(0, item.share))}%` }" />
        </span>
        <span class="rank-value">{{ item.display }}</span>
        <el-button size="small" @click="onDetail(item)">查看明细</el-button>
      </li>
    </ol>
    <el-drawer
      :model-value="drawerItem !== null"
      :title="drawerItem ? `${drawerItem.name} 明细` : '排行明细'"
      direction="rtl"
      @update:model-value="(v: boolean) => { if (!v) drawerItem = null }"
    >
      <div v-if="drawerItem">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="名称">{{ drawerItem.name }}</el-descriptions-item>
          <el-descriptions-item label="数值">{{ drawerItem.display }}</el-descriptions-item>
          <el-descriptions-item label="占比">{{ drawerItem.share }}%</el-descriptions-item>
        </el-descriptions>
      </div>
    </el-drawer>
  </div>
</template>

<style scoped>
.rank-list h3 {
  margin: 0 0 8px;
  font-size: 15px;
}
.rank-ol {
  list-style: none;
  margin: 0;
  padding: 0;
}
.rank-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.rank-no {
  width: 18px;
  color: var(--el-text-color-secondary);
}
.rank-name {
  width: 110px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.rank-bar {
  flex: 1;
  height: 8px;
  background: var(--el-fill-color-light);
  border-radius: 4px;
  overflow: hidden;
}
.rank-bar-fill {
  display: block;
  height: 100%;
  background: var(--el-color-primary);
}
.rank-value {
  min-width: 64px;
  text-align: right;
  font-variant-numeric: tabular-nums;
}
</style>
