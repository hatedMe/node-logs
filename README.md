# node-logs

一个面向网站和移动端的 PV/UV 统计服务。当前主链路使用 Go 实现：采集请求由 HTTP 服务快速写入 Redis Stream，后台 worker 批量消费并将原始事件与日聚合指标写入 PostgreSQL，报表 API 只读聚合表。

## 当前能力

- 页面浏览量 PV
- 独立访客 UV（应用 + UTC 自然日 + visitor_id 去重）
- 独立 IP（应用 + UTC 自然日 + 哈希后的 IP 去重）
- 会话数与跳出率（默认 30 分钟会话超时）
- 页面排行：PV、UV、会话数
- 设备维度：浏览器、操作系统、设备类型
- 浏览器埋点 SDK，兼容旧参数和旧采集路径
- 用户注册、JWT 登录、应用管理和应用属主校验
- /healthz、/readyz、/metrics 运维接口

来源统计接口目前保留为兼容入口，返回空数组；地域、运营商、留存率等能力尚未接入当前 Go 聚合链路。

## 技术架构

~~~text
浏览器 SDK / 移动端
        │ GET/POST
        ▼
Go server (:8080)
  参数归一化、校验、应用状态缓存、IP HMAC 脱敏
        │ Redis Stream: events:pageviews
        ▼
Go worker（消费组 + 批量处理 + 重试/DLQ）
        │
        ▼
PostgreSQL 16
  raw_events、sessions、daily_* 聚合表
        │
        ▼
报表 API（overview/trend/pages/devices）
~~~

server 和 worker 是两个独立进程，可分别扩容。采集热路径不直接写 PostgreSQL；event_id 用于幂等，重复投递不会重复计入统计。Redis Stream 消费失败会重试，超过次数的消息进入死信 Stream。

### 统计口径

| 指标 | 当前定义 |
| --- | --- |
| PV | 每个唯一 event_id 的页面访问事件计 1 |
| UV | 同一应用、同一 UTC 自然日内，相同 visitor_id 只计 1 |
| 独立 IP | 同一应用、同一 UTC 自然日内，相同 ip_hash 只计 1；明文 IP 不落库 |
| 会话 | 同一访客连续访问，间隔不超过 SESSION_TIMEOUT 归为一个会话 |
| 跳出率 | 单事件会话数 / 总会话数；没有会话时返回 null |

原始事件默认保留 30 天，聚合结果不随原始事件清理任务删除。

## 快速开始

### Docker Compose

需要 Docker 和 Docker Compose。先创建 .env：

~~~sh
cat > .env <<'EOF'
POSTGRES_DB=nodelogs
POSTGRES_USER=app
POSTGRES_PASSWORD=change-me-local-only
HTTP_PORT=8080
JWT_SECRET=change-me-min-32-chars
IP_SALT=change-me-per-deployment
EOF
~~~

POSTGRES_PASSWORD 会被拼入连接串，建议只使用字母和数字；生产环境必须替换 JWT_SECRET 和 IP_SALT，并为每套部署使用独立的 IP_SALT。

启动服务：

~~~sh
docker compose up -d --build
docker compose ps
curl -s http://127.0.0.1:8080/healthz
curl -s http://127.0.0.1:8080/readyz
~~~

数据库表由 server/worker 启动时自动迁移，deploy/postgres/init.sql 只负责首次创建数据卷时设置 PostgreSQL 时区为 UTC。

### 本地运行 Go 服务

需要 Go 1.26、PostgreSQL 16 和 Redis 7。设置 DATABASE_URL、JWT_SECRET、IP_SALT 和 REDIS_ADDR 后，在两个终端分别运行：

~~~sh
go run ./cmd/server
go run ./cmd/worker
~~~

## 浏览器 SDK

构建 SDK：

~~~sh
yarn install
yarn build
yarn test
~~~

Node 17+ 使用 webpack 4 构建时，如果遇到 OpenSSL 错误，执行：

~~~sh
NODE_OPTIONS=--openssl-legacy-provider yarn build
~~~

页面接入示例：

~~~html
<script src="/dist/index.js"></script>
<script>
  siteH5.configure({
    pvid: '800800',
    // 跨域采集时填写完整地址；同源时可省略
    collectUrl: 'https://collect.example.com/api/v1/collect'
  });
  siteH5.pgv();
  siteH5.clickStat('BTN_CLICK');
</script>
~~~

SDK 默认上报 GET /pingd/front/web，也支持 POST /api/v1/collect。每次事件携带唯一 event_id；uvmark 保存在 localStorage，session_id 保存在 sessionStorage。sendBeacon 不可用时会回退到图片 Beacon（fmt=gif）。旧参数 pvid、uvmark、url、arg、v、scr、lg、tz、d 保持兼容，遗留参数 ty、scl、random 继续发送并存入事件 Extra。

