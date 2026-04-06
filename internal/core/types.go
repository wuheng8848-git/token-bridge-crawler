// Package core 提供情报系统的核心类型定义
package core

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// IntelType 情报类型
type IntelType string

const (
	// 供给侧信号
	IntelTypePrice   IntelType = "price"   // 价格情报
	IntelTypeAPIDoc  IntelType = "api_doc" // API文档变更
	IntelTypeProduct IntelType = "product" // 产品发布
	IntelTypePolicy  IntelType = "policy"  // 政策变更

	// 需求侧信号
	IntelTypeCommunity IntelType = "community" // 社区讨论
	IntelTypeNews      IntelType = "news"      // 行业新闻
	IntelTypeUserPain  IntelType = "user_pain" // 用户痛点

	// 入口侧信号
	IntelTypeToolEcosystem IntelType = "tool_ecosystem" // 工具生态
	IntelTypeIntegration   IntelType = "integration"    // 集成机会

	// 自有经营信号
	IntelTypeUserAcquisition IntelType = "user_acquisition" // 用户获取
	IntelTypeConversion      IntelType = "conversion"       // 转化情况
	IntelTypeUsagePattern    IntelType = "usage_pattern"    // 使用模式
)

// IntelItem 统一情报项结构
type IntelItem struct {
	// 核心字段
	ID        string    `json:"id" db:"id"`
	IntelType IntelType `json:"intel_type" db:"intel_type"`
	Source    string    `json:"source" db:"source"`       // 来源标识，如 "openai", "anthropic"
	SourceID  string    `json:"source_id" db:"source_id"` // 源系统ID

	// 内容字段
	Title   string `json:"title" db:"title"`     // 标题
	Content string `json:"content" db:"content"` // 内容/摘要
	URL     string `json:"url" db:"url"`         // 原始链接

	// 元数据（JSONB扩展）
	Metadata Metadata `json:"metadata" db:"metadata"` // 类型特定的扩展数据

	// 时间字段
	CapturedAt  time.Time  `json:"captured_at" db:"captured_at"`
	PublishedAt *time.Time `json:"published_at,omitempty" db:"published_at"`

	// 处理状态
	Status    IntelStatus `json:"status" db:"status"` // 'new', 'processed', 'alerted', 'ignored'
	CreatedAt time.Time   `json:"created_at" db:"created_at"`

	// 质量评分字段（处理层填充）
	QualityScore *float64 `json:"quality_score,omitempty" db:"quality_score"` // 质量分数 0-100
	IsNoise      *bool    `json:"is_noise,omitempty" db:"is_noise"`           // 是否噪声
	FilterReason *string  `json:"filter_reason,omitempty" db:"filter_reason"` // 过滤原因
	CustomerTier *string  `json:"customer_tier,omitempty" db:"customer_tier"` // 客户等级 S/A/B/C
	SignalType   *string  `json:"signal_type,omitempty" db:"signal_type"`     // 信号类型
	PainScore    *float64 `json:"pain_score,omitempty" db:"pain_score"`       // 痛点评分 0-100
}

// IntelStatus 情报处理状态
type IntelStatus string

const (
	IntelStatusNew       IntelStatus = "new"       // 新采集
	IntelStatusProcessed IntelStatus = "processed" // 已处理
	IntelStatusAlerted   IntelStatus = "alerted"   // 已告警
	IntelStatusIgnored   IntelStatus = "ignored"   // 已忽略
)

// Metadata 情报元数据（使用JSONB存储）
type Metadata map[string]interface{}

// Scan 实现sql.Scanner接口
func (m *Metadata) Scan(value interface{}) error {
	if value == nil {
		*m = make(Metadata)
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, m)
	case string:
		return json.Unmarshal([]byte(v), m)
	default:
		*m = make(Metadata)
		return nil
	}
}

