// Package policy 提供政策变更情报采集功能
package policy

import (
	"context"
	"time"

	"token-bridge-crawler/internal/core"
)

// PolicyCollector 政策变更采集器接口
type PolicyCollector interface {
	core.Collector
	CollectPolicyChanges(ctx context.Context) ([]core.IntelItem, error)
}

// BasePolicyCollector 基础政策采集器
type BasePolicyCollector struct {
	name     string
	source   string
	interval time.Duration
}

// NewBasePolicyCollector 创建基础政策采集器
func NewBasePolicyCollector(name, source string, interval time.Duration) *BasePolicyCollector {
	return &BasePolicyCollector{
		name:     name,
		source:   source,
		interval: interval,
	}
}

// Name 返回采集器名称
func (c *BasePolicyCollector) Name() string {
	return c.name
}

// Source 返回数据源
func (c *BasePolicyCollector) Source() string {
	return c.source
}

// Interval 返回采集间隔
func (c *BasePolicyCollector) Interval() time.Duration {
	return c.interval
}

// IntelType 返回情报类型
func (c *BasePolicyCollector) IntelType() core.IntelType {
	return core.IntelTypePolicy
}

// Fetch 采集政策变更
func (c *BasePolicyCollector) Fetch(ctx context.Context) ([]core.IntelItem, error) {
	return c.CollectPolicyChanges(ctx)
}

// RateLimit 返回请求间隔
func (c *BasePolicyCollector) RateLimit() time.Duration {
	return c.interval
}

// CollectPolicyChanges 采集政策变更（需要子类实现）
func (c *BasePolicyCollector) CollectPolicyChanges(ctx context.Context) ([]core.IntelItem, error) {
	return nil, nil
}
