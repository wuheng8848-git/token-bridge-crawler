// Package community 提供社区情报采集功能
package community

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"token-bridge-crawler/internal/core"
)

// LinkedInCollector LinkedIn采集器
type LinkedInCollector struct {
	name     string
	source   string
	interval time.Duration
	client   *http.Client
}

// NewLinkedInCollector 创建LinkedIn采集器
func NewLinkedInCollector() *LinkedInCollector {
	return &LinkedInCollector{
		name:     "linkedin_collector",
		interval: 12 * time.Hour,
		source:   "linkedin",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name 返回采集器名称
func (c *LinkedInCollector) Name() string {
	return c.name
}

// Source 返回数据源
func (c *LinkedInCollector) Source() string {
	return c.source
}

// Interval 返回采集间隔
func (c *LinkedInCollector) Interval() time.Duration {
	return c.interval
}

// IntelType 返回情报类型
func (c *LinkedInCollector) IntelType() core.IntelType {
	return core.IntelTypeCommunity
}

// Fetch 采集LinkedIn数据
func (c *LinkedInCollector) Fetch(ctx context.Context) ([]core.IntelItem, error) {
	var items []core.IntelItem

	// 1. 采集AI相关内容
	aiContent, err := c.fetchAIContent(ctx)
	if err == nil {
		items = append(items, aiContent...)
	}

	// 2. 采集创始人内容
	founderContent, err := c.fetchFounderContent(ctx)
	if err == nil {
		items = append(items, founderContent...)
	}

	// 3. 采集跨境合规内容
	complianceContent, err := c.fetchComplianceContent(ctx)
	if err == nil {
		items = append(items, complianceContent...)
	}

	return items, nil
}

// RateLimit 返回请求间隔
func (c *LinkedInCollector) RateLimit() time.Duration {
	return 3 * time.Second
}

// fetchAIContent 采集AI相关内容
func (c *LinkedInCollector) fetchAIContent(ctx context.Context) ([]core.IntelItem, error) {
	// 模拟LinkedIn AI相关内容
	sampleContent := []struct {
		Title     string
		Content   string
		Author    string
		Company   string
		Likes     int
		Comments  int
		Shares    int
		URL       string
		PostDate  string
		MediaType string
	}{{
		Title:     "The Future of AI API Management",
		Content:   "As AI adoption grows, managing API costs becomes critical for businesses...",
		Author:    "John Doe",
		Company:   "Tech Innovations Inc",
		Likes:     245,
		Comments:  32,
		Shares:    45,
		URL:       "https://www.linkedin.com/posts/johndoe_ai-api-management-activity-1234567890",
		PostDate:  "2026-03-30",
		MediaType: "article",
	}, {
		Title:     "How Token Bridge is Revolutionizing AI Cost Optimization",
		Content:   "Token Bridge's approach to AI API cost optimization is changing the game...",
		Author:    "Jane Smith",
		Company:   "Token Bridge",
		Likes:     189,
		Comments:  25,
		Shares:    32,
		URL:       "https://www.linkedin.com/posts/janesmith_token-bridge-ai-cost-optimization-activity-1234567891",
		PostDate:  "2026-03-29",
		MediaType: "video",
	}}

	var items []core.IntelItem
	for _, content := range sampleContent {
		item := core.NewIntelItem(core.IntelTypeCommunity, "linkedin")
		item.Title = fmt.Sprintf("LinkedIn: %s", content.Title)
		item.Content = content.Content
		item.URL = content.URL
		item.SourceID = content.Title

		// 设置元数据
		item.Metadata = core.Metadata{
			"platform":       "linkedin",
			"author":         content.Author,
			"company":        content.Company,
			"likes":          content.Likes,
			"comments_count": content.Comments,
			"shares":         content.Shares,
			"media_type":     content.MediaType,
			"content_type":   "ai_related",
		}

		// 解析发布时间
		if t, err := time.Parse("2006-01-02", content.PostDate); err == nil {
			item.PublishedAt = &t
		}

		items = append(items, item)
	}

	return items, nil
}

// fetchFounderContent 采集创始人内容
func (c *LinkedInCollector) fetchFounderContent(ctx context.Context) ([]core.IntelItem, error) {
	// 模拟创始人内容
	sampleContent := []struct {
		Founder  string
		Title    string
		Content  string
		Likes    int
		Comments int
		Shares   int
		URL      string
		PostDate string
	}{{
		Founder:  "Elon Musk",
		Title:    "Building the Future of AI Infrastructure",
		Content:  "Our approach to AI infrastructure is focused on scalability and cost efficiency...",
		Likes:    15000,
		Comments: 850,
		Shares:   2300,
		URL:      "https://www.linkedin.com/posts/elonmusk_ai-infrastructure-activity-1234567892",
		PostDate: "2026-03-30",
	}, {
		Founder:  "Sam Altman",
		Title:    "The Evolution of OpenAI's API Strategy",
		Content:  "We're constantly refining our API pricing and capabilities to better serve developers...",
		Likes:    8700,
		Comments: 420,
		Shares:   1200,
		URL:      "https://www.linkedin.com/posts/samaltman_openai-api-strategy-activity-1234567893",
		PostDate: "2026-03-28",
	}}

	var items []core.IntelItem
	for _, content := range sampleContent {
		item := core.NewIntelItem(core.IntelTypeCommunity, "linkedin")
		item.Title = fmt.Sprintf("LinkedIn: %s by %s", content.Title, content.Founder)
		item.Content = content.Content
		item.URL = content.URL
		item.SourceID = content.Title

		// 设置元数据
		item.Metadata = core.Metadata{
			"platform":       "linkedin",
			"founder":        content.Founder,
			"likes":          content.Likes,
			"comments_count": content.Comments,
			"shares":         content.Shares,
			"content_type":   "founder_insight",
		}

		// 解析发布时间
		if t, err := time.Parse("2006-01-02", content.PostDate); err == nil {
			item.PublishedAt = &t
		}

		items = append(items, item)
	}

	return items, nil
}

// fetchComplianceContent 采集跨境合规内容
func (c *LinkedInCollector) fetchComplianceContent(ctx context.Context) ([]core.IntelItem, error) {
	// 模拟跨境合规内容
	sampleContent := []struct {
		Title    string
		Content  string
		Author   string
		Company  string
		Likes    int
		Comments int
		URL      string
		PostDate string
	}{{
		Title:    "AI API Compliance in a Global Market",
		Content:  "Navigating data privacy regulations across different regions is crucial for AI API providers...",
		Author:   "Legal Expert",
		Company:  "Global Compliance Solutions",
		Likes:    120,
		Comments: 18,
		URL:      "https://www.linkedin.com/posts/legalexpert_ai-compliance-activity-1234567894",
		PostDate: "2026-03-29",
	}, {
		Title:    "Token Bridge's Approach to Cross-Border AI Compliance",
		Content:  "How Token Bridge ensures compliance with GDPR, CCPA, and other regional regulations...",
		Author:   "Legal Team",
		Company:  "Token Bridge",
		Likes:    89,
		Comments: 12,
		URL:      "https://www.linkedin.com/posts/legalteam_token-bridge-compliance-activity-1234567895",
		PostDate: "2026-03-27",
	}}

	var items []core.IntelItem
	for _, content := range sampleContent {
		item := core.NewIntelItem(core.IntelTypeCommunity, "linkedin")
		item.Title = fmt.Sprintf("LinkedIn: %s", content.Title)
		item.Content = content.Content
		item.URL = content.URL
		item.SourceID = content.Title

		// 设置元数据
		item.Metadata = core.Metadata{
			"platform":       "linkedin",
			"author":         content.Author,
			"company":        content.Company,
			"likes":          content.Likes,
			"comments_count": content.Comments,
			"content_type":   "compliance",
			"topic":          "cross_border",
		}

		// 解析发布时间
		if t, err := time.Parse("2006-01-02", content.PostDate); err == nil {
			item.PublishedAt = &t
		}

		items = append(items, item)
	}

	return items, nil
}
