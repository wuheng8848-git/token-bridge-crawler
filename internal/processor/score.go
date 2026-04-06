// Package processor 情报处理器
package processor

import (
	"strings"
	"time"

	"token-bridge-crawler/internal/core"
)

// QualityScorer 质量评分器
type QualityScorer struct {
	// 信号关键词（用于估算痛点分数）
	signalKeywords []string
}

// NewQualityScorer 创建质量评分器
func NewQualityScorer() *QualityScorer {
	return &QualityScorer{
		signalKeywords: []string{
			"价格痛点",
			"迁移意愿",
			"功能需求",
			"竞品动态",
			"成本压力",
			"性能问题",
			"质量问题",
		},
	}
}

// ScoreComponents 评分组件
type ScoreComponents struct {
	Relevance    float64 `json:"relevance"`    // 相关性 0-100
	Completeness float64 `json:"completeness"` // 完整度 0-100
	Timeliness   float64 `json:"timeliness"`   // 时效性 0-100
	Influence    float64 `json:"influence"`    // 影响力 0-100
	Engagement   float64 `json:"engagement"`   // 互动度 0-100
}

// Weights 评分权重
type Weights struct {
	Relevance    float64 `json:"relevance"`
	Completeness float64 `json:"completeness"`
	Timeliness   float64 `json:"timeliness"`
	Influence    float64 `json:"influence"`
	Engagement   float64 `json:"engagement"`
}

// DefaultWeights 默认权重
func DefaultWeights() Weights {
	return Weights{
		Relevance:    0.30,
		Completeness: 0.20,
		Timeliness:   0.15,
		Influence:    0.20,
		Engagement:   0.15,
	}
}

// CalculateScore 计算综合质量分数
func (s *QualityScorer) CalculateScore(item core.IntelItem, components ScoreComponents, weights Weights) float64 {
	return components.Relevance*weights.Relevance +
		components.Completeness*weights.Completeness +
		components.Timeliness*weights.Timeliness +
		components.Influence*weights.Influence +
		components.Engagement*weights.Engagement
}

// CalculateRelevance 计算相关性
func (s *QualityScorer) CalculateRelevance(item core.IntelItem, matchedRules []string) float64 {
	score := 0.0
	text := strings.ToLower(item.Title + " " + item.Content)

	// 检查是否与 AI API 相关
	aiKeywords := []string{
		"openai", "gpt", "chatgpt", "api", "llm",
		"claude", "anthropic", "gemini", "llama",
		"mistral", "token", "model", "ai",
	}
	aiMatches := 0
	for _, kw := range aiKeywords {
		if strings.Contains(text, kw) {
			aiMatches++
		}
	}
	score += float64(aiMatches) * 10

	// 命中的信号规则加分
	signalCount := 0
	for _, ruleName := range matchedRules {
		for _, sk := range s.signalKeywords {
			if strings.Contains(ruleName, sk) {
				signalCount++
			}
		}
	}
	score += float64(signalCount) * 20

	// 限制在 0-100
	if score > 100 {
		score = 100
	}
	return score
}

// CalculateCompleteness 计算完整度
func (s *QualityScorer) CalculateCompleteness(item core.IntelItem) float64 {
	score := 0.0

	// 有标题 +20
	if item.Title != "" && len(item.Title) > 5 {
		score += 20
	}

	// 有内容
	if item.Content != "" {
		if len(item.Content) >= 100 {
			score += 40
		} else if len(item.Content) >= 50 {
			score += 30
		} else {
			score += 15
		}
	}

	// 有 URL +15
	if item.URL != "" {
		score += 15
	}

	// 有作者
	if author, ok := item.Metadata["author"].(string); ok && author != "" {
		score += 10
	}

	// 有发布时间
	if item.PublishedAt != nil {
		score += 10
	}

	// 有来源
	if item.Source != "" {
		score += 5
	}

	return score
}

// CalculateTimeliness 计算时效性
func (s *QualityScorer) CalculateTimeliness(item core.IntelItem) float64 {
	if item.PublishedAt == nil {
		return 50 // 无时间信息，给中等分
	}

	age := time.Since(*item.PublishedAt)

	// 1小时内 = 100分
	// 24小时内 = 90分
	// 3天内 = 80分
	// 7天内 = 70分
	// 30天内 = 50分
	// 更久 = 30分

	switch {
	case age < time.Hour:
		return 100
	case age < 24*time.Hour:
		return 90
	case age < 3*24*time.Hour:
		return 80
	case age < 7*24*time.Hour:
		return 70
	case age < 30*24*time.Hour:
		return 50
	default:
		return 30
	}
}

// CalculateInfluence 计算影响力
func (s *QualityScorer) CalculateInfluence(item core.IntelItem) float64 {
	score := 0.0

	// 从元数据获取影响力指标
	if points, ok := item.Metadata["points"].(int); ok {
		// HackerNews points
		score += float64(points) / 10.0
	}
	if points, ok := item.Metadata["points"].(float64); ok {
		score += points / 10.0
	}

	if redditScore, ok := item.Metadata["score"].(int); ok {
		// Reddit score
		score += float64(redditScore) / 10.0
	}
	if scoreFloat, ok := item.Metadata["score"].(float64); ok {
		score += scoreFloat / 10.0
	}

	if karma, ok := item.Metadata["author_karma"].(int); ok {
		// 作者 karma
		score += float64(karma) / 1000.0
	}
	if karma, ok := item.Metadata["author_karma"].(float64); ok {
		score += karma / 1000.0
	}

	// 限制在 0-100
	if score > 100 {
		score = 100
	}

	return score
}

// CalculateEngagement 计算互动度
func (s *QualityScorer) CalculateEngagement(item core.IntelItem) float64 {
	score := 0.0

	// 评论数
	if comments, ok := item.Metadata["num_comments"].(int); ok {
		score += float64(comments) * 5
	}
	if comments, ok := item.Metadata["num_comments"].(float64); ok {
		score += comments * 5
	}

	// 回复数
	if replies, ok := item.Metadata["num_replies"].(int); ok {
		score += float64(replies) * 3
	}
	if replies, ok := item.Metadata["num_replies"].(float64); ok {
		score += replies * 3
	}

	// 限制在 0-100
	if score > 100 {
		score = 100
	}

	return score
}

// CalculateAll 计算所有评分组件
func (s *QualityScorer) CalculateAll(item core.IntelItem, matchedRules []string) ScoreComponents {
	return ScoreComponents{
		Relevance:    s.CalculateRelevance(item, matchedRules),
		Completeness: s.CalculateCompleteness(item),
		Timeliness:   s.CalculateTimeliness(item),
		Influence:    s.CalculateInfluence(item),
		Engagement:   s.CalculateEngagement(item),
	}
}

// Score 计算最终质量分数
func (s *QualityScorer) Score(item core.IntelItem, matchedRules []string) float64 {
	components := s.CalculateAll(item, matchedRules)
	weights := DefaultWeights()
	return s.CalculateScore(item, components, weights)
}
