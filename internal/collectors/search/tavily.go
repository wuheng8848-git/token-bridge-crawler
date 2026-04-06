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

// 默认搜索词 - 针对迁移意愿等用户痛点
var defaultSearchQueries = []SearchQuery{
	// 成本压力
	{Query: "OpenAI API pricing expensive complaints 2024", Category: "cost", Priority: 1},
	{Query: "LLM API cost too high alternative cheaper", Category: "cost", Priority: 1},
	{Query: "ChatGPT API billing issues frustration", Category: "cost", Priority: 2},

	// 迁移意愿 - 这些是关键！
	{Query: "switching from OpenAI to Claude API experience", Category: "migration", Priority: 1},
	{Query: "best alternative to OpenAI API 2024", Category: "migration", Priority: 1},
	{Query: "migrating from OpenAI to Anthropic Claude", Category: "migration", Priority: 1},
	{Query: "OpenAI vs Claude API comparison switching", Category: "migration", Priority: 2},
	{Query: "looking for OpenAI alternative cheaper better", Category: "migration", Priority: 1},
	{Query: "move from OpenAI to Gemini Claude experience", Category: "migration", Priority: 2},

	// 速率限制
	{Query: "OpenAI API rate limit 429 error frustration", Category: "rate_limit", Priority: 1},
	{Query: "OpenAI too many requests limit increase", Category: "rate_limit", Priority: 2},

	// 功能需求
	{Query: "OpenAI API missing features wish list", Category: "feature", Priority: 2},
	{Query: "ChatGPT API limitations problems", Category: "feature", Priority: 2},
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
	reqBody := TavilySearchRequest{
		Query:             query.Query,
		SearchDepth:       "advanced",
		MaxResults:        10,
		IncludeRawContent: true,
		Days:              30, // 最近30天
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
