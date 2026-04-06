# 处理层设计

> 情报处理流水线：噪声过滤 → 质量评分 → 分级处理

---

## 一、概述

### 1.1 目标

处理层负责对采集的情报进行流水线处理，决定：
- 是否入库
- 如何展示
- 是否获取用户画像

### 1.2 在系统中的位置

```
采集器 → [处理层] → 数据库
           │
           ├─ 噪声判断 (调用规则引擎)
           ├─ 质量评分
           ├─ 分级决策
           └─ 触发用户画像获取
```

### 1.3 核心职责

| 职责 | 说明 |
|------|------|
| 噪声过滤 | 调用规则引擎，过滤噪声 |
| 质量评分 | 计算 0-100 分质量分数 |
| 分级处理 | 根据分数决定处理方式 |
| 入库决策 | 决定是否入库、如何标记 |

---

## 二、处理流程

### 2.1 完整处理流程

```
┌─────────────────────────────────────────────────────────────┐
│                     IntelItem 输入                           │
│                  (来自采集器的原始情报)                        │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                 第一步：噪声判断                              │
│                                                             │
│  调用 RuleEngine.Evaluate(item)                             │
│  ├─ 噪声分数 > 阈值 → 噪声                                   │
│  └─ 否则 → 可能是信号                                        │
└─────────────────────────┬───────────────────────────────────┘
                          │
          ┌───────────────┴───────────────┐
          │                               │
          ▼ 是噪声                        ▼ 非噪声
┌───────────────────┐          ┌─────────────────────────────┐
│   直接丢弃        │          │    第二步：质量评分          │
│   不入库          │          │                             │
│   记录统计        │          │  计算各维度分数：            │
│   +1 噪声计数     │          │  ├─ 关键词命中 (30%)         │
└───────────────────┘          │  ├─ 用户影响力 (20%)         │
                               │  ├─ 内容完整度 (20%)         │
                               │  ├─ 时效性 (15%)             │
                               │  └─ 相关性 (15%)             │
                               └─────────────┬───────────────┘
                                             │
                                             ▼
                               ┌─────────────────────────────┐
                               │    第三步：分级处理          │
                               │                             │
                               │  根据分数决定处理方式：       │
                               │  ├─ 0-20：低质量，不入库     │
                               │  ├─ 20-40：中低，入库隐藏    │
                               │  ├─ 40-70：中等，正常入库    │
                               │  └─ 70-100：高质量，入库     │
                               └─────────────┬───────────────┘
                                             │
                          ┌──────────────────┼──────────────────┐
                          │                  │                  │
                          ▼ 0-20             ▼ 20-70            ▼ 70-100
                   ┌─────────────┐    ┌─────────────┐    ┌─────────────────┐
                   │  不入库     │    │  入库       │    │  入库 + 画像    │
                   │  直接丢弃   │    │  标记等级   │    │  获取用户画像   │
                   │             │    │             │    │  客户分级       │
                   └─────────────┘    └─────────────┘    └─────────────────┘
```

### 2.2 处理决策表

| 阶段 | 条件 | 处理方式 | 是否入库 | 是否获取画像 |
|------|------|----------|----------|--------------|
| 噪声判断 | 噪声分数 > 阈值 | 丢弃 | ❌ | ❌ |
| 低质量 | 质量分 0-20 | 丢弃 | ❌ | ❌ |
| 中低质量 | 质量分 20-40 | 入库隐藏 | ✅ hidden | ❌ |
| 中等质量 | 质量分 40-70 | 正常入库 | ✅ | 按需 |
| 高质量 | 质量分 70-100 | 入库+画像 | ✅ | ✅ 自动 |

---

## 三、核心组件

### 3.1 组件架构

```
Processor (处理层)
    │
    ├── NoiseFilter (噪声过滤器)
    │   └── 调用 RuleEngine
    │
    ├── QualityScorer (质量评分器)
    │   ├── KeywordScorer (关键词评分)
    │   ├── InfluenceScorer (影响力评分)
    │   ├── CompletenessScorer (完整度评分)
    │   ├── TimelinessScorer (时效性评分)
    │   └── RelevanceScorer (相关性评分)
    │
    └── TierClassifier (分级器)
        └── 决定处理方式
```

### 3.2 接口定义

