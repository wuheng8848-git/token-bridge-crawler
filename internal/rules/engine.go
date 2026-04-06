// Package rules 提供噪声识别与质量评分的规则引擎
package rules

import (
	"encoding/json"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"token-bridge-crawler/internal/core"
)

// Engine 规则引擎实现
type Engine struct {
	rules   []Rule
	storage RuleStorage
	mu      sync.RWMutex
}

// RuleStorage 规则存储接口
type RuleStorage interface {
	LoadRules() ([]Rule, error)
	SaveRule(rule *Rule) error
	DeleteRule(ruleID int64) error
}

// NewEngine 创建规则引擎
func NewEngine(storage RuleStorage) (*Engine, error) {
	engine := &Engine{
		storage: storage,
	}

	// 加载初始规则
	if err := engine.ReloadRules(); err != nil {
		// 如果加载失败，使用默认规则
		engine.rules = DefaultRules()
		log.Printf("[RulesEngine] Failed to load rules from storage, using defaults: %v", err)
	}

	return engine, nil
}

// NewEngineWithDefaults 创建使用默认规则的引擎
func NewEngineWithDefaults() *Engine {
	return &Engine{
		rules: DefaultRules(),
	}
}

// Evaluate 评估情报项
func (e *Engine) Evaluate(item core.IntelItem) RuleResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := RuleResult{
		NoiseScore:   0,
		QualityScore: 50, // 默认中等质量
		MatchedRules: []string{},
		SignalType:   string(SignalTypeNoise),
	}

	// 按优先级排序规则
	sortedRules := make([]Rule, len(e.rules))
	copy(sortedRules, e.rules)
	sort.Slice(sortedRules, func(i, j int) bool {
		return sortedRules[i].Priority > sortedRules[j].Priority
	})

	// 评估每个规则
	for _, rule := range sortedRules {
		if !rule.IsActive {
			continue
		}

		matched, signalType := e.evaluateRule(item, rule)
		if matched {
			result.NoiseScore += rule.Weight
			result.MatchedRules = append(result.MatchedRules, rule.RuleName)

			// 如果是信号（负权重），且还没有设置信号类型，则设置
			// 优先级：第一个匹配的信号规则决定信号类型
			if rule.Weight < 0 && signalType != "" && result.SignalType == string(SignalTypeNoise) {
				result.SignalType = signalType
			}
		}
	}

	// 判断是否为噪声
	result.IsNoise = IsNoiseByScore(result.NoiseScore)

	// 如果不是噪声，计算质量分数
	if !result.IsNoise {
		result.QualityScore = e.calculateQuality(item, result)
	}

	return result
}

// evaluateRule 评估单个规则
func (e *Engine) evaluateRule(item core.IntelItem, rule Rule) (bool, string) {
	switch rule.RuleType {
	case RuleTypeKeyword:
		return e.evaluateKeywordRule(item, rule)
	case RuleTypeLength:
		return e.evaluateLengthRule(item, rule)
	case RuleTypeRegex:
		return e.evaluateRegexRule(item, rule)
	case RuleTypeUserType:
		return e.evaluateUserTypeRule(item, rule)
	default:
		return false, ""
	}
}

// evaluateKeywordRule 评估关键词规则
func (e *Engine) evaluateKeywordRule(item core.IntelItem, rule Rule) (bool, string) {
	var value KeywordRuleValue
	if err := json.Unmarshal([]byte(rule.RuleValue), &value); err != nil {
		// 简单字符串格式（逗号分隔）
		keywords := strings.Split(rule.RuleValue, ",")
		value = KeywordRuleValue{Keywords: keywords, Mode: "any"}
	}

	text := strings.ToLower(item.Title + " " + item.Content)
	mode := value.Mode
	if mode == "" {
		mode = "any"
	}

	matchCount := 0
	for _, keyword := range value.Keywords {
		keyword = strings.TrimSpace(strings.ToLower(keyword))
		if keyword != "" && strings.Contains(text, keyword) {
			matchCount++
		}
	}

	if mode == "all" {
		return matchCount == len(value.Keywords), e.detectSignalType(text)
	}
	return matchCount > 0, e.detectSignalType(text)
}

// evaluateLengthRule 评估长度规则
func (e *Engine) evaluateLengthRule(item core.IntelItem, rule Rule) (bool, string) {
	var value LengthRuleValue
	if err := json.Unmarshal([]byte(rule.RuleValue), &value); err != nil {
		return false, ""
	}

	length := len(item.Content)

	// 检查是否过短
	if value.Min > 0 && length < value.Min {
		return true, ""
	}

	// 检查是否过长
	if value.Max > 0 && length > value.Max {
		return true, ""
	}

	return false, ""
}

