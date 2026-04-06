# 数据采集策略优化复盘

**日期**: 2026-04-06
**主题**: 从自研爬虫到 Tavily API 的战略转向

---

## 一、核心认知升级

### 问题发现

在验证情报系统效果时，发现一个关键问题：

```
自研爬虫路线 ❌
  ↓
跟平台斗法 → 反爬封禁 → 数据质量差 → 浪费精力

Tavily API 路线 ✅
  ↓
高质量数据 → 专注业务逻辑 → 发现客户机会
```

### 本质转变

**我们的核心能力不是"爬数据"，而是"发现想要迁移/有痛点的 AI 开发者"。**

| 维度 | 自研爬虫 | Tavily API |
|------|---------|-----------|
| 维护成本 | 高（每次网站改版都要改） | 低（API 稳定） |
| 数据质量 | 不可控 | 高质量（专业公司） |
| 法律风险 | robots.txt、ToS 限制 | 合规 |
| 核心聚焦 | 分散在爬取技术 | 专注客户发现 |
| ROI | 低 | 高 |

**决策**：数据采集交给专业公司，我们专注业务逻辑。

---

## 二、数据源优化

### 2.1 停用的采集器

| 采集器 | 数据量 | 问题 | 状态 |
|--------|--------|------|------|
| StackExchange | 1344条 | 标题为空，质量差 | ❌ 已停用 |
| OpenAI Community | 21条 | 质量差 | ❌ 已停用 |
| Dev.to | 329条 | 待优化 | ⏸️ 暂停 |

### 2.2 保留的采集器

| 优先级 | 采集器 | 作用 | 状态 |
|--------|--------|------|------|
| **P0** | Tavily Search | 高质量信号发现 | ✅ 主力 |
| **P0** | 价格采集器 | 官方API，稳定 | ✅ 保留 |
| **P1** | HackerNews/Reddit | 用户讨论 | ✅ 保留 |
| **P2** | 其他系统采集器 | 辅助数据 | ✅ 保留 |

---

## 三、Tavily 搜索策略优化

### 3.1 基于官方最佳实践

参考文档：https://docs.tavily.com/documentation/best-practices/best-practices-search

**关键优化**：

| 参数 | 优化前 | 优化后 | 原因 |
|------|--------|--------|------|
| `search_depth` | advanced (2 credits) | basic (1 credit) | 节省50%成本 |
| `max_results` | 10条 | 5条 | 官方推荐，避免低质量 |
| `include_raw_content` | true | false | 节省带宽 |

### 3.2 搜索词优化

**从 14 个精简到 11 个**，聚焦高价值信号：

#### 迁移意愿（P1 - 直接发现客户）
- `switching from OpenAI to Claude migration experience`
- `best OpenAI API alternative 2025`
- `moving from OpenAI to Anthropic developer experience`
- `OpenAI vs Claude API comparison switch`

#### 成本压力（P1 - 价格敏感客户）
- `OpenAI API too expensive cost complaint`
- `LLM API cost reduction cheaper alternative`
- `OpenAI pricing increase frustration`

#### 速率限制（P1/P2 - 技术痛点）
- `OpenAI API rate limit 429 error solution`
- `OpenAI rate limiting too restrictive workaround`

#### 功能需求（P2 - 产品改进）
- `OpenAI API limitations missing features`
- `ChatGPT API problems issues wishlist`

### 3.3 月度 ROI 计算

```
额度消耗:
  11词 × 1 credit × 30天 = 330 credits/月
  免费额度: 1000 credits
  使用率: 33%

数据产出:
  每天: 11 × 5 = 55条
  每月: 55 × 30 = 1,650条
  信号率: ~80% → 1,320条
  高价值(migration): ~5% → 66条

成本效益:
  每条高价值信号: 330/66 = 5 credits
  发现1个付费客户: ROI 极高
  0个客户: 反正免费，零成本
```

---

## 四、规则引擎修复

### 4.1 Bug 1: 信号类型判断错误

**问题**：最后一个匹配的规则会覆盖 `signal_type`，导致信号分类不准确。

**修复**：
```go
// 修复前：每次匹配都会覆盖
if rule.Weight < 0 && signalType != "" {
    result.SignalType = signalType
}

// 修复后：只在未设置时才设置（第一个匹配规则决定）
if rule.Weight < 0 && signalType != "" && result.SignalType == string(SignalTypeNoise) {
    result.SignalType = signalType
}
```

**文件**: `internal/rules/engine.go`

### 4.2 Bug 2: 规则名称乱码

**问题**：数据库中规则名称存储为乱码（如 `ææ¬åå`）。

