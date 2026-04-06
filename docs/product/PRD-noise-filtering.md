# PRD-噪声清洗优化

> 提升信号质量，过滤无效噪声，让用户看到真正有价值的客户信号

---

## 一、需求背景

### 1.1 问题描述

**核心定义**：

> **有价值的信号** = 能帮助用户发现客户机会的情报
> **噪声** = 不能帮助发现客户的情报

**信号与噪声的本质区别**：

| 类型 | 定义 | 判断标准 | 示例 |
|------|------|----------|------|
| **有价值信号** | 能帮助用户发现客户机会 | 能让用户：发现潜在客户 / 做产品决策 / 找到引流机会 | "OpenAI rate limit 太烦了" |
| **噪声** | 不能帮助发现客户 | 以上都不能 | "Check out my AI tool!" |

**有价值信号的分类**：

| 信号类型 | 定义 | 示例 | 用户能做什么 |
|----------|------|------|--------------|
| 用户痛点 | 对现有 AI API 服务的不满 | "OpenAI rate limit 太烦了" | 发现产品改进机会 |
| 迁移意愿 | 想换服务商的讨论 | "Looking for cheaper alternative to OpenAI" | 精准获客 |
| 功能需求 | 希望有什么功能 | "Wish there was a cheaper GPT-4 API" | 产品迭代 |
| 竞品动态 | 竞品价格/功能变化 | "Claude just lowered their API pricing" | 定价决策 |
| 成本压力 | 对价格的抱怨 | "My OpenAI bill is $500/month, too expensive" | 推优惠、获客 |

**噪声的分类**：

| 噪声类型 | 定义 | 示例 | 为什么是噪声 |
|----------|------|------|--------------|
| 营销推广 | 推销产品/服务 | "Check out my new AI tool!" | 不是潜在客户 |
| 技术闲聊 | 与 AI API 无关 | "Python vs Go, which is better?" | 无法据此行动 |
| 碎片信息 | 无上下文 | "API down"（无来源、无详情） | 不知道如何处理 |
| 非目标用户 | 不是美国开发者 | "Enterprise procurement question" | 不是目标客户 |
| 纯情绪 | 无实质内容 | "AI is amazing!" | 无决策价值 |

**判断流程**：

```
收到一条情报
    │
    ▼
能否让用户发现潜在客户？ ─── 是 ──► 信号 ✓
    │
    否
    │
    ▼
能否让用户做产品/定价决策？ ─── 是 ──► 信号 ✓
    │
    否
    │
    ▼
能否让用户找到回帖引流机会？ ─── 是 ──► 信号 ✓
    │
    否
    │
    ▼
噪声 ✗
```

**当前问题**：采集器采集了大量情报，但未区分信号/噪声，用户不知道哪些能帮助发现客户。

### 1.2 目标用户

| 用户角色 | 使用场景 | 核心诉求 |
|----------|----------|----------|
| 运营决策者 | 每天查看 Admin 后台 | 快速找到有价值的客户信号，不想被噪声干扰 |
| 营销执行者 | 寻找目标客户回帖 | 只看能引流的高质量信号 |

### 1.3 预期收益

- 有效信号率从「未知」提升到 > 30%
- 用户决策效率提升：从「看 100 条找 1 条」到「看 3 条找 1 条」
- 直接支撑北极星指标：有效客户发现数

### 1.4 情报处理流程

#### 完整处理逻辑树

