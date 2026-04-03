# 前端框架细节引用

本文档详细列出从主项目 Admin 复用的前端代码，包括文件路径、关键代码片段和修改建议。

---

## 1. 配置文件

### 1.1 package.json

**来源**: `token-bridge-v2/apps/admin/package.json`

**复制后修改**:
```json
{
  "name": "@tbv2/intelligence-console",
  "private": true,
  "version": "0.1.0",
  "type": "module",
  "scripts": {
    "dev": "vite --port 5174",
    "build": "vite build",
    "preview": "vite preview",
    "lint": "eslint .",
    "typecheck": "tsc --noEmit"
  },
  "dependencies": {
    "@emotion/react": "^11.14.0",
    "@emotion/styled": "^11.14.0",
    "@mui/icons-material": "^5.18.0",
    "@mui/material": "^5.18.0",
    "@tanstack/react-query": "^5.90.2",
    "ra-i18n-polyglot": "^5.14.5",
    "ra-language-english": "^5.14.5",
    "react": "^18.3.1",
    "react-admin": "^5.14.5",
    "react-dom": "^18.3.1",
    "react-error-boundary": "^4.1.2",
    "react-hook-form": "^7.65.0",
    "react-is": "^18.3.1",
    "react-router-dom": "^6.28.0",
    "recharts": "^3.8.0"
  },
  "devDependencies": {
    "@types/node": "20.12.7",
    "@types/react": "^19.2.0",
    "@types/react-dom": "^19.2.0",
    "@typescript-eslint/eslint-plugin": "^8.57.1",
    "@typescript-eslint/parser": "^8.57.1",
    "@vitejs/plugin-react": "^4.7.0",
    "eslint": "^8.57.1",
    "eslint-plugin-react": "^7.37.5",
    "eslint-plugin-react-hooks": "^7.0.1",
    "eslint-plugin-react-refresh": "0.4.20",
    "globals": "^17.4.0",
    "typescript": "5.4.5",
    "vite": "5.2.10"
  }
}
```

**修改说明**:
- 修改 `name` 为爬虫控制台
- 修改 `dev` 端口为 5174（避免与主项目冲突）
- 移除 Playwright 相关依赖（可选）

---

### 1.2 vite.config.ts

**来源**: `token-bridge-v2/apps/admin/vite.config.ts`

**关键修改**:
```typescript
// 修改 API 代理指向爬虫系统
const apiTarget = 'http://localhost:8081'  // 爬虫系统端口

const apiProxy = {
  '/api': {  // 爬虫系统 API 前缀
    target: apiTarget,
    changeOrigin: true,
  },
  '/healthz': {
    target: apiTarget,
    changeOrigin: true,
  },
} as const

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
    },
  },
  server: {
    host: '0.0.0.0',
    port: 5174,  // 修改端口
    strictPort: true,
    proxy: apiProxy,
  },
  preview: {
    host: '0.0.0.0',
    port: 5174,
    proxy: apiProxy,
  },
})
```

---

### 1.3 tsconfig.json

**来源**: `token-bridge-v2/apps/admin/tsconfig.json`

**直接复制，无需修改**

---

## 2. 主题系统 (src/theme/)

### 2.1 AdminThemeContext.tsx

**来源**: `token-bridge-v2/apps/admin/src/theme/AdminThemeContext.tsx`

**复制方式**: 直接复制，无需修改

**用途**: 提供主题切换能力（light/dark）

---

### 2.2 adminThemePresets.ts

**来源**: `token-bridge-v2/apps/admin/src/theme/adminThemePresets.ts`

**复制方式**: 直接复制，无需修改

**关键内容**:
- 主题色定义（primary: #2563EB）
- 字体配置（Inter）
- 阴影样式
- 组件样式覆盖

---

### 2.3 adminRailPresets.ts

**来源**: `token-bridge-v2/apps/admin/src/theme/adminRailPresets.ts`

**复制方式**: 直接复制，无需修改

**用途**: 侧边栏样式预设

---

### 2.4 TbShellLayout.tsx

