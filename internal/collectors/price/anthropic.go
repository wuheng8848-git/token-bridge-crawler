// Package price 提供价格采集器
package price

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"token-bridge-crawler/internal/core"

	"github.com/PuerkitoBio/goquery"
)

// AnthropicCollector Anthropic价格采集器
type AnthropicCollector struct {
	*PriceCollector
}

// NewAnthropicCollector 创建Anthropic价格采集器
func NewAnthropicCollector(staticFile string) *AnthropicCollector {
	config := PriceCollectorConfig{
		WebURL:     "https://www.anthropic.com/pricing",
		StaticFile: staticFile,
		RateLimit:  5 * time.Second,
	}

	base := NewPriceCollector("anthropic", "Anthropic Claude", config)

	collector := &AnthropicCollector{
		PriceCollector: base,
	}

	// 设置 fallback 价格数据
	base.SetFallbackPrices(collector.GetFallbackPrices())

	// 重写web抓取策略
	collector.strategies[0].Fetch = collector.fetchFromWeb

	return collector
}

// fetchFromWeb 从Anthropic官网抓取价格
func (c *AnthropicCollector) fetchFromWeb(ctx context.Context) ([]core.IntelItem, error) {
	doc, err := fetchWebPage(c.config.WebURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}

	var prices []PriceData

	// 尝试多种选择器
	selectors := []string{
		"[class*='pricing'], [class*='card'], [class*='plan']",
		"table tbody tr",
		"[class*='model']",
	}

	for _, selector := range selectors {
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			price := c.parsePriceCard(s)
			if price != nil {
				prices = append(prices, *price)
			}
		})
		if len(prices) > 0 {
			break
		}
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("no valid prices parsed from page")
	}

	// 保存静态备份
	if err := c.saveStaticBackup(prices); err != nil {
		c.LogFetchResult("web", len(prices), fmt.Errorf("backup failed: %w", err))
	}

	return c.pricesToIntelItems(prices), nil
}

// parsePriceCard 解析价格卡片
func (c *AnthropicCollector) parsePriceCard(s *goquery.Selection) *PriceData {
	// 提取模型名称
	modelName := strings.TrimSpace(s.Find("h2, h3, [class*='title'], [class*='name']").Text())
	if modelName == "" {
		return nil
	}

	// 清理模型名称
	modelName = cleanModelName(modelName)
	modelCode := c.modelNameToCode(modelName)

	// 提取价格信息
	text := s.Text()

	// 解析输入价格（prompt）
	inputPrice := parseAnthropicPrice(text, "input", "prompt")

	// 解析输出价格（completion）
	outputPrice := parseAnthropicPrice(text, "output", "completion")

	if inputPrice <= 0 && outputPrice <= 0 {
		return nil
	}

	return &PriceData{
		ModelCode:   modelCode,
		ModelName:   modelName,
		InputPrice:  inputPrice,
		OutputPrice: outputPrice,
		Currency:    "USD",
	}
}

// modelNameToCode 将模型名称转换为代码
func (c *AnthropicCollector) modelNameToCode(name string) string {
	code := strings.ToLower(name)
	code = strings.ReplaceAll(code, " ", "-")
	code = strings.ReplaceAll(code, "(", "")
	code = strings.ReplaceAll(code, ")", "")

	// Anthropic模型映射
	mappings := map[string]string{
		"claude-4-6":      "claude-4.6",
		"claude-4-5":      "claude-4.5",
		"claude-4-0":      "claude-4.0",
		"claude-3-5":      "claude-3.5",
		"claude-3-0":      "claude-3.0",
		"claude-3-opus":   "claude-3-opus",
		"claude-3-sonnet": "claude-3-sonnet",
		"claude-3-haiku":  "claude-3-haiku",
		"claude-2":        "claude-2",
	}

	for key, value := range mappings {
		if strings.Contains(code, key) {
			return value
		}
	}

	return code
}

// parseAnthropicPrice 解析Anthropic价格
func parseAnthropicPrice(text, inputKey, outputKey string) float64 {
	// 匹配模式："$2.50 per 1M tokens" 或 "$0.015 per 1K tokens"
	patterns := []string{
		`\$([\d.]+)\s*per\s*([\d.]+)([KM])\s*tokens`,
		`([\d.]+)\s*\$?\s*per\s*([\d.]+)([KM])\s*tokens`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(text, -1)

		for _, match := range matches {
			if len(match) >= 4 {
				price, err := strconv.ParseFloat(match[1], 64)
				if err != nil {
					continue
				}

				_, err = strconv.ParseFloat(match[2], 64)
				if err != nil {
					continue
				}

				unit := match[3]

				// 转换为 per million tokens
				if unit == "K" {
					price *= 1000
				}
				// unit == "M" 已经是 million，无需转换

				return price
			}
		}
	}

	return 0
}

// GetFallbackPrices 获取备用价格
func (c *AnthropicCollector) GetFallbackPrices() []PriceData {
	return []PriceData{
		{ModelCode: "claude-4.6", ModelName: "Claude 4.6", InputPrice: 3.00, OutputPrice: 15.00, Currency: "USD"},
		{ModelCode: "claude-4.5", ModelName: "Claude 4.5", InputPrice: 1.50, OutputPrice: 7.50, Currency: "USD"},
		{ModelCode: "claude-4.0", ModelName: "Claude 4.0", InputPrice: 10.00, OutputPrice: 30.00, Currency: "USD"},
		{ModelCode: "claude-3.5-sonnet", ModelName: "Claude 3.5 Sonnet", InputPrice: 3.00, OutputPrice: 15.00, Currency: "USD"},
		{ModelCode: "claude-3.5-haiku", ModelName: "Claude 3.5 Haiku", InputPrice: 0.25, OutputPrice: 1.25, Currency: "USD"},
		{ModelCode: "claude-3-opus", ModelName: "Claude 3 Opus", InputPrice: 15.00, OutputPrice: 75.00, Currency: "USD"},
		{ModelCode: "claude-3-sonnet", ModelName: "Claude 3 Sonnet", InputPrice: 3.00, OutputPrice: 15.00, Currency: "USD"},
		{ModelCode: "claude-3-haiku", ModelName: "Claude 3 Haiku", InputPrice: 0.25, OutputPrice: 1.25, Currency: "USD"},
	}
}
