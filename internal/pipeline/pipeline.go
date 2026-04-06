// Package pipeline 情报采集处理管道
package pipeline

import (
	"context"
	"log"
	"sync"
	"time"

	"token-bridge-crawler/internal/core"
	"token-bridge-crawler/internal/processor"
	"token-bridge-crawler/internal/rules"
	"token-bridge-crawler/internal/storage"
)

// Pipeline 情报处理管道
type Pipeline struct {
	processor *processor.IntelProcessor
	storage   storage.IntelligenceStorage
	metrics   *PipelineMetrics
	config    PipelineConfig
}

// PipelineConfig 管道配置
type PipelineConfig struct {
	EnableNoiseFilter bool `yaml:"enable_noise_filter" json:"enable_noise_filter"`
	BatchSize         int  `yaml:"batch_size" json:"batch_size"`
}

// DefaultPipelineConfig 默认配置
func DefaultPipelineConfig() PipelineConfig {
	return PipelineConfig{
		EnableNoiseFilter: true,
		BatchSize:         100,
	}
}

// PipelineMetrics 管道指标
type PipelineMetrics struct {
	mu              sync.RWMutex
	TotalProcessed  int64
	TotalNoise      int64
	TotalSignal     int64
	AvgQualityScore float64
	LastProcessedAt time.Time
}

// NewPipeline 创建情报处理管道
func NewPipeline(ruleEngine rules.RuleEngine, intelStorage storage.IntelligenceStorage, config PipelineConfig) *Pipeline {
	proc := processor.NewDefaultProcessor(ruleEngine)
	return &Pipeline{
		processor: proc,
		storage:   intelStorage,
		metrics:   &PipelineMetrics{},
		config:    config,
	}
}

// Process 处理情报项
func (p *Pipeline) Process(ctx context.Context, items []core.IntelItem) *ProcessResult {
	start := time.Now()
	result := &ProcessResult{
		Total:     len(items),
		Processed: make([]processor.ProcessResult, 0, len(items)),
	}

	processResults := p.processor.ProcessBatch(ctx, items)

	var signalItems []core.IntelItem
	var noiseCount int

	for _, pr := range processResults {
		result.Processed = append(result.Processed, pr)

		item := pr.Item
		if item.Metadata == nil {
			item.Metadata = make(core.Metadata)
		}

		// 填充质量评分字段（新数据库字段）
		item.QualityScore = &pr.QualityScore
		item.PainScore = &pr.PainScore
		item.CustomerTier = &pr.CustomerTier
		item.SignalType = &pr.SignalType
		item.IsNoise = &pr.IsNoise

		// 同时保留 metadata 中的数据（向后兼容）
		item.Metadata["quality_score"] = pr.QualityScore
		item.Metadata["pain_score"] = pr.PainScore
		item.Metadata["customer_tier"] = pr.CustomerTier
		item.Metadata["signal_type"] = pr.SignalType
		item.Metadata["matched_rules"] = pr.MatchedRules

		if pr.IsNoise {
			noiseCount++
			// 噪声也保存，但标记为噪声
			filterReason := "noise_detected"
			item.FilterReason = &filterReason
		}

		signalItems = append(signalItems, item)
	}

	result.Signal = len(signalItems)
	result.Noise = noiseCount
	result.Items = signalItems

	p.updateMetrics(result)

	elapsed := time.Since(start)
	log.Printf("[Pipeline] Processed %d items: signal=%d, noise=%d, elapsed=%v",
		result.Total, result.Signal, result.Noise, elapsed)

	return result
}

// ProcessAndStore 处理并存储情报项
func (p *Pipeline) ProcessAndStore(ctx context.Context, items []core.IntelItem) (*ProcessResult, error) {
	result := p.Process(ctx, items)

	if len(result.Items) > 0 {
		if err := p.storage.SaveItems(ctx, result.Items); err != nil {
			log.Printf("[Pipeline] Failed to save items: %v", err)
			return result, err
		}
	}

	return result, nil
}

// updateMetrics 更新指标
func (p *Pipeline) updateMetrics(result *ProcessResult) {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()

	p.metrics.TotalProcessed += int64(result.Total)
	p.metrics.TotalNoise += int64(result.Noise)
	p.metrics.TotalSignal += int64(result.Signal)
	p.metrics.LastProcessedAt = time.Now()

	if len(result.Processed) > 0 {
		var qualitySum float64
		for _, pr := range result.Processed {
			qualitySum += pr.QualityScore
		}
		p.metrics.AvgQualityScore = qualitySum / float64(len(result.Processed))
	}
}

