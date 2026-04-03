// Package community 提供社区情报采集功能
package community

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"token-bridge-crawler/internal/core"
)

// ProductHuntCollector Product Hunt采集器
type ProductHuntCollector struct {
	name     string
	source   string
	interval time.Duration
	client   *http.Client
}

// NewProductHuntCollector 创建Product Hunt采集器
func NewProductHuntCollector() *ProductHuntCollector {
	return &ProductHuntCollector{
		name:     "producthunt_collector",
		source:   "producthunt",
		interval: 6 * time.Hour,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name 返回采集器名称
func (c *ProductHuntCollector) Name() string {
	return c.name
}

// Source 返回数据源
func (c *ProductHuntCollector) Source() string {
	return c.source
}

// Interval 返回采集间隔
func (c *ProductHuntCollector) Interval() time.Duration {
	return c.interval
}

// IntelType 返回情报类型
func (c *ProductHuntCollector) IntelType() core.IntelType {
	return core.IntelTypeCommunity
}

// Fetch 采集Product Hunt数据
func (c *ProductHuntCollector) Fetch(ctx context.Context) ([]core.IntelItem, error) {
	var items []core.IntelItem

	// 1. 采集热门AI产品
	hotProducts, err := c.fetchHotProducts(ctx)
	if err == nil {
		items = append(items, hotProducts...)
	}

	// 2. 采集相关讨论
	discussions, err := c.fetchDiscussions(ctx)
	if err == nil {
		items = append(items, discussions...)
	}

	return items, nil
}

// RateLimit 返回请求间隔
func (c *ProductHuntCollector) RateLimit() time.Duration {
	return 2 * time.Second
}

// fetchHotProducts 采集热门AI产品
func (c *ProductHuntCollector) fetchHotProducts(ctx context.Context) ([]core.IntelItem, error) {
	// 使用Product Hunt API（需要认证）
	// 这里使用模拟数据作为示例

	// 模拟热门AI产品数据
	sampleProducts := []struct {
		Name        string
		Description string
		Votes       int
		Comments    int
		URL         string
		LaunchDate  string
	}{{
		Name:        "Token Bridge",
		Description: "AI API价格优化和管理平台",
		Votes:       1250,
		Comments:    89,
		URL:         "https://www.producthunt.com/posts/token-bridge",
		LaunchDate:  "2026-03-30",
	}, {
		Name:        "AI Gateway",
		Description: "智能API网关和负载均衡",
		Votes:       980,
		Comments:    67,
		URL:         "https://www.producthunt.com/posts/ai-gateway",
		LaunchDate:  "2026-03-29",
	}, {
		Name:        "LLM Cost Optimizer",
		Description: "大语言模型成本优化工具",
		Votes:       720,
		Comments:    45,
		URL:         "https://www.producthunt.com/posts/llm-cost-optimizer",
		LaunchDate:  "2026-03-28",
	}}

	var items []core.IntelItem
	for _, product := range sampleProducts {
		item := core.NewIntelItem(core.IntelTypeCommunity, "producthunt")
		item.Title = fmt.Sprintf("Product Hunt: %s", product.Name)
		item.Content = product.Description
		item.URL = product.URL
		item.SourceID = product.Name

		// 设置元数据
		item.Metadata = core.Metadata{
			"platform":      "producthunt",
			"votes":         product.Votes,
			"comments_count": product.Comments,
			"product_name":  product.Name,
			"launch_date":   product.LaunchDate,
			"category":      "ai_tool",
		}

		// 解析发布时间
		if t, err := time.Parse("2006-01-02", product.LaunchDate); err == nil {
			item.PublishedAt = &t
		}

		items = append(items, item)
	}

	return items, nil
}

// fetchDiscussions 采集相关讨论
func (c *ProductHuntCollector) fetchDiscussions(ctx context.Context) ([]core.IntelItem, error) {
	// 模拟讨论数据
	sampleDiscussions := []struct {
		Title      string
		Content    string
		Author     string
		Votes      int
		Comments   int
		URL        string
		PostDate   string
		Topic      string
	}{{
		Title:    "Best AI API cost optimization tools",
		Content:  "Looking for recommendations on tools that can help optimize AI API costs...",
		Author:   "AI_Enthusiast",
		Votes:    45,
		Comments: 12,
		URL:      "https://www.producthunt.com/discussions/best-ai-api-cost-optimization-tools",
		PostDate: "2026-03-30",
		Topic:    "cost_optimization",
	}, {
		Title:    "Token Bridge vs alternatives",
		Content:  "Has anyone tried Token Bridge for managing AI API costs? How does it compare to other solutions?",
		Author:   "DevOps_Guru",
		Votes:    32,
		Comments: 8,
		URL:      "https://www.producthunt.com/discussions/token-bridge-vs-alternatives",
		PostDate: "2026-03-29",
		Topic:    "comparison",
	}}

	var items []core.IntelItem
	for _, discussion := range sampleDiscussions {
		item := core.NewIntelItem(core.IntelTypeCommunity, "producthunt")
		item.Title = fmt.Sprintf("PH Discussion: %s", discussion.Title)
		item.Content = discussion.Content
		item.URL = discussion.URL
		item.SourceID = discussion.Title

		// 设置元数据
		item.Metadata = core.Metadata{
			"platform":       "producthunt",
			"author":         discussion.Author,
			"votes":          discussion.Votes,
			"comments_count": discussion.Comments,
			"topic":          discussion.Topic,
			"discussion_type": "forum",
		}

		// 解析发布时间
		if t, err := time.Parse("2006-01-02", discussion.PostDate); err == nil {
			item.PublishedAt = &t
		}

		items = append(items, item)
	}

	return items, nil
}
