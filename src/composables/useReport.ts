import { computed, ref } from 'vue'
import type { ReportFilters, ReportResult } from '../types/report'
import { DEFAULT_METRICS, MAX_METRICS } from '../types/report'
import { getPresetRange, isValidRange, rangesOverlap } from '../utils/date'
import { fetchReport } from '../mocks/report'

/** 页面默认筛选：最近 7 天、全部来源/访客、按省、PV+UV。 */
export function defaultFilters(): ReportFilters {
  const { startDate, endDate } = getPresetRange('last7')
  return {
    rangePreset: 'last7',
    startDate,
    endDate,
    compareEnabled: false,
    compareStartDate: '',
    compareEndDate: '',
    source: 'all',
    visitorType: 'all',
    geoLevel: 'province',
    metrics: [...DEFAULT_METRICS],
    areas: [],
    engines: []
  }
}

export interface FiltersValidation {
  valid: boolean
  errors: string[]
}

/** 提交前校验：非法时阻止查询并返回明确原因。 */
export function validateFilters(f: ReportFilters): FiltersValidation {
  const errors: string[] = []
  if (!isValidRange(f.startDate, f.endDate)) {
    errors.push('日期范围无效：结束日期不得早于开始日期')
  }
  if (f.metrics.length === 0) {
    errors.push('请至少选择 1 项指标')
  } else if (f.metrics.length > MAX_METRICS) {
    errors.push(`指标最多选择 ${MAX_METRICS} 项`)
  }
  if (f.compareEnabled) {
    if (!f.compareStartDate || !f.compareEndDate) {
      errors.push('请完整选择对比日期段')
    } else if (!isValidRange(f.compareStartDate, f.compareEndDate)) {
      errors.push('对比日期范围无效：结束日期不得早于开始日期')
    } else if (rangesOverlap(f.startDate, f.endDate, f.compareStartDate, f.compareEndDate)) {
      errors.push('对比日期段不得与当前日期段重叠')
    }
  }
  return { valid: errors.length === 0, errors }
}

/**
 * 页面级报表状态：唯一数据源。
 * 每次查询递增 request id，只接收最后一次请求结果（防连续切换竞态）；
 * 失败时清空旧结果，不残留旧表格数据。
 */
export function useReport() {
  const filters = ref<ReportFilters>(defaultFilters())
  const result = ref<ReportResult | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const validationErrors = ref<string[]>([])
  let requestId = 0

  const hasData = computed(() => !!result.value && result.value.geoRows.length > 0)

  async function query(): Promise<boolean> {
    const v = validateFilters(filters.value)
    validationErrors.value = v.errors
    if (!v.valid) {
      error.value = v.errors[0]
      return false
    }
    const id = ++requestId
    loading.value = true
    error.value = null
    try {
      const r = await fetchReport(filters.value)
      if (id !== requestId) return false
      result.value = r
      return true
    } catch (e) {
      if (id !== requestId) return false
      result.value = null
      error.value = e instanceof Error ? e.message : '查询失败，请重试'
      return false
    } finally {
      if (id === requestId) loading.value = false
    }
  }

  async function reset(): Promise<boolean> {
    filters.value = defaultFilters()
    validationErrors.value = []
    return query()
  }

  function retry(): Promise<boolean> {
    return query()
  }

  return { filters, result, loading, error, validationErrors, hasData, query, reset, retry }
}
