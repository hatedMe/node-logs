<script setup lang="ts">
import type { StatCard } from '../../types/modules'

withDefaults(
  defineProps<{
    cards: StatCard[]
    loading?: boolean
  }>(),
  { loading: false }
)
</script>

<template>
  <div class="stat-cards">
    <el-skeleton v-if="loading && cards.length === 0" :rows="2" animated />
    <el-empty v-else-if="cards.length === 0" description="暂无指标数据" />
    <el-row v-else :gutter="12">
      <el-col v-for="c in cards" :key="c.key" :xs="12" :sm="8" :md="6">
        <el-card class="stat-card" shadow="hover">
          <div class="stat-label">{{ c.label }}</div>
          <div class="stat-value">{{ c.value }}</div>
          <div class="stat-change" :class="{ up: c.up === true, down: c.up === false }">
            环比 {{ c.change }}
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<style scoped>
.stat-label {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.stat-value {
  font-size: 24px;
  font-weight: 600;
  margin: 4px 0;
}
.stat-change {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.stat-change.up {
  color: var(--el-color-danger);
}
.stat-change.down {
  color: var(--el-color-success);
}
</style>
