// Package marketing 提供营销信号模型和动作触发规则
package marketing

import (
	"fmt"
	"time"

	"token-bridge-crawler/internal/core"
	"token-bridge-crawler/internal/marketing/detectors"
	"token-bridge-crawler/internal/marketing/generators"
	"token-bridge-crawler/internal/marketing/types"
)

// SignalDetector 信号检测器接口
type SignalDetector interface {
	DetectFromIntel(item core.IntelItem) ([]types.CustomerSignal, error)
	GetSupportedTypes() []types.SignalType
}

// SignalQualifier 信号资格评估器接口
type SignalQualifier interface {
	QualifySignal(signal types.CustomerSignal) (types.QualifiedSignal, error)
	QualifySignals(signals []types.CustomerSignal) ([]types.QualifiedSignal, error)
}

// ActionGenerator 动作生成器接口
type ActionGenerator interface {
	GenerateActions(signals []types.QualifiedSignal) ([]types.MarketingAction, error)
	GetSupportedActions() []types.ActionType
}

// SignalModel 信号模型
type SignalModel struct {
	Detectors  []SignalDetector
	Qualifier  SignalQualifier
	Generators []ActionGenerator
}

// NewSignalModel 创建信号模型
func NewSignalModel() *SignalModel {
	return &SignalModel{
		Detectors: []SignalDetector{
			detectors.NewCostPressureDetector(),
			detectors.NewConfigFrictionDetector(),
			detectors.NewToolFragmentationDetector(),
			detectors.NewGovernanceStartDetector(),
			detectors.NewMigrationIntentDetector(),
			detectors.NewGeneralInterestDetector(),
		},
		Qualifier: NewDefaultSignalQualifier(),
		Generators: []ActionGenerator{
			generators.NewCostActionGenerator(),
			generators.NewConfigActionGenerator(),
			generators.NewToolActionGenerator(),
			generators.NewGovernanceActionGenerator(),
			generators.NewMigrationActionGenerator(),
		},
	}
}

// DefaultSignalQualifier 默认信号资格评估器
type DefaultSignalQualifier struct{}

// NewDefaultSignalQualifier 创建默认信号资格评估器
func NewDefaultSignalQualifier() SignalQualifier {
	return &DefaultSignalQualifier{}
}

// QualifySignal 评估单个信号
func (q *DefaultSignalQualifier) QualifySignal(signal types.CustomerSignal) (types.QualifiedSignal, error) {
	// 计算资格分数
	score := q.calculateScore(signal)
	
	// 确定资格状态
	status := types.QualificationStatusQualified
	if score < 30 {
		status = types.QualificationStatusUnqualified
	} else if score < 60 {
		status = types.QualificationStatusPending
	}
	
	// 确定客户阶段
	stage := q.determineCustomerStage(signal)
	
	// 生成评估理由
	reason := q.generateReason(signal, score, stage)
	
	return types.QualifiedSignal{
		Signal:        signal,
		Status:        status,
		CustomerStage: stage,
		Score:         score,
		Reason:        reason,
		Metadata: map[string]interface{}{
			"signal_strength": signal.Strength,
			"platform":        signal.Platform,
		},
		QualifiedAt: time.Now().UTC(),
	}, nil
}

// QualifySignals 批量评估信号
func (q *DefaultSignalQualifier) QualifySignals(signals []types.CustomerSignal) ([]types.QualifiedSignal, error) {
	var qualifiedSignals []types.QualifiedSignal
	for _, signal := range signals {
		qualified, err := q.QualifySignal(signal)
		if err != nil {
			continue
		}
		qualifiedSignals = append(qualifiedSignals, qualified)
	}
	return qualifiedSignals, nil
}

// calculateScore 计算信号分数
func (q *DefaultSignalQualifier) calculateScore(signal types.CustomerSignal) float64 {
	score := 0.0
	
	// 信号类型权重
	switch signal.Type {
	case types.SignalTypeCostPressure, types.SignalTypeConfigFriction, types.SignalTypeToolFragmentation:
		score += 40
	case types.SignalTypeGovernanceStart, types.SignalTypeMigrationIntent:
		score += 30
	case types.SignalTypeGeneralInterest:
		score += 10
	}
	
	// 信号强度权重
	switch signal.Strength {
	case types.SignalStrengthHigh:
		score += 30
	case types.SignalStrengthMedium:
		score += 20
	case types.SignalStrengthLow:
		score += 10
	}
	
	// 平台权重
	switch signal.Platform {
	case "hacker_news", "indie_hackers":
		score += 20
	case "reddit":
		score += 15
	case "linkedin":
		score += 10
	default:
		score += 5
	}
	
	// 内容质量（简单评估）
	if len(signal.Content) > 100 {
		score += 10
	}
	
	// 确保分数在 0-100 范围内
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	
	return score
}

