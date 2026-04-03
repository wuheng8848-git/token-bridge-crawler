// Package generators 提供各种动作生成器的实现
package generators

import (
	"time"

	"token-bridge-crawler/internal/marketing/types"
)

// ToolActionGenerator 工具碎片化动作生成器
type ToolActionGenerator struct {}

// NewToolActionGenerator 创建工具碎片化动作生成器
func NewToolActionGenerator() *ToolActionGenerator {
	return &ToolActionGenerator{}
}

// GetSupportedActions 返回支持的动作类型
func (g *ToolActionGenerator) GetSupportedActions() []types.ActionType {
	return []types.ActionType{
		types.ActionTypeInternalNote,
		types.ActionTypeStrategy,
	}
}

// GenerateActions 根据工具碎片化信号生成营销动作
func (g *ToolActionGenerator) GenerateActions(signals []types.QualifiedSignal) ([]types.MarketingAction, error) {
	var actions []types.MarketingAction

	// 筛选工具碎片化信号
	toolSignals := []types.QualifiedSignal{}
	for _, signal := range signals {
		if signal.Signal.Type == types.SignalTypeToolFragmentation {
			toolSignals = append(toolSignals, signal)
		}
	}

	if len(toolSignals) == 0 {
		return actions, nil
	}

	// 生成动作
	for _, signal := range toolSignals {
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

		// 生成技术文章（内部建议）
		if signal.Signal.Strength == types.SignalStrengthHigh {
			technicalPost := g.generateTechnicalPost(signal, priority)
			actions = append(actions, technicalPost)
		}
	}

	return actions, nil
}

// generateShortResponse 生成短内容回应
func (g *ToolActionGenerator) generateShortResponse(signal types.QualifiedSignal, priority int) types.MarketingAction {
	content := "【内部动作建议】工具碎片化信号响应：用户在管理多个AI工具时遇到困难，建议提供Token Bridge的统一接口解决方案，简化多模型和多提供商的管理。"

	return types.MarketingAction{
		ID:          generateActionID(),
		Type:        types.ActionTypeInternalNote,
		Channel:     types.ChannelInternal,
		Title:       "工具碎片化信号响应建议",
		Content:     content,
		TargetAudience: "Developers using multiple AI tools",
		Priority:    priority,
		SignalIDs:   []string{signal.Signal.ID},
		Metadata: map[string]interface{}{
			"signal_type": signal.Signal.Type,
			"signal_strength": signal.Signal.Strength,
			"platform": signal.Signal.Platform,
			"qualified_score": signal.Score,
			"customer_stage": signal.CustomerStage,
		},
		CreatedAt:   time.Now().UTC(),
		Status:      "draft",
		AutoExecute: false,
		CustomerStage: signal.CustomerStage,
		QualifiedScore: signal.Score,
	}
}

// generateTechnicalPost 生成技术文章
func (g *ToolActionGenerator) generateTechnicalPost(signal types.QualifiedSignal, priority int) types.MarketingAction {
	title := "【策略建议】统一AI工具管理内容创作"
	content := `【内部策略建议】

基于检测到的工具碎片化信号，建议创建一篇关于Token Bridge如何简化AI工作流的技术文章。

文章要点：
1. 多工具管理的痛点分析
2. Token Bridge的统一API解决方案
3. 多提供商无缝切换能力
4. 集中化的账单和使用跟踪
5. 实际案例和配置示例

目标受众：管理多个AI工具的开发者
推荐发布渠道：技术博客、GitHub文档、开发者社区`

	return types.MarketingAction{
		ID:          generateActionID(),
		Type:        types.ActionTypeStrategy,
		Channel:     types.ChannelInternal,
		Title:       title,
		Content:     content,
		TargetAudience: "Developers managing multiple AI tools",
		Priority:    priority,
		SignalIDs:   []string{signal.Signal.ID},
		Metadata: map[string]interface{}{
			"signal_type": signal.Signal.Type,
			"signal_strength": signal.Signal.Strength,
			"platform": signal.Signal.Platform,
			"qualified_score": signal.Score,
			"customer_stage": signal.CustomerStage,
		},
		CreatedAt:   time.Now().UTC(),
		Status:      "draft",
		AutoExecute: false,
		CustomerStage: signal.CustomerStage,
		QualifiedScore: signal.Score,
	}
}
