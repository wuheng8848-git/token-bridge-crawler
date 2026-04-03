# 爬虫控制台技术框架设计

## 1. 技术栈选型

完全复用主项目 Admin 的技术栈，保持一致性：

| 技术 | 版本 | 用途 | 来源 |
|------|------|------|------|
| **React** | 18.3.1 | UI 框架 | 主项目 Admin |
| **react-admin** | 5.14.5 | 后台管理框架 | 主项目 Admin |
| **MUI (Material-UI)** | 5.18.0 | 组件库 | 主项目 Admin |
| **@tanstack/react-query** | 5.90.2 | 数据获取 | 主项目 Admin |
| **Vite** | 5.2.10 | 构建工具 | 主项目 Admin |
| **TypeScript** | 5.4.5 | 类型系统 | 主项目 Admin |
| **recharts** | 3.8.0 | 图表库 | 主项目 Admin |

---

## 2. 项目结构

```
console/                           # 爬虫控制台前端项目
├── docs/                          # 文档
│   ├── 01-product-spec.md         # 产品文档
│   └── 02-tech-architecture.md    # 技术框架（本文档）
├── public/                        # 静态资源
├── src/
│   ├── api/                       # API 层
│   │   ├── httpError.ts           # 错误处理（复用主项目）
│   │   ├── collectors.ts          # 采集器 API
│   │   ├── intelligence.ts        # 情报 API
│   │   ├── signals.ts             # 信号 API
│   │   ├── translation.ts         # 翻译 API
│   │   └── dashboard.ts           # 大盘数据 API
│   ├── components/                # 组件
│   │   ├── common/                # 通用组件
│   │   │   └── StateNotice.tsx    # 状态提示（复用主项目）
│   │   ├── Dashboard/             # 监控大盘
│   │   │   ├── index.tsx
│   │   │   ├── KPICard.tsx        # KPI 卡片（复用主项目）
│   │   │   ├── CollectorStatus.tsx
│   │   │   └── TrendChart.tsx
│   │   ├── Collectors/            # 采集器管理
│   │   ├── Intelligence/          # 情报浏览
│   │   ├── Signals/               # 信号调试
│   │   ├── Translation/           # 翻译服务
│   │   └── Settings/              # 系统配置
│   ├── theme/                     # 主题系统（复用主项目）
│   │   ├── AdminThemeContext.tsx
│   │   ├── adminThemePresets.ts
│   │   ├── adminRailPresets.ts
│   │   ├── TbShellLayout.tsx
│   │   └── CustomAppBar.tsx
│   ├── ui/                        # UI 封装（复用主项目）
│   │   └── mui/
│   │       ├── components.ts      # MUI 组件统一导出
│   │       ├── icons.ts           # MUI 图标统一导出
│   │       └── styles.ts          # MUI 样式统一导出
│   ├── App.tsx                    # 应用入口
│   ├── dataProvider.ts            # react-admin 数据提供者
│   ├── authProvider.ts            # 认证提供者（简化版）
│   ├── AppMenu.tsx                # 应用菜单
│   └── main.tsx                   # 渲染入口
├── index.html
├── package.json                   # 依赖配置（复用主项目）
├── tsconfig.json                  # TypeScript 配置（复用主项目）
├── vite.config.ts                 # Vite 配置（修改 proxy）
└── .env.example                   # 环境变量示例
```

---

## 3. API 设计

爬虫系统 Go 后端需要提供的 API：

### 3.1 监控大盘

```typescript
// GET /api/v1/dashboard/overview
interface DashboardOverview {
  collectors: CollectorStatus[]
  intelligenceStats: IntelligenceStats
  signalStats: SignalStats
  translationStats: TranslationStats
}

interface CollectorStatus {
  name: string
  type: string
  status: 'running' | 'degraded' | 'error' | 'stopped'
  lastRunAt: string
  successRate: number
  lastError?: string
}
```

### 3.2 采集器管理

```typescript
// GET /api/v1/collectors
interface CollectorListResponse {
  items: Collector[]
  total: number
}

// POST /api/v1/collectors/:name/trigger  // 手动触发
// PATCH /api/v1/collectors/:name          // 更新配置
// GET /api/v1/collectors/:name/logs       // 获取日志
```

### 3.3 情报浏览

