// 本地 mock 数据与查询函数：唯一允许出现模拟数据的地方，视图组件不得直连。
// 数据由筛选条件种子化伪随机生成：同条件返回稳定结果，不同条件返回不同结果。

import type {
  GeoRow,
  MetricKey,
  MetricValues,
  ReportFilters,
  ReportResult,
  SummaryItem,
  TrendPoint
} from '../types/report'
import { COUNTRY_NAMES, MAX_METRICS, NATIONWIDE, PROVINCE_NAMES, SEARCH_ENGINE_OPTIONS } from '../types/report'
import { eachDay } from '../utils/date'

/** 省/国家名单：定义在 types 供筛选组件共用，这里保留别名兼容旧引用。 */
export const PROVINCES = PROVINCE_NAMES
export const COUNTRIES = COUNTRY_NAMES

export const MOCK_ERROR_MESSAGE = '模拟查询失败，请重试'

export type MockMode = 'ok' | 'empty' | 'error'

interface MockScenario {
  mode: MockMode
  delayMs: number
}

const scenario: MockScenario = { mode: 'ok', delayMs: 20 }

/** 测试/演示用：切换 mock 场景（正常/空结果/失败）与延迟。 */
export function setMockScenario(patch: Partial<MockScenario>): void {
  Object.assign(scenario, patch)
}

export function resetMockScenario(): void {
  scenario.mode = 'ok'
  scenario.delayMs = 20
}

function hashString(s: string): number {
  let h = 2166136261
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i)
    h = Math.imul(h, 16777619)
  }
  return h >>> 0
}

