// 流转/路径步骤带（页面上下游、转化路径）：步骤卡横向排列，展示人数与转化率。

<script setup lang="ts">
import type { FunnelStep } from '../../types/modules'

withDefaults(
  defineProps<{
    steps: FunnelStep[]
    title?: string
    valueUnit?: string
    loading?: boolean
  }>(),
  { title: '', valueUnit: '人', loading: false }
)
</script>

<template>
  <div class="flow-panel">
    <h3 v-if="title" class="flow-title">{{ title }}</h3>
    <el-skeleton v-if="loading && steps.length === 0" :rows="3" animated />
    <el-empty v-else-if="steps.length === 0" description="暂无路径数据" />
    <div v-else class="flow-track">
      <div v-for="(step, i) in steps" :key="step.label" class="flow-node">
        <b>步骤 {{ i + 1 }}</b>
        <span>{{ step.label }}</span>
        <small>{{ step.display }} {{ valueUnit }} · {{ step.rate }}</small>
      </div>
    </div>
  </div>
</template>

<style scoped>
.flow-title {
  margin: 0;
  font-size: 15px;
}
</style>
