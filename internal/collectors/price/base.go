// Package price 提供价格采集器
package price

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"token-bridge-crawler/internal/core"

	"github.com/PuerkitoBio/goquery"
)

// PriceCollector 价格采集器基类
type PriceCollector struct {
	core.BaseCollector

	vendor      string
	displayName string
	config      PriceCollectorConfig

	// 多策略抓取函数
	strategies []core.FetchStrategy

	// fallback 价格数据（子类设置）
	fallbackPrices []PriceData
}

// PriceCollectorConfig 价格采集器配置
type PriceCollectorConfig struct {
	WebURL     string
	APIURL     string
	StaticFile string
	RateLimit  time.Duration
}

// PriceData 价格数据
type PriceData struct {
	ModelCode           string  `json:"model_code"`
	ModelName           string  `json:"model_name"`
	InputPrice          float64 `json:"input_price"`
	OutputPrice         float64 `json:"output_price"`
	Currency            string  `json:"currency"`
	ChangeType          string  `json:"change_type,omitempty"` // "new", "updated", "unchanged"
	PreviousInputPrice  float64 `json:"previous_input_price,omitempty"`
	PreviousOutputPrice float64 `json:"previous_output_price,omitempty"`
}

// NewPriceCollector 创建价格采集器
func NewPriceCollector(vendor, displayName string, config PriceCollectorConfig) *PriceCollector {
	pc := &PriceCollector{
		BaseCollector: core.NewBaseCollector(
			fmt.Sprintf("price_%s", vendor),
			core.IntelTypePrice,
			vendor,
			config.RateLimit,
		),
		vendor:      vendor,
		displayName: displayName,
		config:      config,
	}

	// 初始化策略链
	pc.initStrategies()

	return pc
}

// initStrategies 初始化抓取策略链
func (pc *PriceCollector) initStrategies() {
	pc.strategies = []core.FetchStrategy{
		{
			Name:     "web",
			Priority: 1,
			Fetch:    pc.fetchFromWeb,
		},
		{
			Name:     "api",
			Priority: 2,
			Fetch:    pc.fetchFromAPI,
		},
		{
			Name:     "static",
			Priority: 3,
			Fetch:    pc.fetchFromStatic,
		},
		{
			Name:     "fallback",
			Priority: 4,
			Fetch:    pc.fetchFromFallback,
		},
	}
}

// Fetch 实现基础采集接口
func (pc *PriceCollector) Fetch(ctx context.Context) ([]core.IntelItem, error) {
	return pc.FetchWithFallback(ctx)
}

// FetchWithFallback 带降级策略的抓取
func (pc *PriceCollector) FetchWithFallback(ctx context.Context) ([]core.IntelItem, error) {
	executor := core.NewStrategyExecutor(pc.strategies)
	items, _, err := executor.Execute(ctx)
	return items, err
}

// fetchFromWeb 从网页抓取（子类实现）
func (pc *PriceCollector) fetchFromWeb(ctx context.Context) ([]core.IntelItem, error) {
	return nil, fmt.Errorf("web fetch not implemented")
}

// fetchFromAPI 从API抓取（子类实现）
func (pc *PriceCollector) fetchFromAPI(ctx context.Context) ([]core.IntelItem, error) {
	return nil, fmt.Errorf("api fetch not implemented")
}

// fetchFromStatic 从静态文件抓取
func (pc *PriceCollector) fetchFromStatic(ctx context.Context) ([]core.IntelItem, error) {
	if pc.config.StaticFile == "" {
		return nil, fmt.Errorf("static file not configured")
	}

	data, err := os.ReadFile(pc.config.StaticFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read static file: %w", err)
	}

	var prices []PriceData
	if err := json.Unmarshal(data, &prices); err != nil {
		return nil, fmt.Errorf("failed to unmarshal static data: %w", err)
	}

	return pc.pricesToIntelItems(prices), nil
}

// fetchFromFallback 从内置 fallback 数据抓取
func (pc *PriceCollector) fetchFromFallback(ctx context.Context) ([]core.IntelItem, error) {
	if len(pc.fallbackPrices) == 0 {
		return nil, fmt.Errorf("no fallback prices configured")
	}

	log.Printf("[Collector:%s] Using fallback prices (web/api/static all failed)", pc.vendor)
	return pc.pricesToIntelItems(pc.fallbackPrices), nil
}

