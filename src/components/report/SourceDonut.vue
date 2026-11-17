// 来源构成环图 + 图例：按占比渲染 conic-gradient，悬停/点击图例可联动明细表。

<script setup lang="ts">
import { computed } from 'vue'
import type { RankItem } from '../../types/modules'

const props = withDefaults(
  defineProps<{
    items: RankItem[]
    title?: string
    loading?: boolean
  }>(),
  { title: '来源构成', loading: false }
)

const TONES = ['#1a6eff', '#ff9448', '#5a9bff', '#5ec8a0', '#a052ff', '#ff7ab8', '#95bcff', '#d3d9e6']

const total = computed(() => props.items.reduce((s, r) => s + r.share, 0) || 1)

/** conic-gradient 角度按占比累计，视觉与原版占比环一致。 */
const donutStyle = computed(() => {
  let acc = 0
  const stops = props.items.map((item, i) => {
    const from = acc
    acc += (item.share / total.value) * 360
    return `${TONES[i % TONES.length]} ${from.toFixed(1)}deg ${acc.toFixed(1)}deg`
  })
  return { background: `conic-gradient(${stops.join(', ')})` }
})

function toneOf(i: number): string {
  return TONES[i % TONES.length]
}
</script>

<template>
  <div class="source-donut-wrap">
    <h3 v-if="title" class="donut-title">{{ title }}</h3>
    <el-skeleton v-if="loading && items.length === 0" :rows="3" animated />
    <el-empty v-else-if="items.length === 0" description="暂无构成数据" />
    <template v-else>
      <div class="donut" :style="donutStyle" role="img" :aria-label="`${title}：${items.map((i) => `${i.name} ${i.share}%`).join('、')}`">
        <span class="donut-hole">{{ title }}</span>
      </div>
      <div class="source-legend">
        <span v-for="(item, i) in items" :key="item.name">
          <i :style="{ background: toneOf(i) }" />{{ item.name }}<b>{{ item.share }}%</b>
        </span>
      </div>
    </template>
  </div>
</template>

<style scoped>
.donut-title {
  margin: 0 0 8px;
  font-size: 15px;
}
.donut {
  position: relative;
  width: 150px;
  height: 150px;
  margin: 22px auto 16px;
  border-radius: 50%;
}
.donut-hole {
  position: absolute;
  inset: 34px;
  display: grid;
  place-items: center;
  border-radius: 50%;
  background: #fff;
  color: #282c33;
  font-size: 12px;
  text-align: center;
}
.source-legend span {
  gap: 6px;
}
.source-legend i {
  background: #1a6eff;
}
.source-legend b {
  margin-left: auto;
}
</style>
