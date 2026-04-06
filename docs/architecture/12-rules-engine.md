# 规则引擎设计

> 噪声识别与质量评分的配置化规则系统

---

## 一、概述

### 1.1 目标

规则引擎负责对采集的情报进行：
- **噪声判断**：识别并过滤无效情报
- **质量评分**：计算情报质量分数（0-100）

### 1.2 核心能力

| 能力 | 说明 |
|------|------|
| 配置化规则 | 规则存储在数据库，无需修改代码 |
| 热更新 | 规则变更即时生效，无需重启服务 |
| 权重机制 | 支持规则权重，正数=噪声特征，负数=信号特征 |
| 可扩展 | 支持多种规则类型（关键词、长度、正则等） |

### 1.3 在系统中的位置

```
采集器 → [规则引擎] → 处理层 → 入库
           │
           ├─ 噪声判断
           └─ 质量评分
```

---

## 二、规则类型

### 2.1 支持的规则类型

| 类型 | 标识 | 说明 | 示例 |
|------|------|------|------|
| 关键词 | `keyword` | 匹配特定关键词 | "free trial", "expensive" |
| 长度 | `length` | 内容长度限制 | 最小 20 字符 |
| 正则 | `regex` | 正则表达式匹配 | 邮箱、URL 模式 |
| 用户类型 | `user_type` | 用户身份判断 | 企业采购、HR 招聘 |

### 2.2 规则权重含义

| 权重范围 | 含义 | 示例 |
|----------|------|------|
| 正数 (+1 ~ +10) | 噪声特征 | "free trial" (+10) |
| 负数 (-1 ~ -10) | 信号特征 | "expensive" (-10) |
| 0 | 中性 | 仅记录，不影响判断 |

---

## 三、规则数据结构

### 3.1 数据库表

```sql
-- 噪声规则配置表
CREATE TABLE noise_rules (
    id SERIAL PRIMARY KEY,
    rule_type VARCHAR(50) NOT NULL,     -- 'keyword', 'length', 'regex', 'user_type'
    rule_name VARCHAR(100) NOT NULL,    -- 规则名称
    rule_value TEXT NOT NULL,           -- 规则值（JSON 或字符串）
    weight INT NOT NULL DEFAULT 1,      -- 权重（正数=噪声，负数=信号）
    is_active BOOLEAN DEFAULT TRUE,     -- 是否启用
    priority INT DEFAULT 0,             -- 执行优先级（越大越先执行）
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 索引
CREATE INDEX idx_noise_rules_type ON noise_rules(rule_type);
CREATE INDEX idx_noise_rules_active ON noise_rules(is_active);
```

### 3.2 初始规则数据

```sql
-- 噪声关键词规则
INSERT INTO noise_rules (rule_type, rule_name, rule_value, weight, priority) VALUES
('keyword', '营销推广', '["free trial", "promo code", "discount", "check out my", "try for free"]', 10, 100),
('keyword', '无价值回复', '["awesome", "cool", "thanks", "+1", "me too", "lol", "nice"]', 8, 90),
('keyword', '非目标用户', '["hiring", "job opening", "we''re looking for", "enterprise license", "procurement"]', 7, 80);

-- 信号关键词规则（负权重）
INSERT INTO noise_rules (rule_type, rule_name, rule_value, weight, priority) VALUES
('keyword', '成本痛点', '["expensive", "too costly", "price increase", "billing issue", "too expensive"]', -10, 100),
('keyword', '限流痛点', '["rate limit", "quota exceeded", "throttling", "429 error", "too many requests"]', -10, 95),
('keyword', '迁移意愿', '["alternative to", "switching from", "moving away", "cheaper option"]', -10, 90),
('keyword', '功能需求', '["wish there was", "need feature", "would be great if", "looking for"]', -8, 85);

-- 长度规则
INSERT INTO noise_rules (rule_type, rule_name, rule_value, weight, priority) VALUES
('length', '最小内容长度', '{"min": 20}', 5, 50);

-- 正则规则
INSERT INTO noise_rules (rule_type, rule_name, rule_value, weight, priority) VALUES
('regex', '邮箱模式', '\\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Z|a-z]{2,}\\b', 3, 40),
('regex', '促销链接', '(promo|discount|coupon)\\.\\w+', 6, 45);
```

### 3.3 Go 结构体