## HTTP API

成功响应统一为 { "code": 200, "message": "ok", "data": ... }，错误响应的 code 与 HTTP 状态码一致。

### 采集事件

~~~sh
curl -i 'http://127.0.0.1:8080/pingd/front/web?pvid=800800&uvmark=user-1&url=/home&event_id=demo-1&session_id=session-1&d=1762992000000'
~~~

采集成功返回 204 No Content；添加 fmt=gif 时返回 200 和 1×1 透明 GIF。pvid（应用 ID）必填，应用必须存在且状态为 active。

### 注册、登录和应用

~~~sh
curl -X POST http://127.0.0.1:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"示例站点","category":"web","username":"alice","password":"change-me","domain":"example.com"}'

curl -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"change-me"}'
~~~

登录返回 JWT。创建更多应用时，使用 Authorization: Bearer <token> 调用 POST /api/v1/apps；注册或创建应用的响应会返回 app_id 和 secret_key，前端采集主要使用 app_id 作为 pvid。

### 报表

以下接口都需要 JWT，并要求当前用户是 app_id 的属主。from、to 使用 YYYY-MM-DD，日期按 UTC 自然日解释，默认最大范围为 90 天。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | /api/v1/reports/overview | PV、UV、独立 IP、会话、跳出率总览 |
| GET | /api/v1/reports/trend | 按日趋势 |
| GET | /api/v1/reports/pages | 页面排行；sort=pv\|uv\|sessions，支持 page、page_size |
| GET | /api/v1/reports/devices | 浏览器、系统、设备类型 PV |
| GET | /api/v1/reports/sources | 当前返回空数组，保留兼容性 |

示例：

~~~sh
curl 'http://127.0.0.1:8080/api/v1/reports/overview?app_id=800800&from=2026-11-01&to=2026-11-12' \
  -H 'Authorization: Bearer <JWT>'
~~~

### 运维接口

| 路径 | 用途 |
| --- | --- |
| /healthz | 进程存活检查 |
| /readyz | PostgreSQL 和 Redis 就绪检查 |
| /metrics | Prometheus 文本指标，包括采集状态、队列长度和数据库连接池 |

worker 另在 :9090（METRICS_ADDR 可覆盖，置空禁用）暴露同一组运维接口，
/metrics 还包括消费成功/失败计数、死信计数、消费组 pending 积压与消费延迟直方图。

## 配置项

| 环境变量 | 默认值 | 说明 |
| --- | --- | --- |
| HTTP_ADDR | :8080 | HTTP 监听地址 |
| METRICS_ADDR | :9090 | worker 运维端口监听地址，置空禁用 |
| DATABASE_URL | 必填 | PostgreSQL DSN；也支持 POSTGRES_DSN |
| REDIS_ADDR | 127.0.0.1:6379 | Redis 地址 |
| REDIS_PASSWORD / REDIS_DB | 空 / 0 | Redis 认证和逻辑库 |
| JWT_SECRET | server 必填 | JWT 签名密钥 |
| IP_SALT | server 必填 | IP HMAC 脱敏密钥 |
| STREAM_NAME | events:pageviews | 主 Redis Stream |
| CONSUMER_GROUP | analytics-workers | worker 消费组 |
| DEAD_LETTER_STREAM | events:pageviews:dlq | 死信 Stream |
| BATCH_SIZE / BATCH_TIMEOUT | 500 / 2s | worker 批量参数 |
| SESSION_TIMEOUT | 30m | 会话超时时间 |
| RETENTION_DAYS | 30 | 原始事件保留天数 |
| TRUST_PROXY_HEADERS | false | 是否信任 X-Forwarded-For、X-Real-IP |
| MAX_URL_LENGTH | 2048 | URL 字符数上限 |
| REPORT_MAX_RANGE_DAYS / REPORT_TIMEOUT | 90 / 5s | 报表查询限制 |
| APP_CACHE_TTL | 1m | 采集端应用状态缓存时间 |

## 开发与验证

~~~sh
go test -count=1 ./...
go vet ./...
gofmt -l .
yarn test
~~~

GitHub Actions 还会执行 PostgreSQL 迁移检查、运维脚本 smoke test 和 Docker 镜像构建。

## 运维文档

- [迁移与切流运行手册](docs/migration-runbook.md)
- [Go 迁移计划](docs/superpowers/plans/2026-11-12-golang-migration.md)
- [数据库迁移脚本](migrations/)

Node.js 旧服务目录 server/ 暂保留用于迁移期间的回滚和协议兼容；新部署、开发和报表链路以 Go 服务为准。

