// Package storage 提供带翻译功能的存储包装器
package storage

import (
	"context"
	"log"
	"time"

	"token-bridge-crawler/internal/core"
)

// TranslationStrategy 翻译策略配置
type TranslationStrategy struct {
	// 质量分数阈值，只有达到阈值的情报才会被翻译
	MinQualityScore float64
	// 是否只翻译高价值信号（S级和A级）
	HighValueOnly bool
	// 最大翻译条数（每批），0表示不限制
	MaxItemsPerBatch int
}

// DefaultTranslationStrategy 默认翻译策略
func DefaultTranslationStrategy() TranslationStrategy {
	return TranslationStrategy{
		MinQualityScore:  40, // 只翻译质量分≥40的（B级及以上）
		HighValueOnly:    false,
		MaxItemsPerBatch: 50, // 每批最多翻译50条
	}
}

// ConservativeTranslationStrategy 保守翻译策略（省API额度）
func ConservativeTranslationStrategy() TranslationStrategy {
	return TranslationStrategy{
		MinQualityScore:  60, // 只翻译质量分≥60的（A级及以上）
		HighValueOnly:    true,
		MaxItemsPerBatch: 20, // 每批最多翻译20条
	}
}

// TranslatedStorage 带翻译功能的存储
type TranslatedStorage struct {
	base       IntelligenceStorage
	translator core.TranslationService
	strategy   TranslationStrategy
}

// NewTranslatedStorage 创建带翻译功能的存储
func NewTranslatedStorage(base IntelligenceStorage, translator core.TranslationService) *TranslatedStorage {
	return &TranslatedStorage{
		base:       base,
		translator: translator,
		strategy:   DefaultTranslationStrategy(),
	}
}

// NewTranslatedStorageWithStrategy 创建带自定义翻译策略的存储
func NewTranslatedStorageWithStrategy(base IntelligenceStorage, translator core.TranslationService, strategy TranslationStrategy) *TranslatedStorage {
	return &TranslatedStorage{
		base:       base,
		translator: translator,
		strategy:   strategy,
	}
}

// SetStrategy 设置翻译策略
func (s *TranslatedStorage) SetStrategy(strategy TranslationStrategy) {
	s.strategy = strategy
}

// shouldTranslate 判断情报是否应该被翻译
func (s *TranslatedStorage) shouldTranslate(item *core.IntelItem) bool {
	// 如果没有质量分数，使用默认值判断
	qualityScore := 50.0 // 默认中等质量
	if item.QualityScore != nil {
		qualityScore = *item.QualityScore
	}

	// 检查质量分数是否达到阈值
	if qualityScore < s.strategy.MinQualityScore {
		return false
	}

	// 如果设置了只翻译高价值信号，检查客户等级
	if s.strategy.HighValueOnly {
		if item.CustomerTier != nil {
			tier := *item.CustomerTier
			// 只翻译 S级 和 A级
			if tier != "S" && tier != "A" {
				return false
			}
		}
	}

	return true
}

// SaveItem 保存单个情报项（自动翻译）
func (s *TranslatedStorage) SaveItem(ctx context.Context, item *core.IntelItem) error {
	// 先翻译
	if s.translator != nil {
		_ = s.translator.TranslateIntelItem(item)
	}

	// 再保存
	return s.base.SaveItem(ctx, item)
}

// SaveItems 批量保存情报项（自动翻译）
func (s *TranslatedStorage) SaveItems(ctx context.Context, items []core.IntelItem) error {
	// 筛选需要翻译的情报（基于质量分数和客户等级）
	var itemsToTranslate []core.IntelItem
	for i := range items {
		if s.shouldTranslate(&items[i]) {
			itemsToTranslate = append(itemsToTranslate, items[i])
		}
	}

	// 限制每批翻译数量
	if s.strategy.MaxItemsPerBatch > 0 && len(itemsToTranslate) > s.strategy.MaxItemsPerBatch {
		log.Printf("[TranslatedStorage] 翻译数量超过限制: %d 条，将只翻译前 %d 条高质量情报",
			len(itemsToTranslate), s.strategy.MaxItemsPerBatch)
		itemsToTranslate = itemsToTranslate[:s.strategy.MaxItemsPerBatch]
	}

	// 批量翻译筛选后的情报
	if s.translator != nil && len(itemsToTranslate) > 0 {
		log.Printf("[TranslatedStorage] 开始翻译 %d/%d 条高质量情报项 (质量分≥%.0f)",
			len(itemsToTranslate), len(items), s.strategy.MinQualityScore)
		_ = s.translator.TranslateIntelItems(itemsToTranslate)
		log.Printf("[TranslatedStorage] 翻译完成，跳过 %d 条低价值情报",
			len(items)-len(itemsToTranslate))
	} else {
		log.Printf("[TranslatedStorage] 跳过翻译: %d 条情报质量分低于 %.0f，不消耗API额度",
			len(items), s.strategy.MinQualityScore)
	}

	// 保存所有情报（无论是否翻译）
	return s.base.SaveItems(ctx, items)
}

