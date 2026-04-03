// Package reporter 提供报告生成功能
package reporter

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"token-bridge-crawler/internal/core"
	"token-bridge-crawler/internal/storage"
)

// DailyReporter 日报生成器
type DailyReporter struct {
	storage storage.IntelligenceStorage
	config  DailyReportConfig
}

// DailyReportConfig 日报配置
type DailyReportConfig struct {
	Enabled     bool
	Cron        string
	Email       bool
	EmailTo     []string
	EmailFrom   string
	EmailSubject string
	IncludeTypes []core.IntelType
}

// NewDailyReporter 创建日报生成器
func NewDailyReporter(storage storage.IntelligenceStorage, config DailyReportConfig) *DailyReporter {
	return &DailyReporter{
		storage: storage,
		config:  config,
	}
}

// Generate 生成日报
func (r *DailyReporter) Generate(ctx context.Context) (string, error) {
	// 计算时间范围：过去24小时
	endTime := time.Now().UTC()
	startTime := endTime.Add(-24 * time.Hour)

	// 构建查询过滤器
	filter := storage.IntelFilter{
		StartTime: &startTime,
		EndTime:   &endTime,
		Limit:     100,
	}

	// 查询数据
	items, err := r.storage.GetItems(ctx, filter)
	if err != nil {
		return "", fmt.Errorf("failed to get intelligence items: %w", err)
	}

	// 按类型分组
	itemsByType := r.groupItemsByType(items)

	// 生成报告
	report := r.generateReport(itemsByType, startTime, endTime)

	// 发送邮件（如果启用）
	if r.config.Email {
		r.sendEmail(report)
	}

	return report, nil
}

// groupItemsByType 按类型分组
func (r *DailyReporter) groupItemsByType(items []core.IntelItem) map[core.IntelType][]core.IntelItem {
	result := make(map[core.IntelType][]core.IntelItem)

	for _, item := range items {
		result[item.IntelType] = append(result[item.IntelType], item)
	}

	return result
}

// generateReport 生成报告内容
func (r *DailyReporter) generateReport(itemsByType map[core.IntelType][]core.IntelItem, startTime, endTime time.Time) string {
	var report strings.Builder

	// 报告头部
	report.WriteString(fmt.Sprintf("📊 Token Bridge 情报日报 - %s\n\n", endTime.Format("2006-01-02")))
	report.WriteString(fmt.Sprintf("时间范围: %s 至 %s\n\n", 
		startTime.Format("2006-01-02 15:04"), 
		endTime.Format("2006-01-02 15:04")))

	// 价格情报
	if items, ok := itemsByType[core.IntelTypePrice]; ok && len(items) > 0 {
		report.WriteString("## 价格情报\n")
		r.reportPriceItems(items, &report)
		report.WriteString("\n")
	}

	// API文档变更
	if items, ok := itemsByType[core.IntelTypeAPIDoc]; ok && len(items) > 0 {
		report.WriteString("## API文档变更\n")
		r.reportAPIDocItems(items, &report)
		report.WriteString("\n")
	}

	// 社区情报
	if items, ok := itemsByType[core.IntelTypeCommunity]; ok && len(items) > 0 {
		report.WriteString("## 社区动态\n")
		r.reportCommunityItems(items, &report)
		report.WriteString("\n")
	}

	// 新闻情报
	if items, ok := itemsByType[core.IntelTypeNews]; ok && len(items) > 0 {
		report.WriteString("## 行业新闻\n")
		report.WriteString(fmt.Sprintf("共 %d 条新闻\n", len(items)))
		report.WriteString("\n")
	}

	// 总结
	totalItems := 0
	for _, items := range itemsByType {
		totalItems += len(items)
	}

	report.WriteString(fmt.Sprintf("## 总结\n"))
	report.WriteString(fmt.Sprintf("- 总情报数: %d\n", totalItems))
	report.WriteString(fmt.Sprintf("- 价格情报: %d\n", len(itemsByType[core.IntelTypePrice])))
	report.WriteString(fmt.Sprintf("- API变更: %d\n", len(itemsByType[core.IntelTypeAPIDoc])))
	report.WriteString(fmt.Sprintf("- 社区动态: %d\n", len(itemsByType[core.IntelTypeCommunity])))
	report.WriteString(fmt.Sprintf("- 行业新闻: %d\n", len(itemsByType[core.IntelTypeNews])))

	return report.String()
}

// reportPriceItems 报告价格情报
func (r *DailyReporter) reportPriceItems(items []core.IntelItem, report *strings.Builder) {
	// 按来源分组
	sources := make(map[string][]core.IntelItem)
	for _, item := range items {
		sources[item.Source] = append(sources[item.Source], item)
	}

	for source, items := range sources {
		report.WriteString(fmt.Sprintf("### %s\n", source))

		for _, item := range items {
			if item.Title != "" {
				report.WriteString(fmt.Sprintf("- %s\n", item.Title))
			}
			if item.Content != "" {
				report.WriteString(fmt.Sprintf("  %s\n", item.Content))
			}
			if item.URL != "" {
				report.WriteString(fmt.Sprintf("  [查看详情](%s)\n", item.URL))
			}
			report.WriteString("\n")
		}
	}
}

// reportAPIDocItems 报告API文档变更
func (r *DailyReporter) reportAPIDocItems(items []core.IntelItem, report *strings.Builder) {
	// 按类型分组
	types := make(map[string][]core.IntelItem)
	for _, item := range items {
		changeType := "unknown"
		if changeTypeVal, ok := item.Metadata["change_type"]; ok {
			changeType = fmt.Sprintf("%v", changeTypeVal)
		}
		types[changeType] = append(types[changeType], item)
	}

	for changeType, items := range types {
		report.WriteString(fmt.Sprintf("### %s\n", changeType))

		for _, item := range items {
			if item.Title != "" {
				report.WriteString(fmt.Sprintf("- %s\n", item.Title))
			}
			if item.Content != "" {
				report.WriteString(fmt.Sprintf("  %s\n", item.Content))
			}
			if item.URL != "" {
				report.WriteString(fmt.Sprintf("  [查看详情](%s)\n", item.URL))
			}
			report.WriteString("\n")
		}
	}
}

// reportCommunityItems 报告社区动态
func (r *DailyReporter) reportCommunityItems(items []core.IntelItem, report *strings.Builder) {
	for _, item := range items {
		if item.Title != "" {
			report.WriteString(fmt.Sprintf("- %s\n", item.Title))
		}
		if item.Content != "" {
			report.WriteString(fmt.Sprintf("  %s\n", item.Content))
		}
		if item.URL != "" {
			report.WriteString(fmt.Sprintf("  [查看详情](%s)\n", item.URL))
		}
		report.WriteString("\n")
	}
}

// sendEmail 发送邮件
func (r *DailyReporter) sendEmail(report string) {
	// 这里可以集成邮件发送逻辑
	// 暂时只记录日志
	log.Printf("[Reporter] 发送日报邮件到: %v", r.config.EmailTo)
	log.Printf("[Reporter] 邮件内容长度: %d 字符", len(report))
}
