// 报表领域类型：前后端接入边界。
// mock 层与视图组件只能通过这些类型协作；后端接入时请求参数与返回类型以此为准。

export type RangePreset = 'today' | 'yesterday' | 'last7' | 'last30' | 'custom'

export type SourceFilter = 'all' | 'direct' | 'search' | 'external' | 'internal' | 'custom'

export type VisitorType = 'all' | 'new' | 'old'

export type GeoLevel = 'province' | 'country'

export type MetricKey =
  | 'pv'
  | 'uv'
  | 'ip'
  | 'bounceRate'
  | 'avgDuration'
  | 'conversionRate'
  | 'visits'
  | 'avgPages'

/** 最多同时选择的指标数（需求硬约束）。 */
export const MAX_METRICS = 6

/** 首屏默认指标。 */
export const DEFAULT_METRICS: MetricKey[] = ['pv', 'uv']

export const METRIC_ORDER: MetricKey[] = [
  'pv',
  'uv',
  'ip',
  'visits',
  'avgPages',
  'bounceRate',
  'avgDuration',
  'conversionRate'
]

export const METRIC_LABELS: Record<MetricKey, string> = {
  pv: 'PV 浏览量',
  uv: 'UV 访客数',
  ip: 'IP 数',
  bounceRate: '跳出率',
  avgDuration: '平均访问时长',
  conversionRate: '转化率',
  visits: '访问次数',
  avgPages: '平均访问页数'
}

export const RANGE_PRESET_OPTIONS: Array<{ value: RangePreset; label: string }> = [
  { value: 'today', label: '今天' },
  { value: 'yesterday', label: '昨天' },
  { value: 'last7', label: '最近 7 天' },
  { value: 'last30', label: '最近 30 天' },
  { value: 'custom', label: '自定义' }
]

export const SOURCE_OPTIONS: Array<{ value: SourceFilter; label: string }> = [
  { value: 'all', label: '全部来源' },
  { value: 'direct', label: '直接访问' },
  { value: 'search', label: '搜索引擎' },
  { value: 'external', label: '外部链接' },
  { value: 'internal', label: '站内来源' },
  { value: 'custom', label: '自定义来源' }
]

export const VISITOR_OPTIONS: Array<{ value: VisitorType; label: string }> = [
  { value: 'all', label: '全部访客' },
  { value: 'new', label: '新访客' },
  { value: 'old', label: '老访客' }
]

export const GEO_LEVEL_OPTIONS: Array<{ value: GeoLevel; label: string }> = [
  { value: 'province', label: '按省' },
  { value: 'country', label: '按国家' }
]

/** 地域筛选：全国表示全部省份；空数组表示全部地域（默认）。 */
export const NATIONWIDE = '全国'

export const PROVINCE_NAMES = [
  '广东', '江苏', '浙江', '山东', '河南', '四川', '湖北', '湖南', '福建', '安徽',
  '河北', '陕西', '江西', '重庆', '辽宁', '云南', '广西', '山西', '贵州', '新疆'
]

export const COUNTRY_NAMES = [
  '中国', '美国', '日本', '韩国', '德国', '英国', '法国', '新加坡', '澳大利亚', '加拿大', '印度', '巴西'
]

/** 来源为搜索引擎时的二级选项。 */
export const SEARCH_ENGINE_OPTIONS: Array<{ value: string; label: string }> = [
  { value: 'baidu', label: '百度' },
  { value: 'google', label: 'Google' },
  { value: 'sogou', label: '搜狗' },
  { value: 'bing', label: 'Bing' },
  { value: '360', label: '360搜索' }
]

export interface MetricGroup {
  key: string
  label: string
  metrics: MetricKey[]
}

/** 指标选择器分组（与百度统计帮助文档的指标分类一致）。 */
export const METRIC_GROUPS: MetricGroup[] = [
  { key: 'basic', label: '网站基础指标', metrics: ['pv', 'uv', 'ip', 'visits'] },
  { key: 'quality', label: '流量质量指标', metrics: ['bounceRate', 'avgDuration', 'avgPages'] },
  { key: 'conversion', label: '转化指标', metrics: ['conversionRate'] }
]

/** 指标选择器中的指标说明。 */
export const METRIC_DESCRIPTIONS: Record<MetricKey, string> = {
  pv: '页面被浏览的次数，同一访客多次浏览重复计数',
  uv: '去重后的独立访客数',
  ip: '去重后的访问 IP 数',
  visits: '访问会话数量，一次连续访问计为一次',
  bounceRate: '仅浏览一个页面的访问占比',
  avgDuration: '访问会话的平均停留时间',
  avgPages: '每次访问平均浏览的页面数量',
  conversionRate: '转化次数占访问量的比例'
}

export interface ReportFilters {
  rangePreset: RangePreset
  startDate: string
  endDate: string
  compareEnabled: boolean
  compareStartDate: string
  compareEndDate: string
  source: SourceFilter
  visitorType: VisitorType
  geoLevel: GeoLevel
  metrics: MetricKey[]
  /** 地域筛选：空数组=全部地域；含“全国”=全部省份；否则按省名过滤（仅按省生效）。 */
  areas?: string[]
  /** 搜索引擎二级选项：仅 source=search 时生效，空数组=全部引擎。 */
  engines?: string[]
}

export type MetricValues = Record<MetricKey, number>

export function zeroMetricValues(): MetricValues {
  return {
    pv: 0,
    uv: 0,
    ip: 0,
    bounceRate: 0,
    avgDuration: 0,
    conversionRate: 0,
    visits: 0,
    avgPages: 0
  }
}

export interface SummaryItem {
  key: MetricKey
  value: number
  compareValue: number | null
  changeRate: number | null
}

export interface TrendPoint {
  date: string
  values: MetricValues
  compareValues: MetricValues | null
}

export interface GeoRow {
  name: string
  code: string
  values: MetricValues
}

export interface ReportResult {
  summary: SummaryItem[]
  trend: TrendPoint[]
  geoRows: GeoRow[]
  total: number
  queryKey: string
}

/** 指标值展示格式：比率为百分比，时长为分:秒，其余为整数。CSV 与界面共用。 */
export function formatMetricValue(key: MetricKey, value: number): string {
  if (!Number.isFinite(value)) return '-'
  if (key === 'bounceRate' || key === 'conversionRate') return `${(value * 100).toFixed(2)}%`
  if (key === 'avgDuration') {
    const total = Math.max(0, Math.round(value))
    return `${Math.floor(total / 60)}:${String(total % 60).padStart(2, '0')}`
  }
  if (key === 'avgPages') return value.toFixed(2)
  return String(Math.round(value))
}

/** 环比展示格式：+1.23% / -0.50% / -（无对比）。 */
export function formatChangeRate(rate: number | null): string {
  if (rate === null || !Number.isFinite(rate)) return '-'
  const sign = rate > 0 ? '+' : ''
  return `${sign}${(rate * 100).toFixed(2)}%`
}
