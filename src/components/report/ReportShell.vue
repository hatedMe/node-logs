// 报表页外壳：标题 + 工具 + 筛选条 + 提示条 + 错误重试 + 内容插槽。
// 20 个还原页面共用；布局与百度统计报表页一致（.navigation-report* 样式）。

<script setup lang="ts">
import type { ReportPageApi } from '../../types/reports'
import ReportFilterStrip from './ReportFilterStrip.vue'

withDefaults(
  defineProps<{
    page: ReportPageApi
    title: string
    desc: string
    testId: string
    showSource?: boolean
    showKeyword?: boolean
    keywordPlaceholder?: string
  }>(),
  { showSource: true, showKeyword: true, keywordPlaceholder: '关键词过滤' }
)

/** 搜索词提示条：原版报表页固定展示（仅限 SEO/SEM 用途 + 数据存储时长说明）。 */
</script>

<template>
  <section class="navigation-report" :data-testid="testId">
    <header class="navigation-report__header">
      <div>
        <span class="navigation-report__eyebrow">百度统计 / DEMO REPORT</span>
        <h1>{{ title }}</h1>
        <p>{{ desc }}</p>
      </div>
      <div class="navigation-report__tools">
        <button type="button" class="tool-btn" :disabled="!page.hasContent.value" @click="page.onExport()">⇩ 下载</button>
        <span aria-hidden="true">📌</span>
      </div>
    </header>

    <ReportFilterStrip
      v-model:filters="page.filters.value"
      :loading="page.loading.value"
      :can-export="page.hasContent.value"
      :show-source="showSource"
      :show-keyword="showKeyword"
      :keyword-placeholder="keywordPlaceholder"
      @submit="page.onQuery"
      @reset="page.onReset"
      @export="page.onExport"
    />

    <div class="navigation-report__notice">
      <strong>您在使用百度统计服务过程中看到搜索词或者关键词，仅能用于站长的SEO或SEM优化，不得用于其他商业用途或者不正当用途。</strong><br />
      数据存储时长默认为1年，如需延长数据保留年限，可发送您的公司名称、联系人姓名及电话号码与我们联系。
    </div>

    <el-alert
      v-if="page.error.value"
      class="report-error"
      type="error"
      :title="page.error.value"
      show-icon
      :closable="false"
    >
      <el-button size="small" data-testid="retry-btn" @click="page.onRetry()">重试</el-button>
    </el-alert>

    <slot />
  </section>
</template>

<style scoped>
.tool-btn {
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  font-size: inherit;
  cursor: pointer;
}
.tool-btn:disabled {
  color: #a8b0bf;
  cursor: not-allowed;
}
.report-error {
  margin: 10px 0;
}
</style>