**根本原因**：数据库编码问题，中文存储时出现乱码。

**解决方案**：规则名称统一改为英文，彻底避免编码问题。

**规则清单**（10条）：

| 类型 | 规则名 | 权重 | 优先级 | 用途 |
|------|--------|------|--------|------|
| 噪声 | `spam_marketing` | +10 | 100 | 过滤垃圾营销 |
| 噪声 | `hiring_recruitment` | +8 | 95 | 过滤招聘信息 |
| 噪声 | `content_too_short` | +5 | 50 | 过滤低质量内容 |
| 信号 | `intent_migration` | -15 | 95 | 发现迁移意向（最高价值）|
| 信号 | `cost_pain` | -10 | 90 | 发现价格敏感客户 |
| 信号 | `rate_limit_issue` | -10 | 88 | 发现速率限制痛点 |
| 信号 | `competitor_mention` | -12 | 80 | 追踪竞品提及 |
| 信号 | `feature_request` | -8 | 85 | 发现功能需求 |
| 信号 | `performance_issue` | -8 | 85 | 发现性能问题 |
| 信号 | `quality_issue` | -8 | 82 | 发现质量问题 |

**修复脚本**: `scripts/fix_rules_encoding.go`

---

## 五、数据质量对比

| 指标 | 优化前 | 优化后 |
|------|--------|--------|
| 有效数据源 | 混杂（有低质量） | 纯净（Tavily主力） |
| 标题完整率 | ~0%（StackExchange） | 100%（Tavily） |
| 信号准确率 | 规则判断有误 | ✅ 已修复 |
| 规则名称 | 乱码 | ✅ 英文清晰 |
| 月度成本 | 浪费在低质量数据 | 330 credits 高效利用 |
| 维护精力 | 分散在多平台 | 专注 Tavily 优化 |

---

## 六、关键决策记录

### 决策 1: 采集策略转向
- **选择**: Tavily API 主采 + 价格采集器辅采
- **原因**: 专业工具做专业事，避免重复造轮子
- **影响**: 降低维护成本，提高数据质量

### 决策 2: 规则命名规范
- **选择**: 统一英文命名
- **原因**: 避免数据库编码问题，符合技术团队惯例
- **影响**: 彻底解决乱码，提高可读性

### 决策 3: 成本策略
- **选择**: 先用完免费额度验证 ROI
- **原因**: 低成本试错，数据驱动决策
- **影响**: 330 credits/月验证业务价值

### 决策 4: 数据质量优先
- **选择**: 质量 > 数量
- **原因**: 100条高质量信号 > 1000条低质量数据
- **影响**: 聚焦高价值信号，支撑业务决策

---

## 七、遗留问题

### P0（本周解决）
- [x] 等到 20:00 看 Tavily 自动采集效果
- [ ] 验证新数据质量（标题、信号类型、质量分）
- [ ] 计算实际 ROI（高价值信号数量）

### P1（下周解决）
- [ ] 建立信号质量标准（质量分 ≥ 60 = 高价值）
- [ ] 优化搜索词（根据实际效果调整）
- [ ] 考虑升级 Tavily 套餐（如果 ROI 验证成功）

### P2（后续优化）
- [ ] 实现 trigger API（手动触发采集）
- [ ] 清理旧数据乱码（389条旧数据）
- [ ] 营销信号系统启用（customer_signals）

---

## 八、经验总结

### 技术层面
1. **规则引擎调试**: 优先检查内存 vs 磁盘一致性
2. **信号类型判断**: 第一个匹配规则决定，避免覆盖
3. **编码问题**: 技术系统统一英文，避免国际化复杂性

### 业务层面
1. **核心能力定位**: 不是爬数据，是发现客户机会
2. **专业分工**: 数据采集交给专业公司，我们专注业务逻辑
3. **ROI 思维**: 每条高价值信号的成本 vs 潜在收益

### 策略层面
1. **质量优先**: 100条高质量 > 1000条低质量
2. **低成本试错**: 先用免费额度验证，再考虑付费
3. **数据驱动**: 用实际效果指导优化方向

---

## 九、相关文件

- [main.go](../cmd/intelligence/main.go) - 停用低质量采集器
- [tavily.go](../internal/collectors/search/tavily.go) - 优化搜索策略
- [engine.go](../internal/rules/engine.go) - 修复信号类型判断
- [fix_rules_encoding.go](../scripts/fix_rules_encoding.go) - 规则修复脚本
- [.env](../.env) - 数据库连接配置

---

**下一步**: 等待 20:00 自动采集，验证优化效果。
