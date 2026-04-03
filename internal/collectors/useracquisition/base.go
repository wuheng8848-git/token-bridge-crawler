// Package useracquisition 提供用户获取情报采集功能
package useracquisition

import (
	"context"
	"time"

	"token-bridge-crawler/internal/core"
)

// UserAcquisitionCollector 用户获取采集器接口
type UserAcquisitionCollector interface {
	core.Collector
	CollectUserAcquisition(ctx context.Context) ([]core.IntelItem, error)
}

// BaseUserAcquisitionCollector 基础用户获取采集器
type BaseUserAcquisitionCollector struct {
	name     string
	source   string
	interval time.Duration
}

// NewBaseUserAcquisitionCollector 创建基础用户获取采集器
func NewBaseUserAcquisitionCollector(name, source string, interval time.Duration) *BaseUserAcquisitionCollector {
	return &BaseUserAcquisitionCollector{
		name:     name,
		source:   source,
		interval: interval,
	}
}

// Name 返回采集器名称
func (c *BaseUserAcquisitionCollector) Name() string {
	return c.name
}

// Source 返回数据源
func (c *BaseUserAcquisitionCollector) Source() string {
	return c.source
}

// Interval 返回采集间隔
func (c *BaseUserAcquisitionCollector) Interval() time.Duration {
	return c.interval
}

// IntelType 返回情报类型
func (c *BaseUserAcquisitionCollector) IntelType() core.IntelType {
	return core.IntelTypeUserAcquisition
}

// Fetch 采集用户获取数据
func (c *BaseUserAcquisitionCollector) Fetch(ctx context.Context) ([]core.IntelItem, error) {
	return c.CollectUserAcquisition(ctx)
}

// RateLimit 返回请求间隔
func (c *BaseUserAcquisitionCollector) RateLimit() time.Duration {
	return c.interval
}

// CollectUserAcquisition 采集用户获取数据（需要子类实现）
func (c *BaseUserAcquisitionCollector) CollectUserAcquisition(ctx context.Context) ([]core.IntelItem, error) {
	return nil, nil
}
