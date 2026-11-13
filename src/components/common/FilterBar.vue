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
    showDevice?: boolean
    showKeyword?: boolean
    keywordPlaceholder?: string
  }>(),
  {
    loading: false,
    canExport: false,
    showSource: true,
    showDevice: true,
    showKeyword: false,
    keywordPlaceholder: '关键词过滤'
  }
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
  <el-form class="module-filters" label-width="84px" @submit.prevent="onSubmit">
    <el-form-item label="日期预设">
      <el-radio-group :model-value="local.rangePreset" @update:model-value="onPreset($event)">
        <el-radio-button v-for="o in RANGE_PRESET_OPTIONS" :key="o.value" :value="o.value">
          {{ o.label }}
        </el-radio-button>
      </el-radio-group>
    </el-form-item>

    <el-form-item label="日期范围">
      <el-date-picker
        :model-value="local.startDate"
        type="date"
        value-format="YYYY-MM-DD"
        placeholder="开始日期"
        aria-label="开始日期"
        @update:model-value="update({ startDate: $event ?? '', rangePreset: 'custom' })"
      />
      <span class="date-sep">至</span>
      <el-date-picker
        :model-value="local.endDate"
        type="date"
        value-format="YYYY-MM-DD"
        placeholder="结束日期"
        aria-label="结束日期"
        @update:model-value="update({ endDate: $event ?? '', rangePreset: 'custom' })"
      />
    </el-form-item>

    <el-form-item v-if="showSource" label="来源">
      <el-select :model-value="local.source" @update:model-value="update({ source: $event })">
        <el-option v-for="o in SOURCE_OPTIONS" :key="o.value" :value="o.value" :label="o.label" />
      </el-select>
    </el-form-item>

    <el-form-item v-if="showDevice" label="设备">
      <el-select :model-value="local.device" @update:model-value="update({ device: $event })">
        <el-option v-for="o in DEVICE_OPTIONS" :key="o.value" :value="o.value" :label="o.label" />
      </el-select>
    </el-form-item>

    <el-form-item v-if="showKeyword" label="关键词">
      <el-input
        :model-value="local.keyword"
        :placeholder="keywordPlaceholder"
        clearable
        @update:model-value="update({ keyword: $event })"
      />
    </el-form-item>

    <el-alert v-if="error" type="error" :title="error" :closable="false" show-icon class="filters-error" />

    <el-form-item>
      <el-button type="primary" :loading="loading" @click="onSubmit">查询</el-button>
      <el-button :disabled="loading" @click="emit('reset')">重置</el-button>
      <el-button :disabled="!canExport" @click="emit('export')">导出 CSV</el-button>
    </el-form-item>
  </el-form>
</template>

<style scoped>
.date-sep {
  margin: 0 8px;
  color: var(--el-text-color-secondary);
}
.filters-error {
  margin-bottom: 12px;
}
</style>