```typescript
// GET /api/v1/intelligence
interface IntelligenceListParams {
  page: number
  perPage: number
  type?: string
  source?: string
  status?: string
  startDate?: string
  endDate?: string
}

// GET /api/v1/intelligence/:id
// POST /api/v1/intelligence/:id/retranslate  // 重新翻译
```

### 3.4 信号调试

```typescript
// GET /api/v1/signals
// PATCH /api/v1/signals/:id/validate         // 标记有效性
// GET /api/v1/signals/stats                  // 统计信息
```

### 3.5 翻译服务

```typescript
// GET /api/v1/translation/status
// GET /api/v1/translation/stats
// POST /api/v1/translation/retry-failed      // 重试失败任务
```

---

## 4. 复用主项目的模块

### 4.1 直接复制的文件

以下文件可直接从主项目 Admin 复制，无需修改：

```
src/theme/
├── AdminThemeContext.tsx      # 主题上下文
├── adminThemePresets.ts       # 主题预设
├── adminRailPresets.ts        # 侧边栏预设
├── TbShellLayout.tsx          # 布局组件
└── CustomAppBar.tsx           # 顶部栏

src/ui/mui/
├── components.ts              # MUI 组件封装
├── icons.ts                   # MUI 图标封装
└── styles.ts                  # MUI 样式封装

src/components/common/
└── StateNotice.tsx            # 状态提示组件

src/api/
└── httpError.ts               # HTTP 错误处理
```

### 4.2 需要修改的文件

```
src/api/*.ts                   # 所有 API 调用改为爬虫系统接口
src/AppMenu.tsx                # 改为爬虫控制台的菜单结构
src/dataProvider.ts            # 资源映射改为爬虫系统资源
src/authProvider.ts            # 简化认证（或对接主项目认证）
vite.config.ts                 # 修改 proxy 指向爬虫 API
```

---

## 5. 路由设计

```typescript
// 路由结构
const routes = [
  { path: '/', element: <Dashboard /> },
  { path: '/collectors', element: <CollectorList /> },
  { path: '/collectors/:name/logs', element: <CollectorLogs /> },
  { path: '/intelligence', element: <IntelligenceList /> },
  { path: '/intelligence/:id', element: <IntelligenceDetail /> },
  { path: '/signals', element: <SignalList /> },
  { path: '/translation', element: <TranslationStatus /> },
  { path: '/settings', element: <Settings /> },
]
```

---

## 6. 与主项目 Admin 的关系

```
┌─────────────────────────────────────────────────────────────┐
│                      用户浏览器                              │
└─────────────────────────────────────────────────────────────┘
                              │
          ┌───────────────────┴───────────────────┐
          ▼                                       ▼
┌──────────────────────┐              ┌──────────────────────┐
│   主项目 Admin       │              │   爬虫控制台         │
│   (5173 端口)        │              │   (5174 端口)        │
│                      │              │                      │
│   - 情报处理         │              │   - 采集监控         │
│   - 业务决策         │              │   - 技术调试         │
│   - 日报分析         │              │   - 配置管理         │
└──────────┬───────────┘              └──────────┬───────────┘
           │                                      │
           │    ┌────────────────────────────┐   │
           └───►│   TB API (8080 端口)       │◄──┘
                │                            │
                │   - 情报相关接口            │
                │   - 用户认证                │
                └────────────┬───────────────┘
                             │
                ┌────────────┴───────────────┐
                ▼                            ▼
        ┌──────────────┐            ┌──────────────┐
        │  爬虫 API    │            │  数据库      │
        │  (新开发)    │            │  PostgreSQL  │
        └──────────────┘            └──────────────┘
```

**部署方案**:
1. **独立部署**: 爬虫控制台单独运行，通过 API 访问爬虫系统
2. **嵌入主项目**: 通过 iframe 或微前端嵌入主项目 Admin

---

## 7. 开发计划

| 阶段 | 任务 | 工作量 |
|------|------|--------|
| **Phase 1** | 基础框架搭建（复制主题、配置） | 1d |
| **Phase 2** | Go API 开发（大盘、采集器） | 2d |
| **Phase 3** | 监控大盘 + 采集器管理页面 | 2d |
| **Phase 4** | 情报浏览 + 信号调试页面 | 2d |
| **Phase 5** | 翻译服务 + 系统配置页面 | 1d |
| **Phase 6** | 联调测试 | 1d |

**总计**: 约 9 个工作日

---

**文档维护**: Token Bridge Team
**最后更新**: 2026-04-03
**版本**: v1.0