function mulberry32(seed: number): () => number {
  let a = seed >>> 0
  return () => {
    a |= 0
    a = (a + 0x6d2b79f5) | 0
    let t = Math.imul(a ^ (a >>> 15), 1 | a)
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

/**
 * 筛选条件归一化为稳定 key：同条件同 key，不同条件不同 key。
 * engines 仅在搜索来源生效、areas 仅在按省生效：未生效的残留值不进 key，
 * 避免切换来源/层级后旧值污染新结果。
 */
export function buildQueryKey(f: ReportFilters): string {
  return JSON.stringify([
    f.rangePreset,
    f.startDate,
    f.endDate,
    f.compareEnabled,
    f.compareStartDate,
    f.compareEndDate,
    f.source,
    f.visitorType,
    f.geoLevel,
    [...f.metrics].sort(),
    f.geoLevel === 'province' ? [...(f.areas ?? [])].sort() : [],
    f.source === 'search' ? [...(f.engines ?? [])].sort() : []
  ])
}

const SOURCE_SCALE: Record<string, number> = {
  all: 1,
  direct: 0.32,
  search: 0.41,
  external: 0.18,
  internal: 0.22,
  custom: 0.12
}

const VISITOR_SCALE: Record<string, number> = { all: 1, new: 0.36, old: 0.64 }

const COUNT_KEYS: MetricKey[] = ['pv', 'uv', 'ip', 'visits']

function genDayValues(rng: () => number, scale: number, weekend: boolean): MetricValues {
  const boost = weekend ? 0.82 : 1
  const pv = Math.round((9000 + rng() * 6000) * scale * boost)
  const uv = Math.round(pv * (0.58 + rng() * 0.18))
  const ip = Math.round(uv * (0.82 + rng() * 0.1))
  const visits = Math.round(pv * (0.68 + rng() * 0.12))
  return {
    pv,
    uv,
    ip,
    visits,
    avgPages: Number((pv / Math.max(1, visits)).toFixed(2)),
    bounceRate: Number((0.3 + rng() * 0.28).toFixed(4)),
    avgDuration: Math.round(70 + rng() * 220),
    conversionRate: Number((0.012 + rng() * 0.06).toFixed(4))
  }
}

/** 地域名单：按省时按 areas 过滤（空/全国=全部省份）；按国家时忽略 areas。 */
function geoNames(filters: ReportFilters): string[] {
  if (filters.geoLevel === 'country') return COUNTRIES
  const areas = filters.areas ?? []
  if (areas.length === 0 || areas.includes(NATIONWIDE)) return PROVINCES
  return PROVINCES.filter((n) => areas.includes(n))
}

/** 搜索引擎二级选项的量级系数：勾选引擎越少流量越小；非搜索来源恒为 1。 */
function engineScale(filters: ReportFilters): number {
  const engines = filters.engines ?? []
  if (filters.source !== 'search' || engines.length === 0) return 1
  return 0.2 + (0.8 * engines.length) / SEARCH_ENGINE_OPTIONS.length
}

function summarize(key: MetricKey, trend: TrendPoint[]): { value: number; compareValue: number | null } {
  if (trend.length === 0) return { value: 0, compareValue: null }
  const hasCompare = trend[0].compareValues !== null
  if (COUNT_KEYS.includes(key)) {
    const value = trend.reduce((s, p) => s + p.values[key], 0)
    const compareValue = hasCompare
      ? trend.reduce((s, p) => s + (p.compareValues as MetricValues)[key], 0)
      : null
    return { value, compareValue }
  }
  const value = trend.reduce((s, p) => s + p.values[key], 0) / trend.length
  const compareValue = hasCompare
    ? trend.reduce((s, p) => s + (p.compareValues as MetricValues)[key], 0) / trend.length
    : null
  return { value, compareValue }
}

/**
 * 模拟异步报表查询。防御性截断指标到 MAX_METRICS 项。
 * - mode=empty：返回空趋势/空地域行/空摘要；
 * - mode=error：reject 固定错误信息。
 */
export function fetchReport(filters: ReportFilters): Promise<ReportResult> {
  const metrics = filters.metrics.slice(0, MAX_METRICS)
  const key = buildQueryKey({ ...filters, metrics })
  return new Promise((resolve, reject) => {
    setTimeout(() => {
      if (scenario.mode === 'error') {
        reject(new Error(MOCK_ERROR_MESSAGE))
        return
      }
      const empty = scenario.mode === 'empty'
      const dates = empty ? [] : eachDay(filters.startDate, filters.endDate)
      const scale = (SOURCE_SCALE[filters.source] ?? 1) * (VISITOR_SCALE[filters.visitorType] ?? 1) * engineScale(filters)

      const trend: TrendPoint[] = dates.map((date) => {
        const rng = mulberry32(hashString(`${key}|${date}`))
        const weekday = new Date(`${date}T00:00:00`).getDay()
        const weekend = weekday === 0 || weekday === 6
        const values = genDayValues(rng, scale, weekend)
        let compareValues: MetricValues | null = null
        if (filters.compareEnabled) {
          compareValues = genDayValues(mulberry32(hashString(`${key}|cmp|${date}`)), scale * 0.9, weekend)
        }
        return { date, values, compareValues }
      })

      const names = geoNames(filters)
      const totalPv = trend.reduce((s, p) => s + p.values.pv, 0)
      const weights = names.map((n) => 0.3 + mulberry32(hashString(`${key}|w|${n}`))())
      const weightSum = weights.reduce((s, w) => s + w, 0)
      const geoRows: GeoRow[] = empty
        ? []
        : names
            .map((name, i) => {
              const rng = mulberry32(hashString(`${key}|geo|${name}`))
              const pv = Math.max(1, Math.round((totalPv * weights[i]) / weightSum))
              const uv = Math.round(pv * (0.58 + rng() * 0.18))
              const ip = Math.round(uv * (0.82 + rng() * 0.1))
              const visits = Math.round(pv * (0.68 + rng() * 0.12))
              const values: MetricValues = {
                pv,
                uv,
                ip,
                visits,
                avgPages: Number((pv / Math.max(1, visits)).toFixed(2)),
                bounceRate: Number((0.3 + rng() * 0.28).toFixed(4)),
                avgDuration: Math.round(70 + rng() * 220),
                conversionRate: Number((0.012 + rng() * 0.06).toFixed(4))
              }
              return {
                name,
                code: `${filters.geoLevel === 'country' ? 'C' : 'P'}${String(i + 1).padStart(2, '0')}`,
                values
              }
            })
            .sort((a, b) => b.values.pv - a.values.pv)

      const summary: SummaryItem[] = empty
        ? []
        : metrics.map((k) => {
            const { value, compareValue } = summarize(k, trend)
            const changeRate =
              compareValue === null || compareValue === 0 ? null : (value - compareValue) / compareValue
            return { key: k, value, compareValue, changeRate }
          })

      resolve({ summary, trend, geoRows, total: geoRows.length, queryKey: key })
    }, scenario.delayMs)
  })
}
