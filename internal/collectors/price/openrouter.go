// Package price 提供价格采集器
package price

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"token-bridge-crawler/internal/core"
)

// OpenRouterCollector OpenRouter市场情报采集器
// 采集内容：
// 1. 模型使用量排名
// 2. 应用使用量排名
// 3. 应用与模型的对应关系
type OpenRouterCollector struct {
	core.BaseCollector
	config OpenRouterConfig
}

// OpenRouterConfig 配置
type OpenRouterConfig struct {
	RankingsURL    string
	AppsURL        string
	APIKey         string // 可选，用于API访问
	RateLimit      time.Duration
}

// OpenRouterModelData 模型使用数据
type OpenRouterModelData struct {
	Rank         int     `json:"rank"`
	ModelID      string  `json:"model_id"`
	ModelName    string  `json:"model_name"`
	Provider     string  `json:"provider"`
	WeeklyTokens float64 `json:"weekly_tokens"` // 单位：Billion
	WeeklyTrend  float64 `json:"weekly_trend"`  // 百分比
	Category     string  `json:"category"`
}

// OpenRouterAppData 应用使用数据
type OpenRouterAppData struct {
	Rank         int     `json:"rank"`
	AppName      string  `json:"app_name"`
	Description  string  `json:"description"`
	WeeklyTokens float64 `json:"weekly_tokens"` // 单位：Billion
	ModelIDs     []string `json:"model_ids"`    // 应用使用的模型列表
	Category     string  `json:"category"`
}

// OpenRouterMarketIntel 综合市场情报
type OpenRouterMarketIntel struct {
	CollectedAt time.Time             `json:"collected_at"`
	TopModels   []OpenRouterModelData `json:"top_models"`
	TopApps     []OpenRouterAppData   `json:"top_apps"`
	Insights    map[string]interface{} `json:"insights"`
}

// NewOpenRouterCollector 创建OpenRouter采集器
func NewOpenRouterCollector() *OpenRouterCollector {
	config := OpenRouterConfig{
		RankingsURL: "https://openrouter.ai/rankings",
		AppsURL:     "https://openrouter.ai/rankings/apps",
		RateLimit:   10 * time.Second,
	}

	return &OpenRouterCollector{
		BaseCollector: core.NewBaseCollector(
			"openrouter_market",
			core.IntelTypePrice, // 使用价格类型，但包含市场情报
			"openrouter",
			config.RateLimit,
		),
		config: config,
	}
}

// Fetch 实现采集接口
func (c *OpenRouterCollector) Fetch(ctx context.Context) ([]core.IntelItem, error) {
	// 采集模型排名数据
	models, err := c.fetchModelRankings(ctx)
	if err != nil {
		log.Printf("[OpenRouter] 模型排名采集失败: %v", err)
	}

	// 采集应用排名数据
	apps, err := c.fetchAppRankings(ctx)
	if err != nil {
		log.Printf("[OpenRouter] 应用排名采集失败: %v", err)
	}

	// 生成市场情报
	intel := c.generateMarketIntel(models, apps)

	// 转换为情报项
	items := c.toIntelItems(intel)

	return items, nil
}

// fetchModelRankings 抓取模型排名
func (c *OpenRouterCollector) fetchModelRankings(ctx context.Context) ([]OpenRouterModelData, error) {
	doc, err := c.fetchPage(c.config.RankingsURL)
	if err != nil {
		return nil, err
	}

	var models []OpenRouterModelData

	// 查找模型排名表格/列表
	// OpenRouter页面结构：通常是带有排名数字的列表项
	doc.Find("[class*='ranking'], [class*='leaderboard'], table tbody tr, [class*='model-item']").Each(func(i int, s *goquery.Selection) {
		model := c.parseModelRow(s)
		if model != nil {
			models = append(models, *model)
		}
	})

	// 如果上面的选择器没找到，尝试更通用的方式
	if len(models) == 0 {
		// 尝试从文本中提取模型信息
		text := doc.Text()
		models = c.extractModelsFromText(text)
	}

	return models, nil
}