```go
// Rule 规则定义
type Rule struct {
    ID       int         `json:"id"`
    Type     string      `json:"type"`      // keyword, length, regex, user_type
    Name     string      `json:"name"`
    Value    interface{} `json:"value"`     // 具体规则值
    Weight   int         `json:"weight"`    // 权重
    IsActive bool        `json:"is_active"`
    Priority int         `json:"priority"`  // 执行优先级
}

// KeywordRuleValue 关键词规则值
type KeywordRuleValue struct {
    Keywords []string `json:"keywords"`
    Mode     string   `json:"mode"` // "any" 或 "all"
}

// LengthRuleValue 长度规则值
type LengthRuleValue struct {
    Min int `json:"min"`
    Max int `json:"max,omitempty"`
}

// RegexRuleValue 正则规则值
type RegexRuleValue struct {
    Pattern string `json:"pattern"`
}

// RuleEvaluationResult 规则评估结果
type RuleEvaluationResult struct {
    IsNoise    bool     `json:"is_noise"`
    NoiseScore int      `json:"noise_score"`  // 正数表示噪声倾向
    SignalScore int     `json:"signal_score"` // 正数表示信号倾向
    MatchedRules []string `json:"matched_rules"`
    Reasons    []string `json:"reasons"`
}
```

---

## 四、规则评估流程

### 4.1 评估流程图

```
┌─────────────────┐
│   IntelItem     │
│  (原始情报)      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  LoadRules()    │  加载激活的规则（缓存）
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  按优先级排序    │
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────────┐
│           逐条评估规则                │
│  ┌─────────────────────────────────┐│
│  │ keyword 规则 → 关键词匹配        ││
│  │ length 规则 → 长度检查          ││
│  │ regex 规则  → 正则匹配          ││
│  │ user_type 规则 → 用户类型判断   ││
│  └─────────────────────────────────┘│
└────────┬────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────┐
│           汇总结果                   │
│  noise_score = Σ(正权重匹配)         │
│  signal_score = Σ(负权重匹配绝对值)  │
└────────┬────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────┐
│           判断结果                   │
│  is_noise = noise_score > threshold │
└─────────────────────────────────────┘
```

### 4.2 噪声判断逻辑

```go
// 噪声判断阈值
const (
    NoiseThreshold = 10  // 噪声分数超过此值判定为噪声
)

// IsNoise 判断是否为噪声
func (r *RuleEvaluationResult) IsNoise() bool {
    // 噪声分数 > 信号分数 + 阈值 → 噪声
    return r.NoiseScore > r.SignalScore + NoiseThreshold
}
```

### 4.3 评估示例

**示例 1：营销推广（噪声）**

```
输入：
"My new AI tool is live! Free trial at example.com, use promo code LAUNCH"

评估过程：
1. 命中 "free trial" → noise_score += 10
2. 命中 "promo code" → noise_score += 10
3. 命中促销链接正则 → noise_score += 6

结果：
noise_score = 26, signal_score = 0
判定：噪声 ✓
```

**示例 2：成本痛点（信号）**

```
输入：
"My OpenAI bill is $500/month, too expensive for my startup"

评估过程：
1. 命中 "too expensive" → signal_score += 10
2. 命中 "bill" → signal_score += 10
3. 内容长度 > 50 → 无惩罚

结果：
noise_score = 0, signal_score = 20
判定：信号 ✓
```

**示例 3：边界情况**

```
输入：
"API down"

评估过程：
1. 内容长度 < 20 → noise_score += 5
2. 无其他匹配

结果：
noise_score = 5, signal_score = 0
判定：非噪声（5 < 10），但质量低
```

---

## 五、热更新机制

### 5.1 缓存策略

```go
type RuleEngine struct {
    db        *sql.DB
    rules     []Rule
    rulesMu   sync.RWMutex
    lastLoad  time.Time
    ttl       time.Duration  // 缓存有效期
}

// LoadRules 加载规则（带缓存）
func (e *RuleEngine) LoadRules(ctx context.Context) error {
    e.rulesMu.RLock()
    if time.Since(e.lastLoad) < e.ttl && len(e.rules) > 0 {
        e.rulesMu.RUnlock()
        return nil
    }
    e.rulesMu.RUnlock()

    return e.Reload(ctx)
}
```

### 5.2 热更新触发方式

| 方式 | 实现复杂度 | 实时性 | 推荐场景 |
|------|------------|--------|----------|
| 定时轮询 | 低 | 秒级 | 通用场景 |
| 数据库通知 | 中 | 实时 | 高实时性要求 |
| HTTP 接口触发 | 低 | 实时 | Admin 后台操作后调用 |

