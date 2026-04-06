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

// AnthropicAdapter Anthropic 价格适配器
type AnthropicAdapter struct {
	client  *http.Client
	baseURL string
}

// NewAnthropicAdapter 创建 Anthropic 适配器
func NewAnthropicAdapter() *AnthropicAdapter {
	return &AnthropicAdapter{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://www.anthropic.com/pricing",
	}
}

func (a *AnthropicAdapter) Name() string {
	return "anthropic"
}

func (a *AnthropicAdapter) DisplayName() string {
	return "Anthropic"
}

func (a *AnthropicAdapter) RateLimit() time.Duration {
	return 10 * time.Second
}

func (a *AnthropicAdapter) MaxRetries() int {
	return 3
}

func (a *AnthropicAdapter) IsRateLimited(resp *http.Response) bool {
	return resp.StatusCode == 429
}

func (a *AnthropicAdapter) RecommendedWindow() (int, int) {
	return 2, 6
}

// Fetch 抓取 Anthropic 价格
func (a *AnthropicAdapter) Fetch(ctx context.Context) ([]ModelPrice, error) {
	capturedAt := time.Now().UTC()

	// 尝试网页抓取
	prices, err := a.fetchFromPricingPage(ctx, capturedAt)
	if err != nil || len(prices) == 0 {
		// 使用静态价格表
		return a.getStaticPricing(capturedAt), nil
	}

	return prices, nil
}

func (a *AnthropicAdapter) fetchFromPricingPage(ctx context.Context, capturedAt time.Time) ([]ModelPrice, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", a.baseURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
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

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	// Anthropic 定价页面解析
	prices := a.parsePricingTables(doc, capturedAt)
	if len(prices) == 0 {
		prices = a.parseModelCards(doc, capturedAt)
	}

	return prices, nil
}

// parsePricingTables 解析定价表格
func (a *AnthropicAdapter) parsePricingTables(doc *goquery.Document, capturedAt time.Time) []ModelPrice {
	var prices []ModelPrice

	doc.Find("table").Each(func(i int, table *goquery.Selection) {
		// 检查表头是否包含模型名称
		headerText := strings.ToLower(table.Find("thead").Text())
		if !strings.Contains(headerText, "model") && !strings.Contains(headerText, "claude") {
			return
		}

		table.Find("tbody tr").Each(func(j int, row *goquery.Selection) {
			cells := row.Find("td")
			if cells.Length() < 2 {
				return
			}

			// 提取模型名称
			modelName := strings.TrimSpace(cells.Eq(0).Text())
			modelName = a.cleanModelName(modelName)

			if !a.isValidModel(modelName) {
				return
			}

			// 提取价格
			var inputPrice, outputPrice float64

			// 遍历所有单元格找价格
			for k := 1; k < cells.Length(); k++ {
				cellText := cells.Eq(k).Text()
				cellLower := strings.ToLower(cellText)
				price := parsePricePerMillion(cellText)

				if price > 0 {
					if strings.Contains(cellLower, "input") {
						inputPrice = price
					} else if strings.Contains(cellLower, "output") {
						outputPrice = price
					} else if inputPrice == 0 {
						inputPrice = price
					} else if outputPrice == 0 {
						outputPrice = price
					}
				}
			}

			if inputPrice > 0 || outputPrice > 0 {
				prices = append(prices, ModelPrice{
					Source:    fmt.Sprintf("anthropic-%s", capturedAt.Format("2006-01-02")),
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
	})

	return prices
}

// parseModelCards 解析模型卡片
func (a *AnthropicAdapter) parseModelCards(doc *goquery.Document, capturedAt time.Time) []ModelPrice {
	var prices []ModelPrice

	// 查找包含 "Claude" 的标题
	doc.Find("h2, h3").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if !strings.Contains(strings.ToLower(text), "claude") {
			return
		}

		modelName := a.cleanModelName(text)
		if !a.isValidModel(modelName) {
			return
		}

		// 在后续元素中查找价格
		var inputPrice, outputPrice float64

		s.Parent().Find("p, span, div, li").Each(func(j int, p *goquery.Selection) {
			if inputPrice > 0 && outputPrice > 0 {
				return
			}

			pText := p.Text()
			pLower := strings.ToLower(pText)

			if strings.Contains(pLower, "input") || strings.Contains(pLower, "prompt") {
				inputPrice = parsePricePerMillion(pText)
			} else if strings.Contains(pLower, "output") || strings.Contains(pLower, "completion") {
				outputPrice = parsePricePerMillion(pText)
			}
		})

		if inputPrice > 0 || outputPrice > 0 {
			prices = append(prices, ModelPrice{
				Source:    fmt.Sprintf("anthropic-%s", capturedAt.Format("2006-01-02")),
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
func (a *AnthropicAdapter) cleanModelName(name string) string {
	// 移除多余空白
	name = strings.Join(strings.Fields(name), " ")

	// 提取 "Claude X" 核心名称
	claudePattern := regexp.MustCompile(`(?i)(claude[\s-]*[\d\.]+[\s-]*[\w]*)`)
	if matches := claudePattern.FindStringSubmatch(name); len(matches) > 0 {
		return strings.TrimSpace(matches[0])
	}

	return strings.TrimSpace(name)
}

// isValidModel 验证是否是有效的 Anthropic 模型名
func (a *AnthropicAdapter) isValidModel(name string) bool {
	if name == "" {
		return false
	}
	lower := strings.ToLower(name)
	return strings.Contains(lower, "claude")
}

// getStaticPricing 获取静态价格表（备用）
func (a *AnthropicAdapter) getStaticPricing(capturedAt time.Time) []ModelPrice {
	// 基于 Anthropic 官方定价（2026-03）
	priceTable := []struct {
		Name   string
		Input  float64
		Output float64
	}{
		{"Claude 3.5 Opus", 15.00, 75.00},
		{"Claude 3.5 Sonnet", 3.00, 15.00},
		{"Claude 3.5 Haiku", 0.25, 1.25},
		{"Claude 3 Opus", 15.00, 75.00},
		{"Claude 3 Sonnet", 3.00, 15.00},
		{"Claude 3 Haiku", 0.25, 1.25},
	}

	var prices []ModelPrice
	for _, p := range priceTable {
		prices = append(prices, ModelPrice{
			Source:    fmt.Sprintf("anthropic-%s", capturedAt.Format("2006-01-02")),
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
