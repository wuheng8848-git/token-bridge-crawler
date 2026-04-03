// Package detectors 提供各种信号检测器的实现
package detectors

import (
	"regexp"
	"strings"
	"time"

	"token-bridge-crawler/internal/core"
	"token-bridge-crawler/internal/marketing/types"
)

// CostPressureDetector 成本压力信号检测器
type CostPressureDetector struct {}

// NewCostPressureDetector 创建成本压力信号检测器
func NewCostPressureDetector() *CostPressureDetector {
	return &CostPressureDetector{}
}

// GetSupportedTypes 返回支持的信号类型
func (d *CostPressureDetector) GetSupportedTypes() []types.SignalType {
	return []types.SignalType{types.SignalTypeCostPressure}
}

// DetectFromIntel 从情报项中检测成本压力信号
func (d *CostPressureDetector) DetectFromIntel(item core.IntelItem) ([]types.CustomerSignal, error) {
	var signals []types.CustomerSignal

	// 检查内容是否包含成本压力相关关键词
	content := strings.ToLower(item.Title + " " + item.Content)

	// 成本压力关键词
	costKeywords := []string{
		"bill", "billing", "cost", "costs", "expensive", "overpriced",
		"over budget", "budget", "subscription", "subscriptions", "paying too much",
		"token cost", "token costs", "api cost", "api costs", "save money",
		"reduce cost", "cut cost", "8.5x", "savings", "expensive",
		"too expensive", "costly", "pricey", "high cost", "high costs",
		"monthly bill", "monthly costs", "usage cost", "usage costs",
	}

	// 检测关键词
	foundKeywords := []string{}
	for _, keyword := range costKeywords {
		if strings.Contains(content, strings.ToLower(keyword)) {
			foundKeywords = append(foundKeywords, keyword)
		}
	}

	// 检查是否有成本相关的正则表达式匹配
	costRegexes := []*regexp.Regexp{
		regexp.MustCompile(`\$\d+(\.\d+)?`), // 价格
		regexp.MustCompile(`\d+\s*\$`),     // 价格
		regexp.MustCompile(`cost.*\d+`),      // 成本数字
		regexp.MustCompile(`budget.*\d+`),    // 预算数字
	}

	for _, regex := range costRegexes {
		if regex.MatchString(content) {
			foundKeywords = append(foundKeywords, "price_number")
			break
		}
	}

	// 如果找到足够的关键词，生成信号
	if len(foundKeywords) >= 2 {
		// 计算信号强度
		strength := types.SignalStrengthMedium
		if len(foundKeywords) >= 4 {
			strength = types.SignalStrengthHigh
		} else if len(foundKeywords) == 1 {
			strength = types.SignalStrengthLow
		}

		// 从元数据中获取平台和作者信息
		platform := "unknown"
		author := "unknown"
		if item.Metadata != nil {
			if p, ok := item.Metadata["platform"].(string); ok {
				platform = p
			}
			if a, ok := item.Metadata["author"].(string); ok {
				author = a
			}
		}

		signal := types.CustomerSignal{
			ID:        generateSignalID(),
			Type:      types.SignalTypeCostPressure,
			Strength:  strength,
			Content:   item.Title,
			Platform:  platform,
			Author:    author,
			URL:       item.URL,
			Metadata: map[string]interface{}{
				"found_keywords": foundKeywords,
				"intel_type":     item.IntelType,
				"source":         item.Source,
			},
			DetectedAt:    time.Now().UTC(),
			RelatedIntel:  item.ID,
		}

		signals = append(signals, signal)
	}

	return signals, nil
}

// generateSignalID 生成信号ID
func generateSignalID() string {
	return time.Now().UTC().Format("20060102150405") + "-" + randomString(6)
}

// randomString 生成随机字符串
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
