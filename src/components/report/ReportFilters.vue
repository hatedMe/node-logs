<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import type { MetricKey, ReportFilters } from '../../types/report'
import {
  GEO_LEVEL_OPTIONS,
  MAX_METRICS,
  METRIC_DESCRIPTIONS,
  METRIC_GROUPS,
  METRIC_LABELS,
  METRIC_ORDER,
  NATIONWIDE,
  PROVINCE_NAMES,
  RANGE_PRESET_OPTIONS,
  SEARCH_ENGINE_OPTIONS,
  SOURCE_OPTIONS,
  VISITOR_OPTIONS
} from '../../types/report'
import { addDays, eachDay, getPresetRange } from '../../utils/date'
import { validateFilters } from '../../composables/useReport'

const props = withDefaults(
  defineProps<{
    filters: ReportFilters
    loading?: boolean
    canDownload?: boolean
  }>(),
  { loading: false, canDownload: false }
)

const emit = defineEmits<{
  (e: 'update:filters', value: ReportFilters): void
  (e: 'submit'): void
  (e: 'reset'): void
  (e: 'download'): void
}>()

const local = ref<ReportFilters>(clone(props.filters))
const errors = ref<string[]>([])
// 高级筛选区默认展开；收起用 v-show，DOM 常在，旧测试不受影响。
const advancedOpen = ref(true)

function clone(f: ReportFilters): ReportFilters {
  return { ...f, metrics: [...f.metrics], areas: [...(f.areas ?? [])], engines: [...(f.engines ?? [])] }
}

watch(
  () => props.filters,
  (v) => {
    local.value = clone(v)
  },
  { deep: true }
)

function update(patch: Partial<ReportFilters>): void {
  local.value = { ...local.value, ...patch }
  errors.value = []
  emit('update:filters', clone(local.value))
}

function onPreset(preset: ReportFilters['rangePreset']): void {
  if (preset === 'custom') {
    update({ rangePreset: preset })
    return
  }
  const { startDate, endDate } = getPresetRange(preset)
  update({ rangePreset: preset, startDate, endDate })
}

function onStartDate(v: string | null): void {
  local.value.startDate = v ?? ''
  update({ startDate: local.value.startDate, endDate: local.value.endDate, rangePreset: 'custom' })
}

function onEndDate(v: string | null): void {
  local.value.endDate = v ?? ''
  update({ startDate: local.value.startDate, endDate: local.value.endDate, rangePreset: 'custom' })
}

function onCompareToggle(enabled: boolean): void {
  if (!enabled) {
    update({ compareEnabled: false })
    return
  }
  let { compareStartDate, compareEndDate } = local.value
  if (!compareStartDate || !compareEndDate) {
    const days = eachDay(local.value.startDate, local.value.endDate).length || 7
    compareEndDate = addDays(local.value.startDate, -1)
    compareStartDate = addDays(compareEndDate, -(days - 1))
  }
  update({ compareEnabled: true, compareStartDate, compareEndDate })
}

function onMetrics(next: MetricKey[]): void {
  if (next.length > MAX_METRICS) {
    ElMessage.warning(`指标最多选择 ${MAX_METRICS} 项`)
    return
  }
  update({ metrics: METRIC_ORDER.filter((k) => next.includes(k)) })
}

function isMetricDisabled(k: MetricKey): boolean {
  return local.value.metrics.length >= MAX_METRICS && !local.value.metrics.includes(k)
}

function onSubmit(): void {
  const v = validateFilters(local.value)
  errors.value = v.errors
  if (!v.valid) {
    ElMessage.error(v.errors[0])
    return
  }
  emit('submit')
}
</script>

