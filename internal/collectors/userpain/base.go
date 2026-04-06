// Package userpain 提供用户痛点情报采集功能
package userpain

import (
	"context"
	"time"

	"token-bridge-crawler/internal/core"
)

// UserPainCollector 用户痛点采集器接口
type UserPainCollector interface {
	core.Collector
	CollectUserPains(ctx context.Context) ([]core.IntelItem, error)
}

// BaseUserPainCollector 基础用户痛点采集器
type BaseUserPainCollector struct {
	name     string
	source   string
	interval time.Duration
}

// NewBaseUserPainCollector 创建基础用户痛点采集器
func NewBaseUserPainCollector(name, source string, interval time.Duration) *BaseUserPainCollector {
	return &BaseUserPainCollector{
		name:     name,
		source:   source,
		interval: interval,
	}
}

// Name 返回采集器名称
func (c *BaseUserPainCollector) Name() string {
	return c.name
}

// Source 返回数据源
func (c *BaseUserPainCollector) Source() string {
	return c.source
}

// Interval 返回采集间隔
func (c *BaseUserPainCollector) Interval() time.Duration {
	return c.interval
}

// IntelType 返回情报类型
func (c *BaseUserPainCollector) IntelType() core.IntelType {
	return core.IntelTypeUserPain
}

// Fetch 采集用户痛点
func (c *BaseUserPainCollector) Fetch(ctx context.Context) ([]core.IntelItem, error) {
	return c.CollectUserPains(ctx)
}

// RateLimit 返回请求间隔
func (c *BaseUserPainCollector) RateLimit() time.Duration {
	return c.interval
}

// CollectUserPains 采集用户痛点（需要子类实现）
func (c *BaseUserPainCollector) CollectUserPains(ctx context.Context) ([]core.IntelItem, error) {
	return nil, nil
}
