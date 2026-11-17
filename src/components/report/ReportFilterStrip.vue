// 报表页筛选条：时间预设/日期范围/设备/来源/关键词 + 查询/重置/导出。
// 视觉沿用原版筛选条（.navigation-report__filters），交互接真实 ModuleFilters。

<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import type { ModuleFilters } from '../../types/modules'
import { DEVICE_OPTIONS } from '../../types/modules'
import { RANGE_PRESET_OPTIONS, SOURCE_OPTIONS } from '../../types/report'
import { getPresetRange, isValidRange } from '../../utils/date'

const props = withDefaults(
  defineProps<{
    filters: ModuleFilters
    loading?: boolean
    canExport?: boolean
    showSource?: boolean
    showKeyword?: boolean
    keywordPlaceholder?: string
  }>(),
  { loading: false, canExport: false, showSource: true, showKeyword: true, keywordPlaceholder: '关键词过滤' }
)

const emit = defineEmits<{
  (e: 'update:filters', value: ModuleFilters): void
  (e: 'submit'): void
  (e: 'reset'): void
  (e: 'export'): void
}>()

const local = ref<ModuleFilters>({ ...props.filters })
const error = ref('')

watch(
  () => props.filters,
  (v) => {
    local.value = { ...v }
  },
  { deep: true }
)

function update(patch: Partial<ModuleFilters>): void {
  local.value = { ...local.value, ...patch }
  error.value = ''
  emit('update:filters', { ...local.value })
}

function onPreset(preset: ModuleFilters['rangePreset']): void {
  if (preset === 'custom') {
    update({ rangePreset: preset })
    return
  }
  const { startDate, endDate } = getPresetRange(preset)
  update({ rangePreset: preset, startDate, endDate })
}

function onSubmit(): void {
  if (!isValidRange(local.value.startDate, local.value.endDate)) {
    error.value = '日期范围无效：结束日期不得早于开始日期'
    ElMessage.error(error.value)
    return
  }
  emit('submit')
}
</script>

<template>
  <div>
    <div class="navigation-report__filters">
      <span>时间：</span>
      <button
        v-for="o in RANGE_PRESET_OPTIONS"
        :key="o.value"
        type="button"
        :class="{ 'is-selected': local.rangePreset === o.value }"
        @click="onPreset(o.value)"
      >
        {{ o.label }}
      </button>
      <span class="filter-date">
        <el-date-picker
          :model-value="local.startDate"
          type="date"
          size="small"
          value-format="YYYY-MM-DD"
          class="filter-date-input"
          aria-label="开始日期"
          @update:model-value="update({ startDate: $event ?? '', rangePreset: 'custom' })"
        />
        至
        <el-date-picker
          :model-value="local.endDate"
          type="date"
          size="small"
          value-format="YYYY-MM-DD"
          class="filter-date-input"
          aria-label="结束日期"
          @update:model-value="update({ endDate: $event ?? '', rangePreset: 'custom' })"
        />
      </span>

      <span>设备：</span>
      <button
        v-for="o in DEVICE_OPTIONS"
        :key="o.value"
        type="button"
        :class="{ 'is-selected': local.device === o.value }"
        @click="update({ device: o.value })"
      >
        {{ o.label }}
      </button>

      <template v-if="showSource">
        <span>来源：</span>
        <button
          v-for="o in SOURCE_OPTIONS"
          :key="o.value"
          type="button"
          :class="{ 'is-selected': local.source === o.value }"
          @click="update({ source: o.value })"
        >
          {{ o.label }}
        </button>
      </template>

      <span v-if="showKeyword">关键词：</span>
      <el-input
        v-if="showKeyword"
        :model-value="local.keyword"
        size="small"
        class="filter-keyword"
        :placeholder="keywordPlaceholder"
        clearable
        aria-label="关键词过滤"
        @update:model-value="update({ keyword: $event })"
        @keyup.enter="onSubmit"
      />

      <span class="filter-actions">
        <button type="button" class="filter-submit" :disabled="loading" @click="onSubmit">查询</button>
        <button type="button" :disabled="loading" @click="emit('reset')">重置</button>
        <button type="button" :disabled="!canExport" @click="emit('export')">导出</button>
      </span>
    </div>
    <el-alert v-if="error" type="error" :title="error" :closable="false" show-icon class="filters-error" />
  </div>
</template>

<style scoped>
.filter-date {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.filter-date-input {
  width: 128px;
}
.filter-keyword {
  width: 160px;
}
.filter-actions {
  display: inline-flex;
  gap: 0;
  margin-left: 14px;
}
.filter-actions button:disabled {
  color: #a8b0bf;
  cursor: not-allowed;
}
.filter-submit {
  color: #fff !important;
  background: #1a6eff !important;
}
.filters-error {
  margin: 8px 0;
}
</style>