// GetMetrics 获取指标
func (p *Pipeline) GetMetrics() PipelineMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()
	return *p.metrics
}

// ProcessResult 处理结果
type ProcessResult struct {
	Total     int                       `json:"total"`
	Signal    int                       `json:"signal"`
	Noise     int                       `json:"noise"`
	Items     []core.IntelItem          `json:"items"`
	Processed []processor.ProcessResult `json:"processed"`
}

// GetStats 获取统计信息
func (r *ProcessResult) GetStats() map[string]interface{} {
	noiseRate := 0.0
	if r.Total > 0 {
		noiseRate = float64(r.Noise) / float64(r.Total) * 100
	}
	return map[string]interface{}{
		"total":       r.Total,
		"signal":      r.Signal,
		"noise":       r.Noise,
		"noise_rate":  noiseRate,
		"signal_rate": 100 - noiseRate,
	}
}

// CollectorRunner 采集器运行器
type CollectorRunner struct {
	pipeline *Pipeline
	storage  storage.IntelligenceStorage
	metrics  *core.MetricsCollector
}

// NewCollectorRunner 创建采集器运行器
func NewCollectorRunner(pipeline *Pipeline, storage storage.IntelligenceStorage) *CollectorRunner {
	return &CollectorRunner{
		pipeline: pipeline,
		storage:  storage,
		metrics:  core.NewMetricsCollector(),
	}
}

// RunCollector 运行采集器
func (r *CollectorRunner) RunCollector(ctx context.Context, collector core.Collector) (*CollectorRunResult, error) {
	start := time.Now()
	result := &CollectorRunResult{
		CollectorName: collector.Name(),
		IntelType:     string(collector.IntelType()),
		Source:        collector.Source(),
	}

	runRecord := &storage.CollectorRun{
		CollectorName: collector.Name(),
		IntelType:     string(collector.IntelType()),
		Source:        collector.Source(),
		Status:        "running",
		StartedAt:     start,
	}

	items, err := collector.Fetch(ctx)
	if err != nil {
		runRecord.Status = "failed"
		runRecord.ErrorMessage = err.Error()
		now := time.Now()
		runRecord.CompletedAt = &now
		runRecord.DurationMs = int(now.Sub(start).Milliseconds())
		_ = r.storage.SaveCollectorRun(ctx, runRecord)

		result.Error = err
		r.metrics.RecordRun(collector.Name(), false, time.Since(start), err)
		return result, err
	}

	processedResult, err := r.pipeline.ProcessAndStore(ctx, items)
	if err != nil {
		runRecord.Status = "partial"
		runRecord.ErrorMessage = err.Error()
	} else {
		runRecord.Status = "success"
	}

	now := time.Now()
	runRecord.CompletedAt = &now
	runRecord.DurationMs = int(now.Sub(start).Milliseconds())
	runRecord.ItemsCount = processedResult.Signal
	_ = r.storage.SaveCollectorRun(ctx, runRecord)

	r.metrics.RecordRun(collector.Name(), true, time.Since(start), nil)

	result.ItemsFetched = len(items)
	result.ItemsKept = processedResult.Signal
	result.ItemsFiltered = processedResult.Noise
	result.Duration = time.Since(start)

	return result, nil
}

// CollectorRunResult 采集器运行结果
type CollectorRunResult struct {
	CollectorName string        `json:"collector_name"`
	IntelType     string        `json:"intel_type"`
	Source        string        `json:"source"`
	ItemsFetched  int           `json:"items_fetched"`
	ItemsKept     int           `json:"items_kept"`
	ItemsFiltered int           `json:"items_filtered"`
	Duration      time.Duration `json:"duration"`
	Error         error         `json:"error,omitempty"`
}

// GetStats 获取统计
func (r *CollectorRunResult) GetStats() map[string]interface{} {
	filterRate := 0.0
	if r.ItemsFetched > 0 {
		filterRate = float64(r.ItemsFiltered) / float64(r.ItemsFetched) * 100
	}
	return map[string]interface{}{
		"collector":   r.CollectorName,
		"fetched":     r.ItemsFetched,
		"kept":        r.ItemsKept,
		"filtered":    r.ItemsFiltered,
		"filter_rate": filterRate,
		"duration_ms": r.Duration.Milliseconds(),
	}
}
