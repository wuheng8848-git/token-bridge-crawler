// Package scheduler 提供情报系统调度功能
package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"token-bridge-crawler/internal/core"
	"token-bridge-crawler/internal/storage"
)

// IntelligenceScheduler 情报调度器
type IntelligenceScheduler struct {
	registry  *core.CollectorRegistry
	storage   storage.IntelligenceStorage
	cron      *cron.Cron
	metrics   *core.MetricsCollector
	
	// 运行状态
	running   map[string]bool
	mu        sync.RWMutex
	
	// 配置
	config    SchedulerConfig
}

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	// 全局配置
	DefaultRateLimit time.Duration
	MaxRetries       int
	RetryDelay       time.Duration
	
	// 采集器特定配置
	CollectorConfigs map[string]CollectorConfig
}

// CollectorConfig 采集器配置
type CollectorConfig struct {
	Enabled   bool
	Cron      string  // 可选，覆盖默认调度
	RateLimit time.Duration
}

// NewIntelligenceScheduler 创建情报调度器
func NewIntelligenceScheduler(
	registry *core.CollectorRegistry,
	storage storage.IntelligenceStorage,
	config SchedulerConfig,
) *IntelligenceScheduler {
	return &IntelligenceScheduler{
		registry: registry,
		storage:  storage,
		cron:     cron.New(cron.WithSeconds()),
		metrics:  core.NewMetricsCollector(),
		running:  make(map[string]bool),
		config:   config,
	}
}

// Start 启动调度器
func (s *IntelligenceScheduler) Start() error {
	log.Println("[Scheduler] 启动情报调度器...")
	
	// 为每个启用的采集器注册定时任务
	for _, collector := range s.registry.List() {
		if !s.isEnabled(collector.Name()) {
			log.Printf("[Scheduler] 采集器 %s 已禁用，跳过", collector.Name())
			continue
		}
		
		// 获取采集器的调度配置
		cronExpr := s.getCronExpression(collector)
		
		_, err := s.cron.AddFunc(cronExpr, func(c core.Collector) func() {
			return func() {
				s.executeCollector(context.Background(), c)
			}
		}(collector))
		
		if err != nil {
			return fmt.Errorf("failed to schedule collector %s: %w", collector.Name(), err)
		}
		
		log.Printf("[Scheduler] 已注册采集器 %s，调度规则: %s", collector.Name(), cronExpr)
	}
	
	s.cron.Start()
	log.Println("[Scheduler] 情报调度器已启动")
	return nil
}

// Stop 停止调度器
func (s *IntelligenceScheduler) Stop() {
	log.Println("[Scheduler] 停止情报调度器...")
	ctx := s.cron.Stop()
	<-ctx.Done()
	log.Println("[Scheduler] 情报调度器已停止")
}

// ExecuteNow 立即执行指定采集器
func (s *IntelligenceScheduler) ExecuteNow(ctx context.Context, collectorName string) error {
	collector, ok := s.registry.Get(collectorName)
	if !ok {
		return fmt.Errorf("collector not found: %s", collectorName)
	}
	
	return s.executeCollector(ctx, collector)
}

// ExecuteAllNow 立即执行所有采集器
func (s *IntelligenceScheduler) ExecuteAllNow(ctx context.Context) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(s.registry.List()))
	
	for _, collector := range s.registry.List() {
		if !s.isEnabled(collector.Name()) {
			continue
		}
		
		wg.Add(1)
		go func(c core.Collector) {
			defer wg.Done()
			if err := s.executeCollector(ctx, c); err != nil {
				errChan <- fmt.Errorf("%s: %w", c.Name(), err)
			}
		}(collector)
	}
	
	wg.Wait()
	close(errChan)
	
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("some collectors failed: %v", errs)
	}
	return nil
}