```
情报进入
    │
    ▼
┌─────────────────────────────────────┐
│ 第一步：噪声判断                      │
│                                     │
│ 是否为噪声？                          │
│ └─ 包含营销关键词？                  │
│ └─ 内容长度 < 20 字符？              │
│ └─ 与 AI API 完全无关？              │
└─────────────────────────────────────┘
    │
    ├── 是噪声 ──────────────────────► 直接丢弃，不入库
    │                                  （记录统计：今日过滤噪声 X 条）
    │
    ▼ 否，可能是信号
┌─────────────────────────────────────┐
│ 第二步：质量评分                      │
│                                     │
│ 质量分 = 关键词(30%) + 影响力(20%)    │
│        + 完整度(20%) + 时效性(15%)    │
│        + 相关性(15%)                 │
└─────────────────────────────────────┘
    │
    ├── 0-20 分 ──────────────────────► 低质量处理
    │                                  └─ 不入库（或入库但标记 hidden）
    │                                  └─ 不做后续处理
    │
    ├── 20-40 分 ──────────────────────► 中低质量处理
    │                                  └─ 入库，标记 is_low_quality=true
    │                                  └─ 默认不在主列表展示
    │                                  └─ 不获取用户画像（节省资源）
    │                                  └─ 用户可手动切换查看
    │
    ├── 40-70 分 ──────────────────────► 中等质量处理
    │                                  └─ 入库，正常展示
    │                                  └─ 不主动获取用户画像
    │                                  └─ 用户点击时按需获取
    │                                  └─ 标记 customer_tier = C
    │
    ▼ 70-100 分（高质量）
┌─────────────────────────────────────┐
│ 第三步：用户画像获取                  │
│                                     │
│ 获取发帖人信息：                      │
│ └─ Bio、技术栈                       │
│ └─ Karma、历史发帖                   │
│ └─ GitHub、项目信息                  │
└─────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────┐
│ 第四步：客户价值分级                  │
│                                     │
│ 是开发者吗？                          │
│ 有实际项目吗？                        │
│ 有付费能力吗？                        │
│ 有多个痛点吗？                        │
└─────────────────────────────────────┘
    │
    ├── 符合 4 项 ──────► S 级客户
    │                      └─ 入库，高亮标记
    │                      └─ 优先展示，置顶
    │                      └─ 自动推送通知（可选）
    │                      └─ 建议：立即回帖引流
    │
    ├── 符合 3 项 ──────► A 级客户
    │                      └─ 入库，重点标记
    │                      └─ 优先展示
    │                      └─ 建议：重点关注
    │
    ├── 符合 2 项 ──────► B 级客户
    │                      └─ 入库，正常展示
    │                      └─ 建议：一般关注
    │
    └── 符合 0-1 项 ────► C 级客户
                           └─ 入库，低优先级展示
                           └─ 建议：暂不跟进
```

#### 各等级处理汇总

| 阶段 | 分数/等级 | 处理方式 | 是否获取画像 | 展示优先级 |
|------|-----------|----------|--------------|------------|
| 噪声 | - | 丢弃 | 否 | 不展示 |
| 低质量 | 0-20 | 不入库 | 否 | 不展示 |
| 中低质量 | 20-40 | 入库+标记 | 否 | 默认隐藏 |
| 中等质量 | 40-70 | 入库+按需画像 | 用户点击时 | 正常 |
| 高质量+S级 | 70+ + 4项符合 | 入库+高亮 | 是 | 置顶 |
| 高质量+A级 | 70+ + 3项符合 | 入库+重点 | 是 | 优先 |
| 高质量+B级 | 70+ + 2项符合 | 入库 | 是 | 正常 |
| 高质量+C级 | 70+ + 0-1项符合 | 入库 | 是 | 低优先级 |

#### 资源优化策略

**为什么不每条都获取用户画像？**

1. API 调用有成本（配额限制、延迟）
2. 低质量情报不值得投入资源
3. 按需获取 = 节省资源 = 更快响应

**策略**：
- 70 分以上：自动获取画像（高价值，值得投入）
- 40-70 分：用户点击时才获取（按需）
- 40 分以下：不获取（不值得）

#### 处理流程示例

**示例 1：S 级客户处理**

```
1. 采集到一条帖子：
   "My OpenAI bill is $500/month, too expensive"
   发帖人: john_dev

2. 噪声判断：无营销关键词，内容完整，与 AI API 相关 → 不是噪声

3. 质量评分：
   - 命中高价值关键词 "expensive", "bill" → +30 分
   - 内容完整 > 50 字符 → +20 分
   - 包含具体数字 "$500" → +15 分
   - 总分 85 分 → 高质量

4. 用户画像获取：
   - Bio: "Indie hacker, AI tools"
   - Karma: 3000+
   - 历史发帖: 也抱怨过 rate limit

5. 客户价值分级：
   - 是开发者 ✓
   - 有实际项目 ✓
   - 有付费能力（$500/月）✓
   - 有多个痛点 ✓
   → S 级客户

6. 入库展示：
   - 高亮标记，置顶
   - 建议：立即回帖引流
```

**示例 2：噪声处理**

```
1. 采集到一条帖子：
   "Check out my new AI tool! Free trial at example.com"

2. 噪声判断：
   - 包含营销关键词 "check out", "free trial" → 噪声 ✓

3. 处理：直接丢弃，不入库
   - 记录统计：今日过滤噪声 +1
```

