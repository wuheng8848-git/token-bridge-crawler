// Package detectors 提供各种信号检测器的实现
package detectors

import (
	"strings"
	"time"

	"token-bridge-crawler/internal/core"
	"token-bridge-crawler/internal/marketing/types"
)

// MigrationIntentDetector 迁移意愿信号检测器
type MigrationIntentDetector struct {}

// NewMigrationIntentDetector 创建迁移意愿信号检测器
func NewMigrationIntentDetector() *MigrationIntentDetector {
	return &MigrationIntentDetector{}
}

// GetSupportedTypes 返回支持的信号类型
func (d *MigrationIntentDetector) GetSupportedTypes() []types.SignalType {
	return []types.SignalType{types.SignalTypeMigrationIntent}
}

// DetectFromIntel 从情报项中检测迁移意愿信号
func (d *MigrationIntentDetector) DetectFromIntel(item core.IntelItem) ([]types.CustomerSignal, error) {
	var signals []types.CustomerSignal

	// 检查内容是否包含迁移意愿相关关键词
	content := strings.ToLower(item.Title + " " + item.Content)

	// 竞品名称
	competitorNames := []string{
		"openrouter", "together", "portkey", "litellm", "onelink",
		"siliconflow", "ai gateway", "api gateway", "llm gateway",
	}

	// 迁移意愿关键词
	migrationKeywords := []string{
		"compare", "comparison", "vs", "versus", "vs.",
		"alternative", "alternatives", "replace", "replacement",
		"switch", "switching", "move", "moving", "migrate", "migration",
		"looking for", "searching for", "want", "need", "looking",
		"better", "best", "recommend", "recommendation",
		"evaluate", "evaluation", "considering", "consider",
	}

	// 检测竞品名称
	foundCompetitors := []string{}
	for _, competitor := range competitorNames {
		if strings.Contains(content, strings.ToLower(competitor)) {
			foundCompetitors = append(foundCompetitors, competitor)
		}
	}

	// 检测迁移意愿关键词
	foundMigrationKeywords := []string{}
	for _, keyword := range migrationKeywords {
		if strings.Contains(content, strings.ToLower(keyword)) {
			foundMigrationKeywords = append(foundMigrationKeywords, keyword)
		}
	}

	// 如果找到竞品名称和迁移意愿关键词，生成信号
	if (len(foundCompetitors) >= 1 && len(foundMigrationKeywords) >= 1) || len(foundMigrationKeywords) >= 2 {
		// 计算信号强度
		strength := types.SignalStrengthMedium
		if len(foundCompetitors) >= 2 && len(foundMigrationKeywords) >= 2 {
			strength = types.SignalStrengthHigh
		} else if (len(foundCompetitors) == 0 && len(foundMigrationKeywords) == 1) || (len(foundCompetitors) == 1 && len(foundMigrationKeywords) == 0) {
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
			Type:      types.SignalTypeMigrationIntent,
			Strength:  strength,
			Content:   item.Title,
			Platform:  platform,
			Author:    author,
			URL:       item.URL,
			Metadata: map[string]interface{}{
				"found_competitors":      foundCompetitors,
				"found_migration_keywords": foundMigrationKeywords,
				"intel_type":            item.IntelType,
				"source":                item.Source,
			},
			DetectedAt:    time.Now().UTC(),
			RelatedIntel:  item.ID,
		}

		signals = append(signals, signal)
	}

	return signals, nil
}