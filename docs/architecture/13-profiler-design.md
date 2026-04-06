# 用户画像设计

> 获取发帖人信息，判断客户价值等级

---

## 一、概述

### 1.1 目标

用户画像模块负责：
- **获取发帖人信息**：从社区 API 获取用户资料
- **判断客户价值**：根据画像信息分级（S/A/B/C）
- **存储画像数据**：缓存画像信息，避免重复请求

### 1.2 触发条件

| 触发场景 | 条件 |
|----------|------|
| 自动触发 | 情报质量分 >= 70 |
| 手动触发 | 用户在 Admin 后台点击"获取画像" |
| 批量触发 | 定时任务刷新高质量用户画像 |

### 1.3 在系统中的位置

```
处理层 (Processor)
    │
    ├─ 质量评分 >= 70
    │       │
    │       ▼
    └─ [用户画像模块] → 客户分级 → 入库
           │
           ├─ HackerNews API
           ├─ Reddit API (二期)
           └─ StackExchange API (二期)
```

---

## 二、画像数据结构

### 2.1 数据库表

```sql
-- 用户画像表
CREATE TABLE user_profiles (
    id SERIAL PRIMARY KEY,
    source_platform VARCHAR(50) NOT NULL,   -- 'hackernews', 'reddit', 'stackexchange'
    source_user_id VARCHAR(100) NOT NULL,   -- 平台用户 ID
    username VARCHAR(200),                   -- 用户名

    -- 基本信息
    bio TEXT,                                -- 个人简介
    karma INT DEFAULT 0,                     -- 社区声望/Karma
    created_at TIMESTAMP,                    -- 账号创建时间

    -- 外部链接
    github_url VARCHAR(500),                 -- GitHub 链接
    website_url VARCHAR(500),                -- 个人网站

    -- 分析字段
    is_developer BOOLEAN DEFAULT FALSE,      -- 是否开发者
    has_active_project BOOLEAN DEFAULT FALSE,-- 是否有活跃项目
    has_budget BOOLEAN DEFAULT FALSE,        -- 是否有付费能力
    pain_point_count INT DEFAULT 0,          -- 痛点数量

    -- 客户等级
    customer_tier VARCHAR(5),                -- 'S', 'A', 'B', 'C'

    -- 时间戳
    first_seen_at TIMESTAMP DEFAULT NOW(),
    last_updated TIMESTAMP DEFAULT NOW(),
    last_fetched_at TIMESTAMP,               -- 最后获取时间

    UNIQUE(source_platform, source_user_id)
);

-- 索引
CREATE INDEX idx_user_profiles_platform ON user_profiles(source_platform);
CREATE INDEX idx_user_profiles_tier ON user_profiles(customer_tier);
CREATE INDEX idx_user_profiles_developer ON user_profiles(is_developer);

-- 用户发帖历史表（摘要）
CREATE TABLE user_post_history (
    id SERIAL PRIMARY KEY,
    user_profile_id INT REFERENCES user_profiles(id) ON DELETE CASCADE,

    -- 帖子信息
    post_id VARCHAR(100),                    -- 平台帖子 ID
    post_title TEXT,                         -- 标题
    post_content TEXT,                       -- 内容摘要
    post_url VARCHAR(500),                   -- 链接
    post_score INT DEFAULT 0,                -- 得分

    -- 时间
    posted_at TIMESTAMP,
    captured_at TIMESTAMP DEFAULT NOW()
);

-- 索引
CREATE INDEX idx_user_post_history_profile ON user_post_history(user_profile_id);
```

### 2.2 Go 结构体

