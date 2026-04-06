# Tavily 采集器设计

> 通过 AI 搜索 API 获取更多情报

---

## 一、概述

### 1.1 目标

Tavily 采集器负责：
- **全网搜索**：不局限于单一平台
- **自然语言查询**：理解搜索意图
- **结果转换**：转换为标准情报格式

### 1.2 与现有采集器的关系

```
现有采集器                     新增采集器
├── price/                    ├── search/
│   ├── google.go             │   └── tavily.go ← 本次新增
│   ├── openai.go             │
│   ├── anthropic.go           │
│   └── openrouter.go          │
├── userpain/                 │
│   ├── hackernews.go         │
│   └── reddit.go             │
└── tool/
    └── ecosystem.go
```

### 1.3 特点

| 特点 | 说明 |
|------|------|
| 全网覆盖 | 搜索整个互联网，不局限于单一社区 |
| AI 增强 | 专为 AI Agent 设计，返回结构化结果 |
| 按需配置 | 可配置搜索词、时间范围、结果数量 |

---

## 二、Tavily API 集成

### 2.1 API 认证

**获取 API Key**：
1. 访问 https://tavily.com
2. 注册账号获取 API Key
3. 配置到 `.env`

```bash
# .env
TAVILY_API_KEY=tvly-dev-xxxxxxxxxxxx
```

### 2.2 API 端点

| 端点 | 用途 |
|------|------|
| `POST https://api.tavily.com/search` | 搜索 |

### 2.3 请求格式

```go
// TavilySearchRequest 搜索请求
type TavilySearchRequest struct {
    APIKey            string   `json:"api_key"`
    Query             string   `json:"query"`
    SearchDepth       string   `json:"search_depth"`       // "basic" 或 "advanced"
    MaxResults        int      `json:"max_results"`        // 返回结果数 (1-10)
    IncludeRawContent bool     `json:"include_raw_content"` // 是否包含原始内容
    IncludeDomains    []string `json:"include_domains"`    // 限定域名
    ExcludeDomains    []string `json:"exclude_domains"`    // 排除域名
    Days              int      `json:"days"`               // 时间范围（天）
}

// TavilySearchResponse 搜索响应
type TavilySearchResponse struct {
    Query         string `json:"query"`
    FollowUpQuery string `json:"follow_up_question"`

    Results []struct {
        Title       string  `json:"title"`
        URL         string  `json:"url"`
        Content     string  `json:"content"`
        Score       float64 `json:"score"`         // 相关度评分
        RawContent  string  `json:"raw_content"`
    } `json:"results"`

    ResponseTime float64 `json:"response_time"`
}
```

### 2.4 调用示例

```go
// TavilyClient Tavily 客户端
type TavilyClient struct {
    apiKey     string
    httpClient *http.Client
    baseURL    string
}

// Search 执行搜索
func (c *TavilyClient) Search(ctx context.Context, req TavilySearchRequest) (*TavilySearchResponse, error) {
    req.APIKey = c.apiKey

    body, _ := json.Marshal(req)
    httpReq, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/search", bytes.NewReader(body))
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("tavily api error: %d", resp.StatusCode)
    }

    var result TavilySearchResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    return &result, nil
}

// 使用示例
client := NewTavilyClient(apiKey)
resp, err := client.Search(ctx, TavilySearchRequest{
    Query:         "OpenAI API expensive complaints",
    SearchDepth:   "advanced",
    MaxResults:    10,
    IncludeRawContent: true,
    Days:          30,
})
```

---

## 三、搜索策略

### 3.1 搜索词配置

```go
// SearchQuery 搜索词配置
type SearchQuery struct {
    ID          int    `json:"id"`
    Query       string `json:"query"`        // 搜索词
    Category    string `json:"category"`     // 分类：cost, rate_limit, migration, feature
    Priority    int    `json:"priority"`     // 优先级
    IsActive    bool   `json:"is_active"`
    LastUsed    *time.Time `json:"last_used"`
}

// 默认搜索词
var defaultSearchQueries = []SearchQuery{
    {Query: "OpenAI API pricing expensive complaints", Category: "cost", Priority: 1},
    {Query: "LLM API cost too high alternative", Category: "cost", Priority: 1},
    {Query: "ChatGPT API billing issues", Category: "cost", Priority: 2},
    {Query: "rate limit OpenAI frustration", Category: "rate_limit", Priority: 1},
    {Query: "OpenAI 429 error too many requests", Category: "rate_limit", Priority: 2},
    {Query: "best cheaper alternative to OpenAI API", Category: "migration", Priority: 1},
    {Query: "switching from OpenAI to Claude API", Category: "migration", Priority: 2},
    {Query: "OpenAI vs Claude vs Gemini API comparison", Category: "migration", Priority: 3},
}
```

