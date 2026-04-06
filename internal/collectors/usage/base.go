// Package usage 提供使用模式情报采集功能
package usage

import (
	"context"
	"time"

	"token-bridge-crawler/internal/core"
)

// UsagePatternCollector 使用模式采集器接口
type UsagePatternCollector interface {
	core.Collector
	CollectUsagePattern(ctx context.Context) ([]core.IntelItem, error)
}

// BaseUsagePatternCollector 基础使用模式采集器
type BaseUsagePatternCollector struct {
	name     string
	source   string
	interval time.Duration
}

// NewBaseUsagePatternCollector 创建基础使用模式采集器
func NewBaseUsagePatternCollector(name, source string, interval time.Duration) *BaseUsagePatternCollector {
	return &BaseUsagePatternCollector{
		name:     name,
		source:   source,
		interval: interval,
	}
}

// Name 返回采集器名称
func (c *BaseUsagePatternCollector) Name() string {
	return c.name
}

// Source 返回数据源
func (c *BaseUsagePatternCollector) Source() string {
	return c.source
}

// Interval 返回采集间隔
func (c *BaseUsagePatternCollector) Interval() time.Duration {
	return c.interval
}

// IntelType 返回情报类型
func (c *BaseUsagePatternCollector) IntelType() core.IntelType {
	return core.IntelTypeUsagePattern
}

// Fetch 采集使用模式
func (c *BaseUsagePatternCollector) Fetch(ctx context.Context) ([]core.IntelItem, error) {
	return c.CollectUsagePattern(ctx)
}

// RateLimit 返回请求间隔
func (c *BaseUsagePatternCollector) RateLimit() time.Duration {
	return c.interval
}

// CollectUsagePattern 采集使用模式（需要子类实现）
func (c *BaseUsagePatternCollector) CollectUsagePattern(ctx context.Context) ([]core.IntelItem, error) {
	return nil, nil
}