```go
// UserProfile 用户画像
type UserProfile struct {
    ID             int        `json:"id"`
    SourcePlatform string     `json:"source_platform"` // hackernews, reddit, stackexchange
    SourceUserID   string     `json:"source_user_id"`
    Username       string     `json:"username"`

    // 基本信息
    Bio           string     `json:"bio"`
    Karma         int        `json:"karma"`
    AccountCreatedAt *time.Time `json:"account_created_at"`

    // 外部链接
    GitHubURL    string     `json:"github_url"`
    WebsiteURL   string     `json:"website_url"`

    // 分析字段
    IsDeveloper      bool   `json:"is_developer"`
    HasActiveProject bool   `json:"has_active_project"`
    HasBudget        bool   `json:"has_budget"`
    PainPointCount   int    `json:"pain_point_count"`

    // 客户等级
    CustomerTier string     `json:"customer_tier"` // S, A, B, C

    // 时间戳
    FirstSeenAt   time.Time `json:"first_seen_at"`
    LastUpdated   time.Time `json:"last_updated"`
    LastFetchedAt *time.Time `json:"last_fetched_at"`

    // 关联数据
    RecentPosts []PostSummary `json:"recent_posts,omitempty"`
}

// PostSummary 帖子摘要
type PostSummary struct {
    PostID      string     `json:"post_id"`
    Title       string     `json:"title"`
    Content     string     `json:"content"`
    URL         string     `json:"url"`
    Score       int        `json:"score"`
    PostedAt    time.Time  `json:"posted_at"`
}

// CustomerTier 客户等级
type CustomerTier string

const (
    TierS CustomerTier = "S"  // 高价值客户
    TierA CustomerTier = "A"  // 重点客户
    TierB CustomerTier = "B"  // 普通客户
    TierC CustomerTier = "C"  // 低价值客户
)
```

---

## 三、数据获取策略

### 3.1 HackerNews API（一期）

**API 端点**：

| 功能 | 端点 | 认证 |
|------|------|------|
| 获取用户信息 | `GET https://hn.algolia.com/api/v1/users/{username}` | 无需 |
| 获取用户发帖 | `GET https://hn.algolia.com/api/v1/search_by_date?author={username}&tags=story` | 无需 |

**获取用户信息**：

```go
// HNUserResponse HackerNews 用户响应
type HNUserResponse struct {
    ID        string `json:"objectID"`
    Username  string `json:"username"`
    Karma     int    `json:"karma"`
    CreatedAt string `json:"created_at"`
    About     string `json:"about"`  // Bio
}

// FetchHNProfile 获取 HackerNews 用户画像
func (f *HNFetcher) FetchProfile(ctx context.Context, username string) (*UserProfile, error) {
    url := fmt.Sprintf("https://hn.algolia.com/api/v1/users/%s", username)

    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    resp, err := f.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode == 404 {
        return nil, ErrUserNotFound
    }

    var hnUser HNUserResponse
    if err := json.NewDecoder(resp.Body).Decode(&hnUser); err != nil {
        return nil, err
    }

    // 转换为 UserProfile
    profile := &UserProfile{
        SourcePlatform: "hackernews",
        SourceUserID:   hnUser.ID,
        Username:       hnUser.Username,
        Bio:           hnUser.About,
        Karma:         hnUser.Karma,
    }

    // 解析创建时间
    if t, err := time.Parse(time.RFC3339, hnUser.CreatedAt); err == nil {
        profile.AccountCreatedAt = &t
    }

    // 提取 GitHub/Website 链接
    profile.GitHubURL = extractGitHubURL(hnUser.About)
    profile.WebsiteURL = extractWebsiteURL(hnUser.About)

    return profile, nil
}
```

**获取用户发帖**：

```go
// HNSearchResponse 搜索响应
type HNSearchResponse struct {
    Hits []struct {
        ObjectID    string `json:"objectID"`
        Title       string `json:"title"`
        URL         string `json:"url"`
        Points      int    `json:"points"`
        CreatedAt   string `json:"created_at"`
        NumComments int    `json:"num_comments"`
    } `json:"hits"`
}

// FetchRecentPosts 获取用户最近发帖
func (f *HNFetcher) FetchRecentPosts(ctx context.Context, username string, limit int) ([]PostSummary, error) {
    url := fmt.Sprintf(
        "https://hn.algolia.com/api/v1/search_by_date?author=%s&tags=story&hitsPerPage=%d",
        username, limit,
    )

    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    resp, err := f.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var searchResp HNSearchResponse
    if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
        return nil, err
    }

    var posts []PostSummary
    for _, hit := range searchResp.Hits {
        post := PostSummary{
            PostID: hit.ObjectID,
            Title:  hit.Title,
            URL:    hit.URL,
            Score:  hit.Points,
        }
        if t, err := time.Parse(time.RFC3339, hit.CreatedAt); err == nil {
            post.PostedAt = t
        }
        posts = append(posts, post)
    }

    return posts, nil
}
```

