// Package generators 提供各种动作生成器的实现
package generators

import (
	"time"

	"token-bridge-crawler/internal/marketing/types"
)

// GovernanceActionGenerator 治理起点动作生成器
type GovernanceActionGenerator struct {}

// NewGovernanceActionGenerator 创建治理起点动作生成器
func NewGovernanceActionGenerator() *GovernanceActionGenerator {
	return &GovernanceActionGenerator{}
}

// GetSupportedActions 返回支持的动作类型
func (g *GovernanceActionGenerator) GetSupportedActions() []types.ActionType {
	return []types.ActionType{
		types.ActionTypeShortResponse,
		types.ActionTypeTechnicalPost,
	}
}

// GenerateActions 根据治理起点信号生成营销动作
func (g *GovernanceActionGenerator) GenerateActions(signals []types.QualifiedSignal) ([]types.MarketingAction, error) {
	var actions []types.MarketingAction

	// 筛选治理起点信号
	governanceSignals := []types.QualifiedSignal{}
	for _, signal := range signals {
		if signal.Signal.Type == types.SignalTypeGovernanceStart {
			governanceSignals = append(governanceSignals, signal)
		}
	}

	if len(governanceSignals) == 0 {
		return actions, nil
	}

	// 生成动作
	for _, signal := range governanceSignals {
		// 根据信号强度生成不同优先级的动作
		priority := 3
		if signal.Signal.Strength == types.SignalStrengthHigh {
			priority = 4
		} else if signal.Signal.Strength == types.SignalStrengthMedium {
			priority = 3
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
func (g *GovernanceActionGenerator) generateInternalAction(signal types.QualifiedSignal, priority int) types.MarketingAction {
	content := "内部动作建议：针对团队治理需求，建议提供预算控制、使用追踪和成本归因功能的解决方案。"

	return types.MarketingAction{
		ID:             generateActionID(),
		Type:           types.ActionTypeShortResponse,
		Channel:        types.ChannelInternal,
		Title:          "团队治理需求动作建议",
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
