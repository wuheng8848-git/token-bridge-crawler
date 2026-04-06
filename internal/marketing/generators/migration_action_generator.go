// Package generators 提供各种动作生成器的实现
package generators

import (
	"time"

	"token-bridge-crawler/internal/marketing/types"
)

// MigrationActionGenerator 迁移意愿动作生成器
type MigrationActionGenerator struct{}

// NewMigrationActionGenerator 创建迁移意愿动作生成器
func NewMigrationActionGenerator() *MigrationActionGenerator {
	return &MigrationActionGenerator{}
}

// GetSupportedActions 返回支持的动作类型
func (g *MigrationActionGenerator) GetSupportedActions() []types.ActionType {
	return []types.ActionType{
		types.ActionTypeShortResponse,
		types.ActionTypeCompetitorComparison,
		types.ActionTypeFollowUp,
	}
}

// GenerateActions 根据迁移意愿信号生成营销动作
func (g *MigrationActionGenerator) GenerateActions(signals []types.QualifiedSignal) ([]types.MarketingAction, error) {
	var actions []types.MarketingAction

	// 筛选迁移意愿信号
	migrationSignals := []types.QualifiedSignal{}
	for _, signal := range signals {
		if signal.Signal.Type == types.SignalTypeMigrationIntent {
			migrationSignals = append(migrationSignals, signal)
		}
	}

	if len(migrationSignals) == 0 {
		return actions, nil
	}

	// 生成动作
	for _, signal := range migrationSignals {
		// 根据信号强度生成不同优先级的动作
		priority := 3
		if signal.Signal.Strength == types.SignalStrengthHigh {
			priority = 5
		} else if signal.Signal.Strength == types.SignalStrengthMedium {
			priority = 4
		}

		// 生成内部动作建议
		if signal.Signal.Strength >= types.SignalStrengthMedium {
			internalAction := g.generateInternalAction(signal, priority)
			actions = append(actions, internalAction)
		}
	}

	return actions, nil
}

// generateInternalAction 生成内部动作建议
func (g *MigrationActionGenerator) generateInternalAction(signal types.QualifiedSignal, priority int) types.MarketingAction {
	content := "内部动作建议：针对迁移意愿信号，建议提供成本节约、易用性和管理优势的解决方案。"

	return types.MarketingAction{
		ID:             generateActionID(),
		Type:           types.ActionTypeShortResponse,
		Channel:        types.ChannelInternal,
		Title:          "迁移意愿动作建议",
		Content:        content,
		TargetAudience: "内部销售与产品团队",
		Priority:       priority,
		SignalIDs:      []string{signal.Signal.ID},
		CustomerStage:  signal.CustomerStage,
		QualifiedScore: signal.Score,
		Metadata: map[string]interface{}{
			"signal_type":     signal.Signal.Type,
			"signal_strength": signal.Signal.Strength,
			"platform":        signal.Signal.Platform,
			"customer_stage":  signal.CustomerStage,
			"qualified_score": signal.Score,
		},
		CreatedAt:   time.Now().UTC(),
		Status:      "draft",
		AutoExecute: false,
	}
}