### 3.2 Reddit API（二期）

**认证要求**：OAuth 2.0

```go
// RedditFetcher Reddit 获取器
type RedditFetcher struct {
    clientID     string
    clientSecret string
    accessToken  string
    httpClient   *http.Client
}

// Authenticate 获取 OAuth Token
func (f *RedditFetcher) Authenticate(ctx context.Context) error {
    // 实现 OAuth 流程
    // POST https://www.reddit.com/api/v1/access_token
    // ...
}

// FetchProfile 获取用户画像
func (f *RedditFetcher) FetchProfile(ctx context.Context, username string) (*UserProfile, error) {
    url := fmt.Sprintf("https://oauth.reddit.com/user/%s/about", username)

    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    req.Header.Set("Authorization", "Bearer "+f.accessToken)

    // ...
}
```

**配置要求**：

```bash
# .env
REDDIT_CLIENT_ID=xxx
REDDIT_CLIENT_SECRET=xxx
```

### 3.3 StackExchange API（二期）

**API 端点**：`https://api.stackexchange.com/2.3/users/{ids}`

**配额限制**：每日 10000 次请求（可申请提高）

---

## 四、客户价值分级

### 4.1 分级标准

| 等级 | 条件 | 特征 | 行动建议 |
|------|------|------|----------|
| **S 级** | 4 项符合 | 开发者 + 有项目 + 有预算 + 多痛点 | 立即回帖引流 |
| **A 级** | 3 项符合 | 缺少其中一项 | 重点关注 |
| **B 级** | 2 项符合 | 普通潜在客户 | 一般关注 |
| **C 级** | 0-1 项符合 | 价值较低 | 暂不跟进 |

### 4.2 分级逻辑

