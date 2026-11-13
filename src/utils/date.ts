import type { RangePreset } from '../types/report'

const DATE_RE = /^\d{4}-\d{2}-\d{2}$/

/** 本地日期格式化为 YYYY-MM-DD。 */
export function formatDate(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

/** 严格解析 YYYY-MM-DD，非法返回 null。 */
export function parseDate(s: string): Date | null {
  if (!DATE_RE.test(s)) return null
  const [ys, ms, ds] = s.split('-')
  const y = Number(ys)
  const m = Number(ms)
  const day = Number(ds)
  const d = new Date(y, m - 1, day)
  if (d.getFullYear() !== y || d.getMonth() !== m - 1 || d.getDate() !== day) return null
  return d
}

/** 日期字符串加减天数；输入非法时原样返回。 */
export function addDays(dateStr: string, n: number): string {
  const d = parseDate(dateStr)
  if (!d) return dateStr
  d.setDate(d.getDate() + n)
  return formatDate(d)
}

/** 日期预设换算为起止日期（含当日）。 */
export function getPresetRange(
  preset: Exclude<RangePreset, 'custom'>,
  today = new Date()
): { startDate: string; endDate: string } {
  const end = formatDate(today)
  switch (preset) {
    case 'today':
      return { startDate: end, endDate: end }
    case 'yesterday': {
      const y = addDays(end, -1)
      return { startDate: y, endDate: y }
    }
    case 'last7':
      return { startDate: addDays(end, -6), endDate: end }
    case 'last30':
      return { startDate: addDays(end, -29), endDate: end }
  }
}

/** 起止日期均合法且结束不早于开始。 */
export function isValidRange(start: string, end: string): boolean {
  const s = parseDate(start)
  const e = parseDate(end)
  if (!s || !e) return false
  return s.getTime() <= e.getTime()
}

/** 两段区间是否重叠；任一非法视为不重叠。 */
export function rangesOverlap(aStart: string, aEnd: string, bStart: string, bEnd: string): boolean {
  const s1 = parseDate(aStart)
  const e1 = parseDate(aEnd)
  const s2 = parseDate(bStart)
  const e2 = parseDate(bEnd)
  if (!s1 || !e1 || !s2 || !e2) return false
  return s1.getTime() <= e2.getTime() && s2.getTime() <= e1.getTime()
}

/** 展开区间内每一天（含端点）；非法返回空数组，最多 maxDays 天。 */
export function eachDay(start: string, end: string, maxDays = 366): string[] {
  const s = parseDate(start)
  const e = parseDate(end)
  if (!s || !e || s.getTime() > e.getTime()) return []
  const out: string[] = []
  const cur = new Date(s)
  while (cur.getTime() <= e.getTime() && out.length < maxDays) {
    out.push(formatDate(cur))
    cur.setDate(cur.getDate() + 1)
  }
  return out
}