// PriceMetadata 价格情报的元数据结构
type PriceMetadata struct {
	ModelCode           string  `json:"model_code"`
	ModelName           string  `json:"model_name"`
	InputPrice          float64 `json:"input_price"`
	OutputPrice         float64 `json:"output_price"`
	Currency            string  `json:"currency"`
	PriceType           string  `json:"price_type"`            // "vendor_list_price"
	SchemaVersion       string  `json:"schema_version"`        // "v1"
	ChangeType          string  `json:"change_type,omitempty"` // "new", "updated", "unchanged"
	PreviousInputPrice  float64 `json:"previous_input_price,omitempty"`
	PreviousOutputPrice float64 `json:"previous_output_price,omitempty"`
	ChangePercent       float64 `json:"change_percent,omitempty"`
}

// APIDocMetadata API文档情报的元数据结构
type APIDocMetadata struct {
	ChangeType        string   `json:"change_type"` // "deprecated", "new_feature", "breaking_change"
	AffectedEndpoints []string `json:"affected_endpoints"`
	Severity          string   `json:"severity"` // "critical", "high", "medium", "low"
	Version           string   `json:"version,omitempty"`
	MigrationGuide    string   `json:"migration_guide,omitempty"`
}

// CommunityMetadata 社区情报的元数据结构
type CommunityMetadata struct {
	Platform      string `json:"platform"` // "hackernews", "reddit"
	Points        int    `json:"points"`   // 点赞数
	CommentsCount int    `json:"comments_count"`
	Sentiment     string `json:"sentiment"` // "positive", "negative", "neutral"
	Author        string `json:"author"`
	Subreddit     string `json:"subreddit,omitempty"` // Reddit专用
}

// NewsMetadata 新闻情报的元数据结构
type NewsMetadata struct {
	Author     string   `json:"author"`
	Tags       []string `json:"tags"`
	SourceName string   `json:"source_name"` // TechCrunch, The Verge等
}

// PolicyMetadata 政策变更的元数据结构
type PolicyMetadata struct {
	PolicyType      string   `json:"policy_type"` // "pricing", "rate_limit", "terms_of_service"
	AffectedRegions []string `json:"affected_regions"`
	EffectiveDate   string   `json:"effective_date"`
	Severity        string   `json:"severity"` // "critical", "high", "medium", "low"
}

// UserPainMetadata 用户痛点的元数据结构
type UserPainMetadata struct {
	PainType        string   `json:"pain_type"` // "cost", "complexity", "compliance", "payment"
	Frequency       int      `json:"frequency"` // 提及频率
	Sentiment       string   `json:"sentiment"` // "negative", "very_negative"
	RelatedProducts []string `json:"related_products"`
}

// ToolEcosystemMetadata 工具生态的元数据结构
type ToolEcosystemMetadata struct {
	ToolName        string   `json:"tool_name"`        // Cursor, Claude Code, Aider等
	Category        string   `json:"category"`         // "code_editor", "chat_interface", "automation"
	GrowthRate      float64  `json:"growth_rate"`      // 增长率
	UserBase        int      `json:"user_base"`        // 用户基数
	IntegrationAPIs []string `json:"integration_apis"` // 可集成的API
	OpennessLevel   string   `json:"openness_level"`   // "open", "semi-open", "closed"
}

// IntegrationMetadata 集成机会的元数据结构
type IntegrationMetadata struct {
	TargetTool      string   `json:"target_tool"`
	IntegrationType string   `json:"integration_type"` // "plugin", "extension", "API"
	Difficulty      string   `json:"difficulty"`       // "easy", "medium", "hard"
	EstimatedROI    float64  `json:"estimated_roi"`
	RequiredSkills  []string `json:"required_skills"`
}

// UserAcquisitionMetadata 用户获取的元数据结构
type UserAcquisitionMetadata struct {
	Channel            string  `json:"channel"` // "content", "social", "referral", "paid"
	Source             string  `json:"source"`  // 具体来源
	ConversionRate     float64 `json:"conversion_rate"`
	CostPerAcquisition float64 `json:"cost_per_acquisition"`
	LifetimeValue      float64 `json:"lifetime_value"`
}

// ConversionMetadata 转化情况的元数据结构
type ConversionMetadata struct {
	Stage            string  `json:"stage"` // "signup", "configuration", "first_call", "recurring"
	ConversionRate   float64 `json:"conversion_rate"`
	DropOffReason    string  `json:"drop_off_reason,omitempty"`
	ValueProposition string  `json:"value_proposition"` // 价值主张
}