**示例 3：中低质量处理**

```
1. 采集到一条帖子：
   "API down"

2. 噪声判断：无营销关键词，与 AI API 相关 → 不是噪声

3. 质量评分：
   - 内容长度 < 20 字符 → -10 分
   - 无具体上下文 → -10 分
   - 总分 15 分 → 低质量

4. 处理：不入库
```

---

## 二、功能需求

### 2.1 功能清单

| ID | 功能点 | 优先级 | 状态 |
|----|--------|--------|------|
| F001 | 噪声识别规则引擎 | P0 | 待开发 |
| F002 | 信号质量评分 | P0 | 待开发 |
| F003 | 低质量信号过滤 | P0 | 待开发 |
| F004 | 用户画像增强 | P0 | 待开发 |
| F005 | Tavily AI 搜索采集 | P1 | 待开发 |
| F006 | 噪声反馈机制 | P1 | 待开发 |
| F007 | 清洗效果统计 | P1 | 待开发 |

### 2.2 详细需求

#### F001: 噪声识别规则引擎

**用户故事**：
```
作为 运营决策者
我希望 系统能自动识别噪声
以便 我只看到有价值的信号
```

**业务规则**：

| 规则类型 | 规则描述 | 示例 |
|----------|----------|------|
| 关键词过滤 | 包含营销/推广关键词 | "free trial", "check out", "promo code" |
| 长度过滤 | 内容过短无上下文 | < 20 字符 |
| 重复过滤 | 相似内容去重 | 同一用户多次发帖 |
| 相关性过滤 | 与 AI API 无关 | 不包含任何关键词 |
| 用户类型过滤 | 非目标用户 | 企业采购、HR 招聘 |

**界面要素**：
- 输入：原始情报内容
- 输出：噪声标记 + 过滤原因
- 交互：后台自动运行，无需用户干预

**验收标准**：
- [ ] 给定包含 "free trial" 的情报，当系统处理时，则标记为噪声
- [ ] 给定内容 < 20 字符的情报，当系统处理时，则标记为低质量
- [ ] 给定不包含任何关键词的情报，当系统处理时，则标记为不相关

#### F002: 信号质量评分

**用户故事**：
```
作为 运营决策者
我希望 看到每条信号的质量评分
以便 快速判断是否值得关注
```

**业务规则**：

| 维度 | 权重 | 评分标准 |
|------|------|----------|
| 关键词命中 | 30% | 命中核心关键词数量 |
| 用户影响力 | 20% | 发帖用户 karma/点赞数 |
| 内容完整度 | 20% | 是否有完整上下文 |
| 时效性 | 15% | 发布时间距今 |
| 相关性 | 15% | 与 AI API 的相关程度 |

**评分公式**：
```
质量分 = 关键词分 × 0.3 + 影响力分 × 0.2 + 完整度分 × 0.2 + 时效性分 × 0.15 + 相关性分 × 0.15
```

**界面要素**：
- 输入：情报内容 + 元数据
- 输出：0-100 质量评分
- 展示：在 Admin 后台情报列表显示评分标签

**验收标准**：
- [ ] 给定高质量信号（含关键词、高影响力用户、完整上下文），当评分时，则分数 > 70
- [ ] 给定低质量信号（无关键词、低影响力用户），当评分时，则分数 < 30

#### F003: 低质量信号过滤

**用户故事**：
```
作为 运营决策者
我希望 低质量信号自动过滤
以便 我不用手动筛选
```

**业务规则**：

| 过滤级别 | 质量分阈值 | 行为 |
|----------|------------|------|
| 噪声 | < 20 | 不入库，直接丢弃 |
| 低质量 | 20-40 | 入库但标记，默认不展示 |
| 中等 | 40-70 | 入库展示，低优先级 |
| 高质量 | > 70 | 入库展示，高优先级 |

**界面要素**：
- 输入：质量评分
- 输出：过滤决策
- 交互：Admin 后台可切换查看级别

**验收标准**：
- [ ] 给定质量分 < 20 的情报，当处理时，则不入库
- [ ] 给定质量分 20-40 的情报，当处理时，则入库但标记为低质量

#### F004: 用户画像增强

**用户故事**：
```
作为 运营决策者
我希望 看到发帖人的更多信息
以便 更准确判断他是否是潜在客户
```

