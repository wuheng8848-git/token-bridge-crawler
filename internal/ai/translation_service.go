// Package ai 提供AI相关的功能，包括翻译服务
package ai

import (
	"context"
	"log"
	"sync"

	"token-bridge-crawler/internal/core"
)

// 确保 TranslationService 实现了 core.TranslationService 接口
var _ core.TranslationService = (*TranslationService)(nil)

// TranslationService 翻译服务
type TranslationService struct {
	translator core.Translator
	enabled    bool
	batchSize  int
}

// NewTranslationService 创建翻译服务
func NewTranslationService(translator core.Translator) *TranslationService {
	return &TranslationService{
		translator: translator,
		enabled:    translator != nil,
		batchSize:  5, // 默认批量翻译5条
	}
}

// NewTranslationServiceWithQueue 创建带有翻译器队列的翻译服务
func NewTranslationServiceWithQueue(openRouterAPIKey, youdaoAppKey, youdaoAppSecret, baiduAppID, baiduAPIKey, baiduAppSecret, volcengineAccessKeyID, volcengineSecretAccessKey string) *TranslationService {
	// 创建翻译器队列
	queue := NewTranslatorQueue()

	// 按免费额度和性能排序：
	// 1. 火山引擎（200万字符/月，最快286ms）
	// 2. 百度大模型（100万字符/月，767ms）
	// 3. 百度通用（5万字符/月，1.0s，有频率限制）
	// 总免费额度：305万字符/月 > 需求210万字符/月

	// 添加火山引擎翻译器（优先级最高）
	if volcengineAccessKeyID != "" && volcengineSecretAccessKey != "" {
		queue.AddTranslator(NewVolcengineTranslator(volcengineAccessKeyID, volcengineSecretAccessKey))
		log.Println("[TranslationService] 已添加火山引擎翻译器（优先级1，200万字符/月）")
	}

	// 添加百度大模型翻译器（优先级第二）
	if baiduAppID != "" && baiduAPIKey != "" {
		queue.AddTranslator(NewBaiduLLMTranslator(baiduAppID, baiduAPIKey))
		log.Println("[TranslationService] 已添加百度大模型翻译器（优先级2，100万字符/月）")
	}

	// 添加百度通用翻译器（优先级第三）
	if baiduAppID != "" && baiduAppSecret != "" {
		queue.AddTranslator(NewBaiduClassicTranslator(baiduAppID, baiduAppSecret))
		log.Println("[TranslationService] 已添加百度通用翻译器（优先级3，5万字符/月）")
	}

	// 添加OpenRouter翻译器（付费备选）
	if openRouterAPIKey != "" {
		queue.AddTranslator(NewOpenRouterTranslator(openRouterAPIKey))
	}

	// 添加有道翻译器（付费备选）
	if youdaoAppKey != "" && youdaoAppSecret != "" {
		queue.AddTranslator(NewYoudaoTranslator(youdaoAppKey, youdaoAppSecret))
	}

	// 添加备用翻译器（当所有API都失败时使用）
	queue.AddTranslator(NewFallbackTranslator())

	return &TranslationService{
		translator: queue,
		enabled:    true,
		batchSize:  5, // 默认批量翻译5条
	}
}

// SetEnabled 设置是否启用翻译
func (s *TranslationService) SetEnabled(enabled bool) {
	s.enabled = enabled
}

// SetBatchSize 设置批量翻译大小
func (s *TranslationService) SetBatchSize(size int) {
	s.batchSize = size
}

// TranslateIntelItem 翻译单个情报项
func (s *TranslationService) TranslateIntelItem(item *core.IntelItem) error {
	if !s.enabled || s.translator == nil {
		return nil
	}

	// 翻译标题
	if item.Title != "" && s.isEnglish(item.Title) {
		translated, err := s.translator.Translate(item.Title, "英文", "中文")
		if err != nil {
			log.Printf("[TranslationService] 翻译标题失败: %v (原文: %s)", err, item.Title)
		} else if translated != "" {
			item.Metadata["title_zh"] = translated
			log.Printf("[TranslationService] 标题翻译成功: %s -> %s", item.Title, translated)
		}
	}

	// 翻译内容
	if item.Content != "" && s.isEnglish(item.Content) {
		translated, err := s.translator.Translate(item.Content, "英文", "中文")
		if err != nil {
			log.Printf("[TranslationService] 翻译内容失败: %v", err)
		} else if translated != "" {
			item.Metadata["content_zh"] = translated
		}
	}

	return nil
}