// UsagePatternMetadata 使用模式的元数据结构
type UsagePatternMetadata struct {
	UserSegment   string             `json:"user_segment"`    // "individual", "startup", "enterprise"
	ModelUsage    map[string]float64 `json:"model_usage"`     // 模型使用量
	CallVolume    int                `json:"call_volume"`     // 调用量
	AverageTokens int                `json:"average_tokens"`  // 平均token数
	PeakUsageTime string             `json:"peak_usage_time"` // 峰值使用时间
}

// ToPriceMetadata 将Metadata转换为PriceMetadata
func (m Metadata) ToPriceMetadata() PriceMetadata {
	var pm PriceMetadata
	data, _ := json.Marshal(m)
	_ = json.Unmarshal(data, &pm)
	return pm
}

// ToAPIDocMetadata 将Metadata转换为APIDocMetadata
func (m Metadata) ToAPIDocMetadata() APIDocMetadata {
	var am APIDocMetadata
	data, _ := json.Marshal(m)
	_ = json.Unmarshal(data, &am)
	return am
}

// ToCommunityMetadata 将Metadata转换为CommunityMetadata
func (m Metadata) ToCommunityMetadata() CommunityMetadata {
	var cm CommunityMetadata
	data, _ := json.Marshal(m)
	_ = json.Unmarshal(data, &cm)
	return cm
}

// ToPolicyMetadata 将Metadata转换为PolicyMetadata
func (m Metadata) ToPolicyMetadata() PolicyMetadata {
	var pm PolicyMetadata
	data, _ := json.Marshal(m)
	_ = json.Unmarshal(data, &pm)
	return pm
}

// ToUserPainMetadata 将Metadata转换为UserPainMetadata
func (m Metadata) ToUserPainMetadata() UserPainMetadata {
	var um UserPainMetadata
	data, _ := json.Marshal(m)
	_ = json.Unmarshal(data, &um)
	return um
}

// ToToolEcosystemMetadata 将Metadata转换为ToolEcosystemMetadata
func (m Metadata) ToToolEcosystemMetadata() ToolEcosystemMetadata {
	var tm ToolEcosystemMetadata
	data, _ := json.Marshal(m)
	_ = json.Unmarshal(data, &tm)
	return tm
}

// ToIntegrationMetadata 将Metadata转换为IntegrationMetadata
func (m Metadata) ToIntegrationMetadata() IntegrationMetadata {
	var im IntegrationMetadata
	data, _ := json.Marshal(m)
	_ = json.Unmarshal(data, &im)
	return im
}

// ToUserAcquisitionMetadata 将Metadata转换为UserAcquisitionMetadata
func (m Metadata) ToUserAcquisitionMetadata() UserAcquisitionMetadata {
	var uam UserAcquisitionMetadata
	data, _ := json.Marshal(m)
	_ = json.Unmarshal(data, &uam)
	return uam
}

// ToConversionMetadata 将Metadata转换为ConversionMetadata
func (m Metadata) ToConversionMetadata() ConversionMetadata {
	var cm ConversionMetadata
	data, _ := json.Marshal(m)
	_ = json.Unmarshal(data, &cm)
	return cm
}

// ToUsagePatternMetadata 将Metadata转换为UsagePatternMetadata
func (m Metadata) ToUsagePatternMetadata() UsagePatternMetadata {
	var um UsagePatternMetadata
	data, _ := json.Marshal(m)
	_ = json.Unmarshal(data, &um)
	return um
}

// NewIntelItem 创建新的情报项
func NewIntelItem(intelType IntelType, source string) IntelItem {
	now := time.Now().UTC()
	return IntelItem{
		ID:         generateID(),
		IntelType:  intelType,
		Source:     source,
		Metadata:   make(Metadata),
		CapturedAt: now,
		Status:     IntelStatusNew,
		CreatedAt:  now,
	}
}

// generateID 生成唯一ID（UUID格式）
func generateID() string {
	return uuid.New().String()
}
