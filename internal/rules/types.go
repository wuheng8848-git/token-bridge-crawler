// Package rules 提供噪声识别与质量评分的规则引擎
package rules

import (
	"time"

	"token-bridge-crawler/internal/core"
)

// RuleType 规则类型
type RuleType string

const (
	RuleTypeKeyword  RuleType = "keyword"   // 关键词匹配
	RuleTypeLength   RuleType = "length"    // 长度检查
	RuleTypeRegex    RuleType = "regex"     // 正则匹配
	RuleTypeUserType RuleType = "user_type" // 用户类型判断
)

// Rule 规则定义
type Rule struct {
	ID        int64     `json:"id"`
	RuleType  RuleType  `json:"rule_type"`
	RuleName  string    `json:"rule_name"`
	RuleValue string    `json:"rule_value"` // JSON 或字符串
	Weight    int       `json:"weight"`     // 正数=噪声特征，负数=信号特征
	IsActive  bool      `json:"is_active"`
	Priority  int       `json:"priority"` // 执行优先级，越大越先执行
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RuleResult 规则评估结果
type RuleResult struct {
	IsNoise      bool     `json:"is_noise"`      // 是否为噪声
	NoiseScore   int      `json:"noise_score"`   // 噪声分数（正数=噪声）
	QualityScore float64  `json:"quality_score"` // 质量分数 0-100
	MatchedRules []string `json:"matched_rules"` // 命中的规则名称
	SignalType   string   `json:"signal_type"`   // 信号类型（如果非噪声）
}

// RuleEngine 规则引擎接口
type RuleEngine interface {
	// Evaluate 评估情报项
	Evaluate(item core.IntelItem) RuleResult

	// ReloadRules 重新加载规则
	ReloadRules() error

	// AddRule 添加规则
	AddRule(rule Rule) error

	// RemoveRule 移除规则
	RemoveRule(ruleID int64) error

	// ListRules 列出所有规则
	ListRules() ([]Rule, error)
}

// KeywordRuleValue 关键词规则值
type KeywordRuleValue struct {
	Keywords []string `json:"keywords"` // 关键词列表
	Mode     string   `json:"mode"`     // "any" 或 "all"
}

// LengthRuleValue 长度规则值
type LengthRuleValue struct {
	Min int `json:"min"` // 最小长度
	Max int `json:"max"` // 最大长度（0 表示无限制）
}

// RegexRuleValue 正则规则值
type RegexRuleValue struct {
	Pattern string `json:"pattern"` // 正则表达式
	Flags   string `json:"flags"`   // 正则标志（如 "i" 表示忽略大小写）
}

// UserTypeRuleValue 用户类型规则值
type UserTypeRuleValue struct {
	UserTypes []string `json:"user_types"` // 用户类型列表
	Exclude   bool     `json:"exclude"`    // true=排除这些类型，false=仅包含这些类型
}

// SignalType 信号类型
type SignalType string

const (
	SignalTypeUserPain     SignalType = "user_pain"     // 用户痛点
	SignalTypeMigration    SignalType = "migration"     // 迁移意愿
	SignalTypeFeature      SignalType = "feature"       // 功能需求
	SignalTypeCompetitor   SignalType = "competitor"    // 竞品动态
	SignalTypeCostPressure SignalType = "cost_pressure" // 成本压力
	SignalTypeNoise        SignalType = "noise"         // 噪声
)

// WeightConstants 权重常量
const (
	// 噪声权重（正数）
	WeightSpam       = 10 // 营销推广
	WeightShort      = 5  // 内容过短
	WeightIrrelevant = 8  // 无关内容
	WeightFragmented = 6  // 碎片信息

	// 信号权重（负数，绝对值越大信号越强）
	WeightPainPoint    = -10 // 痛点
	WeightMigration    = -15 // 迁移意愿
	WeightFeature      = -8  // 功能需求
	WeightCompetitor   = -12 // 竞品动态
	WeightCostPressure = -10 // 成本压力
)

// QualityScoreParams 质量评分参数
type QualityScoreParams struct {
	KeywordMatch   float64 // 关键词匹配度 0-100
	InfluenceScore float64 // 影响力分数 0-100
	Completeness   float64 // 完整度 0-100
	Timeliness     float64 // 时效性 0-100
	Relevance      float64 // 相关性 0-100
}

// CalculateQualityScore 计算质量分数
// 公式：质量分 = 关键词30% + 影响力20% + 完整度20% + 时效性15% + 相关性15%
func CalculateQualityScore(params QualityScoreParams) float64 {
	return params.KeywordMatch*0.30 +
		params.InfluenceScore*0.20 +
		params.Completeness*0.20 +
		params.Timeliness*0.15 +
		params.Relevance*0.15
}

// NoiseThreshold 噪声阈值
const NoiseThreshold = 5

// IsNoiseByScore 根据分数判断是否为噪声
func IsNoiseByScore(noiseScore int) bool {
	return noiseScore >= NoiseThreshold
}
