// Package userpain 提供用户痛点情报采集功能
package userpain

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"token-bridge-crawler/internal/core"
)

// StackExchangeCollector StackExchange/Stack Overflow 用户痛点采集器
type StackExchangeCollector struct {
	base      *BaseUserPainCollector
	client    *http.Client
	site      string // stackoverflow, serverfault, etc.
	tags      []string
	questions []string // 搜索关键词
}

// NewStackExchangeCollector 创建 StackExchange 采集器
func NewStackExchangeCollector() *StackExchangeCollector {
	return &StackExchangeCollector{
		base: NewBaseUserPainCollector(
			"stackexchange_pain",
			"stackexchange",
			6*time.Hour,
		),
		client: &http.Client{Timeout: 30 * time.Second},
		site:   "stackoverflow", // 默认使用 Stack Overflow
		tags: []string{
			"openai-api",
			"chatgpt",
			"llm",
			"anthropic-claude",
		},
		questions: []string{
			"API pricing expensive",
			"API cost too high",
			"rate limit exceeded",
			"API billing unexpected",
			"OpenAI alternative",
			"Claude vs OpenAI cost",
		},
	}
}

// Name 返回采集器名称
func (c *StackExchangeCollector) Name() string {
	return c.base.Name()
}

// Source 返回数据源
func (c *StackExchangeCollector) Source() string {
	return c.base.Source()
}

// IntelType 返回情报类型
func (c *StackExchangeCollector) IntelType() core.IntelType {
	return c.base.IntelType()
}

// RateLimit 返回请求间隔
func (c *StackExchangeCollector) RateLimit() time.Duration {
	return c.base.RateLimit()
}

// Fetch 执行采集
func (c *StackExchangeCollector) Fetch(ctx context.Context) ([]core.IntelItem, error) {
	return c.CollectUserPains(ctx)
}

// CollectUserPains 从 StackExchange 采集用户痛点
func (c *StackExchangeCollector) CollectUserPains(ctx context.Context) ([]core.IntelItem, error) {
	var items []core.IntelItem

	// 1. 按标签搜索问题
	for _, tag := range c.tags {
		tagItems, err := c.searchByTag(ctx, tag)
		if err != nil {
			log.Printf("[StackExchange] Error searching tag %s: %v", tag, err)
			continue
		}
		items = append(items, tagItems...)
		time.Sleep(200 * time.Millisecond) // 避免请求过快
	}

	// 2. 按关键词搜索
	for _, query := range c.questions {
		queryItems, err := c.searchByQuery(ctx, query)
		if err != nil {
			log.Printf("[StackExchange] Error searching query %s: %v", query, err)
			continue
		}
		items = append(items, queryItems...)
		time.Sleep(200 * time.Millisecond)
	}

	return items, nil
}

// searchByTag 按标签搜索问题
func (c *StackExchangeCollector) searchByTag(ctx context.Context, tag string) ([]core.IntelItem, error) {
	// StackExchange API: https://api.stackexchange.com/2.3/questions
	apiURL := fmt.Sprintf(
		"https://api.stackexchange.com/2.3/questions?order=desc&sort=activity&tagged=%s&site=%s&pagesize=30&filter=withbody",
		url.QueryEscape(tag),
		c.site,
	)

	return c.fetchQuestions(ctx, apiURL, tag)
}

// searchByQuery 按关键词搜索
func (c *StackExchangeCollector) searchByQuery(ctx context.Context, query string) ([]core.IntelItem, error) {
	// StackExchange 搜索 API
	apiURL := fmt.Sprintf(
		"https://api.stackexchange.com/2.3/search?order=desc&sort=relevance&intitle=%s&site=%s&pagesize=20&filter=withbody",
		url.QueryEscape(query),
		c.site,
	)

	return c.fetchQuestions(ctx, apiURL, query)
}

