// Package tool 提供工具生态情报采集功能
package tool

import (
	"context"
	"time"

	"token-bridge-crawler/internal/core"
)

// ToolEcosystemCollector 工具生态采集器接口
type ToolEcosystemCollector interface {
	core.Collector
	CollectToolEcosystem(ctx context.Context) ([]core.IntelItem, error)
}

// BaseToolEcosystemCollector 基础工具生态采集器
type BaseToolEcosystemCollector struct {
	name     string
	source   string
	interval time.Duration
}

// NewBaseToolEcosystemCollector 创建基础工具生态采集器
func NewBaseToolEcosystemCollector(name, source string, interval time.Duration) *BaseToolEcosystemCollector {
	return &BaseToolEcosystemCollector{
		name:     name,
		source:   source,
		interval: interval,
	}
}

// Name 返回采集器名称
func (c *BaseToolEcosystemCollector) Name() string {
	return c.name
}

// Source 返回数据源
func (c *BaseToolEcosystemCollector) Source() string {
	return c.source
}

// Interval 返回采集间隔
func (c *BaseToolEcosystemCollector) Interval() time.Duration {
	return c.interval
}

// IntelType 返回情报类型
func (c *BaseToolEcosystemCollector) IntelType() core.IntelType {
	return core.IntelTypeToolEcosystem
}

// Fetch 采集工具生态
func (c *BaseToolEcosystemCollector) Fetch(ctx context.Context) ([]core.IntelItem, error) {
	return c.CollectToolEcosystem(ctx)
}

// RateLimit 返回请求间隔
func (c *BaseToolEcosystemCollector) RateLimit() time.Duration {
	return c.interval
}

// CollectToolEcosystem 采集工具生态（需要子类实现）
func (c *BaseToolEcosystemCollector) CollectToolEcosystem(ctx context.Context) ([]core.IntelItem, error) {
	return nil, nil
}
