// 新模块页的通用装配：筛选 + 查询 + 导出 + 全局上下文联动。
// 各视图只需提供取数函数、CSV 构建与"有内容"判断，模板保持一致交互。

import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import type { ModuleFilters, ModuleQueryContext } from '../types/modules'
import { downloadCsv } from '../utils/export'
import { useAppContext } from './useAppContext'
import { useModuleQuery } from './useModuleQuery'

export interface ModulePageOptions<T> {
  fetchData: (ctx: ModuleQueryContext) => T
  buildCsv: (data: T) => string
  exportFilename: string
  hasContent: (data: T | null) => boolean
}

export function defaultModuleFilters(): ModuleFilters {
  const app = useAppContext()
  return {
    rangePreset: app.rangePreset.value,
    startDate: app.startDate.value,
    endDate: app.endDate.value,
    source: 'all',
    device: 'all',
    keyword: ''
  }
}

export function useModulePage<T>(opts: ModulePageOptions<T>) {
  const app = useAppContext()
  const filters = ref<ModuleFilters>(defaultModuleFilters())
  const { data, loading, error, query, retry } = useModuleQuery<T>(() =>
    opts.fetchData({ ...filters.value, siteId: app.siteId.value })
  )

  const hasContent = computed(() => opts.hasContent(data.value))

  function onQuery(): Promise<boolean> {
    return query()
  }

  function onReset(): Promise<boolean> {
    filters.value = defaultModuleFilters()
    return query()
  }

  function onRetry(): Promise<boolean> {
    return retry()
  }

  function onExport(): void {
    if (!hasContent.value || !data.value) {
      ElMessage.warning('当前无数据可下载')
      return
    }
    downloadCsv(opts.buildCsv(data.value), opts.exportFilename)
    ElMessage.success('已导出当前结果')
  }

  // 顶部站点切换：保留本页筛选，直接重查。
  watch(
    () => app.siteId.value,
    () => {
      void query()
    }
  )
  // 顶部日期切换：同步本页日期后重查（本页筛选是局部覆盖，下次全局切换时再同步）。
  watch(
    () => [app.rangePreset.value, app.startDate.value, app.endDate.value] as const,
    ([preset, start, end]) => {
      filters.value = { ...filters.value, rangePreset: preset, startDate: start, endDate: end }
      void query()
    }
  )

  onMounted(() => {
    void query()
  })

  return { filters, data, loading, error, hasContent, onQuery, onReset, onRetry, onExport }
}
