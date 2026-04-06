// Package processor 情报处理器
package processor

import (
	"context"
	"log"
	"time"

	"token-bridge-crawler/internal/core"
	"token-bridge-crawler/internal/rules"
)

// Action 处理动作
type Action string

const (
	ActionKeep      Action = "keep"      // 保留入库
	ActionDiscard   Action = "discard"   // 丢弃（噪声）
	ActionEscalate  Action = "escalate"  // 提升优先级（高质量信号）
	ActionHighlight Action = "highlight" // 高亮显示
)

// ProcessResult 处理结果
type ProcessResult struct {
	Item          core.IntelItem `json:"item"`
	IsNoise       bool           `json:"is_noise"`
	QualityScore  float64        `json:"quality_score"`
	PainScore     float64        `json:"pain_score"`    // 痛点强度 0-100
	CustomerTier  string         `json:"customer_tier"` // S/A/B/C
	SignalType    string         `json:"signal_type"`   // 信号类型
	Action        Action         `json:"action"`        // 处理动作
	MatchedRules  []string       `json:"matched_rules"` // 命中的规则
	ProcessedAt   time.Time      `json:"processed_at"`
	ProcessorDeps ProcessorDeps  `json:"processor_deps"` // 处理详情
}

// ProcessorDeps 处理详情
type ProcessorDeps struct {
	RuleResult      RuleResultInfo      `json:"rule_result"`
	SentimentResult SentimentResultInfo `json:"sentiment_result"`
	ProfilerResult  ProfilerResultInfo  `json:"profiler_result"`
}

// RuleResultInfo 规则处理详情
type RuleResultInfo struct {
	NoiseScore   int      `json:"noise_score"`
	QualityScore float64  `json:"quality_score"`
	MatchedRules []string `json:"matched_rules"`
}

// SentimentResultInfo 情感分析详情
type SentimentResultInfo struct {
	PainScore  float64 `json:"pain_score"`
	Sentiment  string  `json:"sentiment"` // positive/negative/neutral
	Confidence float64 `json:"confidence"`
}

// ProfilerResultInfo 用户画像详情
type ProfilerResultInfo struct {
	CustomerTier string            `json:"customer_tier"`
	UserType     string            `json:"user_type"`
	Karma        int               `json:"karma"`
	Company      string            `json:"company"`
	SocialLinks  map[string]string `json:"social_links"`
}

// Processor 处理器接口
type Processor interface {
	// Process 处理单个情报项
	Process(ctx context.Context, item core.IntelItem) ProcessResult

	// ProcessBatch 批量处理
	ProcessBatch(ctx context.Context, items []core.IntelItem) []ProcessResult
}

// IntelProcessor 情报处理器实现
type IntelProcessor struct {
	ruleEngine    rules.RuleEngine
	sentiment     SentimentAnalyzer
	profiler      Profiler
	qualityScorer *QualityScorer
	config        ProcessorConfig
}

// ProcessorConfig 处理器配置
type ProcessorConfig struct {
	// 噪声阈值
	NoiseThreshold int `yaml:"noise_threshold" json:"noise_threshold"`

	// 高质量信号阈值
	HighQualityThreshold float64 `yaml:"high_quality_threshold" json:"high_quality_threshold"`

	// 高痛点阈值
	HighPainThreshold float64 `yaml:"high_pain_threshold" json:"high_pain_threshold"`

	// 是否启用情感分析
	EnableSentiment bool `yaml:"enable_sentiment" json:"enable_sentiment"`

	// 是否启用用户画像
	EnableProfiler bool `yaml:"enable_profiler" json:"enable_profiler"`
}

// DefaultProcessorConfig 默认配置
func DefaultProcessorConfig() ProcessorConfig {
	return ProcessorConfig{
		NoiseThreshold:       5,
		HighQualityThreshold: 70.0,
		HighPainThreshold:    60.0,
		EnableSentiment:      true,
		EnableProfiler:       true,
	}
}

// SentimentAnalyzer 情感分析接口
type SentimentAnalyzer interface {
	Analyze(text string) SentimentResult
}