### 3.2 数据库存储

```sql
-- 搜索词配置表
CREATE TABLE tavily_search_queries (
    id SERIAL PRIMARY KEY,
    query TEXT NOT NULL,
    category VARCHAR(50),         -- cost, rate_limit, migration, feature
    priority INT DEFAULT 1,
    is_active BOOLEAN DEFAULT TRUE,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 搜索结果去重表
CREATE TABLE tavily_searched_urls (
    id SERIAL PRIMARY KEY,
    url VARCHAR(1000) NOT NULL,
    query_id INT REFERENCES tavily_search_queries(id),
    searched_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(url)
);

-- 索引
CREATE INDEX idx_tavily_queries_active ON tavily_search_queries(is_active);
CREATE INDEX idx_tavily_searched_urls ON tavily_searched_urls(url);
```

### 3.3 搜索调度

```go
// SearchScheduler 搜索调度器
type SearchScheduler struct {
    client   *TavilyClient
    storage  Storage
    queries  []SearchQuery
    interval time.Duration
}

// Run 执行搜索
func (s *SearchScheduler) Run(ctx context.Context) error {
    // 获取激活的搜索词
    queries, err := s.storage.GetActiveQueries(ctx)
    if err != nil {
        return err
    }

    var allResults []IntelItem

    for _, query := range queries {
        // 执行搜索
        resp, err := s.client.Search(ctx, TavilySearchRequest{
            Query:         query.Query,
            SearchDepth:   "advanced",
            MaxResults:    10,
            IncludeRawContent: true,
            Days:          30,
        })

        if err != nil {
            log.Printf("search error for %s: %v", query.Query, err)
            continue
        }

        // 转换结果
        for _, r := range resp.Results {
            // 去重检查
            if s.storage.IsURLSearched(ctx, r.URL) {
                continue
            }

            item := s.convertToIntelItem(r, query)
            allResults = append(allResults, item)

            // 记录已搜索
            s.storage.MarkURLSearched(ctx, r.URL, query.ID)
        }

        // 更新最后使用时间
        s.storage.UpdateQueryLastUsed(ctx, query.ID)

        // 避免请求过快
        time.Sleep(500 * time.Millisecond)
    }

    return nil
}
```

---

## 四、结果转换

### 4.1 转换逻辑

```go
// convertToIntelItem 转换为情报格式
func (s *SearchScheduler) convertToIntelItem(result TavilyResult, query SearchQuery) IntelItem {
    // 解析来源平台
    platform := parsePlatform(result.URL)

    // 解析作者
    author := parseAuthor(result.URL, result.Content)

    item := IntelItem{
        ID:             generateID(),
        SourceType:     "search",
        SourcePlatform: platform,
        SourceURL:      result.URL,
        Title:          result.Title,
        Content:        result.Content,
        RawContent:     result.RawContent,
        AuthorName:     author,
        RelevanceScore: result.Score,
        CollectedAt:    time.Now(),

        // 元数据
        Metadata: map[string]interface{}{
            "query_id":    query.ID,
            "query_text":  query.Query,
            "category":    query.Category,
            "source":      "tavily",
        },
    }

    // 解析发布时间（如果可能）
    if publishedAt := parsePublishedTime(result.Content); publishedAt != nil {
        item.PublishedAt = publishedAt
    }

    return item
}

// parsePlatform 从 URL 解析平台
func parsePlatform(url string) string {
    domains := map[string]string{
        "news.ycombinator.com":  "hackernews",
        "reddit.com":            "reddit",
        "stackoverflow.com":     "stackoverflow",
        "twitter.com":           "twitter",
        "x.com":                 "twitter",
        "dev.to":                "devto",
        "medium.com":            "medium",
    }

    for domain, platform := range domains {
        if strings.Contains(url, domain) {
            return platform
        }
    }

    return "web"
}
```

### 4.2 与现有采集器对齐

