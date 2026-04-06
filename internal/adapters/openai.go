// Package adapters 提供厂商适配器实现
package adapters

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// OpenAIAdapter OpenAI 价格适配器
type OpenAIAdapter struct {
	client  *http.Client
	apiKey  string
	baseURL string
}

// NewOpenAIAdapter 创建 OpenAI 适配器
// apiKey 可为空，为空时使用网页抓取模式
func NewOpenAIAdapter(apiKey string) *OpenAIAdapter {
	return &OpenAIAdapter{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey:  apiKey,
		baseURL: "https://openai.com/api/pricing",
	}
}

func (a *OpenAIAdapter) Name() string {
	return "openai"
}

func (a *OpenAIAdapter) DisplayName() string {
	return "OpenAI"
}

func (a *OpenAIAdapter) RateLimit() time.Duration {
	return 10 * time.Second
}

func (a *OpenAIAdapter) MaxRetries() int {
	return 3
}

func (a *OpenAIAdapter) IsRateLimited(resp *http.Response) bool {
	return resp.StatusCode == 429
}

func (a *OpenAIAdapter) RecommendedWindow() (int, int) {
	return 2, 6
}

// Fetch 抓取 OpenAI 价格
// 优先使用 API（如果有 key），否则使用网页抓取
func (a *OpenAIAdapter) Fetch(ctx context.Context) ([]ModelPrice, error) {
	capturedAt := time.Now().UTC()

	// 无论是否有 API Key，都使用网页抓取（更可靠）
	return a.fetchFromPricingPage(ctx, capturedAt)
}

// fetchFromPricingPage 从定价页面抓取
func (a *OpenAIAdapter) fetchFromPricingPage(ctx context.Context, capturedAt time.Time) ([]ModelPrice, error) {
	// OpenAI 定价页面
	urls := []string{
		"https://openai.com/api/pricing",
		"https://platform.openai.com/docs/pricing",
	}

	var allPrices []ModelPrice

	for _, url := range urls {
		req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")

		resp, err := a.client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			continue
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			continue
		}

		// 解析页面
		prices := a.parsePricingPage(doc, capturedAt)
		if len(prices) > 0 {
			allPrices = append(allPrices, prices...)
			break // 成功获取数据
		}
	}

	// 如果网页抓取失败，使用内置价格表
	if len(allPrices) == 0 {
		allPrices = a.getStaticPricing(capturedAt)
	}

	return allPrices, nil
}

// parsePricingPage 解析定价页面
func (a *OpenAIAdapter) parsePricingPage(doc *goquery.Document, capturedAt time.Time) []ModelPrice {
	var prices []ModelPrice

	// OpenAI 定价页面通常有模型卡片或表格
	// 尝试多种选择器模式

	// 模式 1: 查找模型卡片
	doc.Find("[class*='model'], [class*='pricing']").Each(func(i int, s *goquery.Selection) {
		// 提取模型名称
		modelName := a.extractModelName(s)
		if modelName == "" {
			return
		}

		// 提取价格
		inputPrice, outputPrice := a.extractPricesFromElement(s)

		if inputPrice > 0 || outputPrice > 0 {
			prices = append(prices, ModelPrice{
				Source:    fmt.Sprintf("openai-%s", capturedAt.Format("2006-01-02")),
				ModelCode: normalizeModelCode(modelName),
				ModelName: modelName,
				Vendor:    a.Name(),
				PricingRaw: PricingRaw{
					InputUSDPerMillion:  floatPtr(inputPrice),
					OutputUSDPerMillion: floatPtr(outputPrice),
					Currency:            "USD",
					PriceType:           "vendor_list_price",
					SchemaVersion:       "v1",
					CapturedAt:          capturedAt,
				},
			})
		}
	})

	return prices
}

// extractModelName 提取模型名称
func (a *OpenAIAdapter) extractModelName(s *goquery.Selection) string {
	// 尝试多种选择器
	selectors := []string{
		"h2", "h3", "h4",
		".model-name",
		"[class*='title']",
		"[class*='name']",
	}

	for _, sel := range selectors {
		name := strings.TrimSpace(s.Find(sel).First().Text())
		// 验证是否是有效的 OpenAI 模型名
		if a.isValidModelName(name) {
			return name
		}
	}

	return ""
}

// isValidModelName 验证是否是有效的 OpenAI 模型名
func (a *OpenAIAdapter) isValidModelName(name string) bool {
	if name == "" {
		return false
	}
	// OpenAI 模型名通常包含 gpt 或 o1
	lower := strings.ToLower(name)
	return strings.Contains(lower, "gpt") ||
		strings.Contains(lower, "o1") ||
		strings.Contains(lower, "o3") ||
		strings.Contains(lower, "davinci") ||
		strings.Contains(lower, "whisper") ||
		strings.Contains(lower, "embedding")
}

// extractPricesFromElement 从元素中提取价格
func (a *OpenAIAdapter) extractPricesFromElement(s *goquery.Selection) (input, output float64) {
	text := s.Text()

	// 使用正则表达式匹配价格
	// 模式: Input: $X.XX / 1M tokens, Output: $Y.YY / 1M tokens
	inputPattern := regexp.MustCompile(`[Ii]nput[:\s]*\$?([0-9.]+)`)
	outputPattern := regexp.MustCompile(`[Oo]utput[:\s]*\$?([0-9.]+)`)

	if matches := inputPattern.FindStringSubmatch(text); len(matches) > 1 {
		input = parseFloat(matches[1])
	}
	if matches := outputPattern.FindStringSubmatch(text); len(matches) > 1 {
		output = parseFloat(matches[1])
	}

	return input, output
}

// getStaticPricing 获取静态价格表（作为备用）
func (a *OpenAIAdapter) getStaticPricing(capturedAt time.Time) []ModelPrice {
	// 基于 OpenAI 官方定价（2026-03）
	priceTable := []struct {
		Name   string
		Input  float64
		Output float64
	}{
		{"GPT-4o", 2.50, 10.00},
		{"GPT-4o mini", 0.15, 0.60},
		{"GPT-4 Turbo", 10.00, 30.00},
		{"GPT-4", 30.00, 60.00},
		{"GPT-3.5 Turbo", 0.50, 1.50},
		{"o1", 15.00, 60.00},
		{"o1-mini", 3.00, 12.00},
		{"o3-mini", 1.10, 4.40},
		{"text-embedding-3-small", 0.02, 0.00},
		{"text-embedding-3-large", 0.13, 0.00},
		{"whisper", 0.006, 0.00}, // per minute
	}

	var prices []ModelPrice
	for _, p := range priceTable {
		prices = append(prices, ModelPrice{
			Source:    fmt.Sprintf("openai-%s", capturedAt.Format("2006-01-02")),
			ModelCode: normalizeModelCode(p.Name),
			ModelName: p.Name,
			Vendor:    a.Name(),
			PricingRaw: PricingRaw{
				InputUSDPerMillion:  floatPtr(p.Input),
				OutputUSDPerMillion: floatPtr(p.Output),
				Currency:            "USD",
				PriceType:           "vendor_list_price",
				SchemaVersion:       "v1",
				CapturedAt:          capturedAt,
			},
		})
	}

	return prices
}

func parseFloat(s string) float64 {
	var f float64
	_, _ = fmt.Sscanf(s, "%f", &f)
	return f
}
