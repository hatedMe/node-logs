import type { GeoRow, MetricKey } from '../types/report'
import { METRIC_LABELS, METRIC_ORDER, formatMetricValue } from '../types/report'

/** 导出列：地域 + PV + UV + 当前选择指标（去重，按指标顺序排列）。 */
export function exportColumns(metrics: MetricKey[]): MetricKey[] {
  const cols: MetricKey[] = ['pv', 'uv']
  for (const k of METRIC_ORDER) {
    if (metrics.includes(k) && !cols.includes(k)) cols.push(k)
  }
  return cols
}

function escapeCell(s: string): string {
  return /[",\n\r]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s
}

/**
 * 生成带 UTF-8 BOM 的 CSV 文本。纯文本转换，不触发浏览器下载。
 * 只导出当前已筛选结果行。
 */
export function exportRows(rows: GeoRow[], metrics: MetricKey[]): string {
  const cols = exportColumns(metrics)
  const header = ['地域', ...cols.map((k) => METRIC_LABELS[k])]
  const lines = rows.map((r) => [r.name, ...cols.map((k) => formatMetricValue(k, r.values[k]))])
  return `﻿${[header, ...lines].map((cells) => cells.map(escapeCell).join(',')).join('\r\n')}`
}

/** 浏览器下载 CSV 文件（由页面层调用）。 */
export function downloadCsv(csv: string, filename: string): void {
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}
