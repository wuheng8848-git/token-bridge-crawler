// Package generators 提供各种动作生成器的实现
package generators

import (
	"time"

	"token-bridge-crawler/internal/marketing/types"
)

// CostActionGenerator 成本动作生成器
type CostActionGenerator struct {}

// NewCostActionGenerator 创建成本动作生成器
func NewCostActionGenerator() *CostActionGenerator {
	return &CostActionGenerator{}
}

// GetSupportedActions 返回支持的动作类型
func (g *CostActionGenerator) GetSupportedActions() []types.ActionType {
	return []types.ActionType{
		types.ActionTypeShortResponse,
		types.ActionTypeTechnicalPost,
		types.ActionTypeCompetitorComparison,
		types.ActionTypeInternalNote,
		types.ActionTypeStrategy,
	}
}

// GenerateActions 根据成本压力信号生成营销动作
func (g *CostActionGenerator) GenerateActions(signals []types.QualifiedSignal) ([]types.MarketingAction, error) {
	var actions []types.MarketingAction

	// 筛选成本压力信号
	costSignals := []types.QualifiedSignal{}
	for _, signal := range signals {
		if signal.Signal.Type == types.SignalTypeCostPressure {
			costSignals = append(costSignals, signal)
		}
	}

	if len(costSignals) == 0 {
		return actions, nil
	}

	// 生成动作（全部为内部建议，不自动外发）
	for _, signal := range costSignals {
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

		// 生成技术文章（内部策略建议）
		if signal.Signal.Strength == types.SignalStrengthHigh {
			technicalPost := g.generateTechnicalPost(signal, priority)
			actions = append(actions, technicalPost)
		}

		// 生成竞品对比（内部策略建议）
		if signal.Signal.Strength >= types.SignalStrengthMedium {
			competitorComparison := g.generateCompetitorComparison(signal, priority)
			actions = append(actions, competitorComparison)
		}
	}

	return actions, nil
}

// generateShortResponse 生成短内容回应（内部建议）
func (g *CostActionGenerator) generateShortResponse(signal types.QualifiedSignal, priority int) types.MarketingAction {
	content := "【内部动作建议】成本压力信号响应：用户对AI使用成本表示担忧，建议提供85%成本节省的案例和Token Bridge的API接入方案。"

	return types.MarketingAction{
		ID:          generateActionID(),
		Type:        types.ActionTypeInternalNote,
		Channel:     types.ChannelInternal,
		Title:       "成本压力信号响应建议",
		Content:     content,
		TargetAudience: "Cost-sensitive developers",
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

// generateTechnicalPost 生成技术文章（内部策略建议）
func (g *CostActionGenerator) generateTechnicalPost(signal types.QualifiedSignal, priority int) types.MarketingAction {
	title := "【策略建议】技术内容创作：成本优化主题"
	content := `【内部策略建议】

基于检测到的高优先级成本压力信号，建议创作一篇技术深度文章，主题为"如何通过API接入减少85%的AI使用成本"。

文章要点：
1. 订阅制vs直接API接入的成本对比
2. Token Bridge的多模型路由优势
3. 实际案例和具体配置步骤
4. 成本监控和优化建议

目标受众：技术开发者和成本决策者
推荐发布渠道：Hacker News、Reddit编程社区`

	return types.MarketingAction{
		ID:          generateActionID(),
		Type:        types.ActionTypeStrategy,
		Channel:     types.ChannelInternal,
		Title:       title,
		Content:     content,
		TargetAudience: "Technical developers",
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

// generateCompetitorComparison 生成竞品对比（内部策略建议）
func (g *CostActionGenerator) generateCompetitorComparison(signal types.QualifiedSignal, priority int) types.MarketingAction {
	title := "【策略建议】竞品对比内容"
	content := `【内部策略建议】

基于检测到的成本压力信号，建议创建一份竞品对比分析，重点突出API vs订阅的成本差异。

对比要点：
1. 订阅制成本：$400/月（有限使用）
2. 直接API成本：$47/月（相同使用量）
3. Token Bridge的优势：统一API、多模型支持、成本透明
4. 迁移步骤和注意事项

目标受众：成本意识强的决策者
推荐发布渠道：LinkedIn、Reddit商业社区`

	return types.MarketingAction{
		ID:          generateActionID(),
		Type:        types.ActionTypeStrategy,
		Channel:     types.ChannelInternal,
		Title:       title,
		Content:     content,
		TargetAudience: "Cost-conscious decision makers",
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

// generateActionID 生成动作ID
func generateActionID() string {
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
