// Package detectors 提供各种信号检测器的实现
package detectors

import (
	"strings"
	"time"

	"token-bridge-crawler/internal/core"
	"token-bridge-crawler/internal/marketing/types"
)

// ToolFragmentationDetector 工具碎片化信号检测器
type ToolFragmentationDetector struct {}

// NewToolFragmentationDetector 创建工具碎片化信号检测器
func NewToolFragmentationDetector() *ToolFragmentationDetector {
	return &ToolFragmentationDetector{}
}

// GetSupportedTypes 返回支持的信号类型
func (d *ToolFragmentationDetector) GetSupportedTypes() []types.SignalType {
	return []types.SignalType{types.SignalTypeToolFragmentation}
}

// DetectFromIntel 从情报项中检测工具碎片化信号
func (d *ToolFragmentationDetector) DetectFromIntel(item core.IntelItem) ([]types.CustomerSignal, error) {
	var signals []types.CustomerSignal

	// 检查内容是否包含工具碎片化相关关键词
	content := strings.ToLower(item.Title + " " + item.Content)

	// 工具名称列表
	toolNames := []string{
		"cursor", "claude", "claude code", "openai", "gpt", "chatgpt",
		"gemini", "google gemini", "anthropic", "claude", "llama", "mistral",
		"aider", "openclaw", "cline", "github copilot", "copilot",
		"vscode", "visual studio code", "jetbrains", "intellij",
		"notion", "slack", "discord", "telegram",
	}

	// 碎片化关键词
	fragmentationKeywords := []string{
		"multiple", "many", "several", "various", "different",
		"switch", "alternate", "alternating", "toggle", "change",
		"manage", "management", "organize", "organization",
		"unified", "unify", "centralize", "centralized",
		"single interface", "one interface", "统一", "集中",
	}

	// 检测工具名称
	foundTools := []string{}
	for _, tool := range toolNames {
		if strings.Contains(content, strings.ToLower(tool)) {
			foundTools = append(foundTools, tool)
		}
	}

	// 检测碎片化关键词
	foundFragmentationKeywords := []string{}
	for _, keyword := range fragmentationKeywords {
		if strings.Contains(content, strings.ToLower(keyword)) {
			foundFragmentationKeywords = append(foundFragmentationKeywords, keyword)
		}
	}

	// 如果找到足够的工具名称和碎片化关键词，生成信号
	if len(foundTools) >= 2 || (len(foundTools) >= 1 && len(foundFragmentationKeywords) >= 1) {
		// 计算信号强度
		strength := types.SignalStrengthMedium
		if len(foundTools) >= 4 || (len(foundTools) >= 2 && len(foundFragmentationKeywords) >= 2) {
			strength = types.SignalStrengthHigh
		} else if len(foundTools) == 1 && len(foundFragmentationKeywords) == 0 {
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
			Type:      types.SignalTypeToolFragmentation,
			Strength:  strength,
			Content:   item.Title,
			Platform:  platform,
			Author:    author,
			URL:       item.URL,
			Metadata: map[string]interface{}{
				"found_tools":                foundTools,
				"found_fragmentation_keywords": foundFragmentationKeywords,
				"intel_type":                 item.IntelType,
				"source":                     item.Source,
			},
			DetectedAt:    time.Now().UTC(),
			RelatedIntel:  item.ID,
		}

		signals = append(signals, signal)
	}

	return signals, nil
}