// executeCollector 执行单个采集器
func (s *IntelligenceScheduler) executeCollector(ctx context.Context, collector core.Collector) error {
	// 检查是否已在运行
	if !s.setRunning(collector.Name(), true) {
		log.Printf("[Scheduler] 采集器 %s 正在运行中，跳过本次执行", collector.Name())
		return nil
	}
	defer s.setRunning(collector.Name(), false)
	
	startTime := time.Now().UTC()
	log.Printf("[Scheduler] 开始执行采集器 %s", collector.Name())
	
	// 创建运行记录
	run := &storage.CollectorRun{
		CollectorName: collector.Name(),
		IntelType:     string(collector.IntelType()),
		Source:        collector.Source(),
		StartedAt:     startTime,
		Status:        "running",
	}
	
	// 执行采集
	var items []core.IntelItem
	var err error
	var strategyUsed string
	
	// 如果是增强版采集器，使用带降级的抓取
	if enhanced, ok := collector.(core.EnhancedCollector); ok {
		items, err = enhanced.FetchWithFallback(ctx)
		strategyUsed = "enhanced"
	} else {
		items, err = collector.Fetch(ctx)
		strategyUsed = "default"
	}
	
	duration := time.Since(startTime)
	
	// 处理结果
	if err != nil {
		run.Status = "failed"
		run.ErrorMessage = err.Error()
		run.CompletedAt = func() *time.Time { t := time.Now().UTC(); return &t }()
		run.DurationMs = int(duration.Milliseconds())
		run.StrategyUsed = strategyUsed
		
		s.storage.SaveCollectorRun(ctx, run)
		s.metrics.RecordRun(collector.Name(), false, duration, err)
		
		log.Printf("[Scheduler] 采集器 %s 执行失败: %v", collector.Name(), err)
		return err
	}
	
	// 验证数据（如果是增强版）
	if enhanced, ok := collector.(core.EnhancedCollector); ok {
		if validateErr := enhanced.Validate(items); validateErr != nil {
			log.Printf("[Scheduler] 采集器 %s 数据验证警告: %v", collector.Name(), validateErr)
			// 验证失败不中断，继续保存
		}
	}
	
	// 保存数据
	if len(items) > 0 {
		if saveErr := s.storage.SaveItems(ctx, items); saveErr != nil {
			run.Status = "partial"
			run.ErrorMessage = saveErr.Error()
			log.Printf("[Scheduler] 采集器 %s 保存数据失败: %v", collector.Name(), saveErr)
		} else {
			run.Status = "success"
			log.Printf("[Scheduler] 采集器 %s 成功保存 %d 条数据", collector.Name(), len(items))
		}
		run.ItemsCount = len(items)
	} else {
		run.Status = "success"
		log.Printf("[Scheduler] 采集器 %s 未获取到数据", collector.Name())
	}
	
	completedAt := time.Now().UTC()
	run.CompletedAt = &completedAt
	run.DurationMs = int(duration.Milliseconds())
	run.StrategyUsed = strategyUsed
	
	// 保存运行记录
	s.storage.SaveCollectorRun(ctx, run)
	s.metrics.RecordRun(collector.Name(), true, duration, nil)
	
	return nil
}

// isEnabled 检查采集器是否启用
func (s *IntelligenceScheduler) isEnabled(collectorName string) bool {
	if cfg, ok := s.config.CollectorConfigs[collectorName]; ok {
		return cfg.Enabled
	}
	return true // 默认启用
}

// getCronExpression 获取采集器的调度表达式
func (s *IntelligenceScheduler) getCronExpression(collector core.Collector) string {
	// 如果配置了特定规则，使用配置
	if cfg, ok := s.config.CollectorConfigs[collector.Name()]; ok && cfg.Cron != "" {
		return cfg.Cron
	}
	
	// 根据情报类型使用默认规则
	switch collector.IntelType() {
	case core.IntelTypePrice:
		return "0 0 * * * *" // 每小时
	case core.IntelTypeAPIDoc:
		return "0 0 */6 * * *" // 每6小时
	case core.IntelTypeCommunity:
		return "0 0 */2 * * *" // 每2小时
	case core.IntelTypeNews:
		return "0 0 9 * * *" // 每天9点
	default:
		return "0 0 * * * *" // 默认每小时
	}
}

// setRunning 设置采集器运行状态，返回是否成功获取锁
func (s *IntelligenceScheduler) setRunning(name string, running bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if running && s.running[name] {
		return false // 已在运行
	}
	
	s.running[name] = running
	return true
}

// GetMetrics 获取采集器指标
func (s *IntelligenceScheduler) GetMetrics() []*core.CollectorMetrics {
	return s.metrics.GetAllMetrics()
}

// GetCollectorStatus 获取采集器状态
func (s *IntelligenceScheduler) GetCollectorStatus(collectorName string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	collector, ok := s.registry.Get(collectorName)
	if !ok {
		return "", fmt.Errorf("collector not found: %s", collectorName)
	}
	
	// 如果是增强版，使用健康检查
	if enhanced, ok := collector.(core.EnhancedCollector); ok {
		status, err := enhanced.HealthCheck()
		if err != nil {
			return "unhealthy", err
		}
		return status, nil
	}
	
	// 基础版返回运行状态
	if s.running[collectorName] {
		return "running", nil
	}
	return "idle", nil
}