```go
// Processor 处理器接口
type Processor interface {
    // Process 处理单条情报
    Process(ctx context.Context, item IntelItem) ProcessResult

    // ProcessBatch 批量处理
    ProcessBatch(ctx context.Context, items []IntelItem) []ProcessResult
}

// ProcessResult 处理结果
type ProcessResult struct {
    Item          IntelItem       // 处理后的情报
    IsNoise       bool            // 是否噪声
    QualityScore  int             // 质量分数 (0-100)
    CustomerTier  string          // 客户等级 (S/A/B/C)
    ShouldStore   bool            // 是否入库
    ShouldProfile bool            // 是否获取画像
    IsHidden      bool            // 是否默认隐藏
    FilterReason  string          // 过滤原因
    ScoreDetails  ScoreDetails    // 评分详情
}

// ScoreDetails 评分详情
type ScoreDetails struct {
    KeywordScore     int `json:"keyword_score"`     // 关键词分 (满分30)
    InfluenceScore   int `json:"influence_score"`   // 影响力分 (满分20)
    CompletenessScore int `json:"completeness_score"` // 完整度分 (满分20)
    TimelinessScore  int `json:"timeliness_score"`  // 时效性分 (满分15)
    RelevanceScore   int `json:"relevance_score"`   // 相关性分 (满分15)
    TotalScore       int `json:"total_score"`       // 总分 (满分100)
}
```

---

## 四、评分实现

### 4.1 关键词评分（权重 30%，满分 30）

```go
// KeywordScorer 关键词评分器
type KeywordScorer struct {
    highValueKeywords []string // 高价值关键词
    midValueKeywords  []string // 中价值关键词
    lowValueKeywords  []string // 低价值关键词
}

func (s *KeywordScorer) Score(item IntelItem) int {
    content := strings.ToLower(item.Content + " " + item.Title)
    score := 0

    // 高价值关键词 ×1.5
    for _, kw := range s.highValueKeywords {
        if strings.Contains(content, strings.ToLower(kw)) {
            score += 15
        }
    }

    // 中价值关键词 ×1.0
    for _, kw := range s.midValueKeywords {
        if strings.Contains(content, strings.ToLower(kw)) {
            score += 10
        }
    }

    // 低价值关键词 ×0.5
    for _, kw := range s.lowValueKeywords {
        if strings.Contains(content, strings.ToLower(kw)) {
            score += 5
        }
    }

    // 封顶 30 分
    if score > 30 {
        score = 30
    }

    return score
}

// 关键词配置
var defaultHighValueKeywords = []string{
    "expensive", "costly", "bill", "pricing", "rate limit",
    "quota", "throttling", "alternative", "cheaper", "switch",
}

var defaultMidValueKeywords = []string{
    "issue", "problem", "frustrated", "migration", "migrate",
    "complaint", "expensive", "limit",
}

var defaultLowValueKeywords = []string{
    "api", "llm", "openai", "claude", "gpt", "model",
}
```

### 4.2 用户影响力评分（权重 20%，满分 20）

```go
// InfluenceScorer 影响力评分器
type InfluenceScorer struct{}

func (s *InfluenceScorer) Score(item IntelItem) int {
    karma := item.AuthorKarma

    // 不同平台 karma 换算
    switch item.SourcePlatform {
    case "hackernews":
        return min(karma/100, 20)
    case "reddit":
        return min(karma/500, 20)
    case "stackexchange":
        return min(karma/500, 20)
    default:
        return 10 // 默认中等分数
    }
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

### 4.3 内容完整度评分（权重 20%，满分 20）

```go
// CompletenessScorer 完整度评分器
type CompletenessScorer struct{}

func (s *CompletenessScorer) Score(item IntelItem) int {
    score := 0

    // 长度因子（满分 10 分）
    length := len(item.Content)
    switch {
    case length > 200:
        score += 10
    case length > 100:
        score += 8
    case length > 50:
        score += 5
    default:
        score += 2
    }

    // 结构因子（满分 10 分）
    content := item.Content

    // 包含具体数字/金额
    if regexp.MustCompile(`\$\d+|\d+\s*(dollars?|USD)`).MatchString(content) {
        score += 3
    }

    // 包含问题描述
    if regexp.MustCompile(`(problem|issue|error|fail|broken)`).MatchString(content) {
        score += 3
    }

    // 包含产品/服务名称
    if regexp.MustCompile(`(OpenAI|Claude|GPT|Anthropic|OpenRouter)`).MatchString(content) {
        score += 2
    }

    // 包含情感表达
    if regexp.MustCompile(`(frustrated|annoying|terrible|awful|expensive|too\s+\w+)`).MatchString(content) {
        score += 2
    }

    // 封顶 20 分
    if score > 20 {
        score = 20
    }

    return score
}
```

### 4.4 时效性评分（权重 15%，满分 15）

```go
// TimelinessScorer 时效性评分器
type TimelinessScorer struct{}

