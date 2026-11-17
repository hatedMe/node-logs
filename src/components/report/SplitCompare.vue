// 新老访客对比环（原版新老访客页的双环占比图）。

<script setup lang="ts">
import type { RankItem } from '../../types/modules'

defineProps<{
  items: RankItem[]
}>()
</script>

<template>
  <div class="visitor-split">
    <div v-for="item in items" :key="item.name">
      <strong>{{ item.share }}%</strong>
      <span>{{ item.name }} {{ item.display }}</span>
    </div>
  </div>
</template>