// evaluateRegexRule 评估正则规则
func (e *Engine) evaluateRegexRule(item core.IntelItem, rule Rule) (bool, string) {
	var value RegexRuleValue
	if err := json.Unmarshal([]byte(rule.RuleValue), &value); err != nil {
		return false, ""
	}

	flags := value.Flags
	re := value.Pattern
	if strings.Contains(flags, "i") {
		re = "(?i)" + re
	}

	matched, err := regexp.MatchString(re, item.Title+" "+item.Content)
	if err != nil {
		log.Printf("[RulesEngine] Invalid regex pattern %s: %v", re, err)
		return false, ""
	}

	return matched, ""
}

// evaluateUserTypeRule 评估用户类型规则
func (e *Engine) evaluateUserTypeRule(item core.IntelItem, rule Rule) (bool, string) {
	var value UserTypeRuleValue
	if err := json.Unmarshal([]byte(rule.RuleValue), &value); err != nil {
		return false, ""
	}

	// 从元数据获取用户类型
	userType, ok := item.Metadata["user_type"].(string)
	if !ok {
		return false, ""
	}

	for _, ut := range value.UserTypes {
		if strings.EqualFold(userType, ut) {
			return !value.Exclude, ""
		}
	}

	return value.Exclude, ""
}

// detectSignalType 检测信号类型
func (e *Engine) detectSignalType(text string) string {
	text = strings.ToLower(text)

	// 迁移意愿
	migrationKeywords := []string{"alternative", "switch to", "switching", "migrate", "migration", "move to"}
	for _, kw := range migrationKeywords {
		if strings.Contains(text, kw) {
			return string(SignalTypeMigration)
		}
	}

	// 成本压力
	costKeywords := []string{"expensive", "cost too much", "pricey", "too expensive", "bill is", "billing"}
	for _, kw := range costKeywords {
		if strings.Contains(text, kw) {
			return string(SignalTypeCostPressure)
		}
	}

	// 功能需求
	featureKeywords := []string{"wish there was", "need a", "would be great if", "feature request"}
	for _, kw := range featureKeywords {
		if strings.Contains(text, kw) {
			return string(SignalTypeFeature)
		}
	}

	// 竞品动态
	competitorKeywords := []string{"claude", "anthropic", "gemini", "llama", "mistral"}
	for _, kw := range competitorKeywords {
		if strings.Contains(text, kw) {
			return string(SignalTypeCompetitor)
		}
	}

	// 默认痛点
	painKeywords := []string{"frustrated", "annoying", "problem", "issue", "error", "fail", "broken"}
	for _, kw := range painKeywords {
		if strings.Contains(text, kw) {
			return string(SignalTypeUserPain)
		}
	}

	return ""
}

// calculateQuality 计算质量分数
func (e *Engine) calculateQuality(item core.IntelItem, result RuleResult) float64 {
	params := QualityScoreParams{
		KeywordMatch:   e.calculateKeywordMatch(item, result),
		InfluenceScore: e.calculateInfluence(item),
		Completeness:   e.calculateCompleteness(item),
		Timeliness:     e.calculateTimeliness(item),
		Relevance:      e.calculateRelevance(item, result),
	}

	return CalculateQualityScore(params)
}

// calculateKeywordMatch 计算关键词匹配度
func (e *Engine) calculateKeywordMatch(item core.IntelItem, result RuleResult) float64 {
	// 命中的信号规则数量越多，匹配度越高
	signalCount := 0
	for _, ruleName := range result.MatchedRules {
		for _, rule := range e.rules {
			if rule.RuleName == ruleName && rule.Weight < 0 {
				signalCount++
			}
		}
	}

	// 最多 3 个信号规则就满分
	score := float64(signalCount) / 3.0 * 100
	if score > 100 {
		score = 100
	}
	return score
}

// calculateInfluence 计算影响力分数
func (e *Engine) calculateInfluence(item core.IntelItem) float64 {
	// 基于元数据中的互动数据
	points := 0.0
	if p, ok := item.Metadata["points"].(int); ok {
		points = float64(p)
	}
	if p, ok := item.Metadata["points"].(float64); ok {
		points = p
	}

	// 100 点满分
	score := points / 100.0 * 100
	if score > 100 {
		score = 100
	}
	return score
}

// calculateCompleteness 计算完整度
func (e *Engine) calculateCompleteness(item core.IntelItem) float64 {
	score := 0.0

	// 有标题 +20
	if item.Title != "" {
		score += 20
	}

	// 有内容 +40
	if item.Content != "" && len(item.Content) >= 50 {
		score += 40
	} else if item.Content != "" {
		score += 20
	}

	// 有 URL +20
	if item.URL != "" {
		score += 20
	}

	// 有作者 +10
	if author, ok := item.Metadata["author"].(string); ok && author != "" {
		score += 10
	}

	// 有发布时间 +10
	if item.PublishedAt != nil {
		score += 10
	}

	return score
}

