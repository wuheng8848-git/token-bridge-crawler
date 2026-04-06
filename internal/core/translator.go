// Package core 提供翻译器接口
package core

// Translator 翻译器接口
type Translator interface {
	Translate(text string, sourceLang, targetLang string) (string, error)
	TranslateBatch(texts []string, sourceLang, targetLang string) ([]string, error)
}

// TranslationService 翻译服务接口
type TranslationService interface {
	TranslateIntelItem(item *IntelItem) error
	TranslateIntelItems(items []IntelItem) error
	SetEnabled(enabled bool)
	GetTranslationStatus() map[string]interface{}
}
