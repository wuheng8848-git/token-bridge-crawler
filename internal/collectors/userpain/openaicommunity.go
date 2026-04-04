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

// OpenAICommunityCollector OpenAI 社区论坛用户痛点采集器
type OpenAICommunityCollector struct {
	base     *BaseUserPainCollector
	client   *http.Client
	baseURL  string
	keywords []string
}

// NewOpenAICommunityCollector 创建 OpenAI 社区论坛采集器
func NewOpenAICommunityCollector() *OpenAICommunityCollector {
	return &OpenAICommunityCollector{
		base: NewBaseUserPainCollector(
			"openai_community_pain",
			"openai_community",
			6*time.Hour,
		),
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: "https://community.openai.com",
		keywords: []string{
			"pricing",
			"cost",
			"expensive",
			"billing",
			"rate limit",
			"API overcharge",
			"alternative",
			"unexpected charge",
		},
	}
}

// Name 返回采集器名称
func (c *OpenAICommunityCollector) Name() string {
	return c.base.Name()
}

// Source 返回数据源
func (c *OpenAICommunityCollector) Source() string {
	return c.base.Source()
}

// IntelType 返回情报类型
func (c *OpenAICommunityCollector) IntelType() core.IntelType {
	return c.base.IntelType()
}

// RateLimit 返回请求间隔
func (c *OpenAICommunityCollector) RateLimit() time.Duration {
	return c.base.RateLimit()
}

// Fetch 执行采集
func (c *OpenAICommunityCollector) Fetch(ctx context.Context) ([]core.IntelItem, error) {
	return c.CollectUserPains(ctx)
}

// CollectUserPains 从 OpenAI 社区论坛采集用户痛点
func (c *OpenAICommunityCollector) CollectUserPains(ctx context.Context) ([]core.IntelItem, error) {
	var items []core.IntelItem

	for _, keyword := range c.keywords {
		searchItems, err := c.searchPosts(ctx, keyword)
		if err != nil {
			continue
		}
		items = append(items, searchItems...)
		time.Sleep(500 * time.Millisecond) // 避免请求过快
	}

	return items, nil
}

// searchPosts 搜索帖子
func (c *OpenAICommunityCollector) searchPosts(ctx context.Context, query string) ([]core.IntelItem, error) {
	// Discourse 搜索 API
	// https://community.openai.com/search?q=pricing
	searchURL := fmt.Sprintf("%s/search.json?q=%s", c.baseURL, query)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI community API returned status %d", resp.StatusCode)
	}

	// Discourse 搜索结果结构
	var result struct {
		Posts []struct {
			ID             int       `json:"id"`
			TopicID        int       `json:"topic_id"`
			TopicTitle     string    `json:"topic_title"`
			Content        string    `json:"blurb"`
			Username       string    `json:"username"`
			CreatedAt      time.Time `json:"created_at"`
			LikeCount      int       `json:"like_count"`
			ReplyCount     int       `json:"reply_count"`
			ReplyToPostNum int       `json:"reply_to_post_number"`
		} `json:"posts"`
		Topics []struct {
			ID           int       `json:"id"`
			Title        string    `json:"title"`
			PostsCount   int       `json:"posts_count"`
			ViewCount    int       `json:"views"`
			LikeCount    int       `json:"like_count"`
			CreatedAt    time.Time `json:"created_at"`
			LastPostedAt time.Time `json:"last_posted_at"`
			Slug         string    `json:"slug"`
		} `json:"topics"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var items []core.IntelItem

	// 处理帖子
	for _, post := range result.Posts {
		// 放宽筛选：只要有互动即可
		if post.LikeCount < 1 {
			continue
		}

		item := core.NewIntelItem(core.IntelTypeUserPain, "openai_community")
		item.Title = post.TopicTitle
		item.Content = post.Content
		item.URL = fmt.Sprintf("%s/t/%d/%d", c.baseURL, post.TopicID, post.ID)
		item.SourceID = fmt.Sprintf("%d", post.ID)

		painType := c.detectPainType(post.TopicTitle + " " + post.Content)

		item.Metadata = core.Metadata{
			"platform":     "openai_community",
			"topic_id":     post.TopicID,
			"author":       post.Username,
			"like_count":   post.LikeCount,
			"pain_type":    painType,
			"query":        query,
		}

		item.PublishedAt = &post.CreatedAt

		items = append(items, item)
	}

	// 处理主题
	for _, topic := range result.Topics {
		// 筛选：有一定关注度
		if topic.ViewCount < 100 && topic.LikeCount < 5 {
			continue
		}

		item := core.NewIntelItem(core.IntelTypeUserPain, "openai_community")
		item.Title = topic.Title
		item.Content = "" // 主题本身没有内容，需要单独获取
		item.URL = fmt.Sprintf("%s/t/%s/%d", c.baseURL, topic.Slug, topic.ID)
		item.SourceID = fmt.Sprintf("topic_%d", topic.ID)

		painType := c.detectPainType(topic.Title)

		item.Metadata = core.Metadata{
			"platform":      "openai_community",
			"topic_id":      topic.ID,
			"posts_count":   topic.PostsCount,
			"view_count":    topic.ViewCount,
			"like_count":    topic.LikeCount,
			"pain_type":     painType,
			"query":         query,
			"is_topic":      true,
		}

		item.PublishedAt = &topic.CreatedAt

		items = append(items, item)
	}

	return items, nil
}

// detectPainType 检测痛点类型
func (c *OpenAICommunityCollector) detectPainType(text string) string {
	textLower := strings.ToLower(text)

	if strings.Contains(textLower, "cost") || strings.Contains(textLower, "expensive") ||
		strings.Contains(textLower, "price") || strings.Contains(textLower, "pricing") ||
		strings.Contains(textLower, "overcharg") || strings.Contains(textLower, "bill") {
		return "cost"
	}
	if strings.Contains(textLower, "rate limit") || strings.Contains(textLower, "throttle") ||
		strings.Contains(textLower, "quota") || strings.Contains(textLower, "limit") {
		return "rate_limit"
	}
	if strings.Contains(textLower, "alternative") || strings.Contains(textLower, "switch") ||
		strings.Contains(textLower, "competitor") {
		return "switching"
	}
	if strings.Contains(textLower, "error") || strings.Contains(textLower, "fail") ||
		strings.Contains(textLower, "bug") || strings.Contains(textLower, "issue") {
		return "technical"
	}
	if strings.Contains(textLower, "slow") || strings.Contains(textLower, "latency") ||
		strings.Contains(textLower, "timeout") {
		return "performance"
	}

	return "general"
}
