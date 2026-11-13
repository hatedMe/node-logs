<script setup lang="ts">
import { computed, ref } from 'vue'

// 报告标题工具栏：下载、问题搜索框、帮助说明弹层。
// 注意：按钮文案避开“查询/重置/下载/重试/查看明细”，以免干扰旧测试的按钮查找。

defineProps<{
  canDownload?: boolean
}>()

const emit = defineEmits<{
  (e: 'download'): void
}>()

interface Faq {
  q: string
  a: string
}

const FAQS: Faq[] = [
  {
    q: '地域数据是哪里来的？',
    a: '根据访客 IP 与 IP 地域库推断；移动网络、代理或 VPN 可能导致地域不准确。'
  },
  {
    q: '为什么选择省份后没有数据？',
    a: '所选地域在该时间段内没有访问，或与来源、访客筛选组合后没有交集。'
  },
  {
    q: '对比时间段有什么规则？',
    a: '需要完整选择起止日期，且对比区间不得与当前区间重叠。'
  },
  {
    q: '搜索引擎二级选项何时可用？',
    a: '来源选择“搜索引擎”后，可进一步勾选百度、Google、搜狗、Bing、360。'
  },
  {
    q: '地域筛选在按国家查看时为什么不可用？',
    a: '地域筛选面向国内省份；切换到按省后可选择全国或具体省份。'
  },
  {
    q: '如何下载当前报表？',
    a: '点击工具栏或筛选区的下载按钮，导出当前结果为 CSV 文件（UTF-8 编码）。'
  }
]

const helpVisible = ref(false)
const helpQuery = ref('')

const filteredFaqs = computed(() => {
  const kw = helpQuery.value.trim().toLowerCase()
  if (!kw) return FAQS
  return FAQS.filter((f) => `${f.q} ${f.a}`.toLowerCase().includes(kw))
})

function openHelp(): void {
  helpVisible.value = true
}
</script>

<template>
  <div class="visitor-toolbar">
    <el-button :disabled="!canDownload" data-testid="toolbar-download" @click="emit('download')">
      下载报表
    </el-button>
    <el-input
      v-model="helpQuery"
      class="toolbar-search"
      placeholder="搜索常见问题，回车查看帮助"
      aria-label="问题搜索框"
      clearable
      data-testid="toolbar-search"
      @keyup.enter="openHelp"
    />
    <el-button data-testid="toolbar-help" aria-label="帮助说明" @click="openHelp">帮助</el-button>

    <el-dialog v-model="helpVisible" title="地域分布报告说明" width="560px">
      <p class="help-intro">
        地域分布报告展示不同地域给网站带来的访问量和访客量，用于观察地域分布、判断重点流量地域和区域推广效果。
        先选择日期与对比时间段，再按来源、访客、地域筛选，最后在“按省 / 按国家”之间切换并选择指标。
      </p>
      <el-input
        v-model="helpQuery"
        placeholder="过滤常见问题"
        aria-label="过滤常见问题"
        clearable
        class="help-filter"
      />
      <el-empty v-if="filteredFaqs.length === 0" description="没有匹配的常见问题" />
      <dl v-else class="faq-list">
        <div v-for="f in filteredFaqs" :key="f.q" class="faq-item">
          <dt>{{ f.q }}</dt>
          <dd>{{ f.a }}</dd>
        </div>
      </dl>
      <template #footer>
        <el-button @click="helpVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.visitor-toolbar {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}
.toolbar-search {
  flex: 1;
  min-width: 200px;
  max-width: 420px;
}
.help-intro {
  margin: 0 0 12px;
  font-size: 13px;
  color: var(--el-text-color-regular);
}
.help-filter {
  margin-bottom: 8px;
}
.faq-list {
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.faq-item dt {
  font-weight: 600;
  font-size: 13px;
}
.faq-item dd {
  margin: 2px 0 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
</style>
