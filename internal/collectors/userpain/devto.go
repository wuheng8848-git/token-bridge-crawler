// Package userpain 提供用户痛点情报采集功能
package userpain

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"token-bridge-crawler/internal/core"
)

// DevToCollector Dev.to 用户痛点采集器
type DevToCollector struct {
	base     *BaseUserPainCollector
	client   *http.Client
	baseURL  string
	tags     []string
	keywords []string
}

// NewDevToCollector 创建 Dev.to 采集器
func NewDevToCollector() *DevToCollector {
	return &DevToCollector{
		base: NewBaseUserPainCollector(
			"devto_pain",
			"devto",
			6*time.Hour,
		),
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: "https://dev.to/api",
		tags: []string{
			"openai",
			"chatgpt",
			"llm",
			"ai",
			"anthropic",
			"claude",
		},
		keywords: []string{
			"pricing",
			"cost",
			"expensive",
			"alternative",
			"billing",
			"rate limit",
		},
	}
}

// Name 返回采集器名称
func (c *DevToCollector) Name() string {
	return c.base.Name()
}

// Source 返回数据源
func (c *DevToCollector) Source() string {
	return c.base.Source()
}

// IntelType 返回情报类型
func (c *DevToCollector) IntelType() core.IntelType {
	return c.base.IntelType()
}

// RateLimit 返回请求间隔
func (c *DevToCollector) RateLimit() time.Duration {
	return c.base.RateLimit()
}

// Fetch 执行采集
func (c *DevToCollector) Fetch(ctx context.Context) ([]core.IntelItem, error) {
	return c.CollectUserPains(ctx)
}

// CollectUserPains 从 Dev.to 采集用户痛点
func (c *DevToCollector) CollectUserPains(ctx context.Context) ([]core.IntelItem, error) {
	var items []core.IntelItem

	// 1. 按标签获取文章
	for _, tag := range c.tags {
		tagItems, err := c.getArticlesByTag(ctx, tag)
		if err != nil {
			continue
		}
		items = append(items, tagItems...)
		time.Sleep(200 * time.Millisecond)
	}

	// 2. 搜索文章
	for _, keyword := range c.keywords {
		searchItems, err := c.searchArticles(ctx, keyword)
		if err != nil {
			continue
		}
		items = append(items, searchItems...)
		time.Sleep(200 * time.Millisecond)
	}

	return items, nil
}

// getArticlesByTag 按标签获取文章
func (c *DevToCollector) getArticlesByTag(ctx context.Context, tag string) ([]core.IntelItem, error) {
	// Dev.to API: https://developers.forem.com/api/v1#tag/articles
	apiURL := fmt.Sprintf("%s/articles?tag=%s&per_page=30&top=7", c.baseURL, tag)

	return c.fetchArticles(ctx, apiURL, tag)
}

// searchArticles 搜索文章
func (c *DevToCollector) searchArticles(ctx context.Context, query string) ([]core.IntelItem, error) {
	// Dev.to 搜索 API
	apiURL := fmt.Sprintf("%s/articles?search=%s&per_page=20", c.baseURL, query)

	return c.fetchArticles(ctx, apiURL, query)
}

// fetchArticles 获取文章列表
func (c *DevToCollector) fetchArticles(ctx context.Context, apiURL, queryContext string) ([]core.IntelItem, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "TokenBridge-Intelligence/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Dev.to API returned status %d", resp.StatusCode)
	}

	// Dev.to 文章结构
	var articles []struct {
		ID          int      `json:"id"`
		Title       string   `json:"title"`
		Description string   `json:"description"`
		URL         string   `json:"url"`
		Slug        string   `json:"slug"`
		Tags        []string `json:"tag_list"`
		User        struct {
			Name     string `json:"name"`
			Username string `json:"username"`
		} `json:"user"`
		PublicReactionsCount   int       `json:"public_reactions_count"`
		PositiveReactionsCount int       `json:"positive_reactions_count"`
		CommentsCount          int       `json:"comments_count"`
		ReadingTime            int       `json:"reading_time"`
		PublishedAt            time.Time `json:"published_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&articles); err != nil {
		return nil, err
	}

	var items []core.IntelItem
	for _, article := range articles {
		// 放宽筛选：只要有互动即可
		if article.PublicReactionsCount < 1 && article.CommentsCount < 1 {
			continue
		}

		// 检查是否包含痛点相关内容
		fullText := article.Title + " " + article.Description
		if !c.isPainPointRelated(fullText) {
			continue
		}

		item := core.NewIntelItem(core.IntelTypeUserPain, "devto")
		item.Title = article.Title
		item.Content = article.Description
		item.URL = article.URL
		item.SourceID = fmt.Sprintf("%d", article.ID)

		painType := c.detectPainType(fullText)

		item.Metadata = core.Metadata{
			"platform":        "devto",
			"tags":            article.Tags,
			"author":          article.User.Name,
			"author_username": article.User.Username,
			"reactions_count": article.PublicReactionsCount,
			"comments_count":  article.CommentsCount,
			"reading_time":    article.ReadingTime,
			"pain_type":       painType,
			"query_context":   queryContext,
		}

		item.PublishedAt = &article.PublishedAt

		items = append(items, item)
	}

	return items, nil
}

// isPainPointRelated 检查是否与痛点相关
func (c *DevToCollector) isPainPointRelated(text string) bool {
	textLower := strings.ToLower(text)

	painKeywords := []string{
		"cost", "expensive", "pricing", "billing", "charge",
		"rate limit", "throttle", "quota",
		"alternative", "switch", "migrate",
		"problem", "issue", "error", "fail",
		"slow", "latency", "timeout",
		"vs", "comparison", "compare",
	}

	for _, keyword := range painKeywords {
		if strings.Contains(textLower, keyword) {
			return true
		}
	}
	return false
}

// detectPainType 检测痛点类型
func (c *DevToCollector) detectPainType(text string) string {
	textLower := strings.ToLower(text)

	if strings.Contains(textLower, "cost") || strings.Contains(textLower, "expensive") ||
		strings.Contains(textLower, "price") || strings.Contains(textLower, "pricing") ||
		strings.Contains(textLower, "bill") || strings.Contains(textLower, "charge") {
		return "cost"
	}
	if strings.Contains(textLower, "rate limit") || strings.Contains(textLower, "throttle") ||
		strings.Contains(textLower, "quota") {
		return "rate_limit"
	}
	if strings.Contains(textLower, "alternative") || strings.Contains(textLower, "switch") ||
		strings.Contains(textLower, "migrate") || strings.Contains(textLower, "vs") {
		return "switching"
	}
	if strings.Contains(textLower, "error") || strings.Contains(textLower, "fail") ||
		strings.Contains(textLower, "issue") || strings.Contains(textLower, "problem") {
		return "technical"
	}
	if strings.Contains(textLower, "slow") || strings.Contains(textLower, "latency") {
		return "performance"
	}

	return "general"
}
