// Package core 提供情报系统的核心接口
package core

import (
	"context"
	"time"
)

// Collector 情报采集器接口
type Collector interface {
	// Name 返回采集器名称
	Name() string
	
	// IntelType 返回采集的情报类型
	IntelType() IntelType
	
	// Source 返回数据来源标识
	Source() string
	
	// Fetch 执行数据采集
	Fetch(ctx context.Context) ([]IntelItem, error)
	
	// RateLimit 返回请求间隔（防封）
	RateLimit() time.Duration
}

// EnhancedCollector 增强版采集器接口（支持多策略、健康检查等）
type EnhancedCollector interface {
	Collector
	
	// FetchWithFallback 带降级策略的采集
	// 按优先级尝试多种抓取策略，任一成功即返回
	FetchWithFallback(ctx context.Context) ([]IntelItem, error)
	
	// Validate 验证采集数据的合理性
	Validate(items []IntelItem) error
	
	// HealthCheck 健康检查
	// 返回采集器当前状态："healthy", "degraded", "unhealthy"
	HealthCheck() (status string, err error)
	
	// GetStrategies 获取支持的抓取策略列表
	GetStrategies() []FetchStrategy
}

// FetchStrategy 抓取策略
type FetchStrategy struct {
	Name     string                // 策略名称："web", "api", "static"
	Priority int                   // 优先级，数字越小优先级越高
	Fetch    func(ctx context.Context) ([]IntelItem, error) // 执行函数
}

// StrategyExecutor 策略执行器
type StrategyExecutor struct {
	strategies []FetchStrategy
}

// NewStrategyExecutor 创建策略执行器
func NewStrategyExecutor(strategies []FetchStrategy) *StrategyExecutor {
	return &StrategyExecutor{strategies: strategies}
}

// Execute 按优先级执行策略，直到成功
func (se *StrategyExecutor) Execute(ctx context.Context) ([]IntelItem, string, error) {
	for _, strategy := range se.strategies {
		items, err := strategy.Fetch(ctx)
		if err == nil && len(items) > 0 {
			return items, strategy.Name, nil
		}
	}
	return nil, "", ErrAllStrategiesFailed{}
}

// ErrAllStrategiesFailed 所有策略都失败
type ErrAllStrategiesFailed struct{}

func (e ErrAllStrategiesFailed) Error() string {
	return "all fetch strategies failed"
}

// BaseCollector 采集器基类（提供通用功能）
type BaseCollector struct {
	name      string
	intelType IntelType
	source    string
	rateLimit time.Duration
}

// NewBaseCollector 创建基础采集器
func NewBaseCollector(name string, intelType IntelType, source string, rateLimit time.Duration) BaseCollector {
	return BaseCollector{
		name:      name,
		intelType: intelType,
		source:    source,
		rateLimit: rateLimit,
	}
}

// Name 返回采集器名称
func (bc *BaseCollector) Name() string {
	return bc.name
}

// IntelType 返回情报类型
func (bc *BaseCollector) IntelType() IntelType {
	return bc.intelType
}

// Source 返回数据来源
func (bc *BaseCollector) Source() string {
	return bc.source
}

// RateLimit 返回请求间隔
func (bc *BaseCollector) RateLimit() time.Duration {
	return bc.rateLimit
}

// CollectorRegistry 采集器注册表
type CollectorRegistry struct {
	collectors map[string]Collector
}

// NewCollectorRegistry 创建注册表
func NewCollectorRegistry() *CollectorRegistry {
	return &CollectorRegistry{
		collectors: make(map[string]Collector),
	}
}

// Register 注册采集器
func (cr *CollectorRegistry) Register(c Collector) {
	cr.collectors[c.Name()] = c
}

// Get 获取采集器
func (cr *CollectorRegistry) Get(name string) (Collector, bool) {
	c, ok := cr.collectors[name]
	return c, ok
}

// GetByType 按类型获取采集器
func (cr *CollectorRegistry) GetByType(intelType IntelType) []Collector {
	var result []Collector
	for _, c := range cr.collectors {
		if c.IntelType() == intelType {
			result = append(result, c)
		}
	}
	return result
}

// List 列出所有采集器
func (cr *CollectorRegistry) List() []Collector {
	var result []Collector
	for _, c := range cr.collectors {
		result = append(result, c)
	}
	return result
}

// CollectorMetrics 采集器指标
type CollectorMetrics struct {
	CollectorName   string
	TotalRuns       int64
	SuccessRuns     int64
	FailedRuns      int64
	LastRunAt       *time.Time
	LastSuccessAt   *time.Time
	LastError       string
	LastErrorAt     *time.Time
	AvgFetchTime    time.Duration
	DataQualityScore float64 // 0-100
}

// MetricsCollector 指标收集器
type MetricsCollector struct {
	metrics map[string]*CollectorMetrics
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string]*CollectorMetrics),
	}
}

// RecordRun 记录一次采集运行
func (mc *MetricsCollector) RecordRun(collectorName string, success bool, duration time.Duration, err error) {
	m, ok := mc.metrics[collectorName]
	if !ok {
		m = &CollectorMetrics{CollectorName: collectorName}
		mc.metrics[collectorName] = m
	}
	
	m.TotalRuns++
	now := time.Now().UTC()
	m.LastRunAt = &now
	
	if success {
		m.SuccessRuns++
		m.LastSuccessAt = &now
	} else {
		m.FailedRuns++
		if err != nil {
			m.LastError = err.Error()
		}
		m.LastErrorAt = &now
	}
	
	// 计算平均耗时（简单移动平均）
	if m.AvgFetchTime == 0 {
		m.AvgFetchTime = duration
	} else {
		m.AvgFetchTime = (m.AvgFetchTime + duration) / 2
	}
}

// GetMetrics 获取采集器指标
func (mc *MetricsCollector) GetMetrics(collectorName string) (*CollectorMetrics, bool) {
	m, ok := mc.metrics[collectorName]
	return m, ok
}

// GetAllMetrics 获取所有指标
func (mc *MetricsCollector) GetAllMetrics() []*CollectorMetrics {
	var result []*CollectorMetrics
	for _, m := range mc.metrics {
		result = append(result, m)
	}
	return result
}

// CalculateSuccessRate 计算成功率
func (cm *CollectorMetrics) CalculateSuccessRate() float64 {
	if cm.TotalRuns == 0 {
		return 0
	}
	return float64(cm.SuccessRuns) / float64(cm.TotalRuns) * 100
}
