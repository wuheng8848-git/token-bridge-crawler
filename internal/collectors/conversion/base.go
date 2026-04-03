// Package conversion 提供转化情况情报采集功能
package conversion

import (
	"context"
	"time"

	"token-bridge-crawler/internal/core"
)

// ConversionCollector 转化情况采集器接口
type ConversionCollector interface {
	core.Collector
	CollectConversion(ctx context.Context) ([]core.IntelItem, error)
}

// BaseConversionCollector 基础转化情况采集器
type BaseConversionCollector struct {
	name     string
	source   string
	interval time.Duration
}

// NewBaseConversionCollector 创建基础转化情况采集器
func NewBaseConversionCollector(name, source string, interval time.Duration) *BaseConversionCollector {
	return &BaseConversionCollector{
		name:     name,
		source:   source,
		interval: interval,
	}
}

// Name 返回采集器名称
func (c *BaseConversionCollector) Name() string {
	return c.name
}

// Source 返回数据源
func (c *BaseConversionCollector) Source() string {
	return c.source
}

// Interval 返回采集间隔
func (c *BaseConversionCollector) Interval() time.Duration {
	return c.interval
}

// IntelType 返回情报类型
func (c *BaseConversionCollector) IntelType() core.IntelType {
	return core.IntelTypeConversion
}

// Fetch 采集转化情况
func (c *BaseConversionCollector) Fetch(ctx context.Context) ([]core.IntelItem, error) {
	return c.CollectConversion(ctx)
}

// RateLimit 返回请求间隔
func (c *BaseConversionCollector) RateLimit() time.Duration {
	return c.interval
}

// CollectConversion 采集转化情况（需要子类实现）
func (c *BaseConversionCollector) CollectConversion(ctx context.Context) ([]core.IntelItem, error) {
	return nil, nil
}
