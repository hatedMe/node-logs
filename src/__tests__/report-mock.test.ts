import { afterEach, describe, expect, it } from 'vitest'
import type { ReportFilters } from '../types/report'
import {
  DEFAULT_METRICS,
  MAX_METRICS,
  METRIC_LABELS,
  METRIC_ORDER
} from '../types/report'
import {
  eachDay,
  formatDate,
  getPresetRange,
  isValidRange,
  parseDate,
  rangesOverlap
} from '../utils/date'
import {
  buildQueryKey,
  fetchReport,
  MOCK_ERROR_MESSAGE,
  resetMockScenario,
  setMockScenario
} from '../mocks/report'

function baseFilters(patch: Partial<ReportFilters> = {}): ReportFilters {
  return {
    rangePreset: 'last7',
    startDate: '2026-11-06',
    endDate: '2026-11-12',
    compareEnabled: false,
    compareStartDate: '',
    compareEndDate: '',
    source: 'all',
    visitorType: 'all',
    geoLevel: 'province',
    metrics: [...DEFAULT_METRICS],
    ...patch
  }
}

afterEach(() => {
  resetMockScenario()
})

describe('日期工具', () => {
  it('预设换算：今天/昨天为单天，最近7/30天含当日且天数正确', () => {
    const today = new Date(2026, 10, 13)
    expect(getPresetRange('today', today)).toEqual({ startDate: '2026-11-13', endDate: '2026-11-13' })
    expect(getPresetRange('yesterday', today)).toEqual({
      startDate: '2026-11-12',
      endDate: '2026-11-12'
    })
    const last7 = getPresetRange('last7', today)
    expect(last7.endDate).toBe('2026-11-13')
    expect(eachDay(last7.startDate, last7.endDate)).toHaveLength(7)
    const last30 = getPresetRange('last30', today)
    expect(last30.endDate).toBe('2026-11-13')
    expect(eachDay(last30.startDate, last30.endDate)).toHaveLength(30)
  })

  it('format/parse 往返一致，非法日期返回 null', () => {
    expect(parseDate('2026-02-30')).toBeNull()
    expect(parseDate('2026-13-01')).toBeNull()
    expect(parseDate('not-a-date')).toBeNull()
    const d = parseDate('2026-11-13')
    expect(d).not.toBeNull()
    expect(formatDate(d as Date)).toBe('2026-11-13')
  })

  it('区间校验：结束早于开始为非法', () => {
    expect(isValidRange('2026-11-13', '2026-11-12')).toBe(false)
    expect(isValidRange('2026-11-12', '2026-11-12')).toBe(true)
    expect(isValidRange('bad', '2026-11-12')).toBe(false)
  })

  it('重叠判断：相交为 true，相离/非法为 false', () => {
    expect(rangesOverlap('2026-11-01', '2026-11-07', '2026-11-05', '2026-11-10')).toBe(true)
    expect(rangesOverlap('2026-11-01', '2026-11-07', '2026-11-08', '2026-11-10')).toBe(false)
    expect(rangesOverlap('bad', '2026-11-07', '2026-11-08', '2026-11-10')).toBe(false)
  })

  it('超长区间截断到 366 天', () => {
    expect(eachDay('2025-01-01', '2026-12-31')).toHaveLength(366)
  })
})

describe('指标类型约束', () => {
  it('默认展示 PV、UV，上限为 6 项', () => {
    expect(DEFAULT_METRICS).toEqual(['pv', 'uv'])
    expect(MAX_METRICS).toBe(6)
  })

  it('8 个指标键均有展示标签', () => {
    expect(METRIC_ORDER).toHaveLength(8)
    for (const k of METRIC_ORDER) {
      expect(METRIC_LABELS[k]).toBeTruthy()
    }
  })
})

describe('mock 查询协议', () => {
  it('返回 summary/trend/geoRows/total/queryKey，且趋势天数与区间一致', async () => {
    setMockScenario({ delayMs: 1 })
    const r = await fetchReport(baseFilters())
    expect(r.queryKey).toBe(buildQueryKey(baseFilters()))
    expect(r.trend).toHaveLength(7)
    expect(r.trend[0].date).toBe('2026-11-06')
    expect(r.geoRows.length).toBeGreaterThan(0)
    expect(r.total).toBe(r.geoRows.length)
    expect(r.summary.map((s) => s.key)).toEqual(['pv', 'uv'])
    for (const p of r.trend) {
      expect(p.values.pv).toBeGreaterThan(0)
      expect(p.compareValues).toBeNull()
    }
  })

  it('同条件返回稳定数据，不同条件返回不同数据', async () => {
    setMockScenario({ delayMs: 1 })
    const a = await fetchReport(baseFilters())
    const b = await fetchReport(baseFilters())
    expect(b).toEqual(a)
    const c = await fetchReport(baseFilters({ source: 'search' }))
    expect(c.queryKey).not.toBe(a.queryKey)
    expect(c.trend[0].values.pv).not.toBe(a.trend[0].values.pv)
  })

  it('指标超过 6 项时 mock 截断到 6 项', async () => {
    setMockScenario({ delayMs: 1 })
    const r = await fetchReport(baseFilters({ metrics: [...METRIC_ORDER, 'pv'] }))
    expect(r.summary).toHaveLength(6)
  })

  it('开启对比后趋势点携带对比值', async () => {
    setMockScenario({ delayMs: 1 })
    const r = await fetchReport(
      baseFilters({ compareEnabled: true, compareStartDate: '2026-10-01', compareEndDate: '2026-10-07' })
    )
    expect(r.trend[0].compareValues).not.toBeNull()
    expect(r.summary[0].compareValue).not.toBeNull()
  })

  it('empty 场景返回空趋势/空地域行/空摘要', async () => {
    setMockScenario({ mode: 'empty', delayMs: 1 })
    const r = await fetchReport(baseFilters())
    expect(r.trend).toEqual([])
    expect(r.geoRows).toEqual([])
    expect(r.total).toBe(0)
    expect(r.summary).toEqual([])
  })

  it('error 场景 reject 固定错误信息', async () => {
    setMockScenario({ mode: 'error', delayMs: 1 })
    await expect(fetchReport(baseFilters())).rejects.toThrow(MOCK_ERROR_MESSAGE)
  })
})
