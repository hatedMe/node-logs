// 指标条：原版报表页顶部的“核心指标/今日流量”数字条（.navigation-report__metrics）。

<script setup lang="ts">
import type { StatCard } from '../../types/modules'

withDefaults(
  defineProps<{
    label: string
    cards: StatCard[]
    loading?: boolean
  }>(),
  { loading: false }
)
</script>

<template>
  <div class="navigation-report__metrics" :class="{ 'is-empty': !loading && cards.length === 0 }">
    <div class="metrics-label">{{ label }}</div>
    <template v-if="loading && cards.length === 0">
      <article v-for="n in 4" :key="n"><span>加载中</span><strong>--</strong><small>较上期 -</small></article>
    </template>
    <template v-else>
      <article v-for="c in cards" :key="c.key">
        <span>{{ c.label }}</span>
        <strong>{{ c.value }}</strong>
        <small :class="{ up: c.up === true, down: c.up === false }">较上期 {{ c.change }}</small>
      </article>
    </template>
  </div>
</template>

<style scoped>
.metrics-label {
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 5px;
  color: #282c33;
  font-size: 14px;
}
.metrics-label ~ article {
  min-height: 76px;
  padding: 0 18px;
  border-left: 1px solid #e2e6f0;
}
small.up {
  color: #13805a;
}
small.down {
  color: #c35c49;
}
</style>
