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

// HackerNewsCollector HackerNews用户痛点采集器
type HackerNewsCollector struct {
	*BaseUserPainCollector
	client *http.Client
}

// NewHackerNewsCollector 创建HackerNews采集器
func NewHackerNewsCollector() *HackerNewsCollector {
	return &HackerNewsCollector{
		BaseUserPainCollector: NewBaseUserPainCollector(
			"hackernews_pain",
			"hackernews",
			6*time.Hour,
		),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CollectUserPains 从HackerNews采集用户痛点
func (c *HackerNewsCollector) CollectUserPains(ctx context.Context) ([]core.IntelItem, error) {
	var items []core.IntelItem

	// 搜索与AI API相关的痛点关键词
	keywords := []string{
		"OpenAI API cost",
		"API pricing too expensive",
		"LLM API bill shock",
		"switching LLM provider",
		"API rate limit",
		" Claude API alternative",
	}

	for _, keyword := range keywords {
		searchItems, err := c.searchStories(ctx, keyword)
		if err != nil {
			continue
		}
		items = append(items, searchItems...)
		time.Sleep(1 * time.Second) // 避免请求过快
	}

	return items, nil
}

// searchStories 搜索HackerNews故事
func (c *HackerNewsCollector) searchStories(ctx context.Context, query string) ([]core.IntelItem, error) {
	// 使用Algolia HN Search API
	url := fmt.Sprintf("https://hn.algolia.com/api/v1/search?query=%s&tags=story&numericFilters=created_at_i>%d",
		strings.ReplaceAll(query, " ", "+"),
		time.Now().AddDate(0, -1, 0).Unix(), // 最近一个月
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HN API returned status %d", resp.StatusCode)
	}

	var result struct {
		Hits []struct {
			ObjectID    string `json:"objectID"`
			Title       string `json:"title"`
			URL         string `json:"url"`
			Author      string `json:"author"`
			Points      int    `json:"points"`
			NumComments int    `json:"num_comments"`
			CreatedAt   string `json:"created_at"`
			StoryText   string `json:"story_text"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var items []core.IntelItem
	for _, hit := range result.Hits {
		// 只关注高互动内容
		if hit.Points < 50 && hit.NumComments < 20 {
			continue
		}

		item := core.NewIntelItem(core.IntelTypeUserPain, "hackernews")
		item.Title = hit.Title
		item.Content = hit.StoryText
		item.URL = hit.URL
		if item.URL == "" {
			item.URL = fmt.Sprintf("https://news.ycombinator.com/item?id=%s", hit.ObjectID)
		}
		item.SourceID = hit.ObjectID

		// 分析痛点类型
		painType := c.detectPainType(hit.Title + " " + hit.StoryText)

		// 设置元数据
		item.Metadata = core.Metadata{
			"platform":       "hackernews",
			"points":         hit.Points,
			"comments_count": hit.NumComments,
			"author":         hit.Author,
			"pain_type":      painType,
			"sentiment":      "negative",
			"query_matched":  query,
		}

		// 解析发布时间
		if t, err := time.Parse("2006-01-02T15:04:05.000Z", hit.CreatedAt); err == nil {
			item.PublishedAt = &t
		}

		items = append(items, item)
	}

	return items, nil
}

// detectPainType 检测痛点类型
func (c *HackerNewsCollector) detectPainType(text string) string {
	textLower := strings.ToLower(text)

	if strings.Contains(textLower, "cost") || strings.Contains(textLower, "expensive") || strings.Contains(textLower, "price") || strings.Contains(textLower, "bill") {
		return "cost"
	}
	if strings.Contains(textLower, "complex") || strings.Contains(textLower, "difficult") || strings.Contains(textLower, "confusing") {
		return "complexity"
	}
	if strings.Contains(textLower, "compliance") || strings.Contains(textLower, "regulation") || strings.Contains(textLower, "gdpr") {
		return "compliance"
	}
	if strings.Contains(textLower, "payment") || strings.Contains(textLower, "billing") || strings.Contains(textLower, "invoice") {
		return "payment"
	}
	if strings.Contains(textLower, "rate limit") || strings.Contains(textLower, "throttle") || strings.Contains(textLower, "quota") {
		return "rate_limit"
	}

	return "general"
}
