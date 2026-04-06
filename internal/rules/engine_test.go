package rules_test

import (
	"testing"
	"time"

	"token-bridge-crawler/internal/core"
	"token-bridge-crawler/internal/rules"
)

func TestRuleEngine_Evaluate(t *testing.T) {
	engine := rules.NewEngineWithDefaults()

	tests := []struct {
		name           string
		item           core.IntelItem
		expectNoise    bool
		expectSignal   bool
		minQuality     float64
	}{
		{
			name: "营销推广噪声",
			item: core.IntelItem{
				Title:   "Check out my new AI tool!",
				Content: "Visit my website for the best AI tool ever.",
				Source:  "hackernews",
			},
			expectNoise: true,
		},
		{
			name: "价格痛点信号",
			item: core.IntelItem{
				Title:   "OpenAI API is too expensive",
				Content: "The pricing for GPT-4 is way too expensive for my small business. Looking for alternatives.",
				Source:  "hackernews",
			},
			expectNoise:  false,
			expectSignal: true,
			minQuality:   50,
		},
		{
			name: "迁移意愿信号",
			item: core.IntelItem{
				Title:   "Switching from OpenAI to Claude",
				Content: "I'm looking for an alternative to OpenAI. The rate limits are killing me.",
				Source:  "hackernews",
			},
			expectNoise:  false,
			expectSignal: true,
			minQuality:   60,
		},
		{
			name: "过短内容噪声",
			item: core.IntelItem{
				Title:   "API down?",
				Content: "down", // 少于 20 字符
				Source:  "hackernews",
			},
			expectNoise: true,
		},
		{
			name: "普通讨论",
			item: core.IntelItem{
				Title:   "Best practices for LLM prompting",
				Content: "Here are some tips for getting better results from large language models...",
				Source:  "hackernews",
			},
			expectNoise: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.Evaluate(tt.item)

			if result.IsNoise != tt.expectNoise {
				t.Errorf("IsNoise = %v, want %v", result.IsNoise, tt.expectNoise)
			}

			if tt.expectSignal && result.SignalType == string(rules.SignalTypeNoise) {
				t.Errorf("Expected signal, got noise")
			}

			if tt.minQuality > 0 && result.QualityScore < tt.minQuality {
				t.Errorf("QualityScore = %.1f, want >= %.1f", result.QualityScore, tt.minQuality)
			}

			t.Logf("Result: IsNoise=%v, NoiseScore=%d, Quality=%.1f, SignalType=%s, Matched=%v",
				result.IsNoise, result.NoiseScore, result.QualityScore, result.SignalType, result.MatchedRules)
		})
	}
}

func TestRuleEngine_DefaultRules(t *testing.T) {
	rules := rules.DefaultRules()

	if len(rules) == 0 {
		t.Error("Default rules should not be empty")
	}

	// 检查有噪声规则（正权重）
	hasNoiseRule := false
	for _, r := range rules {
		if r.Weight > 0 {
			hasNoiseRule = true
			break
		}
	}
	if !hasNoiseRule {
		t.Error("Should have at least one noise rule (positive weight)")
	}

	// 检查有信号规则（负权重）
	hasSignalRule := false
	for _, r := range rules {
		if r.Weight < 0 {
			hasSignalRule = true
			break
		}
	}
	if !hasSignalRule {
		t.Error("Should have at least one signal rule (negative weight)")
	}
}

func TestRuleEngine_AddRemoveRule(t *testing.T) {
	engine := rules.NewEngineWithDefaults()

	// 添加新规则
	newRule := rules.Rule{
		ID:        999,
		RuleType:  rules.RuleTypeKeyword,
		RuleName:  "测试规则",
		RuleValue: `{"keywords":["测试关键词"],"mode":"any"}`,
		Weight:    5,
		IsActive:  true,
		Priority:  100,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := engine.AddRule(newRule)
	if err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	// 验证规则已添加
	rules, err := engine.ListRules()
	if err != nil {
		t.Fatalf("ListRules failed: %v", err)
	}

	found := false
	for _, r := range rules {
		if r.RuleName == "测试规则" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Added rule not found in list")
	}

	// 删除规则
	err = engine.RemoveRule(999)
	if err != nil {
		t.Fatalf("RemoveRule failed: %v", err)
	}

	// 验证规则已删除
	rules, _ = engine.ListRules()
	for _, r := range rules {
		if r.RuleName == "测试规则" {
			t.Error("Rule should have been removed")
		}
	}
}