// GetItemByID 根据ID获取情报项
func (s *TranslatedStorage) GetItemByID(ctx context.Context, id string) (*core.IntelItem, error) {
	return s.base.GetItemByID(ctx, id)
}

// GetItems 查询情报项列表
func (s *TranslatedStorage) GetItems(ctx context.Context, filter IntelFilter) ([]core.IntelItem, error) {
	return s.base.GetItems(ctx, filter)
}

// GetItemsCount 获取符合条件的情报项总数
func (s *TranslatedStorage) GetItemsCount(ctx context.Context, filter IntelFilter) (int64, error) {
	return s.base.GetItemsCount(ctx, filter)
}

// UpdateItemStatus 更新情报项状态
func (s *TranslatedStorage) UpdateItemStatus(ctx context.Context, id string, status core.IntelStatus) error {
	return s.base.UpdateItemStatus(ctx, id, status)
}

// GetSourceStats 获取按来源分组的统计数据
func (s *TranslatedStorage) GetSourceStats(ctx context.Context, timeRange ...time.Time) (map[string]int64, error) {
	return s.base.GetSourceStats(ctx, timeRange...)
}

// GetTranslationStats 获取翻译覆盖率统计
func (s *TranslatedStorage) GetTranslationStats(ctx context.Context) (map[string]int64, error) {
	return s.base.GetTranslationStats(ctx)
}

// GetQualityAnalysis 获取采集器质量分析
func (s *TranslatedStorage) GetQualityAnalysis(ctx context.Context, source string, limit int) (map[string]interface{}, error) {
	return s.base.GetQualityAnalysis(ctx, source, limit)
}

// SaveCollectorRun 保存采集器运行记录
func (s *TranslatedStorage) SaveCollectorRun(ctx context.Context, run *CollectorRun) error {
	return s.base.SaveCollectorRun(ctx, run)
}

// GetCollectorRuns 获取采集器运行记录
func (s *TranslatedStorage) GetCollectorRuns(ctx context.Context, collectorName string, limit int) ([]CollectorRun, error) {
	return s.base.GetCollectorRuns(ctx, collectorName, limit)
}

// GetAlertRules 获取告警规则
func (s *TranslatedStorage) GetAlertRules(ctx context.Context, enabledOnly bool) ([]AlertRule, error) {
	return s.base.GetAlertRules(ctx, enabledOnly)
}

// SaveAlertHistory 保存告警历史
func (s *TranslatedStorage) SaveAlertHistory(ctx context.Context, alert *AlertHistory) error {
	return s.base.SaveAlertHistory(ctx, alert)
}

// UpdateAlertStatus 更新告警状态
func (s *TranslatedStorage) UpdateAlertStatus(ctx context.Context, alertID string, status string) error {
	return s.base.UpdateAlertStatus(ctx, alertID, status)
}

// GetStats 获取统计信息
func (s *TranslatedStorage) GetStats(ctx context.Context, startTime, endTime time.Time) (IntelStats, error) {
	return s.base.GetStats(ctx, startTime, endTime)
}

// SaveSignals 批量保存客户信号
func (s *TranslatedStorage) SaveSignals(ctx context.Context, signals []CustomerSignal) error {
	return s.base.SaveSignals(ctx, signals)
}

// SaveActions 批量保存营销动作
func (s *TranslatedStorage) SaveActions(ctx context.Context, actions []MarketingAction) error {
	return s.base.SaveActions(ctx, actions)
}

// GetPendingActions 获取待处理的营销动作
func (s *TranslatedStorage) GetPendingActions(ctx context.Context, limit int) ([]MarketingAction, error) {
	return s.base.GetPendingActions(ctx, limit)
}

// UpdateActionStatus 更新营销动作状态
func (s *TranslatedStorage) UpdateActionStatus(ctx context.Context, actionID string, status string) error {
	return s.base.UpdateActionStatus(ctx, actionID, status)
}

// Close 关闭连接
func (s *TranslatedStorage) Close() {
	s.base.Close()
}

// SetTranslationEnabled 设置是否启用翻译
func (s *TranslatedStorage) SetTranslationEnabled(enabled bool) {
	if s.translator != nil {
		s.translator.SetEnabled(enabled)
	}
}

// GetTranslationStatus 获取翻译状态
func (s *TranslatedStorage) GetTranslationStatus() map[string]interface{} {
	if s.translator != nil {
		return s.translator.GetTranslationStatus()
	}
	return map[string]interface{}{
		"enabled":       false,
		"hasTranslator": false,
	}
}