func (s *TimelinessScorer) Score(item IntelItem) int {
    // 发布时间距今天数
    days := time.Since(item.PublishedAt).Hours() / 24

    // 时间衰减公式：max(0, 15 - 天数 × 0.5)
    score := int(15 - days*0.5)

    if score < 0 {
        score = 0
    }

    return score
}
```

### 4.5 相关性评分（权重 15%，满分 15）

```go
// RelevanceScorer 相关性评分器
type RelevanceScorer struct{}

func (s *RelevanceScorer) Score(item IntelItem) int {
    content := strings.ToLower(item.Content + " " + item.Title)

    // 高相关：直接讨论 AI API 价格/限流/问题
    highPatterns := []string{
        `(api|API).*(pricing|price|cost|bill)`,
        `(rate limit|quota|throttling)`,
        `(OpenAI|Claude|GPT|Anthropic).*(expensive|cost|alternative)`,
    }

    for _, pattern := range highPatterns {
        if matched, _ := regexp.MatchString(pattern, content); matched {
            return 15
        }
    }

    // 中相关：讨论 AI/LLM 相关话题
    midPatterns := []string{
        `(LLM|AI|GPT|language model)`,
        `(OpenAI|Claude|Anthropic|OpenRouter)`,
    }

    for _, pattern := range midPatterns {
        if matched, _ := regexp.MatchString(pattern, content); matched {
            return 10
        }
    }

    // 低相关：仅提及关键词
    lowPatterns := []string{
        `AI`, `machine learning`, `neural`,
    }

    for _, pattern := range lowPatterns {
        if matched, _ := regexp.MatchString(pattern, content); matched {
            return 5
        }
    }

    return 0
}
```

### 4.6 综合评分

```go
// QualityScorer 综合评分器
type QualityScorer struct {
    keyword     *KeywordScorer
    influence   *InfluenceScorer
    completeness *CompletenessScorer
    timeliness  *TimelinessScorer
    relevance   *RelevanceScorer
}

func (s *QualityScorer) Score(item IntelItem) ScoreDetails {
    details := ScoreDetails{
        KeywordScore:      s.keyword.Score(item),
        InfluenceScore:    s.influence.Score(item),
        CompletenessScore: s.completeness.Score(item),
        TimelinessScore:   s.timeliness.Score(item),
        RelevanceScore:    s.relevance.Score(item),
    }

    details.TotalScore = details.KeywordScore +
        details.InfluenceScore +
        details.CompletenessScore +
        details.TimelinessScore +
        details.RelevanceScore

    return details
}
```

---

## 五、分级处理逻辑

### 5.1 分级决策器

```go
// TierClassifier 分级决策器
type TierClassifier struct{}

// Classify 根据质量分数决定处理方式
func (c *TierClassifier) Classify(score int) ProcessDecision {
    switch {
    case score < 20:
        return ProcessDecision{
            ShouldStore:   false,
            ShouldProfile: false,
            IsHidden:      false,
            Tier:          "",
            Reason:        "low_quality",
        }

    case score < 40:
        return ProcessDecision{
            ShouldStore:   true,
            ShouldProfile: false,
            IsHidden:      true,
            Tier:          "C",
            Reason:        "mid_low_quality",
        }

    case score < 70:
        return ProcessDecision{
            ShouldStore:   true,
            ShouldProfile: false, // 按需获取
            IsHidden:      false,
            Tier:          "C",
            Reason:        "mid_quality",
        }

    default: // score >= 70
        return ProcessDecision{
            ShouldStore:   true,
            ShouldProfile: true, // 自动获取
            IsHidden:      false,
            Tier:          "", // 待画像后确定
            Reason:        "high_quality",
        }
    }
}