<template>
  <el-form class="report-filters" label-width="84px" @submit.prevent="onSubmit">
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
        @update:model-value="onStartDate($event)"
      />
      <span class="date-sep">至</span>
      <el-date-picker
        :model-value="local.endDate"
        type="date"
        value-format="YYYY-MM-DD"
        placeholder="结束日期"
        aria-label="结束日期"
        @update:model-value="onEndDate($event)"
      />
    </el-form-item>

    <el-form-item v-show="advancedOpen" label="对比日期">
      <el-switch :model-value="local.compareEnabled" @update:model-value="onCompareToggle($event)" />
      <template v-if="local.compareEnabled">
        <el-date-picker
          :model-value="local.compareStartDate"
          type="date"
          value-format="YYYY-MM-DD"
          placeholder="对比开始日期"
          aria-label="对比开始日期"
          class="compare-date"
          @update:model-value="update({ compareStartDate: $event ?? '' })"
        />
        <span class="date-sep">至</span>
        <el-date-picker
          :model-value="local.compareEndDate"
          type="date"
          value-format="YYYY-MM-DD"
          placeholder="对比结束日期"
          aria-label="对比结束日期"
          @update:model-value="update({ compareEndDate: $event ?? '' })"
        />
      </template>
    </el-form-item>

    <el-form-item label="来源">
      <el-select :model-value="local.source" @update:model-value="update({ source: $event })">
        <el-option v-for="o in SOURCE_OPTIONS" :key="o.value" :value="o.value" :label="o.label" />
      </el-select>
    </el-form-item>

    <el-form-item v-if="local.source === 'search'" label="搜索引擎">
      <el-checkbox-group
        :model-value="local.engines ?? []"
        @update:model-value="update({ engines: $event })"
      >
        <el-checkbox v-for="o in SEARCH_ENGINE_OPTIONS" :key="o.value" :value="o.value">
          {{ o.label }}
        </el-checkbox>
      </el-checkbox-group>
    </el-form-item>

    <el-form-item label="访客类型">
      <el-select :model-value="local.visitorType" @update:model-value="update({ visitorType: $event })">
        <el-option v-for="o in VISITOR_OPTIONS" :key="o.value" :value="o.value" :label="o.label" />
      </el-select>
    </el-form-item>

    <el-form-item v-show="advancedOpen" label="地域筛选">
      <div>
        <el-select
          :model-value="local.areas ?? []"
          multiple
          collapse-tags
          :disabled="local.geoLevel === 'country'"
          placeholder="全部地域（默认）"
          aria-label="地域筛选"
          @update:model-value="update({ areas: $event })"
        >
          <el-option :value="NATIONWIDE" label="全国（全部省份）" />
          <el-option v-for="n in PROVINCE_NAMES" :key="n" :value="n" :label="n" />
        </el-select>
        <p class="areas-hint">按国家查看时不可用；不选即全部地域</p>
      </div>
    </el-form-item>

    <el-form-item label="地域层级">
      <el-radio-group :model-value="local.geoLevel" @update:model-value="update({ geoLevel: $event })">
        <el-radio-button v-for="o in GEO_LEVEL_OPTIONS" :key="o.value" :value="o.value">
          {{ o.label }}
        </el-radio-button>
      </el-radio-group>
    </el-form-item>

    <el-form-item class="advanced-toggle-row">
      <el-button link @click="advancedOpen = !advancedOpen">
        {{ advancedOpen ? '收起高级筛选' : '展开高级筛选' }}
      </el-button>
    </el-form-item>

    <el-form-item v-show="advancedOpen" label="指标选择">
      <div>
        <el-checkbox-group :model-value="local.metrics" @update:model-value="onMetrics($event)">
          <div v-for="g in METRIC_GROUPS" :key="g.key" class="metric-group">
            <p class="metric-group-label">{{ g.label }}</p>
            <el-checkbox
              v-for="k in g.metrics"
              :key="k"
              :value="k"
              :disabled="isMetricDisabled(k)"
              :title="METRIC_DESCRIPTIONS[k]"
            >
              {{ METRIC_LABELS[k] }}
              <span class="metric-desc">{{ METRIC_DESCRIPTIONS[k] }}</span>
            </el-checkbox>
          </div>
        </el-checkbox-group>
        <p class="metrics-hint">最多选择 {{ MAX_METRICS }} 项（已选 {{ local.metrics.length }} 项）</p>
      </div>
    </el-form-item>

    <el-alert
      v-if="errors.length > 0"
      type="error"
      :title="errors[0]"
      :closable="false"
      show-icon
      class="filters-error"
    />

    <el-form-item>
      <el-button type="primary" :loading="loading" @click="onSubmit">查询</el-button>
      <el-button :disabled="loading" @click="emit('reset')">重置</el-button>
      <el-button :disabled="!canDownload" @click="emit('download')">下载 CSV</el-button>
    </el-form-item>
  </el-form>
</template>

<style scoped>
.date-sep {
  margin: 0 8px;
  color: var(--el-text-color-secondary);
}
.compare-date {
  margin-left: 12px;
}
.metrics-hint {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.metric-group-label {
  margin: 8px 0 0;
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-regular);
}
.metric-group:first-child .metric-group-label {
  margin-top: 0;
}
.metric-desc {
  margin-left: 4px;
  font-size: 12px;
  font-weight: 400;
  color: var(--el-text-color-secondary);
}
.areas-hint {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.advanced-toggle-row {
  margin-bottom: 0;
}
.filters-error {
  margin-bottom: 12px;
}
</style>