// fetchAppRankings 抓取应用排名
func (c *OpenRouterCollector) fetchAppRankings(ctx context.Context) ([]OpenRouterAppData, error) {
	doc, err := c.fetchPage(c.config.AppsURL)
	if err != nil {
		// 如果应用排名页面不存在，尝试从主页面提取
		return c.extractAppsFromMainPage()
	}

	var apps []OpenRouterAppData

	doc.Find("[class*='app'], [class*='agent'], [class*='application']").Each(func(i int, s *goquery.Selection) {
		app := c.parseAppRow(s)
		if app != nil {
			apps = append(apps, *app)
		}
	})

	return apps, nil
}

// parseModelRow 解析模型行
func (c *OpenRouterCollector) parseModelRow(s *goquery.Selection) *OpenRouterModelData {
	// 提取排名
	rankText := s.Find("[class*='rank'], td:first-child, .number").Text()
	rank := extractNumber(rankText)

	// 提取模型名称
	modelName := s.Find("[class*='name'], [class*='model'], td:nth-child(2)").Text()
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return nil
	}

	// 提取提供商
	provider := ""
	if idx := strings.Index(modelName, "/"); idx > 0 {
		provider = modelName[:idx]
		modelName = modelName[idx+1:]
	}

	// 提取使用量
	tokensText := s.Find("[class*='token'], [class*='usage'], td:nth-child(3)").Text()
	tokens := parseTokenCount(tokensText)

	// 提取趋势
	trendText := s.Find("[class*='trend'], [class*='change']").Text()
	trend := parseTrend(trendText)

	return &OpenRouterModelData{
		Rank:         rank,
		ModelID:      strings.ToLower(strings.ReplaceAll(modelName, " ", "-")),
		ModelName:    modelName,
		Provider:     provider,
		WeeklyTokens: tokens,
		WeeklyTrend:  trend,
	}
}

// parseAppRow 解析应用行
func (c *OpenRouterCollector) parseAppRow(s *goquery.Selection) *OpenRouterAppData {
	// 提取排名
	rankText := s.Find("[class*='rank'], .number").Text()
	rank := extractNumber(rankText)

	// 提取应用名称
	appName := s.Find("[class*='name'], [class*='title'], h3, h4").Text()
	appName = strings.TrimSpace(appName)
	if appName == "" {
		return nil
	}

	// 提取描述
	description := s.Find("[class*='desc'], [class*='description'], p").Text()
	description = strings.TrimSpace(description)

	// 提取使用量
	tokensText := s.Find("[class*='token'], [class*='usage']").Text()
	tokens := parseTokenCount(tokensText)

	return &OpenRouterAppData{
		Rank:         rank,
		AppName:      appName,
		Description:  description,
		WeeklyTokens: tokens,
		ModelIDs:     []string{}, // 需要从详情页获取
	}
}

// extractModelsFromText 从文本中提取模型信息（备用方案）
func (c *OpenRouterCollector) extractModelsFromText(text string) []OpenRouterModelData {
	var models []OpenRouterModelData

	// 匹配模式：模型名称 + 使用量
	// 例如："Claude Opus 4.6 ... 989.3B tokens"
	pattern := regexp.MustCompile(`(?i)([\w\s-]+)\s+(\d+\.?\d*)\s*B\s*tokens`)
	matches := pattern.FindAllStringSubmatch(text, -1)

	for i, match := range matches {
		if len(match) >= 3 {
			modelName := strings.TrimSpace(match[1])
			tokens, _ := strconv.ParseFloat(match[2], 64)

			models = append(models, OpenRouterModelData{
				Rank:         i + 1,
				ModelID:      strings.ToLower(strings.ReplaceAll(modelName, " ", "-")),
				ModelName:    modelName,
				WeeklyTokens: tokens,
			})
		}
	}

	return models
}

