# 情感分析模块设计

> 分析情报内容的情感倾向，识别痛点强度

---

## 一、概述

### 1.1 目标

情感分析模块负责：
- **情感判断**：正面/负面/中性
- **痛点强度**：抱怨、不满的程度
- **综合评分**：情感分纳入质量评分体系

### 1.2 与价值判断的关系

```
情报内容
    │
    ├──────────────────┬──────────────────┐
    │                  │                  │
    ▼                  ▼                  ▼
情感分析            价值判断           质量评分
(内容情绪)         (客户机会)         (情报质量)
    │                  │                  │
    │                  │                  │
    └──────────────────┼──────────────────┘
                       │
                       ▼
               ┌───────────────┐
               │   综合评分     │
               │ 情感 × 价值    │
               └───────────────┘
```

**定位差异**：

| 维度 | 情感分析 | 价值判断 |
|------|----------|----------|
| 关注点 | "他在说什么" | "他值不值得跟进" |
| 输入 | 内容文本 | 内容 + 用户画像 |
| 输出 | 情感倾向 + 痛点强度 | 客户等级 |
| 用途 | 判断痛点强度 | 判断商业机会 |

**关键洞察**：负面情感 = 痛点 = 高价值信号

---

## 二、情感词典

### 2.1 词典结构

```go
// SentimentDictionary 情感词典
type SentimentDictionary struct {
    Positive     map[string]int `json:"positive"`      // 正面词
    Negative     map[string]int `json:"negative"`      // 负面词
    Intensifiers map[string]float64 `json:"intensifiers"` // 强调词
    Negators     map[string]bool `json:"negators"`      // 否定词
}
```

### 2.2 负面词（高价值 - 痛点信号）

```go
// 强负面词（痛点强烈，得分高）
var strongNegativeWords = map[string]int{
    // 成本痛点
    "expensive":   10,
    "overpriced":  10,
    "costly":      9,
    "ripoff":      10,

    // 限流痛点
    "throttling":  10,
    "unusable":    10,
    "broken":      9,

    // 情绪表达
    "hate":        10,
    "terrible":    10,
    "awful":       9,
    "horrible":    9,
    "worst":       10,
    "ridiculous":  8,
    "garbage":     9,
    "trash":       9,

    // 挫败感
    "frustrating": 8,
    "frustrated":  8,
    "annoying":    7,
    "annoyed":     7,
    "fed up":      8,
}

// 中等负面词
var mediumNegativeWords = map[string]int{
    "expensive":  6,
    "slow":       5,
    "issue":      4,
    "problem":    5,
    "difficult":  4,
    "confusing":  5,
    "limited":    4,
    "restrictive": 5,
    "inconsistent": 5,
    "unreliable": 6,
}

// 弱负面词
var weakNegativeWords = map[string]int{
    "issue":     3,
    "bug":       3,
    "error":     3,
    "fail":      3,
    "down":      3,
    "bad":       2,
    "poor":      2,
}
```

### 2.3 正面词（低价值 - 满意用户）

```go
// 正面词（满意用户，不是潜在客户）
var positiveWords = map[string]int{
    "love":       8,
    "amazing":    7,
    "great":      6,
    "excellent":  7,
    "awesome":    6,
    "perfect":    8,
    "fantastic":  7,
    "wonderful":  6,
    "best":       7,
    "recommend":  5,
    "helpful":    4,
    "useful":     4,
}
```

### 2.4 强调词和否定词

```go
// 强调词（放大前后词的情感强度）
var intensifiers = map[string]float64{
    "very":       1.5,
    "really":     1.4,
    "extremely":  2.0,
    "so":         1.3,
    "too":        1.5,
    "absolutely": 1.8,
    "completely": 1.7,
    "totally":    1.6,
    "incredibly": 1.8,
    "insanely":   1.9,
}

// 否定词（反转情感）
var negators = map[string]bool{
    "not":      true,
    "no":       true,
    "never":    true,
    "none":     true,
    "nothing":  true,
    "neither":  true,
    "nobody":   true,
    "hardly":   true,
    "barely":   true,
    "rarely":   true,
    "doesn't":  true,
    "don't":    true,
    "isn't":    true,
    "aren't":   true,
    "wasn't":   true,
    "weren't":  true,
}
```