```go
// Classifier 分级器
type Classifier struct{}

// Classify 客户价值分级
func (c *Classifier) Classify(profile *UserProfile) CustomerTier {
    score := 0

    // 判断 1：是否开发者
    if c.isDeveloper(profile) {
        score++
    }

    // 判断 2：是否有活跃项目
    if c.hasActiveProject(profile) {
        score++
    }

    // 判断 3：是否有付费能力
    if c.hasBudget(profile) {
        score++
    }

    // 判断 4：是否有多个痛点
    if c.hasMultiplePainPoints(profile) {
        score++
    }

    // 分级
    switch score {
    case 4:
        return TierS
    case 3:
        return TierA
    case 2:
        return TierB
    default:
        return TierC
    }
}

// isDeveloper 判断是否开发者
func (c *Classifier) isDeveloper(profile *UserProfile) bool {
    // 检查 Bio 中的开发者关键词
    bio := strings.ToLower(profile.Bio)
    developerKeywords := []string{
        "developer", "engineer", "programmer", "coder",
        "indie hacker", "founder", "cto", "tech lead",
        "software", "full stack", "backend", "frontend",
    }

    for _, kw := range developerKeywords {
        if strings.Contains(bio, kw) {
            return true
        }
    }

    // 检查是否有 GitHub 链接
    if profile.GitHubURL != "" {
        return true
    }

    return false
}

// hasActiveProject 判断是否有活跃项目
func (c *Classifier) hasActiveProject(profile *UserProfile) bool {
    // 检查 Bio 中的项目关键词
    bio := strings.ToLower(profile.Bio)
    projectKeywords := []string{
        "building", "working on", "developing", "my app", "my project",
        "startup", "side project", "saas", "product",
    }

    for _, kw := range projectKeywords {
        if strings.Contains(bio, kw) {
            return true
        }
    }

    // 检查发帖历史中的项目讨论
    for _, post := range profile.RecentPosts {
        content := strings.ToLower(post.Title + " " + post.Content)
        for _, kw := range projectKeywords {
            if strings.Contains(content, kw) {
                return true
            }
        }
    }

    return false
}

// hasBudget 判断是否有付费能力
func (c *Classifier) hasBudget(profile *UserProfile) bool {
    // 检查 Bio 和发帖中的预算相关内容
    budgetPatterns := []string{
        `\$\d+`,                           // 金额 $100
        `\d+\s*(dollars?|USD)`,            // 100 dollars
        `(budget|spend|pay|paying|paid)`,
        `(subscription|enterprise|team)`,
    }

    bio := profile.Bio
    for _, pattern := range budgetPatterns {
        if matched, _ := regexp.MatchString(pattern, bio); matched {
            return true
        }
    }

    // 高 Karma 用户通常有付费能力
    if profile.Karma > 5000 {
        return true
    }

    return false
}

// hasMultiplePainPoints 判断是否有多个痛点
func (c *Classifier) hasMultiplePainPoints(profile *UserProfile) bool {
    painKeywords := []string{
        "expensive", "costly", "rate limit", "throttling",
        "issue", "problem", "error", "fail", "broken",
        "frustrated", "annoying", "alternative", "switch",
    }

    count := 0

    // 检查发帖历史中的痛点
    for _, post := range profile.RecentPosts {
        content := strings.ToLower(post.Title + " " + post.Content)
        for _, kw := range painKeywords {
            if strings.Contains(content, kw) {
                count++
                break // 每个帖子只算一次
            }
        }
    }

    return count >= 2
}
```

---

## 五、缓存策略

### 5.1 缓存 TTL

| 数据类型 | TTL | 说明 |
|----------|-----|------|
| 用户画像 | 24 小时 | 基本信息变化较慢 |
| 发帖历史 | 6 小时 | 需要较新数据 |
| 客户等级 | 永久（直到重新评估） | 需要手动刷新 |

### 5.2 缓存实现

```go
// ProfileCache 画像缓存
type ProfileCache struct {
    db        *sql.DB
    localCache *cache.Cache
}

// Get 获取缓存的画像
func (c *ProfileCache) Get(ctx context.Context, platform, userID string) (*UserProfile, error) {
    // 先查本地缓存
    key := fmt.Sprintf("%s:%s", platform, userID)
    if profile, ok := c.localCache.Get(key); ok {
        return profile.(*UserProfile), nil
    }

    // 查数据库
    profile, err := c.getFromDB(ctx, platform, userID)
    if err != nil {
        return nil, err
    }

    // 检查是否过期
    if time.Since(profile.LastFetchedAt) < 24*time.Hour {
        c.localCache.Set(key, profile, cache.DefaultExpiration)
        return profile, nil
    }

    // 已过期，返回 nil 表示需要刷新
    return nil, ErrCacheExpired
}

// Set 设置缓存
func (c *ProfileCache) Set(ctx context.Context, profile *UserProfile) error {
    now := time.Now()
    profile.LastFetchedAt = &now

    // 存入数据库
    if err := c.saveToDB(ctx, profile); err != nil {
        return err
    }

    // 存入本地缓存
    key := fmt.Sprintf("%s:%s", profile.SourcePlatform, profile.SourceUserID)
    c.localCache.Set(key, profile, 24*time.Hour)

    return nil
}
```

---

## 六、接口定义

### 6.1 Profiler 接口