// determineCustomerStage 确定客户阶段
func (q *DefaultSignalQualifier) determineCustomerStage(signal types.CustomerSignal) types.CustomerStage {
	switch signal.Type {
	case types.SignalTypeGeneralInterest:
		return types.CustomerStageAwareness
	case types.SignalTypeCostPressure, types.SignalTypeConfigFriction, types.SignalTypeToolFragmentation:
		return types.CustomerStageConsideration
	case types.SignalTypeMigrationIntent:
		return types.CustomerStageDecision
	case types.SignalTypeGovernanceStart:
		return types.CustomerStageRetention
	default:
		return types.CustomerStageAwareness
	}
}

// generateReason 生成评估理由
func (q *DefaultSignalQualifier) generateReason(signal types.CustomerSignal, score float64, stage types.CustomerStage) string {
	reason := "信号评估理由："
	
	switch signal.Type {
	case types.SignalTypeCostPressure:
		reason += "成本压力信号，表明用户对AI使用成本敏感，"
	case types.SignalTypeConfigFriction:
		reason += "配置摩擦信号，表明用户在API配置方面遇到困难，"
	case types.SignalTypeToolFragmentation:
		reason += "工具碎片化信号，表明用户使用多个AI工具，"
	case types.SignalTypeGovernanceStart:
		reason += "治理起点信号，表明用户需要团队级别的AI使用管理，"
	case types.SignalTypeMigrationIntent:
		reason += "迁移意愿信号，表明用户正在考虑替代方案，"
	case types.SignalTypeGeneralInterest:
		reason += "泛兴趣信号，表明用户对AI技术有一般兴趣，"
	}
	
	switch signal.Strength {
	case types.SignalStrengthHigh:
		reason += "信号强度高，"
	case types.SignalStrengthMedium:
		reason += "信号强度中等，"
	case types.SignalStrengthLow:
		reason += "信号强度低，"
	}
	
	reason += "来自" + signal.Platform + "平台，"
	reason += "评估分数：" + fmt.Sprintf("%.1f", score) + "，"
	reason += "客户阶段：" + string(stage)
	
	return reason
}

// ProcessIntel 处理情报项，检测信号并生成动作
func (m *SignalModel) ProcessIntel(item core.IntelItem) ([]types.CustomerSignal, []types.MarketingAction, error) {
	// 检测信号
	var allSignals []types.CustomerSignal
	for _, detector := range m.Detectors {
		signals, err := detector.DetectFromIntel(item)
		if err != nil {
			continue
		}
		allSignals = append(allSignals, signals...)
	}

	// 资格评估
	qualifiedSignals, err := m.Qualifier.QualifySignals(allSignals)
	if err != nil {
		return allSignals, nil, err
	}

	// 过滤未资格化的信号
	var validSignals []types.QualifiedSignal
	for _, signal := range qualifiedSignals {
		if signal.Status == types.QualificationStatusQualified {
			validSignals = append(validSignals, signal)
		}
	}

	// 生成动作
	var allActions []types.MarketingAction
	if len(validSignals) > 0 {
		for _, generator := range m.Generators {
			actions, err := generator.GenerateActions(validSignals)
			if err != nil {
				continue
			}
			allActions = append(allActions, actions...)
		}
	}

	return allSignals, allActions, nil
}

// GetSignalPriority 获取信号优先级
func GetSignalPriority(signalType types.SignalType) int {
	switch signalType {
	case types.SignalTypeCostPressure, types.SignalTypeConfigFriction, types.SignalTypeToolFragmentation:
		return 5 // P0
	case types.SignalTypeGovernanceStart, types.SignalTypeMigrationIntent:
		return 3 // P1
	case types.SignalTypeGeneralInterest:
		return 1 // P3
	default:
		return 2
	}
}

// GetChannelPriority 获取渠道优先级
func GetChannelPriority(channel types.Channel) int {
	switch channel {
	case types.ChannelHackerNews, types.ChannelReddit, types.ChannelIndieHackers:
		return 5 // P0
	case types.ChannelProductHunt, types.ChannelLinkedIn:
		return 3 // P1
	case types.ChannelTwitter:
		return 2 // P2
	default:
		return 1
	}
}