**推荐实现：定时轮询 + HTTP 接口触发**

```go
// 定时轮询（后台任务）
func (e *RuleEngine) StartBackgroundRefresh(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    go func() {
        for {
            select {
            case <-ticker.C:
                e.Reload(ctx)
            case <-ctx.Done():
                ticker.Stop()
                return
            }
        }
    }()
}

// HTTP 接口触发（Admin 后台调用）
func (e *RuleEngine) Reload(ctx context.Context) error {
    rules, err := e.loadFromDB(ctx)
    if err != nil {
        return err
    }

    e.rulesMu.Lock()
    e.rules = rules
    e.lastLoad = time.Now()
    e.rulesMu.Unlock()

    return nil
}
```

---

## 六、接口定义

### 6.1 RuleEngine 接口

```go
// RuleEngine 规则引擎接口
type RuleEngine interface {
    // LoadRules 加载所有激活的规则
    LoadRules(ctx context.Context) error

    // Evaluate 评估情报
    // 返回：是否噪声、噪声分数、信号分数、匹配的规则、原因
    Evaluate(item *IntelItem) RuleEvaluationResult

    // Reload 重新加载规则（热更新）
    Reload(ctx context.Context) error

    // AddRule 添加规则
    AddRule(ctx context.Context, rule Rule) error

    // UpdateRule 更新规则
    UpdateRule(ctx context.Context, rule Rule) error

    // DeleteRule 删除规则
    DeleteRule(ctx context.Context, ruleID int) error

    // ListRules 列出所有规则
    ListRules(ctx context.Context, ruleType string) ([]Rule, error)
}
```

### 6.2 评估器接口

```go
// RuleEvaluator 规则评估器接口
type RuleEvaluator interface {
    // Evaluate 评估情报
    Evaluate(item *IntelItem, rule Rule) (matched bool, score int, reason string)

    // Type 返回规则类型
    Type() string
}

// 具体评估器实现
type KeywordEvaluator struct{}  // 关键词评估器
type LengthEvaluator struct{}   // 长度评估器
type RegexEvaluator struct{}    // 正则评估器
type UserTypeEvaluator struct{} // 用户类型评估器
```

---

## 七、配置管理

### 7.1 Admin 后台规则管理

**规则列表页面**：

| 字段 | 说明 |
|------|------|
| 规则名称 | 规则的显示名称 |
| 规则类型 | keyword / length / regex / user_type |
| 规则值 | 具体配置（JSON 或字符串） |
| 权重 | 正数=噪声，负数=信号 |
| 状态 | 启用/禁用 |
| 操作 | 编辑/删除 |

**规则编辑页面**：

- 关键词规则：支持多关键词输入，批量添加
- 长度规则：设置最小/最大长度
- 正则规则：正则表达式测试工具
- 权重滑块：可视化调整权重

### 7.2 规则导入/导出

```json
// 规则导出格式
{
  "version": "1.0",
  "exported_at": "2026-04-05T10:00:00Z",
  "rules": [
    {
      "rule_type": "keyword",
      "rule_name": "营销推广",
      "rule_value": "[\"free trial\", \"promo code\"]",
      "weight": 10,
      "is_active": true
    }
  ]
}
```

### 7.3 规则模板

系统提供预设规则模板，便于快速配置：

| 模板名称 | 包含规则 |
|----------|----------|
| 基础噪声过滤 | 营销关键词、短内容、无价值回复 |
| 客户信号识别 | 成本痛点、限流痛点、迁移意愿 |
| 完整版 | 基础 + 客户信号 + 正则规则 |

---

## 八、性能要求

| 指标 | 要求 |
|------|------|
| 单条评估时间 | < 10ms |
| 规则加载时间 | < 100ms |
| 支持规则数量 | > 1000 条 |
| 并发评估 | 支持 |

---

## 九、错误处理

| 错误类型 | 处理方式 |
|----------|----------|
| 规则加载失败 | 使用缓存规则，记录日志 |
| 正则表达式错误 | 跳过该规则，记录错误日志 |
| 数据库连接失败 | 降级为内存默认规则 |

---

## 十、依赖关系

```
规则引擎
    │
    ├── 依赖
    │   ├── PostgreSQL (noise_rules 表)
    │   └── 配置文件 (默认规则)
    │
    └── 被依赖
        ├── 处理层 (Processor)
        └── Admin 后台 (规则管理)
```

---

**文档版本**：v1.0
**最后更新**：2026-04-05