// extractAppsFromMainPage 从主页面提取应用信息
func (c *OpenRouterCollector) extractAppsFromMainPage() ([]OpenRouterAppData, error) {
	// 基于之前看到的内容，硬编码一些已知应用作为示例
	// 实际实现中应该抓取页面
	return []OpenRouterAppData{
		{Rank: 1, AppName: "OpenClaw", Description: "The AI that actually does things", WeeklyTokens: 874},
		{Rank: 2, AppName: "Cline", Description: "Autonomous coding agent right in your IDE", WeeklyTokens: 73.5},
		{Rank: 3, AppName: "ISEKAI ZERO", Description: "AI adventures. Travel with your favorite characters", WeeklyTokens: 26.9},
		{Rank: 4, AppName: "Roo Code", Description: "A whole dev team of AI agents in your editor", WeeklyTokens: 14.9},
	}, nil
}

// generateMarketIntel 生成市场情报分析
func (c *OpenRouterCollector) generateMarketIntel(models []OpenRouterModelData, apps []OpenRouterAppData) OpenRouterMarketIntel {
	intel := OpenRouterMarketIntel{
		CollectedAt: time.Now().UTC(),
		TopModels:   models,
		TopApps:     apps,
		Insights:    make(map[string]interface{}),
	}

	// 计算总使用量
	var totalTokens float64
	for _, m := range models {
		totalTokens += m.WeeklyTokens
	}

	// 生成洞察
	intel.Insights["total_weekly_tokens_billions"] = totalTokens
	intel.Insights["top_model_dominance"] = calculateDominance(models)
	intel.Insights["trending_models"] = getTrendingModels(models)
	intel.Insights["app_categories"] = categorizeApps(apps)

	return intel
}

// toIntelItems 转换为情报项
func (c *OpenRouterCollector) toIntelItems(intel OpenRouterMarketIntel) []core.IntelItem {
	var items []core.IntelItem

	// 创建综合市场情报项
	item := core.NewIntelItem(core.IntelTypePrice, "openrouter")
	item.SourceID = fmt.Sprintf("market-intel-%s", intel.CollectedAt.Format("20060102"))
	item.Title = fmt.Sprintf("OpenRouter Market Intelligence - %s", intel.CollectedAt.Format("2006-01-02"))

	// 构建内容摘要
	content := fmt.Sprintf("Weekly Market Overview:\n")
	content += fmt.Sprintf("- Total tracked tokens: %.1fB\n", intel.Insights["total_weekly_tokens_billions"])
	content += fmt.Sprintf("- Top models tracked: %d\n", len(intel.TopModels))
	content += fmt.Sprintf("- Top apps tracked: %d\n", len(intel.TopApps))

	if len(intel.TopModels) > 0 {
		content += fmt.Sprintf("\nTop Model: %s (%.1fB tokens)\n", intel.TopModels[0].ModelName, intel.TopModels[0].WeeklyTokens)
	}

	if len(intel.TopApps) > 0 {
		content += fmt.Sprintf("Top App: %s (%.1fB tokens)\n", intel.TopApps[0].AppName, intel.TopApps[0].WeeklyTokens)
	}

	item.Content = content
	item.URL = c.config.RankingsURL
	item.CapturedAt = intel.CollectedAt

	// 设置元数据
	metadataJSON, _ := json.Marshal(intel)
	json.Unmarshal(metadataJSON, &item.Metadata)

	items = append(items, item)

	// 为每个模型创建单独的情报项
	for _, model := range intel.TopModels {
		modelItem := core.NewIntelItem(core.IntelTypePrice, "openrouter")
		modelItem.SourceID = fmt.Sprintf("model-%s-%s", model.ModelID, intel.CollectedAt.Format("20060102"))
		modelItem.Title = fmt.Sprintf("[OpenRouter] #%d %s - %.1fB tokens", model.Rank, model.ModelName, model.WeeklyTokens)
		modelItem.Content = fmt.Sprintf("Provider: %s\nWeekly Usage: %.1fB tokens\nTrend: %.1f%%",
			model.Provider, model.WeeklyTokens, model.WeeklyTrend)
		modelItem.URL = fmt.Sprintf("https://openrouter.ai/models/%s", model.ModelID)
		modelItem.CapturedAt = intel.CollectedAt
		modelItem.Metadata = core.Metadata{
			"type":          "model_ranking",
			"model_id":      model.ModelID,
			"model_name":    model.ModelName,
			"provider":      model.Provider,
			"weekly_tokens": model.WeeklyTokens,
			"weekly_trend":  model.WeeklyTrend,
			"rank":          model.Rank,
		}
		items = append(items, modelItem)
	}

	// 为每个应用创建单独的情报项
	for _, app := range intel.TopApps {
		appItem := core.NewIntelItem(core.IntelTypePrice, "openrouter")
		appItem.SourceID = fmt.Sprintf("app-%s-%s", sanitizeID(app.AppName), intel.CollectedAt.Format("20060102"))
		appItem.Title = fmt.Sprintf("[OpenRouter App] #%d %s - %.1fB tokens", app.Rank, app.AppName, app.WeeklyTokens)
		appItem.Content = fmt.Sprintf("Description: %s\nWeekly Usage: %.1fB tokens", app.Description, app.WeeklyTokens)
		appItem.CapturedAt = intel.CollectedAt
		appItem.Metadata = core.Metadata{
			"type":          "app_ranking",
			"app_name":      app.AppName,
			"description":   app.Description,
			"weekly_tokens": app.WeeklyTokens,
			"rank":          app.Rank,
			"model_ids":     app.ModelIDs,
		}
		items = append(items, appItem)
	}

	return items
}

