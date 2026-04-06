// Package detectors 提供各种信号检测器的实现
package detectors

import (
	"strings"
	"time"

	"token-bridge-crawler/internal/core"
	"token-bridge-crawler/internal/marketing/types"
)

// ConfigFrictionDetector 配置摩擦信号检测器
type ConfigFrictionDetector struct{}

// NewConfigFrictionDetector 创建配置摩擦信号检测器
func NewConfigFrictionDetector() *ConfigFrictionDetector {
	return &ConfigFrictionDetector{}
}

// GetSupportedTypes 返回支持的信号类型
func (d *ConfigFrictionDetector) GetSupportedTypes() []types.SignalType {
	return []types.SignalType{types.SignalTypeConfigFriction}
}

// DetectFromIntel 从情报项中检测配置摩擦信号
func (d *ConfigFrictionDetector) DetectFromIntel(item core.IntelItem) ([]types.CustomerSignal, error) {
	var signals []types.CustomerSignal

	// 检查内容是否包含配置摩擦相关关键词
	content := strings.ToLower(item.Title + " " + item.Content)

	// 配置摩擦关键词
	configKeywords := []string{
		"config", "configuration", "setup", "installation", "install",
		"api key", "api keys", "provider", "providers", "base url",
		"mcp", "skills", "tool config", "tool configuration", "difficult",
		"hard", "complicated", "complex", "confusing", "trouble",
		"problem", "issue", "error", "fail", "failed",
		"openai compatible", "compatible", "incompatible", "migration",
		"switch", "change", "provider", "base url", "endpoint",
		"authentication", "auth", "keys", "key management",
	}

	// 检测关键词
	foundKeywords := []string{}
	for _, keyword := range configKeywords {
		if strings.Contains(content, strings.ToLower(keyword)) {
			foundKeywords = append(foundKeywords, keyword)
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
			ID:       generateSignalID(),
			Type:     types.SignalTypeConfigFriction,
			Strength: strength,
			Content:  item.Title,
			Platform: platform,
			Author:   author,
			URL:      item.URL,
			Metadata: map[string]interface{}{
				"found_keywords": foundKeywords,
				"intel_type":     item.IntelType,
				"source":         item.Source,
			},
			DetectedAt:   time.Now().UTC(),
			RelatedIntel: item.ID,
		}

		signals = append(signals, signal)
	}

	return signals, nil
}