```go
// Profiler 用户画像接口
type Profiler interface {
    // FetchProfile 获取用户画像
    FetchProfile(ctx context.Context, platform, userID string) (*UserProfile, error)

    // ClassifyTier 客户价值分级
    ClassifyTier(profile *UserProfile) CustomerTier

    // RefreshProfile 刷新用户画像
    RefreshProfile(ctx context.Context, platform, userID string) (*UserProfile, error)

    // GetCachedProfile 获取缓存的画像
    GetCachedProfile(ctx context.Context, platform, userID string) (*UserProfile, error)
}

// Fetcher 平台获取器接口
type Fetcher interface {
    // FetchProfile 获取用户画像
    FetchProfile(ctx context.Context, userID string) (*UserProfile, error)

    // FetchRecentPosts 获取最近发帖
    FetchRecentPosts(ctx context.Context, userID string, limit int) ([]PostSummary, error)

    // Platform 返回平台标识
    Platform() string
}
```

### 6.2 实现结构

```go
// ProfilerImpl 画像器实现
type ProfilerImpl struct {
    fetchers  map[string]Fetcher  // 平台 -> 获取器
    cache     *ProfileCache
    storage   Storage
    classifier *Classifier
}

func (p *ProfilerImpl) FetchProfile(ctx context.Context, platform, userID string) (*UserProfile, error) {
    // 1. 检查缓存
    if profile, err := p.cache.Get(ctx, platform, userID); err == nil {
        return profile, nil
    }

    // 2. 获取对应平台的 fetcher
    fetcher, ok := p.fetchers[platform]
    if !ok {
        return nil, ErrPlatformNotSupported
    }

    // 3. 获取用户画像
    profile, err := fetcher.FetchProfile(ctx, userID)
    if err != nil {
        return nil, err
    }

    // 4. 获取最近发帖
    posts, err := fetcher.FetchRecentPosts(ctx, userID, 10)
    if err == nil {
        profile.RecentPosts = posts

        // 统计痛点数量
        profile.PainPointCount = countPainPoints(posts)
    }

    // 5. 进行客户分级
    profile.CustomerTier = string(p.classifier.Classify(profile))

    // 6. 分析字段
    profile.IsDeveloper = p.classifier.isDeveloper(profile)
    profile.HasActiveProject = p.classifier.hasActiveProject(profile)
    profile.HasBudget = p.classifier.hasBudget(profile)

    // 7. 缓存结果
    p.cache.Set(ctx, profile)

    return profile, nil
}
```

---

## 七、API 调用限制

### 7.1 各平台限制

| 平台 | 限制 | 应对策略 |
|------|------|----------|
| HackerNews | 无明确限制 | 合理间隔，避免滥用 |
| Reddit | 60 次/分钟 | 使用 OAuth，控制频率 |
| StackExchange | 10000 次/天 | 缓存优先，按需请求 |

### 7.2 限流实现

```go
// RateLimiter 限流器
type RateLimiter struct {
    limit  int
    window time.Duration
    ticker *time.Ticker
    sem    chan struct{}
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
    rl := &RateLimiter{
        limit:  limit,
        window: window,
        sem:    make(chan struct{}, limit),
    }

    // 定期释放
    go func() {
        for range time.NewTicker(window).C {
            for i := 0; i < limit; i++ {
                select {
                case <-rl.sem:
                default:
                }
            }
        }
    }()

    return rl
}

func (rl *RateLimiter) Wait(ctx context.Context) error {
    select {
    case rl.sem <- struct{}{}:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

---

## 八、错误处理

| 错误类型 | 处理方式 |
|----------|----------|
| 用户不存在 | 返回空画像，标记为 C 级 |
| API 限流 | 等待重试，使用缓存 |
| 网络超时 | 重试 3 次，使用缓存 |
| 解析错误 | 记录日志，返回部分数据 |

---

## 九、依赖关系

```
用户画像 (Profiler)
    │
    ├── 依赖
    │   ├── HTTP Client (API 调用)
    │   ├── 数据库 (user_profiles 表)
    │   └── 规则引擎 (客户分级规则)
    │
    └── 被依赖
        └── 处理层 (Processor)
```

---

**文档版本**：v1.0
**最后更新**：2026-04-05