**业务逻辑**：

```
发现一条有价值的帖子
    │
    ▼
获取发帖人信息
    │
    ├─► 用户画像（Bio、技术栈）
    ├─► 历史发帖（项目、痛点、需求）
    ├─► 社区声誉（Karma、点赞数）
    └─► 项目链接（GitHub、网站）
    │
    ▼
综合判断客户价值
    │
    ▼
决定行动优先级
```

**可获取的用户信息**：

| 信息类型 | 来源 | 用途 |
|----------|------|------|
| 用户画像 | 个人主页/Bio | 判断是否开发者、技术栈 |
| 历史发帖 | 帖子列表 | 了解项目、痛点、需求 |
| 社区声誉 | Karma/点赞数 | 判断影响力 |
| 评论历史 | 评论列表 | 发现更多观点 |
| 项目链接 | GitHub/网站 | 判断项目规模、真实性 |

**客户价值判断**：

| 判断维度 | 问题 | 高价值特征 |
|----------|------|------------|
| 身份确认 | 是开发者吗？ | Bio 含 "developer"、"engineer"、"indie hacker" |
| 实际需求 | 有实际项目吗？ | GitHub 有活跃项目、帖子讨论具体技术 |
| 付费能力 | 有预算吗？ | 提到账单金额、有收入来源 |
| 痛点数量 | 有多个痛点？ | 历史发帖有多个抱怨 |
| 影响力 | 有传播力？ | Karma 高、帖子互动多 |

**信号价值等级**：

| 等级 | 特征 | 示例 | 行动优先级 |
|------|------|------|------------|
| S 级 | 明确需求 + 预算 + 在找替代方案 | "My startup spends $2K/month on OpenAI, looking to cut costs" + GitHub 有活跃项目 | 立即回帖引流 |
| A 级 | 有痛点 + 有需求 + 可能转化 | "OpenAI rate limits are killing my app" + 是开发者 | 重点关注 |
| B 级 | 有兴趣 + 可能转化 | "Anyone tried Claude API?" | 一般关注 |
| C 级 | 只是讨论 | "OpenAI vs Claude, what do you think?" | 低优先级 |

**界面要素**：
- 输入：原始情报 + 发帖人 ID
- 输出：用户画像 + 历史发帖摘要 + 客户价值等级
- 展示：在 Admin 后台情报详情页显示用户画像卡片

**验收标准**：
- [ ] 给定一条有价值情报，当获取发帖人信息时，则返回用户画像
- [ ] 给定用户画像，当判断客户价值时，则输出 S/A/B/C 等级
- [ ] 给定 S 级客户情报，当在 Admin 展示时，则高亮标记

#### F005: Tavily AI 搜索采集

**用户故事**：
```
作为 运营决策者
我希望 系统能主动搜索更多相关情报
以便 发现现有采集器覆盖不到的客户信号
```

**业务背景**：

现有采集器局限：
- HackerNews：只能搜 HN
- Reddit：只能搜 Reddit
- Discord：只能获取已加入频道

引入 AI 搜索 API 可以：
- 全网搜索，不局限于单一平台
- 支持自然语言搜索，理解意图
- 发现更多相关讨论

**技术选型**：

| 服务 | 提供商 | 特点 | 价格 |
|------|--------|------|------|
| **Tavily**（推荐） | 海外 | 专为 AI Agent 设计，返回结构化结果 | ~$0.01/次 |
| 博查 | 国内 | 国内最大 AI 搜索 API，日均 3000 万调用 | ~¥0.01/次 |
| 百炼联网搜索 | 阿里云 | 大模型内置搜索 | 按 Token 计费 |
| 火山联网搜索 | 字节 | 大模型内置搜索 | 按次计费 |

**推荐 Tavily 的原因**：
1. 已有 API Key（项目配置中已有）
2. 专为 AI 设计，返回结构化数据
3. 支持自然语言搜索
4. 支持限定时间范围、域名

**搜索策略**：

| 搜索词 | 目标 | 预期结果 |
|--------|------|----------|
| "OpenAI API pricing expensive complaints" | 成本痛点 | 抱怨价格的用户讨论 |
| "LLM API cost too high alternative" | 迁移意愿 | 寻找替代方案的用户 |
| "ChatGPT API billing issues" | 账单问题 | 遇到计费问题的用户 |
| "rate limit OpenAI frustration" | 限流痛点 | 遇到限流问题的用户 |
| "best cheaper alternative to OpenAI API" | 竞品对比 | 对比讨论中的潜在客户 |

