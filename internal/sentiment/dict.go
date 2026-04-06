// Package sentiment 情感分析与痛点强度计算
package sentiment

// SentimentDict 情感词典
type SentimentDict struct {
	// 负面词 -> 分数（负数）
	NegativeWords map[string]float64

	// 正面词 -> 分数（正数）
	PositiveWords map[string]float64

	// 否定词
	Negations map[string]bool

	// 程度词 -> 倍数
	Intensifiers map[string]float64
}

// DefaultSentimentDict 默认情感词典
func DefaultSentimentDict() *SentimentDict {
	return &SentimentDict{
		NegativeWords: defaultNegativeWords(),
		PositiveWords: defaultPositiveWords(),
		Negations:     defaultNegations(),
		Intensifiers:  defaultIntensifiers(),
	}
}

// GetWordScore 获取词语情感分数
func (d *SentimentDict) GetWordScore(word string) float64 {
	if score, ok := d.NegativeWords[word]; ok {
		return score
	}
	if score, ok := d.PositiveWords[word]; ok {
		return score
	}
	return 0
}

// IsNegation 检查是否为否定词
func (d *SentimentDict) IsNegation(word string) bool {
	return d.Negations[word]
}

// GetIntensifierMultiplier 获取程度词倍数
func (d *SentimentDict) GetIntensifierMultiplier(word string) float64 {
	return d.Intensifiers[word]
}

// defaultNegativeWords 默认负面词
func defaultNegativeWords() map[string]float64 {
	return map[string]float64{
		// 痛点相关
		"expensive":    -1.5,
		"pricey":       -1.3,
		"costly":       -1.3,
		"overpriced":   -1.5,
		"unaffordable": -1.6,
		"cheap":        -0.5, // 上下文相关

		// 情绪相关
		"frustrated":    -1.8,
		"frustrating":   -1.7,
		"annoying":      -1.5,
		"annoyed":       -1.6,
		"angry":         -1.9,
		"upset":         -1.6,
		"disappointed":  -1.5,
		"disappointing": -1.5,
		"terrible":      -2.0,
		"horrible":      -2.0,
		"awful":         -1.9,
		"bad":           -1.0,
		"poor":          -1.2,
		"worse":         -1.3,
		"worst":         -1.8,

		// 问题相关
		"problem":    -1.2,
		"issue":      -1.0,
		"error":      -1.3,
		"fail":       -1.5,
		"failed":     -1.5,
		"failure":    -1.6,
		"broken":     -1.4,
		"bug":        -1.1,
		"buggy":      -1.3,
		"crash":      -1.6,
		"crashed":    -1.6,
		"slow":       -1.2,
		"laggy":      -1.3,
		"unstable":   -1.5,
		"unreliable": -1.6,

		// API 相关痛点
		"timeout":   -1.4,
		"rate":      -0.8, // 需要上下文
		"limited":   -1.1,
		"limit":     -0.9,
		"throttle":  -1.3,
		"throttled": -1.4,
		"quota":     -0.8,
		"exceeded":  -1.4,
		"down":      -1.5,
		"outage":    -1.7,
		"downtime":  -1.6,

		// 质量 相关痛点
		"hallucination": -1.8,
		"hallucinate":   -1.7,
		"inaccurate":    -1.4,
		"wrong":         -1.3,
		"incorrect":     -1.4,
		"misleading":    -1.5,
		"confused":      -1.2,
		"confusing":     -1.2,
		"inconsistent":  -1.3,

		// 迁移相关
		"switch":      -0.5, // 需要上下文
		"switching":   -0.6,
		"migrate":     -0.5,
		"migration":   -0.5,
		"alternative": -0.4,
		"replace":     -0.5,
		"replacement": -0.5,
	}
}

// defaultPositiveWords 默认正面词
func defaultPositiveWords() map[string]float64 {
	return map[string]float64{
		"great":      1.5,
		"excellent":  1.7,
		"amazing":    1.8,
		"awesome":    1.6,
		"fantastic":  1.7,
		"wonderful":  1.6,
		"good":       1.0,
		"nice":       0.8,
		"love":       1.8,
		"loved":      1.7,
		"like":       0.7,
		"liked":      0.6,
		"happy":      1.4,
		"satisfied":  1.3,
		"pleased":    1.3,
		"helpful":    1.2,
		"useful":     1.1,
		"perfect":    1.8,
		"best":       1.6,
		"better":     1.1,
		"fast":       1.0,
		"quick":      0.8,
		"reliable":   1.3,
		"stable":     1.1,
		"smooth":     1.0,
		"easy":       1.0,
		"simple":     0.8,
		"intuitive":  1.1,
		"affordable": 1.2,
		"cheap":      0.6,
		"value":      0.8,
		"worth":      0.9,
	}
}

// defaultNegations 默认否定词
func defaultNegations() map[string]bool {
	return map[string]bool{
		"not":       true,
		"no":        true,
		"never":     true,
		"neither":   true,
		"nobody":    true,
		"nothing":   true,
		"nowhere":   true,
		"hardly":    true,
		"barely":    true,
		"scarcely":  true,
		"don't":     true,
		"doesn't":   true,
		"didn't":    true,
		"won't":     true,
		"wouldn't":  true,
		"can't":     true,
		"cannot":    true,
		"couldn't":  true,
		"shouldn't": true,
		"isn't":     true,
		"aren't":    true,
		"wasn't":    true,
		"weren't":   true,
	}
}

// defaultIntensifiers 默认程度词
func defaultIntensifiers() map[string]float64 {
	return map[string]float64{
		"very":          1.5,
		"really":        1.4,
		"extremely":     1.8,
		"absolutely":    1.7,
		"completely":    1.6,
		"totally":       1.5,
		"utterly":       1.7,
		"highly":        1.4,
		"deeply":        1.5,
		"so":            1.3,
		"too":           1.4,
		"quite":         1.2,
		"pretty":        1.2,
		"rather":        1.1,
		"fairly":        1.1,
		"slightly":      0.8,
		"somewhat":      0.9,
		"bit":           0.8,
		"little":        0.8,
		"incredibly":    1.8,
		"insanely":      1.9,
		"ridiculously":  1.8,
		"exceptionally": 1.7,
	}
}

// AddNegativeWord 添加负面词
func (d *SentimentDict) AddNegativeWord(word string, score float64) {
	d.NegativeWords[word] = score
}

// AddPositiveWord 添加正面词
func (d *SentimentDict) AddPositiveWord(word string, score float64) {
	d.PositiveWords[word] = score
}

// AddNegation 添加否定词
func (d *SentimentDict) AddNegation(word string) {
	d.Negations[word] = true
}

// AddIntensifier 添加程度词
func (d *SentimentDict) AddIntensifier(word string, multiplier float64) {
	d.Intensifiers[word] = multiplier
}
