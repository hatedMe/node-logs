<script setup lang="ts">
import { computed } from 'vue'
import type { NavigationChild } from '../../types/navigation'

const props = defineProps<{ item: NavigationChild }>()

interface PageConfig {
  description: string
  metrics: [string, string][]
  columns: [string, string][]
  values: string[][]
}

const pageConfigs: Record<string, PageConfig> = {
  overview: { description: '网站整体流量概况与核心指标。', metrics: [['浏览量(PV)', '35,103'], ['访客数(UV)', '23,136'], ['IP数', '28,714'], ['跳出率', '42.7%'], ['平均访问时长', '02:16'], ['转化次数', '1,284']], columns: [['name', '指标'], ['visits', '数值'], ['rate', '较上期']], values: [['浏览量(PV)', '35,103', '+8.6%'], ['访客数(UV)', '23,136', '+5.2%'], ['IP数', '28,714', '+7.9%'], ['跳出率', '42.7%', '-2.1%'], ['平均访问时长', '02:16', '+8.4%'], ['转化次数', '1,284', '+12.1%']] },
  'traffic-latest': { description: '实时查看当前正在访问网站的访客。', metrics: [['当前访客', '128'], ['今日浏览量', '8,642'], ['平均停留', '02:36'], ['今日跳出率', '38.4%']], columns: [['name', '页面'], ['visits', '当前访客'], ['rate', '占比']], values: [['首页', '42', '32.8%'], ['产品介绍', '31', '24.2%'], ['帮助中心', '19', '14.8%'], ['其他页面', '36', '28.2%']] },
  'traffic-time': { description: '查看网站访问量随时间的变化趋势。', metrics: [['浏览量', '35,103'], ['访客数', '23,136'], ['新访客', '16,902'], ['平均访问深度', '2.84']], columns: [['name', '日期'], ['visits', '浏览量'], ['rate', '访客数']], values: [['11月12日', '5,821', '3,744'], ['11月11日', '5,416', '3,512'], ['11月10日', '4,980', '3,201'], ['11月09日', '4,762', '3,066']] },
  'source-all': { description: '按来源渠道查看网站访问构成。', metrics: [['总浏览量', '35,103'], ['直接访问', '12,490'], ['搜索引擎', '10,874'], ['外部链接', '6,328']], columns: [['name', '来源渠道'], ['visits', '浏览量'], ['rate', '占比']], values: [['直接访问', '12,490', '35.6%'], ['搜索引擎', '10,874', '31.0%'], ['外部链接', '6,328', '18.0%'], ['其他来源', '5,411', '15.4%']] },
  'source-engine': { description: '查看搜索引擎带来的访客和访问量。', metrics: [['搜索流量', '10,874'], ['搜索访客', '7,216'], ['百度', '6,438'], ['搜狗', '1,204']], columns: [['name', '搜索引擎'], ['visits', '浏览量'], ['rate', '占比']], values: [['百度', '6,438', '59.2%'], ['搜狗', '1,204', '11.1%'], ['必应', '942', '8.7%'], ['其他', '2,290', '21.0%']] },
  'source-searchword': { description: '查看带来访问的搜索关键词。', metrics: [['搜索词数', '2,418'], ['搜索访客', '7,216'], ['第一关键词', '1,284'], ['平均排名', '4.8']], columns: [['name', '搜索词'], ['visits', '访问量'], ['rate', '占比']], values: [['百度统计 demo', '1,284', '17.8%'], ['网站数据分析', '932', '12.9%'], ['流量分析工具', '716', '9.9%'], ['其他搜索词', '4,284', '59.4%']] },
  'source-link': { description: '查看外部网站链接带来的访问。', metrics: [['外链流量', '6,328'], ['外链访客', '4,802'], ['站点数', '186'], ['平均停留', '01:54']], columns: [['name', '外部链接'], ['visits', '访问量'], ['rate', '占比']], values: [['search.baidu.com', '1,846', '29.2%'], ['weibo.com', '1,214', '19.2%'], ['mp.weixin.qq.com', '986', '15.6%'], ['其他站点', '2,282', '36.0%']] },
  'source-inner': { description: '查看站内页面之间的来源关系。', metrics: [['站内来源', '15,204'], ['入口页面', '428'], ['平均深度', '3.12'], ['回访率', '26.4%']], columns: [['name', '来源页面'], ['visits', '导入访问'], ['rate', '占比']], values: [['首页', '5,421', '35.7%'], ['产品介绍', '3,216', '21.2%'], ['文章详情', '2,804', '18.4%'], ['其他页面', '3,763', '24.7%']] },
  'visit-toppage': { description: '查看受访页面的浏览量、访客数和停留时长。', metrics: [['浏览量', '35,103'], ['访客数', '23,136'], ['平均停留', '02:16'], ['退出率', '21.4%']], columns: [['name', '受访页面'], ['visits', '浏览量'], ['rate', '退出率']], values: [['首页', '8,421', '18.2%'], ['产品介绍', '6,318', '22.4%'], ['文章详情', '4,852', '26.1%'], ['其他页面', '15,512', '20.3%']] },
  'visit-landingpage': { description: '查看用户进入网站时访问的第一个页面。', metrics: [['入口次数', '23,136'], ['入口访客', '18,902'], ['平均停留', '01:48'], ['跳出率', '42.7%']], columns: [['name', '入口页面'], ['visits', '入口次数'], ['rate', '跳出率']], values: [['首页', '9,248', '39.2%'], ['活动页', '4,126', '44.5%'], ['产品介绍', '3,804', '36.8%'], ['其他页面', '5,958', '48.1%']] },
  'visit-topdomain': { description: '查看受访页面所属域名的访问表现。', metrics: [['受访域名', '38'], ['浏览量', '35,103'], ['访客数', '23,136'], ['平均停留', '02:16']], columns: [['name', '域名'], ['visits', '浏览量'], ['rate', '占比']], values: [['www.example.com', '28,421', '81.0%'], ['m.example.com', '4,816', '13.7%'], ['blog.example.com', '1,204', '3.4%'], ['其他域名', '662', '1.9%']] },
  'visit-pageclick': { description: '查看页面元素点击分布和热门点击区域。', metrics: [['点击次数', '18,624'], ['点击用户', '9,842'], ['点击页面', '126'], ['平均点击', '1.89']], columns: [['name', '页面元素'], ['visits', '点击次数'], ['rate', '占比']], values: [['立即注册', '4,218', '22.6%'], ['产品详情', '3,806', '20.4%'], ['导航菜单', '2,914', '15.6%'], ['其他元素', '7,686', '41.4%']] },
  'visit-flow': { description: '分析用户在页面之间的前后访问路径。', metrics: [['路径数量', '1,284'], ['路径用户', '12,846'], ['平均深度', '3.42'], ['退出率', '18.6%']], columns: [['name', '访问路径'], ['visits', '用户数'], ['rate', '占比']], values: [['首页 → 产品介绍', '3,816', '29.7%'], ['首页 → 文章详情', '2,904', '22.6%'], ['产品介绍 → 注册页', '2,186', '17.0%'], ['其他路径', '3,940', '30.7%']] },
  'visit-event-track': { description: '查看自定义事件的触发次数和用户数。', metrics: [['事件次数', '8,421'], ['触发用户', '4,218'], ['事件数', '36'], ['转化率', '12.8%']], columns: [['name', '事件名称'], ['visits', '触发次数'], ['rate', '用户数']], values: [['注册按钮点击', '2,184', '1,842'], ['下载按钮点击', '1,628', '1,204'], ['视频播放', '1,216', '932'], ['其他事件', '3,393', '2,184']] },
  'visit-subdir': { description: '查看不同子目录下的访问数据。', metrics: [['子目录数', '24'], ['浏览量', '35,103'], ['访客数', '23,136'], ['占比最高', '42.8%']], columns: [['name', '子目录'], ['visits', '浏览量'], ['rate', '占比']], values: [['/product/', '15,012', '42.8%'], ['/article/', '8,246', '23.5%'], ['/help/', '5,184', '14.8%'], ['/other/', '6,661', '19.0%']] },
  'conversion-overview': { description: '查看网站目标和转化效果概况。', metrics: [['转化次数', '1,284'], ['转化率', '5.55%'], ['目标数', '8'], ['转化收益', '¥86,420']], columns: [['name', '转化目标'], ['visits', '完成次数'], ['rate', '转化率']], values: [['注册', '624', '7.2%'], ['提交咨询', '318', '4.8%'], ['下载资料', '214', '3.6%'], ['其他目标', '128', '2.1%']] },
  'conversion-path': { description: '查看用户从访问到完成转化的路径。', metrics: [['转化用户', '1,284'], ['路径数', '428'], ['平均步骤', '3.6'], ['完成率', '5.55%']], columns: [['name', '转化路径'], ['visits', '用户数'], ['rate', '完成率']], values: [['入口 → 产品 → 注册', '624', '7.2%'], ['入口 → 活动 → 咨询', '318', '4.8%'], ['搜索 → 详情 → 下载', '214', '3.6%'], ['其他路径', '128', '2.1%']] },
  'visitor-district': { description: '查看访客来自不同地域的分布情况。', metrics: [['访客数', '23,136'], ['覆盖省份', '31'], ['最高省份', '广东'], ['省份占比', '18.4%']], columns: [['name', '地域'], ['visits', '访客数'], ['rate', '占比']], values: [['广东', '4,258', '18.4%'], ['北京', '3,216', '13.9%'], ['上海', '2,804', '12.1%'], ['其他地区', '12,858', '55.6%']] },
  'visitor-client': { description: '查看访客使用的操作系统、浏览器和设备。', metrics: [['访客数', '23,136'], ['移动占比', '61.2%'], ['桌面占比', '38.8%'], ['主流系统', 'Android']], columns: [['name', '系统环境'], ['visits', '访客数'], ['rate', '占比']], values: [['Android', '8,412', '36.4%'], ['iOS', '5,746', '24.8%'], ['Windows', '6,284', '27.2%'], ['macOS及其他', '2,694', '11.6%']] },
  'visitor-type': { description: '对比新访客与老访客的访问行为。', metrics: [['访客数', '23,136'], ['新访客', '16,902'], ['老访客', '6,234'], ['回访率', '26.9%']], columns: [['name', '访客类型'], ['visits', '访客数'], ['rate', '占比']], values: [['新访客', '16,902', '73.1%'], ['老访客', '6,234', '26.9%'], ['新访客浏览量', '21,316', '60.7%'], ['老访客浏览量', '13,787', '39.3%']] }
}