// SentimentResult 情感分析结果
type SentimentResult struct {
	Sentiment  string  `json:"sentiment"`  // positive/negative/neutral
	PainScore  float64 `json:"pain_score"` // 0-100
	Confidence float64 `json:"confidence"` // 0-1
}

// Profiler 用户画像接口
type Profiler interface {
	Profile(ctx context.Context, item core.IntelItem) ProfileResult
}

// ProfileResult 用户画像结果
type ProfileResult struct {
	CustomerTier string            `json:"customer_tier"`
	UserType     string            `json:"user_type"`
	Karma        int               `json:"karma"`
	Company      string            `json:"company"`
	SocialLinks  map[string]string `json:"social_links"`
}

// NewIntelProcessor 创建情报处理器
func NewIntelProcessor(
	ruleEngine rules.RuleEngine,
	sentiment SentimentAnalyzer,
	profiler Profiler,
	config ProcessorConfig,
) *IntelProcessor {
	if config.NoiseThreshold == 0 {
		config = DefaultProcessorConfig()
	}

	return &IntelProcessor{
		ruleEngine:    ruleEngine,
		sentiment:     sentiment,
		profiler:      profiler,
		qualityScorer: NewQualityScorer(),
		config:        config,
	}
}

// NewDefaultProcessor 创建默认处理器（仅规则引擎）
func NewDefaultProcessor(ruleEngine rules.RuleEngine) *IntelProcessor {
	return NewIntelProcessor(ruleEngine, nil, nil, DefaultProcessorConfig())
}

// Process 处理单个情报项
func (p *IntelProcessor) Process(ctx context.Context, item core.IntelItem) ProcessResult {
	start := time.Now()
	result := ProcessResult{
		Item:          item,
		ProcessedAt:   start,
		ProcessorDeps: ProcessorDeps{},
	}

	// Step 1: 规则引擎评估
	ruleResult := p.ruleEngine.Evaluate(item)
	result.IsNoise = ruleResult.IsNoise
	result.QualityScore = ruleResult.QualityScore
	result.SignalType = ruleResult.SignalType
	result.MatchedRules = ruleResult.MatchedRules
	result.ProcessorDeps.RuleResult = RuleResultInfo{
		NoiseScore:   ruleResult.NoiseScore,
		QualityScore: ruleResult.QualityScore,
		MatchedRules: ruleResult.MatchedRules,
	}

	// 如果是噪声，直接返回丢弃动作
	if result.IsNoise {
		result.Action = ActionDiscard
		log.Printf("[Processor] Item discarded as noise: %s (score=%d)", item.ID, ruleResult.NoiseScore)
		return result
	}

	// Step 2: 情感分析（如果启用）
	if p.sentiment != nil && p.config.EnableSentiment {
		sentimentResult := p.sentiment.Analyze(item.Title + " " + item.Content)
		result.PainScore = sentimentResult.PainScore
		result.ProcessorDeps.SentimentResult = SentimentResultInfo{
			PainScore:  sentimentResult.PainScore,
			Sentiment:  sentimentResult.Sentiment,
			Confidence: sentimentResult.Confidence,
		}
	} else {
		// 使用规则引擎的结果估算痛点分数
		result.PainScore = p.estimatePainScore(ruleResult, item)
	}

	// Step 3: 用户画像（如果启用）
	if p.profiler != nil && p.config.EnableProfiler {
		profileResult := p.profiler.Profile(ctx, item)
		result.CustomerTier = profileResult.CustomerTier
		result.ProcessorDeps.ProfilerResult = ProfilerResultInfo{
			CustomerTier: profileResult.CustomerTier,
			UserType:     profileResult.UserType,
			Karma:        profileResult.Karma,
			Company:      profileResult.Company,
			SocialLinks:  profileResult.SocialLinks,
		}
	} else {
		// 默认客户等级
		result.CustomerTier = "C"
	}

	// Step 4: 决定处理动作
	result.Action = p.decideAction(result)

	// 记录处理耗时
	elapsed := time.Since(start)
	log.Printf("[Processor] Processed item %s: action=%s, quality=%.1f, pain=%.1f, tier=%s, elapsed=%v",
		item.ID, result.Action, result.QualityScore, result.PainScore, result.CustomerTier, elapsed)

	return result
}