// fetchQuestions 获取问题列表
func (c *StackExchangeCollector) fetchQuestions(ctx context.Context, apiURL, queryContext string) ([]core.IntelItem, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	// StackExchange API 要求 User-Agent
	req.Header.Set("User-Agent", "TokenBridge-Intelligence/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("StackExchange API returned status %d", resp.StatusCode)
	}

	var result struct {
		Items []struct {
			QuestionID   int      `json:"question_id"`
			Title        string   `json:"title"`
			Body         string   `json:"body"`
			Tags         []string `json:"tags"`
			Score        int      `json:"score"`
			ViewCount    int      `json:"view_count"`
			AnswerCount  int      `json:"answer_count"`
			IsAnswered   bool     `json:"is_answered"`
			CreationDate int64    `json:"creation_date"`
			Link         string   `json:"link"`
			Owner        struct {
				Name string `json:"display_name"`
			} `json:"owner"`
		} `json:"items"`
		HasMore        bool `json:"has_more"`
		QuotaMax       int  `json:"quota_max"`
		QuotaRemaining int  `json:"quota_remaining"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("[StackExchange] JSON decode error: %v", err)
		return nil, err
	}

	log.Printf("[StackExchange] Fetched %d items from %s", len(result.Items), queryContext)

	var items []core.IntelItem
	for _, q := range result.Items {
		// 放宽筛选条件：只要有浏览量或分数即可
		if q.ViewCount < 20 && q.Score < 1 {
			continue
		}

		item := core.NewIntelItem(core.IntelTypeUserPain, "stackexchange")
		item.Title = q.Title
		item.Content = c.stripHTML(q.Body)
		item.URL = q.Link
		item.SourceID = fmt.Sprintf("%d", q.QuestionID)

		// 分析痛点类型
		painType := c.detectPainType(q.Title + " " + q.Body)

		// 设置元数据
		item.Metadata = core.Metadata{
			"platform":      "stackexchange",
			"site":          c.site,
			"tags":          q.Tags,
			"score":         q.Score,
			"view_count":    q.ViewCount,
			"answer_count":  q.AnswerCount,
			"is_answered":   q.IsAnswered,
			"author":        q.Owner.Name,
			"pain_type":     painType,
			"query_context": queryContext,
		}

		// 解析发布时间
		createdAt := time.Unix(q.CreationDate, 0)
		item.PublishedAt = &createdAt

		items = append(items, item)
	}

	log.Printf("[StackExchange] Returning %d items after filtering", len(items))
	return items, nil
}

// detectPainType 检测痛点类型
func (c *StackExchangeCollector) detectPainType(text string) string {
	textLower := strings.ToLower(text)

	if strings.Contains(textLower, "cost") || strings.Contains(textLower, "expensive") ||
		strings.Contains(textLower, "price") || strings.Contains(textLower, "bill") ||
		strings.Contains(textLower, "pricing") || strings.Contains(textLower, "charge") {
		return "cost"
	}
	if strings.Contains(textLower, "rate limit") || strings.Contains(textLower, "throttle") ||
		strings.Contains(textLower, "quota") || strings.Contains(textLower, "limit exceeded") {
		return "rate_limit"
	}
	if strings.Contains(textLower, "complex") || strings.Contains(textLower, "difficult") ||
		strings.Contains(textLower, "confusing") || strings.Contains(textLower, "how to") {
		return "complexity"
	}
	if strings.Contains(textLower, "alternative") || strings.Contains(textLower, "switch") ||
		strings.Contains(textLower, "migrate") || strings.Contains(textLower, "vs") {
		return "switching"
	}
	if strings.Contains(textLower, "error") || strings.Contains(textLower, "fail") ||
		strings.Contains(textLower, "bug") || strings.Contains(textLower, "not working") {
		return "technical"
	}

	return "general"
}

// stripHTML 移除 HTML 标签
func (c *StackExchangeCollector) stripHTML(s string) string {
	// 简单的 HTML 标签移除
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return strings.TrimSpace(result.String())
}