**来源**: `token-bridge-v2/apps/admin/src/theme/TbShellLayout.tsx`

**复制方式**: 直接复制，无需修改

**用途**: 整体布局组件（侧边栏 + 内容区）

---

### 2.5 CustomAppBar.tsx

**来源**: `token-bridge-v2/apps/admin/src/theme/CustomAppBar.tsx`

**复制方式**: 直接复制，修改标题

**修改建议**:
```typescript
// 修改标题
<Typography variant="h6" sx={{ fontWeight: 700 }}>
  Token Bridge Intelligence
</Typography>
<Typography variant="caption" sx={{ opacity: 0.7 }}>
  爬虫控制台
</Typography>
```

---

## 3. UI 封装 (src/ui/mui/)

### 3.1 components.ts

**来源**: `token-bridge-v2/apps/admin/src/ui/mui/components.ts`

**复制方式**: 直接复制，无需修改

**内容示例**:
```typescript
export {
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  // ... 其他 MUI 组件
} from '@mui/material'
```

---

### 3.2 icons.ts

**来源**: `token-bridge-v2/apps/admin/src/ui/mui/icons.ts`

**复制方式**: 直接复制，按需添加图标

**建议添加的图标**:
```typescript
export {
  // 已有图标...
  
  // 爬虫控制台专用
  MemoryIcon,           // 采集器
  SettingsIcon,         // 配置
  TranslateIcon,        // 翻译
  BugReportIcon,        // 调试
  PlayArrowIcon,        // 运行
  PauseIcon,            // 暂停
  RefreshIcon,          // 刷新
  ScheduleIcon,         // 调度
} from '@mui/icons-material'
```

---

### 3.3 styles.ts

**来源**: `token-bridge-v2/apps/admin/src/ui/mui/styles.ts`

**复制方式**: 直接复制，无需修改

---

## 4. 通用组件 (src/components/common/)

### 4.1 StateNotice.tsx

**来源**: `token-bridge-v2/apps/admin/src/components/common/StateNotice.tsx`

**复制方式**: 直接复制，无需修改

**用途**: 空状态、加载状态、错误状态展示

---

### 4.2 KPICard.tsx

**来源**: `token-bridge-v2/apps/admin/src/components/Dashboard/KPICard.tsx`

**复制方式**: 直接复制，无需修改

**用途**: 监控大盘 KPI 卡片

**使用示例**:
```typescript
<KPICard
  title="今日采集情报"
  value="156"
  changePercent={12.5}
  icon={<MemoryIcon />}
  color="primary"
/>
```

---

## 5. API 基础 (src/api/)

### 5.1 httpError.ts

**来源**: `token-bridge-v2/apps/admin/src/api/httpError.ts`

**复制方式**: 直接复制，修改 API 前缀

**关键修改**:
```typescript
// 修改 API 基础路径
const API_BASE = '/api/v1'

export async function adminFetchJson(
  url: string,
  options?: RequestInit
): Promise<any> {
  const fullUrl = url.startsWith('http') ? url : `${API_BASE}${url}`
  // ... 其余逻辑不变
}
```

---

## 6. 应用入口

### 6.1 main.tsx

**来源**: `token-bridge-v2/apps/admin/src/main.tsx`

**复制方式**: 直接复制，无需修改

---

### 6.2 App.tsx 结构参考

**来源**: `token-bridge-v2/apps/admin/src/App.tsx`

**简化版结构**:
```typescript
import { Admin, Resource } from 'react-admin'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

import { AdminThemeProvider } from './theme/AdminThemeContext'
import { TbShellLayout } from './theme/TbShellLayout'
import { AppMenu } from './AppMenu'
import { dataProvider } from './dataProvider'
import { authProvider } from './authProvider'

// 页面组件
import { Dashboard } from './components/Dashboard'
import { CollectorList } from './components/Collectors'
import { IntelligenceList } from './components/Intelligence'
import { SignalList } from './components/Signals'
import { TranslationStatus } from './components/Translation'
import { Settings } from './components/Settings'

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } }
})

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AdminThemeProvider>
        <Admin
          dataProvider={dataProvider}
          authProvider={authProvider}
          layout={TbShellLayout}
          menu={AppMenu}
          dashboard={Dashboard}
        >
          <Resource name="collectors" list={CollectorList} />
          <Resource name="intelligence" list={IntelligenceList} />
          <Resource name="signals" list={SignalList} />
          <Resource name="translation" list={TranslationStatus} />
          <Resource name="settings" list={Settings} />
        </Admin>
      </AdminThemeProvider>
    </QueryClientProvider>
  )
}
```