// TranslateIntelItems 批量翻译情报项
func (s *TranslationService) TranslateIntelItems(items []core.IntelItem) error {
	if !s.enabled || s.translator == nil || len(items) == 0 {
		return nil
	}

	// 收集需要翻译的标题和内容
	var titlesToTranslate []string
	var contentsToTranslate []string
	titleIndices := make(map[int]int)  // 记录哪些item的标题需要翻译
	contentIndices := make(map[int]int) // 记录哪些item的内容需要翻译

	for i, item := range items {
		if item.Title != "" && s.isEnglish(item.Title) {
			titleIndices[i] = len(titlesToTranslate)
			titlesToTranslate = append(titlesToTranslate, item.Title)
		}
		if item.Content != "" && s.isEnglish(item.Content) {
			contentIndices[i] = len(contentsToTranslate)
			contentsToTranslate = append(contentsToTranslate, item.Content)
		}
	}

	// 批量翻译标题
	if len(titlesToTranslate) > 0 {
		titleTranslations, err := s.translateBatchWithRetry(titlesToTranslate, "英文", "中文")
		if err == nil {
			for itemIdx, transIdx := range titleIndices {
				if transIdx < len(titleTranslations) && titleTranslations[transIdx] != "" {
					items[itemIdx].Metadata["title_zh"] = titleTranslations[transIdx]
				}
			}
		} else {
			log.Printf("[TranslationService] 批量翻译标题失败: %v", err)
		}
	}

	// 批量翻译内容
	if len(contentsToTranslate) > 0 {
		contentTranslations, err := s.translateBatchWithRetry(contentsToTranslate, "英文", "中文")
		if err == nil {
			for itemIdx, transIdx := range contentIndices {
				if transIdx < len(contentTranslations) && contentTranslations[transIdx] != "" {
					items[itemIdx].Metadata["content_zh"] = contentTranslations[transIdx]
				}
			}
		} else {
			log.Printf("[TranslationService] 批量翻译内容失败: %v", err)
		}
	}

	return nil
}

// translateBatchWithRetry 带重试的批量翻译
func (s *TranslationService) translateBatchWithRetry(texts []string, sourceLang, targetLang string) ([]string, error) {
	if len(texts) == 0 {
		return []string{}, nil
	}

	// 如果数量超过批量大小，分批处理
	if len(texts) > s.batchSize {
		var allTranslations []string
		for i := 0; i < len(texts); i += s.batchSize {
			end := i + s.batchSize
			if end > len(texts) {
				end = len(texts)
			}
			batch := texts[i:end]
			translations, err := s.translator.TranslateBatch(batch, sourceLang, targetLang)
			if err != nil {
				return nil, err
			}
			allTranslations = append(allTranslations, translations...)
		}
		return allTranslations, nil
	}

	return s.translator.TranslateBatch(texts, sourceLang, targetLang)
}

// isEnglish 检测文本是否为英文
func (s *TranslationService) isEnglish(text string) bool {
	if text == "" {
		return false
	}

	// 简单检测：如果包含中文字符，则认为是中文
	for _, r := range text {
		if r >= '\u4e00' && r <= '\u9fff' {
			return false // 包含中文字符
		}
	}

	// 如果包含英文字母，则认为是英文
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return true
		}
	}

	return false
}

// TranslationMiddleware 翻译中间件
func (s *TranslationService) TranslationMiddleware() func([]core.IntelItem) ([]core.IntelItem, error) {
	return func(items []core.IntelItem) ([]core.IntelItem, error) {
		if !s.enabled || len(items) == 0 {
			return items, nil
		}

		// 使用goroutine并发翻译
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, 3) // 限制并发数

		for i := range items {
			wg.Add(1)
			semaphore <- struct{}{}

			go func(idx int) {
				defer wg.Done()
				defer func() { <-semaphore }()

				if err := s.TranslateIntelItem(&items[idx]); err != nil {
					log.Printf("[TranslationMiddleware] 翻译失败: %v", err)
				}
			}(i)
		}

		wg.Wait()
		return items, nil
	}
}

// TranslateWithContext 带上下文的翻译
func (s *TranslationService) TranslateWithContext(ctx context.Context, text string) (string, error) {
	if !s.enabled || s.translator == nil || text == "" {
		return text, nil
	}

	if !s.isEnglish(text) {
		return text, nil // 不是英文，直接返回
	}

	// 使用channel和select实现超时控制
	resultChan := make(chan struct {
		translated string
		err        error
	}, 1)

	go func() {
		translated, err := s.translator.Translate(text, "英文", "中文")
		resultChan <- struct {
			translated string
			err        error
		}{translated, err}
	}()

	select {
	case <-ctx.Done():
		return text, ctx.Err()
	case result := <-resultChan:
		return result.translated, result.err
	}
}

// GetTranslationStatus 获取翻译状态
func (s *TranslationService) GetTranslationStatus() map[string]interface{} {
	return map[string]interface{}{
		"enabled":   s.enabled,
		"batchSize": s.batchSize,
		"hasTranslator": s.translator != nil,
	}
}