---

## 三、分析算法

### 3.1 核心算法

```go
// SentimentAnalyzer 情感分析器
type SentimentAnalyzer struct {
    dict *SentimentDictionary
}

// Analyze 分析情感
func (a *SentimentAnalyzer) Analyze(text string) SentimentResult {
    // 1. 预处理
    text = preprocess(text)
    words := tokenize(text)

    // 2. 逐词分析
    result := SentimentResult{
        Words: make([]WordSentiment, 0),
    }

    totalScore := 0
    negate := false
    intensify := 1.0

    for i, word := range words {
        // 检查否定
        if a.dict.Negators[word] {
            negate = !negate
            continue
        }

        // 检查强调
        if m, ok := a.dict.Intensifiers[word]; ok {
            intensify = m
            continue
        }

        // 计算词的情感分
        wordScore := 0
        if s, ok := a.dict.Positive[word]; ok {
            wordScore = s
        }
        if s, ok := a.dict.Negative[word]; ok {
            wordScore = -s
        }

        // 应用强调
        wordScore = int(float64(wordScore) * intensify)

        // 应用否定（反转）
        if negate {
            wordScore = -wordScore
        }

        totalScore += wordScore

        // 记录词的情感
        if wordScore != 0 {
            result.Words = append(result.Words, WordSentiment{
                Word:  word,
                Score: wordScore,
            })
        }

        // 重置状态
        negate = false
        intensify = 1.0
    }

    // 3. 归一化结果
    result.Score = totalScore
    result.Polarity = normalize(totalScore, -100, 100)
    result.Label = classifyLabel(result.Polarity)
    result.Intensity = math.Abs(result.Polarity)
    result.PainStrength = calculatePainStrength(totalScore)

    return result
}

// calculatePainStrength 计算痛点强度
// 负面情感 = 痛点，强度为正
func calculatePainStrength(score int) float64 {
    if score >= 0 {
        return 0
    }
    // 负分转正，归一化到 0-1
    strength := float64(-score) / 100
    if strength > 1 {
        strength = 1
    }
    return strength
}

// classifyLabel 分类标签
func classifyLabel(polarity float64) string {
    switch {
    case polarity > 0.3:
        return "positive"
    case polarity < -0.3:
        return "negative"
    default:
        return "neutral"
    }
}
```

### 3.2 结果结构

```go
// SentimentResult 情感分析结果
type SentimentResult struct {
    Score        int             `json:"score"`         // 原始分数 (-100 to 100)
    Polarity     float64         `json:"polarity"`      // 归一化 (-1 to 1)
    Label        string          `json:"label"`         // positive/negative/neutral
    Intensity    float64         `json:"intensity"`     // 情感强度 (0 to 1)
    PainStrength float64         `json:"pain_strength"` // 痛点强度 (0 to 1)
    Words        []WordSentiment `json:"words"`         // 命中的情感词
}

// WordSentiment 词情感
type WordSentiment struct {
    Word  string `json:"word"`
    Score int    `json:"score"`
}
```

---

## 四、与处理层集成

### 4.1 在处理流程中的位置

```
情报 → 噪声判断 → 质量评分 → 情感分析 → 用户画像 → 客户分级
                          │                      │
                          ▼                      ▼
                     痛点强度               价值判断
                          │                      │
                          └──────────┬───────────┘
                                     │
                                     ▼
                              综合评分
```

### 4.2 综合评分公式

