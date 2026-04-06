// Package search 提供搜索引擎采集功能
package search

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"token-bridge-crawler/internal/core"

	"github.com/google/uuid"
)

const (
	tavilyBaseURL = "https://api.tavily.com"
)

// TavilySearchRequest 搜索请求
type TavilySearchRequest struct {
	Query             string   `json:"query"`
	SearchDepth       string   `json:"search_depth"`              // "basic" 或 "advanced"
	MaxResults        int      `json:"max_results"`               // 返回结果数 (1-10)
	IncludeRawContent bool     `json:"include_raw_content"`       // 是否包含原始内容
	IncludeDomains    []string `json:"include_domains,omitempty"` // 限定域名
	ExcludeDomains    []string `json:"exclude_domains,omitempty"` // 排除域名
	Days              int      `json:"days,omitempty"`            // 时间范围（天）
}

// TavilySearchResponse 搜索响应
type TavilySearchResponse struct {
	Query         string `json:"query"`
	FollowUpQuery string `json:"follow_up_question"`
	Results       []struct {
		Title      string  `json:"title"`
		URL        string  `json:"url"`
		Content    string  `json:"content"`
		Score      float64 `json:"score"` // 相关度评分
		RawContent string  `json:"raw_content"`
	} `json:"results"`
	ResponseTime float64 `json:"response_time"`
}

// TavilyCollector Tavily搜索采集器
type TavilyCollector struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
	queries    []SearchQuery
	rateLimit  time.Duration
}

// SearchQuery 搜索词配置
type SearchQuery struct {
	Query    string `json:"query"`    // 搜索词
	Category string `json:"category"` // 分类：cost, rate_limit, migration, feature
	Priority int    `json:"priority"` // 优先级
}

// 优化后的搜索词 - 基于 Tavily 最佳实践
// 参考: https://docs.tavily.com/documentation/best-practices/best-practices-search
// 关键原则:
// 1. 查询保持在 400 字符以内
// 2. 复杂查询拆分为子查询
// 3. 使用 max_results=5 (默认值) 避免低质量结果
// 4. 使用 time_range 过滤最新内容
// 5. 使用 include_domains/exclude_domains 提高质量
var defaultSearchQueries = []SearchQuery{
	// ========== 迁移意愿（最高优先级 - 直接发现客户） ==========
	{Query: "switching from OpenAI to Claude migration experience", Category: "migration", Priority: 1},
	{Query: "best OpenAI API alternative 2025", Category: "migration", Priority: 1},
	{Query: "moving from OpenAI to Anthropic developer experience", Category: "migration", Priority: 1},
	{Query: "OpenAI vs Claude API comparison switch", Category: "migration", Priority: 1},

	// ========== 成本压力（高价值 - 价格敏感客户） ==========
	{Query: "OpenAI API too expensive cost complaint", Category: "cost", Priority: 1},
	{Query: "LLM API cost reduction cheaper alternative", Category: "cost", Priority: 1},
	{Query: "OpenAI pricing increase frustration", Category: "cost", Priority: 2},

	// ========== 速率限制（技术痛点 - 需要解决方案） ==========
	{Query: "OpenAI API rate limit 429 error solution", Category: "rate_limit", Priority: 1},
	{Query: "OpenAI rate limiting too restrictive workaround", Category: "rate_limit", Priority: 2},

	// ========== 功能需求（产品改进机会） ==========
	{Query: "OpenAI API limitations missing features", Category: "feature", Priority: 2},
	{Query: "ChatGPT API problems issues wishlist", Category: "feature", Priority: 2},
}

// NewTavilyCollector 创建Tavily采集器
func NewTavilyCollector(apiKey string) *TavilyCollector {
	if apiKey == "" {
		log.Println("[Tavily] Warning: API key is empty")
	}

	return &TavilyCollector{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:   tavilyBaseURL,
		queries:   defaultSearchQueries,
		rateLimit: 2 * time.Second, // Tavily 免费版建议每秒不超过1个请求
	}
}

// RateLimit 返回采集器的速率限制
func (c *TavilyCollector) RateLimit() time.Duration {
	return c.rateLimit
}

// Name 采集器名称
func (c *TavilyCollector) Name() string {
	return "tavily_search"
}

// IntelType 情报类型
func (c *TavilyCollector) IntelType() core.IntelType {
	return core.IntelTypeUserPain
}

// Source 来源标识
func (c *TavilyCollector) Source() string {
	return "tavily"
}