**采集流程**：

```
定时任务（每日）
    │
    ├─ 调用 Tavily 搜索 API
    │  ├─ 搜索词 1: "OpenAI API expensive complaints"
    │  ├─ 搜索词 2: "LLM API cost alternative"
    │  └─ ...
    │
    ├─ 获取搜索结果
    │  ├─ URL、标题、摘要
    │  └─ 相关度评分
    │
    ├─ 去重（与现有情报库比对）
    │
    └─ 送入处理器
       ├─ 噪声过滤
       ├─ 质量评分
       └─ 用户画像获取
```

**API 调用示例**：

```go
// Tavily 搜索请求
type TavilySearchRequest struct {
    Query         string   `json:"query"`
    SearchDepth   string   `json:"search_depth"`   // "basic" 或 "advanced"
    MaxResults    int      `json:"max_results"`    // 返回结果数
    IncludeRawContent bool  `json:"include_raw_content"`
    Days          int      `json:"days"`           // 时间范围（天）
}

// 调用示例
request := TavilySearchRequest{
    Query:         "OpenAI API expensive complaints Reddit",
    SearchDepth:   "advanced",
    MaxResults:    10,
    IncludeRawContent: true,
    Days:          30,  // 最近 30 天
}
```

**界面要素**：
- 输入：配置的搜索词列表
- 输出：搜索结果 → 处理器 → 情报入库
- 配置：可在 Admin 后台管理搜索词

**验收标准**：
- [ ] 给定 Tavily API Key，当调用搜索时，则返回搜索结果
- [ ] 给定搜索结果，当去重处理后，则与现有情报不重复
- [ ] 给定搜索配置，当在 Admin 后台修改时，则下次采集生效

#### F006: 噪声反馈机制

**用户故事**：
```
作为 运营决策者
我希望 能标记误判的信号
以便 系统逐步学习优化
```

**业务规则**：
- 用户可在 Admin 后台标记「这是噪声」或「这是有价值信号」
- 系统记录反馈，用于优化规则
- 反馈数据存储在 `signal_feedback` 表

**验收标准**：
- [ ] 给定用户点击「这是噪声」，当反馈时，则记录到反馈表

#### F007: 清洗效果统计

**用户故事**：
```
作为 运营决策者
我希望 看到清洗效果统计
以便 了解系统工作情况
```

**业务规则**：

| 指标 | 定义 | 目标 |
|------|------|------|
| 有效信号率 | 高质量信号数 / 总采集数 | > 30% |
| 噪声过滤率 | 过滤掉的情报数 / 总采集数 | < 50% |
| 用户反馈率 | 有反馈的信号数 / 展示信号数 | > 5% |

**验收标准**：
- [ ] 给定 Admin 后台，当查看统计页面时，则显示有效信号率等指标

---

## 三、非功能需求

| 类型 | 要求 |
|------|------|
| 性能 | 每条情报处理时间 < 100ms |
| 准确率 | 噪声识别准确率 > 80% |
| 误杀率 | 高质量信号被误判为噪声 < 5% |

---

## 四、数据需求

### 4.1 数据输入

| 数据项 | 来源 | 格式 | 示例 |
|--------|------|------|------|
| 原始情报 | 采集器 | IntelItem | {title, content, source...} |
| 用户元数据 | 社区 API | JSON | {karma, posts_count} |
| 用户画像 | 社区主页 | JSON | {bio, github_url, created_at} |
| 历史发帖 | 社区 API | Array | [{title, content, score}...] |

### 4.2 数据输出

| 数据项 | 格式 | 用途 |
|--------|------|------|
| 噪声标记 | boolean | 过滤决策 |
| 质量评分 | int (0-100) | 排序展示 |
| 过滤原因 | string | 用户理解 |
| 客户价值等级 | string (S/A/B/C) | 行动优先级判断 |
| 用户画像摘要 | JSON | 辅助决策 |

---

## 五、技术约束

- 处理逻辑放在采集后的处理层，不影响采集性能
- 规则配置化，便于调整
- 反馈数据独立存储，不影响主流程

---

## 六、验收检查清单

