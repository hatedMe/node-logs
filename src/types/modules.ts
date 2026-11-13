// 各统计模块共用的筛选、卡片、排行、漏斗、热力、表格类型。
// mock 层与视图组件只能通过这些类型协作；不复制后端逻辑。

import type { RangePreset, SourceFilter, TrendPoint } from './report'

export type DeviceFilter = 'all' | 'pc' | 'mobile'

export const DEVICE_OPTIONS: Array<{ value: DeviceFilter; label: string }> = [
  { value: 'all', label: '全部设备' },
  { value: 'pc', label: '计算机' },
  { value: 'mobile', label: '移动设备' }
]

export interface ModuleFilters {
  rangePreset: RangePreset
  startDate: string
  endDate: string
  source: SourceFilter
  device: DeviceFilter
  keyword: string
}

export interface ModuleQueryContext extends ModuleFilters {
  siteId: string
}

export interface StatCard {
  key: string
  label: string
  value: string
  /** 环比展示文本，如 +1.23% / -0.50% / - */
  change: string
  /** null 表示无对比 */
  up: boolean | null
}

export interface RankItem {
  name: string
  value: number
  display: string
  /** 占榜单合计的百分比 0-100 */
  share: number
}

export interface FunnelStep {
  label: string
  value: number
  display: string
  /** 相对第一步的转化率文本 */
  rate: string
}

export interface HeatCell {
  label: string
  value: number
  display: string
  /** 点击强度 0-1，用于颜色深浅 */
  intensity: number
}

export type ColumnFormat = 'int' | 'float2' | 'percent' | 'duration' | 'text'

export interface TableColumn {
  prop: string
  label: string
  sortable?: boolean
  align?: 'left' | 'center' | 'right'
  format?: ColumnFormat
}

export type TableRow = Record<string, string | number>

export interface TableData {
  columns: TableColumn[]
  rows: TableRow[]
}

/** 通用单元格格式化：表格展示与 CSV 导出共用，保证两处一致。 */
export function formatCellValue(format: ColumnFormat | undefined, value: string | number): string {
  if (typeof value === 'string') return value
  if (!Number.isFinite(value)) return '-'
  switch (format) {
    case 'percent':
      return `${(value * 100).toFixed(2)}%`
    case 'float2':
      return value.toFixed(2)
    case 'duration': {
      const total = Math.max(0, Math.round(value))
      return `${Math.floor(total / 60)}:${String(total % 60).padStart(2, '0')}`
    }
    case 'int':
    case 'text':
    default:
      return String(Math.round(value))
  }
}

// ---- 各模块 mock 数据结构 ----

export interface OverviewData {
  cards: StatCard[]
  /** PV/UV 日趋势（复用报表 TrendChart 渲染，compareValues 恒为 null）。 */
  trend: TrendPoint[]
  topSources: RankItem[]
  topPages: RankItem[]
  keywords: TableData
}

export interface PromotionData {
  cards: StatCard[]
  funnel: FunnelStep[]
  plans: TableData
}

export interface TrafficData {
  cards: StatCard[]
  /** PV/UV 日趋势（复用报表 TrendChart 渲染，compareValues 恒为 null）。 */
  trend: TrendPoint[]
  realtime: TableData
  topPages: RankItem[]
}

export interface SourceData {
  cards: StatCard[]
  sourceTypes: RankItem[]
  searchWords: TableData
}

export interface VisitData {
  cards: StatCard[]
  pages: TableData
  entries: RankItem[]
  exits: RankItem[]
}

export interface ConversionData {
  cards: StatCard[]
  funnel: FunnelStep[]
  orders: TableData
}

export interface OptimizeData {
  cards: StatCard[]
  crawlErrors: TableData
  slowPages: RankItem[]
}

export interface HeatmapData {
  pages: string[]
  cells: HeatCell[]
  elements: TableData
}

export interface CustomReportData {
  table: TableData
}

export const CUSTOM_METRICS = ['pv', 'uv', 'conv'] as const
export type CustomMetric = (typeof CUSTOM_METRICS)[number]

export const CUSTOM_METRIC_LABELS: Record<CustomMetric, string> = {
  pv: 'PV',
  uv: 'UV',
  conv: '转化次数'
}

export interface ExportData {
  cards: StatCard[]
  tasks: TableData
  apiKeys: TableData
}

export interface AccountData {
  cards: StatCard[]
  users: TableData
}