// Fetch 执行采集
func (c *TavilyCollector) Fetch(ctx context.Context) ([]core.IntelItem, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("tavily api key is required")
	}

	log.Printf("[Tavily] Starting search with %d queries", len(c.queries))

	var allItems []core.IntelItem

	for i, query := range c.queries {
		log.Printf("[Tavily] Searching: %s (category: %s)", query.Query, query.Category)

		items, err := c.search(ctx, query)
		if err != nil {
			log.Printf("[Tavily] Search error for query '%s': %v", query.Query, err)
			continue
		}

		allItems = append(allItems, items...)
		log.Printf("[Tavily] Query '%s' returned %d items", query.Query, len(items))

		// 避免请求过快
		if i < len(c.queries)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	log.Printf("[Tavily] Total collected %d items", len(allItems))
	return allItems, nil
}

// search 执行单个搜索
func (c *TavilyCollector) search(ctx context.Context, query SearchQuery) ([]core.IntelItem, error) {
	// 根据 Tavily 最佳实践优化参数
	// 参考: https://docs.tavily.com/documentation/best-practices/best-practices-search
	reqBody := TavilySearchRequest{
		Query:             query.Query,
		SearchDepth:       "basic", // basic = 高质量，1 credit（advanced 太贵，2 credits）
		MaxResults:        5,       // Tavily 官方建议：默认5，太高会降低质量
		IncludeRawContent: false,   // 关闭原始内容，节省带宽和处理时间
		Days:              30,      // 最近30天（保持时效性）
		// 可选：include_domains/exclude_domains 用于提高质量
		// 例如：排除低质量站点，专注技术社区
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/search", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tavily api error: status %d", resp.StatusCode)
	}

	var result TavilySearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 转换为情报格式
	var items []core.IntelItem
	for _, r := range result.Results {
		item := c.convertToIntelItem(r, query)
		items = append(items, item)
	}

	return items, nil
}

// convertToIntelItem 转换为情报格式
func (c *TavilyCollector) convertToIntelItem(result struct {
	Title      string  `json:"title"`
	URL        string  `json:"url"`
	Content    string  `json:"content"`
	Score      float64 `json:"score"`
	RawContent string  `json:"raw_content"`
}, query SearchQuery) core.IntelItem {
	// 解析来源平台
	platform := c.parsePlatform(result.URL)

	// 生成稳定的 source_id（用于去重）
	sourceID := generateSourceID(result.URL, result.Title)

	// 生成UUID格式的ID（数据库要求）
	id := uuid.New().String()

	// 使用原始内容（如果有）或摘要
	content := result.RawContent
	if content == "" {
		content = result.Content
	}

	item := core.IntelItem{
		ID:         id,
		IntelType:  core.IntelTypeUserPain,
		Source:     "tavily",
		SourceID:   sourceID, // 使用规范化后的稳定 ID
		Title:      result.Title,
		Content:    content,
		URL:        result.URL, // 保留原始 URL 用于访问
		CapturedAt: time.Now(),
		Status:     core.IntelStatusNew,
		CreatedAt:  time.Now(),
		Metadata: map[string]interface{}{
			"query":            query.Query,
			"category":         query.Category,
			"relevance_score":  result.Score,
			"platform":         platform,
			"normalized_url":   normalizeURL(result.URL), // 记录规范化后的 URL
			"source_id_method": "md5_url_title",          // 记录生成方法
		},
	}

	return item
}

// parsePlatform 从URL解析平台
func (c *TavilyCollector) parsePlatform(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "unknown"
	}

	host := u.Hostname()

	switch {
	case contains(host, "reddit.com"):
		return "reddit"
	case contains(host, "stackoverflow.com"), contains(host, "stackexchange.com"):
		return "stackoverflow"
	case contains(host, "news.ycombinator.com"):
		return "hackernews"
	case contains(host, "dev.to"):
		return "devto"
	case contains(host, "medium.com"):
		return "medium"
	case contains(host, "github.com"):
		return "github"
	case contains(host, "twitter.com"), contains(host, "x.com"):
		return "twitter"
	default:
		return "web"
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// normalizeURL 规范化 URL，去除查询参数和锚点，用于去重
func normalizeURL(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		// 如果解析失败，返回原始字符串的清理版本
		return strings.TrimSpace(urlStr)
	}

	// 去除查询参数和锚点
	u.RawQuery = ""
	u.Fragment = ""

	// 去除尾部斜杠，统一小写
	normalized := strings.TrimSuffix(u.String(), "/")
	normalized = strings.ToLower(normalized)

	return normalized
}

// generateSourceID 生成稳定的 source_id
// 使用规范化 URL + 标题的 MD5 哈希，确保同一内容始终生成相同 ID
func generateSourceID(urlStr, title string) string {
	normalizedURL := normalizeURL(urlStr)

	// 组合 URL 和标题生成唯一标识
	content := normalizedURL + "|" + strings.TrimSpace(title)
	hash := md5.Sum([]byte(content))

	return hex.EncodeToString(hash[:])
}