- [ ] 噪声识别规则引擎实现
- [ ] 信号质量评分实现
- [ ] 低质量信号过滤实现
- [ ] 用户画像增强实现
- [ ] 客户价值等级判断实现
- [ ] Tavily AI 搜索采集器实现
- [ ] 搜索结果去重处理
- [ ] Admin 后台展示质量评分
- [ ] Admin 后台展示用户画像卡片
- [ ] Admin 后台管理搜索词配置
- [ ] 有效信号率统计展示
- [ ] 噪声反馈功能实现

---

## 七、技术设计要点

### 7.1 噪声关键词库

```
# 营销推广类
free trial, promo code, discount, check out my, try for free

# 无价值类
awesome, cool, thanks, +1, me too, lol

# 非目标用户类
hiring, job opening, we're looking for, enterprise license
```

### 7.2 核心关键词库（高价值信号）

```
# 成本相关
expensive, too costly, price increase, billing issue

# 限流相关
rate limit, quota exceeded, throttling, 429 error

# 迁移相关
switching from, alternative to, moving away from

# 功能需求
wish there was, need feature, would be great if
```

### 7.3 数据库变更

```sql
-- 情报表增加质量字段
ALTER TABLE intelligence_items
ADD COLUMN quality_score INT DEFAULT 0,
ADD COLUMN is_noise BOOLEAN DEFAULT FALSE,
ADD COLUMN filter_reason TEXT,
ADD COLUMN customer_tier VARCHAR(5); -- 'S', 'A', 'B', 'C'

-- 用户画像表
CREATE TABLE user_profiles (
    id SERIAL PRIMARY KEY,
    source_platform VARCHAR(50), -- 'hackernews', 'reddit', etc.
    source_user_id VARCHAR(100),
    username VARCHAR(200),
    bio TEXT,
    karma INT DEFAULT 0,
    github_url VARCHAR(500),
    website_url VARCHAR(500),
    created_at TIMESTAMP,
    first_seen_at TIMESTAMP DEFAULT NOW(),
    last_updated TIMESTAMP DEFAULT NOW(),
    UNIQUE(source_platform, source_user_id)
);

-- 用户发帖历史表（简化版，只存储摘要）
CREATE TABLE user_post_history (
    id SERIAL PRIMARY KEY,
    user_profile_id INT REFERENCES user_profiles(id),
    post_title TEXT,
    post_content TEXT,
    post_url VARCHAR(500),
    post_score INT DEFAULT 0,
    posted_at TIMESTAMP,
    captured_at TIMESTAMP DEFAULT NOW()
);

-- 反馈表
CREATE TABLE signal_feedback (
    id SERIAL PRIMARY KEY,
    intel_id VARCHAR(50) REFERENCES intelligence_items(id),
    feedback_type VARCHAR(20), -- 'noise' or 'valuable'
    created_at TIMESTAMP DEFAULT NOW()
);
```

### 7.4 用户画像获取策略

| 社区 | API | 可获取信息 | 限制 |
|------|-----|------------|------|
| HackerNews | `algolia.com/api/v1/users/{username}` | bio, karma, created_at | 无需认证 |
| Reddit | `oauth.reddit.com/user/{username}/about` | link_karma, comment_karma, created_utc | 需要 OAuth |
| StackExchange | `api.stackexchange.com/users/{id}` | reputation, badges, about_me | 有请求配额 |
| Dev.to | `developers.forem.com/api/v1/users/{id}` | username, summary, location | 无需认证 |

**策略**：
1. 发现有价值信号时，异步获取发帖人画像
2. 缓存用户画像，避免重复请求
3. 获取最近 10 条发帖历史作为摘要

### 7.5 规则引擎设计

**设计原则**：规则配置化，无需修改代码即可调整规则

**规则存储**：

```sql
-- 噪声规则配置表
CREATE TABLE noise_rules (
    id SERIAL PRIMARY KEY,
    rule_type VARCHAR(50),        -- 'keyword', 'length', 'regex', 'user_type'
    rule_name VARCHAR(100),       -- 规则名称
    rule_value TEXT,              -- 规则值（JSON 或字符串）
    weight INT DEFAULT 1,         -- 权重
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 示例数据
INSERT INTO noise_rules (rule_type, rule_name, rule_value, weight) VALUES
('keyword', '营销关键词', '["free trial", "promo code", "check out my"]', 10),
('keyword', '噪声关键词', '["awesome", "cool", "thanks", "+1"]', 5),
('keyword', '高价值信号关键词', '["expensive", "rate limit", "alternative", "cheaper"]', -10),
('length', '最小内容长度', '20', 3),
('regex', '邮箱模式', '\\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Z|a-z]{2,}\\b', 2);
```