// calculateTimeliness 计算时效性
func (e *Engine) calculateTimeliness(item core.IntelItem) float64 {
	if item.PublishedAt == nil {
		return 50 // 无时间信息，给中等分
	}

	age := time.Since(*item.PublishedAt)

	// 24小时内 = 100分
	// 7天内 = 80分
	// 30天内 = 60分
	// 更久 = 40分

	if age < 24*time.Hour {
		return 100
	} else if age < 7*24*time.Hour {
		return 80
	} else if age < 30*24*time.Hour {
		return 60
	}
	return 40
}

// calculateRelevance 计算相关性
func (e *Engine) calculateRelevance(item core.IntelItem, result RuleResult) float64 {
	// 如果命中了信号规则，相关性更高
	signalWeight := 0
	for _, ruleName := range result.MatchedRules {
		for _, rule := range e.rules {
			if rule.RuleName == ruleName && rule.Weight < 0 {
				signalWeight += -rule.Weight
			}
		}
	}

	// 归一化到 0-100
	score := float64(signalWeight) / 30.0 * 100
	if score > 100 {
		score = 100
	}
	return score
}

// ReloadRules 重新加载规则
func (e *Engine) ReloadRules() error {
	if e.storage == nil {
		return nil
	}

	rules, err := e.storage.LoadRules()
	if err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules = rules

	log.Printf("[RulesEngine] Reloaded %d rules", len(rules))
	return nil
}

// AddRule 添加规则
func (e *Engine) AddRule(rule Rule) error {
	if e.storage != nil {
		if err := e.storage.SaveRule(&rule); err != nil {
			return err
		}
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules = append(e.rules, rule)
	return nil
}

// RemoveRule 移除规则
func (e *Engine) RemoveRule(ruleID int64) error {
	if e.storage != nil {
		if err := e.storage.DeleteRule(ruleID); err != nil {
			return err
		}
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	for i, r := range e.rules {
		if r.ID == ruleID {
			e.rules = append(e.rules[:i], e.rules[i+1:]...)
			break
		}
	}

	return nil
}

// ListRules 列出所有规则
func (e *Engine) ListRules() ([]Rule, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]Rule, len(e.rules))
	copy(result, e.rules)
	return result, nil
}

// DefaultRules 返回默认规则
func DefaultRules() []Rule {
	now := time.Now()
	return []Rule{
		// 噪声规则（正权重）
		{
			ID:        1,
			RuleType:  RuleTypeKeyword,
			RuleName:  "营销推广",
			RuleValue: `{"keywords":["check out my","my new tool","try our","visit my","subscribe"],"mode":"any"}`,
			Weight:    WeightSpam,
			IsActive:  true,
			Priority:  100,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        2,
			RuleType:  RuleTypeLength,
			RuleName:  "内容过短",
			RuleValue: `{"min":20,"max":0}`,
			Weight:    WeightShort,
			IsActive:  true,
			Priority:  50,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// 信号规则（负权重）
		{
			ID:        10,
			RuleType:  RuleTypeKeyword,
			RuleName:  "价格痛点",
			RuleValue: `{"keywords":["expensive","cost too much","pricey","too expensive","billing issue"],"mode":"any"}`,
			Weight:    WeightPainPoint,
			IsActive:  true,
			Priority:  90,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        11,
			RuleType:  RuleTypeKeyword,
			RuleName:  "迁移意愿",
			RuleValue: `{"keywords":["alternative to","switching to","looking for alternative","migrate from"],"mode":"any"}`,
			Weight:    WeightMigration,
			IsActive:  true,
			Priority:  95,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        12,
			RuleType:  RuleTypeKeyword,
			RuleName:  "功能需求",
			RuleValue: `{"keywords":["wish there was","need a","would be great if","feature request"],"mode":"any"}`,
			Weight:    WeightFeature,
			IsActive:  true,
			Priority:  85,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        13,
			RuleType:  RuleTypeKeyword,
			RuleName:  "竞品动态",
			RuleValue: `{"keywords":["claude","anthropic","gemini","llama","mistral"],"mode":"any"}`,
			Weight:    WeightCompetitor,
			IsActive:  true,
			Priority:  80,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        14,
			RuleType:  RuleTypeKeyword,
			RuleName:  "成本压力",
			RuleValue: `{"keywords":["my bill","billing is","invoice","payment issue","rate limit"],"mode":"any"}`,
			Weight:    WeightCostPressure,
			IsActive:  true,
			Priority:  88,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}