const pageConfig = computed(() => pageConfigs[props.item.key] ?? {
  description: `${props.item.label}数据分析与明细。`,
  metrics: [['访问量', '12,846'], ['访客数', '8,392'], ['访问次数', '9,714'], ['占比', '65.3%']],
  columns: [['name', '名称'], ['visits', '访问量'], ['rate', '占比']],
  values: [[props.item.label, '12,846', '65.3%'], ['本周', '9,714', '63.9%'], ['上周', '8,451', '67.8%'], ['其他', '4,092', '68.7%']]
})

const rows = computed(() => pageConfig.value.values.map((value) => Object.fromEntries(pageConfig.value.columns.map(([key], index) => [key, value[index] ?? '—']))))
</script>

<template>
  <section class="navigation-report" data-testid="navigation-report">
    <header class="navigation-report__header">
      <div>
        <span class="navigation-report__eyebrow">百度统计 / DEMO REPORT</span>
        <h1>{{ item.label }}</h1>
        <p>{{ pageConfig.description }}</p>
      </div>
      <div class="navigation-report__tools"><span>⇩ 下载</span><span>📌</span></div>
    </header>

    <div class="navigation-report__filters">
      <span>时间：</span><button class="is-selected">今天</button><button>昨天</button><button>最近7天</button><button>最近30天</button>
      <span class="filter-date">{{ new Date().toISOString().slice(0, 10) }}　⌄</span>
      <span>设备：</span><button class="is-selected">全部</button><button>计算机</button><button>移动设备</button>
      <span>访客：</span><button class="is-selected">全部</button><button>新访客</button><button>老访客</button>
    </div>
    <div class="navigation-report__notice"><strong>您在使用百度统计服务过程中看到搜索词或者关键词，仅能用于站长的SEO或SEM优化，不得用于其他商业用途或者不正当用途。</strong><br>数据存储时长默认为1年，如需延长数据保留年限，可发送您的公司名称、联系人姓名及电话号码与我们联系。</div>
    <nav class="navigation-report__tabs"><button class="is-active">{{ item.parentKey === 'source' ? '来源类型' : item.label }}</button><button>{{ item.parentKey === 'source' ? '来源网站' : '数据明细' }}</button></nav>

    <div class="navigation-report__metrics">
      <article v-for="metric in pageConfig.metrics" :key="metric[0]">
        <span>{{ metric[0] }}</span>
        <strong>{{ metric[1] }}</strong>
        <small>较上期 +8.6%</small>
      </article>
    </div>

    <div v-if="['source-inner', 'visit-pageclick', 'visit-flow', 'visit-subdir', 'conversion-path'].includes(item.key)" class="original-empty-panel">
      <el-card><div class="empty-state"><span class="empty-state__icon">△</span><strong>暂无数据</strong><p>当前页面暂时没有可展示的数据，请调整时间范围或检查统计代码是否正常安装。</p><button>查看帮助</button></div></el-card>
    </div>
    <div v-else-if="item.key === 'overview'" class="overview-panel">
      <div class="overview-columns"><el-card><template #header><span>趋势图</span></template><div class="trend-legend"><i class="pv" />浏览量(PV)<i class="uv" />访客数(UV)</div><div class="line-chart overview-line"><b v-for="(point, index) in [62, 44, 35, 28, 40, 56, 48, 70, 61, 78, 56, 66]" :key="point" :style="{ left: `${index * 9}%`, bottom: `${point}%` }" /></div><div class="report-axis"><span>0</span><span>12</span><span>24</span></div></el-card><el-card><template #header><span>Top10搜索词</span></template><el-table :data="rows" size="small"><el-table-column prop="name" label="搜索词" /><el-table-column prop="visits" label="浏览量(PV)" /><el-table-column prop="rate" label="占比" /></el-table></el-card></div>
      <div class="overview-columns"><el-card><template #header><span>Top10来源网站</span></template><el-table :data="rows" size="small"><el-table-column prop="name" label="来源网站" /><el-table-column prop="visits" label="浏览量(PV)" /><el-table-column prop="rate" label="占比" /></el-table></el-card><el-card><template #header><span>Top10入口页面</span></template><el-table :data="rows" size="small"><el-table-column prop="name" label="入口页面" /><el-table-column prop="visits" label="浏览量(PV)" /><el-table-column prop="rate" label="占比" /></el-table></el-card></div>
    </div>
    <div v-else-if="item.key === 'traffic-latest'" class="navigation-report__special realtime-panel">
      <el-card><template #header><span>实时访客</span></template><div class="realtime-list"><div v-for="row in rows" :key="row.name"><b>{{ row.name }}</b><span>{{ row.visits }} 人正在访问</span><small>刚刚</small></div></div></el-card>
      <el-card><template #header><span>当前访问页面</span></template><div class="live-number">128<small>当前访客</small></div><div class="report-bars compact"><i v-for="height in [34,52,43,78,65,88,72,95]" :key="height" :style="{ height: `${height}%` }" /></div></el-card>
    </div>
    <div v-else-if="item.key === 'conversion-overview'" class="navigation-report__special conversion-panel">
      <el-card><template #header><span>转化路径</span></template><div class="funnel-step" v-for="(row, index) in rows" :key="row.name" :style="{ width: `${100 - index * 16}%` }"><b>{{ row.name }}</b><span>{{ row.visits }} 人</span></div></el-card>
      <el-card><template #header><span>转化明细</span></template><el-table :data="rows" size="small"><el-table-column v-for="column in pageConfig.columns" :key="column[0]" :prop="column[0]" :label="column[1]" /></el-table></el-card>
    </div>
    <div v-else-if="item.key === 'conversion-path'" class="navigation-report__grid conversion-path-panel">
      <el-card><template #header><span>转化路径分析</span></template><div class="flow-track"><div v-for="(row, index) in rows" :key="row.name" class="flow-node"><b>步骤 {{ index + 1 }}</b><span>{{ row.name }}</span><small>完成率 {{ row.rate }}</small></div></div></el-card>
      <el-card><template #header><span>路径转化明细</span></template><el-table :data="rows" size="small"><el-table-column v-for="column in pageConfig.columns" :key="column[0]" :prop="column[0]" :label="column[1]" /></el-table></el-card>
    </div>
    <div v-else-if="item.key === 'visitor-district'" class="navigation-report__special visitor-panel">
      <el-card><template #header><span>{{ item.label }}分布</span></template><div class="map-placeholder">访客分布地图<div class="map-dot" v-for="n in 9" :key="n" :style="{ left: `${12 + n * 8}%`, top: `${24 + (n % 4) * 14}%` }" /></div></el-card>
      <el-card><template #header><span>排行明细</span></template><el-table :data="rows" size="small"><el-table-column v-for="column in pageConfig.columns" :key="column[0]" :prop="column[0]" :label="column[1]" /></el-table></el-card>
    </div>
    <div v-else-if="item.key === 'traffic-time'" class="navigation-report__grid trend-panel">
      <el-card class="navigation-report__chart"><template #header><span>趋势图</span></template><div class="trend-legend"><i class="pv" />浏览量(PV)<i class="uv" />访客数(UV)</div><div class="line-chart"><b v-for="(point, index) in [40, 58, 48, 70, 62, 82, 68, 92]" :key="point" :style="{ left: `${index * 13}%`, bottom: `${point}%` }" /></div><div class="report-axis"><span>11月05日</span><span>11月09日</span><span>11月12日</span></div></el-card>
      <el-card><template #header><span>按日趋势明细</span></template><el-table :data="rows" size="small"><el-table-column v-for="column in pageConfig.columns" :key="column[0]" :prop="column[0]" :label="column[1]" /></el-table></el-card>
    </div>
    <div v-else-if="item.key === 'source-all'" class="navigation-report__grid source-all-panel">
      <el-card><template #header><span>来源构成</span></template><div class="source-donut" /><div class="source-legend"><span v-for="(row, index) in rows" :key="row.name"><i :class="`tone-${index}`" />{{ row.name }}<b>{{ row.rate }}</b></span></div></el-card>
      <el-card><template #header><span>来源趋势</span></template><div class="multi-line-chart"><i v-for="(height, index) in [70,46,38,54,43,62,75,58,86,72]" :key="index" :style="{ height: `${height}%` }" /></div><el-table :data="rows" size="small"><el-table-column prop="name" label="来源类型" /><el-table-column prop="visits" label="浏览量(PV)" /><el-table-column prop="rate" label="占比" /></el-table></el-card>
    </div>
    <div v-else-if="item.key === 'source-engine'" class="engine-panel">
      <el-card><template #header><span>搜索引擎趋势</span></template><div class="engine-summary"><strong>10,874</strong><span>搜索流量</span><strong>7,216</strong><span>搜索访客</span></div><div class="engine-chart"><i v-for="(height, index) in [85,62,48,38,44,52,57,49,66,78,60,72]" :key="index" :style="{ height: `${height}%` }" /></div></el-card>
      <el-card><template #header><span>搜索引擎明细</span></template><el-table :data="rows" size="small"><el-table-column prop="name" label="搜索引擎" /><el-table-column prop="visits" label="浏览量(PV)" /><el-table-column prop="rate" label="占比" /></el-table></el-card>
    </div>
    <div v-else-if="item.key === 'source-searchword'" class="searchword-panel">
      <el-card><template #header><span>搜索词排行</span></template><div class="searchword-toolbar"><button class="is-selected">按浏览量</button><button>按访客数</button><button>按占比</button></div><el-table :data="rows" size="small"><el-table-column type="index" label="排名" width="70" /><el-table-column prop="name" label="搜索词" /><el-table-column prop="visits" label="浏览量(PV)" /><el-table-column prop="rate" label="占比" /></el-table></el-card>
    </div>
    <div v-else-if="item.key === 'source-link'" class="link-panel">
      <el-card><template #header><span>外部链接来源趋势</span></template><div class="link-chart"><i v-for="(height, index) in [40,52,44,66,58,72,83,61,92,78,88]" :key="index" :style="{ height: `${height}%` }" /></div></el-card>
      <el-card><template #header><span>外部链接明细</span></template><el-table :data="rows" size="small"><el-table-column prop="name" label="外部链接" /><el-table-column prop="visits" label="访问量" /><el-table-column prop="rate" label="占比" /></el-table></el-card>
    </div>
    <div v-else-if="item.key === 'visit-pageclick'" class="navigation-report__grid click-panel">
      <el-card><template #header><span>页面点击图</span></template><div class="click-heatmap"><i v-for="n in 36" :key="n" :style="{ opacity: `${0.25 + ((n * 7) % 70) / 100}` }" /></div></el-card>
      <el-card><template #header><span>热门点击元素</span></template><el-table :data="rows" size="small"><el-table-column v-for="column in pageConfig.columns" :key="column[0]" :prop="column[0]" :label="column[1]" /></el-table></el-card>
    </div>
    <div v-else-if="item.key === 'visit-flow'" class="navigation-report__grid flow-panel">
      <el-card><template #header><span>页面上下游</span></template><div class="flow-track"><div v-for="(row, index) in rows" :key="row.name" class="flow-node"><b>{{ index + 1 }}</b><span>{{ row.name }}</span><small>{{ row.visits }} 人</small></div></div></el-card>
      <el-card><template #header><span>路径明细</span></template><el-table :data="rows" size="small"><el-table-column v-for="column in pageConfig.columns" :key="column[0]" :prop="column[0]" :label="column[1]" /></el-table></el-card>
    </div>
    <div v-else-if="item.key === 'visit-event-track'" class="navigation-report__grid event-panel">
      <el-card><template #header><span>事件触发排行</span></template><div class="event-bars"><div v-for="row in rows" :key="row.name"><span>{{ row.name }}</span><i :style="{ width: `${Math.min(94, Number(row.visits.replace(',', '')) / 24)}%` }" /><b>{{ row.visits }}</b></div></div></el-card>
      <el-card><template #header><span>事件明细</span></template><el-table :data="rows" size="small"><el-table-column v-for="column in pageConfig.columns" :key="column[0]" :prop="column[0]" :label="column[1]" /></el-table></el-card>
    </div>
    <div v-else-if="item.key === 'visit-subdir'" class="navigation-report__grid directory-panel">
      <el-card><template #header><span>子目录访问占比</span></template><div class="directory-bars"><div v-for="row in rows" :key="row.name"><span>{{ row.name }}</span><i :style="{ width: row.rate }" /><b>{{ row.rate }}</b></div></div></el-card>
      <el-card><template #header><span>子目录明细</span></template><el-table :data="rows" size="small"><el-table-column v-for="column in pageConfig.columns" :key="column[0]" :prop="column[0]" :label="column[1]" /></el-table></el-card>
    </div>
    <div v-else-if="item.key === 'visitor-client'" class="navigation-report__grid client-panel">
      <el-card><template #header><span>系统环境分布</span></template><div class="client-bars"><div v-for="row in rows" :key="row.name"><span>{{ row.name }}</span><i :style="{ width: row.rate }" /><b>{{ row.rate }}</b></div></div></el-card>
      <el-card><template #header><span>设备明细</span></template><el-table :data="rows" size="small"><el-table-column v-for="column in pageConfig.columns" :key="column[0]" :prop="column[0]" :label="column[1]" /></el-table></el-card>
    </div>
    <div v-else-if="item.key === 'visitor-type'" class="navigation-report__grid visitor-type-panel">
      <el-card><template #header><span>新老访客对比</span></template><div class="visitor-split"><div><strong>73.1%</strong><span>新访客</span></div><div><strong>26.9%</strong><span>老访客</span></div></div></el-card>
      <el-card><template #header><span>访客类型明细</span></template><el-table :data="rows" size="small"><el-table-column v-for="column in pageConfig.columns" :key="column[0]" :prop="column[0]" :label="column[1]" /></el-table></el-card>
    </div>
    <div v-else class="navigation-report__grid">
      <el-card class="navigation-report__chart">
        <template #header><span>指标：{{ pageConfig.metrics[0][0] }}⌄</span></template>
        <div class="navigation-report__donut"><span>访问来源<br><b>100%</b></span></div>
        <div class="report-bars" aria-label="趋势概览图">
          <i v-for="height in [44, 62, 50, 76, 69, 88, 72, 94, 81, 96, 85, 100]" :key="height" :style="{ height: `${height}%` }" />
        </div>
        <div class="report-axis"><span>11/04</span><span>11/08</span><span>11/12</span></div>
      </el-card>

      <el-card>
        <template #header><span>{{ item.label }}明细</span></template>
        <el-table :data="rows" size="small">
          <el-table-column v-for="column in pageConfig.columns" :key="column[0]" :prop="column[0]" :label="column[1]" />
        </el-table>
      </el-card>
    </div>
  </section>
</template>
