// Package detectors 提供各种信号检测器的实现
package detectors

import (
	"strings"
	"time"

	"token-bridge-crawler/internal/core"
	"token-bridge-crawler/internal/marketing/types"
)

// GovernanceStartDetector 治理起点信号检测器
type GovernanceStartDetector struct {}

// NewGovernanceStartDetector 创建治理起点信号检测器
func NewGovernanceStartDetector() *GovernanceStartDetector {
	return &GovernanceStartDetector{}
}

// GetSupportedTypes 返回支持的信号类型
func (d *GovernanceStartDetector) GetSupportedTypes() []types.SignalType {
	return []types.SignalType{types.SignalTypeGovernanceStart}
}

// DetectFromIntel 从情报项中检测治理起点信号
func (d *GovernanceStartDetector) DetectFromIntel(item core.IntelItem) ([]types.CustomerSignal, error) {
	var signals []types.CustomerSignal

	// 检查内容是否包含治理起点相关关键词
	content := strings.ToLower(item.Title + " " + item.Content)

	// 治理相关关键词
	governanceKeywords := []string{
		"team", "teams", "organization", "company", "enterprise",
		"budget", "budgets", "quota", "quotas", "limit", "limits",
		"tracking", "track", "monitor", "monitoring", "usage",
		"who", "which", "whose", "who is", "who's",
		"cost attribution", "attribution", "allocate", "allocation",
		"billing", "invoice", "invoicing", "accounting",
		"audit", "auditing", "compliance", "policy", "policies",
		"admin", "administrator", "management", "manage",
		"alert", "alerts", "notification", "notifications",
	}

	// 检测关键词
	foundKeywords := []string{}
	for _, keyword := range governanceKeywords {
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
			ID:        generateSignalID(),
			Type:      types.SignalTypeGovernanceStart,
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