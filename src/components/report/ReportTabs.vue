// 报表页数据 Tab（原版“来源类型 / 来源网站”式切换条）。

<script setup lang="ts">
import type { ReportPageTab } from '../../types/reports'

defineProps<{
  tabs: ReportPageTab[]
  modelValue: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()
</script>

<template>
  <nav class="navigation-report__tabs" role="tablist">
    <button
      v-for="t in tabs"
      :key="t.key"
      type="button"
      role="tab"
      :class="{ 'is-active': t.key === modelValue }"
      :aria-selected="t.key === modelValue"
      @click="emit('update:modelValue', t.key)"
    >
      {{ t.label }}
    </button>
  </nav>
</template>