// SetFallbackPrices 设置 fallback 价格数据（子类调用）
func (pc *PriceCollector) SetFallbackPrices(prices []PriceData) {
	pc.fallbackPrices = prices
}

// pricesToIntelItems 将价格数据转换为情报项
func (pc *PriceCollector) pricesToIntelItems(prices []PriceData) []core.IntelItem {
	var items []core.IntelItem
	now := time.Now().UTC()

	for _, price := range prices {
		item := core.NewIntelItem(core.IntelTypePrice, pc.vendor)
		item.SourceID = fmt.Sprintf("%s-%s-%s", pc.vendor, price.ModelCode, now.Format("20060102"))
		item.Title = fmt.Sprintf("%s - %s", pc.displayName, price.ModelName)
		item.Content = fmt.Sprintf("Input: $%.2f/million, Output: $%.2f/million",
			price.InputPrice, price.OutputPrice)
		item.CapturedAt = now

		// 设置元数据
		item.Metadata = core.Metadata{
			"model_code":            price.ModelCode,
			"model_name":            price.ModelName,
			"input_price":           price.InputPrice,
			"output_price":          price.OutputPrice,
			"currency":              price.Currency,
			"price_type":            "vendor_list_price",
			"schema_version":        "v1",
			"change_type":           price.ChangeType,
			"previous_input_price":  price.PreviousInputPrice,
			"previous_output_price": price.PreviousOutputPrice,
		}

		items = append(items, item)
	}

	return items
}

// Validate 验证数据合理性
func (pc *PriceCollector) Validate(items []core.IntelItem) error {
	if len(items) == 0 {
		return nil
	}

	var errors []string

	for i, item := range items {
		// 检查必填字段
		if item.Source == "" {
			errors = append(errors, fmt.Sprintf("item %d: source is empty", i))
		}

		// 检查价格合理性
		if price, ok := item.Metadata["input_price"].(float64); ok {
			if price < 0 || price > 1000 {
				errors = append(errors, fmt.Sprintf("item %d: input_price %.2f seems unreasonable", i, price))
			}
		}

		if price, ok := item.Metadata["output_price"].(float64); ok {
			if price < 0 || price > 1000 {
				errors = append(errors, fmt.Sprintf("item %d: output_price %.2f seems unreasonable", i, price))
			}
		}

		// 检查模型代码
		if modelCode, ok := item.Metadata["model_code"].(string); !ok || modelCode == "" {
			errors = append(errors, fmt.Sprintf("item %d: model_code is empty", i))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %v", errors)
	}

	return nil
}

// HealthCheck 健康检查
func (pc *PriceCollector) HealthCheck() (string, error) {
	// 尝试静态文件（最可靠的源）
	if pc.config.StaticFile != "" {
		if _, err := os.Stat(pc.config.StaticFile); err == nil {
			return "healthy", nil
		}
	}

	// 检查网页可访问性
	if pc.config.WebURL != "" {
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(pc.config.WebURL)
		if err != nil {
			return "degraded", fmt.Errorf("web url not accessible: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			return "healthy", nil
		}
	}

	return "unhealthy", fmt.Errorf("no available data source")
}

// GetStrategies 获取支持的策略列表
func (pc *PriceCollector) GetStrategies() []core.FetchStrategy {
	return pc.strategies
}

// fetchWebPage 通用网页抓取辅助函数
func fetchWebPage(url string) (*goquery.Document, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// 设置请求头，模拟浏览器
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	return doc, nil
}

// saveStaticBackup 保存静态备份
func (pc *PriceCollector) saveStaticBackup(prices []PriceData) error {
	if pc.config.StaticFile == "" {
		return nil
	}

	// 确保目录存在
	dir := filepath.Dir(pc.config.StaticFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(prices, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(pc.config.StaticFile, data, 0644)
}

// LogFetchResult 记录抓取结果（用于监控）
func (pc *PriceCollector) LogFetchResult(strategy string, itemCount int, err error) {
	if err != nil {
		log.Printf("[Collector:%s] Strategy %s failed: %v", pc.vendor, strategy, err)
	} else {
		log.Printf("[Collector:%s] Strategy %s succeeded, got %d items", pc.vendor, strategy, itemCount)
	}
}