// ProcessBatch 批量处理
func (p *IntelProcessor) ProcessBatch(ctx context.Context, items []core.IntelItem) []ProcessResult {
	results := make([]ProcessResult, len(items))
	for i, item := range items {
		results[i] = p.Process(ctx, item)
	}
	return results
}

// estimatePainScore 从规则结果估算痛点分数
func (p *IntelProcessor) estimatePainScore(ruleResult rules.RuleResult, item core.IntelItem) float64 {
	// 基于信号类型和命中规则估算
	score := 0.0

	// 信号类型影响
	switch ruleResult.SignalType {
	case string(rules.SignalTypeMigration):
		score += 80
	case string(rules.SignalTypeCostPressure):
		score += 70
	case string(rules.SignalTypeUserPain):
		score += 60
	case string(rules.SignalTypeFeature):
		score += 50
	case string(rules.SignalTypeCompetitor):
		score += 40
	default:
		score += 30
	}

	// 命中的信号规则数量
	signalCount := 0
	for _, name := range ruleResult.MatchedRules {
		for _, rule := range p.qualityScorer.signalKeywords {
			if name == rule {
				signalCount++
			}
		}
	}
	score += float64(signalCount) * 10

	// 限制在 0-100 范围
	if score > 100 {
		score = 100
	}

	return score
}

// decideAction 决定处理动作
func (p *IntelProcessor) decideAction(result ProcessResult) Action {
	// 高质量 + 高痛点 + 高价值客户 = 提升
	if result.QualityScore >= p.config.HighQualityThreshold &&
		result.PainScore >= p.config.HighPainThreshold &&
		(result.CustomerTier == "S" || result.CustomerTier == "A") {
		return ActionEscalate
	}

	// 高质量 + 高痛点 = 高亮
	if result.QualityScore >= p.config.HighQualityThreshold &&
		result.PainScore >= p.config.HighPainThreshold {
		return ActionHighlight
	}

	// 默认保留
	return ActionKeep
}

// FilterNoise 过滤噪声
func (p *IntelProcessor) FilterNoise(results []ProcessResult) []ProcessResult {
	filtered := make([]ProcessResult, 0, len(results))
	for _, r := range results {
		if !r.IsNoise {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// GetStats 获取处理统计
func (p *IntelProcessor) GetStats(results []ProcessResult) ProcessStats {
	stats := ProcessStats{
		Total: len(results),
	}

	for _, r := range results {
		if r.IsNoise {
			stats.Noise++
		} else {
			stats.Signal++
		}

		switch r.Action {
		case ActionKeep:
			stats.Keep++
		case ActionDiscard:
			stats.Discard++
		case ActionEscalate:
			stats.Escalate++
		case ActionHighlight:
			stats.Highlight++
		}

		stats.QualitySum += r.QualityScore
		stats.PainSum += r.PainScore

		stats.TierCounts[r.CustomerTier]++
		stats.SignalCounts[r.SignalType]++
	}

	if stats.Total > 0 {
		stats.AvgQuality = stats.QualitySum / float64(stats.Total)
		stats.AvgPain = stats.PainSum / float64(stats.Total)
		stats.NoiseRate = float64(stats.Noise) / float64(stats.Total) * 100
		stats.SignalRate = float64(stats.Signal) / float64(stats.Total) * 100
	}

	return stats
}

// ProcessStats 处理统计
type ProcessStats struct {
	Total        int            `json:"total"`
	Noise        int            `json:"noise"`
	Signal       int            `json:"signal"`
	Keep         int            `json:"keep"`
	Discard      int            `json:"discard"`
	Escalate     int            `json:"escalate"`
	Highlight    int            `json:"highlight"`
	QualitySum   float64        `json:"quality_sum"`
	PainSum      float64        `json:"pain_sum"`
	AvgQuality   float64        `json:"avg_quality"`
	AvgPain      float64        `json:"avg_pain"`
	NoiseRate    float64        `json:"noise_rate"`
	SignalRate   float64        `json:"signal_rate"`
	TierCounts   map[string]int `json:"tier_counts"`
	SignalCounts map[string]int `json:"signal_counts"`
}
