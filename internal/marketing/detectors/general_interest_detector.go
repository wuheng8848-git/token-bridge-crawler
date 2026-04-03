// Package detectors 提供各种信号检测器的实现
package detectors

import (
	"strings"
	"time"

	"token-bridge-crawler/internal/core"
	"token-bridge-crawler/internal/marketing/types"
)

// GeneralInterestDetector 泛兴趣信号检测器
type GeneralInterestDetector struct {}

// NewGeneralInterestDetector 创建泛兴趣信号检测器
func NewGeneralInterestDetector() *GeneralInterestDetector {
	return &GeneralInterestDetector{}
}

// GetSupportedTypes 返回支持的信号类型
func (d *GeneralInterestDetector) GetSupportedTypes() []types.SignalType {
	return []types.SignalType{types.SignalTypeGeneralInterest}
}

// DetectFromIntel 从情报项中检测泛兴趣信号
func (d *GeneralInterestDetector) DetectFromIntel(item core.IntelItem) ([]types.CustomerSignal, error) {
	var signals []types.CustomerSignal

	// 检查内容是否包含泛兴趣相关关键词
	content := strings.ToLower(item.Title + " " + item.Content)

	// 泛兴趣关键词
	interestKeywords := []string{
		"ai", "artificial intelligence", "machine learning", "ml",
		"llm", "large language model", "model", "models",
		"gpt", "chatgpt", "openai", "claude", "gemini",
		"llama", "mistral", "ai tools", "ai technology",
		"cool", "awesome", "amazing", "great", "good",
		"try", "trying", "test", "testing", "experiment",
		"learn", "learning", "explore", "exploring", "discover",
		"new", "latest", "update", "updates", "release",
	}

	// 排除其他高优先级信号的关键词
	excludeKeywords := []string{
		"cost", "costs", "bill", "billing", "budget",
		"config", "configuration", "setup", "install", "api key",
		"multiple", "many", "switch", "unified", "centralize",
		"team", "organization", "quota", "tracking", "attribution",
		"compare", "alternative", "replace", "switch", "migrate",
	}

	// 检测泛兴趣关键词
	foundInterestKeywords := []string{}
	for _, keyword := range interestKeywords {
		if strings.Contains(content, strings.ToLower(keyword)) {
			foundInterestKeywords = append(foundInterestKeywords, keyword)
		}
	}

	// 检查是否包含排除关键词
	hasExcludeKeywords := false
	for _, keyword := range excludeKeywords {
		if strings.Contains(content, strings.ToLower(keyword)) {
			hasExcludeKeywords = true
			break
		}
	}

	// 如果找到足够的泛兴趣关键词且没有排除关键词，生成信号
	if len(foundInterestKeywords) >= 2 && !hasExcludeKeywords {
		// 计算信号强度
		strength := types.SignalStrengthLow
		if len(foundInterestKeywords) >= 4 {
			strength = types.SignalStrengthMedium
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
			Type:      types.SignalTypeGeneralInterest,
			Strength:  strength,
			Content:   item.Title,
			Platform:  platform,
			Author:    author,
			URL:       item.URL,
			Metadata: map[string]interface{}{
				"found_keywords": foundInterestKeywords,
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