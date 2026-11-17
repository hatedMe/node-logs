// 20 个还原报表页（百度统计演示站目录）的统一数据形状。
// 每个报表页的生成器返回 ReportPageData，按需填充可选区块；
// mock 层与页面组件只能通过这些类型协作，字段含义与原版报表一一对应。

import type { Ref } from 'vue'
import type { FunnelStep, HeatCell, ModuleFilters, RankItem, StatCard, TableData } from './modules'
import type { GeoRow, TrendPoint } from './report'

export interface ReportPageData {
  /** 顶部指标卡（原版“核心指标/今日流量”条）。 */
  cards: StatCard[]
  /** 趋势图数据（PV/UV 日趋势，复用 TrendChart 渲染）。 */
  trend: TrendPoint[]
  /** 主数据明细表（原版“数据明细”第一个 Tab）。 */
  table: TableData
  /** 第二个 Tab 的明细表（如：来源网站 / 竞价搜索词 / 按国家）。 */
  secondary?: TableData
  /** 排行/构成（TOP 榜、占比环图）。 */
  rank?: RankItem[]
  /** 漏斗或流转路径步骤（转化漏斗、页面上下游、转化路径）。 */
  funnel?: FunnelStep[]
  /** 页面点击热区（页面点击图）。 */
  heat?: HeatCell[]
  /** 地域数据（地域分布地图）。 */
  geo?: GeoRow[]
  /** 页面/子目录等下拉选项（页面点击图、页面上下游）。 */
  pages?: string[]
}

export interface ReportPageTab {
  key: 'main' | 'secondary'
  label: string
}

/** useModulePage 装配结果的最小契约：ReportShell 与各报表页只依赖这些字段。 */
export interface ReportPageApi {
  filters: Ref<ModuleFilters>
  data: Ref<ReportPageData | null>
  loading: Ref<boolean>
  error: Ref<string | null>
  hasContent: Ref<boolean>
  onQuery: () => Promise<boolean>
  onReset: () => Promise<boolean>
  onRetry: () => Promise<boolean>
  onExport: () => void
}

/** 报表页通用 CSV：导出当前 Tab 的明细表。 */
export function activeTableOf(data: ReportPageData, tab: ReportPageTab['key']): TableData {
  return tab === 'secondary' && data.secondary ? data.secondary : data.table
}