---

### 6.3 AppMenu.tsx

**来源**: `token-bridge-v2/apps/admin/src/components/AppMenu.tsx`

**修改版结构**:
```typescript
const MENU_SECTIONS: SectionSpec[] = [
  {
    id: 'dashboard',
    title: '监控大盘',
    icon: DashboardIcon,
    to: '/',
  },
  {
    id: 'collectors',
    title: '采集器管理',
    icon: MemoryIcon,
    to: '/collectors',
  },
  {
    id: 'intelligence',
    title: '情报浏览',
    icon: ArticleIcon,
    to: '/intelligence',
  },
  {
    id: 'signals',
    title: '信号调试',
    icon: BugReportIcon,
    to: '/signals',
  },
  {
    id: 'translation',
    title: '翻译服务',
    icon: TranslateIcon,
    to: '/translation',
  },
  {
    id: 'settings',
    title: '系统配置',
    icon: SettingsIcon,
    to: '/settings',
  },
]
```

---

## 7. 样式复用

### 7.1 adminDatagridSx.ts

**来源**: `token-bridge-v2/apps/admin/src/adminDatagridSx.ts`

**复制方式**: 直接复制，无需修改

**用途**: DataGrid 统一样式

---

### 7.2 常用样式模式

**来自主项目的样式模式**:

```typescript
// 页面头部
<AdminContentHeader 
  title="页面标题" 
  subtitle="页面描述" 
/>

// 卡片容器
<Paper sx={{ 
  px: { xs: 1.5, md: 2.5 }, 
  py: { xs: 1.5, md: 2 },
  borderRadius: 0,
  border: '1px solid',
  borderColor: 'divider',
}}>

// 状态 Chip
<Chip 
  label="正常" 
  color="success" 
  size="small"
/>
```

---

## 8. 文件复制清单

| 文件路径 | 来源 | 修改建议 |
|----------|------|----------|
| `package.json` | 主项目 Admin | 修改 name、port |
| `tsconfig.json` | 主项目 Admin | 无 |
| `vite.config.ts` | 主项目 Admin | 修改 proxy、port |
| `src/theme/*` | 主项目 Admin | 无 |
| `src/ui/mui/*` | 主项目 Admin | 可选添加图标 |
| `src/api/httpError.ts` | 主项目 Admin | 修改 API 前缀 |
| `src/components/common/StateNotice.tsx` | 主项目 Admin | 无 |
| `src/components/Dashboard/KPICard.tsx` | 主项目 Admin | 无 |
| `src/adminDatagridSx.ts` | 主项目 Admin | 无 |
| `src/main.tsx` | 主项目 Admin | 无 |

---

## 9. 需要新建的组件

| 组件 | 说明 | 参考来源 |
|------|------|----------|
| `Dashboard/index.tsx` | 监控大盘 | 参考主项目 Dashboard |
| `Collectors/index.tsx` | 采集器列表 | 新开发 |
| `Intelligence/index.tsx` | 情报列表 | 参考 IntelligenceHandlingList |
| `Signals/index.tsx` | 信号列表 | 新开发 |
| `Translation/index.tsx` | 翻译状态 | 新开发 |
| `Settings/index.tsx` | 系统配置 | 新开发 |
| `AppMenu.tsx` | 菜单 | 修改主项目 AppMenu |
| `dataProvider.ts` | 数据提供者 | 修改主项目 dataProvider |
| `authProvider.ts` | 认证提供者 | 简化版 |

---

**文档维护**: Token Bridge Team
**最后更新**: 2026-04-03
**版本**: v1.0