```go
// calculateCompositeScore 综合评分
func calculateCompositeScore(qualityScore int, sentiment SentimentResult, tier string) int {
    // 质量分 (0-100)
    quality := qualityScore

    // 情感分：负面=高价值（痛点），转换为正向
    sentimentScore := int(sentiment.PainStrength * 30) // 最高 30 分

    // 客户等级加权
    tierBonus := 0
    switch tier {
    case "S":
        tierBonus = 20
    case "A":
        tierBonus = 15
    case "B":
        tierBonus = 10
    case "C":
        tierBonus = 5
    }

    // 综合
    composite := quality + sentimentScore + tierBonus

    if composite > 100 {
        composite = 100
    }

    return composite
}
```

### 4.3 更新处理层

```go
// ProcessResult 增加 Sentiment 字段
type ProcessResult struct {
    // ... 现有字段 ...

    // 新增：情感分析结果
    Sentiment      SentimentResult `json:"sentiment"`
    CompositeScore int             `json:"composite_score"` // 综合评分
}
```

---

## 五、配置化

### 5.1 数据库存储

```sql
-- 情感词典配置表
CREATE TABLE sentiment_words (
    id SERIAL PRIMARY KEY,
    word VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL,     -- 'positive', 'negative', 'intensifier', 'negator'
    score INT DEFAULT 1,           -- 情感分数
    category VARCHAR(50),          -- 分类：cost, rate_limit, emotion, etc.
    is_active BOOLEAN DEFAULT TRUE,
    UNIQUE(word, type)
);

-- 初始数据
INSERT INTO sentiment_words (word, type, score, category) VALUES
-- 负面词
('expensive', 'negative', 10, 'cost'),
('overpriced', 'negative', 10, 'cost'),
('frustrating', 'negative', 8, 'emotion'),
('hate', 'negative', 10, 'emotion'),
('throttling', 'negative', 10, 'rate_limit'),
-- 正面词
('love', 'positive', 8, 'emotion'),
('amazing', 'positive', 7, 'emotion'),
-- 强调词
('very', 'intensifier', 150, NULL),  -- 存储 150 表示 1.5 倍
('extremely', 'intensifier', 200, NULL),
-- 否定词
('not', 'negator', 0, NULL);
```

### 5.2 热更新支持

```go
// 支持热更新，与规则引擎类似
func (a *SentimentAnalyzer) Reload(ctx context.Context) error {
    dict, err := a.loadFromDB(ctx)
    if err != nil {
        return err
    }
    a.dict = dict
    return nil
}
```

---

## 六、性能要求

| 指标 | 要求 |
|------|------|
| 单条分析时间 | < 5ms |
| 批量吞吐量 | > 1000 条/秒 |
| 内存占用 | < 10MB |

---

## 七、示例

### 示例 1：强痛点

```
输入: "OpenAI's rate limiting is extremely frustrating, I can't get any work done!"

分析过程:
1. "rate limiting" → negative(10)
2. "extremely" → intensifier(2.0)
3. "frustrating" → negative(8) × 2.0 = -16

结果:
Score: -26
Polarity: -0.26
Label: "negative"
PainStrength: 0.26 (痛点强度中等)
Words: [{Word: "rate limiting", Score: -10}, {Word: "frustrating", Score: -16}]
```

### 示例 2：否定反转

```
输入: "This is not expensive at all"

分析过程:
1. "not" → negator (激活)
2. "expensive" → negative(10) × -1 = 10 (反转)

结果:
Score: 10
Polarity: 0.1
Label: "positive"
PainStrength: 0 (无痛点)
```

### 示例 3：满意用户

```
输入: "I love OpenAI API, it's amazing!"

分析过程:
1. "love" → positive(8)
2. "amazing" → positive(7)

结果:
Score: 15
Polarity: 0.15
Label: "positive"
PainStrength: 0 (无痛点)
结论: 不是潜在客户（已满意）
```

---

## 八、依赖关系

```
情感分析
    │
    ├── 依赖
    │   ├── 情感词典 (数据库)
    │   └── 配置 (可热更新)
    │
    └── 被依赖
        └── 处理层 (Processor)
```

---

**文档版本**：v1.0
**最后更新**：2026-04-05
