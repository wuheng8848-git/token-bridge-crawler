// Package price 提供价格采集器
package price

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"token-bridge-crawler/internal/core"
)

// GoogleCollector Google价格采集器
type GoogleCollector struct {
	*PriceCollector
}

// NewGoogleCollector 创建Google价格采集器
func NewGoogleCollector(staticFile string) *GoogleCollector {
	config := PriceCollectorConfig{
		WebURL:     "https://ai.google.dev/pricing",
		StaticFile: staticFile,
		RateLimit:  5 * time.Second,
	}

	base := NewPriceCollector("google", "Google Gemini", config)

	collector := &GoogleCollector{
		PriceCollector: base,
	}

	// 设置 fallback 价格数据
	base.SetFallbackPrices(collector.GetFallbackPrices())

	// 重写web抓取策略
	collector.strategies[0].Fetch = collector.fetchFromWeb

	return collector
}

// fetchFromWeb 从Google官网抓取价格
func (c *GoogleCollector) fetchFromWeb(ctx context.Context) ([]core.IntelItem, error) {
	doc, err := fetchWebPage(c.config.WebURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}

	var prices []PriceData

	// Google页面结构：通常是表格或卡片
	selectors := []string{
		"table tbody tr",
		"[class*='pricing'] tr",
		"[class*='price'] tr",
		"[class*='model'] [class*='price']",
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
		c.LogFetchResult("web", len(prices), fmt.Errorf("backup failed: %w", err))
	}

	return c.pricesToIntelItems(prices), nil
}

// parsePriceRow 解析价格行
func (c *GoogleCollector) parsePriceRow(s *goquery.Selection) *PriceData {
	// 提取模型名称
	modelName := strings.TrimSpace(s.Find("td:first-child, th:first-child, .model-name").Text())
	if modelName == "" {
		return nil
	}

	// 清理模型名称
	modelName = cleanModelName(modelName)
	modelCode := c.modelNameToCode(modelName)

	// 提取价格
	var inputPrice, outputPrice float64

	// 尝试从文本中提取价格
	text := s.Text()

	// 查找输入价格模式
	inputMatch := regexp.MustCompile(`(?i)input[:\s]*\$?([\d.]+)`).FindStringSubmatch(text)
	if len(inputMatch) > 1 {
		inputPrice, _ = strconv.ParseFloat(inputMatch[1], 64)
	}

	// 查找输出价格模式
	outputMatch := regexp.MustCompile(`(?i)output[:\s]*\$?([\d.]+)`).FindStringSubmatch(text)
	if len(outputMatch) > 1 {
		outputPrice, _ = strconv.ParseFloat(outputMatch[1], 64)
	}

	// 如果没找到，尝试通用价格模式
	if inputPrice == 0 && outputPrice == 0 {
		prices := regexp.MustCompile(`\$([\d.]+)`).FindAllStringSubmatch(text, -1)
		if len(prices) >= 1 {
			inputPrice, _ = strconv.ParseFloat(prices[0][1], 64)
		}
		if len(prices) >= 2 {
			outputPrice, _ = strconv.ParseFloat(prices[1][1], 64)
		}
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

// modelNameToCode 将模型名称转换为代码
func (c *GoogleCollector) modelNameToCode(name string) string {
	code := strings.ToLower(name)
	code = strings.ReplaceAll(code, " ", "-")
	code = strings.ReplaceAll(code, "(", "")
	code = strings.ReplaceAll(code, ")", "")

	// Google模型映射
	mappings := map[string]string{
		"gemini-2-5-pro":  "gemini-2.5-pro",
		"gemini-2-5-flash": "gemini-2.5-flash",
		"gemini-2-0-pro":  "gemini-2.0-pro",
		"gemini-2-0-flash": "gemini-2.0-flash",
		"gemini-1-5-pro":  "gemini-1.5-pro",
		"gemini-1-5-flash": "gemini-1.5-flash",
		"gemini-pro":      "gemini-pro",
		"gemini-ultra":    "gemini-ultra",
	}

	for key, value := range mappings {
		if strings.Contains(code, key) {
			return value
		}
	}

	return code
}

// GetFallbackPrices 获取备用价格
func (c *GoogleCollector) GetFallbackPrices() []PriceData {
	return []PriceData{
		{ModelCode: "gemini-2.5-pro", ModelName: "Gemini 2.5 Pro", InputPrice: 1.25, OutputPrice: 10.00, Currency: "USD"},
		{ModelCode: "gemini-2.5-flash", ModelName: "Gemini 2.5 Flash", InputPrice: 0.15, OutputPrice: 0.60, Currency: "USD"},
		{ModelCode: "gemini-2.0-pro", ModelName: "Gemini 2.0 Pro", InputPrice: 1.25, OutputPrice: 10.00, Currency: "USD"},
		{ModelCode: "gemini-2.0-flash", ModelName: "Gemini 2.0 Flash", InputPrice: 0.075, OutputPrice: 0.30, Currency: "USD"},
		{ModelCode: "gemini-1.5-pro", ModelName: "Gemini 1.5 Pro", InputPrice: 3.50, OutputPrice: 10.50, Currency: "USD"},
		{ModelCode: "gemini-1.5-flash", ModelName: "Gemini 1.5 Flash", InputPrice: 0.075, OutputPrice: 0.30, Currency: "USD"},
	}
}
