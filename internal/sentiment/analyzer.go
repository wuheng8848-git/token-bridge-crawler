// Package sentiment 情感分析与痛点强度计算
package sentiment

import (
	"strings"
)

// Analyzer 情感分析器
type Analyzer struct {
	dict *SentimentDict
}

// NewAnalyzer 创建情感分析器
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		dict: DefaultSentimentDict(),
	}
}

// NewAnalyzerWithDict 创建带自定义词典的分析器
func NewAnalyzerWithDict(dict *SentimentDict) *Analyzer {
	return &Analyzer{
		dict: dict,
	}
}

// Analyze 分析文本情感
func (a *Analyzer) Analyze(text string) Result {
	text = strings.ToLower(text)
	words := tokenize(text)

	// 计算情感分数
	sentimentScore := 0.0
	painScore := 0.0
	negation := false
	intensifier := 1.0

	for i, word := range words {
		// 检查否定词
		if a.dict.IsNegation(word) {
			negation = true
			continue
		}

		// 检查程度词
		if mult := a.dict.GetIntensifierMultiplier(word); mult > 1.0 {
			intensifier = mult
			continue
		}

		// 获取词语情感值
		wordScore := a.dict.GetWordScore(word)
		if wordScore != 0 {
			score := wordScore * intensifier
			if negation {
				score = -score
			}
			sentimentScore += score

			// 负面词增加痛点分数
			if wordScore < 0 {
				painScore += -score * 10 // 放大痛点分数
			}
		}

		// 重置状态
		if i > 0 && words[i-1] != word {
			negation = false
			intensifier = 1.0
		}
	}

	// 确定情感类型
	sentiment := "neutral"
	if sentimentScore > 0.5 {
		sentiment = "positive"
	} else if sentimentScore < -0.5 {
		sentiment = "negative"
	}

	// 计算置信度
	confidence := a.calculateConfidence(words, sentimentScore)

	// 限制痛点分数在 0-100
	if painScore > 100 {
		painScore = 100
	}
	if painScore < 0 {
		painScore = 0
	}

	return Result{
		Sentiment:  sentiment,
		PainScore:  painScore,
		Confidence: confidence,
	}
}

// tokenize 分词
func tokenize(text string) []string {
	// 简单分词：按空格和标点分割
	text = strings.ToLower(text)
	for _, sep := range []string{",", ".", "!", "?", ";", ":", "'", "\"", "(", ")", "[", "]", "\n", "\t"} {
		text = strings.ReplaceAll(text, sep, " ")
	}
	words := strings.Fields(text)
	return words
}

// calculateConfidence 计算置信度
func (a *Analyzer) calculateConfidence(words []string, score float64) float64 {
	if len(words) == 0 {
		return 0
	}

	// 命中词典的词语数量
	matchedCount := 0
	for _, word := range words {
		if a.dict.GetWordScore(word) != 0 {
			matchedCount++
		}
	}

	// 匹配比例
	matchRatio := float64(matchedCount) / float64(len(words))

	// 分数绝对值（越极端越可信）
	scoreFactor := min(1.0, abs(score)/3.0)

	// 综合置信度
	confidence := matchRatio*0.6 + scoreFactor*0.4
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// Result 情感分析结果
type Result struct {
	Sentiment  string  `json:"sentiment"`  // positive/negative/neutral
	PainScore  float64 `json:"pain_score"` // 0-100
	Confidence float64 `json:"confidence"` // 0-1
}
