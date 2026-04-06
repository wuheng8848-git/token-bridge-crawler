// Package generators 提供各种动作生成器的实现
package generators

import (
	"time"

	"token-bridge-crawler/internal/marketing/types"
)

// ConfigActionGenerator 配置动作生成器
type ConfigActionGenerator struct{}

// NewConfigActionGenerator 创建配置动作生成器
func NewConfigActionGenerator() *ConfigActionGenerator {
	return &ConfigActionGenerator{}
}

// GetSupportedActions 返回支持的动作类型
func (g *ConfigActionGenerator) GetSupportedActions() []types.ActionType {
	return []types.ActionType{
		types.ActionTypeInternalNote,
		types.ActionTypeStrategy,
	}
}

// GenerateActions 根据配置摩擦信号生成营销动作
func (g *ConfigActionGenerator) GenerateActions(signals []types.QualifiedSignal) ([]types.MarketingAction, error) {
	var actions []types.MarketingAction

	// 筛选配置摩擦信号
	configSignals := []types.QualifiedSignal{}
	for _, signal := range signals {
		if signal.Signal.Type == types.SignalTypeConfigFriction {
			configSignals = append(configSignals, signal)
		}
	}

	if len(configSignals) == 0 {
		return actions, nil
	}

	// 生成动作
	for _, signal := range configSignals {
		// 根据信号强度生成不同优先级的动作
		priority := 3
		if signal.Signal.Strength == types.SignalStrengthHigh {
			priority = 5
		} else if signal.Signal.Strength == types.SignalStrengthMedium {
			priority = 4
		}

		// 生成短内容回应（内部建议）
		if signal.Signal.Strength >= types.SignalStrengthMedium {
			shortResponse := g.generateShortResponse(signal, priority)
			actions = append(actions, shortResponse)
		}

		// 生成配置教程（内部建议）
		if signal.Signal.Strength >= types.SignalStrengthMedium {
			setupGuide := g.generateSetupGuide(signal, priority)
			actions = append(actions, setupGuide)
		}
	}

	return actions, nil
}

// generateShortResponse 生成短内容回应
func (g *ConfigActionGenerator) generateShortResponse(signal types.QualifiedSignal, priority int) types.MarketingAction {
	content := "【内部动作建议】配置摩擦信号响应：用户在配置多个AI提供商时遇到困难，建议提供Token Bridge的单API密钥解决方案和简化配置流程。"

	return types.MarketingAction{
		ID:             generateActionID(),
		Type:           types.ActionTypeInternalNote,
		Channel:        types.ChannelInternal,
		Title:          "配置摩擦信号响应建议",
		Content:        content,
		TargetAudience: "Developers struggling with setup",
		Priority:       priority,
		SignalIDs:      []string{signal.Signal.ID},
		Metadata: map[string]interface{}{
			"signal_type":     signal.Signal.Type,
			"signal_strength": signal.Signal.Strength,
			"platform":        signal.Signal.Platform,
			"qualified_score": signal.Score,
			"customer_stage":  signal.CustomerStage,
		},
		CreatedAt:      time.Now().UTC(),
		Status:         "draft",
		AutoExecute:    false,
		CustomerStage:  signal.CustomerStage,
		QualifiedScore: signal.Score,
	}
}

// generateSetupGuide 生成配置教程
func (g *ConfigActionGenerator) generateSetupGuide(signal types.QualifiedSignal, priority int) types.MarketingAction {
	title := "【策略建议】配置教程创作"
	content := `【内部策略建议】

基于检测到的配置摩擦信号，建议创建一份详细的Token Bridge配置教程。

教程要点：
1. Token Bridge的单API密钥优势
2. 多提供商配置流程
3. 代码示例和最佳实践
4. 常见配置问题的解决方案

目标受众：需要配置帮助的开发者
推荐发布渠道：技术博客、GitHub文档`

	return types.MarketingAction{
		ID:             generateActionID(),
		Type:           types.ActionTypeStrategy,
		Channel:        types.ChannelInternal,
		Title:          title,
		Content:        content,
		TargetAudience: "Developers needing configuration help",
		Priority:       priority,
		SignalIDs:      []string{signal.Signal.ID},
		Metadata: map[string]interface{}{
			"signal_type":     signal.Signal.Type,
			"signal_strength": signal.Signal.Strength,
			"platform":        signal.Signal.Platform,
			"qualified_score": signal.Score,
			"customer_stage":  signal.CustomerStage,
		},
		CreatedAt:      time.Now().UTC(),
		Status:         "draft",
		AutoExecute:    false,
		CustomerStage:  signal.CustomerStage,
		QualifiedScore: signal.Score,
	}
}