```go
// Collector 接口（与现有采集器一致）
type Collector interface {
    // Collect 执行采集
    Collect(ctx context.Context) ([]IntelItem, error)

    // Name 采集器名称
    Name() string
}

// TavilyCollector Tavily 采集器
type TavilyCollector struct {
    client    *TavilyClient
    storage   Storage
    scheduler *SearchScheduler
}

func (c *TavilyCollector) Collect(ctx context.Context) ([]IntelItem, error) {
    return c.scheduler.Run(ctx)
}

func (c *TavilyCollector) Name() string {
    return "tavily_search"
}
```

---

## 五、配置管理

### 5.1 配置结构

```yaml
# config.yaml
collectors:
  tavily:
    enabled: true
    api_key: ${TAVILY_API_KEY}
    schedule: "0 6 * * *"  # 每天早上 6 点
    max_results_per_query: 10
    search_depth: "advanced"
    days_range: 30
    rate_limit:
      requests_per_minute: 20
```

### 5.2 Admin 后台管理

**搜索词管理页面**：

| 字段 | 说明 |
|------|------|
| 搜索词 | 搜索查询文本 |
| 分类 | cost / rate_limit / migration / feature |
| 优先级 | 数字越大越先执行 |
| 状态 | 启用/禁用 |
| 最后执行 | 上次搜索时间 |

**操作功能**：
- 添加/编辑/删除搜索词
- 启用/禁用搜索词
- 手动触发搜索
- 查看搜索历史

---

## 六、去重处理

### 6.1 去重策略

```go
// Deduplicator 去重器
type Deduplicator struct {
    storage Storage
}

// IsDuplicate 检查是否重复
func (d *Deduplicator) IsDuplicate(ctx context.Context, item IntelItem) bool {
    // 1. URL 去重
    if d.storage.IsURLSearched(ctx, item.SourceURL) {
        return true
    }

    // 2. 标题相似度去重
    similarItems, _ := d.storage.FindSimilarTitles(ctx, item.Title, 0.9)
    if len(similarItems) > 0 {
        return true
    }

    // 3. 内容哈希去重
    contentHash := md5Hash(item.Content)
    if d.storage.IsContentHashExists(ctx, contentHash) {
        return true
    }

    return false
}

// 标题相似度计算
func titleSimilarity(title1, title2 string) float64 {
    // 使用 Levenshtein 距离或 SimHash
    // ...
}
```

---

## 七、错误处理

### 7.1 错误类型

| 错误类型 | 处理方式 |
|----------|----------|
| API Key 无效 | 记录错误，禁用采集器 |
| 请求限流 | 等待重试，指数退避 |
| 网络超时 | 重试 3 次 |
| 响应解析失败 | 记录日志，跳过该结果 |

### 7.2 重试策略

```go
// RetryConfig 重试配置
type RetryConfig struct {
    MaxRetries  int           // 最大重试次数
    InitialWait time.Duration // 初始等待时间
    MaxWait     time.Duration // 最大等待时间
    Multiplier  float64       // 退避倍数
}

func (c *TavilyClient) SearchWithRetry(ctx context.Context, req TavilySearchRequest, cfg RetryConfig) (*TavilySearchResponse, error) {
    var lastErr error
    wait := cfg.InitialWait

    for i := 0; i < cfg.MaxRetries; i++ {
        resp, err := c.Search(ctx, req)
        if err == nil {
            return resp, nil
        }

        lastErr = err

        // 检查是否为限流错误
        if isRateLimitError(err) {
            time.Sleep(wait)
            wait = time.Duration(float64(wait) * cfg.Multiplier)
            if wait > cfg.MaxWait {
                wait = cfg.MaxWait
            }
            continue
        }

        // 其他错误直接返回
        break
    }

    return nil, lastErr
}
```

---

## 八、性能要求

| 指标 | 要求 |
|------|------|
| 单次搜索耗时 | < 5 秒 |
| 每日搜索量 | 配置决定（默认 8 个搜索词 × 10 结果） |
| API 调用成本 | ~$0.01/次搜索 |

---

## 九、监控指标

| 指标 | 说明 |
|------|------|
| `tavily_search_total` | 总搜索次数 |
| `tavily_results_total` | 总结果数 |
| `tavily_duplicates_total` | 去重数量 |
| `tavily_errors_total` | 错误次数 |
| `tavily_latency_seconds` | 搜索延迟 |

---

## 十、依赖关系

```
Tavily 采集器
    │
    ├── 依赖
    │   ├── Tavily API (外部服务)
    │   ├── 配置文件 (搜索词)
    │   └── 数据库 (去重表)
    │
    └── 输出
        └── 处理层 (Processor)
```

---

**文档版本**：v1.0
**最后更新**：2026-04-05
