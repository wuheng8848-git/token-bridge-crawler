// Package apidoc 提供API文档采集器
package apidoc

import (
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// OpenAIAPIDocCollector OpenAI API文档采集器
type OpenAIAPIDocCollector struct {
	*APIDocCollector
}

// NewOpenAIAPIDocCollector 创建OpenAI API文档采集器
func NewOpenAIAPIDocCollector() *OpenAIAPIDocCollector {
	config := APIDocConfig{
		ChangelogURL: "https://platform.openai.com/docs/changelog",
		APIDocURL:    "https://platform.openai.com/docs/",
		RateLimit:    10 * time.Second,
	}

	base := NewAPIDocCollector("openai", "OpenAI", config)

	return &OpenAIAPIDocCollector{
		APIDocCollector: base,
	}
}

// parseChangelog 解析OpenAI changelog
func (c *OpenAIAPIDocCollector) parseChangelog(doc *goquery.Document) []APIChange {
	var changes []APIChange

	// OpenAI changelog页面结构：按日期分组，每组包含多个变更
	doc.Find("[class*='changelog'], [class*='entry'], [class*='item']").Each(func(i int, s *goquery.Selection) {
		// 提取日期
		dateText := s.Find("[class*='date'], h2, h3").Text()
		date := parseOpenAIDate(dateText)

		// 提取变更项
		s.Find("[class*='change'], [class*='entry'], [class*='item']").Each(func(j int, item *goquery.Selection) {
			change := c.parseOpenAIChange(item, date)
			if change != nil {
				changes = append(changes, *change)
				c.LogChange(*change)
			}
		})
	})

	// 如果上面的选择器没找到，尝试更通用的方式
	if len(changes) == 0 {
		// 尝试从文本中提取变更
		text := doc.Text()
		changes = c.extractChangesFromText(text)
	}

	return changes
}

// parseOpenAIChange 解析OpenAI变更项
func (c *OpenAIAPIDocCollector) parseOpenAIChange(s *goquery.Selection, date time.Time) *APIChange {
	// 提取标题
	title := strings.TrimSpace(s.Find("[class*='title'], h4, h5").Text())
	if title == "" {
		return nil
	}

	// 提取描述
	description := strings.TrimSpace(s.Find("[class*='description'], p").Text())

	// 提取类型
	changeType := c.determineChangeType(title, description)

	// 提取链接
	url := ""
	if link, exists := s.Find("a").Attr("href"); exists {
		if !strings.HasPrefix(link, "http") {
			// 相对链接
			url = c.config.APIDocURL + link
		} else {
			url = link
		}
	}

	// 提取受影响的API
	affectedAPIs := c.extractAffectedAPIs(description)

	return &APIChange{
		Title:       title,
		Description: description,
		Date:        date,
		Type:        changeType,
		URL:         url,
		AffectedAPIs: affectedAPIs,
	}
}

// parseOpenAIDate 解析OpenAI日期
func parseOpenAIDate(dateText string) time.Time {
	dateText = strings.TrimSpace(dateText)

	// 常见格式："March 12, 2026" 或 "2026-03-12"
	formats := []string{
		"January 2, 2006",
		"2006-01-02",
		"Jan 2, 2006",
		"2006/01/02",
	}

	for _, format := range formats {
		if date, err := time.Parse(format, dateText); err == nil {
			return date
		}
	}

	return time.Time{}
}

// determineChangeType 确定变更类型
func (c *OpenAIAPIDocCollector) determineChangeType(title, description string) string {
	text := strings.ToLower(title + " " + description)

	switch {
	case strings.Contains(text, "breaking"):
		return "breaking_change"
	case strings.Contains(text, "deprecated"):
		return "deprecated"
	case strings.Contains(text, "new"):
		return "new_feature"
	case strings.Contains(text, "fix") || strings.Contains(text, "bug"):
		return "bug_fix"
	default:
		return "improvement"
	}
}

// extractAffectedAPIs 提取受影响的API
func (c *OpenAIAPIDocCollector) extractAffectedAPIs(description string) []string {
	var APIs []string

	// 常见API名称
	apiPatterns := []string{
		"GPT-4", "GPT-4o", "GPT-3.5", "GPT-3",
		"DALL-E", "Whisper", "Embedding",
		"Completion", "Chat", "Edit", "Image", "Audio",
		"Assistants", "Threads", "Messages",
	}

	for _, api := range apiPatterns {
		if strings.Contains(description, api) {
			APIs = append(APIs, api)
		}
	}

	return APIs
}

// extractChangesFromText 从文本中提取变更
func (c *OpenAIAPIDocCollector) extractChangesFromText(text string) []APIChange {
	var changes []APIChange

	// 简单的文本匹配
	lines := strings.Split(text, "\n")
	var currentDate time.Time

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 尝试解析日期
		date := parseOpenAIDate(line)
		if !date.IsZero() {
			currentDate = date
			continue
		}

		// 跳过空行
		if line == "" {
			continue
		}

		// 假设非日期行是变更
		if !currentDate.IsZero() {
			changes = append(changes, APIChange{
				Title:       line,
				Description: line,
				Date:        currentDate,
				Type:        "improvement",
				URL:         c.config.ChangelogURL,
			})
		}
	}

	return changes
}
