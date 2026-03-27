// Package adapters 提供各厂商价格适配器
package adapters

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ModelPrice 统一价格结构
type ModelPrice struct {
	Source       string                 // 来源标识，如 "openai-2026-03-24"
	ModelCode    string                 // 官方模型代码，如 "gpt-4o"
	ModelName    string                 // 展示名，如 "GPT-4o"
	Vendor       string                 // 厂商标识
	PricingRaw   PricingRaw             // 价格原始数据
	Capabilities map[string]interface{} // 扩展能力指标
}

// PricingRaw 刊例价原始数据
type PricingRaw struct {
	InputUSDPerMillion  *float64  `json:"input_usd_per_million,omitempty"`
	OutputUSDPerMillion *float64  `json:"output_usd_per_million,omitempty"`
	Currency            string    `json:"currency"`
	PriceType           string    `json:"price_type"`     // "vendor_list_price"
	SchemaVersion       string    `json:"schema_version"` // "v1"
	CapturedAt          time.Time `json:"captured_at"`
}

// VendorAdapter 厂商适配器接口
type VendorAdapter interface {
	// Name 返回厂商标识名
	Name() string

	// DisplayName 返回厂商展示名
	DisplayName() string

	// Fetch 抓取价格数据
	Fetch(ctx context.Context) ([]ModelPrice, error)

	// RateLimit 返回请求间隔（防封）
	RateLimit() time.Duration

	// MaxRetries 返回最大重试次数
	MaxRetries() int

	// IsRateLimited 检测是否触发限流
	IsRateLimited(resp *http.Response) bool

	// RecommendedWindow 返回建议抓取时间窗口 (start_hour, end_hour)
	RecommendedWindow() (int, int)
}

// ErrRateLimited 限流错误
type ErrRateLimited struct {
	Vendor     string
	RetryAfter time.Duration
}

func (e ErrRateLimited) Error() string {
	return "rate limited: " + e.Vendor
}

// normalizeModelCode 标准化模型代码
// 转小写，替换特殊字符为连字符
func normalizeModelCode(name string) string {
	// 转小写
	code := strings.ToLower(name)
	
	// 替换常见分隔符为连字符
	code = strings.ReplaceAll(code, "/", "-")
	code = strings.ReplaceAll(code, "_", "-")
	code = strings.ReplaceAll(code, " ", "-")
	
	// 移除多余连字符
	re := regexp.MustCompile(`-+`)
	code = re.ReplaceAllString(code, "-")
	
	// 移除首尾连字符
	code = strings.Trim(code, "-")
	
	return code
}

// parsePrice 解析价格字符串
// 支持格式: "$2.50", "$0.002", "2.5", "$10.00 / 1M tokens"
func parsePrice(s string) float64 {
	if s == "" || s == "-" || s == "N/A" {
		return 0
	}
	
	// 提取数字部分（包括小数点）
	re := regexp.MustCompile(`[$]?([0-9]*\.?[0-9]+)`)
	matches := re.FindStringSubmatch(s)
	if len(matches) < 2 {
		return 0
	}
	
	val, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0
	}
	
	return val
}

// parsePricePerMillion 解析每百万 token 的价格
// 支持 "$5.00 / 1M tokens" 或 "$0.005 / 1K tokens"
func parsePricePerMillion(s string) float64 {
	if s == "" {
		return 0
	}
	
	// 提取价格
	price := parsePrice(s)
	if price == 0 {
		return 0
	}
	
	// 检测单位
	lower := strings.ToLower(s)
	if strings.Contains(lower, "/ 1k") || strings.Contains(lower, "per 1k") {
		// 每 1K，需要乘以 1000 转换为每 1M
		return price * 1000
	}
	if strings.Contains(lower, "/ 1m") || strings.Contains(lower, "per 1m") || strings.Contains(lower, "million") {
		// 已经是每 1M
		return price
	}
	
	// 默认假设是每 1M
	return price
}

// GenerateID 生成 UUID
func GenerateID() string {
	return uuid.New().String()
}

// floatPtr 返回 float64 指针
func floatPtr(f float64) *float64 {
	return &f
}
