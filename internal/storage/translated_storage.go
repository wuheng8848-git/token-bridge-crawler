// Package storage 提供带翻译功能的存储包装器
package storage

import (
	"context"
	"log"
	"time"

	"token-bridge-crawler/internal/core"
)

// TranslatedStorage 带翻译功能的存储
type TranslatedStorage struct {
	base       IntelligenceStorage
	translator core.TranslationService
}

// NewTranslatedStorage 创建带翻译功能的存储
func NewTranslatedStorage(base IntelligenceStorage, translator core.TranslationService) *TranslatedStorage {
	return &TranslatedStorage{
		base:       base,
		translator: translator,
	}
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
	// 先批量翻译
	if s.translator != nil && len(items) > 0 {
		log.Printf("[TranslatedStorage] 开始翻译 %d 条情报项", len(items))
		_ = s.translator.TranslateIntelItems(items)
		log.Printf("[TranslatedStorage] 翻译完成")
	}

	// 再保存
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
