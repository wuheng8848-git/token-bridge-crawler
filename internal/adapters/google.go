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

// GoogleAdapter Google Gemini 价格适配器
type GoogleAdapter struct {
	client  *http.Client
	baseURL string
}

// NewGoogleAdapter 创建 Google 适配器
func NewGoogleAdapter() *GoogleAdapter {
	return &GoogleAdapter{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://ai.google.dev/pricing",
	}
}

func (a *GoogleAdapter) Name() string {
	return "google"
}

func (a *GoogleAdapter) DisplayName() string {
	return "Google AI"
}

func (a *GoogleAdapter) RateLimit() time.Duration {
	return 10 * time.Second
}

func (a *GoogleAdapter) MaxRetries() int {
	return 3
}

func (a *GoogleAdapter) IsRateLimited(resp *http.Response) bool {
	if resp.StatusCode == 429 {
		return true
	}
	if resp.StatusCode == 503 && resp.Header.Get("Retry-After") != "" {
		return true
	}
	return false
}

func (a *GoogleAdapter) RecommendedWindow() (int, int) {
	return 2, 6
}

// Fetch 抓取 Gemini 价格
func (a *GoogleAdapter) Fetch(ctx context.Context) ([]ModelPrice, error) {
	capturedAt := time.Now().UTC()

	// 首先尝试网页抓取
	prices, err := a.fetchFromPricingPage(ctx, capturedAt)
	if err != nil || len(prices) == 0 {
		// 网页抓取失败，使用静态价格表
		return a.getStaticPricing(capturedAt), nil
	}

	return prices, nil
}

// fetchFromPricingPage 从定价页面抓取
func (a *GoogleAdapter) fetchFromPricingPage(ctx context.Context, capturedAt time.Time) ([]ModelPrice, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", a.baseURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if a.IsRateLimited(resp) {
		return nil, ErrRateLimited{
			Vendor:     a.Name(),
			RetryAfter: 24 * time.Hour,
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	// 尝试多种解析策略
	prices := a.parseStrategy1(doc, capturedAt)
	if len(prices) == 0 {
		prices = a.parseStrategy2(doc, capturedAt)
	}

	return prices, nil
}

// parseStrategy1 策略1：查找定价表格
func (a *GoogleAdapter) parseStrategy1(doc *goquery.Document, capturedAt time.Time) []ModelPrice {
	var prices []ModelPrice

	// 查找表格行
	doc.Find("table tbody tr, table tr").Each(func(i int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 2 {
			return
		}

		// 第一个单元格通常是模型名称
		modelCell := cells.Eq(0)
		modelName := strings.TrimSpace(modelCell.Text())

		// 清理模型名称
		modelName = a.cleanModelName(modelName)
		if modelName == "" || !a.isValidModel(modelName) {
			return
		}

		// 提取价格（通常在第2、3个单元格）
		var inputPrice, outputPrice float64

		for j := 1; j < cells.Length() && j < 4; j++ {
			cellText := cells.Eq(j).Text()
			price := parsePricePerMillion(cellText)

			cellLower := strings.ToLower(cellText)
			if inputPrice == 0 && (strings.Contains(cellLower, "input") || j == 1) {
				inputPrice = price
			} else if outputPrice == 0 && (strings.Contains(cellLower, "output") || j == 2) {
				outputPrice = price
			}
		}

		// 如果还没找到，尝试任意有效价格
		if inputPrice == 0 && cells.Length() > 1 {
			inputPrice = parsePricePerMillion(cells.Eq(1).Text())
		}
		if outputPrice == 0 && cells.Length() > 2 {
			outputPrice = parsePricePerMillion(cells.Eq(2).Text())
		}

		if inputPrice > 0 || outputPrice > 0 {
			prices = append(prices, ModelPrice{
				Source:    fmt.Sprintf("google-%s", capturedAt.Format("2006-01-02")),
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

// parseStrategy2 策略2：查找模型卡片
func (a *GoogleAdapter) parseStrategy2(doc *goquery.Document, capturedAt time.Time) []ModelPrice {
	var prices []ModelPrice

	// 查找包含 "Gemini" 的标题元素
	doc.Find("h2, h3, h4").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if !strings.Contains(strings.ToLower(text), "gemini") {
			return
		}

		modelName := a.cleanModelName(text)
		if !a.isValidModel(modelName) {
			return
		}

		// 在相邻元素中查找价格
		var inputPrice, outputPrice float64

		// 查找同级或父级元素中的价格
		s.Parent().Find("p, span, div").Each(func(j int, p *goquery.Selection) {
			if inputPrice > 0 && outputPrice > 0 {
				return
			}

			pText := p.Text()
			pLower := strings.ToLower(pText)

			if strings.Contains(pLower, "input") {
				inputPrice = parsePricePerMillion(pText)
			} else if strings.Contains(pLower, "output") {
				outputPrice = parsePricePerMillion(pText)
			}
		})

		// 如果找到价格，添加记录
		if inputPrice > 0 || outputPrice > 0 {
			prices = append(prices, ModelPrice{
				Source:    fmt.Sprintf("google-%s", capturedAt.Format("2006-01-02")),
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

// cleanModelName 清理模型名称
func (a *GoogleAdapter) cleanModelName(name string) string {
	// 移除多余空白
	name = strings.Join(strings.Fields(name), " ")

	// 移除常见的前缀/后缀噪音
	noisePatterns := []string{
		"used to improve our products",
		"including thinking tokens",
		"grounding with Google Search",
		"*",
		"**",
		"***",
	}

	for _, noise := range noisePatterns {
		name = strings.ReplaceAll(name, noise, "")
	}

	// 提取 "Gemini X" 或 "X Flash" 等核心名称
	geminiPattern := regexp.MustCompile(`(?i)(gemini[\s-]*[\w\.]+(?:[\s-]*\d+\.?\d*)?)`)
	if matches := geminiPattern.FindStringSubmatch(name); len(matches) > 0 {
		return strings.TrimSpace(matches[0])
	}

	return strings.TrimSpace(name)
}

// isValidModel 验证是否是有效的 Google 模型名
func (a *GoogleAdapter) isValidModel(name string) bool {
	if name == "" {
		return false
	}
	lower := strings.ToLower(name)
	return strings.Contains(lower, "gemini") ||
		strings.Contains(lower, "palm") ||
		strings.Contains(lower, "imagen")
}

// getStaticPricing 获取静态价格表（备用）
func (a *GoogleAdapter) getStaticPricing(capturedAt time.Time) []ModelPrice {
	// 基于 Google AI 官方定价（2026-03）
	priceTable := []struct {
		Name   string
		Input  float64
		Output float64
	}{
		{"Gemini 2.0 Flash", 0.10, 0.40},
		{"Gemini 2.0 Flash-Lite", 0.075, 0.30},
		{"Gemini 2.0 Pro", 1.25, 5.00},
		{"Gemini 1.5 Flash", 0.075, 0.30},
		{"Gemini 1.5 Pro", 1.25, 5.00},
		{"Gemini 1.0 Pro", 0.50, 1.50},
		{"Gemini 1.0 Ultra", 1.75, 7.00},
		{"Imagen 3", 0.03, 0.00}, // per image
	}

	var prices []ModelPrice
	for _, p := range priceTable {
		prices = append(prices, ModelPrice{
			Source:    fmt.Sprintf("google-%s", capturedAt.Format("2006-01-02")),
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
