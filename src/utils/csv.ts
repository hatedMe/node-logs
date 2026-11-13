// 通用 CSV 文本生成：各模块导出共用。只做纯文本转换，浏览器下载走 utils/export 的 downloadCsv。

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
