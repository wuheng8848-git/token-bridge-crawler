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

// OpenAICollector OpenAI价格采集器
type OpenAICollector struct {
	*PriceCollector
}

// NewOpenAICollector 创建OpenAI价格采集器
func NewOpenAICollector(staticFile string) *OpenAICollector {
	config := PriceCollectorConfig{
		WebURL:     "https://openai.com/api/pricing/",
		StaticFile: staticFile,
		RateLimit:  5 * time.Second,
	}

	base := NewPriceCollector("openai", "OpenAI", config)

	collector := &OpenAICollector{
		PriceCollector: base,
	}

	// 设置 fallback 价格数据
	base.SetFallbackPrices(collector.GetFallbackPrices())

	// 重写web抓取策略
	collector.strategies[0].Fetch = collector.fetchFromWeb

	return collector
}

// fetchFromWeb 从OpenAI官网抓取价格
func (c *OpenAICollector) fetchFromWeb(ctx context.Context) ([]core.IntelItem, error) {
	doc, err := fetchWebPage(c.config.WebURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}

	var prices []PriceData

	// 尝试多种选择器（页面结构可能变化）
	selectors := []string{
		"[data-testid='pricing-table'] tbody tr",
		".pricing-table tbody tr",
		"table tbody tr",
		"[class*='pricing'] tr",
		"[class*='price'] tr",
	}

	var rows *goquery.Selection
	for _, selector := range selectors {
		rows = doc.Find(selector)
		if rows.Length() > 0 {
			break
		}
	}

	if rows.Length() == 0 {
		return nil, fmt.Errorf("no pricing rows found with any selector")
	}

	rows.Each(func(i int, s *goquery.Selection) {
		price := c.parsePriceRow(s)
		if price != nil {
			prices = append(prices, *price)
		}
	})

	if len(prices) == 0 {
		return nil, fmt.Errorf("no valid prices parsed from page")
	}

	// 保存静态备份
	if err := c.saveStaticBackup(prices); err != nil {
		// 备份失败不影响主流程
		c.LogFetchResult("web", len(prices), fmt.Errorf("backup failed: %w", err))
	}

	return c.pricesToIntelItems(prices), nil
}

// parsePriceRow 解析价格行
func (c *OpenAICollector) parsePriceRow(s *goquery.Selection) *PriceData {
	// 提取模型名称
	modelName := strings.TrimSpace(s.Find("td:first-child, th:first-child").Text())
	if modelName == "" {
		return nil
	}

	// 清理模型名称
	modelName = cleanModelName(modelName)
	modelCode := modelNameToCode(modelName)

	// 提取价格单元格
	cells := s.Find("td")
	if cells.Length() < 2 {
		return nil
	}

	// 解析输入价格
	inputPriceText := cells.Eq(1).Text()
	inputPrice := parsePrice(inputPriceText)

	// 解析输出价格（如果有）
	outputPrice := inputPrice
	if cells.Length() >= 3 {
		outputPriceText := cells.Eq(2).Text()
		outputPrice = parsePrice(outputPriceText)
	}

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

// cleanModelName 清理模型名称
func cleanModelName(name string) string {
	// 移除多余空白
	name = strings.Join(strings.Fields(name), " ")

	// 移除常见前缀/后缀
	name = strings.TrimPrefix(name, "Model")
	name = strings.TrimSpace(name)

	return name
}

// modelNameToCode 将模型名称转换为代码
func modelNameToCode(name string) string {
	// 转换为小写
	code := strings.ToLower(name)

	// 替换空格和特殊字符
	code = strings.ReplaceAll(code, " ", "-")
	code = strings.ReplaceAll(code, "(", "")
	code = strings.ReplaceAll(code, ")", "")
	code = strings.ReplaceAll(code, ".", "-")

	// 常见模型映射
	mappings := map[string]string{
		"gpt-4o":           "gpt-4o",
		"gpt-4o-mini":      "gpt-4o-mini",
		"gpt-4-turbo":      "gpt-4-turbo",
		"gpt-4":            "gpt-4",
		"gpt-35-turbo":     "gpt-3.5-turbo",
		"gpt-3-5-turbo":    "gpt-3.5-turbo",
		"text-embedding-3": "text-embedding-3",
		"dall-e-3":         "dall-e-3",
		"whisper":          "whisper",
		"tts":              "tts",
	}

	// 尝试匹配
	for key, value := range mappings {
		if strings.Contains(code, key) {
			return value
		}
	}

	return code
}

// parsePrice 解析价格文本
func parsePrice(text string) float64 {
	// 提取数字
	re := regexp.MustCompile(`[\d.]+`)
	matches := re.FindAllString(text, -1)

	if len(matches) == 0 {
		return 0
	}

	// 尝试解析第一个数字
	price, err := strconv.ParseFloat(matches[0], 64)
	if err != nil {
		return 0
	}

	// 检查单位
	textLower := strings.ToLower(text)

	// 如果是 per 1K tokens，转换为 per million
	if strings.Contains(textLower, "1k") || strings.Contains(textLower, "1,000") {
		price *= 1000
	}

	// 如果是 per token，转换为 per million
	if strings.Contains(textLower, "/token") && !strings.Contains(textLower, "1k") && !strings.Contains(textLower, "million") {
		price *= 1000000
	}

	return price
}

// GetFallbackPrices 获取备用价格（当所有策略都失败时使用）
func (c *OpenAICollector) GetFallbackPrices() []PriceData {
	// 2026年3月的基准价格
	return []PriceData{
		{ModelCode: "gpt-4o", ModelName: "GPT-4o", InputPrice: 2.50, OutputPrice: 10.00, Currency: "USD"},
		{ModelCode: "gpt-4o-mini", ModelName: "GPT-4o Mini", InputPrice: 0.15, OutputPrice: 0.60, Currency: "USD"},
		{ModelCode: "gpt-4-turbo", ModelName: "GPT-4 Turbo", InputPrice: 10.00, OutputPrice: 30.00, Currency: "USD"},
		{ModelCode: "gpt-3.5-turbo", ModelName: "GPT-3.5 Turbo", InputPrice: 0.50, OutputPrice: 1.50, Currency: "USD"},
		{ModelCode: "text-embedding-3-small", ModelName: "Text Embedding 3 Small", InputPrice: 0.02, OutputPrice: 0.00, Currency: "USD"},
		{ModelCode: "text-embedding-3-large", ModelName: "Text Embedding 3 Large", InputPrice: 0.13, OutputPrice: 0.00, Currency: "USD"},
		{ModelCode: "dall-e-3", ModelName: "DALL-E 3", InputPrice: 0.04, OutputPrice: 0.00, Currency: "USD"},
		{ModelCode: "whisper", ModelName: "Whisper", InputPrice: 0.006, OutputPrice: 0.00, Currency: "USD"},
		{ModelCode: "tts", ModelName: "TTS", InputPrice: 0.015, OutputPrice: 0.00, Currency: "USD"},
		{ModelCode: "tts-hd", ModelName: "TTS HD", InputPrice: 0.030, OutputPrice: 0.00, Currency: "USD"},
	}
}
