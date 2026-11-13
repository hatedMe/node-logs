<script setup lang="ts">
import { computed } from 'vue'
import type { Component } from 'vue'
import type { MenuKey } from './types/navigation'
import { useHashRoute } from './composables/useHashRoute'
import AppLayout from './components/layout/AppLayout.vue'
import OverviewView from './views/OverviewView.vue'
import PromotionView from './views/PromotionView.vue'
import TrafficView from './views/TrafficView.vue'
import SourceView from './views/SourceView.vue'
import VisitView from './views/VisitView.vue'
import ConversionView from './views/ConversionView.vue'
import VisitorView from './views/VisitorView.vue'
import OptimizeView from './views/OptimizeView.vue'
import HeatmapView from './views/HeatmapView.vue'
import CustomView from './views/CustomView.vue'
import ExportView from './views/ExportView.vue'
import AccountView from './views/AccountView.vue'

// 全同步 import：首屏（访客分析 PV/UV 页）无懒加载延迟，保证旧测试的同步断言稳定。
const VIEW_MAP: Record<MenuKey, Component> = {
  overview: OverviewView,
  promotion: PromotionView,
  traffic: TrafficView,
  source: SourceView,
  visit: VisitView,
  conversion: ConversionView,
  visitor: VisitorView,
  optimize: OptimizeView,
  heatmap: HeatmapView,
  custom: CustomView,
  export: ExportView,
  account: AccountView
}

const { route, navigate } = useHashRoute()
const activeView = computed(() => VIEW_MAP[route.value])
</script>

<template>
  <AppLayout :active-key="route" @navigate="navigate">
    <component :is="activeView" :key="route" />
  </AppLayout>
</template>