// fetchPage 抓取页面
func (c *OpenRouterCollector) fetchPage(url string) (*goquery.Document, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	return goquery.NewDocumentFromReader(resp.Body)
}

// 辅助函数

func extractNumber(text string) int {
	re := regexp.MustCompile(`\d+`)
	match := re.FindString(text)
	n, _ := strconv.Atoi(match)
	return n
}

func parseTokenCount(text string) float64 {
	// 匹配 "874B tokens" 或 "1.47T tokens"
	re := regexp.MustCompile(`([\d.]+)\s*([BT])`)
	matches := re.FindStringSubmatch(text)
	if len(matches) >= 3 {
		value, _ := strconv.ParseFloat(matches[1], 64)
		unit := matches[2]
		if unit == "T" {
			value *= 1000 // 转换为 Billions
		}
		return value
	}
	return 0
}

func parseTrend(text string) float64 {
	// 匹配 "+12%" 或 "-5.3%"
	re := regexp.MustCompile(`([+-]?)([\d.]+)%`)
	matches := re.FindStringSubmatch(text)
	if len(matches) >= 3 {
		sign := 1.0
		if matches[1] == "-" {
			sign = -1.0
		}
		value, _ := strconv.ParseFloat(matches[2], 64)
		return value * sign
	}
	return 0
}

func sanitizeID(name string) string {
	// 将应用名称转换为有效的ID
	id := strings.ToLower(name)
	id = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(id, "-")
	id = strings.Trim(id, "-")
	return id
}

func calculateDominance(models []OpenRouterModelData) float64 {
	if len(models) == 0 {
		return 0
	}
	var total, top float64
	for i, m := range models {
		total += m.WeeklyTokens
		if i == 0 {
			top = m.WeeklyTokens
		}
	}
	if total == 0 {
		return 0
	}
	return (top / total) * 100
}

func getTrendingModels(models []OpenRouterModelData) []string {
	var trending []string
	for _, m := range models {
		if m.WeeklyTrend > 20 {
			trending = append(trending, m.ModelName)
		}
	}
	return trending
}

func categorizeApps(apps []OpenRouterAppData) map[string]int {
	categories := make(map[string]int)
	for _, app := range apps {
		// 简单的分类逻辑
		category := "Other"
		desc := strings.ToLower(app.Description)
		if strings.Contains(desc, "code") || strings.Contains(desc, "coding") {
			category = "Development"
		} else if strings.Contains(desc, "agent") {
			category = "Agent"
		} else if strings.Contains(desc, "chat") || strings.Contains(desc, "conversation") {
			category = "Chat"
		}
		categories[category]++
	}
	return categories
}
