// Package ai 提供AI相关的功能，包括翻译
package ai

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"token-bridge-crawler/internal/core"
)

// md5Hash 计算 MD5 哈希值（32位小写）
func md5Hash(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// 确保 OpenRouterTranslator 实现了 core.Translator 接口
var _ core.Translator = (*OpenRouterTranslator)(nil)

// OpenRouterTranslator 使用OpenRouter API的翻译器
type OpenRouterTranslator struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewOpenRouterTranslator 创建OpenRouter翻译器
func NewOpenRouterTranslator(apiKey string) *OpenRouterTranslator {
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}
	return &OpenRouterTranslator{
		apiKey: apiKey,
		model:  "anthropic/claude-3.5-sonnet", // 默认使用Claude 3.5 Sonnet
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// SetModel 设置翻译模型
func (t *OpenRouterTranslator) SetModel(model string) {
	t.model = model
}

// Translate 翻译单个文本
func (t *OpenRouterTranslator) Translate(text string, sourceLang, targetLang string) (string, error) {
	if text == "" {
		return "", nil
	}

	prompt := fmt.Sprintf(`请将以下%s内容翻译成%s。保持原文的格式和语气，只返回翻译结果，不要添加任何解释。

原文：
%s`, sourceLang, targetLang, text)

	return t.callOpenRouter(prompt)
}

// TranslateBatch 批量翻译
func (t *OpenRouterTranslator) TranslateBatch(texts []string, sourceLang, targetLang string) ([]string, error) {
	if len(texts) == 0 {
		return []string{}, nil
	}

	// 构建批量翻译提示
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("请将以下%d条%s内容翻译成%s。保持原文的格式和语气，按顺序返回翻译结果，每条翻译用---TRANSLATION_SEPARATOR---分隔，不要添加任何解释。\n\n", len(texts), sourceLang, targetLang))

	for i, text := range texts {
		buffer.WriteString(fmt.Sprintf("[%d] %s\n", i+1, text))
	}

	result, err := t.callOpenRouter(buffer.String())
	if err != nil {
		return nil, err
	}

	// 解析批量翻译结果
	translations := parseBatchResult(result, len(texts))
	return translations, nil
}

// callOpenRouter 调用OpenRouter API
func (t *OpenRouterTranslator) callOpenRouter(prompt string) (string, error) {
	if t.apiKey == "" {
		return "", fmt.Errorf("OpenRouter API key not configured")
	}

	reqBody := map[string]interface{}{
		"model": t.model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "你是一个专业的翻译助手，擅长将英文技术内容准确翻译成中文。保持专业术语的准确性，确保翻译自然流畅。",
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.3, // 低温度以获得更稳定的翻译
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	req.Header.Set("HTTP-Referer", "https://tokenbridge.local")
	req.Header.Set("X-Title", "Token Bridge Intelligence")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenRouter API returned status %d", resp.StatusCode)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Error != nil {
		return "", fmt.Errorf("OpenRouter API error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no translation result")
	}

	return result.Choices[0].Message.Content, nil
}

// YoudaoTranslator 使用有道翻译API的翻译器
type YoudaoTranslator struct {
	appKey    string
	appSecret string
	httpClient *http.Client
}

// NewYoudaoTranslator 创建有道翻译器
func NewYoudaoTranslator(appKey, appSecret string) *YoudaoTranslator {
	if appKey == "" {
		appKey = os.Getenv("YOUDAO_APP_KEY")
	}
	if appSecret == "" {
		appSecret = os.Getenv("YOUDAO_APP_SECRET")
	}
	return &YoudaoTranslator{
		appKey:    appKey,
		appSecret: appSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Translate 翻译单个文本
func (t *YoudaoTranslator) Translate(text string, sourceLang, targetLang string) (string, error) {
	if text == "" {
		return "", nil
	}

	// 简化语言代码
	src := "en"
	tgt := "zh"
	if sourceLang == "中文" {
		src = "zh"
		tgt = "en"
	}

	return t.callYoudaoAPI(text, src, tgt)
}

// TranslateBatch 批量翻译
func (t *YoudaoTranslator) TranslateBatch(texts []string, sourceLang, targetLang string) ([]string, error) {
	if len(texts) == 0 {
		return []string{}, nil
	}

	translations := make([]string, len(texts))
	for i, text := range texts {
		translated, err := t.Translate(text, sourceLang, targetLang)
		if err != nil {
			return nil, err
		}
		translations[i] = translated
	}

	return translations, nil
}

// callYoudaoAPI 调用有道翻译API
func (t *YoudaoTranslator) callYoudaoAPI(text, from, to string) (string, error) {
	if t.appKey == "" || t.appSecret == "" {
		return "", fmt.Errorf("Youdao API credentials not configured")
	}

	// 构建请求参数
	reqBody := map[string]string{
		"q":     text,
		"from":  from,
		"to":    to,
		"appKey": t.appKey,
		"salt":   fmt.Sprintf("%d", time.Now().Unix()),
	}

	// 计算签名（这里简化处理，实际需要按有道API文档计算）
	sign := t.appKey + text + reqBody["salt"] + t.appSecret

	reqBody["sign"] = sign

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://openapi.youdao.com/api", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Youdao API returned status %d", resp.StatusCode)
	}

	var result struct {
		Translation []string `json:"translation"`
		Error       int      `json:"errorCode"`
		Message     string   `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Error != 0 {
		return "", fmt.Errorf("Youdao API error: %s (code: %d)", result.Message, result.Error)
	}

	if len(result.Translation) == 0 {
		return "", fmt.Errorf("no translation result")
	}

	return result.Translation[0], nil
}

// BaiduLLMTranslator 百度大模型文本翻译器（优先使用）
type BaiduLLMTranslator struct {
	appID      string
	apiKey     string
	httpClient *http.Client
	rateLimit  *time.Ticker
}

// NewBaiduLLMTranslator 创建百度大模型翻译器
func NewBaiduLLMTranslator(appID, apiKey string) *BaiduLLMTranslator {
	if appID == "" {
		appID = os.Getenv("BAIDU_APP_ID")
	}
	if apiKey == "" {
		apiKey = os.Getenv("BAIDU_API_KEY")
	}
	return &BaiduLLMTranslator{
		appID:   appID,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimit: time.NewTicker(200 * time.Millisecond), // 5 QPS
	}
}

// Translate 翻译单个文本
func (t *BaiduLLMTranslator) Translate(text string, sourceLang, targetLang string) (string, error) {
	if text == "" {
		return "", nil
	}

	src := "en"
	tgt := "zh"
	if sourceLang == "中文" {
		src = "zh"
		tgt = "en"
	}

	return t.callLLMAPI(text, src, tgt)
}

// TranslateBatch 批量翻译
func (t *BaiduLLMTranslator) TranslateBatch(texts []string, sourceLang, targetLang string) ([]string, error) {
	if len(texts) == 0 {
		return []string{}, nil
	}

	translations := make([]string, len(texts))
	for i, text := range texts {
		translated, err := t.Translate(text, sourceLang, targetLang)
		if err != nil {
			return nil, err
		}
		translations[i] = translated
	}

	return translations, nil
}

// callLLMAPI 调用百度大模型翻译API
func (t *BaiduLLMTranslator) callLLMAPI(text, from, to string) (string, error) {
	if t.appID == "" || t.apiKey == "" {
		return "", fmt.Errorf("Baidu LLM API credentials not configured")
	}

	// 限流
	if t.rateLimit != nil {
		<-t.rateLimit.C
	}

	// 构建请求体
	reqBody := map[string]interface{}{
		"appid": t.appID,
		"from":  from,
		"to":    to,
		"q":     text,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://fanyi-api.baidu.com/ait/api/aiTextTranslate", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.apiKey)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var rawResult struct {
		TransResult []struct {
			Src string `json:"src"`
			Dst string `json:"dst"`
		} `json:"trans_result"`
		ErrorCode interface{} `json:"error_code"`
		ErrorMsg  string      `json:"error_msg"`
	}

	if err := json.Unmarshal(bodyBytes, &rawResult); err != nil {
		return "", err
	}

	if rawResult.ErrorCode != nil {
		var code string
		switch v := rawResult.ErrorCode.(type) {
		case string:
			code = v
		case float64:
			code = fmt.Sprintf("%.0f", v)
		default:
			code = fmt.Sprintf("%v", v)
		}
		if code != "0" && code != "" {
			return "", fmt.Errorf("Baidu LLM API error: %s (code: %s)", rawResult.ErrorMsg, code)
		}
	}

	if len(rawResult.TransResult) == 0 {
		return "", fmt.Errorf("no translation result")
	}

	return rawResult.TransResult[0].Dst, nil
}

// BaiduClassicTranslator 百度通用文本翻译器（备选）
type BaiduClassicTranslator struct {
	appID      string
	appSecret  string
	httpClient *http.Client
	rateLimit  *time.Ticker
}

// NewBaiduClassicTranslator 创建百度通用翻译器
func NewBaiduClassicTranslator(appID, appSecret string) *BaiduClassicTranslator {
	if appID == "" {
		appID = os.Getenv("BAIDU_APP_ID")
	}
	if appSecret == "" {
		appSecret = os.Getenv("BAIDU_APP_SECRET")
	}
	return &BaiduClassicTranslator{
		appID:     appID,
		appSecret: appSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimit: time.NewTicker(time.Second), // 标准版 QPS=1
	}
}

// Translate 翻译单个文本
func (t *BaiduClassicTranslator) Translate(text string, sourceLang, targetLang string) (string, error) {
	if text == "" {
		return "", nil
	}

	src := "en"
	tgt := "zh"
	if sourceLang == "中文" {
		src = "zh"
		tgt = "en"
	}

	return t.callClassicAPI(text, src, tgt)
}

// TranslateBatch 批量翻译
func (t *BaiduClassicTranslator) TranslateBatch(texts []string, sourceLang, targetLang string) ([]string, error) {
	if len(texts) == 0 {
		return []string{}, nil
	}

	translations := make([]string, len(texts))
	for i, text := range texts {
		translated, err := t.Translate(text, sourceLang, targetLang)
		if err != nil {
			return nil, err
		}
		translations[i] = translated
	}

	return translations, nil
}

// callClassicAPI 调用百度通用翻译API
func (t *BaiduClassicTranslator) callClassicAPI(text, from, to string) (string, error) {
	if t.appID == "" || t.appSecret == "" {
		return "", fmt.Errorf("Baidu Classic API credentials not configured")
	}

	// 限流
	if t.rateLimit != nil {
		<-t.rateLimit.C
	}

	// 生成随机数
	salt := fmt.Sprintf("%d", time.Now().UnixNano())

	// 计算签名：appid + q + salt + 密钥 的 MD5
	signStr := t.appID + text + salt + t.appSecret
	sign := md5Hash(signStr)

	// 构建请求参数
	params := url.Values{}
	params.Set("q", text)
	params.Set("from", from)
	params.Set("to", to)
	params.Set("appid", t.appID)
	params.Set("salt", salt)
	params.Set("sign", sign)

	req, err := http.NewRequest("POST", "https://fanyi-api.baidu.com/api/trans/vip/translate", strings.NewReader(params.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var rawResult struct {
		TransResult []struct {
			Src string `json:"src"`
			Dst string `json:"dst"`
		} `json:"trans_result"`
		ErrorCode interface{} `json:"error_code"`
		ErrorMsg  string      `json:"error_msg"`
	}

	if err := json.Unmarshal(bodyBytes, &rawResult); err != nil {
		return "", err
	}

	if rawResult.ErrorCode != nil {
		var code string
		switch v := rawResult.ErrorCode.(type) {
		case string:
			code = v
		case float64:
			code = fmt.Sprintf("%.0f", v)
		default:
			code = fmt.Sprintf("%v", v)
		}
		if code != "0" && code != "" {
			return "", fmt.Errorf("Baidu Classic API error: %s (code: %s)", rawResult.ErrorMsg, code)
		}
	}

	if len(rawResult.TransResult) == 0 {
		return "", fmt.Errorf("no translation result")
	}

	return rawResult.TransResult[0].Dst, nil
}

// FallbackTranslator 备用翻译器（当所有API都失败时使用）
type FallbackTranslator struct {}

// NewFallbackTranslator 创建备用翻译器
func NewFallbackTranslator() *FallbackTranslator {
	return &FallbackTranslator{}
}

// Translate 翻译单个文本（返回原文）
func (t *FallbackTranslator) Translate(text string, sourceLang, targetLang string) (string, error) {
	return text, nil // 备用翻译器直接返回原文
}

// TranslateBatch 批量翻译（返回原文）
func (t *FallbackTranslator) TranslateBatch(texts []string, sourceLang, targetLang string) ([]string, error) {
	return texts, nil // 备用翻译器直接返回原文
}

// TranslatorQueue 翻译器队列，按优先级尝试多个翻译器
type TranslatorQueue struct {
	translators []core.Translator
}

// NewTranslatorQueue 创建翻译器队列
func NewTranslatorQueue() *TranslatorQueue {
	return &TranslatorQueue{
		translators: []core.Translator{},
	}
}

// AddTranslator 添加翻译器到队列
func (q *TranslatorQueue) AddTranslator(translator core.Translator) {
	q.translators = append(q.translators, translator)
}

// Translate 翻译单个文本，按队列顺序尝试
func (q *TranslatorQueue) Translate(text string, sourceLang, targetLang string) (string, error) {
	if text == "" {
		return "", nil
	}

	var lastErr error
	for _, translator := range q.translators {
		translated, err := translator.Translate(text, sourceLang, targetLang)
		if err == nil && translated != "" {
			return translated, nil
		}
		lastErr = err
	}

	return "", fmt.Errorf("all translators failed: %v", lastErr)
}

// TranslateBatch 批量翻译，按队列顺序尝试
func (q *TranslatorQueue) TranslateBatch(texts []string, sourceLang, targetLang string) ([]string, error) {
	if len(texts) == 0 {
		return []string{}, nil
	}

	var lastErr error
	for _, translator := range q.translators {
		translations, err := translator.TranslateBatch(texts, sourceLang, targetLang)
		if err == nil && len(translations) == len(texts) {
			return translations, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("all translators failed: %v", lastErr)
}

// parseBatchResult 解析批量翻译结果
func parseBatchResult(result string, expectedCount int) []string {
	// 尝试用分隔符分割
	parts := bytes.Split([]byte(result), []byte("---TRANSLATION_SEPARATOR---"))

	translations := make([]string, 0, expectedCount)
	for _, part := range parts {
		translation := cleanTranslation(string(part))
		if translation != "" {
			translations = append(translations, translation)
		}
	}

	// 如果分割后的数量不对，尝试按行号解析
	if len(translations) != expectedCount {
		translations = parseByNumber(result, expectedCount)
	}

	// 如果还是不对，返回原始结果作为第一条
	if len(translations) == 0 {
		translations = []string{cleanTranslation(result)}
	}

	// 补齐缺失的翻译
	for len(translations) < expectedCount {
		translations = append(translations, "")
	}

	return translations[:expectedCount]
}

// parseByNumber 按行号解析翻译结果
func parseByNumber(result string, expectedCount int) []string {
	translations := make([]string, expectedCount)

	// 简单的按行号匹配，例如 "[1] 翻译内容"
	for i := 1; i <= expectedCount; i++ {
		prefix := fmt.Sprintf("[%d]", i)
		if idx := bytes.Index([]byte(result), []byte(prefix)); idx != -1 {
			start := idx + len(prefix)
			end := len(result)

			// 查找下一个行号或结束
			if i < expectedCount {
				nextPrefix := fmt.Sprintf("[%d]", i+1)
				if nextIdx := bytes.Index([]byte(result[start:]), []byte(nextPrefix)); nextIdx != -1 {
					end = start + nextIdx
				}
			}

			if start < end && start < len(result) {
				translations[i-1] = cleanTranslation(result[start:end])
			}
		}
	}

	return translations
}

// cleanTranslation 清理翻译结果
func cleanTranslation(text string) string {
	// 去除前后空白
	text = string(bytes.TrimSpace([]byte(text)))

	// 去除可能的引号
	text = string(bytes.Trim([]byte(text), `"'`))

	// 去除 "翻译：" 或 "Translation:" 等前缀
	prefixes := []string{"翻译：", "Translation:", "译文：", "中文："}
	for _, prefix := range prefixes {
		if bytes.HasPrefix([]byte(text), []byte(prefix)) {
			text = string(bytes.TrimPrefix([]byte(text), []byte(prefix)))
			text = string(bytes.TrimSpace([]byte(text)))
		}
	}

	return text
}

// VolcengineTranslator 火山引擎翻译器
type VolcengineTranslator struct {
	accessKeyID     string
	secretAccessKey string
	httpClient      *http.Client
	rateLimit       *time.Ticker
}

// NewVolcengineTranslator 创建火山引擎翻译器
func NewVolcengineTranslator(accessKeyID, secretAccessKey string) *VolcengineTranslator {
	if accessKeyID == "" {
		accessKeyID = os.Getenv("VOLCENGINE_ACCESS_KEY_ID")
	}
	if secretAccessKey == "" {
		secretAccessKey = os.Getenv("VOLCENGINE_SECRET_ACCESS_KEY")
	}
	return &VolcengineTranslator{
		accessKeyID:     accessKeyID,
		secretAccessKey: secretAccessKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimit: time.NewTicker(100 * time.Millisecond), // 10 QPS
	}
}

// Translate 翻译单个文本
func (t *VolcengineTranslator) Translate(text string, sourceLang, targetLang string) (string, error) {
	if text == "" {
		return "", nil
	}

	src := "en"
	tgt := "zh"
	if sourceLang == "中文" {
		src = "zh"
		tgt = "en"
	}

	results, err := t.callVolcengineAPI([]string{text}, src, tgt)
	if err != nil {
		return "", err
	}

	if len(results) == 0 {
		return "", fmt.Errorf("no translation result")
	}

	return results[0], nil
}

// TranslateBatch 批量翻译（火山引擎支持最多16条）
func (t *VolcengineTranslator) TranslateBatch(texts []string, sourceLang, targetLang string) ([]string, error) {
	if len(texts) == 0 {
		return []string{}, nil
	}

	src := "en"
	tgt := "zh"
	if sourceLang == "中文" {
		src = "zh"
		tgt = "en"
	}

	// 火山引擎限制：最多16条，总长度5000字符
	batchSize := 16
	if len(texts) > batchSize {
		// 分批处理
		allResults := make([]string, 0, len(texts))
		for i := 0; i < len(texts); i += batchSize {
			end := i + batchSize
			if end > len(texts) {
				end = len(texts)
			}
			batch := texts[i:end]
			results, err := t.callVolcengineAPI(batch, src, tgt)
			if err != nil {
				return nil, err
			}
			allResults = append(allResults, results...)
		}
		return allResults, nil
	}

	return t.callVolcengineAPI(texts, src, tgt)
}

// callVolcengineAPI 调用火山引擎翻译API
func (t *VolcengineTranslator) callVolcengineAPI(texts []string, sourceLang, targetLang string) ([]string, error) {
	if t.accessKeyID == "" || t.secretAccessKey == "" {
		return nil, fmt.Errorf("Volcengine API credentials not configured")
	}

	// 限流
	if t.rateLimit != nil {
		<-t.rateLimit.C
	}

	// 构建请求体
	reqBody := map[string]interface{}{
		"TargetLanguage": targetLang,
		"TextList":       texts,
	}
	if sourceLang != "" && sourceLang != "auto" {
		reqBody["SourceLanguage"] = sourceLang
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	// 火山引擎API参数
	host := "translate.volcengineapi.com"
	action := "TranslateText"
	version := "2020-06-01"
	region := "cn-north-1"
	service := "translate"

	// 生成时间戳
	now := time.Now().UTC()
	xDate := now.Format("20060102T150405Z")
	shortDate := now.Format("20060102")

	// 构建请求URL
	queryString := fmt.Sprintf("Action=%s&Version=%s", action, version)
	urlStr := fmt.Sprintf("https://%s/?%s", host, queryString)

	// 创建请求
	req, err := http.NewRequest("POST", urlStr, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Host", host)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Date", xDate)

	// 计算签名
	_, _, signature := t.signRequestDebug(host, action, version, region, service, xDate, shortDate, queryString, jsonBody)

	// 设置Authorization头
	credentialScope := fmt.Sprintf("%s/%s/%s/request", shortDate, region, service)
	authorization := fmt.Sprintf("HMAC-SHA256 Credential=%s/%s, SignedHeaders=host;x-date, Signature=%s",
		t.accessKeyID, credentialScope, signature)
	req.Header.Set("Authorization", authorization)

	// 发送请求
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		TranslationList []struct {
			Translation            string `json:"Translation"`
			DetectedSourceLanguage string `json:"DetectedSourceLanguage"`
		} `json:"TranslationList"`
		ResponseMetadata struct {
			Error struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"ResponseMetadata"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, err
	}

	if result.ResponseMetadata.Error.Code != "" {
		return nil, fmt.Errorf("Volcengine API error: %s (code: %s)", result.ResponseMetadata.Error.Message, result.ResponseMetadata.Error.Code)
	}

	if len(result.TranslationList) == 0 {
		return nil, fmt.Errorf("no translation result")
	}

	translations := make([]string, len(result.TranslationList))
	for i, t := range result.TranslationList {
		translations[i] = t.Translation
	}

	return translations, nil
}

// signRequestDebug 计算火山引擎签名（带调试输出）
func (t *VolcengineTranslator) signRequestDebug(host, action, version, region, service, xDate, shortDate, queryString string, body []byte) (canonicalRequest, stringToSign, signature string) {
	// Step 1: CanonicalRequest
	canonicalHeaders := fmt.Sprintf("host:%s\nx-date:%s\n", host, xDate)
	signedHeaders := "host;x-date"
	bodyHash := sha256Hash(body)
	canonicalRequest = fmt.Sprintf("POST\n/\n%s\n%s\n%s\n%s", queryString, canonicalHeaders, signedHeaders, bodyHash)

	// Step 2: StringToSign
	canonicalRequestHash := sha256Hash([]byte(canonicalRequest))
	credentialScope := fmt.Sprintf("%s/%s/%s/request", shortDate, region, service)
	stringToSign = fmt.Sprintf("HMAC-SHA256\n%s\n%s\n%s", xDate, credentialScope, canonicalRequestHash)

	// Step 3: Signature
	kDate := hmacSHA256([]byte(t.secretAccessKey), shortDate)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	kSigning := hmacSHA256(kService, "request")
	signature = hmacSHA256Hex(kSigning, stringToSign)

	return canonicalRequest, stringToSign, signature
}

// sha256Hash 计算SHA256哈希
func sha256Hash(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// hmacSHA256 计算HMAC-SHA256
func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

// hmacSHA256Hex 计算HMAC-SHA256并返回十六进制字符串
func hmacSHA256Hex(key []byte, data string) string {
	return hex.EncodeToString(hmacSHA256(key, data))
}
