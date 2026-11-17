<script setup lang="ts">
import { computed } from 'vue'
import type { Component } from 'vue'
import type { ReportPageKey } from './types/navigation'
import { useHashRoute } from './composables/useHashRoute'
import AppLayout from './components/layout/AppLayout.vue'
import OverviewIndexView from './views/reports/OverviewIndexView.vue'
import TrafficLatestView from './views/reports/TrafficLatestView.vue'
import TrafficTimeView from './views/reports/TrafficTimeView.vue'
import SourceAllView from './views/reports/SourceAllView.vue'
import SourceEngineView from './views/reports/SourceEngineView.vue'
import SourceSearchWordView from './views/reports/SourceSearchWordView.vue'
import SourceLinkView from './views/reports/SourceLinkView.vue'
import SourceInnerView from './views/reports/SourceInnerView.vue'
import VisitTopPageView from './views/reports/VisitTopPageView.vue'
import VisitLandingPageView from './views/reports/VisitLandingPageView.vue'
import VisitTopDomainView from './views/reports/VisitTopDomainView.vue'
import VisitPageClickView from './views/reports/VisitPageClickView.vue'
import VisitFlowView from './views/reports/VisitFlowView.vue'
import VisitEventTrackView from './views/reports/VisitEventTrackView.vue'
import VisitSubdirView from './views/reports/VisitSubdirView.vue'
import ConversionOverviewView from './views/reports/ConversionOverviewView.vue'
import ConversionPathView from './views/reports/ConversionPathView.vue'
import VisitorDistrictView from './views/reports/VisitorDistrictView.vue'
import VisitorClientView from './views/reports/VisitorClientView.vue'
import VisitorTypeView from './views/reports/VisitorTypeView.vue'

// 全同步 import：报表页首屏无懒加载延迟，保证测试的同步断言稳定。
const VIEW_MAP: Record<ReportPageKey, Component> = {
  overview: OverviewIndexView,
  'traffic-latest': TrafficLatestView,
  'traffic-time': TrafficTimeView,
  'source-all': SourceAllView,
  'source-engine': SourceEngineView,
  'source-searchword': SourceSearchWordView,
  'source-link': SourceLinkView,
  'source-inner': SourceInnerView,
  'visit-toppage': VisitTopPageView,
  'visit-landingpage': VisitLandingPageView,
  'visit-topdomain': VisitTopDomainView,
  'visit-pageclick': VisitPageClickView,
  'visit-flow': VisitFlowView,
  'visit-event-track': VisitEventTrackView,
  'visit-subdir': VisitSubdirView,
  'conversion-overview': ConversionOverviewView,
  'conversion-path': ConversionPathView,
  'visitor-district': VisitorDistrictView,
  'visitor-client': VisitorClientView,
  'visitor-type': VisitorTypeView
}

const { route, navigate } = useHashRoute()
const activeView = computed(() => VIEW_MAP[route.value])
</script>

<template>
  <AppLayout :active-key="route" @navigate="navigate">
    <component :is="activeView" :key="route" />
  </AppLayout>
</template>
