// Package types 提供营销相关的核心类型定义
package types

import "time"

// SignalType 信号类型
type SignalType string

const (
	// 高优先级信号 (P0)
	SignalTypeCostPressure      SignalType = "cost_pressure"      // 成本压力信号
	SignalTypeConfigFriction    SignalType = "config_friction"    // 配置摩擦信号
	SignalTypeToolFragmentation SignalType = "tool_fragmentation" // 工具碎片化信号

	// 中优先级信号 (P1)
	SignalTypeGovernanceStart SignalType = "governance_start" // 治理起点信号
	SignalTypeMigrationIntent SignalType = "migration_intent" // 迁移意愿信号

	// 低优先级信号 (P3)
	SignalTypeGeneralInterest SignalType = "general_interest" // 泛兴趣信号
)

// SignalStrength 信号强度
type SignalStrength int

const (
	SignalStrengthLow    SignalStrength = 1
	SignalStrengthMedium SignalStrength = 2
	SignalStrengthHigh   SignalStrength = 3
)

// CustomerStage 客户阶段
type CustomerStage string

const (
	CustomerStageAwareness     CustomerStage = "awareness"     // 认知阶段
	CustomerStageConsideration CustomerStage = "consideration" // 考虑阶段
	CustomerStageDecision      CustomerStage = "decision"      // 决策阶段
	CustomerStageRetention     CustomerStage = "retention"     // 留存阶段
)

// QualificationStatus 资格评估状态
type QualificationStatus string

const (
	QualificationStatusQualified   QualificationStatus = "qualified"   // 已资格化
	QualificationStatusUnqualified QualificationStatus = "unqualified" // 未资格化
	QualificationStatusPending     QualificationStatus = "pending"     // 待评估
)

// CustomerSignal 客户信号
type CustomerSignal struct {
	ID           string                 `json:"id"`
	Type         SignalType             `json:"type"`
	Strength     SignalStrength         `json:"strength"`
	Content      string                 `json:"content"`
	Platform     string                 `json:"platform"`
	Author       string                 `json:"author"`
	URL          string                 `json:"url"`
	Metadata     map[string]interface{} `json:"metadata"`
	DetectedAt   time.Time              `json:"detected_at"`
	RelatedIntel string                 `json:"related_intel,omitempty"`
}

// QualifiedSignal 已资格化的信号
type QualifiedSignal struct {
	Signal        CustomerSignal         `json:"signal"`
	Status        QualificationStatus    `json:"status"`
	CustomerStage CustomerStage          `json:"customer_stage"`
	Score         float64                `json:"score"` // 0-100
	Reason        string                 `json:"reason"`
	Metadata      map[string]interface{} `json:"metadata"`
	QualifiedAt   time.Time              `json:"qualified_at"`
}

// ActionType 动作类型
type ActionType string

const (
	ActionTypeShortResponse        ActionType = "short_response"        // 短内容回应
	ActionTypeTechnicalPost        ActionType = "technical_post"        // 长文/技术帖
	ActionTypeSetupGuide           ActionType = "setup_guide"           // 配置教程
	ActionTypeCompetitorComparison ActionType = "competitor_comparison" // 竞品对比
	ActionTypeFollowUp             ActionType = "follow_up"             // 后续触达
	ActionTypeInternalNote         ActionType = "internal_note"         // 内部备注
	ActionTypeResearch             ActionType = "research"              // 调研建议
	ActionTypeStrategy             ActionType = "strategy"              // 策略建议
)

// Channel 渠道
type Channel string

const (
	ChannelHackerNews   Channel = "hacker_news"
	ChannelReddit       Channel = "reddit"
	ChannelIndieHackers Channel = "indie_hackers"
	ChannelProductHunt  Channel = "product_hunt"
	ChannelLinkedIn     Channel = "linkedin"
	ChannelTwitter      Channel = "twitter"
	ChannelInternal     Channel = "internal" // 内部渠道
)

// MarketingAction 营销动作
type MarketingAction struct {
	ID             string                 `json:"id"`
	Type           ActionType             `json:"type"`
	Channel        Channel                `json:"channel"`
	Title          string                 `json:"title"`
	Content        string                 `json:"content"`
	TemplateID     string                 `json:"template_id,omitempty"`
	TargetAudience string                 `json:"target_audience"`
	Priority       int                    `json:"priority"` // 1-5, 5 highest
	SignalIDs      []string               `json:"signal_ids"`
	Metadata       map[string]interface{} `json:"metadata"`
	CreatedAt      time.Time              `json:"created_at"`
	ScheduledAt    *time.Time             `json:"scheduled_at,omitempty"`
	Status         string                 `json:"status"`       // pending, executed, failed, draft
	AutoExecute    bool                   `json:"auto_execute"` // 是否自动执行
	CustomerStage  CustomerStage          `json:"customer_stage"`
	QualifiedScore float64                `json:"qualified_score"`
}
