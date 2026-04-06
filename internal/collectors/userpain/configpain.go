// Package userpain 提供用户痛点情报采集功能
package userpain

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"token-bridge-crawler/internal/core"
)

// ConfigPainCollector 配置痛点采集器
type ConfigPainCollector struct {
	name     string
	source   string
	interval time.Duration
	client   *http.Client
}

// NewConfigPainCollector 创建配置痛点采集器
func NewConfigPainCollector() *ConfigPainCollector {
	return &ConfigPainCollector{
		name:     "config_pain_collector",
		interval: 4 * time.Hour,
		source:   "config_pain",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name 返回采集器名称
func (c *ConfigPainCollector) Name() string {
	return c.name
}

// Source 返回数据源
func (c *ConfigPainCollector) Source() string {
	return c.source
}

// Interval 返回采集间隔
func (c *ConfigPainCollector) Interval() time.Duration {
	return c.interval
}

// IntelType 返回情报类型
func (c *ConfigPainCollector) IntelType() core.IntelType {
	return core.IntelTypeUserPain
}

// Fetch 采集配置痛点数据
func (c *ConfigPainCollector) Fetch(ctx context.Context) ([]core.IntelItem, error) {
	var items []core.IntelItem

	// 1. 采集订阅相关痛点
	subscriptionPains, err := c.fetchSubscriptionPains(ctx)
	if err == nil {
		items = append(items, subscriptionPains...)
	}

	// 2. 采集API配置痛点
	apiConfigPains, err := c.fetchAPIConfigPains(ctx)
	if err == nil {
		items = append(items, apiConfigPains...)
	}

	// 3. 采集MCP和工具配置痛点
	mcpPains, err := c.fetchMCPPains(ctx)
	if err == nil {
		items = append(items, mcpPains...)
	}

	// 4. 采集多工具并存痛点
	multiToolPains, err := c.fetchMultiToolPains(ctx)
	if err == nil {
		items = append(items, multiToolPains...)
	}

	return items, nil
}

// RateLimit 返回请求间隔
func (c *ConfigPainCollector) RateLimit() time.Duration {
	return 2 * time.Second
}

// fetchSubscriptionPains 采集订阅相关痛点
func (c *ConfigPainCollector) fetchSubscriptionPains(ctx context.Context) ([]core.IntelItem, error) {
	// 模拟订阅相关痛点
	samplePains := []struct {
		Title           string
		Content         string
		Platform        string
		Author          string
		Interactions    int
		URL             string
		PostDate        string
		PainType        string
		PainIntensity   int
		ConversionValue string
		SolutionFit     string
		MarketingAngle  string
		TargetAudience  string
	}{
		{
			Title:           "OpenAI subscription not enough for production",
			Content:         "My OpenAI subscription is constantly running out. I need a better way to manage my usage and costs...",
			Platform:        "Reddit",
			Author:          "DevOpsGuy",
			Interactions:    45,
			URL:             "https://reddit.com/r/OpenAI/comments/123456/subscription_issues",
			PostDate:        "2026-03-30",
			PainType:        "subscription",
			PainIntensity:   8,
			ConversionValue: "high",
			SolutionFit:     "excellent",
			MarketingAngle:  "透明计费 + 统一策略层",
			TargetAudience:  "生产环境开发者",
		},
		{
			Title:           "Wasting money on unused AI API credits",
			Content:         "I'm paying for a higher tier than I need. Is there a way to optimize my subscription based on actual usage?",
			Platform:        "HackerNews",
			Author:          "CostOptimizer",
			Interactions:    32,
			URL:             "https://news.ycombinator.com/item?id=123456",
			PostDate:        "2026-03-29",
			PainType:        "subscription_waste",
			PainIntensity:   7,
			ConversionValue: "medium",
			SolutionFit:     "very_good",
			MarketingAngle:  "降低 token 浪费 + 透明计费",
			TargetAudience:  "成本敏感型开发者",
		},
		{
			Title:           "Need flexible AI API billing",
			Content:         "My usage fluctuates a lot. Fixed subscriptions are not working for me...",
			Platform:        "Discord",
			Author:          "StartupDev",
			Interactions:    28,
			URL:             "https://discord.com/channels/123456/789012/345678",
			PostDate:        "2026-03-28",
			PainType:        "billing_flexibility",
			PainIntensity:   6,
			ConversionValue: "medium",
			SolutionFit:     "good",
			MarketingAngle:  "透明计费 + 快速接入",
			TargetAudience:  "创业团队",
		},
	}

	var items []core.IntelItem
	for _, pain := range samplePains {
		item := core.NewIntelItem(core.IntelTypeUserPain, "config_pain")
		item.Title = fmt.Sprintf("Config Pain: %s", pain.Title)
		item.Content = pain.Content
		item.URL = pain.URL
		item.SourceID = pain.Title

		// 设置元数据
		item.Metadata = core.Metadata{
			"platform":          pain.Platform,
			"author":            pain.Author,
			"interactions":      pain.Interactions,
			"pain_type":         pain.PainType,
			"category":          "subscription",
			"content_type":      "config_pain",
			"sentiment":         "negative",
			"pain_intensity":    pain.PainIntensity,
			"conversion_value":  pain.ConversionValue,
			"solution_fit":      pain.SolutionFit,
			"marketing_angle":   pain.MarketingAngle,
			"target_audience":   pain.TargetAudience,
			"value_proposition": "透明计费",
		}

		// 解析发布时间
		if t, err := time.Parse("2006-01-02", pain.PostDate); err == nil {
			item.PublishedAt = &t
		}

		items = append(items, item)
	}

	return items, nil
}

// fetchAPIConfigPains 采集API配置痛点
func (c *ConfigPainCollector) fetchAPIConfigPains(ctx context.Context) ([]core.IntelItem, error) {
	// 模拟API配置痛点
	samplePains := []struct {
		Title           string
		Content         string
		Platform        string
		Author          string
		Interactions    int
		URL             string
		PostDate        string
		PainType        string
		PainIntensity   int
		ConversionValue string
		SolutionFit     string
		MarketingAngle  string
		TargetAudience  string
	}{
		{
			Title:           "Third-party API configuration is a nightmare",
			Content:         "Setting up multiple AI APIs is so time-consuming. Each has different authentication methods and endpoints...",
			Platform:        "Reddit",
			Author:          "FullStackDev",
			Interactions:    56,
			URL:             "https://reddit.com/r/API/comments/234567/config_hell",
			PostDate:        "2026-03-30",
			PainType:        "api_config",
			PainIntensity:   9,
			ConversionValue: "high",
			SolutionFit:     "excellent",
			MarketingAngle:  "快速接入 + 少踩配置坑",
			TargetAudience:  "全栈开发者",
		},
		{
			Title:           "Need OpenAI compatible configuration",
			Content:         "I want to switch between providers but the configuration differences are too big...",
			Platform:        "HackerNews",
			Author:          "DevOpsLead",
			Interactions:    41,
			URL:             "https://news.ycombinator.com/item?id=234567",
			PostDate:        "2026-03-29",
			PainType:        "openai_compatible",
			PainIntensity:   8,
			ConversionValue: "high",
			SolutionFit:     "excellent",
			MarketingAngle:  "统一策略层 + 快速接入",
			TargetAudience:  "DevOps 负责人",
		},
		{
			Title:           "Provider configuration management",
			Content:         "Managing API keys and configurations for multiple providers is a mess...",
			Platform:        "Discord",
			Author:          "StartupCTO",
			Interactions:    33,
			URL:             "https://discord.com/channels/234567/890123/456789",
			PostDate:        "2026-03-27",
			PainType:        "provider_management",
			PainIntensity:   7,
			ConversionValue: "medium",
			SolutionFit:     "very_good",
			MarketingAngle:  "少踩配置坑 + 统一策略层",
			TargetAudience:  "创业公司CTO",
		},
	}

	var items []core.IntelItem
	for _, pain := range samplePains {
		item := core.NewIntelItem(core.IntelTypeUserPain, "config_pain")
		item.Title = fmt.Sprintf("Config Pain: %s", pain.Title)
		item.Content = pain.Content
		item.URL = pain.URL
		item.SourceID = pain.Title

		// 设置元数据
		item.Metadata = core.Metadata{
			"platform":          pain.Platform,
			"author":            pain.Author,
			"interactions":      pain.Interactions,
			"pain_type":         pain.PainType,
			"category":          "api_config",
			"content_type":      "config_pain",
			"sentiment":         "negative",
			"pain_intensity":    pain.PainIntensity,
			"conversion_value":  pain.ConversionValue,
			"solution_fit":      pain.SolutionFit,
			"marketing_angle":   pain.MarketingAngle,
			"target_audience":   pain.TargetAudience,
			"value_proposition": "快速接入",
		}

		// 解析发布时间
		if t, err := time.Parse("2006-01-02", pain.PostDate); err == nil {
			item.PublishedAt = &t
		}

		items = append(items, item)
	}

	return items, nil
}

// fetchMCPPains 采集MCP和工具配置痛点
func (c *ConfigPainCollector) fetchMCPPains(ctx context.Context) ([]core.IntelItem, error) {
	// 模拟MCP和工具配置痛点
	samplePains := []struct {
		Title           string
		Content         string
		Platform        string
		Author          string
		Interactions    int
		URL             string
		PostDate        string
		PainType        string
		PainIntensity   int
		ConversionValue string
		SolutionFit     string
		MarketingAngle  string
		TargetAudience  string
	}{
		{
			Title:           "MCP is burning through tokens",
			Content:         "My Model Context Protocol configuration is using way more tokens than expected...",
			Platform:        "Reddit",
			Author:          "AIResearcher",
			Interactions:    48,
			URL:             "https://reddit.com/r/MachineLearning/comments/345678/mcp_token_waste",
			PostDate:        "2026-03-30",
			PainType:        "mcp_token_waste",
			PainIntensity:   8,
			ConversionValue: "high",
			SolutionFit:     "excellent",
			MarketingAngle:  "降低 token 浪费 + 少踩配置坑",
			TargetAudience:  "AI 研究人员",
		},
		{
			Title:           "Skills configuration is complex",
			Content:         "Setting up skills for AI models is too complicated and error-prone...",
			Platform:        "HackerNews",
			Author:          "SkillDev",
			Interactions:    35,
			URL:             "https://news.ycombinator.com/item?id=345678",
			PostDate:        "2026-03-28",
			PainType:        "skills_config",
			PainIntensity:   7,
			ConversionValue: "medium",
			SolutionFit:     "very_good",
			MarketingAngle:  "少踩配置坑 + 快速接入",
			TargetAudience:  "技能开发者",
		},
		{
			Title:           "Tool config is costing me money",
			Content:         "My tool configurations are causing unnecessary token usage...",
			Platform:        "Discord",
			Author:          "ToolBuilder",
			Interactions:    29,
			URL:             "https://discord.com/channels/345678/901234/567890",
			PostDate:        "2026-03-26",
			PainType:        "tool_config_cost",
			PainIntensity:   8,
			ConversionValue: "high",
			SolutionFit:     "excellent",
			MarketingAngle:  "降低 token 浪费 + 透明计费",
			TargetAudience:  "工具构建者",
		},
	}

	var items []core.IntelItem
	for _, pain := range samplePains {
		item := core.NewIntelItem(core.IntelTypeUserPain, "config_pain")
		item.Title = fmt.Sprintf("Config Pain: %s", pain.Title)
		item.Content = pain.Content
		item.URL = pain.URL
		item.SourceID = pain.Title

		// 设置元数据
		item.Metadata = core.Metadata{
			"platform":          pain.Platform,
			"author":            pain.Author,
			"interactions":      pain.Interactions,
			"pain_type":         pain.PainType,
			"category":          "mcp_tool",
			"content_type":      "config_pain",
			"sentiment":         "negative",
			"pain_intensity":    pain.PainIntensity,
			"conversion_value":  pain.ConversionValue,
			"solution_fit":      pain.SolutionFit,
			"marketing_angle":   pain.MarketingAngle,
			"target_audience":   pain.TargetAudience,
			"value_proposition": "降低 token 浪费",
		}

		// 解析发布时间
		if t, err := time.Parse("2006-01-02", pain.PostDate); err == nil {
			item.PublishedAt = &t
		}

		items = append(items, item)
	}

	return items, nil
}

// fetchMultiToolPains 采集多工具并存痛点
func (c *ConfigPainCollector) fetchMultiToolPains(ctx context.Context) ([]core.IntelItem, error) {
	// 模拟多工具并存痛点
	samplePains := []struct {
		Title           string
		Content         string
		Platform        string
		Author          string
		Interactions    int
		URL             string
		PostDate        string
		PainType        string
		PainIntensity   int
		ConversionValue string
		SolutionFit     string
		MarketingAngle  string
		TargetAudience  string
	}{
		{
			Title:           "Multiple AI tools causing cost chaos",
			Content:         "I'm using multiple AI tools and can't keep track of costs. Need a unified way to manage...",
			Platform:        "Reddit",
			Author:          "AgencyOwner",
			Interactions:    52,
			URL:             "https://reddit.com/r/Entrepreneur/comments/456789/cost_chaos",
			PostDate:        "2026-03-30",
			PainType:        "multi_tool_cost",
			PainIntensity:   9,
			ConversionValue: "high",
			SolutionFit:     "excellent",
			MarketingAngle:  "统一策略层 + 透明计费",
			TargetAudience:  " agency 所有者",
		},
		{
			Title:           "Strategy confusion with multiple AI providers",
			Content:         "I have different providers for different tasks. Need a better way to manage strategy...",
			Platform:        "HackerNews",
			Author:          "CTO",
			Interactions:    43,
			URL:             "https://news.ycombinator.com/item?id=456789",
			PostDate:        "2026-03-29",
			PainType:        "strategy_confusion",
			PainIntensity:   8,
			ConversionValue: "high",
			SolutionFit:     "excellent",
			MarketingAngle:  "统一策略层 + 少踩配置坑",
			TargetAudience:  "企业CTO",
		},
		{
			Title:           "Unified AI tool management",
			Content:         "Managing multiple AI tools is a nightmare. Need a single interface...",
			Platform:        "Discord",
			Author:          "ProductManager",
			Interactions:    38,
			URL:             "https://discord.com/channels/456789/012345/678901",
			PostDate:        "2026-03-27",
			PainType:        "unified_management",
			PainIntensity:   7,
			ConversionValue: "medium",
			SolutionFit:     "very_good",
			MarketingAngle:  "统一策略层 + 快速接入",
			TargetAudience:  "产品经理",
		},
	}

	var items []core.IntelItem
	for _, pain := range samplePains {
		item := core.NewIntelItem(core.IntelTypeUserPain, "config_pain")
		item.Title = fmt.Sprintf("Config Pain: %s", pain.Title)
		item.Content = pain.Content
		item.URL = pain.URL
		item.SourceID = pain.Title

		// 设置元数据
		item.Metadata = core.Metadata{
			"platform":          pain.Platform,
			"author":            pain.Author,
			"interactions":      pain.Interactions,
			"pain_type":         pain.PainType,
			"category":          "multi_tool",
			"content_type":      "config_pain",
			"sentiment":         "negative",
			"pain_intensity":    pain.PainIntensity,
			"conversion_value":  pain.ConversionValue,
			"solution_fit":      pain.SolutionFit,
			"marketing_angle":   pain.MarketingAngle,
			"target_audience":   pain.TargetAudience,
			"value_proposition": "统一策略层",
		}

		// 解析发布时间
		if t, err := time.Parse("2006-01-02", pain.PostDate); err == nil {
			item.PublishedAt = &t
		}

		items = append(items, item)
	}

	return items, nil
}
