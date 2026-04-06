// Package apidoc 提供API文档采集器
package apidoc

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"token-bridge-crawler/internal/core"

	"github.com/PuerkitoBio/goquery"
)

// APIDocCollector API文档采集器基类
type APIDocCollector struct {
	core.BaseCollector
	config APIDocConfig
}

// APIDocConfig 配置
type APIDocConfig struct {
	ChangelogURL string
	APIDocURL    string
	RateLimit    time.Duration
}

// APIChange 文档变更
type APIChange struct {
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Date         time.Time `json:"date"`
	Type         string    `json:"type"` // "new_feature", "breaking_change", "deprecated", "bug_fix", "improvement"
	URL          string    `json:"url"`
	AffectedAPIs []string  `json:"affected_apis"`
}

// NewAPIDocCollector 创建API文档采集器
func NewAPIDocCollector(vendor, displayName string, config APIDocConfig) *APIDocCollector {
	return &APIDocCollector{
		BaseCollector: core.NewBaseCollector(
			fmt.Sprintf("apidoc_%s", vendor),
			core.IntelTypeAPIDoc,
			vendor,
			config.RateLimit,
		),
		config: config,
	}
}

// Fetch 实现采集接口
func (c *APIDocCollector) Fetch(ctx context.Context) ([]core.IntelItem, error) {
	// 抓取changelog页面
	doc, err := c.fetchPage(c.config.ChangelogURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch changelog: %w", err)
	}

	// 解析变更
	changes := c.parseChangelog(doc)
	if len(changes) == 0 {
		return nil, fmt.Errorf("no changes found in changelog")
	}

	// 转换为情报项
	items := c.toIntelItems(changes)

	return items, nil
}

// parseChangelog 解析changelog（子类实现）
func (c *APIDocCollector) parseChangelog(doc *goquery.Document) []APIChange {
	return []APIChange{}
}

// toIntelItems 转换为情报项
func (c *APIDocCollector) toIntelItems(changes []APIChange) []core.IntelItem {
	var items []core.IntelItem

	for _, change := range changes {
		item := core.NewIntelItem(core.IntelTypeAPIDoc, c.Source())
		item.SourceID = fmt.Sprintf("%s-%s-%s", c.Source(), change.Type, change.Date.Format("20060102"))
		item.Title = change.Title
		item.Content = change.Description
		item.URL = change.URL
		item.CapturedAt = time.Now().UTC()

		if !change.Date.IsZero() {
			item.PublishedAt = &change.Date
		}

		// 设置元数据
		item.Metadata = core.Metadata{
			"change_type":    change.Type,
			"affected_apis":  change.AffectedAPIs,
			"published_date": change.Date,
		}

		items = append(items, item)
	}

	return items
}

// fetchPage 抓取页面
func (c *APIDocCollector) fetchPage(url string) (*goquery.Document, error) {
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

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	return doc, nil
}

// LogChange 记录变更
func (c *APIDocCollector) LogChange(change APIChange) {
	log.Printf("[APIDoc:%s] %s - %s", c.Source(), change.Type, change.Title)
}
