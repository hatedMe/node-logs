// 通用 CSV 文本生成：各模块导出共用。只做纯文本转换，浏览器下载走 utils/export 的 downloadCsv。

import type { TableData } from '../types/modules'
import { formatCellValue } from '../types/modules'

function escapeCell(s: string): string {
  return /[",\n\r]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s
}

/** 生成带 UTF-8 BOM 的 CSV 文本，Excel 直接打开不乱码。 */
export function toCsv(header: string[], rows: Array<Array<string | number>>): string {
  const lines = [header, ...rows].map((cells) =>
    cells.map((c) => escapeCell(String(c))).join(',')
  )
  return '\uFEFF' + lines.join('\r\n')
}

/** 报表明细表 → CSV：列头取 label，单元格与界面共用 formatCellValue 格式。 */
export function tableCsv(table: TableData): string {
  return toCsv(
    table.columns.map((c) => c.label),
    table.rows.map((r) => table.columns.map((c) => formatCellValue(c.format, r[c.prop])))
  )
}