// ProcessDecision 处理决策
type ProcessDecision struct {
    ShouldStore   bool   // 是否入库
    ShouldProfile bool   // 是否获取画像
    IsHidden      bool   // 是否默认隐藏
    Tier          string // 客户等级
    Reason        string // 决策原因
}
```

### 5.2 客户价值分级（高质量情报）

当质量分 >= 70 时，获取用户画像后进行客户价值分级：

```go
// CustomerTierClassifier 客户价值分级器
func ClassifyCustomerTier(profile UserProfile) string {
    score := 0

    // 是开发者？
    if isDeveloper(profile.Bio) {
        score++
    }

    // 有实际项目？
    if hasActiveProject(profile) {
        score++
    }

    // 有付费能力？
    if hasBudget(profile) {
        score++
    }

    // 有多个痛点？
    if hasMultiplePainPoints(profile) {
        score++
    }

    // 分级
    switch score {
    case 4:
        return "S"
    case 3:
        return "A"
    case 2:
        return "B"
    default:
        return "C"
    }
}

func isDeveloper(bio string) bool {
    keywords := []string{"developer", "engineer", "indie hacker", "founder", "programmer"}
    bio = strings.ToLower(bio)
    for _, kw := range keywords {
        if strings.Contains(bio, kw) {
            return true
        }
    }
    return false
}
```

---

## 六、处理实现

### 6.1 Processor 实现

```go
// ProcessorImpl 处理器实现
type ProcessorImpl struct {
    ruleEngine RuleEngine
    scorer     *QualityScorer
    classifier *TierClassifier
    profiler   Profiler
    storage    Storage
}

func (p *ProcessorImpl) Process(ctx context.Context, item IntelItem) ProcessResult {
    result := ProcessResult{Item: item}

    // 第一步：噪声判断
    ruleResult := p.ruleEngine.Evaluate(&item)
    if ruleResult.IsNoise {
        result.IsNoise = true
        result.ShouldStore = false
        result.FilterReason = strings.Join(ruleResult.Reasons, "; ")
        return result
    }

    // 第二步：质量评分
    result.ScoreDetails = p.scorer.Score(item)
    result.QualityScore = result.ScoreDetails.TotalScore

    // 第三步：分级处理
    decision := p.classifier.Classify(result.QualityScore)
    result.ShouldStore = decision.ShouldStore
    result.ShouldProfile = decision.ShouldProfile
    result.IsHidden = decision.IsHidden
    result.FilterReason = decision.Reason

    // 第四步：获取用户画像（高质量）
    if decision.ShouldProfile && item.AuthorID != "" {
        profile, err := p.profiler.FetchProfile(ctx, item.SourcePlatform, item.AuthorID)
        if err == nil {
            result.CustomerTier = ClassifyCustomerTier(*profile)
            result.Item.AuthorProfile = profile
        }
    }

    return result
}
```

### 6.2 批量处理

```go
func (p *ProcessorImpl) ProcessBatch(ctx context.Context, items []IntelItem) []ProcessResult {
    results := make([]ProcessResult, len(items))

    // 并发处理
    var wg sync.WaitGroup
    for i, item := range items {
        wg.Add(1)
        go func(idx int, itm IntelItem) {
            defer wg.Done()
            results[idx] = p.Process(ctx, itm)
        }(i, item)
    }
    wg.Wait()

    return results
}
```

---

## 七、性能要求

| 指标 | 要求 | 实现方式 |
|------|------|----------|
| 单条处理时间 | < 100ms | 评分计算优化 |
| 批量处理吞吐量 | > 1000 条/秒 | 并发处理 |
| 内存占用 | < 100MB | 流式处理 |

---

## 八、错误处理

| 错误类型 | 处理方式 |
|----------|----------|
| 规则引擎异常 | 默认非噪声，继续处理 |
| 画像获取失败 | 记录日志，标记为 C 级 |
| 数据库写入失败 | 重试 3 次，记录错误 |

---

## 九、依赖关系

```
处理层 (Processor)
    │
    ├── 依赖
    │   ├── 规则引擎 (RuleEngine)
    │   ├── 用户画像 (Profiler)
    │   └── 数据存储 (Storage)
    │
    └── 被依赖
        ├── 采集器 (调用 Process)
        └── 定时任务 (批量处理)
```

---

**文档版本**：v1.0
**最后更新**：2026-04-05