**规则引擎接口**：

```go
type RuleEngine interface {
    // 加载所有激活的规则
    LoadRules(ctx context.Context) error

    // 评估情报是否为噪声
    Evaluate(item *IntelItem) (isNoise bool, score int, reasons []string)

    // 重新加载规则（热更新）
    Reload(ctx context.Context) error
}

// 规则类型
type Rule struct {
    ID        int
    Type      string      // keyword, length, regex, user_type
    Name      string
    Value     interface{} // 具体规则值
    Weight    int         // 权重（正数=噪声特征，负数=信号特征）
    IsActive  bool
}
```

**热更新机制**：

```
┌─────────────────┐
│  Admin 后台      │
│  规则配置页面    │
└────────┬────────┘
         │ 修改规则
         ▼
┌─────────────────┐
│  noise_rules 表  │
└────────┬────────┘
         │ 触发更新
         ▼
┌─────────────────┐
│  规则引擎        │
│  自动重新加载    │
└─────────────────┘
         │
         ▼
    新规则生效
    无需重启服务
```

**规则配置方式**：

| 方式 | 适用场景 | 优点 |
|------|----------|------|
| 数据库配置 | 需要频繁调整的规则 | 即时生效，无需重启 |
| YAML 文件 | 相对稳定的规则 | 版本可控，易于备份 |
| Admin 后台 UI | 非技术人员操作 | 可视化操作，降低门槛 |

**优先实现**：数据库配置 + Admin 后台 UI（一期），YAML 文件作为备份导入方式（二期）

### 7.6 质量评分量化标准

**评分公式**：
```
质量分 = 关键词分 × 0.30 + 影响力分 × 0.20 + 完整度分 × 0.20 + 时效性分 × 0.15 + 相关性分 × 0.15
```

**各维度量化标准**：

#### 7.6.1 关键词命中（权重 30%，满分 30 分）

| 命中数量 | 得分 | 说明 |
|----------|------|------|
| 0 个 | 0 分 | 无关键词命中 |
| 1 个 | 10 分 | 命中 1 个关键词 |
| 2 个 | 20 分 | 命中 2 个关键词 |
| 3+ 个 | 30 分 | 命中 3 个及以上（封顶） |

**关键词分类**：

| 类型 | 关键词 | 权重加成 |
|------|--------|----------|
| 高价值 | expensive, costly, bill, pricing, rate limit, quota, throttling | ×1.5 |
| 中价值 | alternative, cheaper, switch, migrate, frustrated, issue | ×1.0 |
| 低价值 | api, llm, openai, claude, gpt | ×0.5 |

**计算示例**：
```
命中 "expensive"（高价值）+ "alternative"（中价值）
= 10 × 1.5 + 10 × 1.0 = 25 分（不超过 30 分封顶）
```

#### 7.6.2 用户影响力（权重 20%，满分 20 分）

| 平台 | 计算公式 | 示例 |
|------|----------|------|
| HackerNews | min(karma / 100, 20) | karma 1500 → 15 分 |
| Reddit | min((link_karma + comment_karma) / 500, 20) | 总 karma 5000 → 10 分 |
| StackExchange | min(reputation / 500, 20) | reputation 3000 → 6 分 |
| 未知平台 | 默认 10 分 | 无 karma 数据 |

**影响力分级**：

| 影响力等级 | 分数范围 | 特征 |
|------------|----------|------|
| 高影响力 | 16-20 分 | karma > 1600，或帖子高赞 |
| 中影响力 | 11-15 分 | karma 500-1600 |
| 低影响力 | 0-10 分 | karma < 500，或新用户 |

#### 7.6.3 内容完整度（权重 20%，满分 20 分）

**评分组成**：

| 因子 | 计算 | 满分 |
|------|------|------|
| 长度因子 | min(内容长度 / 20, 10) | 10 分 |
| 结构因子 | 有完整句子 + 有上下文 + 有细节 | 10 分 |

**长度评分**：

| 内容长度 | 得分 |
|----------|------|
| < 50 字符 | 2 分 |
| 50-100 字符 | 5 分 |
| 100-200 字符 | 8 分 |
| > 200 字符 | 10 分 |

