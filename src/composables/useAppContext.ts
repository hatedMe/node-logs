// 顶部全局上下文：站点与日期。单例 ref，跨页面共享。
// 各新模块页挂载时用它初始化筛选；站点/日期变化时自动重查。

import { ref } from 'vue'
import type { RangePreset } from '../types/report'
import { getPresetRange } from '../utils/date'

export interface SiteOption {
  id: string
  name: string
}

export const SITE_OPTIONS: SiteOption[] = [
  { id: '16847648', name: '演示站点A' },
  { id: '5503017', name: '演示站点B' }
]

const initial = getPresetRange('last7')

const siteId = ref(SITE_OPTIONS[0].id)
const rangePreset = ref<RangePreset>('last7')
const startDate = ref(initial.startDate)
const endDate = ref(initial.endDate)

export function useAppContext() {
  function setSite(id: string): void {
    if (SITE_OPTIONS.some((s) => s.id === id)) siteId.value = id
  }

  function setPreset(preset: RangePreset): void {
    rangePreset.value = preset
    if (preset === 'custom') return
    const r = getPresetRange(preset)
    startDate.value = r.startDate
    endDate.value = r.endDate
  }

  function setRange(start: string, end: string): void {
    rangePreset.value = 'custom'
    startDate.value = start
    endDate.value = end
  }

  return { sites: SITE_OPTIONS, siteId, rangePreset, startDate, endDate, setSite, setPreset, setRange }
}

/** 测试用：恢复默认站点与日期。 */
export function resetAppContext(): void {
  const r = getPresetRange('last7')
  siteId.value = SITE_OPTIONS[0].id
  rangePreset.value = 'last7'
  startDate.value = r.startDate
  endDate.value = r.endDate
}
