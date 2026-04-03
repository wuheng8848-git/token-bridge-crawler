// Package integration 提供集成机会情报采集功能
package integration

import (
	"context"
	"time"

	"token-bridge-crawler/internal/core"
)

// IntegrationCollector 集成机会采集器接口
type IntegrationCollector interface {
	core.Collector
	CollectIntegrations(ctx context.Context) ([]core.IntelItem, error)
}

// BaseIntegrationCollector 基础集成机会采集器
type BaseIntegrationCollector struct {
	name     string
	source   string
	interval time.Duration
}

// NewBaseIntegrationCollector 创建基础集成机会采集器
func NewBaseIntegrationCollector(name, source string, interval time.Duration) *BaseIntegrationCollector {
	return &BaseIntegrationCollector{
		name:     name,
		source:   source,
		interval: interval,
	}
}

// Name 返回采集器名称
func (c *BaseIntegrationCollector) Name() string {
	return c.name
}

// Source 返回数据源
func (c *BaseIntegrationCollector) Source() string {
	return c.source
}

// Interval 返回采集间隔
func (c *BaseIntegrationCollector) Interval() time.Duration {
	return c.interval
}

// IntelType 返回情报类型
func (c *BaseIntegrationCollector) IntelType() core.IntelType {
	return core.IntelTypeIntegration
}

// Fetch 采集集成机会
func (c *BaseIntegrationCollector) Fetch(ctx context.Context) ([]core.IntelItem, error) {
	return c.CollectIntegrations(ctx)
}

// RateLimit 返回请求间隔
func (c *BaseIntegrationCollector) RateLimit() time.Duration {
	return c.interval
}

// CollectIntegrations 采集集成机会（需要子类实现）
func (c *BaseIntegrationCollector) CollectIntegrations(ctx context.Context) ([]core.IntelItem, error) {
	return nil, nil
}