**结构评分**：

| 特征 | 得分 |
|------|------|
| 包含具体数字/金额 | +3 分 |
| 包含问题描述 | +3 分 |
| 包含产品/服务名称 | +2 分 |
| 包含情感表达 | +2 分 |

**计算示例**：
```
内容: "My OpenAI bill is $500/month, too expensive for my startup"
- 长度: 58 字符 → 5 分
- 包含金额 "$500" → +3 分
- 包含问题描述 → +3 分
- 包含产品名 "OpenAI" → +2 分
- 包含情感 "too expensive" → +2 分
- 总分: 5 + 3 + 3 + 2 + 2 = 15 分（不超过 20 分封顶）
```

#### 7.6.4 时效性（权重 15%，满分 15 分）

**时间衰减公式**：
```
时效分 = max(0, 15 - 发布距今天数 × 0.5)
```

| 发布时间 | 得分 |
|----------|------|
| 今天 | 15 分 |
| 1-7 天前 | 14.5-11.5 分 |
| 8-14 天前 | 11-8 分 |
| 15-30 天前 | 7.5-0 分 |
| > 30 天 | 0 分 |

#### 7.6.5 相关性（权重 15%，满分 15 分）

**评分标准**：

| 相关程度 | 得分 | 判断标准 |
|----------|------|----------|
| 高相关 | 15 分 | 直接讨论 AI API 价格/限流/问题 |
| 中相关 | 10 分 | 讨论 AI/LLM 相关话题 |
| 低相关 | 5 分 | 仅提及 AI/LLM 关键词 |
| 不相关 | 0 分 | 与 AI API 无关 |

**判断规则**：

| 条件 | 相关性 |
|------|--------|
| 包含 "API" + 任意 LLM 品牌 | 高相关 |
| 包含 "pricing/bill/cost" + LLM 品牌 | 高相关 |
| 包含 "rate limit/quota" + LLM 品牌 | 高相关 |
| 仅包含 LLM 品牌名 | 中相关 |
| 仅包含 "AI" 或 "LLM" | 低相关 |

---

### 7.7 用户画像 API 可行性

| 平台 | API 端点 | 需要 Key | 当前状态 | 优先级 |
|------|----------|----------|----------|--------|
| HackerNews | `https://hn.algolia.com/api/v1/users/{username}` | ❌ 不需要 | ✅ 可用 | P0 |
| Reddit | `https://oauth.reddit.com/user/{username}/about` | ✅ 需要 OAuth | ⚠️ 待配置 | P1 |
| StackExchange | `https://api.stackexchange.com/2.3/users/{id}` | ❌ 不需要（有配额） | ✅ 可用 | P1 |

**HackerNews API 示例**：

```bash
# 获取用户信息
curl "https://hn.algolia.com/api/v1/users/john_dev"

# 返回
{
  "id": "john_dev",
  "karma": 3500,
  "created_at": "2020-01-15T10:30:00Z",
  "about": "Indie hacker, building AI tools",
  "submitted": 150
}
```

**Reddit API 说明**：

- 需要 Reddit App 注册获取 OAuth credentials
- 配置步骤：
  1. 访问 https://www.reddit.com/prefs/apps
  2. 创建 "script" 类型应用
  3. 获取 client_id 和 client_secret
  4. 配置到 `.env` 中

```bash
# .env 配置
REDDIT_CLIENT_ID=xxx
REDDIT_CLIENT_SECRET=xxx
```

**优先级策略**：
- 一期：仅支持 HackerNews（无需额外配置）
- 二期：增加 Reddit 和 StackExchange

---

### 7.8 发帖历史获取

**HackerNews 获取用户发帖**：

```bash
# 获取用户最近发帖
curl "https://hn.algolia.com/api/v1/search_by_date?author=john_dev&tags=story&hitsPerPage=10"
```

**返回字段**：

| 字段 | 说明 |
|------|------|
| objectID | 帖子 ID |
| title | 标题 |
| url | 链接 |
| points | 得分 |
| created_at | 发布时间 |
| num_comments | 评论数 |

**存储策略**：
- 仅存储最近 10 条发帖摘要
- 用于判断用户是否有多个痛点
- 定期更新（每次获取画像时刷新）

---

**需求负责人**：Token Bridge Team
**创建日期**：2026-04-05
**最后更新**：2026-04-05
**状态**：已确认（V2 补充量化标准）
